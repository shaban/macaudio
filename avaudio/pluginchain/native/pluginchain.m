#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AudioUnit.h>
#import <Foundation/Foundation.h>

// Result structures for functions that return pointers
typedef struct {
    void* result;           // The actual result pointer
    const char* error;      // NULL = success, string = error message  
} PluginChainResult;

// Connect effects in series using AVAudioEngine
const char* connect_effects(void* enginePtr, void** effectPtrs, int effectCount) {
    if (!enginePtr) {
        return "Engine pointer is null";
    }
    if (!effectPtrs) {
        return "Effect pointers array is null";
    }
    if (effectCount <= 0) {
        return "Effect count must be positive";
    }

    AVAudioEngine* engine = (__bridge AVAudioEngine*)enginePtr;

    @try {
        // First, disconnect all effects that are currently attached to prevent conflicts
        for (int i = 0; i < effectCount; i++) {
            AVAudioNode* effect = (__bridge AVAudioNode*)effectPtrs[i];
            if (!effect) {
                NSString* errorMsg = [NSString stringWithFormat:@"Invalid effect at index %d", i];
                return [errorMsg UTF8String];
            }

            // Disconnect the node if it's already connected
            if ([engine.attachedNodes containsObject:effect]) {
                [engine disconnectNodeInput:effect bus:0];
                [engine disconnectNodeOutput:effect bus:0];
            }
        }

        // Then, attach all effects to the engine if not already attached
        for (int i = 0; i < effectCount; i++) {
            AVAudioNode* effect = (__bridge AVAudioNode*)effectPtrs[i];

            // Check if already attached to avoid duplicate attachment
            if (![engine.attachedNodes containsObject:effect]) {
                [engine attachNode:effect];
            }
        }

        // Finally, connect effects in series: effect[0] -> effect[1] -> ... -> effect[n-1]
        for (int i = 0; i < effectCount - 1; i++) {
            AVAudioNode* sourceEffect = (__bridge AVAudioNode*)effectPtrs[i];
            AVAudioNode* destinationEffect = (__bridge AVAudioNode*)effectPtrs[i + 1];

            if (!sourceEffect || !destinationEffect) {
                NSString* errorMsg = [NSString stringWithFormat:@"Invalid effect at index %d or %d", i, i + 1];
                return [errorMsg UTF8String];
            }

            // Connect source to destination
            [engine connect:sourceEffect to:destinationEffect format:nil];
        }

        return NULL; // Success
    }
    @catch (NSException* exception) {
        NSString* errorMsg = [NSString stringWithFormat:@"Exception connecting effects: %@", exception.reason];
        return [errorMsg UTF8String];
    }
}

// Get the effect's audio node for external routing
PluginChainResult get_effect_audio_node(void* effectPtr) {
    if (!effectPtr) {
        return (PluginChainResult){NULL, "Effect pointer is null"};
    }

    // Effects are already AVAudioNode instances, so we just return the pointer
    return (PluginChainResult){effectPtr, NULL};
}
