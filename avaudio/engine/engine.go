package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/engine.m"
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
void audioengine_destroy(AudioEngine* wrapper);
const char* audioengine_attach(AudioEngine* wrapper, void* nodePtr);
const char* audioengine_detach(AudioEngine* wrapper, void* nodePtr);
const char* audioengine_connect(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus);
const char* audioengine_connect_with_format(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus, void* formatPtr);
const char* audioengine_set_buffer_size(AudioEngine* wrapper, int bufferSize);
void audioengine_set_mixer_pan(AudioEngine* wrapper, float pan);
const char* audioengine_disconnect_node_input(AudioEngine* wrapper, void* nodePtr, int inputBus);
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

	return &Engine{
		ptr:  (*C.AudioEngine)(result.result),
		spec: spec,
	}, nil
}

// GetSpec returns the audio specifications for this engine
func (e *Engine) GetSpec() AudioSpec {
	if e == nil {
		return AudioSpec{}
	}
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

	return nil
}

// Connect connects two nodes with automatic format handling based on engine's AudioSpec
// This ensures consistent audio quality across all connections in the engine
func (e *Engine) Connect(sourcePtr, destPtr unsafe.Pointer, fromBus, toBus int) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	if sourcePtr == nil || destPtr == nil {
		return errors.New("node pointers cannot be nil")
	}

	// Create an AVAudioFormat from the engine's AudioSpec for proper format control
	formatResult := C.audioengine_create_format(
		C.double(e.spec.SampleRate),
		C.int(e.spec.ChannelCount),
		C.int(e.spec.BitDepth),
	)

	if formatResult.error != nil {
		// Fall back to nil format if we can't create the AudioSpec-based format
		return e.ConnectWithFormat(sourcePtr, destPtr, fromBus, toBus, nil)
	}

	// Use the AudioSpec-based format for connection
	err := e.ConnectWithFormat(sourcePtr, destPtr, fromBus, toBus, unsafe.Pointer(formatResult.result))

	// Clean up the format after use
	C.audioengine_release_format(formatResult.result)

	return err
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
