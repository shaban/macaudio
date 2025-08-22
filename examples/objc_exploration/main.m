#import <AVFoundation/AVFoundation.h>
#import <Foundation/Foundation.h>

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        NSLog(@"🔬 AVAudioEngine TimePitch Exploration");
        NSLog(@"======================================");
        
        // Create engine and nodes
        AVAudioEngine *engine = [[AVAudioEngine alloc] init];
        AVAudioPlayerNode *player = [[AVAudioPlayerNode alloc] init];
        AVAudioUnitTimePitch *timePitch = [[AVAudioUnitTimePitch alloc] init];
        
        NSLog(@"✅ Created engine and nodes");
        
        // Load audio file
        NSString *audioPath = @"../../avaudio/engine/idea.m4a";
        NSURL *audioURL = [NSURL fileURLWithPath:audioPath];
        
        NSError *error = nil;
        AVAudioFile *audioFile = [[AVAudioFile alloc] initForReading:audioURL error:&error];
        if (!audioFile) {
            NSLog(@"❌ Failed to load audio file: %@", error.localizedDescription);
            return 1;
        }
        
        NSLog(@"✅ Loaded audio file: %@", audioPath);
        NSLog(@"   Duration: %.2f seconds", (double)audioFile.length / audioFile.processingFormat.sampleRate);
        NSLog(@"   Sample Rate: %.0f Hz", audioFile.processingFormat.sampleRate);
        NSLog(@"   Channels: %u", audioFile.processingFormat.channelCount);
        
        // Attach nodes to engine
        [engine attachNode:player];
        [engine attachNode:timePitch];
        NSLog(@"✅ Attached nodes to engine");
        
        // Set player volume to ensure we hear audio
        player.volume = 0.8;
        NSLog(@"🔊 Set player volume to 0.8");
        
        // TEST 1: Direct connection (Player -> MainMixer)
        NSLog(@"\n🧪 TEST 1: Direct Connection (Player -> MainMixer)");
        @try {
            [engine connect:player to:engine.mainMixerNode format:audioFile.processingFormat];
            NSLog(@"✅ Connected Player -> MainMixer");
        }
        @catch (NSException *exception) {
            NSLog(@"❌ Connection failed: %@", exception.reason);
        }
        
        // Start engine
        NSError *startError = nil;
        BOOL started = [engine startAndReturnError:&startError];
        if (!started) {
            NSLog(@"❌ Failed to start engine: %@", startError.localizedDescription);
            return 1;
        }
        NSLog(@"✅ Engine started successfully");
        
        // Schedule and play audio with proper buffer management
        AVAudioPCMBuffer *buffer = [[AVAudioPCMBuffer alloc] initWithPCMFormat:audioFile.processingFormat frameCapacity:(AVAudioFrameCount)(audioFile.processingFormat.sampleRate * 3.0)];
        
        NSError *readError = nil;
        [audioFile readIntoBuffer:buffer error:&readError];
        if (readError) {
            NSLog(@"❌ Failed to read audio into buffer: %@", readError.localizedDescription);
            return 1;
        }
        
        [player scheduleBuffer:buffer completionHandler:^{
            NSLog(@"🎵 Direct playback buffer completed");
        }];
        
        NSLog(@"▶️  Playing direct connection for 3 seconds...");
        [player play];
        
        // Wait for playback
        [[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:3.0]];
        
        [player stop];
        [engine stop];
        NSLog(@"⏹️  Stopped direct connection test\n");
        
        // TEST 2: TimePitch connection (Player -> TimePitch -> MainMixer)
        NSLog(@"🧪 TEST 2: TimePitch Connection (Player -> TimePitch -> MainMixer)");
        
        // Disconnect previous connections
        [engine disconnectNodeInput:engine.mainMixerNode bus:0];
        NSLog(@"🔌 Disconnected previous connections");
        
        // Connect through TimePitch unit
        @try {
            [engine connect:player to:timePitch format:audioFile.processingFormat];
            NSLog(@"✅ Connected Player -> TimePitch");
            NSLog(@"   Player outputs: %lu", (unsigned long)player.numberOfOutputs);
            NSLog(@"   TimePitch inputs: %lu, outputs: %lu", (unsigned long)timePitch.numberOfInputs, (unsigned long)timePitch.numberOfOutputs);
            
            [engine connect:timePitch to:engine.mainMixerNode format:audioFile.processingFormat];
            NSLog(@"✅ Connected TimePitch -> MainMixer");
            NSLog(@"   MainMixer inputs: %lu", (unsigned long)engine.mainMixerNode.numberOfInputs);
        }
        @catch (NSException *exception) {
            NSLog(@"❌ TimePitch connection failed: %@", exception.reason);
            return 1;
        }
        
        // Restart engine
        startError = nil;
        started = [engine startAndReturnError:&startError];
        if (!started) {
            NSLog(@"❌ Failed to restart engine: %@", startError.localizedDescription);
            return 1;
        }
        NSLog(@"✅ Engine restarted successfully");
        
        // Test different TimePitch settings - CORRECTED after timestamp analysis
        double targetPlaybackTime = 3.0;  // We want exactly 3 seconds of playback for all tests
        
        NSArray *testCases = @[
            @{@"name": @"Normal (rate=1.0, pitch=0)", @"rate": @1.0, @"pitch": @0.0},
            @{@"name": @"Slow (rate=0.5, pitch=0)", @"rate": @0.5, @"pitch": @0.0},
            @{@"name": @"Fast (rate=2.0, pitch=0)", @"rate": @2.0, @"pitch": @0.0},
            @{@"name": @"High Pitch (rate=1.0, pitch=+600)", @"rate": @1.0, @"pitch": @600.0},
            @{@"name": @"Deep Voice (rate=1.0, pitch=-600)", @"rate": @1.0, @"pitch": @-600.0}
        ];
        
        for (NSDictionary *testCase in testCases) {
            NSLog(@"\n🎛️  Testing: %@", testCase[@"name"]);
            
            // Set TimePitch parameters
            float rate = [testCase[@"rate"] floatValue];
            float pitch = [testCase[@"pitch"] floatValue];
            
            // CORRECTED formula after timestamp analysis: buffer_duration = target_playback_time * rate
            // Fast rate consumes buffer faster, so we need MORE buffer content
            double bufferDuration = targetPlaybackTime * rate;
            
            timePitch.rate = rate;
            timePitch.pitch = pitch;
            
            NSLog(@"   Set rate=%.1f, pitch=%.0f cents", rate, pitch);
            NSLog(@"   CORRECTED Formula: %.1fs target × %.1f rate = %.1fs buffer needed", targetPlaybackTime, rate, bufferDuration);
            
            // Create buffer with CORRECTED duration calculation
            AVAudioFrameCount frameCapacity = (AVAudioFrameCount)(audioFile.processingFormat.sampleRate * bufferDuration);
            AVAudioPCMBuffer *testBuffer = [[AVAudioPCMBuffer alloc] initWithPCMFormat:audioFile.processingFormat 
                                                                         frameCapacity:frameCapacity];
            
            // Reset file position and read fresh audio
            audioFile.framePosition = 0;
            NSError *readError = nil;
            BOOL success = [audioFile readIntoBuffer:testBuffer error:&readError];
            if (!success || readError) {
                NSLog(@"   ❌ Failed to read audio into buffer: %@", readError.localizedDescription);
                continue;
            }
            
            NSLog(@"   📊 Buffer: %.1fs at %.0f Hz = %u frames (target: %.1fs playback)", 
                  (double)testBuffer.frameLength / testBuffer.format.sampleRate,
                  testBuffer.format.sampleRate, 
                  testBuffer.frameLength,
                  targetPlaybackTime);
            
            // Schedule the buffer
            [player scheduleBuffer:testBuffer completionHandler:^{
                NSLog(@"   🎵 %@ buffer completed", testCase[@"name"]);
            }];
            
            NSLog(@"   ▶️  Playing (target: %.1fs of audio)...", targetPlaybackTime);
            @try {
                [player play];
                NSLog(@"   ✅ Play() succeeded - measuring actual duration...");
            }
            @catch (NSException *exception) {
                NSLog(@"   ⚠️  Play() exception: %@", exception.reason);
            }
            
            // Wait for target playback time + 0.5s buffer
            double waitTime = targetPlaybackTime + 0.5;
            [[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:waitTime]];
            
            [player stop];
            NSLog(@"   ⏹️  Stopped after %.1fs wait time", waitTime);
        }
        
        // TEST 3: Connection timing experiment
        NSLog(@"\n🧪 TEST 3: Connection Timing Experiment");
        NSLog(@"Testing if connection order affects disconnected state warnings...");
        
        [engine stop];
        
        // Disconnect everything
        [engine disconnectNodeInput:timePitch bus:0];
        [engine disconnectNodeInput:engine.mainMixerNode bus:0];
        NSLog(@"🔌 Disconnected all connections");
        
        // Try connecting BEFORE starting engine
        NSLog(@"🔗 Connecting nodes before engine start...");
        @try {
            [engine connect:player to:timePitch format:audioFile.processingFormat];
            [engine connect:timePitch to:engine.mainMixerNode format:audioFile.processingFormat];
            NSLog(@"✅ Pre-start connections established");
        }
        @catch (NSException *exception) {
            NSLog(@"❌ Pre-start connection failed: %@", exception.reason);
        }
        
        // NOW start the engine
        startError = nil;
        started = [engine startAndReturnError:&startError];
        if (!started) {
            NSLog(@"❌ Failed to start engine after pre-connection: %@", startError.localizedDescription);
            return 1;
        }
        NSLog(@"✅ Engine started after pre-connection");
        
        // Test if this eliminates the warning
        timePitch.rate = 0.7f;
        timePitch.pitch = 400.0f;
        
        // Create buffer for final test
        AVAudioPCMBuffer *finalBuffer = [[AVAudioPCMBuffer alloc] initWithPCMFormat:audioFile.processingFormat 
                                                                       frameCapacity:(AVAudioFrameCount)(audioFile.processingFormat.sampleRate * 3.0)];
        
        audioFile.framePosition = 0;
        NSError *finalReadError = nil;
        [audioFile readIntoBuffer:finalBuffer error:&finalReadError];
        if (finalReadError) {
            NSLog(@"❌ Failed to read final buffer: %@", finalReadError.localizedDescription);
            return 1;
        }
        
        [player scheduleBuffer:finalBuffer completionHandler:^{
            NSLog(@"🎵 Pre-connection timing test completed");
        }];
        
        NSLog(@"▶️  Testing pre-connection timing (rate=0.7, pitch=+400)...");
        @try {
            [player play];
            NSLog(@"✅ Pre-connection play() - checking for warnings...");
        }
        @catch (NSException *exception) {
            NSLog(@"⚠️  Pre-connection play() exception: %@", exception.reason);
        }
        
        [[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:3.5]];
        
        [player stop];
        [engine stop];
        
        NSLog(@"\n🎉 AVAudioEngine Exploration Complete!");
        NSLog(@"Key findings:");
        NSLog(@"- Direct connections: Check console for warnings");
        NSLog(@"- TimePitch connections: Check console for 'disconnected state' warnings");
        NSLog(@"- Connection timing: Check if pre-connection eliminates warnings");
        NSLog(@"- This isolates native AVAudioEngine behavior from our Go wrapper");
        
        return 0;
    }
}
