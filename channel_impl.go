package macaudio

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/google/uuid"
)

// BaseChannel provides common functionality for all channel types
type BaseChannel struct {
	// UUID hybrid pattern: struct uses uuid.UUID, maps use string keys
	id          uuid.UUID
	name        string
	channelType ChannelType
	engine      *Engine

	// Audio processing
	volume float32
	pan    float32
	muted  bool

	// Plugin chain
	pluginChain *PluginChain

	// AVFoundation integration
	outputMixer unsafe.Pointer // AVAudioMixerNode for this channel

	// Connections
	mu          sync.RWMutex
	connections []Connection
	isRunning   bool
}

// NewBaseChannel creates a new base channel with common initialization
func NewBaseChannel(name string, channelType ChannelType, engine *Engine) *BaseChannel {
	return &BaseChannel{
		id:          uuid.New(), // Generate new UUID
		name:        name,
		channelType: channelType,
		engine:      engine,
		volume:      1.0, // Default volume
		pan:         0.0, // Center pan
		muted:       false,
		pluginChain: NewPluginChain(),
		connections: make([]Connection, 0),
		isRunning:   false,
	}
}

// GetID returns the channel UUID (hybrid pattern)
func (bc *BaseChannel) GetID() uuid.UUID {
	return bc.id
}

// GetIDString returns the channel UUID as string for map keys
func (bc *BaseChannel) GetIDString() string {
	return bc.id.String()
}

// GetName returns the channel name
func (bc *BaseChannel) GetName() string {
	return bc.name
}

// SetName sets the channel name
func (bc *BaseChannel) SetName(name string) {
	bc.name = name
}

// GetType returns the channel type
func (bc *BaseChannel) GetType() ChannelType {
	return bc.channelType
}

// Start starts the channel
func (bc *BaseChannel) Start() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.isRunning {
		return nil // Already running
	}

	bc.isRunning = true
	return nil
}

// Stop stops the channel
func (bc *BaseChannel) Stop() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if !bc.isRunning {
		return nil // Already stopped
	}

	bc.isRunning = false
	return nil
}

// IsRunning returns whether the channel is running
func (bc *BaseChannel) IsRunning() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.isRunning
}

// ConnectTo connects this channel to another channel
func (bc *BaseChannel) ConnectTo(target Channel, bus int) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	connection := Connection{
		SourceChannel: bc.GetIDString(), // Convert UUID to string
		TargetChannel: target.GetIDString(),
		SourceBus:     0, // Most channels have single output bus
		TargetBus:     bus,
	}

	bc.connections = append(bc.connections, connection)
	return nil
}

// DisconnectFrom disconnects this channel from another channel
func (bc *BaseChannel) DisconnectFrom(target Channel, bus int) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	targetID := target.GetIDString() // Get string representation for comparison
	for i, conn := range bc.connections {
		if conn.TargetChannel == targetID && conn.TargetBus == bus {
			// Remove connection
			copy(bc.connections[i:], bc.connections[i+1:])
			bc.connections = bc.connections[:len(bc.connections)-1]
			return nil
		}
	}

	return fmt.Errorf("connection to %s (bus %d) not found", targetID, bus)
}

// GetConnections returns all connections from this channel
func (bc *BaseChannel) GetConnections() []Connection {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	connections := make([]Connection, len(bc.connections))
	copy(connections, bc.connections)
	return connections
}

// GetPluginChain returns the plugin chain
func (bc *BaseChannel) GetPluginChain() *PluginChain {
	return bc.pluginChain
}

// AddPlugin adds a plugin to the channel's plugin chain
func (bc *BaseChannel) AddPlugin(blueprint PluginBlueprint, position int) (*PluginInstance, error) {
	return bc.pluginChain.AddPlugin(blueprint, position)
}

// RemovePlugin removes a plugin from the channel's plugin chain
func (bc *BaseChannel) RemovePlugin(instanceID string) error {
	return bc.pluginChain.RemovePlugin(instanceID)
}

// SetVolume sets the channel volume (0.0 to 1.0)
func (bc *BaseChannel) SetVolume(volume float32) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0.0 and 1.0")
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.volume = volume

	// Apply to actual output mixer node if available
	if bc.outputMixer != nil {
		// Note: Volume control requires AVAudioMixerNode-specific bindings
		// For now, we store the value. Future enhancement: implement mixer volume control
	}

	return nil
}

// GetVolume returns the current channel volume
func (bc *BaseChannel) GetVolume() (float32, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.volume, nil
}

// SetPan sets the channel pan (-1.0 to 1.0, where -1.0 is full left, 1.0 is full right)
func (bc *BaseChannel) SetPan(pan float32) error {
	if pan < -1.0 || pan > 1.0 {
		return fmt.Errorf("pan must be between -1.0 and 1.0")
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.pan = pan

	// Apply to actual output mixer node if available
	if bc.outputMixer != nil {
		// Note: Pan control requires AVAudioMixerNode-specific bindings
		// For now, we store the value. Future enhancement: implement mixer pan control
	}

	return nil
}

// GetPan returns the current channel pan
func (bc *BaseChannel) GetPan() (float32, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.pan, nil
}

// SetMute sets the channel mute state
func (bc *BaseChannel) SetMute(muted bool) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.muted = muted

	// Apply to actual output mixer node if available
	if bc.outputMixer != nil {
		// Note: Mute control requires AVAudioMixerNode-specific bindings
		// For now, we store the value. Future enhancement: implement mixer mute control
	}

	return nil
}

// GetMute returns the current channel mute state
func (bc *BaseChannel) GetMute() (bool, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.muted, nil
}

// GetState returns the serializable state of the channel
func (bc *BaseChannel) GetState() ChannelState {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	connections := make([]Connection, len(bc.connections))
	copy(connections, bc.connections)

	return ChannelState{
		ID:          bc.GetIDString(), // Convert UUID to string for JSON
		Type:        bc.channelType,
		Volume:      bc.volume,
		Pan:         bc.pan,
		Muted:       bc.muted,
		Connections: connections,
		PluginChain: bc.pluginChain.GetState(),
	}
}

// SetState restores the channel from serializable state
func (bc *BaseChannel) SetState(state ChannelState) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.volume = state.Volume
	bc.pan = state.Pan
	bc.muted = state.Muted

	connections := make([]Connection, len(state.Connections))
	copy(connections, state.Connections)
	bc.connections = connections

	// Restore plugin chain
	return bc.pluginChain.SetState(state.PluginChain)
}

// MasterChannel represents the main mixer output channel
type MasterChannel struct {
	*BaseChannel

	// Master-specific functionality
	masterVolume   float32
	limiterEnabled bool
}

// AudioInputConfig holds configuration for audio input channels
type AudioInputConfig struct {
	DeviceUID string // Audio device unique identifier from devices package
	InputBus  int    // Physical input channel of the audio device (0=channel 1, 1=channel 2, etc.)
	// Maps directly to AVAudioInputNode's output bus number
	// DeviceUID + InputBus combination uniquely identifies an audio source
	MonitoringLevel float32 // Input monitoring level (0.0-1.0)
}

// AudioInputChannel represents an audio input channel
type AudioInputChannel struct {
	*BaseChannel

	// Audio input specific
	config          AudioInputConfig
	deviceUID       string
	inputBus        int
	monitoringLevel float32

	// AVFoundation integration
	inputNode unsafe.Pointer // Shared AVAudioInputNode (from engine.inputNodes)
}

// MidiInputConfig holds configuration for MIDI input channels
type MidiInputConfig struct {
	DeviceUID string
	Channel   int // MIDI channel (0-15, -1 for all)
}

// MidiInputChannel represents a MIDI input channel
type MidiInputChannel struct {
	*BaseChannel

	// MIDI input specific
	config    MidiInputConfig
	deviceUID string
	channel   int
}

// PlaybackConfig holds configuration for playback channels
type PlaybackConfig struct {
	FilePath    string
	LoopEnabled bool
	AutoStart   bool
	FadeIn      float32
	FadeOut     float32
}

// PlaybackChannel represents an audio file playback channel
type PlaybackChannel struct {
	*BaseChannel

	// Playback specific
	config      PlaybackConfig
	filePath    string
	loopEnabled bool
	autoStart   bool
	fadeIn      float32
	fadeOut     float32

	// Playback state
	isPlaying bool
	isPaused  bool
	position  float64 // Current position in seconds
}

// AuxConfig holds configuration for auxiliary send channels
type AuxConfig struct {
	SendLevel   float32
	ReturnLevel float32
	PreFader    bool
}

// AuxChannel represents an auxiliary send/return channel
type AuxChannel struct {
	*BaseChannel

	// Aux specific
	config      AuxConfig
	sendLevel   float32
	returnLevel float32
	preFader    bool
}

// NewMasterChannel creates a new master channel
func NewMasterChannel(name string, engine *Engine) (*MasterChannel, error) {
	baseChannel := NewBaseChannel(name, ChannelTypeMaster, engine)

	return &MasterChannel{
		BaseChannel:    baseChannel,
		masterVolume:   1.0,
		limiterEnabled: true, // Enable limiter by default for protection
	}, nil
}

// NewAudioInputChannel creates a new audio input channel
func NewAudioInputChannel(name string, config AudioInputConfig, engine *Engine) (*AudioInputChannel, error) {
	baseChannel := NewBaseChannel(name, ChannelTypeAudioInput, engine)

	// Get or create shared input node for this device/bus combination
	inputNode, err := engine.getOrCreateInputNode(config.DeviceUID, config.InputBus)
	if err != nil {
		return nil, fmt.Errorf("failed to get input node: %w", err)
	}

	// Create output mixer node for this channel
	avEngine := engine.getAVEngine()
	outputMixer, err := avEngine.MainMixerNode() // For now, use main mixer as output
	if err != nil {
		return nil, fmt.Errorf("failed to get output mixer: %w", err)
	}

	channel := &AudioInputChannel{
		BaseChannel:     baseChannel,
		config:          config,
		deviceUID:       config.DeviceUID,
		inputBus:        config.InputBus,
		monitoringLevel: config.MonitoringLevel,
		inputNode:       inputNode,
	}

	// Set the output mixer in base channel
	baseChannel.outputMixer = outputMixer

	return channel, nil
}

// Start starts the audio input channel and creates AVFoundation connections
func (aic *AudioInputChannel) Start() error {
	// Call base channel start first
	if err := aic.BaseChannel.Start(); err != nil {
		return err
	}

	// Connect input node to output mixer via AVFoundation
	avEngine := aic.engine.getAVEngine()

	// Connect: inputNode[inputBus] -> outputMixer[0]
	err := avEngine.Connect(aic.inputNode, aic.outputMixer, aic.inputBus, 0)
	if err != nil {
		return fmt.Errorf("failed to connect input to mixer: %w", err)
	}

	// Start AVFoundation engine if audio graph is ready
	if err := aic.engine.startAVEngineIfReady(); err != nil {
		return fmt.Errorf("failed to start audio engine: %w", err)
	}

	return nil
}

// Stop stops the audio input channel and disconnects AVFoundation connections
func (aic *AudioInputChannel) Stop() error {
	// Disconnect from output mixer
	if aic.outputMixer != nil {
		avEngine := aic.engine.getAVEngine()
		// Disconnect input bus 0 of the output mixer (where this channel connects to)
		avEngine.DisconnectNodeInput(aic.outputMixer, 0)
	}

	// Call base channel stop
	return aic.BaseChannel.Stop()
}

// NewMidiInputChannel creates a new MIDI input channel
func NewMidiInputChannel(name string, config MidiInputConfig, engine *Engine) (*MidiInputChannel, error) {
	baseChannel := NewBaseChannel(name, ChannelTypeMidiInput, engine)

	return &MidiInputChannel{
		BaseChannel: baseChannel,
		config:      config,
		deviceUID:   config.DeviceUID,
		channel:     config.Channel,
	}, nil
}

// NewPlaybackChannel creates a new playback channel
func NewPlaybackChannel(name string, config PlaybackConfig, engine *Engine) (*PlaybackChannel, error) {
	baseChannel := NewBaseChannel(name, ChannelTypePlayback, engine)

	return &PlaybackChannel{
		BaseChannel: baseChannel,
		config:      config,
		filePath:    config.FilePath,
		loopEnabled: config.LoopEnabled,
		autoStart:   config.AutoStart,
		fadeIn:      config.FadeIn,
		fadeOut:     config.FadeOut,
		isPlaying:   false,
		isPaused:    false,
		position:    0.0,
	}, nil
}

// NewAuxChannel creates a new auxiliary channel
func NewAuxChannel(name string, config AuxConfig, engine *Engine) (*AuxChannel, error) {
	baseChannel := NewBaseChannel(name, ChannelTypeAux, engine)

	return &AuxChannel{
		BaseChannel: baseChannel,
		config:      config,
		sendLevel:   config.SendLevel,
		returnLevel: config.ReturnLevel,
		preFader:    config.PreFader,
	}, nil
}

// Master channel specific methods

// SetMasterVolume sets the master output volume
func (mc *MasterChannel) SetMasterVolume(volume float32) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("master volume must be between 0.0 and 1.0")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.masterVolume = volume

	// TODO: Apply to actual master mixer

	return nil
}

// GetMasterVolume returns the master output volume
func (mc *MasterChannel) GetMasterVolume() (float32, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.masterVolume, nil
}

// SetLimiterEnabled enables or disables the output limiter
func (mc *MasterChannel) SetLimiterEnabled(enabled bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.limiterEnabled = enabled

	// TODO: Apply to actual limiter
}

// IsLimiterEnabled returns whether the output limiter is enabled
func (mc *MasterChannel) IsLimiterEnabled() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.limiterEnabled
}

// Playback channel specific methods

// Play starts playback
func (pc *PlaybackChannel) Play() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.isPlaying && !pc.isPaused {
		return nil // Already playing
	}

	pc.isPlaying = true
	pc.isPaused = false

	// TODO: Start actual audio playback

	return nil
}

// Pause pauses playback
func (pc *PlaybackChannel) Pause() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.isPlaying || pc.isPaused {
		return nil // Not playing or already paused
	}

	pc.isPaused = true

	// TODO: Pause actual audio playback

	return nil
}

// Stop stops playback and resets position
func (pc *PlaybackChannel) StopPlayback() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.isPlaying = false
	pc.isPaused = false
	pc.position = 0.0

	// TODO: Stop actual audio playback

	return nil
}

// GetPosition returns current playback position in seconds
func (pc *PlaybackChannel) GetPosition() float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.position
}

// SetPosition sets playback position in seconds
func (pc *PlaybackChannel) SetPosition(position float64) error {
	if position < 0 {
		return fmt.Errorf("position cannot be negative")
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.position = position

	// TODO: Seek in actual audio playback

	return nil
}

// Aux channel specific methods

// SetSendLevel sets the auxiliary send level
func (ac *AuxChannel) SetSendLevel(level float32) error {
	if level < 0.0 || level > 1.0 {
		return fmt.Errorf("send level must be between 0.0 and 1.0")
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.sendLevel = level

	// TODO: Apply to actual aux send

	return nil
}

// GetSendLevel returns the auxiliary send level
func (ac *AuxChannel) GetSendLevel() (float32, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.sendLevel, nil
}

// SetReturnLevel sets the auxiliary return level
func (ac *AuxChannel) SetReturnLevel(level float32) error {
	if level < 0.0 || level > 1.0 {
		return fmt.Errorf("return level must be between 0.0 and 1.0")
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.returnLevel = level

	// TODO: Apply to actual aux return

	return nil
}

// GetReturnLevel returns the auxiliary return level
func (ac *AuxChannel) GetReturnLevel() (float32, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.returnLevel, nil
}

// AuxSend cleanup method as specified in implementation requirements
func (ac *AuxChannel) Cleanup() error {
	// Stop the channel first
	if err := ac.BaseChannel.Stop(); err != nil {
		return fmt.Errorf("failed to stop aux channel during cleanup: %w", err)
	}

	// Clear all connections
	ac.mu.Lock()
	ac.connections = make([]Connection, 0)
	ac.mu.Unlock()

	// Reset to default values
	ac.sendLevel = 0.0
	ac.returnLevel = 0.0

	// Unload all plugins in the chain
	for _, instance := range ac.pluginChain.GetInstances() {
		instance.Unload()
	}

	return nil
}
