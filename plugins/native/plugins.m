#import <Foundation/Foundation.h>
#import <AudioToolbox/AudioToolbox.h>
#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AUComponent.h>

// Global verbose logging variable (defined here, not extern)
int g_verboseLogging = 0;  // Default to silent
double g_preset_loading_timeout = 0.0; // Default timeout for preset loading in seconds
double g_process_update_timeout = 0.0; // Default timeout for process update in seconds
double g_total_timeout = 2.0; // Default total timeout for all operations in seconds

// Setter function for verbose logging
void SetVerboseLogging(int enabled) {
    g_verboseLogging = enabled;
}

// Setter functions for timeouts (controlled from Go)
void SetPresetLoadingTimeout(double seconds) { g_preset_loading_timeout = seconds; }
void SetProcessUpdateTimeout(double seconds) { g_process_update_timeout = seconds; }
void SetTotalTimeout(double seconds) { g_total_timeout = seconds; }
// Millisecond convenience wrappers to avoid C.double casts in some cgo setups
void SetPresetLoadingTimeoutMillis(long long ms) { g_preset_loading_timeout = ((double)ms) / 1000.0; }
void SetProcessUpdateTimeoutMillis(long long ms) { g_process_update_timeout = ((double)ms) / 1000.0; }
void SetTotalTimeoutMillis(long long ms) { g_total_timeout = ((double)ms) / 1000.0; }

// Conditional logging macros
#define VERBOSE_LOG(...) do { if (g_verboseLogging) fprintf(stderr, __VA_ARGS__); } while(0)
#define PROGRESS_LOG(...) fprintf(stderr, __VA_ARGS__)  // Always show progress

// Helper function to convert a FourCharCode (OSType) to an NSString
NSString* StringFromFourCharCode(FourCharCode code) {
    char chars[5];
    chars[0] = (char)((code >> 24) & 0xFF);
    chars[1] = (char)((code >> 16) & 0xFF);
    chars[2] = (char)((code >> 8) & 0xFF);
    chars[3] = (char)(code & 0xFF);
    chars[4] = '\0';
    return [NSString stringWithCString:chars encoding:NSASCIIStringEncoding];
}

// Helper function to convert NSString to FourCharCode (OSType)
FourCharCode FourCharCodeFromString(NSString* string) {
    if (!string || [string length] < 4) {
        return 0;
    }
    
    const char* chars = [string UTF8String];
    return (FourCharCode)((chars[0] << 24) | (chars[1] << 16) | (chars[2] << 8) | chars[3]);
}

// Helper function to convert AudioUnitParameterUnit to NSString
NSString* StringFromAudioUnitParameterUnit(AudioUnitParameterUnit unit) {
    switch (unit) {
        case kAudioUnitParameterUnit_Generic: return @"Generic";
        case kAudioUnitParameterUnit_Indexed: return @"Indexed";
        case kAudioUnitParameterUnit_Boolean: return @"Boolean";
        case kAudioUnitParameterUnit_Percent: return @"Percent";
        case kAudioUnitParameterUnit_Seconds: return @"Seconds";
        case kAudioUnitParameterUnit_SampleFrames: return @"Sample Frames";
        case kAudioUnitParameterUnit_Phase: return @"Phase";
        case kAudioUnitParameterUnit_Rate: return @"Rate";
        case kAudioUnitParameterUnit_Hertz: return @"Hertz";
        case kAudioUnitParameterUnit_Cents: return @"Cents";
        case kAudioUnitParameterUnit_RelativeSemiTones: return @"Relative Semitones";
        case kAudioUnitParameterUnit_MIDINoteNumber: return @"MIDI Note Number";
        case kAudioUnitParameterUnit_MIDIController: return @"MIDI Controller";
        case kAudioUnitParameterUnit_Decibels: return @"Decibels";
        case kAudioUnitParameterUnit_LinearGain: return @"Linear Gain";
        case kAudioUnitParameterUnit_Degrees: return @"Degrees";
        case kAudioUnitParameterUnit_Meters: return @"Meters";
        case kAudioUnitParameterUnit_AbsoluteCents: return @"Absolute Cents";
        case kAudioUnitParameterUnit_Octaves: return @"Octaves";
        case kAudioUnitParameterUnit_BPM: return @"BPM";
        case kAudioUnitParameterUnit_Beats: return @"Beats";
        case kAudioUnitParameterUnit_Milliseconds: return @"Milliseconds";
        case kAudioUnitParameterUnit_Ratio: return @"Ratio";
        case kAudioUnitParameterUnit_CustomUnit: return @"Custom Unit";
        // Missing units from Apple documentation:
        case kAudioUnitParameterUnit_EqualPowerCrossfade: return @"Equal Power Crossfade";
        case kAudioUnitParameterUnit_MixerFaderCurve1: return @"Mixer Fader Curve 1";
        case kAudioUnitParameterUnit_Pan: return @"Pan";
        case kAudioUnitParameterUnit_MIDI2Controller: return @"MIDI 2.0 Controller";
        default: return [NSString stringWithFormat:@"Unknown (%lu)", (unsigned long)unit];
    }
}

@interface AudioUnitInspector : NSObject
- (void)forceParameterInitialization:(AUParameter *)param;
- (NSArray *)getIndexedValuesUsingReflection:(AUParameter *)param;
- (NSArray *)queryParameterAtDifferentStates:(AUParameter *)param audioUnit:(AUAudioUnit *)audioUnit;
- (BOOL)isPresetParameter:(AUParameter *)param audioUnit:(AUAudioUnit *)audioUnit;
- (void)initializePluginAudioUnit:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock;
- (void)simulateAudioProcessing:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock;
- (void)extractIndexedParameterInfo:(AUParameter *)param paramData:(NSMutableDictionary *)paramData audioUnit:(AUAudioUnit *)audioUnit;
- (void)processParametersForAudioUnit:(AUAudioUnit *)audioUnit withName:(NSString *)auName auParameters:(NSMutableArray *)auParameters;
@end

@implementation AudioUnitInspector

- (void)forceParameterInitialization:(AUParameter *)param {
    if (param.unit == kAudioUnitParameterUnit_Indexed) {
        // Set the parameter to different values to trigger initialization
        float currentValue = param.value;

        for (float testValue = param.minValue; testValue <= param.maxValue && testValue <= param.minValue + 5; testValue += 1.0f) {
            param.value = testValue;
            // Small delay to let the parameter settle
            usleep(10000); // 10ms
        }
        param.value = currentValue; // Restore original value
    }
}

- (NSArray *)getIndexedValuesUsingReflection:(AUParameter *)param {
    NSArray *privateValueStrings = nil;
    NSArray *possiblePropertyNames = @[@"_valueStrings", @"_indexedStrings", @"_strings", @"_values", @"valueStrings"];

    for (NSString *propertyName in possiblePropertyNames) {
        @try {
            id value = [param valueForKey:propertyName];
            if ([value isKindOfClass:[NSArray class]]) {
                privateValueStrings = (NSArray *)value;
                if (privateValueStrings.count > 0) {
                    return privateValueStrings;
                }
            }
        } @catch (NSException *exception) {
            // Property doesn't exist or isn't accessible - this is expected for most properties
        }
    }

    return nil;
}

- (NSArray *)queryParameterAtDifferentStates:(AUParameter *)param audioUnit:(AUAudioUnit *)audioUnit {
    NSMutableSet *discoveredValues = [NSMutableSet set];
    float originalValue = param.value;

    int maxTests;
    if (param.unit == kAudioUnitParameterUnit_Indexed) {
        // For indexed parameters, test the full range (up to a reasonable limit)
        maxTests = MIN(1000, (int)(param.maxValue - param.minValue + 1));
    } else {
        // For other parameter types, keep the original 10-test limit
        maxTests = MIN(10, (int)(param.maxValue - param.minValue + 1));
    }

    for (int i = 0; i < maxTests; i++) {
        float testValue = param.minValue + i;
        if (testValue > param.maxValue) break;

        param.value = testValue;
        usleep(5000); // 5ms delay

        NSString *stringRep = [param stringFromValue:&testValue];
        if (stringRep && stringRep.length > 0) {
            // Check if this is a meaningful string (not just the numeric value)
            NSString *numericString = [NSString stringWithFormat:@"%.0f", testValue];
            if (![stringRep isEqualToString:numericString]) {
                [discoveredValues addObject:stringRep];
            }
        }
    }

    param.value = originalValue; // Restore original value

    NSArray *result = discoveredValues.count > 0 ? [discoveredValues.allObjects sortedArrayUsingSelector:@selector(compare:)] : nil;
    return result;
}

- (BOOL)isPresetParameter:(AUParameter *)param audioUnit:(AUAudioUnit *)audioUnit {
    if (param.unit != kAudioUnitParameterUnit_Indexed) return NO;

    NSString *lowerName = [param.displayName lowercaseString];
    NSArray *presetKeywords = @[@"preset", @"patch", @"sound", @"bank", @"program", @"model", @"amp", @"cab", @"scene"];

    for (NSString *keyword in presetKeywords) {
        if ([lowerName containsString:keyword]) {
            return YES;
        }
    }
    return NO;
}

- (void)initializePluginAudioUnit:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock {
    // Set a realistic buffer size
    audioUnit.maximumFramesToRender = 512;

    // Load a default preset if available to trigger full initialization
    if (audioUnit.factoryPresets.count > 0) {
        audioUnit.currentPreset = audioUnit.factoryPresets.firstObject;

        // Wait for preset to load
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(g_preset_loading_timeout * NSEC_PER_SEC)), dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^{
            [self simulateAudioProcessing:audioUnit completion:completionBlock];
        });
    } else {
        [self simulateAudioProcessing:audioUnit completion:completionBlock];
    }
}

- (void)simulateAudioProcessing:(AUAudioUnit *)audioUnit completion:(void(^)(void))completionBlock {
    AVAudioFormat *format = [[AVAudioFormat alloc] initStandardFormatWithSampleRate:44100.0 channels:2];

    if (audioUnit.outputBusses.count > 0) {
        @try {
            // Create silent audio buffers
            AVAudioPCMBuffer *inputBuffer = [[AVAudioPCMBuffer alloc] initWithPCMFormat:format frameCapacity:512];
            if (inputBuffer) {
                inputBuffer.frameLength = 512;

                // Zero out the buffer (silent audio)
                if (inputBuffer.floatChannelData[0]) {
                    memset(inputBuffer.floatChannelData[0], 0, 512 * sizeof(float));
                }
                if (inputBuffer.format.channelCount > 1 && inputBuffer.floatChannelData[1]) {
                    memset(inputBuffer.floatChannelData[1], 0, 512 * sizeof(float));
                }

                // Try to get and call the render block
                AURenderBlock renderBlock = audioUnit.renderBlock;
                if (renderBlock) {
                    // Create output buffer list
                    AudioBufferList *outputBufferList = (AudioBufferList *)calloc(1, sizeof(AudioBufferList) + sizeof(AudioBuffer));
                    outputBufferList->mNumberBuffers = 1;
                    outputBufferList->mBuffers[0].mNumberChannels = 2;
                    outputBufferList->mBuffers[0].mDataByteSize = 512 * 2 * sizeof(float);
                    outputBufferList->mBuffers[0].mData = calloc(512 * 2, sizeof(float));

                    AudioTimeStamp timeStamp = {0};
                    timeStamp.mSampleTime = 0;
                    timeStamp.mFlags = kAudioTimeStampSampleTimeValid;

                    AudioUnitRenderActionFlags flags = 0;
                    OSStatus status = renderBlock(&flags, &timeStamp, 512, 0, outputBufferList, nil);

                    free(outputBufferList->mBuffers[0].mData);
                    free(outputBufferList);
                }
            }
        } @catch (NSException *exception) {
            // Silent error handling
        }
    }

    // Give the plugin time to process and update parameters
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(g_process_update_timeout * NSEC_PER_SEC)), dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^{
        completionBlock();
    });
}

- (void)extractIndexedParameterInfo:(AUParameter *)param paramData:(NSMutableDictionary *)paramData audioUnit:(AUAudioUnit *)audioUnit {
    if (param.unit != kAudioUnitParameterUnit_Indexed) return;

    NSArray<NSString *> *indexedValues = nil;
    NSString *source = nil;

    // Method 1: Standard valueStrings property
    indexedValues = param.valueStrings;
    if (indexedValues && indexedValues.count > 0) {
        source = @"valueStrings";
    }

    // Method 2: Force parameter initialization
    if (!indexedValues) {
        [self forceParameterInitialization:param];

        // Check again after forcing initialization
        indexedValues = param.valueStrings;
        if (indexedValues && indexedValues.count > 0) {
            source = @"valueStrings_after_init";
        }
    }

    // Method 3: Try reflection for private properties
    if (!indexedValues) {
        indexedValues = [self getIndexedValuesUsingReflection:param];
        if (indexedValues) {
            source = @"reflection";
        }
    }

    // Method 4: Check if it's a preset parameter
    if (!indexedValues && [self isPresetParameter:param audioUnit:audioUnit]) {
        NSArray *factoryPresets = audioUnit.factoryPresets;
        if (factoryPresets.count > 0) {
            NSMutableArray *presetNames = [NSMutableArray array];
            for (AUAudioUnitPreset *preset in factoryPresets) {
                [presetNames addObject:preset.name];
            }
            indexedValues = presetNames;
            source = @"factoryPresets";
        }
    }

    // Method 5: Query parameter at different states
    if (!indexedValues) {
        indexedValues = [self queryParameterAtDifferentStates:param audioUnit:audioUnit];
        if (indexedValues) {
            source = @"stringFromValue";
        }
    }

    // Method 6: Generate fallback values based on range
    if (!indexedValues) {
        int minVal = (int)param.minValue;
        int maxVal = (int)param.maxValue;

        if (maxVal - minVal < 100 && maxVal - minVal >= 0) { // Reasonable range
            NSMutableArray *fallbackValues = [NSMutableArray array];
            for (int i = minVal; i <= maxVal; i++) {
                [fallbackValues addObject:[NSString stringWithFormat:@"Option %d", i]];
            }
            indexedValues = fallbackValues;
            source = @"generated_fallback";
        }
    }

    // Store results in the parameter data for JSON output
    if (indexedValues && indexedValues.count > 0) {
        [paramData setObject:indexedValues forKey:@"indexedValues"];
        [paramData setObject:source forKey:@"indexedValuesSource"];
        VERBOSE_LOG("    ✓ %s: extracted %lu values using %s\n",
                [param.displayName UTF8String],
                (unsigned long)indexedValues.count,
                [source UTF8String]);
    } else {
        // Store range information for manual mapping later
        [paramData setObject:[NSNumber numberWithInt:(int)param.minValue] forKey:@"indexedMinValue"];
        [paramData setObject:[NSNumber numberWithInt:(int)param.maxValue] forKey:@"indexedMaxValue"];
        [paramData setObject:@"none_found" forKey:@"indexedValuesSource"];
        VERBOSE_LOG("    ✗ %s: no indexed values found (range %.0f-%.0f)\n",
                [param.displayName UTF8String], param.minValue, param.maxValue);
    }
}

- (void)processParametersForAudioUnit:(AUAudioUnit *)audioUnit withName:(NSString *)auName auParameters:(NSMutableArray *)auParameters {
    AUParameterTree *parameterTree = audioUnit.parameterTree;
    if (!parameterTree) {
        VERBOSE_LOG("  ✗ No parameter tree available\n");
        return;
    }

    NSArray *allParameters = parameterTree.allParameters;
    
    // Early skip optimization: if no parameters, don't waste time
    if (allParameters.count == 0) {
        VERBOSE_LOG("  ⏭️  Skipping - no parameters\n");
        return;
    }

    // Count indexed parameters first
    NSUInteger indexedCount = 0;
    for (AUParameter *param in allParameters) {
        if (param.unit == kAudioUnitParameterUnit_Indexed) {
            indexedCount++;
        }
    }

    VERBOSE_LOG("  Processing %lu parameters (%lu indexed)\n",
            (unsigned long)allParameters.count, indexedCount);

    // Process all parameters (for JSON output like your working version)
    for (AUParameter *param in allParameters) {
        BOOL isWritable = (param.flags & kAudioUnitParameterFlag_IsWritable) != 0;
        BOOL canRamp = (param.flags & kAudioUnitParameterFlag_CanRamp) != 0;
        
        // Include ALL parameters - don't filter by writability or ramp capability
        NSMutableDictionary *paramData = [NSMutableDictionary dictionary];
        [paramData setObject:param.displayName forKey:@"displayName"];
        [paramData setObject:param.identifier forKey:@"identifier"];
        [paramData setObject:[NSNumber numberWithUnsignedLongLong:param.address] forKey:@"address"];
        [paramData setObject:[NSNumber numberWithFloat:param.minValue] forKey:@"minValue"];
        [paramData setObject:[NSNumber numberWithFloat:param.maxValue] forKey:@"maxValue"];

        // For now, use current value as default (we can enhance this later)
        // Note: Getting true default values requires more complex AudioUnit introspection
        [paramData setObject:[NSNumber numberWithFloat:param.value] forKey:@"defaultValue"];
        [paramData setObject:[NSNumber numberWithFloat:param.value] forKey:@"currentValue"];
        [paramData setObject:StringFromAudioUnitParameterUnit(param.unit) forKey:@"unit"];
        [paramData setObject:[NSNumber numberWithBool:isWritable] forKey:@"isWritable"];
        [paramData setObject:[NSNumber numberWithBool:canRamp] forKey:@"canRamp"];
        [paramData setObject:[NSNumber numberWithUnsignedInteger:param.flags] forKey:@"rawFlags"];

        // Enhanced indexed parameter processing
        if (param.unit == kAudioUnitParameterUnit_Indexed) {
            [self extractIndexedParameterInfo:param paramData:paramData audioUnit:audioUnit];
        }

        [auParameters addObject:paramData]; // Add parameter to the AU's parameter array
    }
}

@end

/**
 * IntrospectAudioUnits
 *
 * Omnipotent Audio Unit introspection entry point (called from Go via cgo).
 *
 * Arguments:
 *  - type: 4-char code as C string (e.g., "aufx", "aumu"). NULL => any.
 *  - subtype: 4-char code as C string. NULL => any.
 *  - manufacturerID: 4-char code as C string (e.g., "appl"). NULL => any.
 *  - name: Canonical AudioComponent name (UTF-8). If NULL or empty, runs in
 *          suite mode and returns all plugins that match the triplet
 *          (type, subtype, manufacturerID). If non-empty, runs in single mode
 *          and returns only the plugin whose AudioComponentCopyName exactly
 *          matches the provided name. If no such plugin exists, returns a JSON
 *          error with errorCode = -3.
 *
 * Behavior:
 *  - Enumerates AudioComponents via AudioComponentFindNext using the triplet.
 *  - Early name filter (single mode) avoids heavy instantiation when names
 *    don't match.
 *  - Each matching component is instantiated out-of-process and inspected for
 *    parameters, using a dispatch group for concurrency.
 *  - A global timeout (g_total_timeout seconds) bounds the wait; on timeout
 *    partial results are returned with timedOut = true.
 *
 * Returns:
 *  - success envelope (pretty-printed):
 *      {
 *        "success": true,
 *        "plugins": [ { name, manufacturerID, type, subtype, parameters: [...] }, ...],
 *        "pluginCount": <int>,
 *        "totalPluginsScanned": <int>,
 *        "timedOut": <bool>
 *      }
 *  - on error (e.g., not found in single mode, JSON serialization failure):
 *      {
 *        "success": false,
 *        "error": <string>,
 *        "errorCode": <int>,
 *        "plugins": []
 *      }
 *
 * Ownership:
 *  - The caller owns the returned heap-allocated C string and must free() it.
 */
char *IntrospectAudioUnits(const char *type,
                           const char *subtype,
                           const char *manufacturerID,
                           const char *name) {
    @autoreleasepool {
        // Silent operation by default (follows devices package pattern)
        // Optional canonical name filter (nil/empty => suite mode)
        NSString *requiredName = (name && name[0] != '\0') ? [NSString stringWithUTF8String:name] : nil;

        OSType componentType = type ? FourCharCodeFromString([NSString stringWithUTF8String:type]) : 0;
                OSType componentSubType = subtype ? FourCharCodeFromString([NSString stringWithUTF8String:subtype]) : 0;
                OSType componentManufacturer = manufacturerID ? FourCharCodeFromString([NSString stringWithUTF8String:manufacturerID]) : 0;
                
                AudioComponentDescription searchDescription = {
                    .componentType = componentType,
                    .componentSubType = componentSubType,
                    .componentManufacturer = componentManufacturer,
                    .componentFlags = 0,
                    .componentFlagsMask = 0
                };

        AudioComponent currentComponent = NULL;
        __block int count = 0;

        // Master array to hold all AU dictionaries
        __block NSMutableArray *allAudioUnitsData = [NSMutableArray array];

        AudioUnitInspector *inspector = [[AudioUnitInspector alloc] init];
        dispatch_group_t group = dispatch_group_create();

        do {
            currentComponent = AudioComponentFindNext(currentComponent, &searchDescription);

            if (currentComponent != NULL) {
                // Copy canonical AU name first so we can early-filter by name
                CFStringRef nameCFString = NULL;
                AudioComponentCopyName(currentComponent, &nameCFString);

                AudioComponentDescription componentDesc;
                AudioComponentGetDescription(currentComponent, &componentDesc);

                NSString *auName = (nameCFString != NULL) ? (__bridge NSString *)nameCFString : @"[Unknown Name]";

                // If single-plugin mode with a required name, skip non-matching components early
                if (requiredName && ![auName isEqualToString:requiredName]) {
                    if (nameCFString != NULL) {
                        CFRelease(nameCFString);
                    }
                    continue;
                }

                dispatch_group_enter(group);

                count++;
                VERBOSE_LOG("Found Audio Unit [%d]: %s\n", count, [auName UTF8String]);

                // Create a mutable dictionary for the current Audio Unit's data
                NSMutableDictionary *auData = [NSMutableDictionary dictionary];
                [auData setObject:auName forKey:@"name"];
                [auData setObject:StringFromFourCharCode(componentDesc.componentManufacturer) forKey:@"manufacturerID"];
                [auData setObject:StringFromFourCharCode(componentDesc.componentType) forKey:@"type"];
                [auData setObject:StringFromFourCharCode(componentDesc.componentSubType) forKey:@"subtype"];

                // Array to hold parameters for this AU
                NSMutableArray *auParameters = [NSMutableArray array];
                [auData setObject:auParameters forKey:@"parameters"];

                [AUAudioUnit instantiateWithComponentDescription:componentDesc options:kAudioComponentInstantiation_LoadOutOfProcess completionHandler:^(AUAudioUnit * _Nullable audioUnit, NSError * _Nullable error) {
                    if (audioUnit) {
                        VERBOSE_LOG("  ✓ AudioUnit instantiated successfully\n");

                        // Set up audio format on all busses (from your working version)
                        NSError *busError = nil;
                        AVAudioFormat *renderFormat = [[AVAudioFormat alloc] initStandardFormatWithSampleRate:44100.0 channels:2];
                        if (audioUnit.outputBusses.count > 0 && ![audioUnit.outputBusses[0] setFormat:renderFormat error:&busError]) {
                            VERBOSE_LOG("  ⚠ Could not set render format: %s\n", [busError.localizedDescription UTF8String]);
                        }

                        // Allocate render resources
                        NSError *allocError = nil;
                        if (![audioUnit allocateRenderResourcesAndReturnError:&allocError]) {
                            VERBOSE_LOG("  ⚠ Could not allocate render resources: %s\n", [allocError.localizedDescription UTF8String]);
                        } else {
                            VERBOSE_LOG("  ✓ Render resources allocated\n");
                        }

                        // Enhanced initialization for heavy plugins
                        [inspector initializePluginAudioUnit:audioUnit completion:^{
                            // Check if this plugin has parameters before processing
                            AUParameterTree *parameterTree = audioUnit.parameterTree;
                            NSArray *allParameters = parameterTree ? parameterTree.allParameters : nil;
                            
                            if (!allParameters || allParameters.count == 0) {
                                // Skip plugins with no parameters - they're not useful for live performance control
                                VERBOSE_LOG("  ⏭️  Completed inspection of %s (skipped - no parameters)\n", [auName UTF8String]);
                                
                                // Add to results with empty parameters array (maintains count consistency)
                                @synchronized(allAudioUnitsData) {
                                    [allAudioUnitsData addObject:auData];
                                }
                                if (nameCFString != NULL) {
                                    CFRelease(nameCFString);
                                }
                                dispatch_group_leave(group);
                                return;
                            }
                            
                            // Process parameters and add to auParameters array
                            [inspector processParametersForAudioUnit:audioUnit withName:auName auParameters:auParameters];

                            // Only add plugins that have parameters to the results
                            if (auParameters.count > 0) {
                                VERBOSE_LOG("  ✓ Completed inspection of %s (%lu parameters)\n", [auName UTF8String], (unsigned long)auParameters.count);
                                
                                // Add the collected AU data to the master array
                                @synchronized(allAudioUnitsData) {
                                    [allAudioUnitsData addObject:auData];
                                }
                            } else {
                                VERBOSE_LOG("  ⏭️  Completed inspection of %s (skipped - no usable parameters)\n", [auName UTF8String]);
                            }

                            if (nameCFString != NULL) {
                                CFRelease(nameCFString);
                            }
                            dispatch_group_leave(group);
                        }];

                    } else {
                        // Log errors, but don't add failed instantiations to results (they're not useful)
                        VERBOSE_LOG("  ✗ Failed to instantiate: %s\n", [error.localizedDescription UTF8String]);

                        if (nameCFString != NULL) {
                            CFRelease(nameCFString);
                        }
                        dispatch_group_leave(group);
                    }
                }];
            }
        } while (currentComponent != NULL);

        // Wait with timeout to prevent hanging during development
        
        // Wait with a 30-second timeout to prevent hanging during development
        dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, (int64_t)(g_total_timeout * NSEC_PER_SEC));
        long result = dispatch_group_wait(group, timeout);
        
        if (result != 0) {
            // Timeout occurred - return partial results silently
        }

        NSUInteger usablePlugins = allAudioUnitsData.count;

        // If a specific name was requested and nothing matched, return an error JSON
        if (requiredName && usablePlugins == 0) {
            NSDictionary *errorResult = @{
                @"success": @NO,
                @"error": @"plugin not found for given type:subtype:manufacturerID:name",
                @"errorCode": @(-3),
                @"plugins": @[]
            };
            NSData *jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
            NSString *resultStr = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            return strdup([resultStr UTF8String]);
        }

        // Convert the collected data to JSON and wrap in success result
        NSDictionary *successResult = @{
            @"success": [NSNumber numberWithBool:YES],
            @"plugins": allAudioUnitsData,
            @"pluginCount": @(usablePlugins),
            @"totalPluginsScanned": @(count),
            @"timedOut": [NSNumber numberWithBool:(result != 0)]
        };
        
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:successResult
                                                           options:NSJSONWritingPrettyPrinted
                                                             error:&jsonError];

        if (jsonData && !jsonError) {
            NSString *jsonString = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            return strdup([jsonString UTF8String]);
        } else {
            // Return error result on JSON serialization failure
            NSDictionary *errorResult = @{
                @"success": [NSNumber numberWithBool:NO],
                @"error": @"JSON serialization failed",
                @"errorCode": @(-2),
                @"plugins": @[]
            };
            jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
            NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            return strdup([result UTF8String]);
        }
    }
}

// Quick scan function - just enumerate AudioComponents without instantiation
char *QuickScanAudioUnits(void) {
    @autoreleasepool {
        // Silent operation by default (follows devices package pattern)
        
        AudioComponentDescription searchDescription = {
            .componentType = 0,          // 0 = scan all types
            .componentSubType = 0,       // 0 = scan all subtypes
            .componentManufacturer = 0,  // 0 = scan all manufacturers
            .componentFlags = 0,
            .componentFlagsMask = 0
        };
        
        AudioComponent currentComponent = NULL;
        int count = 0;
        
        // Array to hold basic plugin info (no parameters)
        NSMutableArray *pluginList = [NSMutableArray array];
        
        do {
            currentComponent = AudioComponentFindNext(currentComponent, &searchDescription);
            
            if (currentComponent != NULL) {
                CFStringRef nameCFString = NULL;
                AudioComponentCopyName(currentComponent, &nameCFString);
                
                AudioComponentDescription componentDesc;
                AudioComponentGetDescription(currentComponent, &componentDesc);
                
                NSString *auName = (nameCFString != NULL) ? (__bridge NSString *)nameCFString : @"[Unknown Name]";
                
                count++;
                
                // Add human-readable category for UI filtering
                NSString *category = @"Other";
                switch (componentDesc.componentType) {
                    case kAudioUnitType_Effect:
                        category = @"Effect";
                        break;
                    case kAudioUnitType_MusicEffect:
                        category = @"Music Effect";
                        break;
                    case kAudioUnitType_MusicDevice:
                        category = @"Instrument";
                        break;
                    case kAudioUnitType_Generator:
                        category = @"Generator";
                        break;
                    case kAudioUnitType_Mixer:
                        category = @"Mixer";
                        break;
                    case kAudioUnitType_Panner:
                        category = @"Panner";
                        break;
                    case kAudioUnitType_Output:
                        category = @"Output";
                        break;
                    case kAudioUnitType_FormatConverter:
                        category = @"Format Converter";
                        break;
                    default:
                        category = @"Other";
                        break;
                }
                
                // Create basic plugin info dictionary (NO parameters)
                NSDictionary *pluginInfo = @{
                    @"name": auName,
                    @"manufacturerID": StringFromFourCharCode(componentDesc.componentManufacturer),
                    @"type": StringFromFourCharCode(componentDesc.componentType),
                    @"subtype": StringFromFourCharCode(componentDesc.componentSubType),
                    @"category": category
                };
                
                [pluginList addObject:pluginInfo];
                
                if (nameCFString) {
                    CFRelease(nameCFString);
                }
            }
        } while (currentComponent != NULL);
        
        // Return success result with plugin list (like devices pattern)
        NSDictionary *successResult = @{
            @"success": [NSNumber numberWithBool:YES],
            @"plugins": pluginList,
            @"pluginCount": @([pluginList count]),
            @"totalPluginsScanned": @(count)
        };
        
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:successResult options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSDictionary *errorResult = @{
                @"success": [NSNumber numberWithBool:NO],
                @"error": @"JSON serialization failed",
                @"errorCode": @(-2),
                @"plugins": @[]
            };
            jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([result UTF8String]);
    }
}
