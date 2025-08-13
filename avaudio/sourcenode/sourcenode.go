package sourcenode

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/sourcenode.m"
#include <stdlib.h>

// Function declarations - CGO resolves AudioSourceNodeResult from .m file
AudioSourceNodeResult audiosourcenode_new(int useObjCGeneration);
AudioSourceNodeResult audiosourcenode_new_with_format(int useObjCGeneration, int channelCount);
const char* audiosourcenode_set_frequency(void* wrapper, double frequency);
const char* audiosourcenode_set_amplitude(void* wrapper, double amplitude);
AudioSourceNodeResult audiosourcenode_get_node(void* wrapper);
AudioSourceNodeResult audiosourcenode_get_format(void* wrapper);
const char* audiosourcenode_generate_objc_buffer(void* wrapper, float* buffer, int frameCount);
const char* audiosourcenode_destroy(void* wrapper);
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/format"
)

// SourceNode represents a 1:1 mapping to AVAudioSourceNode
type SourceNode struct {
	ptr       unsafe.Pointer
	frequency float64
	amplitude float64
	phase     float64
	format    *format.Format // Keep reference to prevent garbage collection
}

// New creates a new AVAudioSourceNode instance
// useObjCGeneration: true for pure Objective-C audio generation, false for silence
func New(useObjCGeneration bool) (*SourceNode, error) {
	var useObjC C.int
	if useObjCGeneration {
		useObjC = 1
	}

	result := C.audiosourcenode_new(useObjC)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	if result.result == nil {
		return nil, errors.New("failed to create AVAudioSourceNode")
	}

	return &SourceNode{
		ptr:       unsafe.Pointer(result.result),
		frequency: 440.0,
		amplitude: 0.5,
		phase:     0.0,
	}, nil
}

// NewSilent creates a new silent AVAudioSourceNode (for compatibility with existing tests)
func NewSilent() (*SourceNode, error) {
	return New(false) // Use silence generation
}

// NewTone creates a new AVAudioSourceNode that generates audio using Objective-C (stereo format)
func NewTone() (*SourceNode, error) {
	return New(true) // Use Objective-C generation, stereo format
}

// NewMonoTone creates a new AVAudioSourceNode with mono format for proper channel routing
func NewMonoTone() (*SourceNode, error) {
	monoFormat, err := format.NewMono(44100.0)
	if err != nil {
		return nil, err
	}

	return NewWithFormat(monoFormat, true) // Use Objective-C generation with mono format
}

// NewWithFormat creates a new AVAudioSourceNode with the specified format
func NewWithFormat(audioFormat *format.Format, useObjCGeneration bool) (*SourceNode, error) {
	if audioFormat == nil {
		return nil, errors.New("audio format cannot be nil")
	}

	var useObjC C.int = 0
	if useObjCGeneration {
		useObjC = 1
	}

	formatPtr := audioFormat.GetFormatPtr()
	if formatPtr == nil {
		return nil, errors.New("invalid format pointer")
	}

	channelCount := audioFormat.ChannelCount()
	result := C.audiosourcenode_new_with_format(useObjC, C.int(channelCount))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	if result.result == nil {
		return nil, errors.New("failed to create AVAudioSourceNode with format")
	}

	return &SourceNode{
		ptr:       unsafe.Pointer(result.result),
		frequency: 440.0,
		amplitude: 0.5,
		phase:     0.0,
		format:    audioFormat, // Keep reference to prevent garbage collection
	}, nil
}

// SetFrequency updates the frequency parameter
func (s *SourceNode) SetFrequency(freq float64) error {
	if s == nil || s.ptr == nil {
		return errors.New("source node is nil or destroyed")
	}

	s.frequency = freq
	errorStr := C.audiosourcenode_set_frequency(s.ptr, C.double(freq))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// SetAmplitude updates the amplitude parameter
func (s *SourceNode) SetAmplitude(amp float64) error {
	if s == nil || s.ptr == nil {
		return errors.New("source node is nil or destroyed")
	}

	s.amplitude = amp
	errorStr := C.audiosourcenode_set_amplitude(s.ptr, C.double(amp))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// ============================================================================
// OBJECTIVE-C IMPLEMENTATION
// ============================================================================

// GenerateBuffer generates audio samples using Objective-C (for tone nodes) or silence (for silent nodes)
func (s *SourceNode) GenerateBuffer(frameCount int) ([]float32, error) {
	if s == nil || s.ptr == nil {
		return nil, errors.New("source node is nil or destroyed")
	}

	if frameCount <= 0 {
		return nil, errors.New("frame count must be positive")
	}

	buffer := make([]float32, frameCount)

	// Call Objective-C generation - it will handle silence vs tone based on useObjCGeneration flag
	errorStr := C.audiosourcenode_generate_objc_buffer(s.ptr, (*C.float)(unsafe.Pointer(&buffer[0])), C.int(frameCount))
	if errorStr != nil {
		return nil, errors.New(C.GoString(errorStr))
	}

	return buffer, nil
}

// ============================================================================
// END OBJECTIVE-C IMPLEMENTATION
// ============================================================================

// GetNodePtr returns the underlying AVAudioNode pointer for engine operations
func (s *SourceNode) GetNodePtr() (unsafe.Pointer, error) {
	if s == nil || s.ptr == nil {
		return nil, errors.New("source node is nil or destroyed")
	}

	result := C.audiosourcenode_get_node(s.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	return unsafe.Pointer(result.result), nil
}

// GetFormatPtr returns the underlying AVAudioFormat pointer for connections
func (s *SourceNode) GetFormatPtr() (unsafe.Pointer, error) {
	if s == nil || s.ptr == nil {
		return nil, errors.New("source node is nil or destroyed")
	}

	result := C.audiosourcenode_get_format(s.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	return unsafe.Pointer(result.result), nil
}

// Destroy properly tears down the source node and frees all resources
func (s *SourceNode) Destroy() error {
	if s == nil || s.ptr == nil {
		return errors.New("source node is nil or already destroyed")
	}

	errorStr := C.audiosourcenode_destroy(s.ptr)
	s.ptr = nil
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}
