package macaudio

import (
	"fmt"
	"sync"
	"time"
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
	OpCreateAudioInput OperationType = "create_audio_input"
	OpCreateMidiInput  OperationType = "create_midi_input"
	OpCreatePlayback   OperationType = "create_playback"
	OpCreateAux        OperationType = "create_aux"
	OpRemoveChannel    OperationType = "remove_channel"
	OpConnectChannels  OperationType = "connect_channels"
	OpDisconnectChannels OperationType = "disconnect_channels"
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
	d.mu.RLock()
	defer d.mu.RUnlock()
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
			d.mu.Lock()
			d.lastOperationDuration = duration
			if duration > d.maxOperationDuration {
				d.engine.errorHandler.HandleError(
					fmt.Errorf("topology change took %v, target is sub-300ms", duration))
			}
			d.mu.Unlock()
			
			// Send result back
			op.Response <- result
		}
	}
}

// executeOperation executes a single dispatcher operation
func (d *Dispatcher) executeOperation(op DispatcherOperation) DispatcherResult {
	switch op.Type {
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
		
	case OpConnectChannels:
		data := op.Data.(ConnectChannelsData)
		err := d.connectChannels(data.SourceID, data.TargetID, data.Bus)
		return DispatcherResult{Success: err == nil, Error: err}
		
	case OpDisconnectChannels:
		data := op.Data.(DisconnectChannelsData)
		err := d.disconnectChannels(data.SourceID, data.TargetID, data.Bus)
		return DispatcherResult{Success: err == nil, Error: err}
		
	default:
		return DispatcherResult{
			Success: false,
			Error:   fmt.Errorf("unknown operation type: %s", op.Type),
		}
	}
}

// Data structures for dispatcher operations
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
