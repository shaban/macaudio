package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -L../../ -lmacaudio -Wl,-rpath,/Users/shaban/Code/macaudio
#include "../../native/macaudio.h"
#include <stdlib.h>

// Format function declarations
AudioFormatResult audioformat_new_mono(double sampleRate);
AudioFormatResult audioformat_new_stereo(double sampleRate);
AudioFormatResult audioformat_new_with_channels(double sampleRate, int channels, bool interleaved);
AudioFormatResult audioformat_new_from_spec(double sampleRate, int channels, bool interleaved);
double audioformat_get_sample_rate(AudioFormat* wrapper);
int audioformat_get_channel_count(AudioFormat* wrapper);
bool audioformat_is_interleaved(AudioFormat* wrapper);
const char* audioformat_is_equal(AudioFormat* wrapper1, AudioFormat* wrapper2, bool* result);
void audioformat_log_info(AudioFormat* wrapper);
void audioformat_destroy(AudioFormat* wrapper);
*/
import "C"
import (
	"errors"
	"unsafe"
)

// Format represents a type-safe wrapper around AVAudioFormat
// This provides better type safety than using unsafe.Pointer directly
type Format struct {
	ptr    *C.AudioFormat
	engine *Engine // Reference to the engine that created this format
}

// EnhancedAudioSpec extends the basic AudioSpec with format-specific options
// This consolidates the format package's AudioSpec into the engine package
type EnhancedAudioSpec struct {
	SampleRate   float64 // 44100, 48000, 96000 Hz
	BufferSize   int     // 256, 512, 1024, 2048 samples (engine-specific)
	BitDepth     int     // 16, 24, 32 bits per sample (engine-specific)
	ChannelCount int     // 1 (mono), 2 (stereo), etc.
	Interleaved  bool    // true = interleaved samples, false = non-interleaved (from format package)
}

// NewFormat creates a format with specific specifications
// This consolidates functionality from the old format package
func (e *Engine) NewFormat(spec EnhancedAudioSpec) (*Format, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	if spec.SampleRate <= 0 {
		return nil, errors.New("sample rate must be positive")
	}

	if spec.ChannelCount <= 0 {
		return nil, errors.New("channel count must be positive")
	}

	cInterleaved := C.bool(spec.Interleaved)
	result := C.audioformat_new_from_spec(
		C.double(spec.SampleRate),
		C.int(spec.ChannelCount),
		cInterleaved,
	)

	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	format := &Format{
		ptr:    (*C.AudioFormat)(result.result),
		engine: e,
	}

	return format, nil
}

// NewMonoFormat creates a mono format (1 channel, non-interleaved)
func (e *Engine) NewMonoFormat(sampleRate float64) (*Format, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	if sampleRate <= 0 {
		return nil, errors.New("sample rate must be positive")
	}

	result := C.audioformat_new_mono(C.double(sampleRate))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	format := &Format{
		ptr:    (*C.AudioFormat)(result.result),
		engine: e,
	}

	return format, nil
}

// NewStereoFormat creates a stereo format (2 channels, non-interleaved)
func (e *Engine) NewStereoFormat(sampleRate float64) (*Format, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	if sampleRate <= 0 {
		return nil, errors.New("sample rate must be positive")
	}

	result := C.audioformat_new_stereo(C.double(sampleRate))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	format := &Format{
		ptr:    (*C.AudioFormat)(result.result),
		engine: e,
	}

	return format, nil
}

// NewFormatWithChannels creates a format with specific channel count and interleaving
func (e *Engine) NewFormatWithChannels(sampleRate float64, channels int, interleaved bool) (*Format, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	if sampleRate <= 0 {
		return nil, errors.New("sample rate must be positive")
	}

	if channels <= 0 {
		return nil, errors.New("channel count must be positive")
	}

	cInterleaved := C.bool(interleaved)
	result := C.audioformat_new_with_channels(
		C.double(sampleRate),
		C.int(channels),
		cInterleaved,
	)

	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	format := &Format{
		ptr:    (*C.AudioFormat)(result.result),
		engine: e,
	}

	return format, nil
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
// This creates an EnhancedAudioSpec that can be used to recreate the format
func (f *Format) ToSpec() EnhancedAudioSpec {
	if f == nil || f.ptr == nil {
		return EnhancedAudioSpec{}
	}

	// Use engine's current settings for engine-specific fields
	engineSpec := AudioSpec{}
	if f.engine != nil {
		engineSpec = f.engine.GetSpec()
	}

	return EnhancedAudioSpec{
		SampleRate:   f.SampleRate(),
		BufferSize:   engineSpec.BufferSize, // From engine
		BitDepth:     engineSpec.BitDepth,   // From engine
		ChannelCount: f.ChannelCount(),
		Interleaved:  f.IsInterleaved(),
	}
}

// ToBasicSpec converts to the basic AudioSpec used by the engine
func (f *Format) ToBasicSpec() AudioSpec {
	if f == nil || f.ptr == nil {
		return AudioSpec{}
	}

	// Use engine's current settings for engine-specific fields
	engineSpec := AudioSpec{}
	if f.engine != nil {
		engineSpec = f.engine.GetSpec()
	}

	return AudioSpec{
		SampleRate:   f.SampleRate(),
		BufferSize:   engineSpec.BufferSize,
		BitDepth:     engineSpec.BitDepth,
		ChannelCount: f.ChannelCount(),
	}
}

// LogInfo logs detailed format information for debugging
func (f *Format) LogInfo() {
	if f == nil || f.ptr == nil {
		return
	}

	C.audioformat_log_info(f.ptr)
}

// GetPtr returns the underlying AVAudioFormat pointer for engine operations
// This provides compatibility with existing unsafe.Pointer-based methods
func (f *Format) GetPtr() unsafe.Pointer {
	if f == nil || f.ptr == nil {
		return nil
	}

	// Access the actual AVAudioFormat pointer from the wrapper
	return unsafe.Pointer(f.ptr.format)
}

// Destroy properly tears down the format and frees all resources
func (f *Format) Destroy() {
	if f == nil || f.ptr == nil {
		return
	}

	C.audioformat_destroy(f.ptr)
	f.ptr = nil
	f.engine = nil
}

// ConnectWithTypedFormat connects two nodes using a type-safe Format instead of unsafe.Pointer
// This is a more convenient and safer alternative to Engine.ConnectWithFormat()
func (e *Engine) ConnectWithTypedFormat(sourcePtr, destPtr unsafe.Pointer, fromBus, toBus int, format *Format) error {
	var formatPtr unsafe.Pointer
	if format != nil {
		formatPtr = format.GetPtr()
	}

	return e.ConnectWithFormat(sourcePtr, destPtr, fromBus, toBus, formatPtr)
}

// ConnectWithSpec connects two nodes using an EnhancedAudioSpec
// This creates a format on-the-fly and uses it for the connection
func (e *Engine) ConnectWithSpec(sourcePtr, destPtr unsafe.Pointer, fromBus, toBus int, spec EnhancedAudioSpec) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}

	// Create a format from the spec
	format, err := e.NewFormat(spec)
	if err != nil {
		return errors.New("failed to create format from spec: " + err.Error())
	}
	defer format.Destroy()

	// Use the created format for connection
	return e.ConnectWithFormat(sourcePtr, destPtr, fromBus, toBus, format.GetPtr())
}

// GetEngineFormat creates a format matching the engine's current AudioSpec
// This is useful for creating formats that are compatible with the engine's settings
func (e *Engine) GetEngineFormat() (*Format, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	spec := e.GetSpec()

	// Create enhanced spec from engine spec (assuming non-interleaved)
	enhancedSpec := EnhancedAudioSpec{
		SampleRate:   spec.SampleRate,
		BufferSize:   spec.BufferSize,
		BitDepth:     spec.BitDepth,
		ChannelCount: spec.ChannelCount,
		Interleaved:  false, // Default to non-interleaved
	}

	return e.NewFormat(enhancedSpec)
}

// Common format creation shortcuts for the most frequent use cases

// NewStandardStereoFormat creates the most common stereo format (48kHz, non-interleaved)
// This covers 90% of music and audio playback scenarios
func (e *Engine) NewStandardStereoFormat() (*Format, error) {
	return e.NewStereoFormat(48000)
}

// NewStandardMonoFormat creates the most common mono format (48kHz, non-interleaved)
// This is ideal for voice recordings, phone calls, and single-channel audio
func (e *Engine) NewStandardMonoFormat() (*Format, error) {
	return e.NewMonoFormat(48000)
}

// NewCDAudioFormat creates CD-quality stereo format (44.1kHz, non-interleaved)
// This matches the standard CD audio format
func (e *Engine) NewCDAudioFormat() (*Format, error) {
	return e.NewStereoFormat(44100)
}

// NewInterleavedStereoFormat creates interleaved stereo format
// Use this when you need interleaved samples (less common, but sometimes required)
func (e *Engine) NewInterleavedStereoFormat(sampleRate float64) (*Format, error) {
	return e.NewFormatWithChannels(sampleRate, 2, true)
}
