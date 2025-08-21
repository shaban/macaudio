package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -L../../ -lmacaudio -Wl,-rpath,/Users/shaban/Code/macaudio
#include "../../native/macaudio.h"
// Function declarations for CGO
AudioEngineResult audioengine_new();
void audioengine_prepare(AudioEngine* wrapper);
const char* audioengine_start(AudioEngine* wrapper);
void audioengine_stop(AudioEngine* wrapper);
void audioengine_pause(AudioEngine* wrapper);
void audioengine_reset(AudioEngine* wrapper);
const char* audioengine_is_running(AudioEngine* wrapper);
void audioengine_remove_taps(AudioEngine* wrapper);
AudioEngineResult audioengine_output_node(AudioEngine* wrapper);
AudioEngineResult audioengine_input_node(AudioEngine* wrapper);
AudioEngineResult audioengine_main_mixer_node(AudioEngine* wrapper);
AudioEngineResult audioengine_create_mixer_node(AudioEngine* wrapper);
void audioengine_destroy(AudioEngine* wrapper);
const char* audioengine_attach(AudioEngine* wrapper, void* nodePtr);
const char* audioengine_detach(AudioEngine* wrapper, void* nodePtr);
const char* audioengine_connect(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus);
const char* audioengine_connect_with_format(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus, void* formatPtr);
void audioengine_set_mixer_pan(AudioEngine* wrapper, float pan);
const char* audioengine_disconnect_node_input(AudioEngine* wrapper, void* nodePtr, int inputBus);
AudioEngineResult audioengine_create_format(double sampleRate, int channelCount, int bitDepth);
void audioengine_release_format(void* formatPtr);
// NOTE: The above C format functions are legacy - Go code now uses the consolidated format system in format.go
const char* audioengine_set_buffer_size(AudioEngine* wrapper, int bufferSize);
const char* audioengine_set_mixer_volume(AudioEngine* wrapper, void* mixerNodePtr, float volume);
float audioengine_get_mixer_volume(AudioEngine* wrapper, void* mixerNodePtr);
const char* audioengine_disconnect_node_output(AudioEngine* wrapper, void* nodePtr, int outputBus);
const char* audioengine_wait_for_readiness(AudioEngine* wrapper, double timeoutSeconds);
const char* audioengine_is_ready_for_playback(AudioEngine* wrapper, bool* isReady);
const char* audioengine_prime_with_silence(AudioEngine* wrapper, double timeoutSeconds);
*/
import "C"
import (
	"context"
	"errors"
	"time"
	"unsafe"
)

// AudioSpec defines the foundational audio settings for an engine
type AudioSpec struct {
	SampleRate   float64 // 44100, 48000, 96000 Hz
	BufferSize   int     // 256, 512, 1024, 2048 samples
	BitDepth     int     // 16, 24, 32 bits per sample
	ChannelCount int     // 1 (mono), 2 (stereo)
}

// DefaultAudioSpec returns commonly used audio settings
func DefaultAudioSpec() AudioSpec {
	return AudioSpec{
		SampleRate:   48000, // Common modern default
		BufferSize:   512,   // Balanced latency/performance
		BitDepth:     32,    // Engines use 32-bit float internally
		ChannelCount: 2,     // Stereo
	}
}

// Engine represents a 1:1 mapping to AVAudioEngine with audio specifications
type Engine struct {
	ptr  *C.AudioEngine
	spec AudioSpec

	// Connection tracking for smart per-bus control (optional enhancement)
	// mixerConnections maps mixer pointer -> bus -> source pointer
	mixerConnections map[unsafe.Pointer]map[int]unsafe.Pointer
}

// New creates a new AVAudioEngine instance with specified audio settings
func New(spec AudioSpec) (*Engine, error) {
	result := C.audioengine_new()
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	if result.result == nil {
		return nil, errors.New("engine creation returned null pointer")
	}

	engine := &Engine{
		ptr:              (*C.AudioEngine)(result.result),
		spec:             spec,
		mixerConnections: make(map[unsafe.Pointer]map[int]unsafe.Pointer),
	}

	// Apply the specified buffer size immediately after creation
	if spec.BufferSize > 0 {
		if err := engine.SetBufferSize(spec.BufferSize); err != nil {
			// If we can't set the buffer size, clean up and return error
			engine.Destroy()
			return nil, errors.New("failed to set buffer size: " + err.Error())
		}
	}

	return engine, nil
}

// GetNativeEngine returns the native AVAudioEngine pointer for taps
func (e *Engine) GetNativeEngine() unsafe.Pointer {
	if e.ptr != nil {
		return unsafe.Pointer(e.ptr.engine) // Access the actual AVAudioEngine
	}
	return nil
}

// GetSpec returns the engine's audio specification
func (e *Engine) GetSpec() AudioSpec {
	return e.spec
}

// SetBufferSize changes the buffer size at runtime for performance optimization
// This is your "fix dropouts" use case - can be changed while engine is running
func (e *Engine) SetBufferSize(size int) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if size <= 0 {
		return errors.New("buffer size must be positive")
	}

	// Update the spec to reflect the new buffer size
	e.spec.BufferSize = size

	// Call the C function to actually change AVAudioEngine buffer size
	result := C.audioengine_set_buffer_size(e.ptr, C.int(size))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// Prepare prepares the engine for starting
func (e *Engine) Prepare() {
	if e == nil || e.ptr == nil {
		return
	}

	C.audioengine_prepare(e.ptr)
}

// Start starts the engine
func (e *Engine) Start() error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	result := C.audioengine_start(e.ptr)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// Stop stops the engine
func (e *Engine) Stop() {
	if e == nil || e.ptr == nil {
		return
	}

	C.audioengine_stop(e.ptr)
}

// Pause pauses the engine
func (e *Engine) Pause() {
	if e.ptr == nil {
		return
	}

	C.audioengine_pause(e.ptr)
}

// Reset resets the engine
func (e *Engine) Reset() {
	if e == nil || e.ptr == nil {
		return
	}

	C.audioengine_reset(e.ptr)
}

// IsRunning returns true if the engine is running
func (e *Engine) IsRunning() bool {
	if e == nil || e.ptr == nil {
		return false
	}

	result := C.audioengine_is_running(e.ptr)
	// For is_running, NULL means engine is running (success), non-NULL means not running
	return result == nil
}

// OutputNode returns the output node as an unsafe.Pointer
func (e *Engine) OutputNode() (unsafe.Pointer, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	result := C.audioengine_output_node(e.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return unsafe.Pointer(result.result), nil
}

// InputNode returns the input node as an unsafe.Pointer
func (e *Engine) InputNode() (unsafe.Pointer, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	result := C.audioengine_input_node(e.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return unsafe.Pointer(result.result), nil
}

// MainMixerNode returns the main mixer node as an unsafe.Pointer
func (e *Engine) MainMixerNode() (unsafe.Pointer, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	result := C.audioengine_main_mixer_node(e.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return unsafe.Pointer(result.result), nil
}

// CreateMixerNode creates a new individual mixer node for channels
func (e *Engine) CreateMixerNode() (unsafe.Pointer, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	result := C.audioengine_create_mixer_node(e.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return unsafe.Pointer(result.result), nil
}

// SetMixerVolume sets the volume of a specific mixer node
func (e *Engine) SetMixerVolume(mixerNodePtr unsafe.Pointer, volume float32) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if mixerNodePtr == nil {
		return errors.New("mixer node pointer is nil")
	}

	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}

	result := C.audioengine_set_mixer_volume(e.ptr, mixerNodePtr, C.float(volume))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// GetMixerVolume gets the volume of a specific mixer node
func (e *Engine) GetMixerVolume(mixerNodePtr unsafe.Pointer) (float32, error) {
	if e == nil || e.ptr == nil {
		return 0.0, errors.New("engine is nil")
	}

	if mixerNodePtr == nil {
		return 0.0, errors.New("mixer node pointer is nil")
	}

	volume := C.audioengine_get_mixer_volume(e.ptr, mixerNodePtr)
	return float32(volume), nil
}

// Destroy properly tears down the engine and frees all resources
func (e *Engine) Destroy() {
	if e == nil || e.ptr == nil {
		return
	}

	C.audioengine_destroy(e.ptr)
	e.ptr = nil
}

// Attach attaches a node to the engine - 1:1 mapping to attachNode:
func (e *Engine) Attach(nodePtr unsafe.Pointer) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}

	result := C.audioengine_attach(e.ptr, nodePtr)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// Detach detaches a node from the engine - 1:1 mapping to detachNode:
func (e *Engine) Detach(nodePtr unsafe.Pointer) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}

	errorStr := C.audioengine_detach(e.ptr, nodePtr)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}

	return nil
}

// ConnectWithFormat connects two nodes with an explicit audio format
func (e *Engine) ConnectWithFormat(sourcePtr, destPtr unsafe.Pointer, fromBus, toBus int, formatPtr unsafe.Pointer) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if sourcePtr == nil || destPtr == nil {
		return errors.New("node pointers cannot be nil")
	}

	errorStr := C.audioengine_connect_with_format(e.ptr, sourcePtr, destPtr, C.int(fromBus), C.int(toBus), formatPtr)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}

	// Track connection for smart per-bus control
	e.trackConnection(sourcePtr, destPtr, toBus)

	return nil
}

// trackConnection records a source->dest connection for smart per-bus control
func (e *Engine) trackConnection(sourcePtr, destPtr unsafe.Pointer, toBus int) {
	if e.mixerConnections == nil {
		e.mixerConnections = make(map[unsafe.Pointer]map[int]unsafe.Pointer)
	}

	// Create bus map for this mixer if it doesn't exist
	if e.mixerConnections[destPtr] == nil {
		e.mixerConnections[destPtr] = make(map[int]unsafe.Pointer)
	}

	// Record the connection: mixer[bus] -> source
	e.mixerConnections[destPtr][toBus] = sourcePtr
}

// untrackConnection removes a connection record
func (e *Engine) untrackConnection(destPtr unsafe.Pointer, inputBus int) {
	if e.mixerConnections == nil {
		return
	}

	if busMap, exists := e.mixerConnections[destPtr]; exists {
		delete(busMap, inputBus)

		// Clean up empty mixer map
		if len(busMap) == 0 {
			delete(e.mixerConnections, destPtr)
		}
	}
}

// this is unused!!!
// getConnectedSources returns sources connected to a mixer as arrays for C interop
/*func (e *Engine) getConnectedSources(mixerPtr unsafe.Pointer) ([]unsafe.Pointer, int) {
	if e.mixerConnections == nil {
		return nil, 0
	}

	busMap, exists := e.mixerConnections[mixerPtr]
	if !exists {
		return nil, 0
	}

	// Find maximum bus number to size array
	maxBus := 0
	for bus := range busMap {
		if bus > maxBus {
			maxBus = bus
		}
	}

	// Create source array (nil for unconnected buses)
	sources := make([]unsafe.Pointer, maxBus+1)
	for bus, source := range busMap {
		sources[bus] = source
	}

	return sources, len(sources)
}*/

// Connect connects two nodes with automatic format handling based on engine's AudioSpec
// This ensures consistent audio quality across all connections in the engine
// Now uses the consolidated format system for better efficiency and type safety
func (e *Engine) Connect(sourcePtr, destPtr unsafe.Pointer, fromBus, toBus int) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if sourcePtr == nil || destPtr == nil {
		return errors.New("node pointers cannot be nil")
	}

	// Use the consolidated format system instead of inline C format creation
	engineFormat, err := e.GetEngineFormat()
	if err != nil {
		// Fall back to nil format if we can't create the engine-compatible format
		return e.ConnectWithFormat(sourcePtr, destPtr, fromBus, toBus, nil)
	}
	defer engineFormat.Destroy() // Automatic cleanup with proper lifecycle management

	// Use the consolidated format for connection
	return e.ConnectWithFormat(sourcePtr, destPtr, fromBus, toBus, engineFormat.GetPtr())
}

// SetMixerPan sets the pan of the main mixer node (-1.0 = hard left, 0.0 = center, 1.0 = hard right)
func (e *Engine) SetMixerPan(pan float32) {
	if e == nil || e.ptr == nil {
		return
	}

	C.audioengine_set_mixer_pan(e.ptr, (C.float)(pan))
}

// DisconnectNodeInput disconnects a specific input bus of a node from any connected source
// This is useful for dynamic routing changes and breaking connections
func (e *Engine) DisconnectNodeInput(nodePtr unsafe.Pointer, inputBus int) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}

	if inputBus < 0 {
		return errors.New("input bus cannot be negative")
	}

	errorStr := C.audioengine_disconnect_node_input(e.ptr, nodePtr, C.int(inputBus))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}

	// Remove connection tracking
	e.untrackConnection(nodePtr, inputBus)

	return nil
}

// DisconnectNodeOutput disconnects a specific output bus of a node from any connected destination
// This is useful for breaking outgoing connections when rerouting audio
func (e *Engine) DisconnectNodeOutput(nodePtr unsafe.Pointer, outputBus int) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}

	if outputBus < 0 {
		return errors.New("output bus cannot be negative")
	}

	errorStr := C.audioengine_disconnect_node_output(e.ptr, nodePtr, C.int(outputBus))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}

	// Note: Output disconnection is harder to track since we don't know which destination
	// was disconnected. For now, we'll let the connection tracking be eventually consistent.
	// TODO: Consider more sophisticated connection tracking for bidirectional cleanup

	return nil
}

// Ptr returns the unsafe.Pointer to the underlying AVAudioEngine for use with other packages
func (e *Engine) Ptr() unsafe.Pointer {
	if e == nil || e.ptr == nil {
		return nil
	}
	return unsafe.Pointer(e.ptr.engine)
}

// StartWith starts the engine honoring a context deadline. No validation is performed here.
// The managed layer is responsible for graph validation and safety policies.
func (e *Engine) StartWith(ctx context.Context, mute bool) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}
	// Optional best-effort mute when requested
	// Note: we intentionally avoid default muting at engine.New; callers control this.
	// Prepare prior to start
	e.Prepare()
	errCh := make(chan error, 1)
	go func() { errCh <- e.Start() }()
	if deadline, ok := ctx.Deadline(); ok {
		select {
		case err := <-errCh:
			return err
		case <-time.After(time.Until(deadline)):
			e.Stop()
			return ctx.Err()
		}
	}
	return <-errCh
}
