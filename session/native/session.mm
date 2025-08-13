#import <Foundation/Foundation.h>
#import <AVFoundation/AVFoundation.h>

// Fix: Include header with correct path
#include "session.h"

// Forward declare the Go callback
extern void configurationChanged(void);

// Global state
static AVAudioEngine* monitored_engine = NULL;
static id config_observer = nil;

// Set up configuration change monitoring for an AVAudioEngine
void macaudio_setup_config_monitoring(void* engine_ptr) {
    monitored_engine = (__bridge AVAudioEngine*)engine_ptr;
    
    // Remove any existing observer
    if (config_observer != nil) {
        [[NSNotificationCenter defaultCenter] removeObserver:config_observer];
        config_observer = nil;
        NSLog(@"🔄 Removed existing configuration observer");
    }
    
    if (monitored_engine == NULL) {
        NSLog(@"⚠️ NULL engine pointer - setting up global monitoring for testing");
    } else {
        NSLog(@"✅ Setting up monitoring for specific AVAudioEngine");
    }
    
    // Set up new observer for configuration changes
    // Listen to ALL engines (object:nil) to support both real usage and testing
    config_observer = [[NSNotificationCenter defaultCenter] 
        addObserverForName:AVAudioEngineConfigurationChangeNotification
        object:nil  // Listen to all AVAudioEngine instances
        queue:[NSOperationQueue mainQueue]
        usingBlock:^(NSNotification *note) {
            NSLog(@"🔄 AVAudioEngine configuration changed (object: %@)", note.object);
            // Call back to Go
            configurationChanged();
        }];
    
    NSLog(@"✅ Set up global configuration monitoring for all AVAudioEngines");
}



// Clean up monitoring
void macaudio_cleanup_config_monitoring(void) {
    if (config_observer != nil) {
        [[NSNotificationCenter defaultCenter] removeObserver:config_observer];
        config_observer = nil;
        NSLog(@"🧹 Cleaned up configuration monitoring");
    }
    monitored_engine = NULL;
}

// Test function to simulate configuration changes
void macaudio_simulate_hotplug(void* engine_ptr) {
    NSLog(@"🧪 Simulating hotplug event");
    
    AVAudioEngine* engine;
    BOOL shouldReleaseEngine = NO;
    
    if (engine_ptr == NULL) {
        // For testing: if we have a monitored engine, use it
        // Otherwise create a temporary one
        if (monitored_engine != NULL) {
            NSLog(@"🎯 Using existing monitored engine for simulation");
            engine = monitored_engine;
        } else {
            NSLog(@"📦 Creating temporary AVAudioEngine for simulation");
            engine = [[AVAudioEngine alloc] init];
            shouldReleaseEngine = YES;
        }
    } else {
        engine = (__bridge AVAudioEngine*)engine_ptr;
    }
    
    if (engine == NULL) {
        NSLog(@"❌ Failed to get engine for simulation");
        return;
    }
    
    // Manually post the notification to simulate a device change
    NSLog(@"📤 Posting AVAudioEngineConfigurationChangeNotification");
    [[NSNotificationCenter defaultCenter] 
        postNotificationName:AVAudioEngineConfigurationChangeNotification
                      object:engine];
    
    // Clean up temporary engine if we created one
    if (shouldReleaseEngine) {
        NSLog(@"🧹 Releasing temporary engine");
        engine = nil;
    }
    
    NSLog(@"✅ Hotplug simulation complete");
}
