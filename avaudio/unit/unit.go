package unit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AudioUnit.h>

// AVAudioUnitEffect creation from plugin info
void* create_unit_effect(uint32_t type, uint32_t subtype, uint32_t manufacturer) {
    AudioComponentDescription desc = {
        .componentType = type,
        .componentSubType = subtype,
        .componentManufacturer = manufacturer,
        .componentFlags = 0,
        .componentFlagsMask = 0
    };

    AVAudioUnitEffect* effect = [[AVAudioUnitEffect alloc] initWithAudioComponentDescription:desc];

    if (!effect) {
        NSLog(@"Failed to create AVAudioUnitEffect");
        return NULL;
    }

    NSLog(@"Created AVAudioUnitEffect: %@", effect);
    return (__bridge_retained void*)effect;
}

// Release effect
void release_unit_effect(void* effectPtr) {
    if (!effectPtr) return;
    CFBridgingRelease(effectPtr);
}

// Set parameter using address from plugins package
bool set_effect_parameter(void* effectPtr, uint64_t address, float value) {
    if (!effectPtr) return false;

    AVAudioUnitEffect* effect = (__bridge AVAudioUnitEffect*)effectPtr;

    @try {
        AudioUnit audioUnit = effect.audioUnit;
        if (audioUnit == NULL) return false;

        // Use the address from plugins.Parameter.Address
        OSStatus status = AudioUnitSetParameter(audioUnit, (AudioUnitParameterID)address,
                                              kAudioUnitScope_Global, 0, value, 0);
        return status == noErr;
    }
    @catch (NSException* exception) {
        NSLog(@"Exception setting parameter: %@", exception);
        return false;
    }
}

// Get parameter using address from plugins package
float get_effect_parameter(void* effectPtr, uint64_t address) {
    if (!effectPtr) return 0.0f;

    AVAudioUnitEffect* effect = (__bridge AVAudioUnitEffect*)effectPtr;

    @try {
        AudioUnit audioUnit = effect.audioUnit;
        if (audioUnit == NULL) return 0.0f;

        AudioUnitParameterValue value = 0.0f;
        OSStatus status = AudioUnitGetParameter(audioUnit, (AudioUnitParameterID)address,
                                              kAudioUnitScope_Global, 0, &value);
        return status == noErr ? value : 0.0f;
    }
    @catch (NSException* exception) {
        NSLog(@"Exception getting parameter: %@", exception);
        return 0.0f;
    }
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/shaban/macaudio/plugins"
)

// Effect represents an AVAudioUnitEffect instance
type Effect struct {
	ptr    unsafe.Pointer
	plugin plugins.Plugin // Store plugin metadata for parameter access
}

// CreateEffect creates an AVAudioUnitEffect from a plugins.Plugin
func CreateEffect(plugin plugins.Plugin) (*Effect, error) {
	// Convert string IDs to OSTypes
	typeID := stringToOSType(plugin.Type)
	subtypeID := stringToOSType(plugin.Subtype)
	manufacturerID := stringToOSType(plugin.ManufacturerID)

	ptr := C.create_unit_effect(C.uint32_t(typeID), C.uint32_t(subtypeID), C.uint32_t(manufacturerID))
	if ptr == nil {
		return nil, fmt.Errorf("failed to create effect: %s by %s", plugin.Name, plugin.ManufacturerID)
	}

	return &Effect{
		ptr:    ptr,
		plugin: plugin,
	}, nil
}

// Release frees the effect resources
func (e *Effect) Release() {
	if e.ptr != nil {
		C.release_unit_effect(e.ptr)
		e.ptr = nil
	}
}

// SetParameter sets a parameter value using the parameter from plugins introspection
func (e *Effect) SetParameter(param plugins.Parameter, value float32) error {
	if e.ptr == nil {
		return fmt.Errorf("effect has been released")
	}

	success := bool(C.set_effect_parameter(e.ptr, C.uint64_t(param.Address), C.float(value)))
	if !success {
		return fmt.Errorf("failed to set parameter %s (address: %d) to %.3f", param.DisplayName, param.Address, value)
	}

	return nil
}

// GetParameter gets a parameter value using the parameter from plugins introspection
func (e *Effect) GetParameter(param plugins.Parameter) (float32, error) {
	if e.ptr == nil {
		return 0, fmt.Errorf("effect has been released")
	}

	value := float32(C.get_effect_parameter(e.ptr, C.uint64_t(param.Address)))
	return value, nil
}

// GetPlugin returns the plugin metadata for this effect
func (e *Effect) GetPlugin() plugins.Plugin {
	return e.plugin
}

// Ptr returns the unsafe.Pointer for use with engine package
func (e *Effect) Ptr() unsafe.Pointer {
	return e.ptr
}

// stringToOSType converts a 4-character string to OSType (uint32)
func stringToOSType(s string) uint32 {
	if len(s) != 4 {
		return 0
	}
	return uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3])
}
