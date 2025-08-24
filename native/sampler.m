#import <AVFoundation/AVFoundation.h>
#import "macaudio.h"

// ==============================================
// Minimal Audio Sampler Implementation
// ==============================================

AudioSamplerResult audiosampler_create(void* enginePtr) {
    AudioSamplerResult result = {0};
    
    if (!enginePtr) {
        result.error = "Engine pointer is required";
        return result;
    }
    
    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)enginePtr;
        
        // Create AVAudioUnitSampler
        AVAudioUnitSampler* sampler = [[AVAudioUnitSampler alloc] init];
        if (!sampler) {
            result.error = "Failed to create AVAudioUnitSampler";
            return result;
        }
        
        // Attach to engine
        [engine attachNode:sampler];
        
        // Create wrapper
        AudioSampler* wrapper = malloc(sizeof(AudioSampler));
        if (!wrapper) {
            [engine detachNode:sampler];
            result.error = "Failed to allocate memory for sampler wrapper";
            return result;
        }
        
        wrapper->samplerNode = (__bridge_retained void*)sampler;
        wrapper->engine = enginePtr;
        wrapper->isConnected = false;
        
        result.result = wrapper;
        
    } @catch (NSException *exception) {
        result.error = [[NSString stringWithFormat:@"Exception creating sampler: %@", exception.reason] UTF8String];
    }
    
    return result;
}

const char* audiosampler_start_note(AudioSampler* sampler, int note, int velocity, int channel) {
    if (!sampler || !sampler->samplerNode) {
        return "Invalid sampler";
    }
    
    if (note < 0 || note > 127) {
        return "Note must be between 0 and 127";
    }
    if (velocity < 0 || velocity > 127) {
        return "Velocity must be between 0 and 127"; 
    }
    if (channel < 0 || channel > 15) {
        return "Channel must be between 0 and 15";
    }
    
    @try {
        AVAudioUnitSampler* samplerNode = (__bridge AVAudioUnitSampler*)sampler->samplerNode;
        [samplerNode startNote:(UInt8)note withVelocity:(UInt8)velocity onChannel:(UInt8)channel];
        
    } @catch (NSException *exception) {
        return [[NSString stringWithFormat:@"Exception starting note: %@", exception.reason] UTF8String];
    }
    
    return NULL; // Success
}

const char* audiosampler_stop_note(AudioSampler* sampler, int note, int channel) {
    if (!sampler || !sampler->samplerNode) {
        return "Invalid sampler";
    }
    
    if (note < 0 || note > 127) {
        return "Note must be between 0 and 127";
    }
    if (channel < 0 || channel > 15) {
        return "Channel must be between 0 and 15";
    }
    
    @try {
        AVAudioUnitSampler* samplerNode = (__bridge AVAudioUnitSampler*)sampler->samplerNode;
        [samplerNode stopNote:(UInt8)note onChannel:(UInt8)channel];
        
    } @catch (NSException *exception) {
        return [[NSString stringWithFormat:@"Exception stopping note: %@", exception.reason] UTF8String];
    }
    
    return NULL; // Success
}

const char* audiosampler_connect_to_mixer(AudioSampler* sampler, void* mixerPtr, int busIndex) {
    if (!sampler || !sampler->samplerNode || !sampler->engine || !mixerPtr) {
        return "Invalid parameters";
    }
    
    if (busIndex < 0) {
        return "Bus index must be non-negative";
    }
    
    @try {
        AVAudioEngine* engine = (__bridge AVAudioEngine*)sampler->engine;
        AVAudioUnitSampler* samplerNode = (__bridge AVAudioUnitSampler*)sampler->samplerNode;
        AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;
        
        // Connect sampler to mixer
        [engine connect:samplerNode to:mixer fromBus:0 toBus:busIndex format:nil];
        sampler->isConnected = true;
        
    } @catch (NSException *exception) {
        return [[NSString stringWithFormat:@"Exception connecting sampler: %@", exception.reason] UTF8String];
    }
    
    return NULL; // Success
}

void audiosampler_destroy(AudioSampler* sampler) {
    if (!sampler) {
        return;
    }
    
    @try {
        if (sampler->samplerNode && sampler->engine) {
            AVAudioEngine* engine = (__bridge AVAudioEngine*)sampler->engine;
            AVAudioUnitSampler* samplerNode = (__bridge AVAudioUnitSampler*)sampler->samplerNode;
            
            // Disconnect if connected
            if (sampler->isConnected) {
                [engine disconnectNodeOutput:samplerNode];
            }
            
            // Detach from engine
            [engine detachNode:samplerNode];
            
            // Release bridged reference
            CFBridgingRelease(sampler->samplerNode);
        }
        
        // Free wrapper
        free(sampler);
        
    } @catch (NSException *exception) {
        NSLog(@"Exception destroying sampler: %@", exception.reason);
        free(sampler); // Still free to prevent memory leak
    }
}
