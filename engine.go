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
	EngineCreated     EngineInitState = iota // AVFoundation engine created, no channels
	MasterReady       EngineInitState = iota // Master channel initialized
	ChannelsReady     EngineInitState = iota // At least one audio channel exists
	AudioGraphReady   EngineInitState = iota // Complete audio path validated
	EngineRunning     EngineInitState = iota // AVFoundation engine started successfully
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
	BufferSize      int          // Required: 64, 128, 256, 512, 1024 frames
	OutputDeviceUID string       // Single output device for entire engine
	ErrorHandler    ErrorHandler // Optional: defaults to DefaultErrorHandler
	// ❌ REMOVED: AudioDeviceUID - individual channels bind to their own input devices
	// ❌ REMOVED: MidiDeviceUID - individual channels bind to their own MIDI devices  
	// ❌ REMOVED: SampleRate - AVAudioEngine handles sample rate conversion automatically
}

// NewEngine creates a new audio engine with the specified configuration
func NewEngine(config EngineConfig) (*Engine, error) {
	if config.BufferSize <= 0 {
		config.BufferSize = 512 // Default buffer size
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

	// Create AVFoundation engine (no sample rate - AVAudioEngine handles automatically)
	audioSpec := engine.AudioSpec{
		BufferSize:   config.BufferSize,
		BitDepth:     32, // AVAudioEngine uses 32-bit float internally
		ChannelCount: 2,  // Stereo
	}

	avEngine, err := engine.New(audioSpec)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create AVFoundation engine: %w", err)
	}

	engineInstance := &Engine{
		id:               uuid.New(),
		name:             "MacAudio Engine",
		ctx:              ctx,
		cancel:           cancel,
		channels:         make(map[string]Channel),
		avEngine:         avEngine,
		inputNodes:       make(map[string]unsafe.Pointer),
		bufferSize:       config.BufferSize,
		outputDeviceUID:  config.OutputDeviceUID,
		errorHandler:     config.ErrorHandler,
		initState:        EngineCreated,
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

	// Prepare the AVFoundation engine but don't start yet
	// The actual start happens when the audio graph is complete
	e.avEngine.Prepare()

	// Start device monitoring with adaptive polling
	if err := e.deviceMonitor.Start(); err != nil {
		e.avEngine.Stop()
		return fmt.Errorf("failed to start device monitor: %w", err)
	}

	// Start dispatcher for topology changes
	if err := e.dispatcher.Start(); err != nil {
		e.avEngine.Stop()
		e.deviceMonitor.Stop()
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}

	e.isRunning = true
	return nil
}// Stop halts all engine operations and cleanup
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return nil // Already stopped
	}

	// Stop all channels first
	for _, channel := range e.channels {
		if err := channel.Stop(); err != nil {
			e.errorHandler.HandleError(fmt.Errorf("error stopping channel %s: %w", channel.GetID(), err))
		}
	}

	// Stop dispatcher
	if err := e.dispatcher.Stop(); err != nil {
		e.errorHandler.HandleError(fmt.Errorf("error stopping dispatcher: %w", err))
	}

	// Stop device monitor
	if err := e.deviceMonitor.Stop(); err != nil {
		e.errorHandler.HandleError(fmt.Errorf("error stopping device monitor: %w", err))
	}

	// Stop AVFoundation engine
	e.stopAVEngine()

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

	return EngineConfig{
		BufferSize:      e.bufferSize,
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

// removeInputNode removes a shared input node when no longer needed
func (e *Engine) removeInputNode(deviceUID string, inputBus int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	key := fmt.Sprintf("%s:%d", deviceUID, inputBus)
	delete(e.inputNodes, key)
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

	return nil
}
