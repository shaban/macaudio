package unit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/unit.m"
#include <stdlib.h>

// Function declarations - CGO resolves UnitResult from .m file
UnitResult create_unit_effect(uint32_t type, uint32_t subtype, uint32_t manufacturer);
const char* release_unit_effect(void* effectPtr);
const char* set_effect_parameter(void* effectPtr, uint64_t address, float value);
UnitResult get_effect_parameter(void* effectPtr, uint64_t address);
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/shaban/macaudio/plugins"
)

// Effect represents an AVAudioUnitEffect instance
type Effect struct {
	ptr    unsafe.Pointer
	plugin *plugins.Plugin // Store plugin metadata for parameter access
}

// CreateEffect creates an AVAudioUnitEffect from a plugins.Plugin
func CreateEffect(plugin *plugins.Plugin) (*Effect, error) {
	// Convert string IDs to OSTypes
	typeID := stringToOSType(plugin.Type)
	subtypeID := stringToOSType(plugin.Subtype)
	manufacturerID := stringToOSType(plugin.ManufacturerID)

	result := C.create_unit_effect(C.uint32_t(typeID), C.uint32_t(subtypeID), C.uint32_t(manufacturerID))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}
	if result.result == nil {
		return nil, fmt.Errorf("failed to create effect: %s by %s", plugin.Name, plugin.ManufacturerID)
	}

	return &Effect{
		ptr:    unsafe.Pointer(result.result),
		plugin: plugin,
	}, nil
}

// Release frees the effect resources
func (e *Effect) Release() error {
	if e.ptr != nil {
		errorStr := C.release_unit_effect(e.ptr)
		e.ptr = nil
		if errorStr != nil {
			return errors.New(C.GoString(errorStr))
		}
	}
	return nil
}

// SetParameter sets a parameter value using the parameter from plugins introspection
func (e *Effect) SetParameter(param plugins.Parameter, value float32) error {
	if e.ptr == nil {
		return fmt.Errorf("effect has been released")
	}

	errorStr := C.set_effect_parameter(e.ptr, C.uint64_t(param.Address), C.float(value))
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// GetParameter gets a parameter value using the parameter from plugins introspection
func (e *Effect) GetParameter(param plugins.Parameter) (float32, error) {
	if e.ptr == nil {
		return 0, fmt.Errorf("effect has been released")
	}

	result := C.get_effect_parameter(e.ptr, C.uint64_t(param.Address))
	if result.error != nil {
		return 0, errors.New(C.GoString(result.error))
	}

	// The result contains a float* that we need to dereference and free
	valuePtr := (*C.float)(result.result)
	value := float32(*valuePtr)
	C.free(result.result) // Free the malloc'd memory from native code

	return value, nil
}

// GetPlugin returns the plugin metadata for this effect
func (e *Effect) GetPlugin() *plugins.Plugin {
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
