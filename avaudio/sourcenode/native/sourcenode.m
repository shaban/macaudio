#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>
#include <math.h>

// Result structures for functions that return pointers
typedef struct {
    void* result;           // The actual result pointer
    const char* error;      // NULL = success, string = error message  
} AudioSourceNodeResult;

typedef struct {
    void* sourceNode;  // AVAudioSourceNode*
    double frequency;
    double amplitude;
    double phase;
    double sampleRate;  // Cache the actual sample rate
    int useObjCGeneration;
} AudioSourceNode;

// ============================================================================
// PURE OBJECTIVE-C IMPLEMENTATION - PERFORMANCE CRITICAL
// ============================================================================

// Pure Objective-C sine wave generation - no CGO calls in audio thread
float objc_generate_sine_sample(void* wrapper) {
    if (!wrapper) return 0.0f;

    AudioSourceNode* sourceNode = (AudioSourceNode*)wrapper;
    float sample = (float)(sourceNode->amplitude * sin(sourceNode->phase));
    sourceNode->phase += 2.0 * M_PI * sourceNode->frequency / sourceNode->sampleRate; // Use actual sample rate!

    // Clean phase wrapping with fmod - prevents accumulation and drift
    if (sourceNode->phase >= 2.0 * M_PI) {
        sourceNode->phase = fmod(sourceNode->phase, 2.0 * M_PI);
    }

    return sample;
}

// For benchmarking - generate a buffer of samples in pure Objective-C
const char* audiosourcenode_generate_objc_buffer(void* wrapper, float* buffer, int frameCount) {
    if (!wrapper) {
        return "AudioSourceNode wrapper is null";
    }
    if (!buffer) {
        return "Buffer pointer is null";
    }
    if (frameCount <= 0) {
        return "Frame count must be positive";
    }

    AudioSourceNode* sourceNode = (AudioSourceNode*)wrapper;

    @try {
        if (sourceNode->useObjCGeneration) {
            // Generate audio samples
            for (int i = 0; i < frameCount; i++) {
                buffer[i] = objc_generate_sine_sample(sourceNode);
            }
        } else {
            // Generate silence
            for (int i = 0; i < frameCount; i++) {
                buffer[i] = 0.0f;
            }
        }
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to generate audio buffer: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

// ============================================================================
// END PURE OBJECTIVE-C IMPLEMENTATION
// ============================================================================

// Create new AVAudioSourceNode with configurable generation method and format
AudioSourceNodeResult audiosourcenode_new_with_format(int useObjCGeneration, int channelCount) {
    if (channelCount <= 0 || channelCount > 8) {
        return (AudioSourceNodeResult){NULL, "Channel count must be between 1 and 8"};
    }

    @try {
        AudioSourceNode* wrapper = malloc(sizeof(AudioSourceNode));
        if (!wrapper) {
            return (AudioSourceNodeResult){NULL, "Failed to allocate memory for AudioSourceNode"};
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
            // Multi-channel format
            format = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32
                                                       sampleRate:44100.0
                                                         channels:channelCount
                                                      interleaved:NO];
            NSLog(@"Creating %d-channel format source node", channelCount);
        }

        if (!format) {
            free(wrapper);
            return (AudioSourceNodeResult){NULL, "Failed to create audio format"};
        }

        AVAudioSourceNode* sourceNode = [[AVAudioSourceNode alloc]
            initWithFormat:format  // Use explicit format
            renderBlock:^OSStatus(BOOL *isSilence, const AudioTimeStamp *timestamp, AVAudioFrameCount frameCount, AudioBufferList *outputData) {

                if (wrapper->useObjCGeneration) {
                    // ================================================================
                    // CLEAN OBJECTIVE-C AUDIO CALLBACK - PERFORMANCE CRITICAL
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
            return (AudioSourceNodeResult){NULL, "Failed to create AVAudioSourceNode"};
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

        return (AudioSourceNodeResult){wrapper, NULL};
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to create source node: %@", exception.reason];
        return (AudioSourceNodeResult){NULL, [errorMsg UTF8String]};
    }
}

// Compatibility wrapper for existing code (creates stereo by default)
AudioSourceNodeResult audiosourcenode_new(int useObjCGeneration) {
    return audiosourcenode_new_with_format(useObjCGeneration, 2); // Default to stereo (2 channels)
}

// Set parameters for audio generation
const char* audiosourcenode_set_frequency(void* wrapper, double frequency) {
    if (!wrapper) {
        return "AudioSourceNode wrapper is null";
    }
    if (frequency < 0.0) {
        return "Frequency cannot be negative";
    }
    if (frequency > 22050.0) {
        return "Frequency cannot exceed Nyquist limit (22050 Hz)";
    }

    AudioSourceNode* sourceNode = (AudioSourceNode*)wrapper;
    @try {
        sourceNode->frequency = frequency;
        // Reset phase to prevent discontinuity when changing frequency
        // This eliminates pops/clicks during frequency transitions
        sourceNode->phase = 0.0;
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to set frequency: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

const char* audiosourcenode_set_amplitude(void* wrapper, double amplitude) {
    if (!wrapper) {
        return "AudioSourceNode wrapper is null";
    }
    if (amplitude < 0.0 || amplitude > 1.0) {
        return "Amplitude must be between 0.0 and 1.0";
    }

    AudioSourceNode* sourceNode = (AudioSourceNode*)wrapper;
    @try {
        sourceNode->amplitude = amplitude;
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to set amplitude: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

// Get the underlying node pointer for engine operations
AudioSourceNodeResult audiosourcenode_get_node(void* wrapper) {
    if (!wrapper) {
        return (AudioSourceNodeResult){NULL, "AudioSourceNode wrapper is null"};
    }
    
    AudioSourceNode* sourceNode = (AudioSourceNode*)wrapper;
    if (!sourceNode->sourceNode) {
        return (AudioSourceNodeResult){NULL, "Source node is null (already destroyed?)"};
    }

    return (AudioSourceNodeResult){sourceNode->sourceNode, NULL};
}

// Get the audio format from the source node
AudioSourceNodeResult audiosourcenode_get_format(void* wrapper) {
    if (!wrapper) {
        return (AudioSourceNodeResult){NULL, "AudioSourceNode wrapper is null"};
    }
    
    AudioSourceNode* sourceNode = (AudioSourceNode*)wrapper;
    if (!sourceNode->sourceNode) {
        return (AudioSourceNodeResult){NULL, "Source node is null (already destroyed?)"};
    }

    @try {
        AVAudioSourceNode* node = (__bridge AVAudioSourceNode*)sourceNode->sourceNode;
        AVAudioFormat* format = [node outputFormatForBus:0];

        if (format) {
            NSLog(@"Retrieved format: %.0f Hz, %d channels, interleaved: %@",
                  format.sampleRate, format.channelCount, format.isInterleaved ? @"YES" : @"NO");
            return (AudioSourceNodeResult){(__bridge void*)format, NULL};
        } else {
            return (AudioSourceNodeResult){NULL, "No format available for output bus 0"};
        }
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to get format: %@", exception.reason];
        return (AudioSourceNodeResult){NULL, [errorMsg UTF8String]};
    }
}

// Destroy the source node and free resources
const char* audiosourcenode_destroy(void* wrapper) {
    if (!wrapper) {
        return "AudioSourceNode wrapper is null";
    }

    AudioSourceNode* sourceNode = (AudioSourceNode*)wrapper;
    @try {
        if (sourceNode->sourceNode) {
            // Release the retained reference
            AVAudioSourceNode* node = (__bridge_transfer AVAudioSourceNode*)sourceNode->sourceNode;
            node = nil;
            sourceNode->sourceNode = NULL;
        }

        // Free the wrapper
        free(sourceNode);
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to destroy source node: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}
