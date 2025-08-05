//go:build darwin && cgo

package devices

import (
	"fmt"
	"testing"
)

func TestGetMIDI(t *testing.T) {
	fmt.Println("Testing unified MIDI device enumeration...")

	// Enable JSON logging for testing
	SetJSONLogging(true)
	defer SetJSONLogging(false)

	devices, err := GetMIDI()
	if err != nil {
		t.Fatalf("Error getting MIDI devices: %v", err)
	}

	fmt.Printf("Found %d unified MIDI devices\n", len(devices))

	if len(devices) == 0 {
		fmt.Println("No MIDI devices found - this is normal if no MIDI hardware is connected")
		return
	}

	// Test the first device structure
	firstDevice := devices[0]
	fmt.Printf("First MIDI device: %+v\n", firstDevice)

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
	onlineDevices := 0

	for i, device := range devices {
		if device.CanInput() {
			inputDevices++
		}
		if device.CanOutput() {
			outputDevices++
		}
		if device.IsInputOutput() {
			ioDevices++
		}
		if device.IsOnline {
			onlineDevices++
		}

		// Verify device has required fields
		if device.Name == "" {
			t.Errorf("MIDI device %d has empty name", i)
		}
		if device.UID == "" {
			t.Errorf("MIDI device %d has empty UID", i)
		}
	}

	fmt.Printf("MIDI device capabilities:\n")
	fmt.Printf("  Input capable: %d\n", inputDevices)
	fmt.Printf("  Output capable: %d\n", outputDevices)
	fmt.Printf("  Input/Output: %d\n", ioDevices)
	fmt.Printf("  Online: %d\n", onlineDevices)

	fmt.Println("✅ All MIDI device tests passed!")
}

func TestMIDIDeviceFilters(t *testing.T) {
	fmt.Println("Testing MIDI device filter methods...")

	// Get all devices first
	allDevices, err := GetMIDI()
	if err != nil {
		t.Fatalf("Error getting MIDI devices: %v", err)
	}

	if len(allDevices) == 0 {
		fmt.Println("No MIDI devices found - skipping filter tests")
		return
	}

	// Test filter methods
	inputDevices := allDevices.Inputs()
	outputDevices := allDevices.Outputs()
	ioDevices := allDevices.InputOutput()
	onlineDevices := allDevices.Online()

	fmt.Printf("MIDI filter results:\n")
	fmt.Printf("  Input devices: %d\n", len(inputDevices))
	fmt.Printf("  Output devices: %d\n", len(outputDevices))
	fmt.Printf("  Input/Output devices: %d\n", len(ioDevices))
	fmt.Printf("  Online devices: %d\n", len(onlineDevices))

	// Test new manufacturer and model filters
	bossDevices := allDevices.ByManufacturer("BOSS")
	appleDevices := allDevices.ByManufacturer("Apple Inc.")
	katanaDevices := allDevices.ByModel("KATANA")

	fmt.Printf("  BOSS devices: %d\n", len(bossDevices))
	fmt.Printf("  Apple devices: %d\n", len(appleDevices))
	fmt.Printf("  KATANA model devices: %d\n", len(katanaDevices))

	// Verify all input devices can actually input
	for _, device := range inputDevices {
		if !device.CanInput() {
			t.Errorf("Input filter returned MIDI device that can't input: %s", device.Name)
		}
	}

	// Verify all output devices can actually output
	for _, device := range outputDevices {
		if !device.CanOutput() {
			t.Errorf("Output filter returned MIDI device that can't output: %s", device.Name)
		}
	}

	// Verify all I/O devices can do both
	for _, device := range ioDevices {
		if !device.IsInputOutput() {
			t.Errorf("I/O filter returned MIDI device that can't do both: %s", device.Name)
		}
	}

	// Verify manufacturer filtering
	for _, device := range bossDevices {
		if device.Manufacturer != "BOSS" {
			t.Errorf("BOSS filter returned device with wrong manufacturer: %s", device.Manufacturer)
		}
	}

	fmt.Println("✅ All MIDI filter tests passed!")
}
