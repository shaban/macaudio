package engine

import (
	"testing"
)

func TestMasterVolumeControl(t *testing.T) {
	// Create engine
	spec := DefaultAudioSpec()
	spec.SampleRate = 48000
	spec.BufferSize = 256
	
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Get the main mixer node
	mixerNode, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}

	// Test initial volume (should be 1.0)
	initialVolume, err := engine.GetMixerVolume(mixerNode)
	if err != nil {
		t.Fatalf("Failed to get initial mixer volume: %v", err)
	}
	t.Logf("Initial mixer volume: %f", initialVolume)

	// Test setting volume to 0.5
	testVolume := float32(0.5)
	err = engine.SetMixerVolume(mixerNode, testVolume)
	if err != nil {
		t.Fatalf("Failed to set mixer volume: %v", err)
	}

	// Verify the volume was set correctly
	currentVolume, err := engine.GetMixerVolume(mixerNode)
	if err != nil {
		t.Fatalf("Failed to get current mixer volume: %v", err)
	}

	if currentVolume != testVolume {
		t.Errorf("Expected volume %f, got %f", testVolume, currentVolume)
	}

	t.Logf("Successfully set and verified mixer volume: %f", currentVolume)

	// Test setting volume to 0.0 (mute)
	err = engine.SetMixerVolume(mixerNode, 0.0)
	if err != nil {
		t.Fatalf("Failed to mute mixer: %v", err)
	}

	muteVolume, err := engine.GetMixerVolume(mixerNode)
	if err != nil {
		t.Fatalf("Failed to get mute volume: %v", err)
	}

	if muteVolume != 0.0 {
		t.Errorf("Expected mute volume 0.0, got %f", muteVolume)
	}

	t.Logf("Successfully muted mixer: %f", muteVolume)

	// Test setting volume to maximum (1.0)
	err = engine.SetMixerVolume(mixerNode, 1.0)
	if err != nil {
		t.Fatalf("Failed to set max volume: %v", err)
	}

	maxVolume, err := engine.GetMixerVolume(mixerNode)
	if err != nil {
		t.Fatalf("Failed to get max volume: %v", err)
	}

	if maxVolume != 1.0 {
		t.Errorf("Expected max volume 1.0, got %f", maxVolume)
	}

	t.Logf("Successfully set max volume: %f", maxVolume)
}

func TestMasterVolumeEdgeCases(t *testing.T) {
	spec := DefaultAudioSpec()
	spec.SampleRate = 48000
	spec.BufferSize = 256
	
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Get the main mixer node
	mixerNode, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}

	// Test volume above maximum (should be clamped to 1.0 or return error)
	err = engine.SetMixerVolume(mixerNode, 1.5)
	if err == nil {
		// If it doesn't error, check that it was clamped
		volume, getErr := engine.GetMixerVolume(mixerNode)
		if getErr != nil {
			t.Fatalf("Failed to get volume after setting 1.5: %v", getErr)
		}
		if volume > 1.0 {
			t.Errorf("Volume was not clamped: expected <= 1.0, got %f", volume)
		}
		t.Logf("Volume 1.5 was handled appropriately: %f", volume)
	} else {
		t.Logf("Setting volume 1.5 returned expected error: %v", err)
	}

	// Test negative volume (should be clamped to 0.0 or return error)  
	err = engine.SetMixerVolume(mixerNode, -0.5)
	if err == nil {
		// If it doesn't error, check that it was clamped
		volume, getErr := engine.GetMixerVolume(mixerNode)
		if getErr != nil {
			t.Fatalf("Failed to get volume after setting -0.5: %v", getErr)
		}
		if volume < 0.0 {
			t.Errorf("Volume was not clamped: expected >= 0.0, got %f", volume)
		}
		t.Logf("Volume -0.5 was handled appropriately: %f", volume)
	} else {
		t.Logf("Setting volume -0.5 returned expected error: %v", err)
	}
}
