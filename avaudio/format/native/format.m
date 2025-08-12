#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>
#import <stdlib.h>
#include <stdbool.h>

typedef struct {
    void* format;
} AudioFormat;

typedef struct {
    void* result;
    const char* error;
} AudioFormatResult;

// Create mono format (1 channel, non-interleaved, float32)
AudioFormatResult audioformat_new_mono(double sampleRate) {
    if (sampleRate <= 0) {
        return (AudioFormatResult){NULL, "Sample rate must be positive"};
    }
    
    AVAudioFormat* format = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32
                                                             sampleRate:sampleRate
                                                               channels:1
                                                            interleaved:NO];
    
    if (!format) {
        return (AudioFormatResult){NULL, "Failed to create mono audio format"};
    }
    
    AudioFormat* wrapper = malloc(sizeof(AudioFormat));
    if (!wrapper) {
        return (AudioFormatResult){NULL, "Memory allocation failed"};
    }
    
    wrapper->format = (__bridge_retained void*)format;
    NSLog(@"Created MONO format: %.0f Hz, 1 channel", sampleRate);
    return (AudioFormatResult){wrapper, NULL};  // NULL = success
}

// Create stereo format (2 channels, non-interleaved, float32)
AudioFormatResult audioformat_new_stereo(double sampleRate) {
    if (sampleRate <= 0) {
        return (AudioFormatResult){NULL, "Sample rate must be positive"};
    }
    
    AVAudioFormat* format = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32
                                                             sampleRate:sampleRate
                                                               channels:2
                                                            interleaved:NO];
    
    if (!format) {
        return (AudioFormatResult){NULL, "Failed to create stereo audio format"};
    }
    
    AudioFormat* wrapper = malloc(sizeof(AudioFormat));
    if (!wrapper) {
        return (AudioFormatResult){NULL, "Memory allocation failed"};
    }
    
    wrapper->format = (__bridge_retained void*)format;
    NSLog(@"Created STEREO format: %.0f Hz, 2 channels", sampleRate);
    return (AudioFormatResult){wrapper, NULL};  // NULL = success
}

// Create format with specific channel count and interleaving
AudioFormatResult audioformat_new_with_channels(double sampleRate, int channels, bool interleaved) {
    if (sampleRate <= 0) {
        return (AudioFormatResult){NULL, "Sample rate must be positive"};
    }
    
    if (channels <= 0) {
        return (AudioFormatResult){NULL, "Channel count must be positive"};
    }
    
    AVAudioFormat* format = [[AVAudioFormat alloc] initWithCommonFormat:AVAudioPCMFormatFloat32
                                                             sampleRate:sampleRate
                                                               channels:channels
                                                            interleaved:interleaved];
    
    if (!format) {
        return (AudioFormatResult){NULL, "Failed to create audio format with specified channels"};
    }
    
    AudioFormat* wrapper = malloc(sizeof(AudioFormat));
    if (!wrapper) {
        return (AudioFormatResult){NULL, "Memory allocation failed"};
    }
    
    wrapper->format = (__bridge_retained void*)format;
    NSLog(@"Created %d-channel format: %.0f Hz, %s", 
          channels, sampleRate, interleaved ? "interleaved" : "non-interleaved");
    return (AudioFormatResult){wrapper, NULL};  // NULL = success
}

// Copy an existing format
AudioFormatResult audioformat_copy(AudioFormat* wrapper) {
    if (!wrapper) {
        return (AudioFormatResult){NULL, "Format pointer is null"};
    }
    
    if (!wrapper->format) {
        return (AudioFormatResult){NULL, "Format object is null"};
    }
    
    AVAudioFormat* originalFormat = (__bridge AVAudioFormat*)wrapper->format;
    
    AVAudioFormat* newFormat = [[AVAudioFormat alloc] initWithCommonFormat:originalFormat.commonFormat
                                                               sampleRate:originalFormat.sampleRate
                                                                 channels:originalFormat.channelCount
                                                              interleaved:originalFormat.isInterleaved];
    
    if (!newFormat) {
        return (AudioFormatResult){NULL, "Failed to create copy of audio format"};
    }
    
    AudioFormat* newWrapper = malloc(sizeof(AudioFormat));
    if (!newWrapper) {
        return (AudioFormatResult){NULL, "Memory allocation failed"};
    }
    
    newWrapper->format = (__bridge_retained void*)newFormat;
    return (AudioFormatResult){newWrapper, NULL};  // NULL = success
}

// Get the underlying AVAudioFormat pointer for engine operations
AudioFormatResult audioformat_get_format(AudioFormat* wrapper) {
    if (!wrapper) {
        return (AudioFormatResult){NULL, "Format pointer is null"};
    }
    
    if (!wrapper->format) {
        return (AudioFormatResult){NULL, "Format object is null"};
    }
    
    return (AudioFormatResult){wrapper->format, NULL};  // NULL = success
}

// Get sample rate
double audioformat_get_sample_rate(AudioFormat* wrapper) {
    if (!wrapper || !wrapper->format) {
        return 0.0;
    }
    
    AVAudioFormat* format = (__bridge AVAudioFormat*)wrapper->format;
    return format.sampleRate;
}

// Get channel count
int audioformat_get_channel_count(AudioFormat* wrapper) {
    if (!wrapper || !wrapper->format) {
        return 0;
    }
    
    AVAudioFormat* format = (__bridge AVAudioFormat*)wrapper->format;
    return (int)format.channelCount;
}

// Check if interleaved
bool audioformat_is_interleaved(AudioFormat* wrapper) {
    if (!wrapper || !wrapper->format) {
        return false;
    }
    
    AVAudioFormat* format = (__bridge AVAudioFormat*)wrapper->format;
    return format.isInterleaved;
}

// Compare two formats for equality
const char* audioformat_is_equal(AudioFormat* wrapper1, AudioFormat* wrapper2, bool* result) {
    if (!result) {
        return "Result pointer is null";
    }
    
    if (!wrapper1) {
        return "First format pointer is null";
    }
    
    if (!wrapper2) {
        return "Second format pointer is null";
    }
    
    if (!wrapper1->format) {
        return "First format object is null";
    }
    
    if (!wrapper2->format) {
        return "Second format object is null";
    }
    
    AVAudioFormat* format1 = (__bridge AVAudioFormat*)wrapper1->format;
    AVAudioFormat* format2 = (__bridge AVAudioFormat*)wrapper2->format;
    
    *result = [format1 isEqual:format2];
    return NULL;  // NULL = success
}

// Log format information for debugging
void audioformat_log_info(AudioFormat* wrapper) {
    if (!wrapper || !wrapper->format) {
        NSLog(@"AudioFormat: NULL");
        return;
    }
    
    AVAudioFormat* format = (__bridge AVAudioFormat*)wrapper->format;
    NSLog(@"AudioFormat: %.0f Hz, %d channels, %s, format: %@", 
          format.sampleRate, 
          (int)format.channelCount,
          format.isInterleaved ? "interleaved" : "non-interleaved",
          format.formatDescription);
}

// Destroy the format and free resources
void audioformat_destroy(AudioFormat* wrapper) {
    if (!wrapper) {
        return;
    }
    
    if (wrapper->format) {
        // Release the retained reference
        AVAudioFormat* format = (__bridge_transfer AVAudioFormat*)wrapper->format;
        format = nil;
        wrapper->format = NULL;
    }
    
    // Free the wrapper
    free(wrapper);
}
