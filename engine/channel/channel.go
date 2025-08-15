// Package channel provides the base interface and common functionality
// shared by all audio channels (input channels, mix buses, etc.)
package channel

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
	"github.com/shaban/macaudio/avaudio/pluginchain"
	"github.com/shaban/macaudio/avaudio/tap"
	"github.com/shaban/macaudio/engine/queue"
	"github.com/shaban/macaudio/plugins"
)

// Channel defines the minimal interface shared by audio channels used by the
// engine and buses. BaseChannel implements this.
type Channel interface {
	// Identity
	GetName() string
	SetName(string)
	GetDisplayName() string
	SetDisplayName(string)

	// Mix controls
	SetVolume(float32) error
	GetVolume() (float32, error)
	SetMute(bool) error
	GetMute() (bool, error)
	SetPan(float32) error
	GetPan() (float32, error)

	// Plugin Chain Management
	GetPluginChain() *pluginchain.PluginChain
	AddEffect(plugin *plugins.Plugin) error
	AddEffectFromPluginInfo(pluginInfo plugins.PluginInfo) error

	// Routing
	GetInputNode() unsafe.Pointer  // For connecting sources to this channel
	GetOutputNode() unsafe.Pointer // For connecting this channel to destinations

	// Lifecycle
	Release()
	IsReleased() bool

	// Status
	Summary() string
}

// ChannelKind identifies allowed routing policy for a channel.
type ChannelKind int

const (
	ChannelUnknown ChannelKind = iota
	ChannelInput
	ChannelPlayer
)

// Send describes an auxiliary path from a channel to a destination bus.
// Level/mute are logical controls; level application in the graph may be
// implemented by the destination bus input gain or a future per-send gain node.
type Send struct {
	Name        string
	Destination Channel
	Level       float32
	Mute        bool
	Mode        SendMode

	// internal wiring state
	mixer    unsafe.Pointer // per-send gain mixer (created on connect)
	busInput unsafe.Pointer // destination bus mixer input pointer
	busIndex int            // destination bus input index
	prev     float32        // previous non-zero level for unmute restoration
}

// SendMode chooses where in the signal flow a send taps the audio:
//   - PreFader: after inserts (plugin chain), before the channel fader/pan
//   - PostFader: after the channel fader/pan (i.e., mixer output)
type SendMode int

const (
	// PreFader taps after inserts (plugin chain output) and before volume/pan
	PreFader SendMode = iota
	// PostFader taps after volume/pan (channel mixer output)
	PostFader
)

// SoloManager coordinates solo state across channels in a group.
// When any channels are soloed, all others are muted (solo-muted) until no solos remain.
type SoloManager struct {
	mu      sync.Mutex
	members map[*BaseChannel]struct{}
	soloed  map[*BaseChannel]struct{}
}

var DefaultSolo = &SoloManager{members: map[*BaseChannel]struct{}{}, soloed: map[*BaseChannel]struct{}{}}

func (sm *SoloManager) Register(ch *BaseChannel) {
	if sm == nil || ch == nil {
		return
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.members[ch] = struct{}{}
}

func (sm *SoloManager) Unregister(ch *BaseChannel) {
	if sm == nil || ch == nil {
		return
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.members, ch)
	delete(sm.soloed, ch)
	sm.recompute()
}

func (sm *SoloManager) SetSolo(ch *BaseChannel, on bool) {
	if sm == nil || ch == nil {
		return
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if on {
		sm.soloed[ch] = struct{}{}
	} else {
		delete(sm.soloed, ch)
	}
	sm.recompute()
}

func (sm *SoloManager) IsSoloed(ch *BaseChannel) bool {
	if sm == nil || ch == nil {
		return false
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	_, ok := sm.soloed[ch]
	return ok
}

// recompute applies solo-muted state to all members based on soloed set.
func (sm *SoloManager) recompute() {
	hasSolo := len(sm.soloed) > 0
	for ch := range sm.members {
		_, isSolo := sm.soloed[ch]
		ch.markSoloMuted(hasSolo && !isSolo)
	}
}

// BaseChannel provides a default implementation of Channel that composes a
// plugin chain (for inserts) and a per-channel mixer (for fader/pan). It does
// not own the lifetime of the Engine; callers pass an Engine when connecting.
type BaseChannel struct {
	name              string
	displayName       string
	enginePtr         unsafe.Pointer
	engineInstance    *engine.Engine // Reference to engine for accessing AudioSpec
	dispatcher        *queue.Dispatcher
	dispatcherOwned   bool
	routeMu           sync.Mutex // serialize graph mutations for this channel
	pluginChain       *pluginchain.PluginChain
	outputMixer       unsafe.Pointer   // For volume and mute control (Node)
	sends             map[string]*Send // Auxiliary sends
	sendsMu           sync.RWMutex     // Protects sends map and send state
	kind              ChannelKind
	released          bool
	connectedToMaster bool // Track master connection state
	// state controls
	userMuted  bool    // explicit mute requested by user
	soloMuted  bool    // muted due to another channel's solo state
	lastVolume float32 // remembered volume for unmute
	// metering
	meterMu    sync.RWMutex
	meterTap   *tap.Tap
	sendMeters map[string]*tap.Tap
}

// BaseChannelConfig declares the inputs required to construct a BaseChannel.
type BaseChannelConfig struct {
	Name           string
	DisplayName    string
	EnginePtr      unsafe.Pointer // AVAudioEngine pointer from avaudio/engine package
	EngineInstance *engine.Engine // Engine instance for accessing AudioSpec
	Dispatcher     *queue.Dispatcher
	Kind           ChannelKind
}

// NewBaseChannel instantiates a BaseChannel with its own plugin chain and
// output mixer. It does not perform any graph connections.
func NewBaseChannel(config BaseChannelConfig) (*BaseChannel, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("channel name cannot be empty")
	}
	if config.EnginePtr == nil {
		return nil, fmt.Errorf("engine pointer cannot be nil")
	}
	if config.EngineInstance == nil {
		return nil, fmt.Errorf("engine instance cannot be nil")
	}

	// Create plugin chain for this channel
	chainLabel := config.DisplayName
	if chainLabel == "" {
		chainLabel = config.Name
	}
	if chainLabel == "" {
		chainLabel = "Channel"
	}
	pluginChain := pluginchain.NewPluginChain(pluginchain.ChainConfig{
		Name:       chainLabel + " Chain",
		EnginePtr:  config.EnginePtr,
		Dispatcher: config.Dispatcher, // may be nil; will be set below if we auto-create
	})

	// Create output mixer for volume and mute control
	outputMixer, err := node.CreateMixer()
	if err != nil || outputMixer == nil {
		return nil, fmt.Errorf("failed to create output mixer for channel %s: %v", config.Name, err)
	}

	bc := &BaseChannel{
		name:              config.Name,
		displayName:       firstNonEmpty(config.DisplayName, config.Name),
		enginePtr:         config.EnginePtr,
		engineInstance:    config.EngineInstance,
		dispatcher:        config.Dispatcher,
		dispatcherOwned:   false,
		routeMu:           sync.Mutex{},
		pluginChain:       pluginChain,
		outputMixer:       outputMixer,
		sends:             make(map[string]*Send),
		kind:              config.Kind,
		released:          false,
		connectedToMaster: false,
		userMuted:         false,
		soloMuted:         false,
		lastVolume:        0.8, // sensible default fader value
		meterTap:          nil,
		sendMeters:        make(map[string]*tap.Tap),
	}
	// Default to Input kind if unspecified for backward compatibility
	if bc.kind == ChannelUnknown {
		bc.kind = ChannelInput
	}
	// If no dispatcher provided, create a default one bound to the engine instance
	if bc.dispatcher == nil && bc.engineInstance != nil {
		q := queue.New(32)
		bc.dispatcher = queue.NewDispatcher(bc.engineInstance, q)
		bc.dispatcher.Start()
		bc.dispatcherOwned = true
	}
	// Ensure plugin chain uses the same dispatcher
	if pluginChain != nil && bc.dispatcher != nil {
		// Replace internal chain with one that has dispatcher if it was nil
		pluginChain = pluginchain.NewPluginChain(pluginchain.ChainConfig{
			Name:       firstNonEmpty(bc.displayName, bc.name) + " Chain",
			EnginePtr:  config.EnginePtr,
			Dispatcher: bc.dispatcher,
		})
		bc.pluginChain = pluginChain
	}
	// Initialize mixer volume to lastVolume
	_ = node.SetMixerVolume(outputMixer, bc.lastVolume, 0)
	// Auto-register for global solo coordination
	DefaultSolo.Register(bc)
	return bc, nil
}

// GetName returns the channel name
func (bc *BaseChannel) GetName() string {
	return bc.name
}

// SetName updates the channel name
func (bc *BaseChannel) SetName(name string) {
	// Back-compat: interpret SetName as DisplayName update
	bc.SetDisplayName(name)
}

// GetDisplayName returns the user-facing label, defaulting to Name.
func (bc *BaseChannel) GetDisplayName() string {
	if bc.displayName == "" {
		return bc.name
	}
	return bc.displayName
}

// SetDisplayName updates the UI label and propagates to the plugin chain label.
func (bc *BaseChannel) SetDisplayName(name string) {
	bc.displayName = name
	if bc.pluginChain != nil {
		bc.pluginChain.SetName(bc.GetDisplayName() + " Chain")
	}
}

// GetAudioSpec returns the Engine's AudioSpec so channel logic can adapt to
// the current sample rate/buffer size without owning the engine itself.
func (bc *BaseChannel) GetAudioSpec() engine.AudioSpec {
	if bc.engineInstance == nil {
		// Return empty spec if no engine instance (shouldn't happen)
		return engine.AudioSpec{}
	}
	return bc.engineInstance.GetSpec()
}

// SetVolume sets the channel output volume on input bus 0 of the channel mixer.
func (bc *BaseChannel) SetVolume(volume float32) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return fmt.Errorf("output mixer not available")
	}

	if volume < 0 || volume > 1 {
		return fmt.Errorf("volume must be between 0.0 and 1.0")
	}
	// Update lastVolume if non-zero and not muted by user
	if volume > 0 {
		bc.lastVolume = volume
	}
	// Apply immediately only if not currently muted by user/solo
	if bc.userMuted || bc.soloMuted {
		return nil
	}
	return node.SetMixerVolume(bc.outputMixer, volume, 0)
}

// GetVolume reads the channel output volume from input bus 0 of the channel mixer.
func (bc *BaseChannel) GetVolume() (float32, error) {
	if bc.released {
		return 0, fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return 0, fmt.Errorf("output mixer not available")
	}

	return node.GetMixerVolume(bc.outputMixer, 0)
}

// SetMute sets mute by driving volume to 0.0, and unmute restores a nominal
// volume (temporary behavior; TODO: remember previous fader level).
func (bc *BaseChannel) SetMute(muted bool) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	bc.userMuted = muted
	// Use dispatcher to apply a tiny ramp to avoid clicks; fall back to immediate.
	if bc.dispatcher != nil {
		// Capture locals for closure
		mix := bc.outputMixer
		vol := bc.lastVolume
		if bc.userMuted || bc.soloMuted {
			vol = 0
		}
		// Apply quick 2-step ramp: halfway then target, spaced by ~2ms
		_ = bc.dispatcher.Enqueue(queue.Func(func(ctx context.Context) error {
			if mix == nil {
				return nil
			}
			// halfway toward the target
			cur, _ := node.GetMixerVolume(mix, 0)
			target := vol
			mid := (cur + target) / 2
			_ = node.SetMixerVolume(mix, mid, 0)
			// small sleep; context-aware
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Millisecond):
			}
			return node.SetMixerVolume(mix, target, 0)
		}))
		return nil
	}
	return bc.applyEffectiveVolume()
}

// GetMute reports mute state approximately by checking if volume == 0.0.
func (bc *BaseChannel) GetMute() (bool, error) {
	if bc.released {
		return false, fmt.Errorf("channel has been released")
	}
	return bc.userMuted, nil
}

// SetPan sets stereo balance on input bus 0 of the channel mixer.
func (bc *BaseChannel) SetPan(pan float32) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return fmt.Errorf("output mixer not available")
	}
	return node.SetMixerPan(bc.outputMixer, pan, 0)
}

// GetPan reads stereo balance from input bus 0 of the channel mixer.
func (bc *BaseChannel) GetPan() (float32, error) {
	if bc.released {
		return 0, fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return 0, fmt.Errorf("output mixer not available")
	}
	return node.GetMixerPan(bc.outputMixer, 0)
}

// GetPluginChain returns the channel's plugin chain
func (bc *BaseChannel) GetPluginChain() *pluginchain.PluginChain {
	return bc.pluginChain
}

// AddEffect adds an effect to the channel's plugin chain
func (bc *BaseChannel) AddEffect(plugin *plugins.Plugin) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.pluginChain == nil {
		return fmt.Errorf("plugin chain not available")
	}

	if err := bc.pluginChain.AddEffect(plugin); err != nil {
		return err
	}
	return bc.ConnectPluginChainToMixer()
}

// AddEffectFromPluginInfo adds an insert effect by introspecting via PluginInfo.
func (bc *BaseChannel) AddEffectFromPluginInfo(pluginInfo plugins.PluginInfo) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.pluginChain == nil {
		return fmt.Errorf("plugin chain not available")
	}

	if err := bc.pluginChain.AddEffectFromPluginInfo(pluginInfo); err != nil {
		return err
	}
	return bc.ConnectPluginChainToMixer()
}

// GetOutputNode returns the output mixer node for external routing
func (bc *BaseChannel) GetOutputNode() unsafe.Pointer {
	if bc.outputMixer == nil {
		return nil
	}
	return bc.outputMixer
}

// GetInputNode returns the node to which sources should connect: the plugin
// chain input when the chain has effects, else the channel's mixer.
func (bc *BaseChannel) GetInputNode() unsafe.Pointer {
	// If we have effects in the plugin chain, input goes to the chain
	if bc.pluginChain != nil && !bc.pluginChain.IsEmpty() {
		inputNode, _ := bc.pluginChain.GetInputNode()
		return inputNode
	}

	// Otherwise, input goes directly to output mixer
	if bc.outputMixer != nil {
		return bc.outputMixer
	}

	return nil
}

// CreateSend defines a post-fader send by default (backward compatible).
func (bc *BaseChannel) CreateSend(name string, destination Channel, level float32) error {
	return bc.CreateSendWithMode(name, destination, level, PostFader)
}

// CreateSendWithMode defines a named send with explicit pre/post-fader mode.
func (bc *BaseChannel) CreateSendWithMode(name string, destination Channel, level float32, mode SendMode) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if name == "" {
		return fmt.Errorf("send name cannot be empty")
	}
	if destination == nil {
		return fmt.Errorf("send destination cannot be nil")
	}
	if level < 0.0 || level > 1.0 {
		return fmt.Errorf("send level must be between 0.0 and 1.0")
	}

	// Check if send already exists
	bc.sendsMu.RLock()
	_, exists := bc.sends[name]
	bc.sendsMu.RUnlock()
	if exists {
		return fmt.Errorf("send '%s' already exists", name)
	}

	bc.sendsMu.Lock()
	bc.sends[name] = &Send{Name: name, Destination: destination, Level: level, Mute: false, Mode: mode, prev: level}
	bc.sendsMu.Unlock()

	return nil
}

// Aux send helpers: one well-known post-fader send per input channel
const auxSendName = "aux"

// CreateAuxSend enables a post-fader send from an Input channel to the Aux bus.
func (bc *BaseChannel) CreateAuxSend(level float32) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.kind != ChannelInput {
		return fmt.Errorf("aux send not supported for this channel type")
	}
	if level < 0.0 || level > 1.0 {
		return fmt.Errorf("send level must be between 0.0 and 1.0")
	}
	bc.sendsMu.RLock()
	_, exists := bc.sends[auxSendName]
	bc.sendsMu.RUnlock()
	if exists {
		return fmt.Errorf("aux send already exists")
	}
	bc.sendsMu.Lock()
	bc.sends[auxSendName] = &Send{Name: auxSendName, Level: level, Mute: false, Mode: PostFader, prev: level}
	bc.sendsMu.Unlock()
	return nil
}

// ConnectAux connects the channel's Aux send to the provided Aux bus (allocates next input).
func (bc *BaseChannel) ConnectAux(aux *Aux) (int, error) {
	if bc.engineInstance == nil {
		return -1, fmt.Errorf("engine instance not available")
	}
	if aux == nil || aux.bus == nil {
		return -1, fmt.Errorf("aux bus not initialized")
	}
	// Ensure aux send exists
	bc.sendsMu.RLock()
	_, exists := bc.sends[auxSendName]
	bc.sendsMu.RUnlock()
	if !exists {
		if err := bc.CreateAuxSend(0.0); err != nil {
			return -1, err
		}
	}
	idx := aux.bus.NextInput()
	if err := bc.ConnectSendToBus(bc.engineInstance, auxSendName, aux.bus.mixer, idx); err != nil {
		return -1, err
	}
	return idx, nil
}

// SetAuxSendLevel updates the Aux send level if present.
func (bc *BaseChannel) SetAuxSendLevel(level float32) error {
	return bc.SetSendLevel(auxSendName, level)
}

// DisconnectAux disconnects the Aux send if present.
func (bc *BaseChannel) DisconnectAux() error {
	if bc.engineInstance == nil {
		return fmt.Errorf("engine instance not available")
	}
	return bc.DisconnectSend(bc.engineInstance, auxSendName)
}

// SetSendLevel adjusts the level of an auxiliary send
func (bc *BaseChannel) SetSendLevel(sendName string, level float32) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if level < 0.0 || level > 1.0 {
		return fmt.Errorf("send level must be between 0.0 and 1.0")
	}

	bc.sendsMu.RLock()
	send, exists := bc.sends[sendName]
	bc.sendsMu.RUnlock()
	if !exists {
		return fmt.Errorf("send '%s' does not exist", sendName)
	}

	// Update logical state and remember previous non-zero
	if level > 0 {
		send.prev = level
	}
	send.Level = level

	// If wired, apply to per-send mixer (respect mute)
	if send.mixer != nil {
		newVol := level
		if send.Mute {
			newVol = 0
		}
		if err := node.SetMixerVolume(send.mixer, newVol, 0); err != nil {
			return fmt.Errorf("set send volume: %w", err)
		}
	}
	return nil
}

// SetSendMute mutes/unmutes a named send by driving its per-send mixer volume.
func (bc *BaseChannel) SetSendMute(sendName string, muted bool) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	bc.sendsMu.RLock()
	send, exists := bc.sends[sendName]
	bc.sendsMu.RUnlock()
	if !exists {
		return fmt.Errorf("send '%s' does not exist", sendName)
	}
	send.Mute = muted
	if send.mixer != nil {
		vol := send.Level
		if muted {
			vol = 0
		}
		if err := node.SetMixerVolume(send.mixer, vol, 0); err != nil {
			return fmt.Errorf("set send mute: %w", err)
		}
	}
	return nil
}

// GetSends returns all auxiliary sends for this channel
func (bc *BaseChannel) GetSends() map[string]*Send {
	bc.sendsMu.RLock()
	defer bc.sendsMu.RUnlock()
	sendsCopy := make(map[string]*Send, len(bc.sends))
	for name, send := range bc.sends {
		sendsCopy[name] = send
	}
	return sendsCopy
}

// GetSendLevel returns the current logical level of a named send.
func (bc *BaseChannel) GetSendLevel(sendName string) (float32, error) {
	bc.sendsMu.RLock()
	defer bc.sendsMu.RUnlock()
	send, ok := bc.sends[sendName]
	if !ok {
		return 0, fmt.Errorf("send '%s' does not exist", sendName)
	}
	return send.Level, nil
}

// GetSendMute returns the current mute state of a named send.
func (bc *BaseChannel) GetSendMute(sendName string) (bool, error) {
	bc.sendsMu.RLock()
	defer bc.sendsMu.RUnlock()
	send, ok := bc.sends[sendName]
	if !ok {
		return false, fmt.Errorf("send '%s' does not exist", sendName)
	}
	return send.Mute, nil
}

// ConnectSendToBus wires a previously defined send to a destination bus input.
// Source node depends on mode (PostFader uses channel mixer; PreFader uses
// plugin chain output when available). This call does not currently apply per-
// send gain; that can be provided by the destination bus or a future gain node.
func (bc *BaseChannel) ConnectSendToBus(eng *engine.Engine, sendName string, busInput unsafe.Pointer, toBus int) error {
	bc.routeMu.Lock()
	defer bc.routeMu.Unlock()
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	if busInput == nil {
		return fmt.Errorf("bus input pointer cannot be nil")
	}
	bc.sendsMu.RLock()
	send, ok := bc.sends[sendName]
	bc.sendsMu.RUnlock()
	if !ok {
		return fmt.Errorf("send '%s' does not exist", sendName)
	}
	// We will wire a per-send mixer to control level/mute even if current volume is 0.
	// Determine source node per mode
	var source unsafe.Pointer
	if send.Mode == PostFader {
		source = bc.outputMixer
		// Ensure attached
		if installed, err := node.IsInstalledOnEngine(source); err == nil && !installed {
			if bc.dispatcher != nil {
				_ = bc.dispatcher.Attach(source)
			} else if err := eng.Attach(source); err != nil {
				return fmt.Errorf("attach mixer failed: %w", err)
			}
		}
	} else {
		// PreFader: prefer plugin chain output when available
		if bc.pluginChain != nil && !bc.pluginChain.IsEmpty() {
			outPtr, err := bc.pluginChain.GetOutputNode()
			if err != nil {
				return fmt.Errorf("get chain output: %w", err)
			}
			source = outPtr
		} else {
			source = bc.outputMixer
			if installed, err := node.IsInstalledOnEngine(source); err == nil && !installed {
				if bc.dispatcher != nil {
					_ = bc.dispatcher.Attach(source)
				} else if err := eng.Attach(source); err != nil {
					return fmt.Errorf("attach mixer failed: %w", err)
				}
			}
		}
	}
	if source == nil {
		return fmt.Errorf("send source node is nil")
	}

	// Ensure we have a per-send mixer to apply level/mute
	if send.mixer == nil {
		m, err := node.CreateMixer()
		if err != nil || m == nil {
			return fmt.Errorf("create send mixer: %v", err)
		}
		// Attach and set initial volume (respect mute)
		if bc.dispatcher != nil {
			_ = bc.dispatcher.Attach(m)
		} else if err := eng.Attach(m); err != nil {
			return fmt.Errorf("attach send mixer: %w", err)
		}
		initVol := send.Level
		if send.Mute {
			initVol = 0
		}
		if err := node.SetMixerVolume(m, initVol, 0); err != nil {
			return fmt.Errorf("init send mixer volume: %w", err)
		}
		send.mixer = m
	} else {
		// If destination unchanged, just ensure mixer volume reflects current state
		if send.busInput == busInput && send.busIndex == toBus {
			vol := send.Level
			if send.Mute {
				vol = 0
			}
			_ = node.SetMixerVolume(send.mixer, vol, 0)
			return nil
		}
		// Rewire: disconnect previous destination input and our mixer input
		if send.busInput != nil {
			if bc.dispatcher != nil {
				_ = bc.dispatcher.DisconnectNodeInput(send.busInput, send.busIndex)
			} else {
				_ = eng.DisconnectNodeInput(send.busInput, send.busIndex)
			}
		}
		if bc.dispatcher != nil {
			_ = bc.dispatcher.DisconnectNodeInput(send.mixer, 0)
		} else {
			_ = eng.DisconnectNodeInput(send.mixer, 0)
		}
	}

	// Wire source -> send.mixer -> bus
	if bc.dispatcher != nil {
		_ = bc.dispatcher.Connect(source, send.mixer, 0, 0)
		_ = bc.dispatcher.Connect(send.mixer, busInput, 0, toBus)
	} else {
		if err := eng.Connect(source, send.mixer, 0, 0); err != nil {
			return fmt.Errorf("connect source->send mixer failed: %w", err)
		}
		if err := eng.Connect(send.mixer, busInput, 0, toBus); err != nil {
			return fmt.Errorf("connect send mixer->bus failed: %w", err)
		}
	}
	send.busInput = busInput
	send.busIndex = toBus
	return nil
}

// ConnectSendTo wraps ConnectSendToBus using a Bus helper, auto-allocating the
// next input on the bus. Uses the channel's engine instance.
func (bc *BaseChannel) ConnectSendTo(sendName string, bus *Bus) (int, error) {
	if bc.engineInstance == nil {
		return -1, fmt.Errorf("engine instance not available")
	}
	if bus == nil || bus.mixer == nil {
		return -1, fmt.Errorf("bus not initialized")
	}
	idx := bus.NextInput()
	if err := bc.ConnectSendToBus(bc.engineInstance, sendName, bus.mixer, idx); err != nil {
		return -1, err
	}
	return idx, nil
}

// CreateAndConnectSend creates a send and connects it to the given Bus in one call.
func (bc *BaseChannel) CreateAndConnectSend(name string, dest Channel, bus *Bus, level float32, mode SendMode) (int, error) {
	if err := bc.CreateSendWithMode(name, dest, level, mode); err != nil {
		return -1, err
	}
	return bc.ConnectSendTo(name, bus)
}

// DisconnectSend disconnects and releases resources for a named send.
func (bc *BaseChannel) DisconnectSend(eng *engine.Engine, sendName string) error {
	bc.routeMu.Lock()
	defer bc.routeMu.Unlock()
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	bc.sendsMu.RLock()
	send, ok := bc.sends[sendName]
	bc.sendsMu.RUnlock()
	if !ok {
		return fmt.Errorf("send '%s' does not exist", sendName)
	}
	// Disconnect bus input if known
	if send.busInput != nil {
		if bc.dispatcher != nil {
			_ = bc.dispatcher.DisconnectNodeInput(send.busInput, send.busIndex)
		} else {
			_ = eng.DisconnectNodeInput(send.busInput, send.busIndex)
		}
	}
	// Release per-send mixer if allocated
	if send.mixer != nil {
		node.ReleaseMixer(send.mixer)
		send.mixer = nil
	}
	// Keep logical send entry so it can be reconnected later
	return nil
}

// RemoveSend disconnects and removes a named send from the channel.
func (bc *BaseChannel) RemoveSend(eng *engine.Engine, sendName string) error {
	bc.routeMu.Lock()
	defer bc.routeMu.Unlock()
	if err := bc.DisconnectSend(eng, sendName); err != nil {
		return err
	}
	bc.sendsMu.Lock()
	delete(bc.sends, sendName)
	bc.sendsMu.Unlock()
	return nil
}

// Release releases all resources used by the base channel
func (bc *BaseChannel) Release() {
	if bc.released {
		return
	}

	// Release plugin chain
	if bc.pluginChain != nil {
		bc.pluginChain.Release()
		bc.pluginChain = nil
	}

	// Release output mixer
	if bc.outputMixer != nil {
		node.ReleaseMixer(bc.outputMixer)
		bc.outputMixer = nil
	}

	// Phase invert node removed in Core v1

	// Clear sends
	bc.sends = nil

	// Unregister from solo manager
	DefaultSolo.Unregister(bc)

	// If we own a dispatcher, close it
	if bc.dispatcherOwned && bc.dispatcher != nil {
		bc.dispatcher.Close()
		bc.dispatcher = nil
		bc.dispatcherOwned = false
	}

	bc.released = true
}

// IsReleased returns true if the channel has been released
func (bc *BaseChannel) IsReleased() bool {
	return bc.released
}

// Summary returns a brief summary of the base channel
func (bc *BaseChannel) Summary() string {
	if bc.released {
		return fmt.Sprintf("Channel '%s': RELEASED", bc.name)
	}

	effectCount := 0
	if bc.pluginChain != nil {
		effectCount = bc.pluginChain.GetEffectCount()
	}

	sendCount := len(bc.sends)

	return fmt.Sprintf("Channel '%s': %d effects, %d sends",
		firstNonEmpty(bc.displayName, bc.name), effectCount, sendCount)
}

// ConnectPluginChainToMixer connects the plugin chain output to the channel's
// mixer (bus 0 -> 0). Safe to call when the chain is empty (no-op).
func (bc *BaseChannel) ConnectPluginChainToMixer() error {
	bc.routeMu.Lock()
	defer bc.routeMu.Unlock()
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.pluginChain == nil || bc.outputMixer == nil {
		return fmt.Errorf("plugin chain or output mixer not available")
	}

	// If plugin chain is empty, nothing to connect
	if bc.pluginChain.IsEmpty() {
		return nil
	}

	// Ensure output mixer is attached to engine before connecting
	if bc.engineInstance == nil {
		return fmt.Errorf("engine instance not available")
	}

	// Best-effort: attach mixer if not already installed on engine
	if installed, err := node.IsInstalledOnEngine(bc.outputMixer); err == nil && !installed {
		if bc.dispatcher != nil {
			_ = bc.dispatcher.Attach(bc.outputMixer)
		} else if err := bc.engineInstance.Attach(bc.outputMixer); err != nil {
			return fmt.Errorf("attach mixer failed: %w", err)
		}
	}

	// Obtain the chain output node and connect to channel mixer (bus 0 → 0)
	outPtr, err := bc.pluginChain.GetOutputNode()
	if err != nil {
		return fmt.Errorf("get chain output: %w", err)
	}
	if outPtr == nil {
		return fmt.Errorf("chain output node is nil")
	}
	// Idempotency: ensure previous connection on mixer input bus 0 is cleared
	if bc.dispatcher != nil {
		_ = bc.dispatcher.DisconnectNodeInput(bc.outputMixer, 0)
	} else {
		_ = bc.engineInstance.DisconnectNodeInput(bc.outputMixer, 0)
	}

	// Phase invert path removed in Core v1: direct connect only

	// Default: direct connect
	// Ensure clean direct connection
	if bc.dispatcher != nil {
		_ = bc.dispatcher.Connect(outPtr, bc.outputMixer, 0, 0)
		return nil
	}
	if err := bc.engineInstance.Connect(outPtr, bc.outputMixer, 0, 0); err != nil {
		return fmt.Errorf("connect chain→mixer failed: %w", err)
	}
	return nil
}

// ConnectToMaster attaches the channel mixer if needed and connects it to the
// engine's main mixer (bus 0 -> 0). It tracks connection state to prevent dupes.
func (bc *BaseChannel) ConnectToMaster(eng *engine.Engine) error {
	bc.routeMu.Lock()
	defer bc.routeMu.Unlock()
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return fmt.Errorf("channel output mixer not available")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	if bc.connectedToMaster {
		// Idempotent: already connected
		return nil
	}

	// Ensure our mixer is attached to the engine
	if installed, err := node.IsInstalledOnEngine(bc.outputMixer); err == nil && !installed {
		if bc.dispatcher != nil {
			_ = bc.dispatcher.Attach(bc.outputMixer)
		} else if err := eng.Attach(bc.outputMixer); err != nil {
			return fmt.Errorf("attach mixer failed: %w", err)
		}
	}

	// Get main mixer node from engine
	mainMixerPtr, err := eng.MainMixerNode()
	if err != nil {
		return fmt.Errorf("failed to get main mixer node from engine: %w", err)
	}
	if mainMixerPtr == nil {
		return fmt.Errorf("failed to get main mixer node from engine: returned nil pointer")
	}

	// Connect our output mixer to the main mixer (bus 0 to bus 0 for stereo)
	if bc.dispatcher != nil {
		_ = bc.dispatcher.Connect(bc.outputMixer, mainMixerPtr, 0, 0)
		bc.connectedToMaster = true
		return nil
	}
	err = eng.Connect(bc.outputMixer, mainMixerPtr, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to connect channel to main mixer: %w", err)
	}

	bc.connectedToMaster = true
	return nil
}

// DisconnectFromMaster disconnects the main mixer's input bus 0 and updates
// connection state. This supports dynamic re-routing and performance tuning.
func (bc *BaseChannel) DisconnectFromMaster(eng *engine.Engine) error {
	bc.routeMu.Lock()
	defer bc.routeMu.Unlock()
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	if !bc.connectedToMaster {
		// Idempotent: already disconnected
		return nil
	}

	// Get main mixer node from engine
	mainMixerPtr, err := eng.MainMixerNode()
	if err != nil {
		return fmt.Errorf("failed to get main mixer node from engine: %w", err)
	}
	if mainMixerPtr == nil {
		return fmt.Errorf("failed to get main mixer node from engine: returned nil pointer")
	}

	// Disconnect the main mixer's input bus 0 (where our channel is connected)
	if bc.dispatcher != nil {
		_ = bc.dispatcher.DisconnectNodeInput(mainMixerPtr, 0)
	} else {
		err = eng.DisconnectNodeInput(mainMixerPtr, 0)
	}
	if err != nil {
		return fmt.Errorf("failed to disconnect channel from main mixer: %w", err)
	}

	bc.connectedToMaster = false
	return nil
}

// IsConnectedToMaster returns true if this channel is currently connected to master output
func (bc *BaseChannel) IsConnectedToMaster() bool {
	return bc.connectedToMaster && !bc.released
}

// ConnectToBus connects this channel to a stereo bus mixer
// This enables routing to mix buses, effects returns, etc.
func (bc *BaseChannel) ConnectToBus(eng *engine.Engine, busInput unsafe.Pointer, fromBus, toBus int) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return fmt.Errorf("channel output mixer not available")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	if busInput == nil {
		return fmt.Errorf("bus input pointer cannot be nil")
	}

	// Connect our output mixer to the specified bus input
	if bc.dispatcher != nil {
		return bc.dispatcher.Connect(bc.outputMixer, busInput, fromBus, toBus)
	}
	if err := eng.Connect(bc.outputMixer, busInput, fromBus, toBus); err != nil {
		return fmt.Errorf("failed to connect channel to bus: %w", err)
	}

	return nil
}

// Phase inversion was removed in Core v1.
// Phase inversion removed in Core v1

// internal: applyEffectiveVolume computes and applies volume based on user and solo state
func (bc *BaseChannel) applyEffectiveVolume() error {
	if bc.outputMixer == nil {
		return fmt.Errorf("output mixer not available")
	}
	vol := bc.lastVolume
	if bc.userMuted || bc.soloMuted {
		vol = 0
	}
	return node.SetMixerVolume(bc.outputMixer, vol, 0)
}

// markSoloMuted is called by SoloManager to set/clear solo-induced mute
func (bc *BaseChannel) markSoloMuted(m bool) {
	bc.soloMuted = m
	_ = bc.applyEffectiveVolume()
}

// SetSolo toggles solo for this channel using the DefaultSolo manager.
func (bc *BaseChannel) SetSolo(on bool) {
	DefaultSolo.SetSolo(bc, on)
}

// IsSoloed reports whether this channel is currently soloed via the manager.
func (bc *BaseChannel) IsSoloed() bool {
	return DefaultSolo.IsSoloed(bc)
}

// EnableOutputMetering installs or removes a tap on the channel's output mixer bus 0.
func (bc *BaseChannel) EnableOutputMetering(eng *engine.Engine, enable bool) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	bc.meterMu.Lock()
	defer bc.meterMu.Unlock()
	if enable {
		if bc.meterTap != nil && bc.meterTap.IsInstalled() {
			return nil
		}
		if installed, err := node.IsInstalledOnEngine(bc.outputMixer); err == nil && !installed {
			if bc.dispatcher != nil {
				_ = bc.dispatcher.Attach(bc.outputMixer)
			} else if err := eng.Attach(bc.outputMixer); err != nil {
				return fmt.Errorf("attach mixer for meter: %w", err)
			}
		}
		t, err := tap.InstallTap(eng.Ptr(), bc.outputMixer, 0)
		if err != nil {
			return err
		}
		bc.meterTap = t
		return nil
	}
	if bc.meterTap != nil {
		_ = bc.meterTap.Remove()
		bc.meterTap = nil
	}
	return nil
}

// OutputRMS returns the current RMS level from the output meter tap.
func (bc *BaseChannel) OutputRMS() (float64, error) {
	bc.meterMu.RLock()
	defer bc.meterMu.RUnlock()
	if bc.meterTap == nil || !bc.meterTap.IsInstalled() {
		return 0, fmt.Errorf("output metering not enabled")
	}
	m, err := bc.meterTap.GetMetrics()
	if err != nil {
		return 0, err
	}
	return m.RMS, nil
}

// EnableSendMetering installs/removes a tap on the per-send mixer output.
func (bc *BaseChannel) EnableSendMetering(eng *engine.Engine, sendName string, enable bool) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	bc.sendsMu.RLock()
	send, ok := bc.sends[sendName]
	bc.sendsMu.RUnlock()
	if !ok {
		return fmt.Errorf("send '%s' does not exist", sendName)
	}
	bc.meterMu.Lock()
	defer bc.meterMu.Unlock()
	if enable {
		if send.mixer == nil {
			return fmt.Errorf("send '%s' is not connected", sendName)
		}
		if _, exists := bc.sendMeters[sendName]; exists {
			return nil
		}
		if installed, err := node.IsInstalledOnEngine(send.mixer); err == nil && !installed {
			if bc.dispatcher != nil {
				_ = bc.dispatcher.Attach(send.mixer)
			} else if err := eng.Attach(send.mixer); err != nil {
				return fmt.Errorf("attach send mixer for meter: %w", err)
			}
		}
		t, err := tap.InstallTap(eng.Ptr(), send.mixer, 0)
		if err != nil {
			return err
		}
		bc.sendMeters[sendName] = t
		return nil
	}
	if t, ok := bc.sendMeters[sendName]; ok {
		_ = t.Remove()
		delete(bc.sendMeters, sendName)
	}
	return nil
}

// SendRMS returns the current RMS level for a metered send.
func (bc *BaseChannel) SendRMS(sendName string) (float64, error) {
	bc.meterMu.RLock()
	defer bc.meterMu.RUnlock()
	t, ok := bc.sendMeters[sendName]
	if !ok || t == nil || !t.IsInstalled() {
		return 0, fmt.Errorf("send metering not enabled for '%s'", sendName)
	}
	m, err := t.GetMetrics()
	if err != nil {
		return 0, err
	}
	return m.RMS, nil
}

// helper: pick first non-empty string
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
