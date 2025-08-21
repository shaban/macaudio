package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -L../ -lmacaudio -Wl,-rpath,/Users/shaban/Code/macaudio
#include "../native/macaudio.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"errors"
	"unsafe"

	"github.com/shaban/macaudio/devices"
)

// Engine represents the main 8-channel mixing engine
// The engine IS the parameter tree - all state is directly serializable
type Engine struct {
	// Fixed array of 8 channels (not slice)
	Channels     [8]*Channel `json:"channels"`
	MasterVolume float32     `json:"masterVolume"`

	// Engine configuration
	SampleRate int `json:"sampleRate"`
	BufferSize int `json:"bufferSize"`

	// Device assignments
	InputDevice  *devices.AudioDevice `json:"inputDevice,omitempty"`
	OutputDevice *devices.AudioDevice `json:"outputDevice,omitempty"`

	// Internal engine state (not serialized)
	nativeEngine *C.AudioEngine `json:"-"` // Direct C AudioEngine pointer
}

// Channel represents a unified channel that can be input or playback
type Channel struct {
	// Base channel properties
	BusIndex int     `json:"busIndex"`
	Volume   float32 `json:"volume"`
	Pan      float32 `json:"pan"`

	// Optional type-specific data (nil when not applicable)
	PlaybackOptions *PlaybackOptions `json:"playbackOptions,omitempty"`
	InputOptions    *InputOptions    `json:"inputOptions,omitempty"`
}

// IsInput returns true if this is an input channel
func (c *Channel) IsInput() bool {
	return c.InputOptions != nil
}

// IsPlayback returns true if this is a playback channel
func (c *Channel) IsPlayback() bool {
	return c.PlaybackOptions != nil
}

// PlaybackOptions contains playback-specific configuration
type PlaybackOptions struct {
	FilePath string  `json:"filePath"`
	Rate     float32 `json:"rate"`  // 0.25x to 1.25x
	Pitch    float32 `json:"pitch"` // ±12 semitones
}

// InputOptions contains input-specific configuration
type InputOptions struct {
	Device       *devices.AudioDevice `json:"device"`       // Complete device info with capabilities
	ChannelIndex int                  `json:"channelIndex"` // Channel index on the device
	PluginChain  *PluginChain         `json:"pluginChain"`  // Effects chain
}

// NewEngine creates a new 8-channel mixing engine with specified device and settings
// outputDevice: the audio output device to use
// sampleRateIndex: index into the device's supported sample rates (UI friendly)
// bufferSize: buffer size in samples
// NewEngine creates a new audio engine for the specified output device.
//
// Parameters:
//   - outputDevice: The audio device to use for output (from devices package)
//   - sampleRateIndex: Index into the device's SupportedSampleRates array
//   - bufferSize: Requested buffer size in frames (actual size determined by system)
//
// Returns the initialized Engine or an error if creation fails.
// Note: The actual buffer size may differ from the requested size as it's
// controlled by the audio hardware and system preferences.
func NewEngine(outputDevice *devices.AudioDevice, sampleRateIndex int, bufferSize int) (*Engine, error) {
	// Validate output device
	if outputDevice == nil {
		return nil, errors.New("output device cannot be nil")
	}

	// Get the actual sample rate from device capabilities
	if sampleRateIndex < 0 || sampleRateIndex >= len(outputDevice.SupportedSampleRates) {
		return nil, errors.New("invalid sample rate index")
	}
	actualSampleRate := outputDevice.SupportedSampleRates[sampleRateIndex]

	// Validate buffer size
	if bufferSize < 16 {
		return nil, errors.New("buffer size must be at least 16 samples")
	}
	if bufferSize > 2048 {
		return nil, errors.New("buffer size must be at most 2048 samples")
	}

	// Create the native C AudioEngine using AudioEngineResult
	result := C.audioengine_new()
	if result.error != nil {
		errorMsg := C.GoString(result.error)
		return nil, errors.New("failed to create native engine: " + errorMsg)
	}
	if result.result == nil {
		return nil, errors.New("native engine creation returned null")
	}

	// Cast to AudioEngine pointer
	nativeEnginePtr := (*C.AudioEngine)(result.result)

	// Create our engine wrapper
	engine := &Engine{
		SampleRate:   int(actualSampleRate),
		BufferSize:   int(bufferSize), // Note: This is the requested size, actual size may differ
		MasterVolume: 1.0,
		OutputDevice: outputDevice,
		nativeEngine: nativeEnginePtr,
	}

	return engine, nil
}

// Start starts the audio engine. Returns an error if the engine fails to start.
func (e *Engine) Start() error {
	if e.nativeEngine == nil {
		return errors.New("engine is not initialized")
	}

	errorStr := C.audioengine_start(e.nativeEngine)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}

	return nil
}

// Stop stops the audio engine but preserves state
func (e *Engine) Stop() {
	if e.nativeEngine != nil {
		C.audioengine_stop(e.nativeEngine)
	}
}

// Pause pauses the audio engine (similar to Stop but may have different behavior in the C implementation)
func (e *Engine) Pause() {
	C.audioengine_pause(e.nativeEngine)
}

// Prepare prepares the audio engine for playback (sets up audio graph connections)
func (e *Engine) Prepare() {
	C.audioengine_prepare(e.nativeEngine)
}

// Reset resets the audio engine to a clean state
func (e *Engine) Reset() {
	C.audioengine_reset(e.nativeEngine)
}

// Destroy completely shuts down and cleans up the engine
func (e *Engine) Destroy() {
	if e.nativeEngine == nil {
		return // Already destroyed or never initialized
	}

	// Clean up all Channels first (disconnect from mixer buses, etc.)
	for i := range e.Channels {
		if e.Channels[i] != nil {
			e.DestroyChannel(i)
		}
	}

	// Destroy the native C AudioEngine (handles stop, tap removal, reset, and cleanup)
	C.audioengine_destroy(e.nativeEngine)

	// Clear the pointer to prevent double-destroy
	e.nativeEngine = nil
}

// =============================================================================
// Public API - Channel Management
// =============================================================================

// CreateInputChannel creates an input channel connected to an audio device
func (e *Engine) CreateInputChannel(device *devices.AudioDevice, channelIndex int) (*Channel, error) {
	// Find available channel slot
	busIndex := e.findAvailableChannelslot()
	if busIndex == -1 {
		return nil, errors.New("no available channel slots (maximum 8)")
	}

	// TODO: Validate channelIndex is within device's channel count

	channel := &Channel{
		BusIndex: busIndex,
		Volume:   1.0,
		Pan:      0.0,
		InputOptions: &InputOptions{
			Device:       device,
			ChannelIndex: channelIndex,
			PluginChain:  NewPluginChain(),
		},
	}

	e.Channels[busIndex] = channel
	return channel, nil
}

// CreatePlaybackChannel creates a playback channel for an audio file
func (e *Engine) CreatePlaybackChannel(filePath string) (*Channel, error) {
	// Find available channel slot
	busIndex := e.findAvailableChannelslot()
	if busIndex == -1 {
		return nil, errors.New("no available channel slots (maximum 8)")
	}

	// TODO: Validate file format and size (200MB limit)

	channel := &Channel{
		BusIndex: busIndex,
		Volume:   1.0,
		Pan:      0.0,
		PlaybackOptions: &PlaybackOptions{
			FilePath: filePath,
			Rate:     1.0, // Normal playback rate
			Pitch:    0.0, // No pitch shift
		},
	}

	e.Channels[busIndex] = channel
	return channel, nil
} // DestroyChannel removes a channel and frees its bus
func (e *Engine) DestroyChannel(index int) error {
	if index < 0 || index >= 8 {
		return errors.New("invalid channel index (must be 0-7)")
	}

	if e.Channels[index] == nil {
		return errors.New("channel slot already empty")
	}

	// TODO: Disconnect channel from mixer bus
	// TODO: Clean up channel resources

	e.Channels[index] = nil
	return nil
}

// =============================================================================
// Public API - Master Controls
// =============================================================================

// SetMasterVolume sets the master output volume (0.0 to 1.0)
func (e *Engine) SetMasterVolume(volume float32) error {
	// Get the main mixer node first
	result := C.audioengine_main_mixer_node(e.nativeEngine)
	if result.error != nil {
		e.MasterVolume = 0.0 // Safety: any failure in volume setting = assume dangerous state
		return errors.New(C.GoString(result.error))
	}

	// Set volume on the main mixer (C function handles all validation)
	errorStr := C.audioengine_set_mixer_volume(e.nativeEngine, result.result, C.float(volume))
	if errorStr != nil {
		e.MasterVolume = 0.0 // Safety: hardware failure = assume dangerous state
		return errors.New(C.GoString(errorStr))
	}

	e.MasterVolume = volume
	return nil
}

// GetMasterVolume returns the current master volume
func (e *Engine) GetMasterVolume() float32 {
	// Get the main mixer node first
	result := C.audioengine_main_mixer_node(e.nativeEngine)
	if result.error != nil || result.result == nil {
		return 0.0 // Can't access mixer = no sound = volume is effectively 0
	}

	// Get volume from the main mixer
	volume := C.audioengine_get_mixer_volume(e.nativeEngine, result.result)
	e.MasterVolume = float32(volume) // Update cached value for serialization
	return float32(volume)
}

// IsRunning returns true if the engine is currently running
func (e *Engine) IsRunning() bool {
	if e.nativeEngine == nil {
		return false // Engine not initialized
	}

	result := C.audioengine_is_running(e.nativeEngine)
	if result == nil {
		return false // Error occurred or not running
	}

	// The C function returns a string: "true" or "false" (or error message)
	runningStr := C.GoString(result)
	return runningStr == "true"
}

// GetMainMixerNode returns a pointer to the main mixer node for advanced operations
func (e *Engine) GetMainMixerNode() unsafe.Pointer {
	if e.nativeEngine == nil {
		return nil // Engine not initialized
	}

	result := C.audioengine_main_mixer_node(e.nativeEngine)
	if result.error != nil || result.result == nil {
		return nil // Error or null result
	}

	return result.result
}

// =============================================================================
// Public API - State Management
// =============================================================================

// SerializeState exports complete engine state as JSON
func (e *Engine) SerializeState() ([]byte, error) {
	return json.Marshal(e) // Engine IS the parameter tree
}

// DeserializeState imports engine state from JSON
func (e *Engine) DeserializeState(data []byte) error {
	return json.Unmarshal(data, e) // Deserialize directly into engine
}

// =============================================================================
// Private Helper Methods
// =============================================================================

// findAvailableChannelslot returns the first available channel index, or -1 if full
func (e *Engine) findAvailableChannelslot() int {
	for i, channel := range e.Channels {
		if channel == nil {
			return i
		}
	}
	return -1 // All slots occupied
}
