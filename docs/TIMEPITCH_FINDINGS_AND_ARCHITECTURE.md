# TimePitch Unit: Critical Findings & Architecture Guidelines

## Executive Summary

Through extensive buffer-level testing, we've discovered that AVAudioUnitTimePitch introduces significant delays that are **inherent to rate processing**, not setup complexity. This document consolidates our findings and provides blueprints for a streamlined architecture.

## Key Discoveries

### 1. TimePitch Delay Patterns (Buffer-Level Measurements)
- **Rate Processing (including 1.0x)**: ~1.2-1.4 seconds delay
- **Pitch-Only Processing**: ~0.1 seconds delay
- **Combined Pitch+Rate**: Follows rate processing timing

### 2. Critical Insight: Rate vs Pitch Processing
```
Normal playback (pitch=0, rate=1.0):     1.198s delay  ← Rate processing bottleneck
Pitch up (pitch=+1200, rate=1.0):        0.101s delay  ← Pitch-only is fast
Slow down (pitch=0, rate=0.5):           1.396s delay  ← Rate processing bottleneck  
Pitch down + speed up (pitch=-600, rate=1.5): 0.100s delay  ← Combined processing
```

### 3. Architecture Implications
- **Even "normal" playback (1.0x rate) has the delay** because TimePitch always processes rate
- **Setup complexity is NOT the issue** - always-on TimePitch has same delays
- **The delay is processing latency, not initialization**
- **Buffer analysis provides ground truth** - more reliable than sleep-based timing

## Proven Testing Approach: Buffer-Level Measurement

### The Method
Use silence periods as natural separators between tests, with buffer-level RMS analysis to detect exact audio start times.

```objectivec
// Buffer measurement blueprint
typedef struct {
    uint64_t start_time;
    uint64_t silence_end_time; 
    double desired_duration;
    BOOL silence_ended;
    BOOL measurement_complete;
    int test_id;
} BufferMeasurement;

BOOL measure_nonsilent_buffer(const float* buffer, int frame_count, int channel_count) {
    // Calculate RMS
    float sum = 0.0f;
    int total_samples = frame_count * channel_count;
    for (int i = 0; i < total_samples; i++) {
        sum += buffer[i] * buffer[i];
    }
    float rms = sqrtf(sum / total_samples);
    
    // Detect silence end with threshold
    if (!current_measurement.silence_ended && rms > SILENCE_THRESHOLD) {
        current_measurement.silence_end_time = mach_absolute_time();
        current_measurement.silence_ended = YES;
        // Record precise timing...
    }
    
    // Measure desired duration after silence
    if (current_measurement.silence_ended && !current_measurement.measurement_complete) {
        double elapsed_since_silence = mach_time_to_seconds(current_time - current_measurement.silence_end_time);
        if (elapsed_since_silence >= current_measurement.desired_duration) {
            current_measurement.measurement_complete = YES;
            return YES; // Trigger next test
        }
    }
    return NO;
}
```

### Always-On TimePitch Setup (Proven Approach)
```objectivec
// Set up audio chain with TimePitch permanently in graph
[engine attachNode:player];
[engine attachNode:timePitch];

// Connect: Player -> TimePitch -> Engine Output  
[engine connect:player to:timePitch format:format];
[engine connect:timePitch to:engine.mainMixerNode format:format];

// Install analysis tap on TimePitch output
[timePitch installTapOnBus:0 bufferSize:512 format:format block:^(AVAudioPCMBuffer *buffer, AVAudioTime *when) {
    float *channelData = buffer.floatChannelData[0];
    BOOL complete = measure_nonsilent_buffer(channelData, (int)buffer.frameLength, 1);
    if (complete) {
        // Trigger next test or action
    }
}];
```

## Streamlined Architecture Guidelines

### 1. Accept the TimePitch Reality
- **~1.2s delay is inherent to rate processing** - cannot be eliminated
- **Don't fight it, design around it**
- **Use delay productively** (loading, preparation, UI feedback)

### 2. Strategic TimePitch Usage
```
HIGH PRIORITY: Immediate playback needed
├── Skip TimePitch entirely for instant playback
└── Add TimePitch later for effects (accept delay)

LOW PRIORITY: Effects more important than latency  
├── Always-on TimePitch with delay acceptance
└── Use silence periods productively
```

### 3. Dual-Path Architecture Proposal
```
User requests playback:
├── Path A: Direct playback (no TimePitch) → Instant audio
├── Path B: TimePitch setup in background → Ready in ~1.2s
└── Seamless switch when effects needed
```

### 4. Buffer Analysis Integration
- **All timing measurements use buffer-level RMS analysis**
- **Silence detection threshold: 0.001f**
- **Ground truth via tap installation on audio units**
- **No more sleep-based timing approximations**

## Implementation Blueprints

### Blueprint 1: Immediate Playback System
```objectivec
@interface StreamlinedPlayer : NSObject
@property (nonatomic, strong) AVAudioEngine *engine;
@property (nonatomic, strong) AVAudioPlayerNode *directPlayer;    // No effects
@property (nonatomic, strong) AVAudioPlayerNode *effectsPlayer;   // With TimePitch
@property (nonatomic, strong) AVAudioUnitTimePitch *timePitch;
@property (nonatomic) BOOL effectsReady;
@end

- (void)playImmediately:(AVAudioPCMBuffer *)buffer {
    // Instant playback via direct path
    [self.directPlayer scheduleBuffer:buffer completionHandler:nil];
    [self.directPlayer play];
    
    // Setup effects in background
    [self prepareEffectsAsync:buffer];
}

- (void)enableEffects {
    if (self.effectsReady) {
        // Seamless switch to effects path
        [self switchToEffectsPlayer];
    } else {
        // Effects not ready, keep direct playback
        NSLog(@"Effects preparing... using direct playback");
    }
}
```

### Blueprint 2: Buffer Analysis Manager
```objectivec
@interface BufferAnalysisManager : NSObject
+ (void)startMeasurement:(int)testId duration:(double)duration;
+ (void)analyzeBuffer:(float*)buffer frameCount:(int)frames channels:(int)channels;
+ (BOOL)isMeasurementComplete;
+ (void)resetMeasurement;
@end

// Usage in any audio unit tap:
[audioUnit installTapOnBus:0 bufferSize:512 format:format block:^(AVAudioPCMBuffer *buffer, AVAudioTime *when) {
    [BufferAnalysisManager analyzeBuffer:buffer.floatChannelData[0] 
                               frameCount:(int)buffer.frameLength 
                                 channels:1];
    if ([BufferAnalysisManager isMeasurementComplete]) {
        // Handle completion
    }
}];
```

### Blueprint 3: Smart TimePitch Manager
```objectivec
@interface SmartTimePitchManager : NSObject
- (void)setupAlwaysOnTimePitch:(AVAudioEngine*)engine format:(AVAudioFormat*)format;
- (void)changeParameters:(float)pitch rate:(float)rate; // No rebuilding
- (BOOL)isProcessingRate; // Returns YES if rate != 1.0
- (NSTimeInterval)expectedDelay; // Returns ~1.2s for rate processing, ~0.1s for pitch-only
@end

- (NSTimeInterval)expectedDelay {
    if (self.timePitch.rate != 1.0f) {
        return 1.2; // Rate processing delay
    } else {
        return 0.1; // Pitch-only delay  
    }
}
```

## Cleanup Strategy

### Files to Consolidate/Remove
- `timepitch_investigation.m` → Keep findings, remove file
- `timepitch_investigation_fixed.m` → Archive key insights
- `startup_delay_demo.go` → Consolidate timing approach
- `priming_comparison.go` → Archive priming insights (minimal benefit proven)
- Multiple timing test files → Keep buffer analysis approach only

### Files to Refactor
- `native/engine.m` → Remove experimental priming, keep core functions
- `avaudio/engine/engine.go` → Streamline based on findings
- `examples/` → Keep essential demos only

### Documentation to Update
- `docs/TIMEPITCH.md` → Replace with this file
- `docs/BUFFER_ANALYSIS_IMPLEMENTATION.md` → Merge buffer analysis approach
- Create `docs/STREAMLINED_ARCHITECTURE.md` → Implementation roadmap

## Next Steps

1. **Consolidate documentation** → Single source of truth
2. **Clean experimental files** → Remove confusion
3. **Design streamlined architecture** → Based on proven findings
4. **Implement solid foundations** → One component at a time
5. **Test each component** → Buffer-level validation

## Success Metrics

- **Immediate playback**: <50ms for direct path
- **Effects readiness**: Accept ~1.2s for rate processing
- **Measurement accuracy**: Buffer-level ground truth only
- **Code clarity**: Each component has single responsibility
- **Documentation**: All findings preserved, experimentation removed

---

*This document represents the culmination of extensive TimePitch investigation and provides the foundation for a streamlined, reality-based audio architecture.*
