package sourcenode

import (
	"errors"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/format"
)

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#import <AVFoundation/AVFoundation.h>
#include <math.h>

typedef struct {
    void* sourceNode;  // AVAudioSourceNode*
    double frequency;
    double amplitude;
    double phase;
    double sampleRate;  // Cache the actual sample rate
    int useObjCGeneration;
} AudioSourceNode;

// ============================================================================
// PURE OBJECTIVE-C IMPLEMENTATION
// ============================================================================

// Pure Objective-C sine wave generation - no CGO calls in audio thread
float objc_generate_sine_sample(AudioSourceNode* wrapper) {
    if (!wrapper) return 0.0f;

    float sample = (float)(wrapper->amplitude * sin(wrapper->phase));
    wrapper->phase += 2.0 * M_PI * wrapper->frequency / wrapper->sampleRate; // Use actual sample rate!

    // Clean phase wrapping with fmod - prevents accumulation and drift
    if (wrapper->phase >= 2.0 * M_PI) {
        wrapper->phase = fmod(wrapper->phase, 2.0 * M_PI);
    }

    return sample;
}

// For benchmarking - generate a buffer of samples in pure Objective-C
void audiosourcenode_generate_objc_buffer(AudioSourceNode* wrapper, float* buffer, int frameCount) {
    if (!wrapper || !buffer) return;

    if (wrapper->useObjCGeneration) {
        // Generate audio samples
        for (int i = 0; i < frameCount; i++) {
            buffer[i] = objc_generate_sine_sample(wrapper);
        }
    } else {
        // Generate silence
        for (int i = 0; i < frameCount; i++) {
            buffer[i] = 0.0f;
        }
    }
}

// ============================================================================
// END PURE OBJECTIVE-C IMPLEMENTATION
// ============================================================================

// Create new AVAudioSourceNode with configurable generation method and format
AudioSourceNode* audiosourcenode_new_with_format(int useObjCGeneration, int channelCount) {
    AudioSourceNode* wrapper = malloc(sizeof(AudioSourceNode));
    if (!wrapper) {
        return NULL;
    }

    // Initialize parameters with defaults
    wrapper->frequency = 440.0;
    wrapper->amplitude = 0.5;
    wrapper->phase = 0.0;
    wrapper->sampleRate = 44100.0; // Default fallback, will be updated
    wrapper->useObjCGeneration = useObjCGeneration;

    // Create audio format - mono or stereo
    AVAudioFormat *format;
    if (channelCount == 1) {
        // Mono format for true channel separation
        format = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32
                                                   sampleRate:44100.0
                                                     channels:1
                                                  interleaved:NO];
        NSLog(@"Creating MONO format source node (1 channel)");
    } else {
        // Stereo format (default)
        format = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32
                                                   sampleRate:44100.0
                                                     channels:2
                                                  interleaved:NO];
        NSLog(@"Creating STEREO format source node (2 channels)");
    }

    AVAudioSourceNode* sourceNode = [[AVAudioSourceNode alloc]
        initWithFormat:format  // Use explicit format
        renderBlock:^OSStatus(BOOL *isSilence, const AudioTimeStamp *timestamp, AVAudioFrameCount frameCount, AudioBufferList *outputData) {

            if (wrapper->useObjCGeneration) {
                // ================================================================
                // CLEAN OBJECTIVE-C AUDIO CALLBACK
                // ================================================================
                *isSilence = NO;

                // Generate each frame ONCE, then copy to all channels
                // Channel routing will be handled by AVAudioEngine format/connection logic
                for (AVAudioFrameCount frame = 0; frame < frameCount; frame++) {
                    float sample = objc_generate_sine_sample(wrapper);

                    // Simple approach: same sample to all available channels
                    // AVAudioEngine will handle proper routing based on format
                    for (UInt32 bufferIndex = 0; bufferIndex < outputData->mNumberBuffers; bufferIndex++) {
                        float* buffer = (float*)outputData->mBuffers[bufferIndex].mData;
                        buffer[frame] = sample;
                    }
                }
                // ================================================================
                // END CLEAN OBJECTIVE-C AUDIO CALLBACK
                // ================================================================
            } else {
                // Go generation will fill buffers via separate mechanism
                // For now, output silence and let Go handle it
                *isSilence = YES;
                for (UInt32 i = 0; i < outputData->mNumberBuffers; i++) {
                    memset(outputData->mBuffers[i].mData, 0, outputData->mBuffers[i].mDataByteSize);
                }
            }

            return noErr;
        }];

    if (!sourceNode) {
        free(wrapper);
        return NULL;
    }

    wrapper->sourceNode = (__bridge_retained void*)sourceNode;

    // Get and cache the actual sample rate from the audio format
    AVAudioFormat* outputFormat = [sourceNode outputFormatForBus:0];
    if (outputFormat && outputFormat.sampleRate > 0) {
        wrapper->sampleRate = outputFormat.sampleRate;
        NSLog(@"SourceNode detected sample rate: %.1f Hz, channels: %d, interleaved: %s",
              wrapper->sampleRate,
              outputFormat.channelCount,
              outputFormat.isInterleaved ? "YES" : "NO");
    } else {
        NSLog(@"SourceNode using fallback sample rate: %.1f Hz", wrapper->sampleRate);
    }

    return wrapper;
}

// Compatibility wrapper for existing code (creates stereo by default)
AudioSourceNode* audiosourcenode_new(int useObjCGeneration) {
    return audiosourcenode_new_with_format(useObjCGeneration, 2); // Default to stereo (2 channels)
}

// Set parameters for audio generation
void audiosourcenode_set_frequency(AudioSourceNode* wrapper, double frequency) {
    if (wrapper) {
        wrapper->frequency = frequency;
        // Reset phase to prevent discontinuity when changing frequency
        // This eliminates pops/clicks during frequency transitions
        wrapper->phase = 0.0;
    }
}

void audiosourcenode_set_amplitude(AudioSourceNode* wrapper, double amplitude) {
    if (wrapper) {
        wrapper->amplitude = amplitude;
    }
}

// Get the underlying node pointer for engine operations
void* audiosourcenode_get_node(AudioSourceNode* wrapper) {
    if (!wrapper || !wrapper->sourceNode) {
        return NULL;
    }

    return wrapper->sourceNode;
}

// Get the audio format from the source node
void* audiosourcenode_get_format(AudioSourceNode* wrapper) {
    if (!wrapper || !wrapper->sourceNode) {
        return NULL;
    }

    AVAudioSourceNode* sourceNode = (__bridge AVAudioSourceNode*)wrapper->sourceNode;
    AVAudioFormat* format = [sourceNode outputFormatForBus:0];

    if (format) {
        NSLog(@"Retrieved format: %.0f Hz, %d channels, interleaved: %@",
              format.sampleRate, format.channelCount, format.isInterleaved ? @"YES" : @"NO");
        return (__bridge void*)format;
    }

    return NULL;
}

// Destroy the source node and free resources
void audiosourcenode_destroy(AudioSourceNode* wrapper) {
    if (!wrapper) {
        return;
    }

    if (wrapper->sourceNode) {
        // Release the retained reference
        AVAudioSourceNode* sourceNode = (__bridge_transfer AVAudioSourceNode*)wrapper->sourceNode;
        sourceNode = nil;
        wrapper->sourceNode = NULL;
    }

    // Free the wrapper
    free(wrapper);
}
*/
import "C"

// SourceNode represents a 1:1 mapping to AVAudioSourceNode
type SourceNode struct {
	ptr       *C.AudioSourceNode
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

	ptr := C.audiosourcenode_new(useObjC)
	if ptr == nil {
		return nil, errors.New("failed to create AVAudioSourceNode")
	}

	return &SourceNode{
		ptr:       ptr,
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
	ptr := C.audiosourcenode_new_with_format(useObjC, C.int(channelCount))
	if ptr == nil {
		return nil, errors.New("failed to create AVAudioSourceNode with format")
	}

	return &SourceNode{
		ptr:       ptr,
		frequency: 440.0,
		amplitude: 0.5,
		phase:     0.0,
		format:    audioFormat, // Keep reference to prevent garbage collection
	}, nil
}

// SetFrequency updates the frequency parameter
func (s *SourceNode) SetFrequency(freq float64) {
	if s == nil || s.ptr == nil {
		return
	}

	s.frequency = freq
	C.audiosourcenode_set_frequency(s.ptr, C.double(freq))
}

// SetAmplitude updates the amplitude parameter
func (s *SourceNode) SetAmplitude(amp float64) {
	if s == nil || s.ptr == nil {
		return
	}

	s.amplitude = amp
	C.audiosourcenode_set_amplitude(s.ptr, C.double(amp))
}

// ============================================================================
// OBJECTIVE-C IMPLEMENTATION
// ============================================================================

// GenerateBuffer generates audio samples using Objective-C (for tone nodes) or silence (for silent nodes)
func (s *SourceNode) GenerateBuffer(frameCount int) []float32 {
	if s == nil || s.ptr == nil {
		return nil
	}

	buffer := make([]float32, frameCount)

	// Check if this is a tone-generating node by checking the C struct
	if s.ptr != nil {
		// Call Objective-C generation - it will handle silence vs tone based on useObjCGeneration flag
		C.audiosourcenode_generate_objc_buffer(s.ptr, (*C.float)(unsafe.Pointer(&buffer[0])), C.int(frameCount))
	}

	return buffer
}

// ============================================================================
// END OBJECTIVE-C IMPLEMENTATION
// ============================================================================

// GetNodePtr returns the underlying AVAudioNode pointer for engine operations
func (s *SourceNode) GetNodePtr() unsafe.Pointer {
	if s == nil || s.ptr == nil {
		return nil
	}

	return unsafe.Pointer(C.audiosourcenode_get_node(s.ptr))
}

// GetFormatPtr returns the underlying AVAudioFormat pointer for connections
func (s *SourceNode) GetFormatPtr() unsafe.Pointer {
	if s == nil || s.ptr == nil {
		return nil
	}

	return unsafe.Pointer(C.audiosourcenode_get_format(s.ptr))
}

// Destroy properly tears down the source node and frees all resources
func (s *SourceNode) Destroy() {
	if s == nil || s.ptr == nil {
		return
	}

	C.audiosourcenode_destroy(s.ptr)
	s.ptr = nil
}
