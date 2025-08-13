package format

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/format.m"
#include <stdlib.h>


AudioFormatResult audioformat_new_mono(double sampleRate);
AudioFormatResult audioformat_new_stereo(double sampleRate);
AudioFormatResult audioformat_new_with_channels(double sampleRate, int channels, bool interleaved);
AudioFormatResult audioformat_new_from_spec(double sampleRate, int channels, bool interleaved);
AudioFormatResult audioformat_get_format(AudioFormat* wrapper);
void audioformat_destroy(AudioFormat* wrapper);
double audioformat_get_sample_rate(AudioFormat* wrapper);
int audioformat_get_channel_count(AudioFormat* wrapper);
bool audioformat_is_interleaved(AudioFormat* wrapper);
const char* audioformat_is_equal(AudioFormat* wrapper1, AudioFormat* wrapper2, bool* result);
void audioformat_log_info(AudioFormat* wrapper);
*/
import "C"
import (
	"errors"
	"unsafe"
)

// AudioSpec defines audio format specifications
type AudioSpec struct {
	SampleRate   float64
	ChannelCount int
	Interleaved  bool
}

// Format represents a 1:1 mapping to AVAudioFormat
// This is a pure primitive - no routing assumptions
type Format struct {
	ptr *C.AudioFormat
}

// NewMono creates a new mono format (1 channel, float32, non-interleaved)
func NewMono(sampleRate float64) (*Format, error) {
	result := C.audioformat_new_mono(C.double(sampleRate))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return &Format{ptr: (*C.AudioFormat)(result.result)}, nil
}

// NewStereo creates a new stereo format (2 channels, float32, non-interleaved)
func NewStereo(sampleRate float64) (*Format, error) {
	result := C.audioformat_new_stereo(C.double(sampleRate))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return &Format{ptr: (*C.AudioFormat)(result.result)}, nil
}

// NewWithChannels creates a format with specific channel count and interleaving
func NewWithChannels(sampleRate float64, channels int, interleaved bool) (*Format, error) {
	cInterleaved := C.bool(false)
	if interleaved {
		cInterleaved = C.bool(true)
	}

	result := C.audioformat_new_with_channels(C.double(sampleRate), C.int(channels), cInterleaved)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return &Format{ptr: (*C.AudioFormat)(result.result)}, nil
}

// NewFromSpec creates a format from explicit specifications
func NewFromSpec(spec AudioSpec) (*Format, error) {
	cInterleaved := C.bool(false)
	if spec.Interleaved {
		cInterleaved = C.bool(true)
	}

	result := C.audioformat_new_from_spec(C.double(spec.SampleRate), C.int(spec.ChannelCount), cInterleaved)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return &Format{ptr: (*C.AudioFormat)(result.result)}, nil
}

// GetFormatPtr returns the underlying AVAudioFormat pointer for engine operations
func (f *Format) GetFormatPtr() unsafe.Pointer {
	if f == nil || f.ptr == nil {
		return nil
	}

	// For now, return the wrapper's format pointer directly
	// TODO: Use audioformat_get_format with proper error handling
	return unsafe.Pointer(f.ptr.format)
}

// SampleRate returns the sample rate of the format
func (f *Format) SampleRate() float64 {
	if f == nil || f.ptr == nil {
		return 0.0
	}

	return float64(C.audioformat_get_sample_rate(f.ptr))
}

// ChannelCount returns the number of channels
func (f *Format) ChannelCount() int {
	if f == nil || f.ptr == nil {
		return 0
	}

	return int(C.audioformat_get_channel_count(f.ptr))
}

// IsInterleaved returns true if the format uses interleaved samples
func (f *Format) IsInterleaved() bool {
	if f == nil || f.ptr == nil {
		return false
	}

	return bool(C.audioformat_is_interleaved(f.ptr))
}

// IsEqual compares two formats for equality
func (f *Format) IsEqual(other *Format) bool {
	if f == nil || f.ptr == nil || other == nil || other.ptr == nil {
		return false
	}

	var result C.bool
	errStr := C.audioformat_is_equal(f.ptr, other.ptr, &result)
	if errStr != nil {
		return false
	}

	return bool(result)
}

// ToSpec extracts specifications from an existing format
func (f *Format) ToSpec() AudioSpec {
	if f == nil || f.ptr == nil {
		return AudioSpec{}
	}

	return AudioSpec{
		SampleRate:   f.SampleRate(),
		ChannelCount: f.ChannelCount(),
		Interleaved:  f.IsInterleaved(),
	}
}

// LogInfo logs detailed format information for debugging
func (f *Format) LogInfo() {
	if f == nil || f.ptr == nil {
		return
	}

	C.audioformat_log_info(f.ptr)
}

// Destroy properly tears down the format and frees all resources
func (f *Format) Destroy() {
	if f == nil || f.ptr == nil {
		return
	}

	C.audioformat_destroy(f.ptr)
	f.ptr = nil
}
