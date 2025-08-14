#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>
#import <AudioToolbox/AudioToolbox.h>

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

// Per-connection (source -> mixer bus) controls using AVAudioMixingDestination.
// These allow controlling the gain/pan of a specific input bus as seen from the source.

static AVAudioMixingDestination* _getDestinationFor(void* sourcePtr, void* mixerPtr, int destBus, const char** err) {
    if (err) { *err = NULL; }
    if (!sourcePtr) { if (err) *err = "Source node pointer is null"; return nil; }
    if (!mixerPtr)  { if (err) *err = "Mixer pointer is null"; return nil; }
    if (destBus < 0) { if (err) *err = "Destination bus cannot be negative"; return nil; }

    AVAudioNode* sourceNode = (__bridge AVAudioNode*)sourcePtr;
    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (![sourceNode conformsToProtocol:@protocol(AVAudioMixing)]) {
        if (err) *err = "Source node does not support AVAudioMixing (no per-connection control)";
        return nil;
    }

    id<AVAudioMixing> mixingSource = (id<AVAudioMixing>)sourceNode;
    if (![mixingSource respondsToSelector:@selector(destinationForMixer:bus:)]) {
        if (err) *err = "AVAudioMixingDestination API not available";
        return nil;
    }

    AVAudioMixingDestination* dest = [mixingSource destinationForMixer:mixer bus:destBus];
    if (!dest) {
        NSString* msg = [NSString stringWithFormat:@"No destination for mixer %@ bus %d", mixer, destBus];
        if (err) *err = [msg UTF8String];
        return nil;
    }
    return dest;
}

const char* audiomixer_set_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float volume) {
    if (volume < 0.0f || volume > 1.0f) {
        return "Volume must be between 0.0 and 1.0";
    }
    const char* e = NULL;
    AVAudioMixingDestination* dest = _getDestinationFor(sourcePtr, mixerPtr, destBus, &e);
    if (!dest) { return e; }
    @try {
        dest.volume = volume;
        NSLog(@"Set per-connection volume %.2f on bus %d", volume, destBus);
        return NULL;
    } @catch (NSException* ex) {
        NSString* msg = [NSString stringWithFormat:@"Failed to set per-connection volume: %@", ex.reason];
        return [msg UTF8String];
    }
}

const char* audiomixer_get_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result) {
    if (!result) { return "Result pointer is null"; }
    const char* e = NULL;
    AVAudioMixingDestination* dest = _getDestinationFor(sourcePtr, mixerPtr, destBus, &e);
    if (!dest) { return e; }
    @try {
        *result = dest.volume;
        return NULL;
    } @catch (NSException* ex) {
        NSString* msg = [NSString stringWithFormat:@"Failed to get per-connection volume: %@", ex.reason];
        return [msg UTF8String];
    }
}

const char* audiomixer_set_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float pan) {
    if (pan < -1.0f || pan > 1.0f) {
        return "Pan must be between -1.0 and 1.0";
    }
    const char* e = NULL;
    AVAudioMixingDestination* dest = _getDestinationFor(sourcePtr, mixerPtr, destBus, &e);
    if (!dest) { return e; }
    @try {
        dest.pan = pan;
        NSLog(@"Set per-connection pan %.2f on bus %d", pan, destBus);
        return NULL;
    } @catch (NSException* ex) {
        NSString* msg = [NSString stringWithFormat:@"Failed to set per-connection pan: %@", ex.reason];
        return [msg UTF8String];
    }
}

const char* audiomixer_get_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result) {
    if (!result) { return "Result pointer is null"; }
    const char* e = NULL;
    AVAudioMixingDestination* dest = _getDestinationFor(sourcePtr, mixerPtr, destBus, &e);
    if (!dest) { return e; }
    @try {
        *result = dest.pan;
        return NULL;
    } @catch (NSException* ex) {
        NSString* msg = [NSString stringWithFormat:@"Failed to get per-connection pan: %@", ex.reason];
        return [msg UTF8String];
    }
}

// -------------------------
// Generic node helpers
// -------------------------
const char* audionode_release(void* nodePtr) {
    if (!nodePtr) { return NULL; }
    @try {
        // ARC will manage lifetime; clearing strong refs on Go side suffices.
        // We keep this for API symmetry.
        return NULL;
    } @catch (NSException* ex) {
        NSString* msg = [NSString stringWithFormat:@"Failed to release node: %@", ex.reason];
        return [msg UTF8String];
    }
}

// -------------------------
// Matrix Mixer (invert stage)
// -------------------------
static AVAudioUnit* _instantiateComponent(AudioComponentDescription desc, const char** err) {
    __block AVAudioUnit* unit = nil;
    __block NSString* errorMsg = nil;
    dispatch_semaphore_t sema = dispatch_semaphore_create(0);
    [AVAudioUnit instantiateWithComponentDescription:desc options:0 completionHandler:^(AVAudioUnit * _Nullable au, NSError * _Nullable error) {
        if (error) {
            errorMsg = [NSString stringWithFormat:@"Instantiate failed: %@", error.localizedDescription];
        } else {
            unit = au;
        }
        dispatch_semaphore_signal(sema);
    }];
    // Wait up to 2 seconds
    dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, (int64_t)(2 * NSEC_PER_SEC));
    if (dispatch_semaphore_wait(sema, timeout) != 0) {
        if (err) *err = "Timed out instantiating audio unit";
        return nil;
    }
    if (!unit && errorMsg && err) { *err = [errorMsg UTF8String]; }
    return unit;
}

AudioNodeResult matrixmixer_create(void) {
    AudioComponentDescription desc;
    desc.componentType = kAudioUnitType_Mixer;           // 'aumx'
    desc.componentSubType = kAudioUnitSubType_MatrixMixer; // 'mxmx'
    desc.componentManufacturer = kAudioUnitManufacturer_Apple; // 'appl'
    desc.componentFlags = 0;
    desc.componentFlagsMask = 0;
    const char* e = NULL;
    AVAudioUnit* unit = _instantiateComponent(desc, &e);
    if (!unit) {
        return (AudioNodeResult){NULL, e ? e : "Failed to create MatrixMixer"};
    }
    return (AudioNodeResult){(__bridge void*)unit, NULL};
}

// Configure the matrix mixer to invert polarity: diagonal gains set to -1.0
// The matrix size is inputChannels x outputChannels.
const char* matrixmixer_configure_invert(void* unitPtr) {
    if (!unitPtr) { return "Matrix mixer pointer is null"; }
    AVAudioUnit* unit = (__bridge AVAudioUnit*)unitPtr;
    @try {
        AVAudioFormat* inFmt = [unit AUAudioUnit] ? nil : nil; // placeholder to satisfy compiler
    } @catch (NSException* ex) {
        // continue
    }
    // Query formats directly from node
    AVAudioNode* node = (__bridge AVAudioNode*)unitPtr;
    AVAudioFormat* inFmt = [node inputFormatForBus:0];
    AVAudioFormat* outFmt = [node outputFormatForBus:0];
    if (!inFmt || !outFmt) { return "Matrix mixer formats unavailable"; }
    UInt32 inCh = (UInt32)inFmt.channelCount;
    UInt32 outCh = (UInt32)outFmt.channelCount;
    if (inCh == 0 || outCh == 0) { return "Zero channel count on matrix mixer"; }

    AudioUnit au = unit.audioUnit;
    if (au == NULL) { return "Underlying AudioUnit missing"; }

    // Build matrix with -1 on diagonal
    UInt32 count = inCh * outCh;
    Float32* gains = (Float32*)calloc(count, sizeof(Float32));
    if (!gains) { return "alloc failed"; }
    UInt32 min = (inCh < outCh) ? inCh : outCh;
    for (UInt32 i = 0; i < min; i++) {
        // Row-major: out channel major or in? Apple docs: levels are [outChan][inChan].
        // We'll fill gains[out*inCh + in] with -1 on diagonal.
        gains[i*inCh + i] = -1.0f;
    }
    OSStatus st = AudioUnitSetProperty(au, kAudioUnitProperty_MatrixLevels, kAudioUnitScope_Global, 0, gains, count * sizeof(Float32));
    free(gains);
    if (st != noErr) {
        NSString* msg = [NSString stringWithFormat:@"AudioUnitSetProperty(MATRIX_LEVELS) failed: %d", (int)st];
        return [msg UTF8String];
    }
    return NULL;
}
