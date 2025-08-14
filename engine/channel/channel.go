// Package channel provides the base interface and common functionality
// shared by all audio channels (input channels, mix buses, etc.)
package channel

import (
	"fmt"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
	"github.com/shaban/macaudio/avaudio/pluginchain"
	"github.com/shaban/macaudio/plugins"
)

// Channel represents the common interface shared by all audio channels
type Channel interface {
	// Basic Properties
	GetName() string
	SetName(string)

	// Audio Control
	SetVolume(volume float32) error
	GetVolume() (float32, error)
	SetMute(muted bool) error
	GetMute() (bool, error)

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

// Send represents an auxiliary send to another channel or bus
type Send struct {
	Name        string
	Destination Channel
	Level       float32
	Mute        bool
}

// BaseChannel provides common functionality for all channel types
type BaseChannel struct {
	name              string
	enginePtr         unsafe.Pointer
	engineInstance    *engine.Engine // Reference to engine for accessing AudioSpec
	pluginChain       *pluginchain.PluginChain
	outputMixer       unsafe.Pointer   // For volume and mute control (Node)
	sends             map[string]*Send // Auxiliary sends
	released          bool
	connectedToMaster bool // Track master connection state
}

// BaseChannelConfig holds configuration for creating a base channel
type BaseChannelConfig struct {
	Name           string
	EnginePtr      unsafe.Pointer // AVAudioEngine pointer from avaudio/engine package
	EngineInstance *engine.Engine // Engine instance for accessing AudioSpec
}

// NewBaseChannel creates a new base channel with common functionality
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
	pluginChain := pluginchain.NewPluginChain(pluginchain.ChainConfig{
		Name:      config.Name + " Chain",
		EnginePtr: config.EnginePtr,
	})

	// Create output mixer for volume and mute control
	outputMixer, err := node.CreateMixer()
	if err != nil || outputMixer == nil {
		return nil, fmt.Errorf("failed to create output mixer for channel %s: %v", config.Name, err)
	}

	return &BaseChannel{
		name:              config.Name,
		enginePtr:         config.EnginePtr,
		engineInstance:    config.EngineInstance,
		pluginChain:       pluginChain,
		outputMixer:       outputMixer,
		sends:             make(map[string]*Send),
		released:          false,
		connectedToMaster: false,
	}, nil
}

// GetName returns the channel name
func (bc *BaseChannel) GetName() string {
	return bc.name
}

// SetName updates the channel name
func (bc *BaseChannel) SetName(name string) {
	bc.name = name
	if bc.pluginChain != nil {
		bc.pluginChain.SetName(name + " Chain")
	}
}

// GetAudioSpec returns the audio specifications from the engine
// This allows channels to inherit the engine's audio format settings
func (bc *BaseChannel) GetAudioSpec() engine.AudioSpec {
	if bc.engineInstance == nil {
		// Return empty spec if no engine instance (shouldn't happen)
		return engine.AudioSpec{}
	}
	return bc.engineInstance.GetSpec()
}

// SetVolume sets the channel output volume (0.0 to 1.0)
func (bc *BaseChannel) SetVolume(volume float32) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return fmt.Errorf("output mixer not available")
	}

	return node.SetMixerVolume(bc.outputMixer, volume, 0)
}

// GetVolume gets the channel output volume
func (bc *BaseChannel) GetVolume() (float32, error) {
	if bc.released {
		return 0, fmt.Errorf("channel has been released")
	}
	if bc.outputMixer == nil {
		return 0, fmt.Errorf("output mixer not available")
	}

	return node.GetMixerVolume(bc.outputMixer, 0)
}

// SetMute sets the channel mute state
func (bc *BaseChannel) SetMute(muted bool) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}

	// Mute by setting volume to 0, unmute by restoring previous volume
	if muted {
		return bc.SetVolume(0.0)
	} else {
		// For now, unmute sets to 0.8 - in a real implementation you'd store the previous volume
		return bc.SetVolume(0.8)
	}
}

// GetMute gets the channel mute state (approximated by checking if volume is 0)
func (bc *BaseChannel) GetMute() (bool, error) {
	volume, err := bc.GetVolume()
	if err != nil {
		return false, err
	}
	return volume == 0.0, nil
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

	return bc.pluginChain.AddEffect(plugin)
}

// AddEffectFromPluginInfo adds an effect using plugin info
func (bc *BaseChannel) AddEffectFromPluginInfo(pluginInfo plugins.PluginInfo) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if bc.pluginChain == nil {
		return fmt.Errorf("plugin chain not available")
	}

	return bc.pluginChain.AddEffectFromPluginInfo(pluginInfo)
}

// GetOutputNode returns the output mixer node for external routing
func (bc *BaseChannel) GetOutputNode() unsafe.Pointer {
	if bc.outputMixer == nil {
		return nil
	}
	return bc.outputMixer
}

// GetInputNode returns the input of the plugin chain, or the output mixer if no effects
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

// CreateSend creates an auxiliary send to another channel or bus
func (bc *BaseChannel) CreateSend(name string, destination Channel, level float32) error {
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
	if _, exists := bc.sends[name]; exists {
		return fmt.Errorf("send '%s' already exists", name)
	}

	bc.sends[name] = &Send{
		Name:        name,
		Destination: destination,
		Level:       level,
		Mute:        false,
	}

	return nil
}

// SetSendLevel adjusts the level of an auxiliary send
func (bc *BaseChannel) SetSendLevel(sendName string, level float32) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if level < 0.0 || level > 1.0 {
		return fmt.Errorf("send level must be between 0.0 and 1.0")
	}

	send, exists := bc.sends[sendName]
	if !exists {
		return fmt.Errorf("send '%s' does not exist", sendName)
	}

	send.Level = level
	return nil
}

// GetSends returns all auxiliary sends for this channel
func (bc *BaseChannel) GetSends() map[string]*Send {
	sendsCopy := make(map[string]*Send)
	for name, send := range bc.sends {
		sendsCopy[name] = send
	}
	return sendsCopy
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

	// Clear sends
	bc.sends = nil

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
		bc.name, effectCount, sendCount)
}

// ConnectPluginChainToMixer connects the plugin chain output to the output mixer
// This should be called by concrete channel implementations after setup
func (bc *BaseChannel) ConnectPluginChainToMixer() error {
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
		if err := bc.engineInstance.Attach(bc.outputMixer); err != nil {
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

	if err := bc.engineInstance.Connect(outPtr, bc.outputMixer, 0, 0); err != nil {
		return fmt.Errorf("connect chain→mixer failed: %w", err)
	}
	return nil
}

// ConnectToMaster connects this channel to the engine's main mixer output
// This enables the channel to be heard through the master output
// Note: Requires the actual Engine instance since enginePtr only contains the C engine pointer
func (bc *BaseChannel) ConnectToMaster(eng *engine.Engine) error {
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
		return fmt.Errorf("channel is already connected to master")
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
	err = eng.Connect(bc.outputMixer, mainMixerPtr, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to connect channel to main mixer: %w", err)
	}

	bc.connectedToMaster = true
	return nil
}

// DisconnectFromMaster disconnects this channel from the engine's main mixer
// This is essential for dynamic routing changes and performance optimization
func (bc *BaseChannel) DisconnectFromMaster(eng *engine.Engine) error {
	if bc.released {
		return fmt.Errorf("channel has been released")
	}
	if eng == nil {
		return fmt.Errorf("engine instance cannot be nil")
	}
	if !bc.connectedToMaster {
		return fmt.Errorf("channel is not connected to master")
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
	err = eng.DisconnectNodeInput(mainMixerPtr, 0)
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
	err := eng.Connect(bc.outputMixer, busInput, fromBus, toBus)
	if err != nil {
		return fmt.Errorf("failed to connect channel to bus: %w", err)
	}

	return nil
}
