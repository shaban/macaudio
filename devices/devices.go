//go:build darwin && cgo

package devices

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreAudio -framework AudioToolbox -framework CoreMIDI -framework AVFoundation
#include "native/devices.m"
#include <stdlib.h>

// Function declarations
char* getAudioDevices(void);
char* getMIDIDevices(void);
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// JSON logging control
var enableJSONLogging bool = false

// SetJSONLogging enables or disables JSON logging for debugging
func SetJSONLogging(enable bool) {
	enableJSONLogging = enable
}

// Device represents the common properties of any device
type Device struct {
	Name     string `json:"name"`
	UID      string `json:"uid"`
	IsOnline bool   `json:"isOnline"`
}

// AudioDevice represents a unified audio device with full capabilities
type AudioDevice struct {
	Device                      // Embedded base device
	DeviceID             int    `json:"deviceId"`
	InputChannelCount    int    `json:"inputChannelCount"`
	OutputChannelCount   int    `json:"outputChannelCount"`
	IsDefaultInput       bool   `json:"isDefaultInput"`
	IsDefaultOutput      bool   `json:"isDefaultOutput"`
	SupportedSampleRates []int  `json:"supportedSampleRates"`
	SupportedBitDepths   []int  `json:"supportedBitDepths"`
	DeviceType           string `json:"deviceType"`    // "builtin", "usb", "aggregate"
	TransportType        string `json:"transportType"` // "usb", "firewire", "bluetooth"
}

// Helper methods for capability checking
func (a AudioDevice) CanInput() bool {
	return a.InputChannelCount > 0
}

func (a AudioDevice) CanOutput() bool {
	return a.OutputChannelCount > 0
}

func (a AudioDevice) IsInputOutput() bool {
	return a.CanInput() && a.CanOutput()
}

func (a AudioDevice) IsInputOnly() bool {
	return a.CanInput() && !a.CanOutput()
}

func (a AudioDevice) IsOutputOnly() bool {
	return a.CanOutput() && !a.CanInput()
}

// CommonSampleRates returns sample rates supported by both devices
func (a AudioDevice) CommonSampleRates(other AudioDevice) []int {
	if len(a.SupportedSampleRates) == 0 || len(other.SupportedSampleRates) == 0 {
		return []int{}
	}

	// Create a map for fast lookup
	otherRates := make(map[int]bool)
	for _, rate := range other.SupportedSampleRates {
		otherRates[rate] = true
	}

	// Find intersection, preserving order from first device
	var common []int
	for _, rate := range a.SupportedSampleRates {
		if otherRates[rate] {
			common = append(common, rate)
		}
	}

	return common
}

// CommonBitDepths returns bit depths supported by both devices
func (a AudioDevice) CommonBitDepths(other AudioDevice) []int {
	if len(a.SupportedBitDepths) == 0 || len(other.SupportedBitDepths) == 0 {
		return []int{}
	}

	// Create a map for fast lookup
	otherDepths := make(map[int]bool)
	for _, depth := range other.SupportedBitDepths {
		otherDepths[depth] = true
	}

	// Find intersection, preserving order from first device
	var common []int
	for _, depth := range a.SupportedBitDepths {
		if otherDepths[depth] {
			common = append(common, depth)
		}
	}

	return common
}

// AudioDevices represents a slice of AudioDevice with filter methods
type AudioDevices []AudioDevice

// Inputs returns only devices that can capture audio
func (devices AudioDevices) Inputs() AudioDevices {
	var inputs AudioDevices
	for _, device := range devices {
		if device.CanInput() {
			inputs = append(inputs, device)
		}
	}
	return inputs
}

// Outputs returns only devices that can play audio
func (devices AudioDevices) Outputs() AudioDevices {
	var outputs AudioDevices
	for _, device := range devices {
		if device.CanOutput() {
			outputs = append(outputs, device)
		}
	}
	return outputs
}

// InputOutput returns only devices that can both capture and play audio
func (devices AudioDevices) InputOutput() AudioDevices {
	var ioDevices AudioDevices
	for _, device := range devices {
		if device.IsInputOutput() {
			ioDevices = append(ioDevices, device)
		}
	}
	return ioDevices
}

// Online returns only devices that are currently online/connected
func (devices AudioDevices) Online() AudioDevices {
	var onlineDevices AudioDevices
	for _, device := range devices {
		if device.IsOnline {
			onlineDevices = append(onlineDevices, device)
		}
	}
	return onlineDevices
}

// ByType returns only devices of a specific type (e.g., "usb", "builtin", "bluetooth")
func (devices AudioDevices) ByType(deviceType string) AudioDevices {
	var filteredDevices AudioDevices
	for _, device := range devices {
		if device.DeviceType == deviceType {
			filteredDevices = append(filteredDevices, device)
		}
	}
	return filteredDevices
}

// MIDIDevice represents a MIDI device with input/output capabilities
type MIDIDevice struct {
	Device                  // Embedded base device
	DeviceName       string `json:"deviceName"`   // Parent device name (e.g. "Morgan Bridge", "KATANA")
	Manufacturer     string `json:"manufacturer"` // Device manufacturer (e.g. "Apple", "Roland")
	Model            string `json:"model"`        // Device model (e.g. "IAC Driver", "KATANA-100")
	EntityName       string `json:"entityName"`   // Entity name within device (e.g. "Bus 1", "MIDI Port")
	DisplayName      string `json:"displayName"`  // Full display name (e.g. "KATANA KATANA DAW CTRL")
	SysExSpeed       int    `json:"sysExSpeed"`   // Maximum SysEx transfer speed in bytes/sec
	InputEndpointID  int    `json:"inputEndpointId"`
	OutputEndpointID int    `json:"outputEndpointId"`
	IsInput          bool   `json:"isInput"`
	IsOutput         bool   `json:"isOutput"`
}

// Helper methods for MIDI capability checking
func (m MIDIDevice) CanInput() bool {
	return m.IsInput
}

func (m MIDIDevice) CanOutput() bool {
	return m.IsOutput
}

func (m MIDIDevice) IsInputOutput() bool {
	return m.IsInput && m.IsOutput
}

func (m MIDIDevice) IsInputOnly() bool {
	return m.IsInput && !m.IsOutput
}

func (m MIDIDevice) IsOutputOnly() bool {
	return m.IsOutput && !m.IsInput
}

// Convenience methods for endpoint access
func (m MIDIDevice) GetInputEndpoint() int {
	if m.IsInput {
		return m.InputEndpointID
	}
	return 0
}

func (m MIDIDevice) GetOutputEndpoint() int {
	if m.IsOutput {
		return m.OutputEndpointID
	}
	return 0
}

// GetPrimaryEndpoint returns the input endpoint for input devices,
// output endpoint for output-only devices
func (m MIDIDevice) GetPrimaryEndpoint() int {
	if m.IsInput {
		return m.InputEndpointID
	}
	return m.OutputEndpointID
}

// MIDIDevices represents a slice of MIDIDevice with filter methods
type MIDIDevices []MIDIDevice

// Inputs returns only MIDI devices that can receive MIDI input
func (devices MIDIDevices) Inputs() MIDIDevices {
	var inputs MIDIDevices
	for _, device := range devices {
		if device.CanInput() {
			inputs = append(inputs, device)
		}
	}
	return inputs
}

// Outputs returns only MIDI devices that can send MIDI output
func (devices MIDIDevices) Outputs() MIDIDevices {
	var outputs MIDIDevices
	for _, device := range devices {
		if device.CanOutput() {
			outputs = append(outputs, device)
		}
	}
	return outputs
}

// InputOutput returns only MIDI devices that can both receive and send MIDI
func (devices MIDIDevices) InputOutput() MIDIDevices {
	var ioDevices MIDIDevices
	for _, device := range devices {
		if device.IsInputOutput() {
			ioDevices = append(ioDevices, device)
		}
	}
	return ioDevices
}

// Online returns only MIDI devices that are currently online/connected
func (devices MIDIDevices) Online() MIDIDevices {
	var onlineDevices MIDIDevices
	for _, device := range devices {
		if device.IsOnline {
			onlineDevices = append(onlineDevices, device)
		}
	}
	return onlineDevices
}

// ByManufacturer returns only MIDI devices from a specific manufacturer
func (devices MIDIDevices) ByManufacturer(manufacturer string) MIDIDevices {
	var filteredDevices MIDIDevices
	for _, device := range devices {
		if device.Manufacturer == manufacturer {
			filteredDevices = append(filteredDevices, device)
		}
	}
	return filteredDevices
}

// ByModel returns only MIDI devices of a specific model
func (devices MIDIDevices) ByModel(model string) MIDIDevices {
	var filteredDevices MIDIDevices
	for _, device := range devices {
		if device.Model == model {
			filteredDevices = append(filteredDevices, device)
		}
	}
	return filteredDevices
}

// MIDIDeviceResult represents the result for MIDI devices
type MIDIDeviceResult struct {
	Success     bool         `json:"success"`
	Error       string       `json:"error,omitempty"`
	ErrorCode   int          `json:"errorCode,omitempty"`
	Devices     []MIDIDevice `json:"devices"`
	DeviceCount int          `json:"deviceCount"`
}

// AudioDeviceResult represents the result for audio devices
type AudioDeviceResult struct {
	Success             bool          `json:"success"`
	Error               string        `json:"error,omitempty"`
	ErrorCode           int           `json:"errorCode,omitempty"`
	Devices             []AudioDevice `json:"devices"`
	DeviceCount         int           `json:"deviceCount"`
	TotalDevicesScanned int           `json:"totalDevicesScanned"`
}

// GetAudio returns all audio devices with unified input/output capabilities
func GetAudio() (AudioDevices, error) {
	result := C.getAudioDevices()
	defer C.free(unsafe.Pointer(result))

	jsonStr := C.GoString(result)

	// JSON logging when enabled
	if enableJSONLogging {
		fmt.Printf("🔍 Audio Devices JSON: %s\n", jsonStr)
	}

	var deviceResult AudioDeviceResult
	if err := json.Unmarshal([]byte(jsonStr), &deviceResult); err != nil {
		return nil, fmt.Errorf("failed to parse device result: %v", err)
	}

	if !deviceResult.Success {
		return nil, fmt.Errorf("core audio error (%d): %s", deviceResult.ErrorCode, deviceResult.Error)
	}

	return AudioDevices(deviceResult.Devices), nil
}

// GetMIDI returns all MIDI devices with unified input/output capabilities
func GetMIDI() (MIDIDevices, error) {
	cDeviceList := C.getMIDIDevices()
	defer C.free(unsafe.Pointer(cDeviceList))

	jsonData := C.GoString(cDeviceList)

	// JSON logging when enabled
	if enableJSONLogging {
		fmt.Printf("🔍 MIDI Devices JSON: %s\n", jsonData)
	}

	// Parse JSON response to check for success/error structure like audio devices
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		return nil, fmt.Errorf("failed to parse MIDI response: %v", err)
	}

	// Check if it's an error response
	if success, exists := response["success"]; exists {
		if successBool, ok := success.(bool); ok && !successBool {
			errorMsg := "Unknown MIDI error"
			if errorVal, exists := response["error"]; exists {
				if errorStr, ok := errorVal.(string); ok {
					errorMsg = errorStr
				}
			}
			return nil, fmt.Errorf("MIDI enumeration failed: %s", errorMsg)
		}

		// It's a success response, extract devices array
		if devicesVal, exists := response["devices"]; exists {
			if devicesArray, ok := devicesVal.([]interface{}); ok {
				// Convert back to JSON for parsing into MIDIDevice structs
				devicesJSON, err := json.Marshal(devicesArray)
				if err != nil {
					return nil, fmt.Errorf("failed to re-marshal MIDI devices: %v", err)
				}

				var devices []MIDIDevice
				if err := json.Unmarshal(devicesJSON, &devices); err != nil {
					return nil, fmt.Errorf("failed to parse MIDI devices: %v", err)
				}

				return devices, nil
			}
		}

		return nil, fmt.Errorf("invalid MIDI response structure")
	}

	// Fallback: try to parse as direct array (for backward compatibility)
	var devices []MIDIDevice
	if err := json.Unmarshal([]byte(jsonData), &devices); err != nil {
		return nil, fmt.Errorf("failed to parse MIDI devices: %v", err)
	}

	return MIDIDevices(devices), nil
}
