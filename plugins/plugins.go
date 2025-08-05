//go:build darwin && cgo

// Package plugins provides AU plugin enumeration and introspection for macOS
package plugins

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit
#include "native/plugins.m"
#include <stdlib.h>

// Define the external variable that the Objective-C code expects
int g_verboseLogging = 0;  // 0 = silent, 1 = verbose

// Declare the function so CGO can find it
char *IntrospectAudioUnitsWithTimeout(double timeoutSeconds);
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"strings"
	"unsafe"
)

// JSON logging control (follows devices package pattern)
var enableJSONLogging = false

// SetJSONLogging enables or disables JSON logging for debugging
func SetJSONLogging(enabled bool) {
	enableJSONLogging = enabled
}

// Plugin represents an Audio Unit plugin with its complete metadata and parameters
type Plugin struct {
	Name           string      `json:"name"`
	ManufacturerID string      `json:"manufacturerID"`
	Type           string      `json:"type"`
	Subtype        string      `json:"subtype"`
	Parameters     []Parameter `json:"parameters"`
}

// Parameter represents an Audio Unit parameter with its complete metadata
type Parameter struct {
	DisplayName         string   `json:"displayName"`
	Identifier          string   `json:"identifier"`
	Address             uint64   `json:"address"`
	MinValue            float32  `json:"minValue"`
	MaxValue            float32  `json:"maxValue"`
	DefaultValue        float32  `json:"defaultValue"`
	CurrentValue        float32  `json:"currentValue"`
	Unit                string   `json:"unit"`
	IsWritable          bool     `json:"isWritable"`
	CanRamp             bool     `json:"canRamp"`
	RawFlags            uint     `json:"rawFlags"`
	IndexedValues       []string `json:"indexedValues,omitempty"`
	IndexedValuesSource string   `json:"indexedValuesSource,omitempty"`
	IndexedMinValue     *int     `json:"indexedMinValue,omitempty"`
	IndexedMaxValue     *int     `json:"indexedMaxValue,omitempty"`
}

// Plugins represents a collection of Plugin objects with filtering methods
type Plugins []Plugin

// GetPlugins returns all available AudioUnit plugins with their parameters
func GetPlugins() (Plugins, error) {
	return GetPluginsWithTimeout(30.0) // Default 30-second timeout
}

// GetPluginsWithTimeout returns all available AudioUnit plugins with a specified timeout
func GetPluginsWithTimeout(timeoutSeconds float64) (Plugins, error) {
	cPluginList := C.IntrospectAudioUnitsWithTimeout(C.double(timeoutSeconds))
	if cPluginList == nil {
		return nil, fmt.Errorf("failed to introspect AudioUnit plugins")
	}
	defer C.free(unsafe.Pointer(cPluginList))

	jsonData := C.GoString(cPluginList)

	// JSON logging when enabled (follows devices pattern)
	if enableJSONLogging {
		fmt.Printf("🔍 Plugin Data JSON: %s\n", jsonData)
	}

	var plugins []Plugin
	if err := json.Unmarshal([]byte(jsonData), &plugins); err != nil {
		return nil, fmt.Errorf("failed to parse plugin data: %v", err)
	}

	return Plugins(plugins), nil
}

// Filter methods for Plugins collection

// ByManufacturer returns plugins from a specific manufacturer ID
func (plugins Plugins) ByManufacturer(manufacturerID string) Plugins {
	var filtered Plugins
	for _, plugin := range plugins {
		if plugin.ManufacturerID == manufacturerID {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

// ByType returns plugins of a specific type (e.g., "aufx", "aumu", "aumf")
func (plugins Plugins) ByType(pluginType string) Plugins {
	var filtered Plugins
	for _, plugin := range plugins {
		if plugin.Type == pluginType {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

// BySubtype returns plugins of a specific subtype
func (plugins Plugins) BySubtype(subtype string) Plugins {
	var filtered Plugins
	for _, plugin := range plugins {
		if plugin.Subtype == subtype {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

// WithParameters returns plugins that have at least one parameter
func (plugins Plugins) WithParameters() Plugins {
	var filtered Plugins
	for _, plugin := range plugins {
		if len(plugin.Parameters) > 0 {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

// WithIndexedParameters returns plugins that have at least one indexed parameter
func (plugins Plugins) WithIndexedParameters() Plugins {
	var filtered Plugins
	for _, plugin := range plugins {
		for _, param := range plugin.Parameters {
			if len(param.IndexedValues) > 0 {
				filtered = append(filtered, plugin)
				break
			}
		}
	}
	return filtered
}

// ByName returns plugins matching a specific name pattern (case-insensitive)
func (plugins Plugins) ByName(namePattern string) Plugins {
	var filtered Plugins
	for _, plugin := range plugins {
		if matchesPattern(plugin.Name, namePattern) {
			filtered = append(filtered, plugin)
		}
	}
	return filtered
}

// Helper function for name pattern matching
func matchesPattern(name, pattern string) bool {
	// Simple case-insensitive contains check for now
	// Could be enhanced with regex patterns if needed
	nameUpper := strings.ToUpper(name)
	patternUpper := strings.ToUpper(pattern)
	return strings.Contains(nameUpper, patternUpper)
}

// Parameter filtering methods for individual plugins

// GetParametersByUnit returns parameters of a specific unit type
func (plugin Plugin) GetParametersByUnit(unit string) []Parameter {
	var filtered []Parameter
	for _, param := range plugin.Parameters {
		if param.Unit == unit {
			filtered = append(filtered, param)
		}
	}
	return filtered
}

// GetIndexedParameters returns only parameters with indexed values
func (plugin Plugin) GetIndexedParameters() []Parameter {
	var filtered []Parameter
	for _, param := range plugin.Parameters {
		if len(param.IndexedValues) > 0 {
			filtered = append(filtered, param)
		}
	}
	return filtered
}

// GetWritableParameters returns only writable parameters
func (plugin Plugin) GetWritableParameters() []Parameter {
	var filtered []Parameter
	for _, param := range plugin.Parameters {
		if param.IsWritable {
			filtered = append(filtered, param)
		}
	}
	return filtered
}

// GetRampableParameters returns only parameters that can ramp
func (plugin Plugin) GetRampableParameters() []Parameter {
	var filtered []Parameter
	for _, param := range plugin.Parameters {
		if param.CanRamp {
			filtered = append(filtered, param)
		}
	}
	return filtered
}

// Summary methods for quick information

// Summary returns a brief summary of the plugin
func (plugin Plugin) Summary() string {
	return fmt.Sprintf("%s (%s) - %d parameters",
		plugin.Name, plugin.ManufacturerID, len(plugin.Parameters))
}

// ParameterCount returns the total number of parameters
func (plugin Plugin) ParameterCount() int {
	return len(plugin.Parameters)
}

// IndexedParameterCount returns the number of parameters with indexed values
func (plugin Plugin) IndexedParameterCount() int {
	count := 0
	for _, param := range plugin.Parameters {
		if len(param.IndexedValues) > 0 {
			count++
		}
	}
	return count
}

// WritableParameterCount returns the number of writable parameters
func (plugin Plugin) WritableParameterCount() int {
	count := 0
	for _, param := range plugin.Parameters {
		if param.IsWritable {
			count++
		}
	}
	return count
}
