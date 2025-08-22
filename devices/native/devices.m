//
//  devices.m
//  Silent device enumeration library
//

#import <Foundation/Foundation.h>
#import <CoreAudio/CoreAudio.h>
#import <CoreMIDI/CoreMIDI.h>

// Safe property collection functions for real data integration
NSString* getDeviceManufacturer(MIDIDeviceRef device) {
    CFStringRef manufacturer;
    OSStatus status = MIDIObjectGetStringProperty(device, kMIDIPropertyManufacturer, &manufacturer);
    if (status == noErr && manufacturer) {
        NSString *result = [(__bridge NSString *)manufacturer copy];
        CFRelease(manufacturer);
        return result;
    }
    return @"Unknown";
}

NSString* getDeviceModel(MIDIDeviceRef device) {
    CFStringRef model;
    OSStatus status = MIDIObjectGetStringProperty(device, kMIDIPropertyModel, &model);
    if (status == noErr && model) {
        NSString *result = [(__bridge NSString *)model copy];
        CFRelease(model);
        return result;
    }
    return @"Unknown";
}

NSString* getEntityName(MIDIEntityRef entity) {
    CFStringRef entityName;
    OSStatus status = MIDIObjectGetStringProperty(entity, kMIDIPropertyName, &entityName);
    if (status == noErr && entityName) {
        NSString *result = [(__bridge NSString *)entityName copy];
        CFRelease(entityName);
        return result;
    }
    return @"Unknown";
}

// Level 3 property collection functions for real data integration
NSString* getEndpointDisplayName(MIDIEndpointRef endpoint) {
    CFStringRef displayName;
    OSStatus status = MIDIObjectGetStringProperty(endpoint, kMIDIPropertyDisplayName, &displayName);
    if (status == noErr && displayName) {
        NSString *result = [(__bridge NSString *)displayName copy];
        CFRelease(displayName);
        return result;
    }
    return @"Unknown";
}

NSNumber* getEndpointSysExSpeed(MIDIEndpointRef endpoint) {
    SInt32 sysExSpeed;
    OSStatus status = MIDIObjectGetIntegerProperty(endpoint, kMIDIPropertyMaxSysExSpeed, &sysExSpeed);
    if (status == noErr) {
        return @(sysExSpeed);
    }
    return @(0);
}
int countMIDIDevices(void) {
    @autoreleasepool {
        ItemCount deviceCount = MIDIGetNumberOfDevices();
        return (int)deviceCount;
    }
}

char* getMIDIDevices(void) {
    @autoreleasepool {
        // Get MIDI device count
        ItemCount deviceCount = MIDIGetNumberOfDevices();
        
        // Replace the error return with success return:
        if (deviceCount == 0) {
            NSDictionary *successResult = @{
                @"success": @YES,
                @"devices": @[],
                @"deviceCount": @(0),
                @"totalDevicesScanned": @(0),
                @"message": @"No MIDI devices found"
            };
            NSData *jsonData = [NSJSONSerialization dataWithJSONObject:successResult options:0 error:nil];
            NSString *json = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            return strdup([json UTF8String]);
        }
        
        // Enumerate MIDI devices and collect both input and output endpoints
        NSMutableDictionary *deviceMap = [[NSMutableDictionary alloc] init];
        
        for (ItemCount i = 0; i < deviceCount; i++) {
            MIDIDeviceRef device = MIDIGetDevice(i);
            if (device == 0) {
                continue;
            }
            
            // Get device name
            CFStringRef deviceName;
            OSStatus status = MIDIObjectGetStringProperty(device, kMIDIPropertyName, &deviceName);
            if (status != noErr) {
                continue;
            }
            
            NSString *deviceNameString = (__bridge NSString *)deviceName;
            
            // Collect real device properties
            NSString *realManufacturer = getDeviceManufacturer(device);
            NSString *realModel = getDeviceModel(device);

            // Get device unique ID
            SInt32 uniqueID;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyUniqueID, &uniqueID);
            if (status != noErr) {
                CFRelease(deviceName);
                continue;
            }
            
            // Check if device is online
            SInt32 isOffline;
            status = MIDIObjectGetIntegerProperty(device, kMIDIPropertyOffline, &isOffline);
            BOOL online = (status != noErr) ? YES : !isOffline;
            
            NSString *deviceUID = [NSString stringWithFormat:@"midi_%d", uniqueID];
            
            // Get entities and collect all endpoints
            ItemCount entityCount = MIDIDeviceGetNumberOfEntities(device);
            
            for (ItemCount j = 0; j < entityCount; j++) {
                MIDIEntityRef entity = MIDIDeviceGetEntity(device, j);
                if (entity == 0) continue;
                
                // Collect real entity name
                NSString *realEntityName = getEntityName(entity);
                
                // Collect INPUT endpoints (sources)
                ItemCount sourceCount = MIDIEntityGetNumberOfSources(entity);
                
                for (ItemCount k = 0; k < sourceCount; k++) {
                    MIDIEndpointRef endpoint = MIDIEntityGetSource(entity, k);
                    if (endpoint == 0) continue;
                    
                    // Get endpoint name
                    CFStringRef endpointName;
                    status = MIDIObjectGetStringProperty(endpoint, kMIDIPropertyName, &endpointName);
                    NSString *finalName = endpointName ? (__bridge NSString *)endpointName : deviceNameString;
                    
                    // Create or update unified device entry
                    NSString *endpointUID = [NSString stringWithFormat:@"%@_%@", deviceUID, finalName];
                    NSMutableDictionary *deviceInfo = deviceMap[endpointUID];
                    if (!deviceInfo) {
                        // Collect real endpoint capabilities
                        NSString *realDisplayName = getEndpointDisplayName(endpoint);
                        NSNumber *realSysExSpeed = getEndpointSysExSpeed(endpoint);
                        
                        deviceInfo = [@{
                            @"name": finalName,
                            @"uid": endpointUID,
                            @"deviceName": deviceNameString,
                            @"manufacturer": realManufacturer,
                            @"model": realModel,
                            @"entityName": realEntityName,
                            @"displayName": realDisplayName,
                            @"sysExSpeed": realSysExSpeed,
                            @"isOnline": @(online),
                            @"isInput": @NO,
                            @"isOutput": @NO,
                            @"inputEndpointId": @(0),
                            @"outputEndpointId": @(0)
                        } mutableCopy];
                        deviceMap[endpointUID] = deviceInfo;
                    }
                    
                    // Mark as input and store endpoint ID
                    deviceInfo[@"isInput"] = @YES;
                    deviceInfo[@"inputEndpointId"] = @(endpoint);
                    
                    if (endpointName) CFRelease(endpointName);
                }
                
                // Collect OUTPUT endpoints (destinations)
                ItemCount destCount = MIDIEntityGetNumberOfDestinations(entity);
                
                for (ItemCount k = 0; k < destCount; k++) {
                    MIDIEndpointRef endpoint = MIDIEntityGetDestination(entity, k);
                    if (endpoint == 0) continue;
                    
                    // Get endpoint name
                    CFStringRef endpointName;
                    status = MIDIObjectGetStringProperty(endpoint, kMIDIPropertyName, &endpointName);
                    NSString *finalName = endpointName ? (__bridge NSString *)endpointName : deviceNameString;
                    
                    // Create or update unified device entry
                    NSString *endpointUID = [NSString stringWithFormat:@"%@_%@", deviceUID, finalName];
                    NSMutableDictionary *deviceInfo = deviceMap[endpointUID];
                    if (!deviceInfo) {
                        // Collect real endpoint capabilities
                        NSString *realDisplayName = getEndpointDisplayName(endpoint);
                        NSNumber *realSysExSpeed = getEndpointSysExSpeed(endpoint);
                        
                        deviceInfo = [@{
                            @"name": finalName,
                            @"uid": endpointUID,
                            @"deviceName": deviceNameString,
                            @"manufacturer": realManufacturer,
                            @"model": realModel,
                            @"entityName": realEntityName,
                            @"displayName": realDisplayName,
                            @"sysExSpeed": realSysExSpeed,
                            @"isOnline": @(online),
                            @"isInput": @NO,
                            @"isOutput": @NO,
                            @"inputEndpointId": @(0),
                            @"outputEndpointId": @(0)
                        } mutableCopy];
                        deviceMap[endpointUID] = deviceInfo;
                    }
                    
                    // Mark as output and store endpoint ID
                    deviceInfo[@"isOutput"] = @YES;
                    deviceInfo[@"outputEndpointId"] = @(endpoint);
                    
                    if (endpointName) CFRelease(endpointName);
                }
            }
            
            CFRelease(deviceName);
        }
        
        // Convert dictionary to array
        NSMutableArray *jsonDevices = [[NSMutableArray alloc] init];
        for (NSMutableDictionary *deviceInfo in [deviceMap allValues]) {
            [jsonDevices addObject:deviceInfo];
        }
        
        // Return success result with unified MIDI devices
        NSDictionary *successResult = @{
            @"success": @YES,
            @"devices": jsonDevices,
            @"deviceCount": @([jsonDevices count]),
            @"totalDevicesScanned": @(deviceCount)
        };
        
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:successResult options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSDictionary *errorResult = @{
                @"success": @NO,
                @"error": @"JSON serialization failed",
                @"errorCode": @(-2),
                @"devices": @[]
            };
            jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([result UTF8String]);
    }
}

int countAudioDevices(void) {
    AudioObjectPropertyAddress addr = {kAudioHardwarePropertyDevices, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain};
    UInt32 size = 0;
    return (AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &size) == noErr) ? (int)(size / sizeof(AudioDeviceID)) : -1;
}

// Unified device enumeration - gets all devices with both input and output capabilities
char* getAudioDevices(void) {
    @autoreleasepool {
        // Get device count - step 1
        AudioObjectPropertyAddress propertyAddress = {
            kAudioHardwarePropertyDevices,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        
        UInt32 dataSize = 0;
        OSStatus status = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize);
        
        if (status != noErr) {
            NSDictionary *errorResult = @{
                @"success": @NO,
                @"error": @"Failed to get device count",
                @"errorCode": @(status),
                @"devices": @[]
            };
            NSData *jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
            NSString *json = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            return strdup([json UTF8String]);
        }
        
        UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
        
        // Get actual device IDs - step 2
        AudioDeviceID *deviceIDs = (AudioDeviceID *)malloc(dataSize);
        if (!deviceIDs) {
            NSDictionary *errorResult = @{
                @"success": @NO,
                @"error": @"Failed to allocate memory for device IDs",
                @"errorCode": @(-1),
                @"devices": @[]
            };
            NSData *jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
            NSString *json = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            return strdup([json UTF8String]);
        }
        
        status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, deviceIDs);
        
        if (status != noErr) {
            free(deviceIDs);
            NSDictionary *errorResult = @{
                @"success": @NO,
                @"error": @"Failed to get device IDs",
                @"errorCode": @(status),
                @"devices": @[]
            };
            NSData *jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
            NSString *json = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
            return strdup([json UTF8String]);
        }
        
        // Get default input and output device IDs for comparison
        AudioDeviceID defaultInputDevice = kAudioDeviceUnknown;
        AudioDeviceID defaultOutputDevice = kAudioDeviceUnknown;
        
        propertyAddress.mSelector = kAudioHardwarePropertyDefaultInputDevice;
        dataSize = sizeof(AudioDeviceID);
        AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, &defaultInputDevice);
        
        propertyAddress.mSelector = kAudioHardwarePropertyDefaultOutputDevice;
        dataSize = sizeof(AudioDeviceID);
        AudioObjectGetPropertyData(kAudioObjectSystemObject, &propertyAddress, 0, NULL, &dataSize, &defaultOutputDevice);
        
        // Step 3 - Enumerate all devices and get both input/output capabilities
        NSMutableArray *allDevices = [[NSMutableArray alloc] init];
        
        for (UInt32 i = 0; i < deviceCount; i++) {
            AudioDeviceID deviceID = deviceIDs[i];
            
            // Get INPUT channel count
            propertyAddress.mSelector = kAudioDevicePropertyStreamConfiguration;
            propertyAddress.mScope = kAudioDevicePropertyScopeInput;
            
            UInt32 inputChannels = 0;
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr) {
                AudioBufferList *bufferList = (AudioBufferList *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
                
                if (status == noErr) {
                    for (UInt32 j = 0; j < bufferList->mNumberBuffers; j++) {
                        inputChannels += bufferList->mBuffers[j].mNumberChannels;
                    }
                }
                free(bufferList);
            }
            
            // Get OUTPUT channel count
            propertyAddress.mSelector = kAudioDevicePropertyStreamConfiguration;
            propertyAddress.mScope = kAudioDevicePropertyScopeOutput;
            
            UInt32 outputChannels = 0;
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr) {
                AudioBufferList *bufferList = (AudioBufferList *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, bufferList);
                
                if (status == noErr) {
                    for (UInt32 j = 0; j < bufferList->mNumberBuffers; j++) {
                        outputChannels += bufferList->mBuffers[j].mNumberChannels;
                    }
                }
                free(bufferList);
            }
            
            // Skip devices with no input or output capabilities
            if (inputChannels == 0 && outputChannels == 0) {
                continue;
            }
            
            // Get device name
            propertyAddress.mSelector = kAudioDevicePropertyDeviceName;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            dataSize = 256;
            char deviceName[256];
            
            NSString *realDeviceName = @"Unknown Device";
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, deviceName);
            if (status == noErr) {
                realDeviceName = [NSString stringWithUTF8String:deviceName];
            }
            
            // Get device UID
            propertyAddress.mSelector = kAudioDevicePropertyDeviceUID;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            CFStringRef deviceUID = NULL;
            dataSize = sizeof(CFStringRef);
            
            NSString *realDeviceUID = [NSString stringWithFormat:@"device_%u", (unsigned int)deviceID];
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &deviceUID);
            if (status == noErr && deviceUID != NULL) {
                realDeviceUID = (__bridge NSString *)deviceUID;
            }
            
            // Get supported sample rates - NO FALLBACKS
            NSMutableArray *sampleRates = [[NSMutableArray alloc] init];
            propertyAddress.mSelector = kAudioDevicePropertyAvailableNominalSampleRates;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr && dataSize > 0) {
                AudioValueRange *sampleRateRanges = (AudioValueRange *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, sampleRateRanges);
                
                if (status == noErr) {
                    UInt32 rangeCount = dataSize / sizeof(AudioValueRange);
                    for (UInt32 r = 0; r < rangeCount; r++) {
                        double minRate = sampleRateRanges[r].mMinimum;
                        double maxRate = sampleRateRanges[r].mMaximum;
                        
                        // Add common sample rates within this range
                        double commonRates[] = {44100, 48000, 88200, 96000, 176400, 192000};
                        for (int cr = 0; cr < 6; cr++) {
                            if (commonRates[cr] >= minRate && commonRates[cr] <= maxRate) {
                                [sampleRates addObject:@(commonRates[cr])];
                            }
                        }
                    }
                }
                free(sampleRateRanges);
            }
            
            // If no sample rates found via ranges, try getting current nominal rate
            if ([sampleRates count] == 0) {
                propertyAddress.mSelector = kAudioDevicePropertyNominalSampleRate;
                propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
                Float64 currentRate = 0;
                dataSize = sizeof(Float64);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &currentRate);
                if (status == noErr && currentRate > 0) {
                    [sampleRates addObject:@(currentRate)];
                }
            }
            
            // If we still couldn't get any sample rates, skip this device
            if ([sampleRates count] == 0) {
                if (deviceUID != NULL) CFRelease(deviceUID);
                continue;
            }
            
            // Get supported bit depths - NO FALLBACKS
            NSMutableArray *bitDepths = [[NSMutableArray alloc] init];
            AudioObjectPropertyScope depthScope = (inputChannels > 0) ? kAudioDevicePropertyScopeInput : kAudioDevicePropertyScopeOutput;
            
            propertyAddress.mSelector = kAudioDevicePropertyStreamFormats;
            propertyAddress.mScope = depthScope;
            
            status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
            if (status == noErr && dataSize > 0) {
                AudioStreamBasicDescription *formats = (AudioStreamBasicDescription *)malloc(dataSize);
                status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, formats);
                
                if (status == noErr) {
                    UInt32 formatCount = dataSize / sizeof(AudioStreamBasicDescription);
                    NSMutableSet *uniqueBitDepths = [[NSMutableSet alloc] init];
                    
                    for (UInt32 f = 0; f < formatCount; f++) {
                        UInt32 bitsPerChannel = formats[f].mBitsPerChannel;
                        if (bitsPerChannel > 0) {
                            [uniqueBitDepths addObject:@(bitsPerChannel)];
                        }
                    }
                    [bitDepths addObjectsFromArray:[uniqueBitDepths allObjects]];
                }
                free(formats);
            }
            
            // If no bit depths found, try the other scope
            if ([bitDepths count] == 0 && inputChannels > 0 && outputChannels > 0) {
                AudioObjectPropertyScope altScope = (depthScope == kAudioDevicePropertyScopeInput) ? 
                    kAudioDevicePropertyScopeOutput : kAudioDevicePropertyScopeInput;
                
                propertyAddress.mScope = altScope;
                status = AudioObjectGetPropertyDataSize(deviceID, &propertyAddress, 0, NULL, &dataSize);
                if (status == noErr && dataSize > 0) {
                    AudioStreamBasicDescription *formats = (AudioStreamBasicDescription *)malloc(dataSize);
                    status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, formats);
                    
                    if (status == noErr) {
                        UInt32 formatCount = dataSize / sizeof(AudioStreamBasicDescription);
                        NSMutableSet *uniqueBitDepths = [[NSMutableSet alloc] init];
                        
                        for (UInt32 f = 0; f < formatCount; f++) {
                            UInt32 bitsPerChannel = formats[f].mBitsPerChannel;
                            if (bitsPerChannel > 0) {
                                [uniqueBitDepths addObject:@(bitsPerChannel)];
                            }
                        }
                        [bitDepths addObjectsFromArray:[uniqueBitDepths allObjects]];
                    }
                    free(formats);
                }
            }
            
            // If we still couldn't get any bit depths, skip this device
            if ([bitDepths count] == 0) {
                if (deviceUID != NULL) CFRelease(deviceUID);
                continue;
            }
            
            // Check if device is online/alive
            BOOL online = YES;
            propertyAddress.mSelector = kAudioDevicePropertyDeviceIsAlive;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            UInt32 isAlive = 1;
            dataSize = sizeof(UInt32);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &isAlive);
            if (status == noErr) {
                online = (isAlive != 0);
            }
            
            // Determine device type and transport type
            NSString *deviceType = @"unknown";
            NSString *transportType = @"unknown";
            
            // Get transport type
            propertyAddress.mSelector = kAudioDevicePropertyTransportType;
            propertyAddress.mScope = kAudioObjectPropertyScopeGlobal;
            UInt32 transport = 0;
            dataSize = sizeof(UInt32);
            status = AudioObjectGetPropertyData(deviceID, &propertyAddress, 0, NULL, &dataSize, &transport);
            if (status == noErr) {
                switch (transport) {
                    case kAudioDeviceTransportTypeBuiltIn:
                        deviceType = @"builtin";
                        transportType = @"builtin";
                        break;
                    case kAudioDeviceTransportTypeUSB:
                        deviceType = @"usb";
                        transportType = @"usb";
                        break;
                    case kAudioDeviceTransportTypeFireWire:
                        deviceType = @"firewire";
                        transportType = @"firewire";
                        break;
                    case kAudioDeviceTransportTypeBluetooth:
                        deviceType = @"bluetooth";
                        transportType = @"bluetooth";
                        break;
                    case kAudioDeviceTransportTypeAggregate:
                        deviceType = @"aggregate";
                        transportType = @"aggregate";
                        break;
                    default:
                        deviceType = @"other";
                        transportType = @"other";
                        break;
                }
            }
            
            // Create device dictionary with unified capabilities
            NSDictionary *deviceJson = @{
                @"name": realDeviceName,
                @"uid": realDeviceUID,
                @"deviceId": @(deviceID),
                @"inputChannelCount": @(inputChannels),
                @"outputChannelCount": @(outputChannels),
                @"isDefaultInput": (deviceID == defaultInputDevice) ? @YES : @NO,
                @"isDefaultOutput": (deviceID == defaultOutputDevice) ? @YES : @NO,
                @"supportedSampleRates": sampleRates,
                @"supportedBitDepths": bitDepths,
                @"deviceType": deviceType,
                @"transportType": transportType,
                @"isOnline": online ? @YES : @NO
            };
            
            [allDevices addObject:deviceJson];
            
            // Clean up CFStringRef if we got it
            if (deviceUID != NULL) {
                CFRelease(deviceUID);
            }
        }
        
        // Clean up
        free(deviceIDs);
        
        // Return success result with unified devices
        NSDictionary *successResult = @{
            @"success": @YES,
            @"devices": allDevices,
            @"deviceCount": @([allDevices count]),
            @"totalDevicesScanned": @(deviceCount)
        };
        
        NSError *jsonError = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:successResult options:0 error:&jsonError];
        
        if (!jsonData || jsonError) {
            NSDictionary *errorResult = @{
                @"success": @NO,
                @"error": @"JSON serialization failed",
                @"errorCode": @(-2),
                @"devices": @[]
            };
            jsonData = [NSJSONSerialization dataWithJSONObject:errorResult options:0 error:nil];
        }
        
        NSString *result = [[NSString alloc] initWithData:jsonData encoding:NSUTF8StringEncoding];
        return strdup([result UTF8String]);
    }
}
