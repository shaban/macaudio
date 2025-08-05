//go:build darwin && cgo

package devices

import (
	"testing"
)

func TestGetAudioDevices(t *testing.T) {
	t.Log("Testing unified getAudioDevices function...")

	// Enable JSON logging for testing
	SetJSONLogging(true)
	defer SetJSONLogging(false)

	// Call the unified function
	devices, err := GetAudio()
	if err != nil {
		t.Fatalf("GetAudio returned error: %v", err)
	}

	// Check that we got at least one device
	if len(devices) == 0 {
		t.Fatal("No devices returned")
	}

	t.Logf("Found %d audio input devices", len(devices))

	// Verify the structure of the devices
	for _, device := range devices {
		t.Logf("First device: %+v", device)

		// Basic validation
		if device.Name == "" {
			t.Error("Device name is empty")
		}
		if device.UID == "" {
			t.Error("Device UID is empty")
		}
		if device.InputChannelCount == 0 && device.OutputChannelCount == 0 {
			t.Error("Device should have either input or output channels")
		}
		if len(device.SupportedSampleRates) == 0 {
			t.Error("Device should have at least one supported sample rate")
		}
		// crude debugging method for seeing device details
		// t.Logf("%+v\n", device)
	}
	t.Log("✅ All tests passed!")
}
