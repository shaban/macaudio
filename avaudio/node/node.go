package node

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/node.m"
#include <stdlib.h>

// Function declarations - CGO resolves AudioNodeResult from .m file
AudioNodeResult audionode_input_format_for_bus(void* nodePtr, int bus);
AudioNodeResult audionode_output_format_for_bus(void* nodePtr, int bus);
const char* audionode_get_number_of_inputs(void* nodePtr, int* result);
const char* audionode_get_number_of_outputs(void* nodePtr, int* result);
const char* audionode_is_installed_on_engine(void* nodePtr, bool* result);
const char* audionode_log_info(void* nodePtr);
const char* audionode_release(void* nodePtr);

// Mixer function declarations
AudioNodeResult audiomixer_create(void);
const char* audiomixer_set_volume(void* mixerPtr, float volume, int inputBus);
const char* audiomixer_set_pan(void* mixerPtr, float pan, int inputBus);
const char* audiomixer_get_volume(void* mixerPtr, int inputBus, float* result);
const char* audiomixer_get_pan(void* mixerPtr, int inputBus, float* result);
const char* audiomixer_release(void* mixerPtr);

// Per-connection (source->mixer bus) control
const char* audiomixer_set_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float volume);
const char* audiomixer_get_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result);
const char* audiomixer_set_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float pan);
const char* audiomixer_get_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result);

// Matrix mixer (invert stage)
AudioNodeResult matrixmixer_create(void);
const char* matrixmixer_configure_invert(void* unitPtr);
*/
import "C"
import (
	"errors"
	"unsafe"
)

// Generic AVAudioNode Functions
// These work on the base AVAudioNode functionality that's consistent across all node types

// GetInputFormatForBus returns the input format for the specified bus
func GetInputFormatForBus(nodePtr unsafe.Pointer, bus int) (unsafe.Pointer, error) {
	if nodePtr == nil {
		return nil, errors.New("node pointer is nil")
	}
	
	result := C.audionode_input_format_for_bus(nodePtr, C.int(bus))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	return unsafe.Pointer(result.result), nil
}

// GetOutputFormatForBus returns the output format for the specified bus
func GetOutputFormatForBus(nodePtr unsafe.Pointer, bus int) (unsafe.Pointer, error) {
	if nodePtr == nil {
		return nil, errors.New("node pointer is nil")
	}
	
	result := C.audionode_output_format_for_bus(nodePtr, C.int(bus))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	return unsafe.Pointer(result.result), nil
}

// GetNumberOfInputs returns the number of input buses on the node
func GetNumberOfInputs(nodePtr unsafe.Pointer) (int, error) {
	if nodePtr == nil {
		return 0, errors.New("node pointer is nil")
	}
	
	var result C.int
	errorStr := C.audionode_get_number_of_inputs(nodePtr, &result)
	if errorStr != nil {
		return 0, errors.New(C.GoString(errorStr))
	}
	return int(result), nil
}

// GetNumberOfOutputs returns the number of output buses on the node
func GetNumberOfOutputs(nodePtr unsafe.Pointer) (int, error) {
	if nodePtr == nil {
		return 0, errors.New("node pointer is nil")
	}
	
	var result C.int
	errorStr := C.audionode_get_number_of_outputs(nodePtr, &result)
	if errorStr != nil {
		return 0, errors.New(C.GoString(errorStr))
	}
	return int(result), nil
}

// IsInstalledOnEngine returns true if the node is currently installed on an AVAudioEngine
func IsInstalledOnEngine(nodePtr unsafe.Pointer) (bool, error) {
	if nodePtr == nil {
		return false, errors.New("node pointer is nil")
	}
	
	var result C.bool
	errorStr := C.audionode_is_installed_on_engine(nodePtr, &result)
	if errorStr != nil {
		return false, errors.New(C.GoString(errorStr))
	}
	return bool(result), nil
}

// LogInfo logs detailed information about the node for debugging
func LogInfo(nodePtr unsafe.Pointer) error {
	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}
	
	errorStr := C.audionode_log_info(nodePtr)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// AVAudioMixerNode Functions

// CreateMixer creates a new AVAudioMixerNode
func CreateMixer() (unsafe.Pointer, error) {
	result := C.audiomixer_create()
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	return unsafe.Pointer(result.result), nil
}

// ReleaseMixer releases the memory for an AVAudioMixerNode
func ReleaseMixer(mixerPtr unsafe.Pointer) error {
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}
	
	errorStr := C.audiomixer_release(mixerPtr)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// SetMixerVolume sets the volume for the mixer on the specified input bus
// volume should be between 0.0 and 1.0
func SetMixerVolume(mixerPtr unsafe.Pointer, volume float32, inputBus int) error {
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}

	errorStr := C.audiomixer_set_volume(mixerPtr, C.float(volume), C.int(inputBus))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// SetMixerPan sets the pan for the mixer on the specified input bus
// pan should be between -1.0 (left) and 1.0 (right), 0.0 is center
func SetMixerPan(mixerPtr unsafe.Pointer, pan float32, inputBus int) error {
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}

	errorStr := C.audiomixer_set_pan(mixerPtr, C.float(pan), C.int(inputBus))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// GetMixerVolume gets the current volume for the mixer on the specified input bus
func GetMixerVolume(mixerPtr unsafe.Pointer, inputBus int) (float32, error) {
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	var result C.float
	errorStr := C.audiomixer_get_volume(mixerPtr, C.int(inputBus), &result)
	if errorStr != nil {
		return 0.0, errors.New(C.GoString(errorStr))
	}
	return float32(result), nil
}

// GetMixerPan gets the current pan for the mixer on the specified input bus
func GetMixerPan(mixerPtr unsafe.Pointer, inputBus int) (float32, error) {
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	var result C.float
	errorStr := C.audiomixer_get_pan(mixerPtr, C.int(inputBus), &result)
	if errorStr != nil {
		return 0.0, errors.New(C.GoString(errorStr))
	}
	return float32(result), nil
}

// SetConnectionInputVolume sets the gain for a specific source->mixer input connection.
func SetConnectionInputVolume(sourcePtr, mixerPtr unsafe.Pointer, destBus int, volume float32) error {
	if sourcePtr == nil { return errors.New("source pointer is nil") }
	if mixerPtr == nil { return errors.New("mixer pointer is nil") }
	errStr := C.audiomixer_set_input_volume_for_connection(sourcePtr, mixerPtr, C.int(destBus), C.float(volume))
	if errStr != nil { return errors.New(C.GoString(errStr)) }
	return nil
}

// GetConnectionInputVolume reads the gain for a specific source->mixer input connection.
func GetConnectionInputVolume(sourcePtr, mixerPtr unsafe.Pointer, destBus int) (float32, error) {
	if sourcePtr == nil { return 0, errors.New("source pointer is nil") }
	if mixerPtr == nil { return 0, errors.New("mixer pointer is nil") }
	var result C.float
	errStr := C.audiomixer_get_input_volume_for_connection(sourcePtr, mixerPtr, C.int(destBus), &result)
	if errStr != nil { return 0, errors.New(C.GoString(errStr)) }
	return float32(result), nil
}

// SetConnectionInputPan sets the pan for a specific source->mixer input connection.
func SetConnectionInputPan(sourcePtr, mixerPtr unsafe.Pointer, destBus int, pan float32) error {
	if sourcePtr == nil { return errors.New("source pointer is nil") }
	if mixerPtr == nil { return errors.New("mixer pointer is nil") }
	errStr := C.audiomixer_set_input_pan_for_connection(sourcePtr, mixerPtr, C.int(destBus), C.float(pan))
	if errStr != nil { return errors.New(C.GoString(errStr)) }
	return nil
}

// GetConnectionInputPan reads the pan for a specific source->mixer input connection.
func GetConnectionInputPan(sourcePtr, mixerPtr unsafe.Pointer, destBus int) (float32, error) {
	if sourcePtr == nil { return 0, errors.New("source pointer is nil") }
	if mixerPtr == nil { return 0, errors.New("mixer pointer is nil") }
	var result C.float
	errStr := C.audiomixer_get_input_pan_for_connection(sourcePtr, mixerPtr, C.int(destBus), &result)
	if errStr != nil { return 0, errors.New(C.GoString(errStr)) }
	return float32(result), nil
}

// ReleaseNode provides a generic release hook for AVAudioNode/AVAudioUnit wrappers.
func ReleaseNode(ptr unsafe.Pointer) error {
	if ptr == nil { return nil }
	if errStr := C.audionode_release(ptr); errStr != nil { return errors.New(C.GoString(errStr)) }
	return nil
}

// CreateMatrixMixer returns a new AVAudioUnitMatrixMixer instance as an AVAudioNode pointer.
func CreateMatrixMixer() (unsafe.Pointer, error) {
	res := C.matrixmixer_create()
	if res.error != nil { return nil, errors.New(C.GoString(res.error)) }
	return unsafe.Pointer(res.result), nil
}

// ConfigureMatrixInvert sets diagonal gains to -1.0 for polarity inversion.
func ConfigureMatrixInvert(unitPtr unsafe.Pointer) error {
	if unitPtr == nil { return errors.New("unit pointer is nil") }
	if errStr := C.matrixmixer_configure_invert(unitPtr); errStr != nil { return errors.New(C.GoString(errStr)) }
	return nil
}

// Legacy helper functions for backward compatibility (these now return errors properly)

// ValidateInputBus checks if a bus number is valid for input
func ValidateInputBus(nodePtr unsafe.Pointer, bus int) error {
	numInputs, err := GetNumberOfInputs(nodePtr)
	if err != nil {
		return err
	}

	if bus < 0 || bus >= numInputs {
		return errors.New("invalid input bus: node has " + string(rune(numInputs+'0')) + " inputs, requested bus " + string(rune(bus+'0')))
	}

	return nil
}

// ValidateOutputBus checks if an output bus number is valid
func ValidateOutputBus(nodePtr unsafe.Pointer, bus int) error {
	numOutputs, err := GetNumberOfOutputs(nodePtr)
	if err != nil {
		return err
	}

	if bus < 0 || bus >= numOutputs {
		return errors.New("invalid output bus: node has " + string(rune(numOutputs+'0')) + " outputs, requested bus " + string(rune(bus+'0')))
	}

	return nil
}
