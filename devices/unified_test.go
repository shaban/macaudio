//go:build darwin && cgo

package devices

import (
	"fmt"
	"testing"
)

func TestGetAudio(t *testing.T) {
	fmt.Println("Testing unified getAudioDevices function...")

	// Enable JSON logging for testing
	SetJSONLogging(true)
	defer SetJSONLogging(false)

	devices, err := GetAudio()
	if err != nil {
		t.Fatalf("Error getting audio devices: %v", err)
	}

	if len(devices) == 0 {
		t.Error("No audio devices found")
		return
	}

	fmt.Printf("Found %d unified audio devices\n", len(devices))

	// Test the first device structure
	firstDevice := devices[0]
	fmt.Printf("First device: %+v\n", firstDevice)

	// Verify the helper methods work
	fmt.Printf("Can input: %t\n", firstDevice.CanInput())
	fmt.Printf("Can output: %t\n", firstDevice.CanOutput())
	fmt.Printf("Is input/output: %t\n", firstDevice.IsInputOutput())
	fmt.Printf("Is input only: %t\n", firstDevice.IsInputOnly())
	fmt.Printf("Is output only: %t\n", firstDevice.IsOutputOnly())

	// Count device types
	inputDevices := 0
	outputDevices := 0
	ioDevices := 0

	for _, device := range devices {
		if device.CanInput() {
			inputDevices++
		}
		if device.CanOutput() {
			outputDevices++
		}
		if device.IsInputOutput() {
			ioDevices++
		}

		// Verify device has required fields
		if device.Name == "" {
			t.Errorf("Device %d has empty name", device.DeviceID)
		}
		if device.UID == "" {
			t.Errorf("Device %d has empty UID", device.DeviceID)
		}
	}

	fmt.Printf("Device capabilities:\n")
	fmt.Printf("  Input capable: %d\n", inputDevices)
	fmt.Printf("  Output capable: %d\n", outputDevices)
	fmt.Printf("  Input/Output: %d\n", ioDevices)

	fmt.Println("✅ All unified device tests passed!")
}

func TestConvenienceFilters(t *testing.T) {
	fmt.Println("Testing convenience filter functions...")

	// Get all devices first
	allDevices, err := GetAudio()
	if err != nil {
		t.Fatalf("Error getting audio devices: %v", err)
	}

	// Test filter methods
	inputDevices := allDevices.Inputs()
	outputDevices := allDevices.Outputs()
	ioDevices := allDevices.InputOutput()
	onlineDevices := allDevices.Online()
	builtinDevices := allDevices.ByType("builtin")

	fmt.Printf("Filter results:\n")
	fmt.Printf("  Input devices: %d\n", len(inputDevices))
	fmt.Printf("  Output devices: %d\n", len(outputDevices))
	fmt.Printf("  Input/Output devices: %d\n", len(ioDevices))
	fmt.Printf("  Online devices: %d\n", len(onlineDevices))
	fmt.Printf("  Built-in devices: %d\n", len(builtinDevices))

	// Verify all input devices can actually input
	for _, device := range inputDevices {
		if !device.CanInput() {
			t.Errorf("Input filter returned device that can't input: %s", device.Name)
		}
	}

	// Verify all output devices can actually output
	for _, device := range outputDevices {
		if !device.CanOutput() {
			t.Errorf("Output filter returned device that can't output: %s", device.Name)
		}
	}

	// Verify all I/O devices can do both
	for _, device := range ioDevices {
		if !device.IsInputOutput() {
			t.Errorf("I/O filter returned device that can't do both: %s", device.Name)
		}
	}

	// Verify all built-in devices have correct type
	for _, device := range builtinDevices {
		if device.DeviceType != "builtin" {
			t.Errorf("Built-in filter returned non-built-in device: %s (type: %s)", device.Name, device.DeviceType)
		}
	}

	fmt.Println("✅ All filter tests passed!")
}
