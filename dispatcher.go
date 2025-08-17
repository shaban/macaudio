package macaudio

import (
	"fmt"
	"sync"
	"time"
	
	"github.com/shaban/macaudio/devices"
)

// DispatcherOperation represents a topology change operation
type DispatcherOperation struct {
	Type     OperationType
	Data     interface{}
	Response chan DispatcherResult
}

// OperationType represents the type of dispatcher operation
type OperationType string

const (
	// Engine operations
	OpCreateEngine     OperationType = "create_engine"
	OpStartEngine      OperationType = "start_engine"
	OpStopEngine       OperationType = "stop_engine"
	
	// Channel creation operations
	OpCreateAudioInput OperationType = "create_audio_input"
	OpCreateMidiInput  OperationType = "create_midi_input"
	OpCreatePlayback   OperationType = "create_playback"
	OpCreateAux        OperationType = "create_aux"
	OpRemoveChannel    OperationType = "remove_channel"
	
	// Connection operations
	OpConnectChannels    OperationType = "connect_channels"
	OpDisconnectChannels OperationType = "disconnect_channels"
	
	// Topology changing operations (require dispatcher)
	OpSetMute           OperationType = "set_mute"
	OpPluginBypass      OperationType = "plugin_bypass"
	OpDeviceChange      OperationType = "device_change"
	OpOutputDeviceChange OperationType = "output_device_change"
)

// DispatcherResult represents the result of a dispatcher operation
type DispatcherResult struct {
	Success bool
	Data    interface{}
	Error   error
}

// Dispatcher manages serialized topology changes to ensure glitch-free operation
type Dispatcher struct {
	engine      *Engine
	mu          sync.RWMutex
	isRunning   bool
	operations  chan DispatcherOperation
	stopChan    chan struct{}
	
	// Performance tracking
	lastOperationDuration time.Duration
	maxOperationDuration  time.Duration
	performanceMu         sync.RWMutex
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(engine *Engine) *Dispatcher {
	return &Dispatcher{
		engine:               engine,
		operations:           make(chan DispatcherOperation, 100), // Buffered channel
		stopChan:             make(chan struct{}),
		maxOperationDuration: 300 * time.Millisecond, // Target: sub-300ms
	}
}

// Start begins the dispatcher loop for serialized topology changes
func (d *Dispatcher) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.isRunning {
		return fmt.Errorf("dispatcher is already running")
	}
	
	d.isRunning = true
	go d.dispatchLoop()
	
	return nil
}

// Stop halts the dispatcher
func (d *Dispatcher) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if !d.isRunning {
		return nil // Already stopped
	}
	
	close(d.stopChan)
	d.isRunning = false
	
	return nil
}

// IsRunning returns whether the dispatcher is active
func (d *Dispatcher) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.isRunning
}

// GetPerformanceStats returns dispatcher performance statistics
func (d *Dispatcher) GetPerformanceStats() (lastDuration, maxDuration time.Duration) {
	d.performanceMu.RLock()
	defer d.performanceMu.RUnlock()
	return d.lastOperationDuration, d.maxOperationDuration
}

// dispatchLoop runs the main dispatch loop for topology changes
func (d *Dispatcher) dispatchLoop() {
	for {
		select {
		case <-d.stopChan:
			return
		case op := <-d.operations:
			start := time.Now()
			result := d.executeOperation(op)
			duration := time.Since(start)
			
			// Update performance tracking
			d.performanceMu.Lock()
			d.lastOperationDuration = duration
			if duration > d.maxOperationDuration {
				d.maxOperationDuration = duration
			}
			d.performanceMu.Unlock()
			
			// Log if operation exceeded target
			if duration > 300*time.Millisecond {
				d.engine.errorHandler.HandleError(
					fmt.Errorf("topology change took %v, target is sub-300ms", duration))
			}
			
			// Send result back
			op.Response <- result
		}
	}
}

// executeOperation executes a single dispatcher operation
func (d *Dispatcher) executeOperation(op DispatcherOperation) DispatcherResult {
	switch op.Type {
	// Engine operations
	case OpStartEngine:
		err := d.startEngine()
		return DispatcherResult{Success: err == nil, Error: err}
		
	case OpStopEngine:
		err := d.stopEngine()
		return DispatcherResult{Success: err == nil, Error: err}
	
	// Channel creation operations
	case OpCreateAudioInput:
		data := op.Data.(CreateAudioInputData)
		channel, err := d.createAudioInput(data.ID, data.Config)
		return DispatcherResult{Success: err == nil, Data: channel, Error: err}
		
	case OpCreateMidiInput:
		data := op.Data.(CreateMidiInputData)
		channel, err := d.createMidiInput(data.ID, data.Config)
		return DispatcherResult{Success: err == nil, Data: channel, Error: err}
		
	case OpCreatePlayback:
		data := op.Data.(CreatePlaybackData)
		channel, err := d.createPlayback(data.ID, data.Config)
		return DispatcherResult{Success: err == nil, Data: channel, Error: err}
		
	case OpCreateAux:
		data := op.Data.(CreateAuxData)
		channel, err := d.createAux(data.ID, data.Config)
		return DispatcherResult{Success: err == nil, Data: channel, Error: err}
		
	case OpRemoveChannel:
		id := op.Data.(string)
		err := d.removeChannel(id)
		return DispatcherResult{Success: err == nil, Error: err}
		
	// Connection operations
	case OpConnectChannels:
		data := op.Data.(ConnectChannelsData)
		err := d.connectChannels(data.SourceID, data.TargetID, data.Bus)
		return DispatcherResult{Success: err == nil, Error: err}
		
	case OpDisconnectChannels:
		data := op.Data.(DisconnectChannelsData)
		err := d.disconnectChannels(data.SourceID, data.TargetID, data.Bus)
		return DispatcherResult{Success: err == nil, Error: err}
	
	// Topology changing operations
	case OpSetMute:
		data := op.Data.(SetMuteData)
		err := d.setMute(data.ChannelID, data.Muted)
		return DispatcherResult{Success: err == nil, Error: err}
		
	case OpPluginBypass:
		data := op.Data.(PluginBypassData)
		err := d.setPluginBypass(data.ChannelID, data.PluginID, data.Bypassed)
		return DispatcherResult{Success: err == nil, Error: err}
		
	case OpDeviceChange:
		data := op.Data.(DeviceChangeData)
		err := d.changeChannelDevice(data.ChannelID, data.NewDeviceUID)
		return DispatcherResult{Success: err == nil, Error: err}
		
	case OpOutputDeviceChange:
		data := op.Data.(OutputDeviceChangeData)
		err := d.changeOutputDevice(data.NewDeviceUID)
		return DispatcherResult{Success: err == nil, Error: err}
		
	default:
		return DispatcherResult{
			Success: false,
			Error:   fmt.Errorf("unknown operation type: %s", op.Type),
		}
	}
}

// Data structures for dispatcher operations

// Engine operation data structures
type CreateEngineData struct {
	Config EngineConfig
}

type SetMuteData struct {
	ChannelID string
	Muted     bool
}

type PluginBypassData struct {
	ChannelID  string
	PluginID   string
	Bypassed   bool
}

type DeviceChangeData struct {
	ChannelID   string
	NewDeviceUID string
}

type OutputDeviceChangeData struct {
	NewDeviceUID string
}

// Channel operation data structures
type CreateAudioInputData struct {
	ID     string
	Config AudioInputConfig
}

type CreateMidiInputData struct {
	ID     string
	Config MidiInputConfig
}

type CreatePlaybackData struct {
	ID     string
	Config PlaybackConfig
}

type CreateAuxData struct {
	ID     string
	Config AuxConfig
}

type ConnectChannelsData struct {
	SourceID string
	TargetID string
	Bus      int
}

type DisconnectChannelsData struct {
	SourceID string
	TargetID string
	Bus      int
}

// Public API methods that queue operations

// CreateAudioInputChannel creates a new audio input channel via dispatcher
func (d *Dispatcher) CreateAudioInputChannel(id string, config AudioInputConfig) (*AudioInputChannel, error) {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpCreateAudioInput,
		Data:     CreateAudioInputData{ID: id, Config: config},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	if result.Success {
		return result.Data.(*AudioInputChannel), nil
	}
	return nil, result.Error
}

// CreateMidiInputChannel creates a new MIDI input channel via dispatcher
func (d *Dispatcher) CreateMidiInputChannel(id string, config MidiInputConfig) (*MidiInputChannel, error) {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpCreateMidiInput,
		Data:     CreateMidiInputData{ID: id, Config: config},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	if result.Success {
		return result.Data.(*MidiInputChannel), nil
	}
	return nil, result.Error
}

// CreatePlaybackChannel creates a new playback channel via dispatcher
func (d *Dispatcher) CreatePlaybackChannel(id string, config PlaybackConfig) (*PlaybackChannel, error) {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpCreatePlayback,
		Data:     CreatePlaybackData{ID: id, Config: config},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	if result.Success {
		return result.Data.(*PlaybackChannel), nil
	}
	return nil, result.Error
}

// CreateAuxChannel creates a new auxiliary channel via dispatcher
func (d *Dispatcher) CreateAuxChannel(id string, config AuxConfig) (*AuxChannel, error) {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpCreateAux,
		Data:     CreateAuxData{ID: id, Config: config},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	if result.Success {
		return result.Data.(*AuxChannel), nil
	}
	return nil, result.Error
}

// RemoveChannel removes a channel via dispatcher
func (d *Dispatcher) RemoveChannel(id string) error {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpRemoveChannel,
		Data:     id,
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	return result.Error
}

// ConnectChannels connects two channels via dispatcher
func (d *Dispatcher) ConnectChannels(sourceID, targetID string, bus int) error {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpConnectChannels,
		Data:     ConnectChannelsData{SourceID: sourceID, TargetID: targetID, Bus: bus},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	return result.Error
}

// DisconnectChannels disconnects two channels via dispatcher
func (d *Dispatcher) DisconnectChannels(sourceID, targetID string, bus int) error {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpDisconnectChannels,
		Data:     DisconnectChannelsData{SourceID: sourceID, TargetID: targetID, Bus: bus},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	return result.Error
}

// Topology-changing operations (require dispatcher for race prevention)

// SetChannelMute sets channel mute state via dispatcher (topology change)
func (d *Dispatcher) SetChannelMute(channelID string, muted bool) error {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpSetMute,
		Data:     SetMuteData{ChannelID: channelID, Muted: muted},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	return result.Error
}

// SetPluginBypass sets plugin bypass state via dispatcher (topology change)
func (d *Dispatcher) SetPluginBypass(channelID, pluginID string, bypassed bool) error {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpPluginBypass,
		Data:     PluginBypassData{ChannelID: channelID, PluginID: pluginID, Bypassed: bypassed},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	return result.Error
}

// ChangeChannelDevice changes the device for a channel via dispatcher (topology change)
func (d *Dispatcher) ChangeChannelDevice(channelID, newDeviceUID string) error {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpDeviceChange,
		Data:     DeviceChangeData{ChannelID: channelID, NewDeviceUID: newDeviceUID},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	return result.Error
}

// ChangeOutputDevice changes the engine's output device via dispatcher (topology change)
func (d *Dispatcher) ChangeOutputDevice(newDeviceUID string) error {
	response := make(chan DispatcherResult, 1)
	
	op := DispatcherOperation{
		Type:     OpOutputDeviceChange,
		Data:     OutputDeviceChangeData{NewDeviceUID: newDeviceUID},
		Response: response,
	}
	
	d.operations <- op
	result := <-response
	
	return result.Error
}

// Internal implementation methods (executed within dispatcher thread)

func (d *Dispatcher) createAudioInput(id string, config AudioInputConfig) (*AudioInputChannel, error) {
	channel, err := NewAudioInputChannel(id, config, d.engine)
	if err != nil {
		return nil, err
	}
	
	if err := d.engine.addChannel(channel); err != nil {
		return nil, err
	}
	
	// Auto-connect to master if specified in config
	// TODO: Add auto-connect configuration
	
	return channel, nil
}

func (d *Dispatcher) createMidiInput(id string, config MidiInputConfig) (*MidiInputChannel, error) {
	channel, err := NewMidiInputChannel(id, config, d.engine)
	if err != nil {
		return nil, err
	}
	
	if err := d.engine.addChannel(channel); err != nil {
		return nil, err
	}
	
	return channel, nil
}

func (d *Dispatcher) createPlayback(id string, config PlaybackConfig) (*PlaybackChannel, error) {
	channel, err := NewPlaybackChannel(id, config, d.engine)
	if err != nil {
		return nil, err
	}
	
	if err := d.engine.addChannel(channel); err != nil {
		return nil, err
	}
	
	// Auto-connect to master
	if err := channel.ConnectTo(d.engine.masterChannel, 0); err != nil {
		d.engine.errorHandler.HandleError(fmt.Errorf("failed to auto-connect playback to master: %w", err))
	}
	
	return channel, nil
}

func (d *Dispatcher) createAux(id string, config AuxConfig) (*AuxChannel, error) {
	channel, err := NewAuxChannel(id, config, d.engine)
	if err != nil {
		return nil, err
	}
	
	if err := d.engine.addChannel(channel); err != nil {
		return nil, err
	}
	
	// Auto-connect to master
	if err := channel.ConnectTo(d.engine.masterChannel, 0); err != nil {
		d.engine.errorHandler.HandleError(fmt.Errorf("failed to auto-connect aux to master: %w", err))
	}
	
	return channel, nil
}

func (d *Dispatcher) removeChannel(id string) error {
	return d.engine.removeChannel(id)
}

func (d *Dispatcher) connectChannels(sourceID, targetID string, bus int) error {
	sourceChannel, exists := d.engine.GetChannel(sourceID)
	if !exists {
		return fmt.Errorf("source channel %s not found", sourceID)
	}
	
	targetChannel, exists := d.engine.GetChannel(targetID)
	if !exists {
		return fmt.Errorf("target channel %s not found", targetID)
	}
	
	return sourceChannel.ConnectTo(targetChannel, bus)
}

func (d *Dispatcher) disconnectChannels(sourceID, targetID string, bus int) error {
	sourceChannel, exists := d.engine.GetChannel(sourceID)
	if !exists {
		return fmt.Errorf("source channel %s not found", sourceID)
	}
	
	targetChannel, exists := d.engine.GetChannel(targetID)
	if !exists {
		return fmt.Errorf("target channel %s not found", targetID)
	}
	
	return sourceChannel.DisconnectFrom(targetChannel, bus)
}

// Engine lifecycle operations (serialized through dispatcher)

func (d *Dispatcher) startEngine() error {
	// This is the actual engine start logic moved from Engine.Start()
	if err := d.engine.validateEngineReadiness(); err != nil {
		return fmt.Errorf("engine validation failed: %w", err)
	}

	if err := d.engine.prepareAVFoundationSafely(); err != nil {
		return fmt.Errorf("failed to prepare AVFoundation engine: %w", err)
	}

	// 🎯 CRITICAL: Start all channels to create direct connections
	// This was missing - channels were never started!
	fmt.Printf("🚀 Starting all channels to create audio connections...\n")
	for id, channel := range d.engine.channels {
		fmt.Printf("🔗 Starting channel %s...\n", id)
		if err := channel.Start(); err != nil {
			return fmt.Errorf("failed to start channel %s: %w", id, err)
		}
		fmt.Printf("✅ Channel %s started successfully\n", id)
	}

	// Start the actual AVFoundation engine now that it's prepared
	if err := d.engine.startAVEngineIfReady(); err != nil {
		return fmt.Errorf("failed to start AVFoundation engine: %w", err)
	}

	// Start device monitoring
	if err := d.engine.deviceMonitor.Start(); err != nil {
		d.engine.avEngine.Stop()
		return fmt.Errorf("failed to start device monitor: %w", err)
	}

	return nil
}

func (d *Dispatcher) stopEngine() error {
	// Stop all channels first
	for _, channel := range d.engine.channels {
		if err := channel.Stop(); err != nil {
			d.engine.errorHandler.HandleError(fmt.Errorf("error stopping channel %s: %w", channel.GetID(), err))
		}
	}

	// Stop device monitor
	if err := d.engine.deviceMonitor.Stop(); err != nil {
		d.engine.errorHandler.HandleError(fmt.Errorf("error stopping device monitor: %w", err))
	}

	// Stop AVFoundation engine
	d.engine.stopAVEngine()

	return nil
}

// Topology changing operations (require dispatcher serialization)

func (d *Dispatcher) setMute(channelID string, muted bool) error {
	channel, exists := d.engine.GetChannel(channelID)
	if !exists {
		return fmt.Errorf("channel %s not found", channelID)
	}

	// This is a topology change, so it goes through dispatcher
	// The actual AVFoundation mute will happen here
	
	// Handle different channel types that embed BaseChannel
	var baseChannel *BaseChannel
	switch ch := channel.(type) {
	case *AudioInputChannel:
		baseChannel = ch.BaseChannel
	case *MasterChannel:
		baseChannel = ch.BaseChannel
	case *BaseChannel:
		baseChannel = ch
	default:
		return fmt.Errorf("unsupported channel type for mute operation")
	}

	if baseChannel != nil {
		baseChannel.mu.Lock()
		defer baseChannel.mu.Unlock()
		
		oldMuted := baseChannel.muted
		baseChannel.muted = muted
		
		// Apply mute directly to AVFoundation without changing volume
		if baseChannel.outputMixer != nil && oldMuted != muted {
			if muted {
				// Mute by setting mixer volume to 0, but don't change baseChannel.volume
				fmt.Printf("🔇 Muting mixer node %p\n", baseChannel.outputMixer)
				if baseChannel.engine != nil {
					avEngine := baseChannel.engine.getAVEngine()
					if avEngine != nil {
						if err := avEngine.SetMixerVolume(baseChannel.outputMixer, 0.0); err != nil {
							fmt.Printf("❌ Failed to mute: %v\n", err)
						} else {
							fmt.Printf("✅ Mute applied successfully\n")
						}
					}
				}
			} else {
				// Unmute by restoring current volume setting
				fmt.Printf("🔊 Unmuting mixer node %p to volume %.2f\n", baseChannel.outputMixer, baseChannel.volume)
				if baseChannel.engine != nil {
					avEngine := baseChannel.engine.getAVEngine()
					if avEngine != nil {
						if err := avEngine.SetMixerVolume(baseChannel.outputMixer, baseChannel.volume); err != nil {
							fmt.Printf("❌ Failed to unmute: %v\n", err)
						} else {
							fmt.Printf("✅ Unmute applied successfully\n")
						}
					}
				}
			}
		}
	}

	return nil
}

func (d *Dispatcher) setPluginBypass(channelID, pluginID string, bypassed bool) error {
	channel, exists := d.engine.GetChannel(channelID)
	if !exists {
		return fmt.Errorf("channel %s not found", channelID)
	}

	pluginChain := channel.GetPluginChain()
	instance, exists := pluginChain.GetInstance(pluginID)
	if !exists {
		return fmt.Errorf("plugin instance %s not found in channel %s", pluginID, channelID)
	}

	// Plugin bypass is a topology change
	// TODO: Add SetBypassed method to PluginInstance
	instance.mu.Lock()
	instance.IsActive = !bypassed // For now, use IsActive as bypass state
	instance.mu.Unlock()
	return nil
}

func (d *Dispatcher) changeChannelDevice(channelID, newDeviceUID string) error {
	channel, exists := d.engine.GetChannel(channelID)
	if !exists {
		return fmt.Errorf("channel %s not found", channelID)
	}

	// Device changes are topology changes that require reconnection
	switch ch := channel.(type) {
	case *AudioInputChannel:
		// Stop current channel
		if err := ch.Stop(); err != nil {
			return fmt.Errorf("failed to stop channel during device change: %w", err)
		}
		
		// Update device configuration
		ch.config.DeviceUID = newDeviceUID
		ch.deviceUID = newDeviceUID
		
		// Get new input node
		inputNode, err := d.engine.getOrCreateInputNode(newDeviceUID, ch.inputBus)
		if err != nil {
			return fmt.Errorf("failed to get new input node: %w", err)
		}
		ch.inputNode = inputNode
		
		// Restart channel with new device
		return ch.Start()
		
	case *MidiInputChannel:
		// Similar logic for MIDI channels
		ch.config.DeviceUID = newDeviceUID
		ch.deviceUID = newDeviceUID
		return nil
		
	default:
		return fmt.Errorf("device change not supported for channel type %T", channel)
	}
}

func (d *Dispatcher) changeOutputDevice(newDeviceUID string) error {
	// Validate new output device exists
	audioDevices, err := devices.GetAudio()
	if err != nil {
		return fmt.Errorf("failed to enumerate audio devices: %w", err)
	}

	device := audioDevices.ByUID(newDeviceUID)
	if device == nil {
		return fmt.Errorf("output device with UID %s not found", newDeviceUID)
	}

	if !device.IsOnline {
		return fmt.Errorf("output device %s is not online", newDeviceUID)
	}

	// Output device change is a major topology change
	// For now, store the new device UID
	d.engine.mu.Lock()
	d.engine.outputDeviceUID = newDeviceUID
	d.engine.mu.Unlock()

	// TODO: Implement actual AVFoundation output device change
	// This requires reconnecting the main mixer to the new output device
	
	return nil
}
