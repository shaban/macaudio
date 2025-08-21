#import <AVFoundation/AVFoundation.h>

#ifdef __cplusplus
extern "C" {
#endif

// Player result structure
typedef struct {
    void* result;
    const char* error;  // NULL for success, error message for failure
} PlayerResult;

// Player wrapper structure
typedef struct {
    void* playerNode;   // AVAudioPlayerNode*
    void* audioFile;    // AVAudioFile*
    void* engine;       // Reference to the engine this player belongs to
    void* timePitchUnit; // AVAudioUnitTimePitch* (nullable)
    bool isPlaying;     // Track playing state
    bool timePitchEnabled; // Whether time/pitch effects are enabled
} AudioPlayer;

// Function declarations for dynamic library export
PlayerResult audioplayer_new(void* enginePtr);
const char* audioplayer_load_file(AudioPlayer* player, const char* filePath);
const char* audioplayer_play(AudioPlayer* player);
const char* audioplayer_play_at_time(AudioPlayer* player, double timeSeconds);
const char* audioplayer_pause(AudioPlayer* player);
const char* audioplayer_stop(AudioPlayer* player);
const char* audioplayer_is_playing(AudioPlayer* player, bool* result);
const char* audioplayer_get_duration(AudioPlayer* player, double* duration);
const char* audioplayer_get_current_time(AudioPlayer* player, double* currentTime);
const char* audioplayer_seek_to_time(AudioPlayer* player, double timeSeconds);
const char* audioplayer_set_volume(AudioPlayer* player, float volume);
const char* audioplayer_get_volume(AudioPlayer* player, float* volume);
const char* audioplayer_set_pan(AudioPlayer* player, float pan);
const char* audioplayer_get_pan(AudioPlayer* player, float* pan);
const char* audioplayer_set_playback_rate(AudioPlayer* player, float rate);
const char* audioplayer_get_playback_rate(AudioPlayer* player, float* rate);
const char* audioplayer_set_pitch(AudioPlayer* player, float pitch);
const char* audioplayer_get_pitch(AudioPlayer* player, float* pitch);
const char* audioplayer_enable_time_pitch_effects(AudioPlayer* player);
const char* audioplayer_disable_time_pitch_effects(AudioPlayer* player);
const char* audioplayer_is_time_pitch_effects_enabled(AudioPlayer* player, bool* enabled);
PlayerResult audioplayer_get_time_pitch_node_ptr(AudioPlayer* player);
PlayerResult audioplayer_get_node_ptr(AudioPlayer* player);
const char* audioplayer_get_file_info(AudioPlayer* player, double* sampleRate, int* channelCount, const char** format);
void audioplayer_destroy(AudioPlayer* player);

// File-based audio analysis - reads the same data that gets played
const char* audioplayer_analyze_file_segment(AudioPlayer* player, double startTimeSeconds, double durationSeconds, double* rms, int* frameCount);

// Create new audio player
PlayerResult audioplayer_new(void* enginePtr) {
    @autoreleasepool {
        if (!enginePtr) {
            return (PlayerResult){NULL, "Engine pointer is null"};
        }
        
        AVAudioEngine* engine = (__bridge AVAudioEngine*)enginePtr;
        
        // Create a new player node
        AVAudioPlayerNode* playerNode = [[AVAudioPlayerNode alloc] init];
        if (!playerNode) {
            return (PlayerResult){NULL, "Failed to create player node"};
        }
        
        // Attach the player node to the engine
        @try {
            [engine attachNode:playerNode];
        }
        @catch (NSException* exception) {
            NSLog(@"Failed to attach player node: %@", exception.reason);
            return (PlayerResult){NULL, "Failed to attach player node to engine"};
        }
        
        // Create player wrapper
        AudioPlayer* player = malloc(sizeof(AudioPlayer));
        if (!player) {
            return (PlayerResult){NULL, "Memory allocation failed"};
        }
        
        player->playerNode = (__bridge_retained void*)playerNode;
        player->audioFile = NULL;
        player->engine = enginePtr;
        player->timePitchUnit = NULL;
        player->isPlaying = false;
        player->timePitchEnabled = false;
        
        NSLog(@"Created audio player successfully");
        return (PlayerResult){player, NULL};  // NULL = success
    }
}

// Load audio file
const char* audioplayer_load_file(AudioPlayer* player, const char* filePath) {
    @autoreleasepool {
        if (!player) {
            return "Player is null";
        }
        
        if (!filePath) {
            return "File path is null";
        }
        
        NSString* path = [NSString stringWithUTF8String:filePath];
        NSURL* fileURL = [NSURL fileURLWithPath:path];
        
        @try {
            // Release previous audio file if it exists
            if (player->audioFile) {
                AVAudioFile* oldFile = (__bridge_transfer AVAudioFile*)player->audioFile;
                oldFile = nil;
                player->audioFile = NULL;
            }
            
            NSError* error = nil;
            AVAudioFile* audioFile = [[AVAudioFile alloc] initForReading:fileURL error:&error];
            
            if (error || !audioFile) {
                NSLog(@"Failed to load audio file: %@", error.localizedDescription);
                return "Failed to load audio file";
            }
            
            // Store the audio file
            player->audioFile = (__bridge_retained void*)audioFile;
            
            NSLog(@"Loaded audio file: %@ (%.2f seconds, %.0f Hz, %d channels)", 
                  path, 
                  (double)audioFile.length / audioFile.processingFormat.sampleRate,
                  audioFile.processingFormat.sampleRate,
                  audioFile.processingFormat.channelCount);
            
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception loading audio file: %@", exception.reason);
            return "Exception loading audio file";
        }
    }
}

// Play the loaded audio file
const char* audioplayer_play(AudioPlayer* player) {
    @autoreleasepool {
        if (!player || !player->playerNode) {
            return "Player or player node is null";
        }
        
        if (!player->audioFile) {
            return "No audio file loaded";
        }
        
        @try {
            AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
            AVAudioFile* audioFile = (__bridge AVAudioFile*)player->audioFile;
            
            // Schedule the entire file for playback
            [playerNode scheduleFile:audioFile atTime:nil completionHandler:^{
                player->isPlaying = false;
                NSLog(@"Audio playback completed");
            }];
            
            // Start playback
            [playerNode play];
            player->isPlaying = true;
            
            NSLog(@"Started audio playback");
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception during playback: %@", exception.reason);
            return "Failed to start playback";
        }
    }
}

// Play from a specific time (with TimePitch rate compensation)
const char* audioplayer_play_at_time(AudioPlayer* player, double timeSeconds) {
    @autoreleasepool {
        if (!player || !player->playerNode) {
            return "Player or player node is null";
        }
        
        if (!player->audioFile) {
            return "No audio file loaded";
        }
        
        if (timeSeconds < 0.0) {
            return "Time cannot be negative";
        }
        
        @try {
            AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
            AVAudioFile* audioFile = (__bridge AVAudioFile*)player->audioFile;
            
            // Calculate frame position
            AVAudioFramePosition startFrame = (AVAudioFramePosition)(timeSeconds * audioFile.processingFormat.sampleRate);
            AVAudioFrameCount remainingFrames = (AVAudioFrameCount)(audioFile.length - startFrame);
            
            if (startFrame >= audioFile.length) {
                return "Start time is beyond file duration";
            }
            
            // CRITICAL FIX: Adjust frame count for TimePitch rate to ensure proper playback duration
            AVAudioFrameCount frameCount = remainingFrames;
            if (player->timePitchEnabled && player->timePitchUnit) {
                AVAudioUnitTimePitch* timePitchUnit = (__bridge AVAudioUnitTimePitch*)player->timePitchUnit;
                float rate = timePitchUnit.rate;
                
                // CORRECT logic for scheduleSegment with TimePitch:
                // We need to schedule the right amount of SOURCE material to get desired playback time
                // 
                // Fast (rate=2.0): 1.5s source → 3s playback (source gets stretched in time)
                // Slow (rate=0.5): 6s source → 3s playback (source gets compressed in time)
                //
                // Formula: source_frames_needed = target_playback_frames / rate
                // But we want to preserve the natural playback duration, so we use:
                // frameCount = remainingFrames * rate (more frames for slow, fewer for fast)
                
                double originalDurationSeconds = (double)remainingFrames / audioFile.processingFormat.sampleRate;
                
                // Apply rate multiplication: fast rates get fewer frames, slow rates get more
                frameCount = (AVAudioFrameCount)((double)remainingFrames * rate);
                
                // Ensure we don't exceed available frames
                if (frameCount > remainingFrames) {
                    frameCount = remainingFrames;
                }
                
                // Calculate what the actual playback duration will be
                double sourceSecondsScheduled = (double)frameCount / audioFile.processingFormat.sampleRate;
                double expectedPlaybackSeconds = sourceSecondsScheduled / rate;
                
                NSLog(@"TimePitch scheduleSegment: rate=%.2f, original=%.2fs (%u frames), scheduled=%.2fs (%u frames), expected_playback=%.2fs", 
                      rate, originalDurationSeconds, remainingFrames, sourceSecondsScheduled, frameCount, expectedPlaybackSeconds);
            }
            
            // Schedule playback from the specified frame with rate-adjusted frame count
            [playerNode scheduleSegment:audioFile 
                            startingFrame:startFrame 
                            frameCount:frameCount 
                                atTime:nil 
                     completionHandler:^{
                player->isPlaying = false;
                NSLog(@"Audio playbook completed");
            }];
            
            [playerNode play];
            player->isPlaying = true;
            
            NSLog(@"Started audio playback from %.2f seconds (frameCount: %u)", timeSeconds, frameCount);
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception during timed playback: %@", exception.reason);
            return "Failed to start timed playback";
        }
    }
}

// Pause playback
const char* audioplayer_pause(AudioPlayer* player) {
    @autoreleasepool {
        if (!player || !player->playerNode) {
            return "Player or player node is null";
        }
        
        @try {
            AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
            [playerNode pause];
            player->isPlaying = false;
            
            NSLog(@"Paused audio playback");
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception during pause: %@", exception.reason);
            return "Failed to pause playback";
        }
    }
}

// Stop playback
const char* audioplayer_stop(AudioPlayer* player) {
    @autoreleasepool {
        if (!player || !player->playerNode) {
            return "Player or player node is null";
        }
        
        @try {
            AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
            [playerNode stop];
            player->isPlaying = false;
            
            NSLog(@"Stopped audio playback");
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception during stop: %@", exception.reason);
            return "Failed to stop playback";
        }
    }
}

// Check if playing
const char* audioplayer_is_playing(AudioPlayer* player, bool* result) {
    if (!player || !result) {
        return "Invalid parameters";
    }
    
    if (!player->playerNode) {
        *result = false;
        return NULL;
    }
    
    @try {
        AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
        *result = playerNode.isPlaying && player->isPlaying;
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception checking playing state: %@", exception.reason);
        *result = false;
        return "Failed to check playing state";
    }
}

// Get file duration in seconds
const char* audioplayer_get_duration(AudioPlayer* player, double* duration) {
    if (!player || !duration) {
        return "Invalid parameters";
    }
    
    if (!player->audioFile) {
        *duration = 0.0;
        return "No audio file loaded";
    }
    
    @try {
        AVAudioFile* audioFile = (__bridge AVAudioFile*)player->audioFile;
        *duration = (double)audioFile.length / audioFile.processingFormat.sampleRate;
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting duration: %@", exception.reason);
        *duration = 0.0;
        return "Failed to get duration";
    }
}

// Get current playback time (approximation)
const char* audioplayer_get_current_time(AudioPlayer* player, double* currentTime) {
    if (!player || !currentTime) {
        return "Invalid parameters";
    }
    
    *currentTime = 0.0;
    
    if (!player->playerNode || !player->audioFile) {
        return "Player node or audio file is null";
    }
    
    @try {
        AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
        AVAudioFile* audioFile = (__bridge AVAudioFile*)player->audioFile;
        
        // Get the last render time (this is an approximation)
        AVAudioTime* nodeTime = [playerNode lastRenderTime];
        if (nodeTime && nodeTime.isSampleTimeValid) {
            *currentTime = (double)nodeTime.sampleTime / audioFile.processingFormat.sampleRate;
        }
        
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting current time: %@", exception.reason);
        return "Failed to get current time";
    }
}

// Seek to specific time (stop and play from time)
const char* audioplayer_seek_to_time(AudioPlayer* player, double timeSeconds) {
    @autoreleasepool {
        if (!player) {
            return "Player is null";
        }
        
        if (timeSeconds < 0.0) {
            return "Time cannot be negative";
        }
        
        // Stop current playback
        const char* stopResult = audioplayer_stop(player);
        if (stopResult) {
            return stopResult;
        }
        
        // Start playback from the new time
        return audioplayer_play_at_time(player, timeSeconds);
    }
}

// Set volume (0.0 to 1.0)
const char* audioplayer_set_volume(AudioPlayer* player, float volume) {
    if (!player || !player->playerNode) {
        return "Player or player node is null";
    }
    
    if (volume < 0.0f || volume > 1.0f) {
        return "Volume must be between 0.0 and 1.0";
    }
    
    @try {
        AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
        playerNode.volume = volume;
        
        NSLog(@"Set player volume to %.2f", volume);
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception setting volume: %@", exception.reason);
        return "Failed to set volume";
    }
}

// Get volume
const char* audioplayer_get_volume(AudioPlayer* player, float* volume) {
    if (!player || !volume) {
        return "Invalid parameters";
    }
    
    if (!player->playerNode) {
        *volume = 0.0f;
        return "Player node is null";
    }
    
    @try {
        AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
        *volume = playerNode.volume;
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting volume: %@", exception.reason);
        *volume = 0.0f;
        return "Failed to get volume";
    }
}

// Set pan (-1.0 to 1.0)
const char* audioplayer_set_pan(AudioPlayer* player, float pan) {
    if (!player || !player->playerNode) {
        return "Player or player node is null";
    }
    
    if (pan < -1.0f || pan > 1.0f) {
        return "Pan must be between -1.0 and 1.0";
    }
    
    @try {
        AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
        playerNode.pan = pan;
        
        NSLog(@"Set player pan to %.2f", pan);
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception setting pan: %@", exception.reason);
        return "Failed to set pan";
    }
}

// Get pan
const char* audioplayer_get_pan(AudioPlayer* player, float* pan) {
    if (!player || !pan) {
        return "Invalid parameters";
    }
    
    if (!player->playerNode) {
        *pan = 0.0f;
        return "Player node is null";
    }
    
    @try {
        AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
        *pan = playerNode.pan;
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting pan: %@", exception.reason);
        *pan = 0.0f;
        return "Failed to get pan";
    }
}

// Set playback rate (0.25 to 4.0, where 1.0 = normal speed)
const char* audioplayer_set_playback_rate(AudioPlayer* player, float rate) {
    if (!player) {
        return "Player is null";
    }
    
    if (!player->timePitchEnabled || !player->timePitchUnit) {
        return "Time/pitch effects not enabled. Call audioplayer_enable_time_pitch_effects() first";
    }
    
    if (rate < 0.25f || rate > 4.0f) {
        return "Playback rate must be between 0.25 and 4.0";
    }
    
    @try {
        AVAudioUnitTimePitch* timePitchUnit = (__bridge AVAudioUnitTimePitch*)player->timePitchUnit;
        timePitchUnit.rate = rate;
        
        NSLog(@"Set playback rate to %.2f", rate);
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception setting playback rate: %@", exception.reason);
        return "Failed to set playback rate";
    }
}

// Get playback rate
const char* audioplayer_get_playback_rate(AudioPlayer* player, float* rate) {
    if (!player || !rate) {
        return "Invalid parameters";
    }
    
    if (!player->timePitchEnabled || !player->timePitchUnit) {
        *rate = 1.0f;
        return "Time/pitch effects not enabled";
    }
    
    @try {
        AVAudioUnitTimePitch* timePitchUnit = (__bridge AVAudioUnitTimePitch*)player->timePitchUnit;
        *rate = timePitchUnit.rate;
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting playback rate: %@", exception.reason);
        *rate = 1.0f;
        return "Failed to get playback rate";
    }
}

// Set pitch in cents (-2400 to 2400, where 0 = no pitch change)
const char* audioplayer_set_pitch(AudioPlayer* player, float pitch) {
    if (!player) {
        return "Player is null";
    }
    
    if (!player->timePitchEnabled || !player->timePitchUnit) {
        return "Time/pitch effects not enabled. Call audioplayer_enable_time_pitch_effects() first";
    }
    
    if (pitch < -2400.0f || pitch > 2400.0f) {
        return "Pitch must be between -2400 and 2400 cents";
    }
    
    @try {
        AVAudioUnitTimePitch* timePitchUnit = (__bridge AVAudioUnitTimePitch*)player->timePitchUnit;
        timePitchUnit.pitch = pitch;
        
        NSLog(@"Set pitch to %.2f cents", pitch);
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception setting pitch: %@", exception.reason);
        return "Failed to set pitch";
    }
}

// Get pitch in cents
const char* audioplayer_get_pitch(AudioPlayer* player, float* pitch) {
    if (!player || !pitch) {
        return "Invalid parameters";
    }
    
    if (!player->timePitchEnabled || !player->timePitchUnit) {
        *pitch = 0.0f;
        return "Time/pitch effects not enabled";
    }
    
    @try {
        AVAudioUnitTimePitch* timePitchUnit = (__bridge AVAudioUnitTimePitch*)player->timePitchUnit;
        *pitch = timePitchUnit.pitch;
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting pitch: %@", exception.reason);
        *pitch = 0.0f;
        return "Failed to get pitch";
    }
}

// Enable time/pitch effects by inserting AVAudioUnitTimePitch between player and output
const char* audioplayer_enable_time_pitch_effects(AudioPlayer* player) {
    @autoreleasepool {
        if (!player || !player->playerNode || !player->engine) {
            return "Player, player node, or engine is null";
        }
        
        if (player->timePitchEnabled) {
            return "Time/pitch effects are already enabled";
        }
        
        @try {
            AVAudioEngine* engine = (__bridge AVAudioEngine*)player->engine;
            AVAudioPlayerNode* playerNode = (__bridge AVAudioPlayerNode*)player->playerNode;
            
            // Create the TimePitch unit
            AVAudioUnitTimePitch* timePitchUnit = [[AVAudioUnitTimePitch alloc] init];
            if (!timePitchUnit) {
                return "Failed to create TimePitch unit";
            }
            
            // Attach the TimePitch unit to the engine
            [engine attachNode:timePitchUnit];
            
            // Store the TimePitch unit reference
            player->timePitchUnit = (__bridge_retained void*)timePitchUnit;
            player->timePitchEnabled = true;
            
            NSLog(@"Enabled time/pitch effects - ready for rate and pitch adjustments");
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception enabling time/pitch effects: %@", exception.reason);
            return "Failed to enable time/pitch effects";
        }
    }
}

// Disable time/pitch effects and remove the TimePitch unit
const char* audioplayer_disable_time_pitch_effects(AudioPlayer* player) {
    @autoreleasepool {
        if (!player) {
            return "Player is null";
        }
        
        if (!player->timePitchEnabled) {
            return "Time/pitch effects are not enabled";
        }
        
        @try {
            // Stop playback first
            if (player->isPlaying) {
                audioplayer_stop(player);
            }
            
            // Remove TimePitch unit from engine
            if (player->timePitchUnit && player->engine) {
                AVAudioEngine* engine = (__bridge AVAudioEngine*)player->engine;
                AVAudioUnitTimePitch* timePitchUnit = (__bridge_transfer AVAudioUnitTimePitch*)player->timePitchUnit;
                
                [engine detachNode:timePitchUnit];
                timePitchUnit = nil;
                player->timePitchUnit = NULL;
            }
            
            player->timePitchEnabled = false;
            
            NSLog(@"Disabled time/pitch effects");
            return NULL;  // NULL = success
        }
        @catch (NSException* exception) {
            NSLog(@"Exception disabling time/pitch effects: %@", exception.reason);
            return "Failed to disable time/pitch effects";
        }
    }
}

// Check if time/pitch effects are enabled
const char* audioplayer_is_time_pitch_effects_enabled(AudioPlayer* player, bool* enabled) {
    if (!player || !enabled) {
        return "Invalid parameters";
    }
    
    *enabled = player->timePitchEnabled;
    return NULL;  // NULL = success
}

// Get the TimePitch unit node pointer (for connecting in audio chain)
PlayerResult audioplayer_get_time_pitch_node_ptr(AudioPlayer* player) {
    if (!player) {
        return (PlayerResult){NULL, "Player is null"};
    }
    
    if (!player->timePitchEnabled || !player->timePitchUnit) {
        return (PlayerResult){NULL, "Time/pitch effects not enabled"};
    }
    
    return (PlayerResult){player->timePitchUnit, NULL};  // NULL = success
}

// Get the player node pointer (for connecting to other nodes)
PlayerResult audioplayer_get_node_ptr(AudioPlayer* player) {
    if (!player || !player->playerNode) {
        return (PlayerResult){NULL, "Player or player node is null"};
    }
    
    return (PlayerResult){player->playerNode, NULL};  // NULL = success
}

// Get file information
const char* audioplayer_get_file_info(AudioPlayer* player, double* sampleRate, int* channelCount, const char** format) {
    if (!player || !sampleRate || !channelCount || !format) {
        return "Invalid parameters";
    }
    
    if (!player->audioFile) {
        *sampleRate = 0.0;
        *channelCount = 0;
        *format = "No file loaded";
        return "No audio file loaded";
    }
    
    @try {
        AVAudioFile* audioFile = (__bridge AVAudioFile*)player->audioFile;
        *sampleRate = audioFile.processingFormat.sampleRate;
        *channelCount = (int)audioFile.processingFormat.channelCount;
        
        // Get file format description
        static NSString* formatDescription;
        formatDescription = [audioFile.fileFormat description];
        *format = [formatDescription UTF8String];
        
        return NULL;  // NULL = success
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting file info: %@", exception.reason);
        *sampleRate = 0.0;
        *channelCount = 0;
        *format = "Error getting file info";
        return "Failed to get file info";
    }
}

// Destroy player and free resources
void audioplayer_destroy(AudioPlayer* player) {
    if (!player) {
        return;
    }
    
    @autoreleasepool {
        // Stop playback if playing
        if (player->isPlaying) {
            audioplayer_stop(player);
        }
        
        // Release TimePitch unit first (if enabled)
        if (player->timePitchUnit && player->engine) {
            @try {
                AVAudioEngine* engine = (__bridge AVAudioEngine*)player->engine;
                AVAudioUnitTimePitch* timePitchUnit = (__bridge_transfer AVAudioUnitTimePitch*)player->timePitchUnit;
                
                [engine detachNode:timePitchUnit];
                timePitchUnit = nil;
                player->timePitchUnit = NULL;
                NSLog(@"Released TimePitch unit");
            }
            @catch (NSException* exception) {
                NSLog(@"Exception releasing TimePitch unit: %@", exception.reason);
            }
        }
        
        // Release player node
        if (player->playerNode) {
            AVAudioPlayerNode* playerNode = (__bridge_transfer AVAudioPlayerNode*)player->playerNode;
            
            // Detach from engine if still attached
            if (player->engine) {
                @try {
                    AVAudioEngine* engine = (__bridge AVAudioEngine*)player->engine;
                    [engine detachNode:playerNode];
                    NSLog(@"Detached player node from engine");
                }
                @catch (NSException* exception) {
                    NSLog(@"Exception detaching player node: %@", exception.reason);
                }
            }
            
            playerNode = nil;
            player->playerNode = NULL;
        }
        
        // Release audio file
        if (player->audioFile) {
            AVAudioFile* audioFile = (__bridge_transfer AVAudioFile*)player->audioFile;
            audioFile = nil;
            player->audioFile = NULL;
        }
        
        // Clear engine reference
        player->engine = NULL;
        player->timePitchUnit = NULL;
        player->isPlaying = false;
        player->timePitchEnabled = false;
        
        NSLog(@"Audio player destroyed");
    }
    
    // Free the wrapper
    free(player);
}

// File-based audio analysis - reads the same data that gets played
const char* audioplayer_analyze_file_segment(AudioPlayer* player, double startTimeSeconds, double durationSeconds, double* rms, int* frameCount) {
    @autoreleasepool {
        if (!player || !player->audioFile) {
            return "No audio file loaded";
        }
        
        if (startTimeSeconds < 0.0 || durationSeconds <= 0.0) {
            return "Invalid time parameters";
        }
        
        @try {
            AVAudioFile* audioFile = (__bridge AVAudioFile*)player->audioFile;
            
            // Calculate frame range to analyze
            double sampleRate = audioFile.processingFormat.sampleRate;
            AVAudioFramePosition startFrame = (AVAudioFramePosition)(startTimeSeconds * sampleRate);
            AVAudioFrameCount analysisFrames = (AVAudioFrameCount)(durationSeconds * sampleRate);
            
            // Ensure we don't read beyond file bounds
            if (startFrame >= audioFile.length) {
                *rms = 0.0;
                *frameCount = 0;
                return NULL; // Valid - just past end of file
            }
            
            if (startFrame + analysisFrames > audioFile.length) {
                analysisFrames = (AVAudioFrameCount)(audioFile.length - startFrame);
            }
            
            // Create buffer for reading file data
            AVAudioPCMBuffer* buffer = [[AVAudioPCMBuffer alloc] initWithPCMFormat:audioFile.processingFormat frameCapacity:analysisFrames];
            if (!buffer) {
                return "Failed to create analysis buffer";
            }
            
            // Read the exact audio data that would be played
            audioFile.framePosition = startFrame;
            NSError* error = nil;
            [audioFile readIntoBuffer:buffer frameCount:analysisFrames error:&error];
            
            if (error) {
                return [[NSString stringWithFormat:@"Failed to read audio data: %@", error.localizedDescription] UTF8String];
            }
            
            // Calculate RMS from the raw audio data
            float calculatedRMS = 0.0f;
            int channels = buffer.format.channelCount;
            int frames = (int)buffer.frameLength;
            
            if (frames > 0) {
                // Analyze all channels and average
                for (int channel = 0; channel < channels; channel++) {
                    float* channelData = buffer.floatChannelData[channel];
                    float channelRMS = 0.0f;
                    
                    for (int i = 0; i < frames; i++) {
                        channelRMS += channelData[i] * channelData[i];
                    }
                    channelRMS = sqrtf(channelRMS / frames);
                    calculatedRMS += channelRMS;
                }
                calculatedRMS /= channels; // Average across channels
            }
            
            *rms = (double)calculatedRMS;
            *frameCount = frames;
            
            return NULL; // Success
            
        } @catch (NSException* exception) {
            return [[NSString stringWithFormat:@"Exception analyzing audio: %@", exception.reason] UTF8String];
        }
    }
}

#ifdef __cplusplus
}
#endif
