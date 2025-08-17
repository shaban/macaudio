#import <AVFoundation/AVFoundation.h>

// Result structure for operations that return pointers
typedef struct {
    void* result;
    const char* error;  // NULL for success, error message for failure
} AudioEngineResult;

typedef struct {
    void* engine;  // AVAudioEngine*
} AudioEngine;


// Create AVAudioFormat from audio specifications
AudioEngineResult audioengine_create_format(double sampleRate, int channelCount, int bitDepth);
void audioengine_release_format(void* formatPtr);

// Create new AVAudioEngine
AudioEngineResult audioengine_new() {
    @autoreleasepool {
        AVAudioEngine* engine = [[AVAudioEngine alloc] init];
        if (!engine) {
            return (AudioEngineResult){NULL, "Audio engine creation failed"};
        }

        AudioEngine* wrapper = malloc(sizeof(AudioEngine));
        if (!wrapper) {
            return (AudioEngineResult){NULL, "Memory allocation failed"};
        }

        wrapper->engine = (__bridge_retained void*)engine;
        return (AudioEngineResult){wrapper, NULL};  // NULL = success
    }
}

// Prepare the engine for starting
void audioengine_prepare(AudioEngine* wrapper) {
    if (!wrapper || !wrapper->engine) {
        return;
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    [engine prepare];
}

// Start the engine
const char* audioengine_start(AudioEngine* wrapper) {
    @autoreleasepool {
        if (!wrapper) {
            return "Engine wrapper is null";
        }
        
        if (!wrapper->engine) {
            return "Engine is invalid";
        }

        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        NSError* error = nil;

        @try {
            bool success = [engine startAndReturnError:&error];
            if (error) {
                NSLog(@"Engine start error: %@", error.localizedDescription);
                return "Engine start failed";
            }
            return success ? NULL : "Engine start failed";  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Engine start exception: %@", exception.reason);
            return "Engine start failed with exception";
        }
    }
}

// Stop the engine
void audioengine_stop(AudioEngine* wrapper) {
    if (!wrapper || !wrapper->engine) {
        return;
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    [engine stop];
}

// Pause the engine
void audioengine_pause(AudioEngine* wrapper) {
    if (!wrapper || !wrapper->engine) {
        return;
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    [engine pause];
}

// Reset the engine
void audioengine_reset(AudioEngine* wrapper) {
    if (!wrapper || !wrapper->engine) {
        return;
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    [engine reset];
}

// Check if engine is running
const char* audioengine_is_running(AudioEngine* wrapper) {
    if (!wrapper) {
        return "Engine wrapper is null";
    }
    
    if (!wrapper->engine) {
        return "Engine is invalid";
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    bool isRunning = engine.isRunning;
    return isRunning ? NULL : "Engine is not running";  // NULL = running (success)
}

// Remove all taps
void audioengine_remove_taps(AudioEngine* wrapper) {
    if (!wrapper || !wrapper->engine) {
        return;
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    // Note: AVAudioEngine doesn't have a direct "remove all taps" method
    // This would need to be tracked at the Go level or implemented differently
    // For now, this is a placeholder that can be called safely
}

// Get output node
AudioEngineResult audioengine_output_node(AudioEngine* wrapper) {
    if (!wrapper) {
        return (AudioEngineResult){NULL, "Engine wrapper is null"};
    }
    
    if (!wrapper->engine) {
        return (AudioEngineResult){NULL, "Engine is invalid"};
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    AVAudioOutputNode* outputNode = engine.outputNode;
    
    if (!outputNode) {
        return (AudioEngineResult){NULL, "Output node is invalid"};
    }
    
    return (AudioEngineResult){(__bridge void*)outputNode, NULL};  // NULL = success
}

// Get input node
AudioEngineResult audioengine_input_node(AudioEngine* wrapper) {
    if (!wrapper) {
        return (AudioEngineResult){NULL, "Engine wrapper is null"};
    }
    
    if (!wrapper->engine) {
        return (AudioEngineResult){NULL, "Engine is invalid"};
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    AVAudioInputNode* inputNode = engine.inputNode;
    
    if (!inputNode) {
        return (AudioEngineResult){NULL, "Input node is invalid"};
    }
    
    return (AudioEngineResult){(__bridge void*)inputNode, NULL};  // NULL = success
}

// Get main mixer node
AudioEngineResult audioengine_main_mixer_node(AudioEngine* wrapper) {
    if (!wrapper) {
        return (AudioEngineResult){NULL, "Engine wrapper is null"};
    }
    
    if (!wrapper->engine) {
        return (AudioEngineResult){NULL, "Engine is invalid"};
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    AVAudioMixerNode* mainMixer = engine.mainMixerNode;
    
    if (!mainMixer) {
        return (AudioEngineResult){NULL, "Main mixer node is invalid"};
    }
    
    return (AudioEngineResult){(__bridge void*)mainMixer, NULL};  // NULL = success
}

// Create a new individual mixer node for channels
AudioEngineResult audioengine_create_mixer_node(AudioEngine* wrapper) {
    @autoreleasepool {
        if (!wrapper || !wrapper->engine) {
            return (AudioEngineResult){NULL, "Invalid engine wrapper"};
        }

        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        
        // Create a new mixer node
        AVAudioMixerNode* newMixer = [[AVAudioMixerNode alloc] init];
        if (!newMixer) {
            return (AudioEngineResult){NULL, "Failed to create mixer node"};
        }
        
        // Attach the mixer to the engine
        [engine attachNode:newMixer];
        
        return (AudioEngineResult){(__bridge_retained void*)newMixer, NULL};
    }
}

// Destroy the engine and free resources
void audioengine_destroy(AudioEngine* wrapper) {
    if (!wrapper) {
        return;
    }

    if (wrapper->engine) {
        AVAudioEngine* engine = (__bridge_transfer AVAudioEngine*)wrapper->engine;

        // Stop if running
        if (engine.isRunning) {
            [engine stop];
        }

        // Enhanced: Remove all taps from all nodes to prevent AVFoundation state corruption
        @try {
            // Remove taps from main mixer node (most common tap location)
            AVAudioMixerNode* mainMixer = engine.mainMixerNode;
            if (mainMixer) {
                for (AVAudioNodeBus bus = 0; bus < mainMixer.numberOfInputs; bus++) {
                    [mainMixer removeTapOnBus:bus];
                }
                for (AVAudioNodeBus bus = 0; bus < mainMixer.numberOfOutputs; bus++) {
                    [mainMixer removeTapOnBus:bus];
                }
            }
            
            // Remove taps from output node
            AVAudioOutputNode* outputNode = engine.outputNode;
            if (outputNode) {
                for (AVAudioNodeBus bus = 0; bus < outputNode.numberOfInputs; bus++) {
                    [outputNode removeTapOnBus:bus];
                }
            }
            
            // Remove taps from input node (if available)
            AVAudioInputNode* inputNode = engine.inputNode;
            if (inputNode) {
                for (AVAudioNodeBus bus = 0; bus < inputNode.numberOfOutputs; bus++) {
                    [inputNode removeTapOnBus:bus];
                }
            }
            
            NSLog(@"Enhanced cleanup: Removed all taps from engine nodes");
        }
        @catch (NSException* exception) {
            // Tap removal can fail if no taps are installed, which is expected
            // Log but don't fail the destroy operation
            NSLog(@"Tap removal during destroy (expected): %@", exception.reason);
        }

        // Give AVFoundation time to clean up tap resources
        usleep(5000); // 5ms - small delay to ensure cleanup completes

        // Reset the engine (this disconnects all nodes)
        [engine reset];

        // Clear the reference
        engine = nil;
        wrapper->engine = NULL;
    }

    // Free the wrapper
    free(wrapper);
}

// Attach node to engine
const char* audioengine_attach(AudioEngine* wrapper, void* nodePtr) {
    if (!wrapper) {
        return "Engine wrapper is null";
    }
    
    if (!wrapper->engine) {
        return "Engine is invalid";
    }
    
    if (!nodePtr) {
        return "Node pointer is null";
    }

    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
        if (!node) { return "Node is invalid"; }

        __block const char* err = NULL;
        void (^work)(void) = ^{
            @try {
                // Quick sanity log to help trace crashes
                // NSLog(@"Attaching node class=%@ ptr=%p", NSStringFromClass([node class]), node);
                if (node.engine == engine) {
                    // Already attached to this engine
                    return;
                }
                [engine attachNode:node];
            } @catch (NSException* ex) {
                // Treat "already attached" as success if that occurs
                NSString* reason = ex.reason ?: @"";
                if ([reason containsString:@"already"] && [reason containsString:@"attached"]) {
                    err = NULL;
                } else {
                    err = [[NSString stringWithFormat:@"Attach exception: %@", reason] UTF8String];
                }
            }
        };

    // Execute directly on the current thread to avoid deadlocks when running under Go tests
    // where the libdispatch main queue may not be actively serviced.
    work();

        return err; // NULL on success, error string on failure
    }
    @catch (NSException* exception) {
        NSLog(@"Engine attach exception: %@", exception.reason);
        return "Failed to attach node";
    }
}

// Detach node from engine
const char* audioengine_detach(AudioEngine* wrapper, void* nodePtr) {
    if (!wrapper) {
        return "Engine wrapper is null";
    }
    
    if (!wrapper->engine) {
        return "Engine is invalid";
    }
    
    if (!nodePtr) {
        return "Node pointer is null";
    }

    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
        __block const char* err = NULL;
        void (^work)(void) = ^{
            @try {
                [engine detachNode:node];
            } @catch (NSException* ex) {
                err = [[NSString stringWithFormat:@"Detach exception: %@", ex.reason] UTF8String];
            }
        };
    // Execute directly on the current thread (see note in audioengine_attach)
    work();
        return err;  // NULL on success
    }
    @catch (NSException* exception) {
        NSLog(@"Engine detach exception: %@", exception.reason);
        return "Failed to detach node";
    }
}

// Connect two nodes
const char* audioengine_connect(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus) {
    if (!wrapper) {
        return "Engine wrapper is null";
    }
    
    if (!wrapper->engine) {
        return "Engine is invalid";
    }
    
    if (!sourcePtr || !destPtr) {
        return "Node pointers cannot be null";
    }

    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        AVAudioNode* sourceNode = (__bridge AVAudioNode*)sourcePtr;
        AVAudioNode* destNode = (__bridge AVAudioNode*)destPtr;
        __block const char* err = NULL;
        void (^work)(void) = ^{
            @try {
                [engine connect:sourceNode to:destNode fromBus:fromBus toBus:toBus format:nil];
            } @catch (NSException* ex) {
                err = [[NSString stringWithFormat:@"Connect exception: %@", ex.reason] UTF8String];
            }
        };
    // Execute directly on the current thread (see note in audioengine_attach)
    work();
        return err;  // NULL on success
    }
    @catch (NSException* exception) {
        NSLog(@"Engine connect exception: %@", exception.reason);
        return "Failed to connect nodes";
    }
}

// Connect two nodes with explicit format
const char* audioengine_connect_with_format(AudioEngine* wrapper, void* sourcePtr, void* destPtr, int fromBus, int toBus, void* formatPtr) {
    if (!wrapper) {
        return "Engine wrapper is null";
    }
    
    if (!wrapper->engine) {
        return "Engine is invalid";
    }
    
    if (!sourcePtr || !destPtr) {
        return "Node pointers cannot be null";
    }

    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        AVAudioNode* sourceNode = (__bridge AVAudioNode*)sourcePtr;
        AVAudioNode* destNode = (__bridge AVAudioNode*)destPtr;
        AVAudioFormat* format = formatPtr ? (__bridge AVAudioFormat*)formatPtr : nil;
        __block const char* err = NULL;
        void (^work)(void) = ^{
            @try {
                if (format) {
                    NSLog(@"Connecting with explicit format: %.0f Hz, %d channels", format.sampleRate, format.channelCount);
                } else {
                    NSLog(@"Connecting with nil format (will use source node's output format)");
                }
                [engine connect:sourceNode to:destNode fromBus:fromBus toBus:toBus format:format];
            } @catch (NSException* ex) {
                err = [[NSString stringWithFormat:@"Connect-with-format exception: %@", ex.reason] UTF8String];
            }
        };
    // Execute directly on the current thread (see note in audioengine_attach)
    work();
        return err;  // NULL on success
    }
    @catch (NSException* exception) {
        NSLog(@"Engine connect with format exception: %@", exception.reason);
        return "Failed to connect nodes with format";
    }
}

// Set pan on the main mixer node (-1.0 = hard left, 0.0 = center, 1.0 = hard right)
void audioengine_set_mixer_pan(AudioEngine* wrapper, float pan) {
    if (!wrapper || !wrapper->engine) {
        return;
    }

    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        AVAudioMixerNode* mixerNode = [engine mainMixerNode];

        if (mixerNode) {
            mixerNode.pan = pan;
            NSLog(@"Set mixer pan to %.2f (-1.0=left, 0.0=center, 1.0=right)", pan);
        } else {
            NSLog(@"Warning: Main mixer node is nil, cannot set pan");
        }
    }
    @catch (NSException* exception) {
        NSLog(@"Engine set mixer pan exception: %@", exception.reason);
    }
}

// Disconnect a node's input bus
const char* audioengine_disconnect_node_input(AudioEngine* wrapper, void* nodePtr, int inputBus) {
    if (!wrapper) {
        NSLog(@"audioengine_disconnect_node_input: wrapper is null");
        return "Engine wrapper is null";
    }
    
    if (!wrapper->engine) {
        NSLog(@"audioengine_disconnect_node_input: engine is null");
        return "Engine is invalid";
    }
    
    if (!nodePtr) {
        NSLog(@"audioengine_disconnect_node_input: nodePtr is null");
        return "Node pointer is null";
    }

    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
        AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;

        // Validate input bus
        if (inputBus < 0) {
            NSLog(@"audioengine_disconnect_node_input: invalid bus %d (must be >= 0)", inputBus);
            return "Invalid input bus (must be >= 0)";
        }
        
        if (inputBus >= node.numberOfInputs) {
            NSLog(@"audioengine_disconnect_node_input: invalid bus %d (node has %d inputs)", inputBus, (int)node.numberOfInputs);
            return "Invalid input bus (exceeds node's input count)";
        }

        __block const char* err = NULL;
        void (^work)(void) = ^{
            @try {
                [engine disconnectNodeInput:node bus:inputBus];
                NSLog(@"Successfully disconnected input bus %d of node %@", inputBus, node);
            } @catch (NSException* ex) {
                err = [[NSString stringWithFormat:@"Disconnect exception: %@", ex.reason] UTF8String];
            }
        };
    // Execute directly on the current thread (see note in audioengine_attach)
    work();
        return err;  // NULL on success
    }
    @catch (NSException* exception) {
        NSLog(@"Engine disconnect node input exception: %@", exception.reason);
        return "Failed to disconnect node input";
    }
}

// Create AVAudioFormat from audio specifications
AudioEngineResult audioengine_create_format(double sampleRate, int channelCount, int bitDepth) {
    @autoreleasepool {
        @try {
            // Use the standard stereo format and let AVFoundation handle the details
            // The exact bit depth handling is complex with AVFoundation's format system
            AVAudioFormat* format = [[AVAudioFormat alloc]
                initStandardFormatWithSampleRate:sampleRate
                channels:(AVAudioChannelCount)channelCount];

            if (format) {
                NSLog(@"Created AVAudioFormat: %.0f Hz, %d channels (standard format)",
                      sampleRate, channelCount);
                return (AudioEngineResult){(__bridge_retained void*)format, NULL};  // NULL = success
            } else {
                NSLog(@"Failed to create AVAudioFormat");
                return (AudioEngineResult){NULL, "Failed to create audio format"};
            }
        }
        @catch (NSException* exception) {
            NSLog(@"Exception creating AVAudioFormat: %@", exception.reason);
            return (AudioEngineResult){NULL, "Exception creating audio format"};
        }
    }
}

// Release AVAudioFormat
void audioengine_release_format(void* formatPtr) {
    @autoreleasepool {
        if (formatPtr) {
            AVAudioFormat* format = (__bridge_transfer AVAudioFormat*)formatPtr;
            format = nil; // ARC will handle deallocation
        }
    }
}

// Set buffer size for the engine
const char* audioengine_set_buffer_size(AudioEngine* wrapper, int bufferSize) {
    if (!wrapper) {
        NSLog(@"audioengine_set_buffer_size: wrapper is null");
        return "Engine wrapper is null";
    }
    
    if (!wrapper->engine) {
        NSLog(@"audioengine_set_buffer_size: engine is null");
        return "Engine is null";
    }
    
    if (bufferSize <= 0) {
        NSLog(@"audioengine_set_buffer_size: invalid buffer size %d", bufferSize);
        return "Buffer size must be positive";
    }
    
    AVAudioEngine* engine = (__bridge AVAudioEngine*)wrapper->engine;
    
    @try {
        // On macOS, we can try to set the buffer size through the output node
        // Note: AVAudioEngine doesn't guarantee exact buffer sizes
        // The actual buffer size is determined by the audio hardware and system
        
        AVAudioOutputNode* outputNode = engine.outputNode;
        if (!outputNode) {
            return "Output node not available";
        }
        
        // Get the current format to work with
        AVAudioFormat* format = [outputNode outputFormatForBus:0];
        if (!format) {
            return "No output format available";
        }
        
        // Log the attempt - this is the best we can do for now
        // The actual buffer size in Core Audio is managed by the system
        NSLog(@"Requested buffer size change to %d frames (%.2f ms at %.0f Hz)",
              bufferSize, 
              (double)bufferSize / format.sampleRate * 1000.0,
              format.sampleRate);
              
        // Note: In a real-world scenario, you might need to:
        // 1. Stop the engine
        // 2. Reconfigure audio units with new buffer preferences  
        // 3. Restart the engine
        // But this is complex and may not always work as expected
        
        return NULL;  // NULL = success (request acknowledged)
    }
    @catch (NSException* exception) {
        NSLog(@"Exception setting buffer size: %@", exception.reason);
        return "Failed to set buffer size";
    }
}

// Set volume of a specific mixer node
const char* audioengine_set_mixer_volume(AudioEngine* wrapper, void* mixerNodePtr, float volume) {
    @autoreleasepool {
        if (!wrapper) {
            return "Engine wrapper is null";
        }
        
        if (!wrapper->engine) {
            return "Engine is null";
        }
        
        if (!mixerNodePtr) {
            return "Mixer node pointer is null";
        }
        
        if (volume < 0.0f || volume > 1.0f) {
            return "Volume must be between 0.0 and 1.0";
        }
        
        @try {
            AVAudioMixerNode* mixerNode = (__bridge AVAudioMixerNode*)mixerNodePtr;
            
            // Set the output volume on the mixer node
            mixerNode.outputVolume = volume;
            
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception setting mixer volume: %@", exception.reason);
            return "Failed to set mixer volume";
        }
    }
}

// Get volume of a specific mixer node
float audioengine_get_mixer_volume(AudioEngine* wrapper, void* mixerNodePtr) {
    @autoreleasepool {
        if (!wrapper || !wrapper->engine || !mixerNodePtr) {
            return 0.0f;
        }
        
        @try {
            AVAudioMixerNode* mixerNode = (__bridge AVAudioMixerNode*)mixerNodePtr;
            return mixerNode.outputVolume;
        }
        @catch (NSException* exception) {
            NSLog(@"Exception getting mixer volume: %@", exception.reason);
            return 0.0f;
        }
    }
}