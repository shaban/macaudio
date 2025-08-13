#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>

// Result structures for functions that return pointers
typedef struct {
    void* result;           // The actual result pointer
    const char* error;      // NULL = success, string = error message  
} AudioNodeResult;

// Generic bus operations that work on ANY AVAudioNode*

AudioNodeResult audionode_input_format_for_bus(void* nodePtr, int bus) {
    if (!nodePtr) {
        return (AudioNodeResult){NULL, "Node pointer is null"};
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;

    if (bus < 0) {
        return (AudioNodeResult){NULL, "Input bus number cannot be negative"};
    }
    
    if (bus >= node.numberOfInputs) {
        NSString* errorMsg = [NSString stringWithFormat:@"Invalid input bus %d (node has %d inputs)", bus, (int)node.numberOfInputs];
        return (AudioNodeResult){NULL, [errorMsg UTF8String]};
    }

    AVAudioFormat* format = [node inputFormatForBus:bus];
    if (!format) {
        NSString* errorMsg = [NSString stringWithFormat:@"No format available for input bus %d", bus];
        return (AudioNodeResult){NULL, [errorMsg UTF8String]};
    }

    NSLog(@"Got input format for bus %d: %.0f Hz, %d channels", bus, format.sampleRate, (int)format.channelCount);
    return (AudioNodeResult){(__bridge void*)format, NULL};
}

AudioNodeResult audionode_output_format_for_bus(void* nodePtr, int bus) {
    if (!nodePtr) {
        return (AudioNodeResult){NULL, "Node pointer is null"};
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;

    if (bus < 0) {
        return (AudioNodeResult){NULL, "Output bus number cannot be negative"};
    }
    
    if (bus >= node.numberOfOutputs) {
        NSString* errorMsg = [NSString stringWithFormat:@"Invalid output bus %d (node has %d outputs)", bus, (int)node.numberOfOutputs];
        return (AudioNodeResult){NULL, [errorMsg UTF8String]};
    }

    AVAudioFormat* format = [node outputFormatForBus:bus];
    if (!format) {
        NSString* errorMsg = [NSString stringWithFormat:@"No format available for output bus %d", bus];
        return (AudioNodeResult){NULL, [errorMsg UTF8String]};
    }

    NSLog(@"Got output format for bus %d: %.0f Hz, %d channels", bus, format.sampleRate, (int)format.channelCount);
    return (AudioNodeResult){(__bridge void*)format, NULL};
}

const char* audionode_get_number_of_inputs(void* nodePtr, int* result) {
    if (!result) {
        return "Result pointer is null";
    }
    
    if (!nodePtr) {
        return "Node pointer is null";
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    *result = (int)node.numberOfInputs;
    NSLog(@"Node has %d inputs", *result);
    return NULL; // Success
}

const char* audionode_get_number_of_outputs(void* nodePtr, int* result) {
    if (!result) {
        return "Result pointer is null";
    }
    
    if (!nodePtr) {
        return "Node pointer is null";
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    *result = (int)node.numberOfOutputs;
    NSLog(@"Node has %d outputs", *result);
    return NULL; // Success
}

const char* audionode_is_installed_on_engine(void* nodePtr, bool* result) {
    if (!result) {
        return "Result pointer is null";
    }
    
    if (!nodePtr) {
        return "Node pointer is null";
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    *result = node.engine != nil;
    NSLog(@"Node installed on engine: %s", *result ? "YES" : "NO");
    return NULL; // Success
}

const char* audionode_log_info(void* nodePtr) {
    if (!nodePtr) {
        return "Node pointer is null";
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    NSLog(@"AudioNode Info:");
    NSLog(@"  Class: %@", [node class]);
    NSLog(@"  Inputs: %d", (int)node.numberOfInputs);
    NSLog(@"  Outputs: %d", (int)node.numberOfOutputs);
    NSLog(@"  Engine: %@", node.engine ? @"Connected" : @"Not connected");
    NSLog(@"  Description: %@", node);
    return NULL; // Success
}

// AVAudioMixerNode specific functions

AudioNodeResult audiomixer_create(void) {
    @try {
        AVAudioMixerNode* mixer = [[AVAudioMixerNode alloc] init];
        if (!mixer) {
            return (AudioNodeResult){NULL, "Failed to allocate AVAudioMixerNode"};
        }
        NSLog(@"Created AVAudioMixerNode: %@", mixer);
        return (AudioNodeResult){(__bridge_retained void*)mixer, NULL};
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to create mixer: %@", exception.reason];
        return (AudioNodeResult){NULL, [errorMsg UTF8String]};
    }
}

const char* audiomixer_set_volume(void* mixerPtr, float volume, int inputBus) {
    if (!mixerPtr) {
        return "Mixer pointer is null";
    }

    if (volume < 0.0f || volume > 1.0f) {
        return "Volume must be between 0.0 and 1.0";
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0) {
        return "Input bus number cannot be negative";
    }
    
    if (inputBus >= mixer.numberOfInputs) {
        NSString* errorMsg = [NSString stringWithFormat:@"Invalid input bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs];
        return [errorMsg UTF8String];
    }

    @try {
        mixer.volume = volume;
        NSLog(@"Set mixer volume to %.2f on bus %d", volume, inputBus);
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to set volume: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

const char* audiomixer_set_pan(void* mixerPtr, float pan, int inputBus) {
    if (!mixerPtr) {
        return "Mixer pointer is null";
    }

    if (pan < -1.0f || pan > 1.0f) {
        return "Pan must be between -1.0 (left) and 1.0 (right)";
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0) {
        return "Input bus number cannot be negative";
    }
    
    if (inputBus >= mixer.numberOfInputs) {
        NSString* errorMsg = [NSString stringWithFormat:@"Invalid input bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs];
        return [errorMsg UTF8String];
    }

    @try {
        mixer.pan = pan;
        NSLog(@"Set mixer pan to %.2f (-1.0=left, 0.0=center, 1.0=right)", pan);
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to set pan: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

const char* audiomixer_get_volume(void* mixerPtr, int inputBus, float* result) {
    if (!result) {
        return "Result pointer is null";
    }
    
    if (!mixerPtr) {
        return "Mixer pointer is null";
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0) {
        return "Input bus number cannot be negative";
    }
    
    if (inputBus >= mixer.numberOfInputs) {
        NSString* errorMsg = [NSString stringWithFormat:@"Invalid input bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs];
        return [errorMsg UTF8String];
    }

    @try {
        *result = mixer.volume;
        NSLog(@"Got mixer volume %.2f from bus %d", *result, inputBus);
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to get volume: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

const char* audiomixer_get_pan(void* mixerPtr, int inputBus, float* result) {
    if (!result) {
        return "Result pointer is null";
    }
    
    if (!mixerPtr) {
        return "Mixer pointer is null";
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0) {
        return "Input bus number cannot be negative";
    }
    
    if (inputBus >= mixer.numberOfInputs) {
        NSString* errorMsg = [NSString stringWithFormat:@"Invalid input bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs];
        return [errorMsg UTF8String];
    }

    @try {
        *result = mixer.pan;
        NSLog(@"Got mixer pan %.2f from bus %d", *result, inputBus);
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to get pan: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

const char* audiomixer_release(void* mixerPtr) {
    if (!mixerPtr) {
        return "Mixer pointer is null";
    }

    @try {
        CFBridgingRelease(mixerPtr);
        NSLog(@"Released AVAudioMixerNode");
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Failed to release mixer: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}
