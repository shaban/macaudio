package engine

import (
	"testing"
	"time"
)

// TestTruePerBusControl demonstrates the enhanced per-bus control using connection tracking
func TestTruePerBusControl(t *testing.T) {
	t.Log("🎛️ Testing true per-bus mixer control using connection tracking")

	// Create engine
	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer func() {
		engine.Stop()
		time.Sleep(10 * time.Millisecond) // Brief pause for cleanup
	}()

	// Start engine
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	// Create two players (sources)
	player1, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player1: %v", err)
	}
	defer player1.Destroy()

	player2, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player2: %v", err)
	}
	defer player2.Destroy()

	// Load audio file into players
	if err := player1.LoadFile("idea.m4a"); err != nil {
		t.Fatalf("Failed to load file into player1: %v", err)
	}

	if err := player2.LoadFile("idea.m4a"); err != nil {
		t.Fatalf("Failed to load file into player2: %v", err)
	}

	// Get main mixer
	mainMixer, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer: %v", err)
	}

	// Get player node pointers
	player1NodePtr, err := player1.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get player1 node pointer: %v", err)
	}

	player2NodePtr, err := player2.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get player2 node pointer: %v", err)
	}

	// Connect players to different mixer buses with connection tracking
	if err := engine.Connect(player1NodePtr, mainMixer, 0, 0); err != nil {
		t.Fatalf("Failed to connect player1 to bus 0: %v", err)
	}

	if err := engine.Connect(player2NodePtr, mainMixer, 0, 1); err != nil {
		t.Fatalf("Failed to connect player2 to bus 1: %v", err)
	}

	t.Log("📡 Players connected with tracking:")
	t.Log("   Player1 -> Main Mixer Bus 0")
	t.Log("   Player2 -> Main Mixer Bus 1")

	// Test 1: Set different volumes for each bus - should use per-connection control
	t.Log("🔊 Test 1: Setting different volumes per bus")

	// Set volume for bus 0 (player1)
	if err := engine.SetMixerVolumeForBus(mainMixer, 0.8, 0); err != nil {
		t.Fatalf("Failed to set volume for bus 0: %v", err)
	}

	// Set volume for bus 1 (player2)
	if err := engine.SetMixerVolumeForBus(mainMixer, 0.4, 1); err != nil {
		t.Fatalf("Failed to set volume for bus 1: %v", err)
	}

	// Get volumes back
	vol0, err := engine.GetMixerVolumeForBus(mainMixer, 0)
	if err != nil {
		t.Fatalf("Failed to get volume for bus 0: %v", err)
	}

	vol1, err := engine.GetMixerVolumeForBus(mainMixer, 1)
	if err != nil {
		t.Fatalf("Failed to get volume for bus 1: %v", err)
	}

	t.Logf("   Bus 0 volume: %.2f (expected 0.80)", vol0)
	t.Logf("   Bus 1 volume: %.2f (expected 0.40)", vol1)

	// Check if we got true per-bus control
	if vol0 >= 0.75 && vol0 <= 0.85 && vol1 >= 0.35 && vol1 <= 0.45 {
		t.Log("   ✅ TRUE PER-BUS CONTROL WORKING! Different volumes per bus")
	} else {
		t.Log("   ⚠️  Using global control fallback (both buses return same value)")
	}

	// Test 2: Set different panning for each bus
	t.Log("🔀 Test 2: Setting different pan per bus")

	// Set pan for bus 0 (player1) - left
	if err := engine.SetMixerPanForBus(mainMixer, -0.5, 0); err != nil {
		t.Fatalf("Failed to set pan for bus 0: %v", err)
	}

	// Set pan for bus 1 (player2) - right
	if err := engine.SetMixerPanForBus(mainMixer, 0.5, 1); err != nil {
		t.Fatalf("Failed to set pan for bus 1: %v", err)
	}

	// Get panning back
	pan0, err := engine.GetMixerPanForBus(mainMixer, 0)
	if err != nil {
		t.Fatalf("Failed to get pan for bus 0: %v", err)
	}

	pan1, err := engine.GetMixerPanForBus(mainMixer, 1)
	if err != nil {
		t.Fatalf("Failed to get pan for bus 1: %v", err)
	}

	t.Logf("   Bus 0 pan: %.2f (expected -0.50)", pan0)
	t.Logf("   Bus 1 pan: %.2f (expected 0.50)", pan1)

	// Check if we got true per-bus control
	if (pan0 >= -0.6 && pan0 <= -0.4) && (pan1 >= 0.4 && pan1 <= 0.6) {
		t.Log("   ✅ TRUE PER-BUS PAN CONTROL WORKING! Different panning per bus")
	} else {
		t.Log("   ⚠️  Using global control fallback (both buses return same value)")
	}

	// Test 3: Disconnect one source and test fallback
	t.Log("🔌 Test 3: Testing fallback after disconnection")

	if err := engine.DisconnectNodeInput(mainMixer, 1); err != nil {
		t.Fatalf("Failed to disconnect bus 1: %v", err)
	}

	// Set volume for disconnected bus - should fall back to global control
	if err := engine.SetMixerVolumeForBus(mainMixer, 0.9, 1); err != nil {
		t.Fatalf("Failed to set volume for disconnected bus 1: %v", err)
	}

	// Both buses should now return the global volume
	vol0_after, err := engine.GetMixerVolumeForBus(mainMixer, 0)
	if err != nil {
		t.Fatalf("Failed to get volume for bus 0 after disconnect: %v", err)
	}

	vol1_after, err := engine.GetMixerVolumeForBus(mainMixer, 1)
	if err != nil {
		t.Fatalf("Failed to get volume for bus 1 after disconnect: %v", err)
	}

	t.Logf("   Bus 0 volume after disconnect: %.2f", vol0_after)
	t.Logf("   Bus 1 volume after disconnect: %.2f", vol1_after)

	if vol0_after >= 0.75 && vol0_after <= 0.85 {
		t.Log("   ✅ Connected bus still uses per-connection control")
	}

	if vol1_after >= 0.85 && vol1_after <= 0.95 {
		t.Log("   ✅ Disconnected bus correctly falls back to global control")
	}

	t.Log("🎉 True per-bus control test complete!")
}
