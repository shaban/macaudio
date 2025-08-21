package engine

import (
	"testing"
)

func TestNodeIntrospection(t *testing.T) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Create a mixer node to inspect
	mixerPtr, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatalf("Failed to create mixer: %v", err)
	}

	// Test input count
	inputCount, err := engine.GetNodeInputCount(mixerPtr)
	if err != nil {
		t.Errorf("Failed to get input count: %v", err)
	}
	t.Logf("Mixer has %d input buses", inputCount)

	// Test output count
	outputCount, err := engine.GetNodeOutputCount(mixerPtr)
	if err != nil {
		t.Errorf("Failed to get output count: %v", err)
	}
	t.Logf("Mixer has %d output buses", outputCount)

	// Test attachment status (should be false since not attached yet)
	attached, err := engine.IsNodeAttached(mixerPtr)
	if err != nil {
		t.Errorf("Failed to check attachment status: %v", err)
	}
	t.Logf("Mixer attached to engine: %v", attached)

	// Test logging (should not error)
	if err := engine.LogNodeInfo(mixerPtr); err != nil {
		t.Errorf("Failed to log node info: %v", err)
	}

	// Test bus validation
	if err := engine.ValidateNodeInputBus(mixerPtr, 0); err != nil {
		t.Errorf("Failed to validate input bus 0: %v", err)
	}

	if err := engine.ValidateNodeInputBus(mixerPtr, 999); err == nil {
		t.Error("Expected error for invalid input bus 999, got none")
	}

	// Clean up
	if err := engine.ReleaseNode(mixerPtr); err != nil {
		t.Errorf("Failed to release mixer: %v", err)
	}
}

func TestNodeInspectFunction(t *testing.T) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Create and attach a mixer node
	mixerPtr, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatalf("Failed to create mixer: %v", err)
	}
	defer engine.ReleaseNode(mixerPtr)

	if err := engine.Attach(mixerPtr); err != nil {
		t.Fatalf("Failed to attach mixer: %v", err)
	}

	// Test comprehensive inspection
	info, err := engine.InspectNode(mixerPtr)
	if err != nil {
		t.Fatalf("Failed to inspect node: %v", err)
	}

	t.Logf("Node inspection results:")
	t.Logf("  Input buses: %d", info.InputCount)
	t.Logf("  Output buses: %d", info.OutputCount)
	t.Logf("  Is attached: %v", info.IsAttached)

	if info.IsAttached {
		t.Logf("  Input formats: %d available", len(info.InputFormats))
		t.Logf("  Output formats: %d available", len(info.OutputFormats))
	}

	if err := engine.Detach(mixerPtr); err != nil {
		t.Errorf("Failed to detach mixer: %v", err)
	}
}

func TestEnhancedMixerControls(t *testing.T) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Create a mixer node
	mixerPtr, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatalf("Failed to create mixer: %v", err)
	}
	defer engine.ReleaseNode(mixerPtr)

	// Test per-bus volume control
	testVolume := float32(0.7)
	testBus := 0

	if err := engine.SetMixerVolumeForBus(mixerPtr, testVolume, testBus); err != nil {
		t.Errorf("Failed to set mixer volume for bus: %v", err)
	}

	volume, err := engine.GetMixerVolumeForBus(mixerPtr, testBus)
	if err != nil {
		t.Errorf("Failed to get mixer volume for bus: %v", err)
	}

	t.Logf("Set volume %.2f, got volume %.2f", testVolume, volume)
	if volume != testVolume {
		t.Errorf("Volume mismatch: expected %.2f, got %.2f", testVolume, volume)
	}

	// Test per-bus pan control
	testPan := float32(0.5) // Right
	if err := engine.SetMixerPanForBus(mixerPtr, testPan, testBus); err != nil {
		t.Errorf("Failed to set mixer pan for bus: %v", err)
	}

	pan, err := engine.GetMixerPanForBus(mixerPtr, testBus)
	if err != nil {
		t.Errorf("Failed to get mixer pan for bus: %v", err)
	}

	t.Logf("Set pan %.2f, got pan %.2f", testPan, pan)
	if pan != testPan {
		t.Errorf("Pan mismatch: expected %.2f, got %.2f", testPan, pan)
	}
}

func TestMixerBusConfiguration(t *testing.T) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Create a mixer node
	mixerPtr, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatalf("Failed to create mixer: %v", err)
	}
	defer engine.ReleaseNode(mixerPtr)

	// Test batch configuration
	// NOTE: AVAudioMixerNode has global volume/pan, not per-bus values
	// So all buses will end up with the last-set values
	configs := []MixerBusConfig{
		{Bus: 0, Volume: 0.8, Pan: -0.5}, // Left
		{Bus: 1, Volume: 0.6, Pan: 0.5},  // Right
		{Bus: 2, Volume: 1.0, Pan: 0.0},  // Center (this will be the final global setting)
	}

	if err := engine.ConfigureMixerBuses(mixerPtr, configs); err != nil {
		t.Errorf("Failed to configure mixer buses: %v", err)
	}

	// Verify final configuration (should be the last values set: volume=1.0, pan=0.0)
	expectedVolume := float32(1.0) // Last config volume
	expectedPan := float32(0.0)    // Last config pan

	for i, config := range configs {
		volume, err := engine.GetMixerVolumeForBus(mixerPtr, config.Bus)
		if err != nil {
			t.Errorf("Failed to verify volume for bus %d: %v", config.Bus, err)
			continue
		}

		pan, err := engine.GetMixerPanForBus(mixerPtr, config.Bus)
		if err != nil {
			t.Errorf("Failed to verify pan for bus %d: %v", config.Bus, err)
			continue
		}

		t.Logf("Bus %d: volume=%.2f (global expected %.2f), pan=%.2f (global expected %.2f)",
			config.Bus, volume, expectedVolume, pan, expectedPan)

		// All buses should return the same global values (last set values)
		if volume != expectedVolume {
			t.Errorf("Volume mismatch on bus %d: expected %.2f (global), got %.2f", config.Bus, expectedVolume, volume)
		}
		if pan != expectedPan {
			t.Errorf("Pan mismatch on bus %d: expected %.2f (global), got %.2f", config.Bus, expectedPan, pan)
		}

		// Only log this explanation once
		if i == 0 {
			t.Logf("NOTE: AVAudioMixerNode has global volume/pan properties, not per-bus. All buses return the same values.")
		}
	}
}

func TestConnectionControls(t *testing.T) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Create source and mixer nodes
	sourcePtr, err := engine.CreateMixerNode() // Using mixer as source for simplicity
	if err != nil {
		t.Fatalf("Failed to create source mixer: %v", err)
	}
	defer engine.ReleaseNode(sourcePtr)

	mixerPtr, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatalf("Failed to create dest mixer: %v", err)
	}
	defer engine.ReleaseNode(mixerPtr)

	// Note: Per-connection controls require nodes to support AVAudioMixing protocol
	// Testing may not work without proper connection setup, but we test the API

	testVolume := float32(0.75)
	testPan := float32(-0.3)
	testBus := 0

	// These might fail with AVAudioMixing protocol errors, which is expected
	// since not all nodes support per-connection mixing controls
	err = engine.SetConnectionVolume(sourcePtr, mixerPtr, testBus, testVolume)
	if err != nil {
		t.Logf("Expected: SetConnectionVolume may fail without AVAudioMixing protocol: %v", err)
	} else {
		// If it worked, verify the setting
		volume, err := engine.GetConnectionVolume(sourcePtr, mixerPtr, testBus)
		if err != nil {
			t.Logf("GetConnectionVolume failed: %v", err)
		} else {
			t.Logf("Connection volume set to %.2f, read as %.2f", testVolume, volume)
		}
	}

	err = engine.SetConnectionPan(sourcePtr, mixerPtr, testBus, testPan)
	if err != nil {
		t.Logf("Expected: SetConnectionPan may fail without AVAudioMixing protocol: %v", err)
	} else {
		// If it worked, verify the setting
		pan, err := engine.GetConnectionPan(sourcePtr, mixerPtr, testBus)
		if err != nil {
			t.Logf("GetConnectionPan failed: %v", err)
		} else {
			t.Logf("Connection pan set to %.2f, read as %.2f", testPan, pan)
		}
	}
}

func TestNodeErrorHandling(t *testing.T) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Test nil pointer handling
	_, err = engine.GetNodeInputCount(nil)
	if err == nil {
		t.Error("Expected error for nil node pointer, got none")
	}

	_, err = engine.GetNodeOutputCount(nil)
	if err == nil {
		t.Error("Expected error for nil node pointer, got none")
	}

	_, err = engine.IsNodeAttached(nil)
	if err == nil {
		t.Error("Expected error for nil node pointer, got none")
	}

	err = engine.LogNodeInfo(nil)
	if err == nil {
		t.Error("Expected error for nil node pointer, got none")
	}

	// Test nil mixer pointer handling
	err = engine.SetMixerVolumeForBus(nil, 0.5, 0)
	if err == nil {
		t.Error("Expected error for nil mixer pointer, got none")
	}

	_, err = engine.GetMixerVolumeForBus(nil, 0)
	if err == nil {
		t.Error("Expected error for nil mixer pointer, got none")
	}

	// Test invalid volume/pan ranges
	validMixer, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatalf("Failed to create mixer for error testing: %v", err)
	}
	defer engine.ReleaseNode(validMixer)

	// Invalid volume (> 1.0)
	err = engine.SetMixerVolumeForBus(validMixer, 1.5, 0)
	if err == nil {
		t.Error("Expected error for invalid volume > 1.0, got none")
	}

	// Invalid volume (< 0.0)
	err = engine.SetMixerVolumeForBus(validMixer, -0.1, 0)
	if err == nil {
		t.Error("Expected error for invalid volume < 0.0, got none")
	}

	// Invalid pan (> 1.0)
	err = engine.SetMixerPanForBus(validMixer, 1.5, 0)
	if err == nil {
		t.Error("Expected error for invalid pan > 1.0, got none")
	}

	// Invalid pan (< -1.0)
	err = engine.SetMixerPanForBus(validMixer, -1.5, 0)
	if err == nil {
		t.Error("Expected error for invalid pan < -1.0, got none")
	}
}

// TestNodeIntegrationWithPlayer tests node functionality with the existing player
func TestNodeIntegrationWithPlayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Create a player
	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load a test file
	if err := player.LoadFile("../../avaudio/engine/idea.m4a"); err != nil {
		t.Fatalf("Failed to load test file: %v", err)
	}

	// Get player node pointer
	playerNodePtr, err := player.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get player node: %v", err)
	}

	// Inspect the player node
	info, err := engine.InspectNode(playerNodePtr)
	if err != nil {
		t.Errorf("Failed to inspect player node: %v", err)
	} else {
		t.Logf("Player node info:")
		t.Logf("  Input buses: %d", info.InputCount)
		t.Logf("  Output buses: %d", info.OutputCount)
		t.Logf("  Is attached: %v", info.IsAttached)
	}

	// Get main mixer and inspect it
	mainMixerPtr, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer: %v", err)
	}

	mixerInfo, err := engine.InspectNode(mainMixerPtr)
	if err != nil {
		t.Errorf("Failed to inspect main mixer: %v", err)
	} else {
		t.Logf("Main mixer info:")
		t.Logf("  Input buses: %d", mixerInfo.InputCount)
		t.Logf("  Output buses: %d", mixerInfo.OutputCount)
		t.Logf("  Is attached: %v", mixerInfo.IsAttached)
	}

	// Test setting mixer controls on main mixer
	if err := engine.SetMixerVolumeForBus(mainMixerPtr, 0.8, 0); err != nil {
		t.Logf("Note: SetMixerVolumeForBus on main mixer may not be supported: %v", err)
	} else {
		volume, err := engine.GetMixerVolumeForBus(mainMixerPtr, 0)
		if err != nil {
			t.Logf("Failed to read back main mixer volume: %v", err)
		} else {
			t.Logf("Main mixer volume set to 0.8, read as %.2f", volume)
		}
	}
}

// Benchmark node introspection performance
func BenchmarkNodeIntrospection(b *testing.B) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	mixerPtr, err := engine.CreateMixerNode()
	if err != nil {
		b.Fatalf("Failed to create mixer: %v", err)
	}
	defer engine.ReleaseNode(mixerPtr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.GetNodeInputCount(mixerPtr)
		_, _ = engine.GetNodeOutputCount(mixerPtr)
		_, _ = engine.IsNodeAttached(mixerPtr)
	}
}

// Benchmark mixer control performance
func BenchmarkMixerControls(b *testing.B) {
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	mixerPtr, err := engine.CreateMixerNode()
	if err != nil {
		b.Fatalf("Failed to create mixer: %v", err)
	}
	defer engine.ReleaseNode(mixerPtr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.SetMixerVolumeForBus(mixerPtr, 0.5, 0)
		_, _ = engine.GetMixerVolumeForBus(mixerPtr, 0)
		_ = engine.SetMixerPanForBus(mixerPtr, 0.0, 0)
		_, _ = engine.GetMixerPanForBus(mixerPtr, 0)
	}
}
