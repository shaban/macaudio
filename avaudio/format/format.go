package format

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/format.m"
#include <stdlib.h>


AudioFormatResult audioformat_new_mono(double sampleRate);
AudioFormatResult audioformat_new_stereo(double sampleRate);
AudioFormatResult audioformat_new_with_channels(double sampleRate, int channels, bool interleaved);
AudioFormatResult audioformat_copy(AudioFormat* wrapper);
AudioFormatResult audioformat_get_format(AudioFormat* wrapper);
void audioformat_destroy(AudioFormat* wrapper);
double audioformat_get_sample_rate(AudioFormat* wrapper);
int audioformat_get_channel_count(AudioFormat* wrapper);
bool audioformat_is_interleaved(AudioFormat* wrapper);
void audioformat_log_info(AudioFormat* wrapper);
*/
import "C"
import (
	"errors"
	"unsafe"
)

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

// Copy creates a copy of the format
func (f *Format) Copy() (*Format, error) {
	if f == nil || f.ptr == nil {
		return nil, errors.New("format is nil")
	}

	result := C.audioformat_copy(f.ptr)
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

	// For now, just return false until we fix the signature issue
	// TODO: Fix audioformat_is_equal signature mismatch
	return false
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
