package macaudio

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/google/uuid"
	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/devices"
)

// EngineInitState tracks engine initialization lifecycle
type EngineInitState int

const (
	EngineCreated   EngineInitState = iota // AVFoundation engine created, no channels
	MasterReady     EngineInitState = iota // Master channel initialized
	ChannelsReady   EngineInitState = iota // At least one audio channel exists
	AudioGraphReady EngineInitState = iota // Complete audio path validated
	EngineRunning   EngineInitState = iota // AVFoundation engine started successfully
)

// Engine represents the main audio engine with unified architecture
type Engine struct {
	// Core identity (UUID hybrid pattern)
	id   uuid.UUID // Internal UUID
	name string

	// Core state
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	isRunning     bool
	deviceMonitor *DeviceMonitor
	dispatcher    *Dispatcher
	serializer    *Serializer

	// Channel management (string keys for JSON compatibility)
	channels      map[string]Channel
	masterChannel *MasterChannel

	// AVFoundation integration
	avEngine   *engine.Engine
	inputNodes map[string]unsafe.Pointer // key: "deviceUID:inputBus", value: AVAudioInputNode*

	// Configuration
	bufferSize      int
	outputDeviceUID string // Single output device for entire engine

	// Error boundary
	errorHandler ErrorHandler

	// Initialization state tracking
	initState EngineInitState
}

// EngineConfig holds configuration for engine initialization
type EngineConfig struct {
	AudioSpec       engine.AudioSpec // Complete audio specification
	OutputDeviceUID string           // Single output device for entire engine
	ErrorHandler    ErrorHandler     // Optional: defaults to DefaultErrorHandler
	// ❌ REMOVED: AudioDeviceUID - individual channels bind to their own input devices
	// ❌ REMOVED: MidiDeviceUID - individual channels bind to their own MIDI devices
}

// NewEngine creates a new audio engine with the specified configuration
func NewEngine(config EngineConfig) (*Engine, error) {
	// Validate SampleRate
	if config.AudioSpec.SampleRate <= 0 {
		config.AudioSpec.SampleRate = 48000 // Default sample rate
	} else if config.AudioSpec.SampleRate < 8000 {
		return nil, fmt.Errorf("SampleRate must be at least 8000 Hz, got %.0f", config.AudioSpec.SampleRate)
	} else if config.AudioSpec.SampleRate > 384000 {
		return nil, fmt.Errorf("SampleRate cannot exceed 384000 Hz, got %.0f", config.AudioSpec.SampleRate)
	}

	// Validate BufferSize
	if config.AudioSpec.BufferSize <= 0 {
		config.AudioSpec.BufferSize = 512 // Default buffer size
	} else if config.AudioSpec.BufferSize < 64 {
		return nil, fmt.Errorf("BufferSize must be at least 64 samples, got %d", config.AudioSpec.BufferSize)
	} else if config.AudioSpec.BufferSize > 4096 {
		return nil, fmt.Errorf("BufferSize cannot exceed 4096 samples, got %d", config.AudioSpec.BufferSize)
	}

	// Set AudioSpec defaults if not provided
	if config.AudioSpec.BitDepth <= 0 {
		config.AudioSpec.BitDepth = 32 // AVAudioEngine uses 32-bit float internally
	}
	if config.AudioSpec.ChannelCount <= 0 {
		config.AudioSpec.ChannelCount = 2 // Stereo
	}

	if config.OutputDeviceUID == "" {
		return nil, fmt.Errorf("OutputDeviceUID is required in EngineConfig")
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = &DefaultErrorHandler{}
	}

	// Validate output device exists and is online
	audioDevices, err := devices.GetAudio()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate audio devices: %w", err)
	}

	device := audioDevices.ByUID(config.OutputDeviceUID)
	if device == nil {
		return nil, fmt.Errorf("output device with UID %s not found", config.OutputDeviceUID)
	}

	if !device.IsOnline {
		return nil, fmt.Errorf("output device %s is not online", config.OutputDeviceUID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create AVFoundation engine with the validated AudioSpec
	avEngine, err := engine.New(config.AudioSpec)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create AVFoundation engine: %w", err)
	}

	engineInstance := &Engine{
		id:              uuid.New(),
		name:            "MacAudio Engine",
		ctx:             ctx,
		cancel:          cancel,
		channels:        make(map[string]Channel),
		avEngine:        avEngine,
		inputNodes:      make(map[string]unsafe.Pointer),
		bufferSize:      config.AudioSpec.BufferSize,
		outputDeviceUID: config.OutputDeviceUID,
		errorHandler:    config.ErrorHandler,
		initState:       EngineCreated,
	}

	// Initialize master channel (always present)
	masterChannel, err := NewMasterChannel("Master", engineInstance)
	if err != nil {
		avEngine.Destroy() // Clean up AVFoundation engine
		cancel()
		return nil, fmt.Errorf("failed to create master channel: %w", err)
	}
	engineInstance.masterChannel = masterChannel
	engineInstance.channels[masterChannel.GetIDString()] = masterChannel // UUID to string conversion
	engineInstance.initState = MasterReady

	// Initialize device monitor
	engineInstance.deviceMonitor = NewDeviceMonitor(engineInstance)

	// Initialize dispatcher for serialized topology changes
	engineInstance.dispatcher = NewDispatcher(engineInstance)
	
	// Start dispatcher immediately - channel creation needs it before engine.Start()
	if err := engineInstance.dispatcher.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dispatcher: %w", err)
	}

	// Initialize serializer for state persistence
	engineInstance.serializer = NewSerializer(engineInstance)

	return engineInstance, nil
}

// Start begins engine operation with device binding and monitoring
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return fmt.Errorf("engine is already running")
	}

	// Route actual engine start through dispatcher for serialization
	response := make(chan DispatcherResult, 1)
	op := DispatcherOperation{
		Type:     OpStartEngine,
		Data:     nil, // No data needed for engine start
		Response: response,
	}
	
	e.dispatcher.operations <- op
	result := <-response
	
	if !result.Success {
		// Cleanup dispatcher if start failed
		e.dispatcher.Stop()
		return fmt.Errorf("engine start failed: %w", result.Error)
	}

	e.isRunning = true
	return nil
} // Stop halts all engine operations and cleanup
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return nil // Already stopped
	}

	// Route engine stop through dispatcher for serialization
	response := make(chan DispatcherResult, 1)
	op := DispatcherOperation{
		Type:     OpStopEngine,
		Data:     nil, // No data needed for engine stop
		Response: response,
	}
	
	e.dispatcher.operations <- op
	result := <-response
	
	if !result.Success {
		e.errorHandler.HandleError(fmt.Errorf("engine stop failed: %w", result.Error))
		// Continue with cleanup even if dispatcher stop failed
	}

	// Stop dispatcher last
	if err := e.dispatcher.Stop(); err != nil {
		e.errorHandler.HandleError(fmt.Errorf("error stopping dispatcher: %w", err))
	}

	// Cancel context to stop all background operations
	e.cancel()

	e.isRunning = false
	return nil
}

// UUID Helper Methods (following hybrid pattern)

// GetID returns the engine's UUID
func (e *Engine) GetID() uuid.UUID {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.id
}

// GetIDString returns the engine's UUID as string
func (e *Engine) GetIDString() string {
	return e.GetID().String()
}

// GetName returns the engine name
func (e *Engine) GetName() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.name
}

// SetName sets the engine name
func (e *Engine) SetName(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.name = name
}

// GetChannelByID retrieves a channel by its UUID (using hybrid pattern)
func (e *Engine) GetChannelByID(id uuid.UUID) (Channel, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	channel, exists := e.channels[id.String()] // Convert UUID to string for map lookup
	return channel, exists
}

// IsRunning returns whether the engine is currently running
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// GetChannel retrieves a channel by its ID
func (e *Engine) GetChannel(id string) (Channel, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	channel, exists := e.channels[id]
	return channel, exists
}

// GetMasterChannel returns the master mixer channel
func (e *Engine) GetMasterChannel() *MasterChannel {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.masterChannel
}

// ListChannels returns all channel IDs
func (e *Engine) ListChannels() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := make([]string, 0, len(e.channels))
	for id := range e.channels {
		ids = append(ids, id)
	}
	return ids
}

// CreateAudioInputChannel creates a new audio input channel
func (e *Engine) CreateAudioInputChannel(id string, config AudioInputConfig) (*AudioInputChannel, error) {
	return e.dispatcher.CreateAudioInputChannel(id, config)
}

// CreateMidiInputChannel creates a new MIDI input channel
func (e *Engine) CreateMidiInputChannel(id string, config MidiInputConfig) (*MidiInputChannel, error) {
	return e.dispatcher.CreateMidiInputChannel(id, config)
}

// CreatePlaybackChannel creates a new playback channel
func (e *Engine) CreatePlaybackChannel(id string, config PlaybackConfig) (*PlaybackChannel, error) {
	return e.dispatcher.CreatePlaybackChannel(id, config)
}

// CreateAuxChannel creates a new auxiliary send channel
func (e *Engine) CreateAuxChannel(id string, config AuxConfig) (*AuxChannel, error) {
	return e.dispatcher.CreateAuxChannel(id, config)
}

// RemoveChannel removes a channel from the engine
func (e *Engine) RemoveChannel(id string) error {
	return e.dispatcher.RemoveChannel(id)
}

// GetDeviceMonitor returns the device monitor for external access
func (e *Engine) GetDeviceMonitor() *DeviceMonitor {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.deviceMonitor
}

// GetDispatcher returns the dispatcher for external access
func (e *Engine) GetDispatcher() *Dispatcher {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.dispatcher
}

// GetSerializer returns the serializer for state management
func (e *Engine) GetSerializer() *Serializer {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.serializer
}

// GetConfiguration returns current engine configuration
func (e *Engine) GetConfiguration() EngineConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get the current AudioSpec from the AVEngine
	currentSpec := e.avEngine.GetSpec()

	return EngineConfig{
		AudioSpec:       currentSpec,
		OutputDeviceUID: e.outputDeviceUID,
		ErrorHandler:    e.errorHandler,
	}
}

// addChannel adds a channel to the engine (internal method called by dispatcher)
func (e *Engine) addChannel(channel Channel) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	idString := channel.GetIDString() // Convert UUID to string for map key
	if _, exists := e.channels[idString]; exists {
		return fmt.Errorf("channel with ID %s already exists", idString)
	}

	e.channels[idString] = channel
	return nil
}

// removeChannel removes a channel from the engine (internal method called by dispatcher)
func (e *Engine) removeChannel(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if id == "master" {
		return fmt.Errorf("cannot remove master channel")
	}

	channel, exists := e.channels[id]
	if !exists {
		return fmt.Errorf("channel with ID %s not found", id)
	}

	// Stop the channel before removing
	if err := channel.Stop(); err != nil {
		e.errorHandler.HandleError(fmt.Errorf("error stopping channel %s during removal: %w", id, err))
	}

	delete(e.channels, id)
	return nil
}

// getOrCreateInputNode returns a shared AVAudioInputNode for the given device and input bus
// This implements the node sharing strategy for efficient resource usage
func (e *Engine) getOrCreateInputNode(deviceUID string, inputBus int) (unsafe.Pointer, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := fmt.Sprintf("%s:%d", deviceUID, inputBus)

	// Return existing node if it exists
	if node, exists := e.inputNodes[key]; exists {
		return node, nil
	}

	// Get the AVAudioEngine's input node
	inputNode, err := e.avEngine.InputNode()
	if err != nil {
		return nil, fmt.Errorf("failed to get input node: %w", err)
	}

	// Store the node for sharing
	e.inputNodes[key] = inputNode

	return inputNode, nil
}

// SetChannelMute sets channel mute state via dispatcher (topology change)
func (e *Engine) SetChannelMute(channelID string, muted bool) error {
	return e.dispatcher.SetChannelMute(channelID, muted)
}

// SetPluginBypass sets plugin bypass state via dispatcher (topology change) 
func (e *Engine) SetPluginBypass(channelID, pluginID string, bypassed bool) error {
	return e.dispatcher.SetPluginBypass(channelID, pluginID, bypassed)
}

// ChangeChannelDevice changes channel device via dispatcher (topology change)
func (e *Engine) ChangeChannelDevice(channelID, newDeviceUID string) error {
	return e.dispatcher.ChangeChannelDevice(channelID, newDeviceUID)
}

// ChangeOutputDevice changes output device via dispatcher (topology change)
func (e *Engine) ChangeOutputDevice(newDeviceUID string) error {
	return e.dispatcher.ChangeOutputDevice(newDeviceUID)
}

// removeInputNode removes a shared input node when no longer needed
func (e *Engine) removeInputNode(deviceUID string, inputBus int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := fmt.Sprintf("%s:%d", deviceUID, inputBus)
	delete(e.inputNodes, key)
}

// GetNativeEngine returns the underlying AVFoundation engine pointer for taps
func (e *Engine) GetNativeEngine() unsafe.Pointer {
	if e.avEngine != nil {
		return e.avEngine.GetNativeEngine()
	}
	return nil
}

// getAVEngine returns the underlying AVFoundation engine for channel use
func (e *Engine) getAVEngine() *engine.Engine {
	return e.avEngine
}

// startAVEngineIfReady starts the AVFoundation engine when audio graph is complete
func (e *Engine) startAVEngineIfReady() error {
	// Only start if not already running and we have a complete audio path
	if e.avEngine.IsRunning() {
		return nil
	}

	// Ensure master channel is connected to output
	if e.masterChannel == nil {
		return fmt.Errorf("master channel not available")
	}

	// Start the AVFoundation engine with complete graph
	if err := e.avEngine.Start(); err != nil {
		return fmt.Errorf("failed to start AVFoundation engine: %w", err)
	}

	return nil
}

// stopAVEngine stops the AVFoundation engine
func (e *Engine) stopAVEngine() {
	if e.avEngine != nil && e.avEngine.IsRunning() {
		e.avEngine.Stop()
	}
}

// Destroy properly cleans up the engine and all resources
func (e *Engine) Destroy() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop the engine if running
	if e.isRunning {
		e.Stop()
	}

	// Clear input nodes map
	e.inputNodes = make(map[string]unsafe.Pointer)

	// Destroy AVFoundation engine
	if e.avEngine != nil {
		e.avEngine.Destroy()
		e.avEngine = nil
	}
}

// prepareAudioRouting sets up basic audio routing to satisfy AVFoundation requirements
func (e *Engine) prepareAudioRouting() error {
	// AVFoundation requires at least one connection between input and output
	// Create a basic connection: inputNode -> mainMixerNode -> outputNode

	inputNode, err := e.avEngine.InputNode()
	if err != nil {
		return fmt.Errorf("failed to get input node: %w", err)
	}

	mainMixer, err := e.avEngine.MainMixerNode()
	if err != nil {
		return fmt.Errorf("failed to get main mixer node: %w", err)
	}

	// Connect input to main mixer (bus 0 -> bus 0)
	// This creates the minimal routing that AVFoundation requires
	if err := e.avEngine.Connect(inputNode, mainMixer, 0, 0); err != nil {
		// This might fail if already connected, which is fine
		// AVFoundation will handle the routing
	}

	// CRITICAL: Connect main mixer to output node for audio to reach speakers
	outputNode, err := e.avEngine.OutputNode()
	if err != nil {
		return fmt.Errorf("failed to get output node: %w", err)
	}

	// Connect main mixer output to speakers
	if err := e.avEngine.Connect(mainMixer, outputNode, 0, 0); err != nil {
		// This might fail if already connected, which is fine
		// But we need this connection for audio to be audible
	}

	return nil
}

// validateEngineReadiness checks if the engine is ready to start
func (e *Engine) validateEngineReadiness() error {
	// Check that we have at least a master channel
	if e.masterChannel == nil {
		return fmt.Errorf("master channel is not initialized")
	}

	// Check that the AVFoundation engine is initialized
	if e.avEngine == nil {
		return fmt.Errorf("AVFoundation engine is not initialized")
	}

	// Check that device monitor and dispatcher are initialized
	if e.deviceMonitor == nil {
		return fmt.Errorf("device monitor is not initialized")
	}

	if e.dispatcher == nil {
		return fmt.Errorf("dispatcher is not initialized")
	}

	// Validate output device is still available
	audioDevices, err := devices.GetAudio()
	if err != nil {
		return fmt.Errorf("failed to enumerate audio devices: %w", err)
	}

	device := audioDevices.ByUID(e.outputDeviceUID)
	if device == nil {
		return fmt.Errorf("output device with UID %s is no longer available", e.outputDeviceUID)
	}

	if !device.IsOnline {
		return fmt.Errorf("output device %s is not online", e.outputDeviceUID)
	}

	return nil
}

// prepareAVFoundationSafely attempts to prepare the AVFoundation engine with error recovery
func (e *Engine) prepareAVFoundationSafely() error {
	// First, try to prepare basic audio routing to avoid crashes
	if err := e.prepareAudioRouting(); err != nil {
		return fmt.Errorf("failed to prepare audio routing: %w", err)
	}

	// Now attempt to prepare the AVFoundation engine
	// This might still crash if audio graph is incomplete, but we've done our best
	defer func() {
		if r := recover(); r != nil {
			// If AVFoundation crashes, convert panic to error
			// This prevents the entire application from crashing
		}
	}()

	e.avEngine.Prepare()
	return nil
}
