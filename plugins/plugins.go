//go:build darwin && cgo

// Package plugins provides AudioUnit (AU) plugin enumeration and introspection for macOS.
//
// Model:
//   - Quick scan (List) returns lightweight PluginInfo entries (no parameters).
//   - Introspection is keyed by a 4‑tuple (type, subtype, manufacturerID, name).
//     Name disambiguates plugins within a (type, subtype, manufacturerID) suite.
//
// Public usage:
//   - List() → PluginInfos, then filter chains (ByType/BySubtype/ByManufacturer/ByName/ByCategory).
//   - PluginInfo.Introspect() → a single Plugin (uses the 4‑tuple including Name).
//   - PluginInfos.Introspect() → []Plugin for batch introspection (maps .Introspect over the slice).
//   - To operate on a suite, filter by the triplet on List() and then introspect each item.
package plugins

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit
#include "native/plugins.m"
#include <stdlib.h>
#include <string.h>

// Declare the functions so CGO can find them
char *QuickScanAudioUnits(void);
// 4-arg version: name == nil/empty => suite mode (all matches)
char *IntrospectAudioUnits(const char *type, const char *subtype, const char *manufacturerID, const char *name);
void SetVerboseLogging(int enabled);
// Timeout setters (configured from Go)
void SetPresetLoadingTimeout(double seconds);
void SetProcessUpdateTimeout(double seconds);
void SetTotalTimeout(double seconds);
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unsafe"
)

// JSON logging control (follows devices package pattern)
var enableJSONLogging = false
var jsonLogWriter io.Writer

// SetJSONLogging enables or disables JSON logging for debugging
// SetJSONLogging enables/disables emission of raw JSON records from the native layer.
// Use SetJSONLogWriter to redirect these records to a file for offline analysis.
func SetJSONLogging(enabled bool) {
	enableJSONLogging = enabled
}

// SetJSONLogWriter sets a destination for JSON logs when JSON logging is enabled.
// If nil, logs will go to stdout.
// SetJSONLogWriter sets a destination for JSON logs when JSON logging is enabled.
// If nil, logs will go to stdout.
func SetJSONLogWriter(w io.Writer) { jsonLogWriter = w }

func logJSON(label, payload string) {
	if !enableJSONLogging {
		return
	}
	if jsonLogWriter != nil {
		fmt.Fprintf(jsonLogWriter, "%s: %s\n", label, payload)
		return
	}
	fmt.Printf("%s: %s\n", label, payload)
}

// Error codes reported by the native layer in JSON envelopes
const (
	ErrCodeUnknown           = -1 // generic / uncategorized error
	ErrCodeJSONSerialization = -2 // JSON serialization failed
	ErrCodeNotFound          = -3 // single-plugin not found for given identifiers
)

// Timeout configuration (seconds). These feed through to the native layer.
// Use small values to speed up development scans; larger for stability.
func SetPresetLoadingTimeout(seconds float64) { C.SetPresetLoadingTimeout(C.double(seconds)) }
func SetProcessUpdateTimeout(seconds float64) { C.SetProcessUpdateTimeout(C.double(seconds)) }
func SetTotalTimeout(seconds float64)         { C.SetTotalTimeout(C.double(seconds)) }

// PluginInfo represents basic AudioUnit plugin information (quick scan)
type PluginInfo struct {
	Name           string `json:"name"`
	ManufacturerID string `json:"manufacturerID"`
	Type           string `json:"type"`
	Subtype        string `json:"subtype"`
	Category       string `json:"category"`
}

// QuickScanResponse represents the response from quick scan (like devices pattern)
type QuickScanResponse struct {
	Success             bool         `json:"success"`
	Plugins             []PluginInfo `json:"plugins"`
	PluginCount         int          `json:"pluginCount"`
	TotalPluginsScanned int          `json:"totalPluginsScanned"`
	Error               string       `json:"error,omitempty"`
	ErrorCode           int          `json:"errorCode,omitempty"`
}

// PluginResult represents the response from introspection (like devices pattern)
// pluginResult is the uniform envelope returned by native JSON (unexported)
type pluginResult struct {
	Success             bool      `json:"success"`
	Plugins             []*Plugin `json:"plugins"`
	PluginCount         int       `json:"pluginCount"`
	TotalPluginsScanned int       `json:"totalPluginsScanned"`
	TimedOut            bool      `json:"timedOut,omitempty"`
	Error               string    `json:"error,omitempty"`
	ErrorCode           int       `json:"errorCode,omitempty"`
}

// PluginInfos represents a collection of PluginInfo objects with filtering methods
type PluginInfos []PluginInfo

// Plugin represents an Audio Unit plugin with its complete metadata and parameters
type Plugin struct {
	Name           string      `json:"name"`
	ManufacturerID string      `json:"manufacturerID"`
	Type           string      `json:"type"`
	Subtype        string      `json:"subtype"`
	Category       string      `json:"category"`
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

// List returns a quick enumeration of all available AudioUnit plugins (no parameters)
// List performs a fast enumeration of installed AudioUnit plugins without
// instantiating them. It returns PluginInfo entries that can be filtered and
// later introspected individually.
func List() (PluginInfos, error) {
	cPluginList := C.QuickScanAudioUnits()
	if cPluginList == nil {
		return nil, fmt.Errorf("failed to scan AudioUnit plugins")
	}
	defer C.free(unsafe.Pointer(cPluginList))

	jsonData := C.GoString(cPluginList)

	// JSON logging when enabled (follows devices pattern)
	logJSON("QuickScan", jsonData)

	var response QuickScanResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return nil, fmt.Errorf("failed to parse plugin list data: %v", err)
	}

	// Check for success status (like devices pattern)
	if !response.Success {
		errorMsg := response.Error
		if errorMsg == "" {
			errorMsg = "unknown error"
		}
		return nil, fmt.Errorf("plugin scan failed: %s (code: %d)", errorMsg, response.ErrorCode)
	}

	return PluginInfos(response.Plugins), nil
}

// Filter methods for PluginInfos collection

// ByManufacturer returns plugin infos from a specific manufacturer ID
func (infos PluginInfos) ByManufacturer(manufacturerID string) PluginInfos {
	var filtered PluginInfos
	for _, info := range infos {
		if info.ManufacturerID == manufacturerID {
			filtered = append(filtered, info)
		}
	}
	return filtered
}

// ByType returns plugin infos of a specific type (e.g., "aufx", "aumu", "aumf")
func (infos PluginInfos) ByType(pluginType string) PluginInfos {
	var filtered PluginInfos
	for _, info := range infos {
		if info.Type == pluginType {
			filtered = append(filtered, info)
		}
	}
	return filtered
}

// BySubtype returns plugin infos of a specific subtype
func (infos PluginInfos) BySubtype(subtype string) PluginInfos {
	var filtered PluginInfos
	for _, info := range infos {
		if info.Subtype == subtype {
			filtered = append(filtered, info)
		}
	}
	return filtered
}

// ByName returns plugin infos matching a specific name pattern (case-insensitive)
func (infos PluginInfos) ByName(namePattern string) PluginInfos {
	var filtered PluginInfos
	for _, info := range infos {
		if matchesPattern(info.Name, namePattern) {
			filtered = append(filtered, info)
		}
	}
	return filtered
}

// ByCategory returns plugin infos of a specific category (e.g., "Effect", "Instrument", "Mixer")
func (infos PluginInfos) ByCategory(category string) PluginInfos {
	var filtered PluginInfos
	for _, info := range infos {
		if info.Category == category {
			filtered = append(filtered, info)
		}
	}
	return filtered
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

/*
// Introspect introspects a specific AudioUnit plugin by its identifiers
func Introspect(pluginType, subtype, manufacturerID string) (Plugin, error) {
	cType := C.CString(pluginType)
	defer C.free(unsafe.Pointer(cType))

	cSubtype := C.CString(subtype)
	defer C.free(unsafe.Pointer(cSubtype))

	cManufacturerID := C.CString(manufacturerID)
	defer C.free(unsafe.Pointer(cManufacturerID))

	cResult := C.Introspect(cType, cSubtype, cManufacturerID)
	if cResult == nil {
		return Plugin{}, fmt.Errorf("failed to introspect plugin %s:%s:%s", pluginType, subtype, manufacturerID)
	}
	defer C.free(unsafe.Pointer(cResult))

	jsonData := C.GoString(cResult)

	// JSON logging when enabled (follows devices pattern)
	if enableJSONLogging {
		fmt.Printf("🔍 Introspect JSON: %s\n", jsonData)
	}

	// Parse JSON into Plugin struct
	var plugin Plugin
	if err := json.Unmarshal([]byte(jsonData), &plugin); err != nil {
		return Plugin{}, fmt.Errorf("failed to parse plugin data: %v", err)
	}

	return plugin, nil
}

// IntrospectFromInfo is a helper function that accepts a PluginInfo object
// This provides a more user-friendly API for introspecting plugins
func IntrospectFromInfo(plugin PluginInfo) (Plugin, error) {
	return Introspect(plugin.Type, plugin.Subtype, plugin.ManufacturerID)
}
*/

// cStringOrNil returns nil for empty strings (used to signal suite mode for name)
func cStringOrNil(s string) *C.char {
	if s == "" {
		return nil
	}
	return C.CString(s)
}

// introspect is the omnipotent internal function: name == "" ⇒ suite; name set ⇒ single
func introspect(pluginType, subtype, manufacturerID, name string) ([]*Plugin, error) {
	cType := cStringOrNil(pluginType)
	cSubtype := cStringOrNil(subtype)
	cMan := cStringOrNil(manufacturerID)
	cName := cStringOrNil(name)

	if cType != nil {
		defer C.free(unsafe.Pointer(cType))
	}
	if cSubtype != nil {
		defer C.free(unsafe.Pointer(cSubtype))
	}
	if cMan != nil {
		defer C.free(unsafe.Pointer(cMan))
	}
	if cName != nil {
		defer C.free(unsafe.Pointer(cName))
	}

	cResult := C.IntrospectAudioUnits(cType, cSubtype, cMan, cName)
	if cResult == nil {
		return nil, fmt.Errorf("failed to introspect plugins")
	}
	defer C.free(unsafe.Pointer(cResult))

	jsonData := C.GoString(cResult)

	// JSON logging when enabled
	logJSON(fmt.Sprintf("Introspect[%s:%s:%s:%s]", pluginType, subtype, manufacturerID, name), jsonData)

	// Parse JSON into pluginResult struct (like devices pattern)
	var result pluginResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		return nil, fmt.Errorf("failed to parse plugin result data: %v", err)
	}

	// Check for success status (like devices pattern)
	if !result.Success {
		errorMsg := result.Error
		if errorMsg == "" {
			errorMsg = "unknown error"
		}
		return nil, fmt.Errorf("plugin introspection failed: %s (code: %d)", errorMsg, result.ErrorCode)
	}

	return result.Plugins, nil
}

// IntrospectSuite returns all plugins for the triplet (0..N)
func (pi PluginInfo) IntrospectSuite() ([]*Plugin, error) {
	return introspect(pi.Type, pi.Subtype, pi.ManufacturerID, "")
}

// Introspect returns exactly one plugin for the quadruplet; errors otherwise
func (pi PluginInfo) Introspect() (*Plugin, error) {
	results, err := introspect(pi.Type, pi.Subtype, pi.ManufacturerID, pi.Name)
	if err != nil {
		return nil, err
	}
	if len(results) != 1 {
		return nil, fmt.Errorf("expected 1 plugin, got %d for %s:%s:%s:%s",
			len(results), pi.Type, pi.Subtype, pi.ManufacturerID, pi.Name)
	}
	return results[0], nil
}

// Introspect method on PluginInfos - returns slice of Plugins
// Introspect maps PluginInfo.Introspect() over the slice, returning fully
// populated Plugin objects with parameter metadata. Fail-fast on first error.
func (infos PluginInfos) Introspect() ([]*Plugin, error) {
	var allPlugins []*Plugin
	for _, info := range infos {
		plugin, err := info.Introspect()
		if err != nil {
			return nil, fmt.Errorf("failed to introspect plugin %s: %v", info.Name, err)
		}
		allPlugins = append(allPlugins, plugin)
	}
	return allPlugins, nil
}
