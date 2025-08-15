#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AudioUnit.h>
#import <Foundation/Foundation.h>

// Result structures for functions that return pointers
typedef struct {
    void* result;           // The actual result pointer
    const char* error;      // NULL = success, string = error message  
} UnitResult;

// AVAudioUnitEffect creation from plugin info
UnitResult create_unit_effect(uint32_t type, uint32_t subtype, uint32_t manufacturer) {
    AudioComponentDescription desc = {
        .componentType = type,
        .componentSubType = subtype,
        .componentManufacturer = manufacturer,
        .componentFlags = 0,
        .componentFlagsMask = 0
    };

    @try {
        AVAudioUnitEffect* effect = [[AVAudioUnitEffect alloc] initWithAudioComponentDescription:desc];

        if (!effect) {
            return (UnitResult){NULL, "Failed to create AVAudioUnitEffect"};
        }

        NSLog(@"Created AVAudioUnitEffect: %@", effect);
        return (UnitResult){(__bridge_retained void*)effect, NULL};
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Exception creating effect: %@", exception.reason];
        return (UnitResult){NULL, [errorMsg UTF8String]};
    }
}

// Release effect
const char* release_unit_effect(void* effectPtr) {
    if (!effectPtr) {
        return "Effect pointer is null";
    }

    @try {
        CFBridgingRelease(effectPtr);
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Exception releasing effect: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

// Set parameter using address from plugins package
const char* set_effect_parameter(void* effectPtr, uint64_t address, float value) {
    if (!effectPtr) {
        return "Effect pointer is null";
    }

    AVAudioUnitEffect* effect = (__bridge AVAudioUnitEffect*)effectPtr;

    @try {
        AudioUnit audioUnit = effect.audioUnit;
        if (audioUnit == NULL) {
            return "AudioUnit is null";
        }

        // Use the address from plugins.Parameter.Address
        OSStatus status = AudioUnitSetParameter(audioUnit, (AudioUnitParameterID)address,
                                              kAudioUnitScope_Global, 0, value, 0);
        if (status != noErr) {
            NSString* errorMsg = [NSString stringWithFormat:@"AudioUnitSetParameter failed with status: %d", (int)status];
            return [errorMsg UTF8String];
        }
        return NULL; // Success
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Exception setting parameter: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

// Get parameter using address from plugins package
UnitResult get_effect_parameter(void* effectPtr, uint64_t address) {
    if (!effectPtr) {
        return (UnitResult){NULL, "Effect pointer is null"};
    }

    AVAudioUnitEffect* effect = (__bridge AVAudioUnitEffect*)effectPtr;

    @try {
        AudioUnit audioUnit = effect.audioUnit;
        if (audioUnit == NULL) {
            return (UnitResult){NULL, "AudioUnit is null"};
        }

        AudioUnitParameterValue value = 0.0f;
        OSStatus status = AudioUnitGetParameter(audioUnit, (AudioUnitParameterID)address,
                                              kAudioUnitScope_Global, 0, &value);
        if (status != noErr) {
            NSString* errorMsg = [NSString stringWithFormat:@"AudioUnitGetParameter failed with status: %d", (int)status];
            return (UnitResult){NULL, [errorMsg UTF8String]};
        }

        // Return the float value as a void pointer (we'll cast it back in Go)
        float* valuePtr = malloc(sizeof(float));
        *valuePtr = value;
        return (UnitResult){(void*)valuePtr, NULL};
    } @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Exception getting parameter: %@", exception.reason];
        return (UnitResult){NULL, [errorMsg UTF8String]};
    }
}

// Set bypass on AVAudioUnitEffect or AUAudioUnit
const char* set_effect_bypass(void* effectPtr, int bypass) {
    if (!effectPtr) {
        return "Effect pointer is null";
    }
    @try {
        AVAudioUnit* unit = (__bridge AVAudioUnit*)effectPtr;
        if ([unit isKindOfClass:[AVAudioUnitEffect class]]) {
            AVAudioUnitEffect* effect = (AVAudioUnitEffect*)unit;
            effect.bypass = (BOOL)(bypass != 0);
            return NULL;
        }
        if (unit.AUAudioUnit) {
            unit.AUAudioUnit.shouldBypassEffect = (BOOL)(bypass != 0);
            return NULL;
        }
        return "Unit is not an AVAudioUnitEffect and has no AUAudioUnit";
    } @catch (NSException* ex) {
        NSString* msg = [NSString stringWithFormat:@"Failed to set bypass: %@", ex.reason];
        return [msg UTF8String];
    }
}

// Get bypass state
const char* get_effect_bypass(void* effectPtr, int* result) {
    if (!result) { return "Result pointer is null"; }
    if (!effectPtr) { return "Effect pointer is null"; }
    @try {
        AVAudioUnit* unit = (__bridge AVAudioUnit*)effectPtr;
        if ([unit isKindOfClass:[AVAudioUnitEffect class]]) {
            AVAudioUnitEffect* effect = (AVAudioUnitEffect*)unit;
            *result = effect.bypass ? 1 : 0;
            return NULL;
        }
        if (unit.AUAudioUnit) {
            *result = unit.AUAudioUnit.shouldBypassEffect ? 1 : 0;
            return NULL;
        }
        return "Unit is not an AVAudioUnitEffect and has no AUAudioUnit";
    } @catch (NSException* ex) {
        NSString* msg = [NSString stringWithFormat:@"Failed to get bypass: %@", ex.reason];
        return [msg UTF8String];
    }
}

// Helper that returns bypass as int and optionally returns error string via out param
int get_effect_bypass_simple(void* effectPtr, const char** err) {
    int r = 0;
    const char* e = get_effect_bypass(effectPtr, &r);
    if (err) { *err = e; }
    return r;
}
