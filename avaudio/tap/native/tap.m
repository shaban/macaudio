#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AudioUnit.h>
#import <Foundation/Foundation.h>

// Result structures for functions that return pointers/data
typedef struct {
    void* result;           // The actual result pointer
    const char* error;      // NULL = success, string = error message  
} TapResult;

// Tap callback info structure
typedef struct {
    void* tapPtr;      // Unique tap identifier
    void* nodePtr;     // AVAudioNode being tapped
    int busIndex;      // Bus index being tapped
    bool isActive;     // Whether tap is currently active
    double sampleRate; // Sample rate of the tapped audio
    int channelCount;  // Number of channels being tapped
} TapInfo;

// Global tap storage (simplified for this implementation)
static NSMutableDictionary* activeTaps = nil;

// Initialize tap storage
void tap_init() {
    if (!activeTaps) {
        activeTaps = [[NSMutableDictionary alloc] init];
    }
}

// Install a tap on an AVAudioNode at the specified bus
const char* tap_install(void* enginePtr, void* nodePtr, int busIndex, const char* tapKey) {
    if (!enginePtr) {
        return "Engine pointer is null";
    }
    if (!nodePtr) {
        return "Node pointer is null";
    }
    if (!tapKey) {
        return "Tap key is null";
    }

    tap_init();

    AVAudioEngine* engine = (__bridge AVAudioEngine*)enginePtr;
    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    NSString* tapKeyString = [NSString stringWithUTF8String:tapKey];

    // Check for key collision (strict enforcement)
    @synchronized(activeTaps) {
        if (activeTaps[tapKeyString]) {
            NSString* errorMsg = [NSString stringWithFormat:@"🚨 TAP KEY COLLISION: '%s' already exists - remove existing tap first", tapKey];
            return [errorMsg UTF8String];
        }
    }

    @try {
        // Check if node is attached to engine
        if (![engine.attachedNodes containsObject:node]) {
            return "Node is not attached to engine";
        }

        // Check bus validity
        if (busIndex < 0 || busIndex >= node.numberOfOutputs) {
            NSString* errorMsg = [NSString stringWithFormat:@"Invalid bus index %d for node with %d outputs",
                                  busIndex, (int)node.numberOfOutputs];
            return [errorMsg UTF8String];
        }

        // Get the format for this bus
        AVAudioFormat* format = [node outputFormatForBus:busIndex];
        if (!format) {
            NSString* errorMsg = [NSString stringWithFormat:@"No format available for bus %d", busIndex];
            return [errorMsg UTF8String];
        }

        // Remove existing tap if present on this bus (safety)
        [node removeTapOnBus:busIndex];

        // Install the tap with a callback that stores audio data
        [node installTapOnBus:busIndex bufferSize:1024 format:format block:^(AVAudioPCMBuffer * _Nonnull buffer, AVAudioTime * _Nonnull when) {
            // Store tap information for retrieval
            @synchronized(activeTaps) {
                NSMutableDictionary* tapData = activeTaps[tapKeyString];
                if (!tapData) {
                    tapData = [[NSMutableDictionary alloc] init];
                    activeTaps[tapKeyString] = tapData;
                }

                // Store latest buffer info (we'll keep this simple for now)
                tapData[@"frameLength"] = @(buffer.frameLength);
                tapData[@"frameCapacity"] = @(buffer.frameCapacity);
                tapData[@"sampleRate"] = @(format.sampleRate);
                tapData[@"channelCount"] = @(format.channelCount);
                tapData[@"lastUpdateTime"] = @([[NSDate date] timeIntervalSince1970]);

                // Calculate RMS for monitoring (simple implementation)
                if (buffer.frameLength > 0 && buffer.floatChannelData) {
                    float rms = 0.0f;
                    float* channelData = buffer.floatChannelData[0]; // Use first channel
                    for (UInt32 i = 0; i < buffer.frameLength; i++) {
                        rms += channelData[i] * channelData[i];
                    }
                    rms = sqrt(rms / buffer.frameLength);
                    tapData[@"rms"] = @(rms);
                }
            }
        }];

        // Store tap info
        @synchronized(activeTaps) {
            NSMutableDictionary* tapData = activeTaps[tapKeyString];
            if (!tapData) {
                tapData = [[NSMutableDictionary alloc] init];
                activeTaps[tapKeyString] = tapData;
            }

            tapData[@"nodePtr"] = [NSValue valueWithPointer:nodePtr];
            tapData[@"enginePtr"] = [NSValue valueWithPointer:enginePtr];
            tapData[@"busIndex"] = @(busIndex);
            tapData[@"isActive"] = @YES;
            tapData[@"sampleRate"] = @(format.sampleRate);
            tapData[@"channelCount"] = @(format.channelCount);
        }

        NSLog(@"tap_install: Successfully installed tap '%s' on bus %d (%.0f Hz, %d channels)",
              tapKey, busIndex, format.sampleRate, (int)format.channelCount);
        return NULL; // Success

    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Exception installing tap: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

// Remove a tap by key
const char* tap_remove(const char* tapKey) {
    if (!tapKey) {
        return "Tap key is null";
    }

    tap_init();

    NSString* tapKeyString = [NSString stringWithUTF8String:tapKey];

    @try {
        @synchronized(activeTaps) {
            NSMutableDictionary* tapData = activeTaps[tapKeyString];
            if (!tapData) {
                return "Tap not found";
            }

            // Get the node and bus from stored data
            void* nodePtr = [[tapData objectForKey:@"nodePtr"] pointerValue];
            int busIndex = [[tapData objectForKey:@"busIndex"] intValue];

            if (nodePtr) {
                AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
                [node removeTapOnBus:busIndex];
            }

            // Remove from our storage
            [activeTaps removeObjectForKey:tapKeyString];
        }

        NSLog(@"tap_remove: Successfully removed tap '%s'", tapKey);
        return NULL; // Success

    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Exception removing tap: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

// Get tap information and metrics
const char* tap_get_info(const char* tapKey, TapInfo* info) {
    if (!tapKey) {
        return "Tap key is null";
    }
    if (!info) {
        return "Info pointer is null";
    }

    tap_init();

    NSString* tapKeyString = [NSString stringWithUTF8String:tapKey];

    @synchronized(activeTaps) {
        NSMutableDictionary* tapData = activeTaps[tapKeyString];
        if (!tapData) {
            return "Tap not found";
        }

        info->tapPtr = (void*)[tapKeyString hash]; // Use string hash as identifier
        info->nodePtr = [[tapData objectForKey:@"nodePtr"] pointerValue];
        info->busIndex = [[tapData objectForKey:@"busIndex"] intValue];
        info->isActive = [[tapData objectForKey:@"isActive"] boolValue];
        info->sampleRate = [[tapData objectForKey:@"sampleRate"] doubleValue];
        info->channelCount = [[tapData objectForKey:@"channelCount"] intValue];

        return NULL; // Success
    }
}

// Get current RMS level from tap
const char* tap_get_rms(const char* tapKey, double* result) {
    if (!tapKey) {
        return "Tap key is null";
    }
    if (!result) {
        return "Result pointer is null";
    }

    tap_init();

    NSString* tapKeyString = [NSString stringWithUTF8String:tapKey];

    @synchronized(activeTaps) {
        NSMutableDictionary* tapData = activeTaps[tapKeyString];
        if (!tapData) {
            return "Tap not found";
        }

        NSNumber* rms = tapData[@"rms"];
        if (rms) {
            *result = [rms doubleValue];
        } else {
            *result = 0.0;
        }
    }

    return NULL; // Success
}

// Get frame count from last buffer
const char* tap_get_frame_count(const char* tapKey, int* result) {
    if (!tapKey) {
        return "Tap key is null";
    }
    if (!result) {
        return "Result pointer is null";
    }

    tap_init();

    NSString* tapKeyString = [NSString stringWithUTF8String:tapKey];

    @synchronized(activeTaps) {
        NSMutableDictionary* tapData = activeTaps[tapKeyString];
        if (!tapData) {
            return "Tap not found";
        }

        NSNumber* frameLength = tapData[@"frameLength"];
        if (frameLength) {
            *result = [frameLength intValue];
        } else {
            *result = 0;
        }
    }

    return NULL; // Success
}

// Remove all taps (cleanup)
const char* tap_remove_all(void) {
    tap_init();

    @synchronized(activeTaps) {
        // We can't easily remove all taps without keeping engine reference
        // So we'll just clear our storage
        [activeTaps removeAllObjects];
        NSLog(@"tap_remove_all: Cleared tap storage");
    }

    return NULL; // Success
}

// Get number of active taps
const char* tap_get_active_count(int* result) {
    if (!result) {
        return "Result pointer is null";
    }

    tap_init();

    @synchronized(activeTaps) {
        *result = (int)[activeTaps count];
    }

    return NULL; // Success
}
