package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -L../../ -lmacaudio -Wl,-rpath,/Users/shaban/Code/macaudio
#include "../../native/macaudio.h"
#include <stdlib.h>

// C function to allocate pointer array
static void** alloc_pointer_array(int count) {
    return (void**)malloc(count * sizeof(void*));
}

// C function to set element in pointer array
static void set_pointer_array_element(void** array, int index, void* ptr) {
    array[index] = ptr;
}

// C function to free pointer array
static void free_pointer_array(void** array) {
    if (array) free(array);
}

// Node introspection function declarations
AudioNodeResult audionode_input_format_for_bus(void* nodePtr, int bus);
AudioNodeResult audionode_output_format_for_bus(void* nodePtr, int bus);
const char* audionode_get_number_of_inputs(void* nodePtr, int* result);
const char* audionode_get_number_of_outputs(void* nodePtr, int* result);
const char* audionode_is_installed_on_engine(void* nodePtr, bool* result);
const char* audionode_log_info(void* nodePtr);
const char* audionode_release(void* nodePtr);

// Enhanced mixer function declarations
const char* audiomixer_set_volume(void* mixerPtr, float volume, int inputBus);
const char* audiomixer_set_pan(void* mixerPtr, float pan, int inputBus);
const char* audiomixer_get_volume(void* mixerPtr, int inputBus, float* result);
const char* audiomixer_get_pan(void* mixerPtr, int inputBus, float* result);

// Per-connection (source->mixer bus) control
const char* audiomixer_set_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float volume);
const char* audiomixer_get_input_volume_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result);
const char* audiomixer_set_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float pan);
const char* audiomixer_get_input_pan_for_connection(void* sourcePtr, void* mixerPtr, int destBus, float* result);

// Smart per-bus controls (tries per-connection first, falls back to global)
const char* audiomixer_set_bus_volume_smart(void* mixerPtr, float volume, int inputBus, void** connectedSources, int sourceCount);
const char* audiomixer_set_bus_pan_smart(void* mixerPtr, float pan, int inputBus, void** connectedSources, int sourceCount);
const char* audiomixer_get_bus_volume_smart(void* mixerPtr, int inputBus, float* result, void** connectedSources, int sourceCount);
const char* audiomixer_get_bus_pan_smart(void* mixerPtr, int inputBus, float* result, void** connectedSources, int sourceCount);
*/
import "C"
import (
	"errors"
	"unsafe"
)

// =============================================================================
// PHASE 1: Node Introspection Functions
// =============================================================================

// GetNodeInputFormat returns the input format for the specified bus as a typed Format object
func (e *Engine) GetNodeInputFormat(nodePtr unsafe.Pointer, bus int) (*Format, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}
	if nodePtr == nil {
		return nil, errors.New("node pointer is nil")
	}

	result := C.audionode_input_format_for_bus(nodePtr, C.int(bus))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	// Wrap the AVAudioFormat pointer in our typed Format struct
	format := &Format{ptr: (*C.AudioFormat)(result.result), engine: e}
	return format, nil
}

// GetNodeOutputFormat returns the output format for the specified bus as a typed Format object
func (e *Engine) GetNodeOutputFormat(nodePtr unsafe.Pointer, bus int) (*Format, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}
	if nodePtr == nil {
		return nil, errors.New("node pointer is nil")
	}

	result := C.audionode_output_format_for_bus(nodePtr, C.int(bus))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	// Wrap the AVAudioFormat pointer in our typed Format struct
	format := &Format{ptr: (*C.AudioFormat)(result.result), engine: e}
	return format, nil
}

// GetNodeInputCount returns the number of input buses on the node
func (e *Engine) GetNodeInputCount(nodePtr unsafe.Pointer) (int, error) {
	if e == nil || e.ptr == nil {
		return 0, errors.New("engine is nil")
	}
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

// GetNodeOutputCount returns the number of output buses on the node
func (e *Engine) GetNodeOutputCount(nodePtr unsafe.Pointer) (int, error) {
	if e == nil || e.ptr == nil {
		return 0, errors.New("engine is nil")
	}
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

// IsNodeAttached returns true if the node is currently installed on an AVAudioEngine
func (e *Engine) IsNodeAttached(nodePtr unsafe.Pointer) (bool, error) {
	if e == nil || e.ptr == nil {
		return false, errors.New("engine is nil")
	}
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

// LogNodeInfo logs detailed information about the node for debugging
func (e *Engine) LogNodeInfo(nodePtr unsafe.Pointer) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}
	if nodePtr == nil {
		return errors.New("node pointer is nil")
	}

	errorStr := C.audionode_log_info(nodePtr)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// ReleaseNode provides a generic release function for AVAudioNode/AVAudioUnit wrappers
func (e *Engine) ReleaseNode(nodePtr unsafe.Pointer) error {
	if nodePtr == nil {
		return nil // Allow releasing nil pointers silently
	}

	errorStr := C.audionode_release(nodePtr)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// ValidateNodeInputBus checks if a bus number is valid for input on the given node
func (e *Engine) ValidateNodeInputBus(nodePtr unsafe.Pointer, bus int) error {
	numInputs, err := e.GetNodeInputCount(nodePtr)
	if err != nil {
		return err
	}

	if bus < 0 || bus >= numInputs {
		return errors.New("invalid input bus: node has " + string(rune(numInputs+'0')) + " inputs, requested bus " + string(rune(bus+'0')))
	}

	return nil
}

// ValidateNodeOutputBus checks if an output bus number is valid on the given node
func (e *Engine) ValidateNodeOutputBus(nodePtr unsafe.Pointer, bus int) error {
	numOutputs, err := e.GetNodeOutputCount(nodePtr)
	if err != nil {
		return err
	}

	if bus < 0 || bus >= numOutputs {
		return errors.New("invalid output bus: node has " + string(rune(numOutputs+'0')) + " outputs, requested bus " + string(rune(bus+'0')))
	}

	return nil
}

// =============================================================================
// PHASE 2: Enhanced Mixer Controls
// =============================================================================

// SetMixerVolumeForBus sets the volume for a specific input bus on the mixer
// Uses connection tracking to provide true per-bus control via AVAudioMixingDestination
func (e *Engine) SetMixerVolumeForBus(mixerPtr unsafe.Pointer, volume float32, inputBus int) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}
	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}

	// Use connection tracking for true per-bus control
	if e.mixerConnections == nil {
		// No connections tracked, fall back to global control
		errorStr := C.audiomixer_set_volume(mixerPtr, C.float(volume), C.int(inputBus))
		if errorStr != nil {
			return errors.New(C.GoString(errorStr))
		}
		return nil
	}

	busMap, exists := e.mixerConnections[mixerPtr]
	if !exists {
		// No connections for this mixer, fall back to global control
		errorStr := C.audiomixer_set_volume(mixerPtr, C.float(volume), C.int(inputBus))
		if errorStr != nil {
			return errors.New(C.GoString(errorStr))
		}
		return nil
	}

	sourcePtr, exists := busMap[inputBus]
	if !exists {
		// No source connected to this bus, fall back to global control
		errorStr := C.audiomixer_set_volume(mixerPtr, C.float(volume), C.int(inputBus))
		if errorStr != nil {
			return errors.New(C.GoString(errorStr))
		}
		return nil
	}

	// Found a connected source - use per-connection control for true per-bus behavior
	errorStr := C.audiomixer_set_input_volume_for_connection(sourcePtr, mixerPtr, C.int(inputBus), C.float(volume))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// GetMixerVolumeForBus gets the volume for a specific input bus on the mixer
// Uses connection tracking to provide true per-bus control via AVAudioMixingDestination
func (e *Engine) GetMixerVolumeForBus(mixerPtr unsafe.Pointer, inputBus int) (float32, error) {
	if e == nil || e.ptr == nil {
		return 0.0, errors.New("engine is nil")
	}
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	// Use connection tracking for true per-bus control
	if e.mixerConnections == nil {
		// No connections tracked, fall back to global control
		var result C.float
		errorStr := C.audiomixer_get_volume(mixerPtr, C.int(inputBus), &result)
		if errorStr != nil {
			return 0.0, errors.New(C.GoString(errorStr))
		}
		return float32(result), nil
	}

	busMap, exists := e.mixerConnections[mixerPtr]
	if !exists {
		// No connections for this mixer, fall back to global control
		var result C.float
		errorStr := C.audiomixer_get_volume(mixerPtr, C.int(inputBus), &result)
		if errorStr != nil {
			return 0.0, errors.New(C.GoString(errorStr))
		}
		return float32(result), nil
	}

	sourcePtr, exists := busMap[inputBus]
	if !exists {
		// No source connected to this bus, fall back to global control
		var result C.float
		errorStr := C.audiomixer_get_volume(mixerPtr, C.int(inputBus), &result)
		if errorStr != nil {
			return 0.0, errors.New(C.GoString(errorStr))
		}
		return float32(result), nil
	}

	// Found a connected source - use per-connection control for true per-bus behavior
	var result C.float
	errorStr := C.audiomixer_get_input_volume_for_connection(sourcePtr, mixerPtr, C.int(inputBus), &result)
	if errorStr != nil {
		return 0.0, errors.New(C.GoString(errorStr))
	}
	return float32(result), nil
}

// SetMixerPanForBus sets the pan for a specific input bus on the mixer
// Uses connection tracking to provide true per-bus control via AVAudioMixingDestination
// pan should be between -1.0 (left) and 1.0 (right), 0.0 is center
func (e *Engine) SetMixerPanForBus(mixerPtr unsafe.Pointer, pan float32, inputBus int) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}
	if pan < -1.0 || pan > 1.0 {
		return errors.New("pan must be between -1.0 (left) and 1.0 (right)")
	}

	// Use connection tracking for true per-bus control
	if e.mixerConnections == nil {
		// No connections tracked, fall back to global control
		errorStr := C.audiomixer_set_pan(mixerPtr, C.float(pan), C.int(inputBus))
		if errorStr != nil {
			return errors.New(C.GoString(errorStr))
		}
		return nil
	}

	busMap, exists := e.mixerConnections[mixerPtr]
	if !exists {
		// No connections for this mixer, fall back to global control
		errorStr := C.audiomixer_set_pan(mixerPtr, C.float(pan), C.int(inputBus))
		if errorStr != nil {
			return errors.New(C.GoString(errorStr))
		}
		return nil
	}

	sourcePtr, exists := busMap[inputBus]
	if !exists {
		// No source connected to this bus, fall back to global control
		errorStr := C.audiomixer_set_pan(mixerPtr, C.float(pan), C.int(inputBus))
		if errorStr != nil {
			return errors.New(C.GoString(errorStr))
		}
		return nil
	}

	// Found a connected source - use per-connection control for true per-bus behavior
	errorStr := C.audiomixer_set_input_pan_for_connection(sourcePtr, mixerPtr, C.int(inputBus), C.float(pan))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// GetMixerPanForBus gets the pan for a specific input bus on the mixer
// Uses connection tracking to provide true per-bus control via AVAudioMixingDestination
func (e *Engine) GetMixerPanForBus(mixerPtr unsafe.Pointer, inputBus int) (float32, error) {
	if e == nil || e.ptr == nil {
		return 0.0, errors.New("engine is nil")
	}
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	// Use connection tracking for true per-bus control
	if e.mixerConnections == nil {
		// No connections tracked, fall back to global control
		var result C.float
		errorStr := C.audiomixer_get_pan(mixerPtr, C.int(inputBus), &result)
		if errorStr != nil {
			return 0.0, errors.New(C.GoString(errorStr))
		}
		return float32(result), nil
	}

	busMap, exists := e.mixerConnections[mixerPtr]
	if !exists {
		// No connections for this mixer, fall back to global control
		var result C.float
		errorStr := C.audiomixer_get_pan(mixerPtr, C.int(inputBus), &result)
		if errorStr != nil {
			return 0.0, errors.New(C.GoString(errorStr))
		}
		return float32(result), nil
	}

	sourcePtr, exists := busMap[inputBus]
	if !exists {
		// No source connected to this bus, fall back to global control
		var result C.float
		errorStr := C.audiomixer_get_pan(mixerPtr, C.int(inputBus), &result)
		if errorStr != nil {
			return 0.0, errors.New(C.GoString(errorStr))
		}
		return float32(result), nil
	}

	// Found a connected source - use per-connection control for true per-bus behavior
	var result C.float
	errorStr := C.audiomixer_get_input_pan_for_connection(sourcePtr, mixerPtr, C.int(inputBus), &result)
	if errorStr != nil {
		return 0.0, errors.New(C.GoString(errorStr))
	}
	return float32(result), nil
}

// SetConnectionVolume sets the gain for a specific source->mixer input connection
// This allows controlling the volume of a specific source as it feeds into a mixer
func (e *Engine) SetConnectionVolume(sourcePtr, mixerPtr unsafe.Pointer, destBus int, volume float32) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}
	if sourcePtr == nil {
		return errors.New("source pointer is nil")
	}
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}
	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}

	errorStr := C.audiomixer_set_input_volume_for_connection(sourcePtr, mixerPtr, C.int(destBus), C.float(volume))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// GetConnectionVolume reads the gain for a specific source->mixer input connection
func (e *Engine) GetConnectionVolume(sourcePtr, mixerPtr unsafe.Pointer, destBus int) (float32, error) {
	if e == nil || e.ptr == nil {
		return 0.0, errors.New("engine is nil")
	}
	if sourcePtr == nil {
		return 0.0, errors.New("source pointer is nil")
	}
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	var result C.float
	errorStr := C.audiomixer_get_input_volume_for_connection(sourcePtr, mixerPtr, C.int(destBus), &result)
	if errorStr != nil {
		return 0.0, errors.New(C.GoString(errorStr))
	}
	return float32(result), nil
}

// SetConnectionPan sets the pan for a specific source->mixer input connection
// pan should be between -1.0 (left) and 1.0 (right), 0.0 is center
func (e *Engine) SetConnectionPan(sourcePtr, mixerPtr unsafe.Pointer, destBus int, pan float32) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}
	if sourcePtr == nil {
		return errors.New("source pointer is nil")
	}
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}
	if pan < -1.0 || pan > 1.0 {
		return errors.New("pan must be between -1.0 (left) and 1.0 (right)")
	}

	errorStr := C.audiomixer_set_input_pan_for_connection(sourcePtr, mixerPtr, C.int(destBus), C.float(pan))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// GetConnectionPan reads the pan for a specific source->mixer input connection
func (e *Engine) GetConnectionPan(sourcePtr, mixerPtr unsafe.Pointer, destBus int) (float32, error) {
	if e == nil || e.ptr == nil {
		return 0.0, errors.New("engine is nil")
	}
	if sourcePtr == nil {
		return 0.0, errors.New("source pointer is nil")
	}
	if mixerPtr == nil {
		return 0.0, errors.New("mixer pointer is nil")
	}

	var result C.float
	errorStr := C.audiomixer_get_input_pan_for_connection(sourcePtr, mixerPtr, C.int(destBus), &result)
	if errorStr != nil {
		return 0.0, errors.New(C.GoString(errorStr))
	}
	return float32(result), nil
}

// =============================================================================
// CONVENIENCE METHODS
// =============================================================================

// InspectNode returns comprehensive information about a node for debugging
type NodeInfo struct {
	InputCount    int
	OutputCount   int
	IsAttached    bool
	InputFormats  []*Format
	OutputFormats []*Format
}

// InspectNode provides a comprehensive overview of a node's capabilities
func (e *Engine) InspectNode(nodePtr unsafe.Pointer) (*NodeInfo, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}
	if nodePtr == nil {
		return nil, errors.New("node pointer is nil")
	}

	info := &NodeInfo{}

	// Get basic counts
	var err error
	info.InputCount, err = e.GetNodeInputCount(nodePtr)
	if err != nil {
		return nil, err
	}

	info.OutputCount, err = e.GetNodeOutputCount(nodePtr)
	if err != nil {
		return nil, err
	}

	info.IsAttached, err = e.IsNodeAttached(nodePtr)
	if err != nil {
		return nil, err
	}

	// Get input formats (only if attached to avoid format errors)
	if info.IsAttached {
		info.InputFormats = make([]*Format, info.InputCount)
		for i := 0; i < info.InputCount; i++ {
			format, err := e.GetNodeInputFormat(nodePtr, i)
			if err != nil {
				// Some buses might not have formats yet - that's okay
				info.InputFormats[i] = nil
			} else {
				info.InputFormats[i] = format
			}
		}

		// Get output formats
		info.OutputFormats = make([]*Format, info.OutputCount)
		for i := 0; i < info.OutputCount; i++ {
			format, err := e.GetNodeOutputFormat(nodePtr, i)
			if err != nil {
				// Some buses might not have formats yet - that's okay
				info.OutputFormats[i] = nil
			} else {
				info.OutputFormats[i] = format
			}
		}
	}

	return info, nil
}

// SetMixerConfiguration applies volume and pan settings to multiple buses on a mixer
type MixerBusConfig struct {
	Bus    int
	Volume float32
	Pan    float32
}

// ConfigureMixerBuses applies volume and pan settings to multiple buses in one call
func (e *Engine) ConfigureMixerBuses(mixerPtr unsafe.Pointer, configs []MixerBusConfig) error {
	if e == nil || e.ptr == nil {
		return errors.New("engine is nil")
	}
	if mixerPtr == nil {
		return errors.New("mixer pointer is nil")
	}

	for i, config := range configs {
		// Set volume
		if err := e.SetMixerVolumeForBus(mixerPtr, config.Volume, config.Bus); err != nil {
			return errors.New("failed to set volume for bus " + string(rune(i+'0')) + ": " + err.Error())
		}

		// Set pan
		if err := e.SetMixerPanForBus(mixerPtr, config.Pan, config.Bus); err != nil {
			return errors.New("failed to set pan for bus " + string(rune(i+'0')) + ": " + err.Error())
		}
	}

	return nil
}
