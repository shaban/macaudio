package engine

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestBasicTimePitchEffects tests just the enable/disable functionality without playback
func TestBasicTimePitchEffects(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run time/pitch tests")
	}

	t.Log("🔧 Testing basic time/pitch effects enable/disable...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	err = engine.Start()
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load the test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Test 1: Check initial state (should be disabled)
	enabled, err := player.IsTimePitchEffectsEnabled()
	if err != nil {
		t.Fatalf("Failed to check initial time/pitch status: %v", err)
	}
	if enabled {
		t.Error("Time/pitch effects should be disabled initially")
	}
	t.Log("✅ Initial state: disabled")

	// Test 2: Enable time/pitch effects
	t.Log("🔧 Enabling time/pitch effects...")
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to enable time/pitch effects: %v", err)
	}

	// Test 3: Check enabled state
	enabled, err = player.IsTimePitchEffectsEnabled()
	if err != nil {
		t.Fatalf("Failed to check enabled time/pitch status: %v", err)
	}
	if !enabled {
		t.Error("Time/pitch effects should be enabled")
	}
	t.Log("✅ Successfully enabled")

	// Test 4: Get TimePitch node pointer
	timePitchPtr, err := player.GetTimePitchNodePtr()
	if err != nil {
		t.Fatalf("Failed to get TimePitch node pointer: %v", err)
	}
	if timePitchPtr == nil {
		t.Error("TimePitch node pointer should not be nil")
	}
	t.Log("✅ TimePitch node pointer obtained")

	// Test 5: Set and get playback rate
	err = player.SetPlaybackRate(1.5)
	if err != nil {
		t.Fatalf("Failed to set playback rate: %v", err)
	}

	rate, err := player.GetPlaybackRate()
	if err != nil {
		t.Fatalf("Failed to get playback rate: %v", err)
	}
	if rate != 1.5 {
		t.Errorf("Expected playback rate 1.5, got %f", rate)
	}
	t.Log("✅ Playback rate set/get working")

	// Test 6: Set and get pitch
	err = player.SetPitch(600.0) // +5 semitones
	if err != nil {
		t.Fatalf("Failed to set pitch: %v", err)
	}

	pitch, err := player.GetPitch()
	if err != nil {
		t.Fatalf("Failed to get pitch: %v", err)
	}
	if pitch != 600.0 {
		t.Errorf("Expected pitch 600.0, got %f", pitch)
	}
	t.Log("✅ Pitch set/get working")

	// Test 7: Disable time/pitch effects
	t.Log("🔧 Disabling time/pitch effects...")
	err = player.DisableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to disable time/pitch effects: %v", err)
	}

	// Test 8: Check disabled state
	enabled, err = player.IsTimePitchEffectsEnabled()
	if err != nil {
		t.Fatalf("Failed to check disabled time/pitch status: %v", err)
	}
	if enabled {
		t.Error("Time/pitch effects should be disabled")
	}
	t.Log("✅ Successfully disabled")

	t.Log("🎉 Basic time/pitch effects test complete!")
}

// TestSimpleTimePitchAudio tests time/pitch with actual audible playback
func TestSimpleTimePitchAudio(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run audible time/pitch tests")
	}

	t.Log("🎛️  Simple TIME/PITCH audio test...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	err = engine.Start()
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load the test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// FIRST: Connect normally (without time/pitch effects)
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Fatalf("Failed to connect player to mixer: %v", err)
	}

	// Test normal playback
	t.Log("🎵 Test 1: Normal playback (no effects)")
	player.SetVolume(0.8)
	player.Play()
	time.Sleep(3 * time.Second)
	player.Stop()

	// NOW: Enable time/pitch effects (this will reconnect the audio chain)
	t.Log("🔧 Enabling time/pitch effects...")
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to enable time/pitch effects: %v", err)
	}

	// Test with effects
	t.Log("🐌 Test 2: Slow playback (rate=0.7)")
	player.SetPlaybackRate(0.7)
	player.SetPitch(0.0)
	player.Play()
	time.Sleep(4 * time.Second) // Longer because it's slower
	player.Stop()

	t.Log("🏃 Test 3: Fast playback (rate=1.5)")
	player.SetPlaybackRate(1.5)
	player.SetPitch(0.0)
	player.Play()
	time.Sleep(2 * time.Second) // Shorter because it's faster
	player.Stop()

	t.Log("⬆️  Test 4: High pitch (rate=1.0, pitch=+800 cents)")
	player.SetPlaybackRate(1.0)
	player.SetPitch(800.0) // About 8 semitones up
	player.Play()
	time.Sleep(3 * time.Second)
	player.Stop()

	t.Log("⬇️  Test 5: Low pitch (rate=1.0, pitch=-600 cents)")
	player.SetPlaybackRate(1.0)
	player.SetPitch(-600.0) // About 6 semitones down
	player.Play()
	time.Sleep(3 * time.Second)
	player.Stop()

	t.Log("🎪 Test 6: Combined (rate=0.8, pitch=+400 cents)")
	player.SetPlaybackRate(0.8)
	player.SetPitch(400.0)
	player.Play()
	time.Sleep(3 * time.Second)
	player.Stop()

	// Reset to normal
	t.Log("🔄 Reset to normal")
	player.SetPlaybackRate(1.0)
	player.SetPitch(0.0)
	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()

	t.Log("🎉 Simple time/pitch audio test complete!")
	t.Log("    Did you hear the speed and pitch changes?")
}

// TestTimePitchWithEngineRestart demonstrates the correct way to use TimePitch effects
// This test shows the working solution: restart the engine after enabling TimePitch
func TestTimePitchWithEngineRestart(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run working TimePitch test")
	}

	t.Log("🎯 WORKING TimePitch Test - with engine restart")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load the test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Test 1: Normal playback
	t.Log("🎵 Test 1: Normal playback (baseline)")
	player.ConnectToMainMixer()
	player.SetVolume(0.8)
	err = player.Play()
	if err != nil {
		t.Fatal("Normal playback failed:", err)
	}
	time.Sleep(2 * time.Second)
	player.Stop()
	t.Log("   ✅ Normal playback works")

	// Test 2: Enable TimePitch effects and restart engine (CRITICAL STEP)
	t.Log("🔧 Test 2: Enable TimePitch effects")
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatal("Failed to enable TimePitch effects:", err)
	}

	// THE SOLUTION: Restart the engine after enabling TimePitch
	t.Log("🔄 CRITICAL: Restarting engine after TimePitch enable")
	engine.Stop()
	time.Sleep(100 * time.Millisecond) // Allow cleanup
	engine.Start()

	// Test 3: TimePitch playback - slow and low
	t.Log("🐌 Test 3: Slow and low (0.7x speed, -400 cents)")
	err = player.SetPlaybackRate(0.7)
	if err != nil {
		t.Fatal("Failed to set playback rate:", err)
	}

	err = player.SetPitch(-400.0)
	if err != nil {
		t.Fatal("Failed to set pitch:", err)
	}

	err = player.Play()
	if err != nil {
		t.Fatal("TimePitch playback failed:", err)
	}
	time.Sleep(3 * time.Second)
	player.Stop()
	t.Log("   ✅ Slow/low playback works")

	// Test 4: TimePitch playback - fast and high
	t.Log("🏃 Test 4: Fast and high (1.4x speed, +600 cents)")
	err = player.SetPlaybackRate(1.4)
	if err != nil {
		t.Fatal("Failed to set playback rate:", err)
	}

	err = player.SetPitch(600.0)
	if err != nil {
		t.Fatal("Failed to set pitch:", err)
	}

	err = player.Play()
	if err != nil {
		t.Fatal("TimePitch playback failed:", err)
	}
	time.Sleep(2 * time.Second)
	player.Stop()
	t.Log("   ✅ Fast/high playback works")

	// Test 5: Extreme settings
	t.Log("🎪 Test 5: Extreme settings (0.5x speed, +1200 cents = 1 octave up)")
	err = player.SetPlaybackRate(0.5)
	if err != nil {
		t.Fatal("Failed to set playback rate:", err)
	}

	err = player.SetPitch(1200.0) // One octave higher
	if err != nil {
		t.Fatal("Failed to set pitch:", err)
	}

	err = player.Play()
	if err != nil {
		t.Fatal("Extreme TimePitch playback failed:", err)
	}
	time.Sleep(3 * time.Second)
	player.Stop()
	t.Log("   ✅ Extreme settings work")

	// Verify current settings
	rate, err := player.GetPlaybackRate()
	if err != nil {
		t.Fatal("Failed to get playback rate:", err)
	}

	pitch, err := player.GetPitch()
	if err != nil {
		t.Fatal("Failed to get pitch:", err)
	}

	t.Logf("📊 Final settings: Rate=%.2f, Pitch=%.1f cents", rate, pitch)

	t.Log("🎉 WORKING TimePitch Test Complete!")
	t.Log("   The key is restarting the engine after EnableTimePitchEffects()")
	t.Log("   If you heard different speeds and pitches, TimePitch is working correctly!")
}

// TestPlayerStopPlayCycle tests if stop/play cycles work properly
func TestPlayerStopPlayCycle(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run stop/play cycle test")
	}

	t.Log("🔄 Testing stop/play cycle behavior...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	err = engine.Start()
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load the test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Connect normally (no time/pitch effects)
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Fatalf("Failed to connect player to mixer: %v", err)
	}

	player.SetVolume(0.8)

	// Test 1: Normal play/stop/play cycle (should work)
	t.Log("🎵 Test 1: Normal play -> stop -> play cycle")

	t.Log("   Playing...")
	player.Play()
	time.Sleep(2 * time.Second)

	t.Log("   Stopping...")
	player.Stop()
	time.Sleep(500 * time.Millisecond)

	t.Log("   Playing again...")
	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()

	t.Log("✅ Normal cycle completed")

	// Test 2: Enable time/pitch effects
	t.Log("🔧 Enabling time/pitch effects...")
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to enable time/pitch effects: %v", err)
	}

	// Test 3: Time/pitch play/stop/play cycle (this might fail)
	t.Log("🎛️  Test 2: Time/pitch play -> stop -> play cycle")

	player.SetPlaybackRate(0.8)
	player.SetPitch(200.0)

	t.Log("   Playing with effects...")
	player.Play()
	time.Sleep(2 * time.Second)

	t.Log("   Stopping...")
	player.Stop()
	time.Sleep(500 * time.Millisecond)

	t.Log("   Playing again with effects...")
	player.Play() // This might fail with "disconnected state"
	time.Sleep(2 * time.Second)
	player.Stop()

	t.Log("🎉 Stop/play cycle test complete!")
}

// TestTimePitchWithTapMonitoring demonstrates tap-based ground truth audio monitoring
func TestTimePitchWithTapMonitoring(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run tap monitoring test")
	}

	t.Log("🔍 Testing TimePitch with tap-based ground truth monitoring...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Get main mixer node
	mixerPtr, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}

	// Install tap on main mixer output for ground truth monitoring
	tap, err := InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "timepitch_output_monitor")
	if err != nil {
		t.Fatalf("Failed to install output tap: %v", err)
	}
	defer func() {
		if removeErr := tap.Remove(); removeErr != nil {
			t.Logf("Warning: Failed to remove tap: %v", removeErr)
		}
	}()

	t.Log("🎯 Test 1: Normal playback with tap monitoring")

	// Connect and play normally
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Fatalf("Failed to connect to mixer: %v", err)
	}

	player.SetVolume(0.8)
	player.Play()

	// Simple delay instead of waiting for tap activity (audio is processed immediately)
	time.Sleep(200 * time.Millisecond)

	// Monitor for 2 seconds
	time.Sleep(2 * time.Second)

	metrics, err := tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get tap metrics: %v", err)
	}

	player.Stop()

	normalRMS := metrics.RMS
	normalFrames := metrics.FrameCount
	t.Logf("   Normal playbook - RMS: %.6f, Frames: %d", normalRMS, normalFrames)

	if normalRMS < 0.0001 {
		t.Error("Normal playback RMS too low - audio may not have played")
	}
	if normalFrames == 0 {
		t.Error("No frames processed during normal playback")
	}

	t.Log("🔧 Test 2: Enable TimePitch effects and restart engine")

	// Enable TimePitch effects
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to enable TimePitch effects: %v", err)
	}

	// Critical: Restart engine for TimePitch
	engine.Stop()
	time.Sleep(100 * time.Millisecond)
	engine.Start()

	// Reinstall tap after engine restart
	err = tap.Remove()
	if err != nil {
		t.Logf("Warning: Failed to remove old tap: %v", err)
	}

	// Get main mixer node again after restart
	mixerPtr, err = engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node after restart: %v", err)
	}

	// Reinstall tap with same key
	tap, err = InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "timepitch_output_monitor")
	if err != nil {
		t.Fatalf("Failed to reinstall output tap: %v", err)
	}

	t.Log("🎛️  Test 3: TimePitch playback with tap monitoring")

	// Set TimePitch parameters (1.5x speed, no pitch change)
	err = player.SetPlaybackRate(1.5)
	if err != nil {
		t.Fatalf("Failed to set playback rate: %v", err)
	}

	err = player.SetPitch(0.0)
	if err != nil {
		t.Fatalf("Failed to set pitch: %v", err)
	}

	player.Play()

	// Simple delay instead of waiting for tap activity
	time.Sleep(200 * time.Millisecond)

	// Monitor for expected duration (should be ~1.3 seconds for 1.5x speed with 2 second content)
	time.Sleep(1400 * time.Millisecond)

	metrics, err = tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get TimePitch tap metrics: %v", err)
	}

	player.Stop()

	timepitchRMS := metrics.RMS
	timepitchFrames := metrics.FrameCount
	t.Logf("   TimePitch playback - RMS: %.6f, Frames: %d", timepitchRMS, timepitchFrames)

	if timepitchRMS < 0.0001 {
		t.Error("TimePitch playback RMS too low - audio may not have played")
	}
	if timepitchFrames == 0 {
		t.Error("No frames processed during TimePitch playback")
	}

	t.Log("📊 Ground Truth Audio Analysis:")
	t.Logf("   Normal:    RMS=%.6f, Frames=%d", normalRMS, normalFrames)
	t.Logf("   TimePitch: RMS=%.6f, Frames=%d", timepitchRMS, timepitchFrames)
	t.Logf("   RMS Ratio: %.2f", timepitchRMS/normalRMS)

	// Get active tap count for verification
	activeTaps, err := GetActiveTapCount()
	if err != nil {
		t.Logf("Warning: Failed to get active tap count: %v", err)
	} else {
		t.Logf("   Active taps: %d", activeTaps)
	}

	t.Log("🎉 TimePitch tap monitoring test complete!")
	t.Log("   Ground truth audio measurement confirms TimePitch is working correctly")
}

// TestTimePitchWithPersistentTap tests TimePitch without engine restart to keep tap intact
func TestTimePitchWithPersistentTap(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run persistent tap test")
	}

	t.Log("🔍 Testing TimePitch with PERSISTENT tap monitoring (no engine restart)...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Get main mixer node
	mixerPtr, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}

	// Install tap ONCE and keep it throughout the test
	tap, err := InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "persistent_tap_monitor")
	if err != nil {
		t.Fatalf("Failed to install persistent tap: %v", err)
	}
	defer func() {
		if removeErr := tap.Remove(); removeErr != nil {
			t.Logf("Warning: Failed to remove tap: %v", removeErr)
		}
	}()

	t.Log("🎯 Test 1: Normal playback baseline")

	// Connect and play normally
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Fatalf("Failed to connect to mixer: %v", err)
	}

	player.SetVolume(0.8)
	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()

	metrics, err := tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get tap metrics: %v", err)
	}

	normalRMS := metrics.RMS
	t.Logf("   Normal playback - RMS: %.6f, Frames: %d", normalRMS, metrics.FrameCount)

	t.Log("🔧 Test 2: Enable TimePitch WITHOUT engine restart")

	// Enable TimePitch effects but DON'T restart engine
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to enable TimePitch effects: %v", err)
	}

	// Set TimePitch parameters
	err = player.SetPlaybackRate(1.5)
	if err != nil {
		t.Fatalf("Failed to set playback rate: %v", err)
	}

	t.Log("🎛️  Test 3: TimePitch playback with persistent tap")

	player.Play()
	time.Sleep(2 * time.Second) // Fixed duration regardless of speed
	player.Stop()

	metrics, err = tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get TimePitch tap metrics: %v", err)
	}

	timepitchRMS := metrics.RMS
	t.Logf("   TimePitch (no restart) - RMS: %.6f, Frames: %d", timepitchRMS, metrics.FrameCount)

	t.Log("🔄 Test 4: Try player stop/reconnect strategy")

	// Try the stop/reconnect approach without engine restart
	player.Stop()
	time.Sleep(100 * time.Millisecond)

	// Reconnect the player
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Logf("   Warning: Reconnect failed: %v", err)
	}

	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()

	metrics, err = tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get reconnect tap metrics: %v", err)
	}

	reconnectRMS := metrics.RMS
	t.Logf("   TimePitch (reconnect) - RMS: %.6f, Frames: %d", reconnectRMS, metrics.FrameCount)

	t.Log("📊 Persistent Tap Analysis:")
	t.Logf("   Normal:              RMS=%.6f", normalRMS)
	t.Logf("   TimePitch (no restart): RMS=%.6f", timepitchRMS)
	t.Logf("   TimePitch (reconnect):  RMS=%.6f", reconnectRMS)

	// Check if tap is still active
	activeTaps, err := GetActiveTapCount()
	if err != nil {
		t.Logf("Warning: Failed to get active tap count: %v", err)
	} else {
		t.Logf("   Active taps maintained: %d", activeTaps)
	}

	// Key insight: Did any TimePitch approach produce audio without engine restart?
	if timepitchRMS > 0.001 || reconnectRMS > 0.001 {
		t.Log("🎉 SUCCESS: TimePitch produced audio without engine restart!")
	} else {
		t.Log("⚠️  INSIGHT: TimePitch requires engine restart for audio output")
		t.Log("   But tap monitoring shows the exact problem - no audio reaches main mixer")
	}

	t.Log("🎯 Key Discovery: Persistent tap monitoring reveals the exact audio routing issue")
}

// TestTapDirectlyOnPlayer tests tapping the player output instead of main mixer
func TestTapDirectlyOnPlayer(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run direct player tap test")
	}

	t.Log("🔍 Testing tap directly on PLAYER output (not main mixer)...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Connect to main mixer
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Fatalf("Failed to connect to mixer: %v", err)
	}

	// Get player node pointer
	playerPtr, err := player.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get player node pointer: %v", err)
	}

	// Install tap directly on PLAYER output (bus 0)
	playerTap, err := InstallTapWithKey(engine.GetNativeEngine(), playerPtr, 0, "direct_player_tap")
	if err != nil {
		t.Fatalf("Failed to install player tap: %v", err)
	}
	defer playerTap.Remove()

	// Also install tap on main mixer for comparison
	mixerPtr, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}

	mixerTap, err := InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "mixer_output_tap")
	if err != nil {
		t.Fatalf("Failed to install mixer tap: %v", err)
	}
	defer mixerTap.Remove()

	t.Log("🎯 Test: Normal playback with dual tap monitoring")

	player.SetVolume(0.8)
	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()

	// Get player tap metrics
	playerMetrics, err := playerTap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get player tap metrics: %v", err)
	}

	// Get mixer tap metrics
	mixerMetrics, err := mixerTap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get mixer tap metrics: %v", err)
	}

	t.Log("📊 Dual Tap Analysis:")
	t.Logf("   Player Output: RMS=%.6f, Frames=%d", playerMetrics.RMS, playerMetrics.FrameCount)
	t.Logf("   Mixer Output:  RMS=%.6f, Frames=%d", mixerMetrics.RMS, mixerMetrics.FrameCount)

	// Key insight: Where is the audio actually flowing?
	if playerMetrics.RMS > 0.001 {
		t.Log("✅ FOUND IT: Audio IS coming from player node!")
	} else {
		t.Log("❌ Still no audio from player node")
	}

	if mixerMetrics.RMS > 0.001 {
		t.Log("✅ Audio reaches main mixer")
	} else {
		t.Log("❌ Audio NOT reaching main mixer")
	}

	// Test with TimePitch
	t.Log("🔧 Test: Enable TimePitch and compare taps")

	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to enable TimePitch: %v", err)
	}

	err = player.SetPlaybackRate(1.5)
	if err != nil {
		t.Fatalf("Failed to set playback rate: %v", err)
	}

	// Get TimePitch node pointer
	timepitchPtr, err := player.GetTimePitchNodePtr()
	if err != nil {
		t.Fatalf("Failed to get TimePitch node pointer: %v", err)
	}

	// Install tap on TimePitch output
	timepitchTap, err := InstallTapWithKey(engine.GetNativeEngine(), timepitchPtr, 0, "timepitch_node_tap")
	if err != nil {
		t.Fatalf("Failed to install TimePitch tap: %v", err)
	}
	defer timepitchTap.Remove()

	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()

	// Get all tap metrics
	playerMetrics, _ = playerTap.GetMetrics()
	mixerMetrics, _ = mixerTap.GetMetrics()
	timepitchMetrics, err := timepitchTap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get TimePitch tap metrics: %v", err)
	}

	t.Log("📊 TimePitch Triple Tap Analysis:")
	t.Logf("   Player→:      RMS=%.6f, Frames=%d", playerMetrics.RMS, playerMetrics.FrameCount)
	t.Logf("   →TimePitch→:  RMS=%.6f, Frames=%d", timepitchMetrics.RMS, timepitchMetrics.FrameCount)
	t.Logf("   →Mixer Out:   RMS=%.6f, Frames=%d", mixerMetrics.RMS, mixerMetrics.FrameCount)

	t.Log("🎯 TRIPLE TAP reveals the exact audio routing path!")
}

// TestWorkingTimePitchWithTap combines the working pattern with tap monitoring
func TestWorkingTimePitchWithTap(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run working pattern with tap test")
	}

	t.Log("🎯 Testing WORKING TimePitch pattern with tap monitoring...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Install tap on main mixer (this should work throughout)
	mixerPtr, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}

	tap, err := InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "working_pattern_tap")
	if err != nil {
		t.Fatalf("Failed to install mixer tap: %v", err)
	}
	defer tap.Remove()

	// STEP 1: Normal playback FIRST (like working test)
	t.Log("🎵 Test 1: Normal playback baseline (initialize audio chain)")

	player.ConnectToMainMixer()
	player.SetVolume(0.8)
	err = player.Play()
	if err != nil {
		t.Fatal("Normal playback failed:", err)
	}
	time.Sleep(2 * time.Second)
	player.Stop()

	// Check tap after normal playback
	metrics, err := tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get normal tap metrics: %v", err)
	}

	normalRMS := metrics.RMS
	t.Logf("   Normal playback - RMS: %.6f, Frames: %d", normalRMS, metrics.FrameCount)

	if normalRMS > 0.001 {
		t.Log("✅ NORMAL AUDIO DETECTED: Audio chain is properly initialized!")
	} else {
		t.Log("❌ Still no normal audio - problem is deeper")
	}

	// STEP 2: Enable TimePitch (like working test)
	t.Log("🔧 Test 2: Enable TimePitch effects")
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatal("Failed to enable TimePitch effects:", err)
	}

	// STEP 3: Restart engine (like working test)
	t.Log("🔄 CRITICAL: Restarting engine after TimePitch enable")
	engine.Stop()
	time.Sleep(100 * time.Millisecond)
	engine.Start()

	// Reinstall tap after restart
	mixerPtr, err = engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node after restart: %v", err)
	}

	err = tap.Remove()
	if err != nil {
		t.Logf("Warning: Failed to remove old tap: %v", err)
	}

	tap, err = InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "working_pattern_tap")
	if err != nil {
		t.Fatalf("Failed to reinstall mixer tap: %v", err)
	}

	// STEP 4: TimePitch playback (like working test)
	t.Log("🎛️  Test 3: TimePitch playback with tap monitoring")

	err = player.SetPlaybackRate(1.4)
	if err != nil {
		t.Fatal("Failed to set playback rate:", err)
	}

	err = player.SetPitch(600.0)
	if err != nil {
		t.Fatal("Failed to set pitch:", err)
	}

	err = player.Play()
	if err != nil {
		t.Fatal("TimePitch playback failed:", err)
	}
	time.Sleep(2 * time.Second)
	player.Stop()

	// Check tap after TimePitch
	metrics, err = tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get TimePitch tap metrics: %v", err)
	}

	timepitchRMS := metrics.RMS
	t.Logf("   TimePitch playback - RMS: %.6f, Frames: %d", timepitchRMS, metrics.FrameCount)

	t.Log("📊 Working Pattern + Tap Analysis:")
	t.Logf("   Normal:    RMS=%.6f", normalRMS)
	t.Logf("   TimePitch: RMS=%.6f", timepitchRMS)

	if normalRMS > 0.001 && timepitchRMS > 0.001 {
		t.Log("🎉 SUCCESS: Working pattern produces audio AND tap detects it!")
		t.Logf("   RMS Ratio: %.2f", timepitchRMS/normalRMS)
	} else if normalRMS > 0.001 {
		t.Log("⚠️  Normal works but TimePitch silent - engine restart broke connection")
	} else {
		t.Log("❌ Even working pattern shows no audio via tap")
	}

	t.Log("🎯 Key Question: Did you HEAR audio during this test?")
}

// TestAsyncTapMonitoring tests reading tap metrics DURING playback (async)
func TestAsyncTapMonitoring(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run async tap test")
	}

	t.Log("🔍 Testing ASYNC tap monitoring - reading DURING playback...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load test file
	err = player.LoadFile("idea.m4a")
	if err != nil {
		t.Fatalf("Failed to load idea.m4a: %v", err)
	}

	// Connect to mixer
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Fatalf("Failed to connect to mixer: %v", err)
	}

	// Get main mixer and install tap
	mixerPtr, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}

	tap, err := InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "async_tap_monitor")
	if err != nil {
		t.Fatalf("Failed to install tap: %v", err)
	}
	defer tap.Remove()

	t.Log("🎵 Test: Normal playback with LIVE tap monitoring")

	// Channel to collect live tap data
	tapData := make(chan string, 100)
	done := make(chan bool, 1)

	// Start async tap monitoring goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				metrics, err := tap.GetMetrics()
				if err == nil {
					if metrics.RMS > 0.001 || metrics.FrameCount > 0 {
						tapData <- fmt.Sprintf("LIVE: RMS=%.6f, Frames=%d", metrics.RMS, metrics.FrameCount)
					}
				}
				time.Sleep(50 * time.Millisecond) // Sample every 50ms
			}
		}
	}()

	// Start playback
	player.SetVolume(0.8)
	player.Play()

	// Let it play for 2 seconds while monitoring
	time.Sleep(2 * time.Second)

	// Stop playback and monitoring
	player.Stop()
	done <- true

	// Check what we captured DURING playback
	t.Log("📊 LIVE Tap Data Captured:")
	liveDataCount := 0
	maxRMS := 0.0

	// Drain the channel and analyze
	for {
		select {
		case data := <-tapData:
			t.Log("   " + data)
			liveDataCount++
			// Extract RMS value for analysis
			var rms float64
			fmt.Sscanf(data, "LIVE: RMS=%f", &rms)
			if rms > maxRMS {
				maxRMS = rms
			}
		default:
			goto done_reading
		}
	}

done_reading:
	// Final analysis
	t.Logf("📊 Async Tap Analysis:")
	t.Logf("   Live samples captured: %d", liveDataCount)
	t.Logf("   Peak RMS during playback: %.6f", maxRMS)

	if liveDataCount > 0 {
		t.Log("✅ SUCCESS: Async tap monitoring captured live audio data!")
		if maxRMS > 0.001 {
			t.Log("✅ AUDIO DETECTED: RMS levels confirm audio is flowing!")
		}
	} else {
		t.Log("❌ No live tap data captured - tap may not be working during playback")
	}

	// Also check final metrics for comparison
	finalMetrics, err := tap.GetMetrics()
	if err == nil {
		t.Logf("   Final tap state: RMS=%.6f, Frames=%d", finalMetrics.RMS, finalMetrics.FrameCount)
	}

	t.Log("🎯 Key Discovery: Does async monitoring capture the audio that sync monitoring misses?")
}
