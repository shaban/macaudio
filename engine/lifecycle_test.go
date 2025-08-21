package engine

import (
	"testing"

	"github.com/shaban/macaudio/devices"
)

func TestEngineLifecycle(t *testing.T) {
	// Get audio devices
	allDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}

	outputDevices := allDevices.Outputs()
	if len(outputDevices) == 0 {
		t.Skip("No output devices available")
	}

	device := &outputDevices[0]
	if len(device.SupportedSampleRates) == 0 {
		t.Skip("Device has no supported sample rates")
	}

	// Create engine
	engine, err := NewEngine(device, 0, 512)
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	defer engine.Destroy()

	// Test initial state
	if engine.IsRunning() {
		t.Error("Engine should not be running initially")
	}

	// Start engine
	if err := engine.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify running
	if !engine.IsRunning() {
		t.Error("Engine should be running after Start")
	}

	// Test volume control
	if err := engine.SetMasterVolume(0.5); err != nil {
		t.Fatalf("SetMasterVolume failed: %v", err)
	}

	volume := engine.GetMasterVolume()
	if volume < 0.4 || volume > 0.6 {
		t.Errorf("Expected volume ~0.5, got %f", volume)
	}

	// Stop engine
	engine.Stop()
	if engine.IsRunning() {
		t.Error("Engine should not be running after Stop")
	}
}
