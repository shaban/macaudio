package node

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#import <AVFoundation/AVFoundation.h>

// Generic bus operations that work on ANY AVAudioNode*
void* audionode_input_format_for_bus(void* nodePtr, int bus) {
    if (!nodePtr) {
        NSLog(@"audionode_input_format_for_bus: nodePtr is NULL");
        return NULL;
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;

    if (bus < 0 || bus >= node.numberOfInputs) {
        NSLog(@"audionode_input_format_for_bus: invalid bus %d (node has %d inputs)", bus, (int)node.numberOfInputs);
        return NULL;
    }

    AVAudioFormat* format = [node inputFormatForBus:bus];
    if (!format) {
        NSLog(@"audionode_input_format_for_bus: no format for bus %d", bus);
        return NULL;
    }

    NSLog(@"Got input format for bus %d: %.0f Hz, %d channels", bus, format.sampleRate, (int)format.channelCount);
    return (__bridge void*)format;
}

void* audionode_output_format_for_bus(void* nodePtr, int bus) {
    if (!nodePtr) {
        NSLog(@"audionode_output_format_for_bus: nodePtr is NULL");
        return NULL;
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;

    if (bus < 0 || bus >= node.numberOfOutputs) {
        NSLog(@"audionode_output_format_for_bus: invalid bus %d (node has %d outputs)", bus, (int)node.numberOfOutputs);
        return NULL;
    }

    AVAudioFormat* format = [node outputFormatForBus:bus];
    if (!format) {
        NSLog(@"audionode_output_format_for_bus: no format for bus %d", bus);
        return NULL;
    }

    NSLog(@"Got output format for bus %d: %.0f Hz, %d channels", bus, format.sampleRate, (int)format.channelCount);
    return (__bridge void*)format;
}

int audionode_number_of_inputs(void* nodePtr) {
    if (!nodePtr) {
        NSLog(@"audionode_number_of_inputs: nodePtr is NULL");
        return 0;
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    int inputs = (int)node.numberOfInputs;
    NSLog(@"Node has %d inputs", inputs);
    return inputs;
}

int audionode_number_of_outputs(void* nodePtr) {
    if (!nodePtr) {
        NSLog(@"audionode_number_of_outputs: nodePtr is NULL");
        return 0;
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    int outputs = (int)node.numberOfOutputs;
    NSLog(@"Node has %d outputs", outputs);
    return outputs;
}

bool audionode_is_installed_on_engine(void* nodePtr) {
    if (!nodePtr) {
        NSLog(@"audionode_is_installed_on_engine: nodePtr is NULL");
        return false;
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    bool installed = node.engine != nil;
    NSLog(@"Node installed on engine: %s", installed ? "YES" : "NO");
    return installed;
}

void audionode_log_info(void* nodePtr) {
    if (!nodePtr) {
        NSLog(@"audionode_log_info: nodePtr is NULL");
        return;
    }

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    NSLog(@"AudioNode Info:");
    NSLog(@"  Class: %@", [node class]);
    NSLog(@"  Inputs: %d", (int)node.numberOfInputs);
    NSLog(@"  Outputs: %d", (int)node.numberOfOutputs);
    NSLog(@"  Engine: %@", node.engine ? @"Connected" : @"Not connected");
    NSLog(@"  Description: %@", node);
}

// AVAudioMixerNode specific functions
void* audiomixer_create(void) {
    AVAudioMixerNode* mixer = [[AVAudioMixerNode alloc] init];
    NSLog(@"Created AVAudioMixerNode: %@", mixer);
    return (__bridge_retained void*)mixer;
}

bool audiomixer_set_volume(void* mixerPtr, float volume, int inputBus) {
    if (!mixerPtr) {
        NSLog(@"audiomixer_set_volume: mixerPtr is NULL");
        return false;
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0 || inputBus >= mixer.numberOfInputs) {
        NSLog(@"audiomixer_set_volume: invalid bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs);
        return false;
    }

    mixer.volume = volume;
    NSLog(@"Set mixer volume to %.2f on bus %d", volume, inputBus);
    return true;
}

bool audiomixer_set_pan(void* mixerPtr, float pan, int inputBus) {
    if (!mixerPtr) {
        NSLog(@"audiomixer_set_pan: mixerPtr is NULL");
        return false;
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0 || inputBus >= mixer.numberOfInputs) {
        NSLog(@"audiomixer_set_pan: invalid bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs);
        return false;
    }

    mixer.pan = pan;
    NSLog(@"Set mixer pan to %.2f on bus %d", pan, inputBus);
    return true;
}

float audiomixer_get_volume(void* mixerPtr, int inputBus) {
    if (!mixerPtr) {
        NSLog(@"audiomixer_get_volume: mixerPtr is NULL");
        return 0.0;
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0 || inputBus >= mixer.numberOfInputs) {
        NSLog(@"audiomixer_get_volume: invalid bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs);
        return 0.0;
    }

    float volume = mixer.volume;
    NSLog(@"Got mixer volume %.2f from bus %d", volume, inputBus);
    return volume;
}

float audiomixer_get_pan(void* mixerPtr, int inputBus) {
    if (!mixerPtr) {
        NSLog(@"audiomixer_get_pan: mixerPtr is NULL");
        return 0.0;
    }

    AVAudioMixerNode* mixer = (__bridge AVAudioMixerNode*)mixerPtr;

    if (inputBus < 0 || inputBus >= mixer.numberOfInputs) {
        NSLog(@"audiomixer_get_pan: invalid bus %d (mixer has %d inputs)", inputBus, (int)mixer.numberOfInputs);
        return 0.0;
    }

    float pan = mixer.pan;
    NSLog(@"Got mixer pan %.2f from bus %d", pan, inputBus);
    return pan;
}

void audiomixer_release(void* mixerPtr) {
    if (!mixerPtr) {
        NSLog(@"audiomixer_release: mixerPtr is NULL");
        return;
    }

    CFBridgingRelease(mixerPtr);
    NSLog(@"Released AVAudioMixerNode");
}
*/
import "C"
import (
	"errors"
	"unsafe"
)

// Shared helper functions that ANY AVAudioNode can use
// These work on the base AVAudioNode functionality that's consistent across all node types

// GetInputFormatForBus returns the input format for the specified bus
// Returns nil if the bus is invalid or no format is set
func GetInputFormatForBus(nodePtr unsafe.Pointer, bus int) unsafe.Pointer {
	if nodePtr == nil {
		return nil
	}
	return unsafe.Pointer(C.audionode_input_format_for_bus(nodePtr, C.int(bus)))
}

// GetOutputFormatForBus returns the output format for the specified bus
// Returns nil if the bus is invalid or no format is set
func GetOutputFormatForBus(nodePtr unsafe.Pointer, bus int) unsafe.Pointer {
	if nodePtr == nil {
		return nil
	}
	return unsafe.Pointer(C.audionode_output_format_for_bus(nodePtr, C.int(bus)))
}

// GetNumberOfInputs returns the number of input buses on the node
func GetNumberOfInputs(nodePtr unsafe.Pointer) int {
	if nodePtr == nil {
		return 0
	}
	return int(C.audionode_number_of_inputs(nodePtr))
}

// GetNumberOfOutputs returns the number of output buses on the node
func GetNumberOfOutputs(nodePtr unsafe.Pointer) int {
	if nodePtr == nil {
		return 0
	}
	return int(C.audionode_number_of_outputs(nodePtr))
}

// IsInstalledOnEngine returns true if the node is currently installed on an AVAudioEngine
func IsInstalledOnEngine(nodePtr unsafe.Pointer) bool {
	if nodePtr == nil {
		return false
	}
	return bool(C.audionode_is_installed_on_engine(nodePtr))
}

// LogInfo logs detailed information about the node for debugging
func LogInfo(nodePtr unsafe.Pointer) {
	if nodePtr == nil {
		return
	}
	C.audionode_log_info(nodePtr)
}

// ValidateBus checks if a bus number is valid for the given direction
func ValidateInputBus(nodePtr unsafe.Pointer, bus int) error {
	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}

	numInputs := GetNumberOfInputs(nodePtr)
	if bus < 0 || bus >= numInputs {
		return errors.New("invalid input bus: node has " + string(rune(numInputs)) + " inputs, requested bus " + string(rune(bus)))
	}

	return nil
}

// ValidateOutputBus checks if an output bus number is valid
func ValidateOutputBus(nodePtr unsafe.Pointer, bus int) error {
	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}

	numOutputs := GetNumberOfOutputs(nodePtr)
	if bus < 0 || bus >= numOutputs {
		return errors.New("invalid output bus: node has " + string(rune(numOutputs)) + " outputs, requested bus " + string(rune(bus)))
	}

	return nil
}

// AVAudioMixerNode functions

// CreateMixer creates a new AVAudioMixerNode
// Returns a pointer to the created mixer node or nil on failure
func CreateMixer() unsafe.Pointer {
	return unsafe.Pointer(C.audiomixer_create())
}

// ReleaseMixer releases the memory for an AVAudioMixerNode
func ReleaseMixer(mixerPtr unsafe.Pointer) {
	if mixerPtr == nil {
		return
	}
	C.audiomixer_release(mixerPtr)
}

// SetMixerVolume sets the volume for the mixer on the specified input bus
// volume should be between 0.0 and 1.0
func SetMixerVolume(mixerPtr unsafe.Pointer, volume float32, inputBus int) error {
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}

	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}

	success := bool(C.audiomixer_set_volume(mixerPtr, C.float(volume), C.int(inputBus)))
	if !success {
		return errors.New("failed to set mixer volume")
	}

	return nil
}

// SetMixerPan sets the pan for the mixer on the specified input bus
// pan should be between -1.0 (left) and 1.0 (right), 0.0 is center
func SetMixerPan(mixerPtr unsafe.Pointer, pan float32, inputBus int) error {
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}

	if pan < -1.0 || pan > 1.0 {
		return errors.New("pan must be between -1.0 and 1.0")
	}

	success := bool(C.audiomixer_set_pan(mixerPtr, C.float(pan), C.int(inputBus)))
	if !success {
		return errors.New("failed to set mixer pan")
	}

	return nil
}

// GetMixerVolume gets the current volume for the mixer on the specified input bus
func GetMixerVolume(mixerPtr unsafe.Pointer, inputBus int) (float32, error) {
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	volume := float32(C.audiomixer_get_volume(mixerPtr, C.int(inputBus)))
	return volume, nil
}

// GetMixerPan gets the current pan for the mixer on the specified input bus
func GetMixerPan(mixerPtr unsafe.Pointer, inputBus int) (float32, error) {
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	pan := float32(C.audiomixer_get_pan(mixerPtr, C.int(inputBus)))
	return pan, nil
}
