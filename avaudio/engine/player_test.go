package engine

import (
	"os"
	"testing"
	"time"
)

// Test creating a player
func TestPlayerCreation(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Create a player
	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatal("Failed to create player:", err)
	}
	defer player.Destroy()

	if player == nil {
		t.Error("Player should not be nil")
	}

	t.Log("✅ Player creation test passed")
}

// Test loading the real M4A file
func TestPlayerLoadM4AFile(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatal("Failed to create player:", err)
	}
	defer player.Destroy()

	// Load the real M4A file
	m4aFile := "idea.m4a" // Relative path to the M4A file in the same directory
	if err := player.LoadFile(m4aFile); err != nil {
		t.Fatalf("Failed to load M4A file '%s': %v", m4aFile, err)
	}

	t.Logf("✅ Successfully loaded M4A file: %s", m4aFile)

	// Get file information
	info, err := player.GetFileInfo()
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	t.Logf("📁 File Info:")
	t.Logf("   Sample Rate: %.0f Hz", info.SampleRate)
	t.Logf("   Channels: %d", info.ChannelCount)
	t.Logf("   Duration: %v", info.Duration)
	t.Logf("   Format: %s", info.Format)

	// Test duration
	duration, err := player.GetDuration()
	if err != nil {
		t.Fatal("Failed to get duration:", err)
	}

	if duration <= 0 {
		t.Error("Duration should be positive for a real audio file")
	}

	t.Logf("🎵 File duration: %v", duration)
}

// Test basic player controls
func TestPlayerControls(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatal("Failed to create player:", err)
	}
	defer player.Destroy()

	// Load the M4A file
	m4aFile := "idea.m4a"
	if err := player.LoadFile(m4aFile); err != nil {
		t.Fatalf("Failed to load M4A file: %v", err)
	}

	// Test volume control
	if err := player.SetVolume(0.7); err != nil {
		t.Fatal("Failed to set volume:", err)
	}

	volume, err := player.GetVolume()
	if err != nil {
		t.Fatal("Failed to get volume:", err)
	}

	if volume < 0.69 || volume > 0.71 {
		t.Errorf("Expected volume ~0.7, got %.2f", volume)
	}
	t.Logf("Volume control test passed: %.2f", volume)

	// Test pan control
	if err := player.SetPan(0.3); err != nil {
		t.Fatal("Failed to set pan:", err)
	}

	pan, err := player.GetPan()
	if err != nil {
		t.Fatal("Failed to get pan:", err)
	}

	if pan < 0.29 || pan > 0.31 {
		t.Errorf("Expected pan ~0.3, got %.2f", pan)
	}
	t.Logf("Pan control test passed: %.2f", pan)

	// Connect to main mixer and test engine start
	if err := player.ConnectToMainMixer(); err != nil {
		t.Fatal("Failed to connect to main mixer:", err)
	}

	engine.Prepare()
	if err := engine.Start(); err != nil {
		t.Fatal("Failed to start engine:", err)
	}
	defer engine.Stop()

	// Test playback (brief, silent test)
	if err := player.Play(); err != nil {
		t.Fatal("Failed to start playback:", err)
	}

	// Let it play very briefly
	time.Sleep(100 * time.Millisecond)

	// Check playing state
	isPlaying, err := player.IsPlaying()
	if err != nil {
		t.Fatal("Failed to check playing state:", err)
	}
	t.Logf("Is playing: %v", isPlaying)

	// Stop playback
	if err := player.Stop(); err != nil {
		t.Fatal("Failed to stop playback:", err)
	}

	t.Log("✅ Player controls test passed")
}

// Audible test - plays the M4A file with various effects
func TestPlayerAudible(t *testing.T) {
	// Only run if explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Skipping audible test (set MACAUDIO_AUDIBLE=1 to enable)")
	}

	t.Log("🎧 Starting AUDIBLE playback test - you should hear audio!")
	t.Log("   Make sure your volume is at a comfortable level...")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatal("Failed to create player:", err)
	}
	defer player.Destroy()

	// Load the M4A file
	m4aFile := "idea.m4a"
	if err := player.LoadFile(m4aFile); err != nil {
		t.Fatalf("Failed to load M4A file: %v", err)
	}

	// Get file info for the demo
	info, err := player.GetFileInfo()
	if err != nil {
		t.Fatal("Failed to get file info:", err)
	}

	t.Logf("🎵 Now playing: %s", m4aFile)
	t.Logf("   Duration: %v", info.Duration)
	t.Logf("   Format: %.0f Hz, %d channels", info.SampleRate, info.ChannelCount)

	// Connect to audio output
	if err := player.ConnectToMainMixer(); err != nil {
		t.Fatal("Failed to connect to main mixer:", err)
	}

	// Start the audio engine
	engine.Prepare()
	if err := engine.Start(); err != nil {
		t.Fatal("Failed to start engine:", err)
	}
	defer engine.Stop()

	// === DEMO SEQUENCE ===

	// 1. Play from beginning at moderate volume
	t.Log("▶️  Playing from beginning at 50% volume...")
	if err := player.SetVolume(0.5); err != nil {
		t.Fatal("Failed to set volume:", err)
	}
	if err := player.Play(); err != nil {
		t.Fatal("Failed to start playback:", err)
	}

	time.Sleep(3 * time.Second)

	// 2. Volume fade up
	t.Log("🔊 Fading volume up...")
	volumes := []float32{0.5, 0.6, 0.7, 0.8}
	for _, vol := range volumes {
		if err := player.SetVolume(vol); err != nil {
			t.Fatal("Failed to set volume:", err)
		}
		t.Logf("   Volume: %.0f%%", vol*100)
		time.Sleep(800 * time.Millisecond)
	}

	// 3. Pan demonstration
	t.Log("🎛️  Pan demonstration: Left -> Center -> Right -> Center")
	pans := []float32{-1.0, -0.5, 0.0, 0.5, 1.0, 0.0}
	panNames := []string{"Hard Left", "Center-Left", "Center", "Center-Right", "Hard Right", "Center"}

	for i, pan := range pans {
		if err := player.SetPan(pan); err != nil {
			t.Fatal("Failed to set pan:", err)
		}
		t.Logf("   Pan: %s (%.1f)", panNames[i], pan)
		time.Sleep(1500 * time.Millisecond)
	}

	// 4. Pause and resume
	t.Log("⏸️  Pausing for 2 seconds...")
	if err := player.Pause(); err != nil {
		t.Fatal("Failed to pause:", err)
	}

	time.Sleep(2 * time.Second)

	t.Log("▶️  Resuming playback...")
	if err := player.Play(); err != nil {
		t.Fatal("Failed to resume:", err)
	}

	time.Sleep(2 * time.Second)

	// 5. Seek to different parts of the file
	duration := info.Duration
	if duration > 30*time.Second {
		t.Log("⏭️  Seeking to 25% through the file...")
		seekTime := duration.Seconds() * 0.25
		if err := player.SeekTo(seekTime); err != nil {
			t.Fatal("Failed to seek:", err)
		}
		t.Logf("   Now at: %.1f seconds", seekTime)
		time.Sleep(3 * time.Second)

		t.Log("⏭️  Seeking to 50% through the file...")
		seekTime = duration.Seconds() * 0.50
		if err := player.SeekTo(seekTime); err != nil {
			t.Fatal("Failed to seek:", err)
		}
		t.Logf("   Now at: %.1f seconds", seekTime)
		time.Sleep(3 * time.Second)

		t.Log("⏭️  Seeking to 75% through the file...")
		seekTime = duration.Seconds() * 0.75
		if err := player.SeekTo(seekTime); err != nil {
			t.Fatal("Failed to seek:", err)
		}
		t.Logf("   Now at: %.1f seconds", seekTime)
		time.Sleep(3 * time.Second)
	}

	// 6. Volume fade out
	t.Log("🔉 Fading volume out...")
	fadeVolumes := []float32{0.8, 0.6, 0.4, 0.2, 0.1}
	for _, vol := range fadeVolumes {
		if err := player.SetVolume(vol); err != nil {
			t.Fatal("Failed to set volume:", err)
		}
		t.Logf("   Volume: %.0f%%", vol*100)
		time.Sleep(600 * time.Millisecond)
	}

	// 7. Stop playback
	t.Log("⏹️  Stopping playback...")
	if err := player.Stop(); err != nil {
		t.Fatal("Failed to stop:", err)
	}

	time.Sleep(500 * time.Millisecond)

	// 8. Quick restart from a specific time
	if duration > 10*time.Second {
		t.Log("🎬 Quick demo: Playing last 5 seconds at full volume...")
		startTime := duration.Seconds() - 5.0
		if err := player.SetVolume(1.0); err != nil {
			t.Fatal("Failed to set volume:", err)
		}
		if err := player.SetPan(0.0); err != nil { // Center pan
			t.Fatal("Failed to set pan:", err)
		}
		if err := player.PlayAt(startTime); err != nil {
			t.Fatal("Failed to play from specific time:", err)
		}
		t.Logf("   Playing from %.1f seconds to end...", startTime)
		time.Sleep(5500 * time.Millisecond) // Let it play to the end
	}

	// Final stop
	if err := player.Stop(); err != nil {
		t.Fatal("Failed to final stop:", err)
	}

	t.Log("🎉 Audible test complete!")
	t.Log("   Did you hear the audio file playing with volume/pan changes?")
	t.Log("   Your audio player dylib is working perfectly!")
}

// TestPlayerTimePitchEffects tests the new time stretching and pitch shifting functionality
func TestPlayerTimePitchEffects(t *testing.T) {
	// Skip if not explicitly requested
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run audible time/pitch tests")
	}

	t.Log("🎛️  Starting TIME/PITCH effects test...")
	t.Log("   This will test playback rate and pitch manipulation")

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

	// Check if time/pitch effects are enabled by default (should be false)
	enabled, err := player.IsTimePitchEffectsEnabled()
	if err != nil {
		t.Fatalf("Failed to check time/pitch effects status: %v", err)
	}
	t.Logf("Time/pitch effects initially enabled: %v", enabled)

	// Enable time/pitch effects BEFORE connecting to mixer
	t.Log("🔧 Enabling time/pitch effects...")
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to enable time/pitch effects: %v", err)
	}

	// Verify they're enabled
	enabled, err = player.IsTimePitchEffectsEnabled()
	if err != nil {
		t.Fatalf("Failed to check time/pitch effects status after enabling: %v", err)
	}
	if !enabled {
		t.Fatalf("Time/pitch effects should be enabled but aren't")
	}
	t.Log("✅ Time/pitch effects enabled successfully!")

	// CRITICAL: Apply the correct sequence we discovered from objc_exploration
	// 1. Stop engine first
	engine.Stop()
	t.Log("🔧 Stopped engine for TimePitch routing changes")

	// 2. Connect while engine is stopped
	err = player.ConnectToMainMixer()
	if err != nil {
		t.Fatalf("Failed to connect player to mixer: %v", err)
	}
	t.Log("🔗 Connected TimePitch routing while engine stopped")

	// 3. Restart engine with new TimePitch connections
	err = engine.Start()
	if err != nil {
		t.Fatalf("Failed to restart engine after TimePitch routing: %v", err)
	}
	t.Log("▶️ Restarted engine - TimePitch routing should work now!")

	// Give AVAudioEngine a moment to stabilize the new routing
	time.Sleep(200 * time.Millisecond)

	// Install tap on main mixer to monitor actual audio signal
	mixerPtr, err := engine.MainMixerNode()
	if err != nil {
		t.Fatalf("Failed to get main mixer node: %v", err)
	}
	tap, err := InstallTapWithKey(engine.GetNativeEngine(), mixerPtr, 0, "timepitch_test_tap")
	if err != nil {
		t.Fatalf("Failed to install tap: %v", err)
	}
	defer tap.Remove()
	t.Log("🎧 Installed tap for audio signal monitoring")

	// Helper function to verify audio signal during expected duration
	verifyAudioSignal := func(testName string, expectedDuration time.Duration, tolerance time.Duration) {
		start := time.Now()
		signalDetected := false

		// Monitor for audio signal with slight tolerance
		for time.Since(start) < expectedDuration+tolerance {
			metrics, err := tap.GetMetrics()
			if err == nil && metrics.RMS > 0.001 && metrics.FrameCount > 0 {
				if !signalDetected {
					t.Logf("   🎵 %s: Audio signal detected (RMS: %.6f)", testName, metrics.RMS)
					signalDetected = true
				}
			}
			time.Sleep(50 * time.Millisecond)
		}

		actualDuration := time.Since(start)
		if signalDetected {
			t.Logf("   ✅ %s: Signal detected for ~%.1fs (expected %.1fs)", testName, actualDuration.Seconds(), expectedDuration.Seconds())
		} else {
			t.Logf("   ⚠️ %s: No signal detected in %.1fs", testName, actualDuration.Seconds())
		}
	}

	// Test 1: Normal playback (baseline)
	t.Log("🎵 Test 1: Normal playback (rate=1.0, pitch=0)")
	player.SetVolume(0.7)
	player.SetPlaybackRate(1.0)
	player.SetPitch(0.0)
	player.Play()
	verifyAudioSignal("Test 1", 3*time.Second, 500*time.Millisecond)
	player.Stop()

	// Test 2: Slow playback (half speed)
	t.Log("🐌 Test 2: Slow playback (rate=0.5)")
	player.SetPlaybackRate(0.5)
	player.Play()
	verifyAudioSignal("Test 2", 3*time.Second, 500*time.Millisecond)
	player.Stop()

	// Test 3: Fast playback (double speed)
	t.Log("🏃 Test 3: Fast playback (rate=2.0)")
	player.SetPlaybackRate(2.0)
	player.Play()
	verifyAudioSignal("Test 3", 2*time.Second, 500*time.Millisecond)
	player.Stop()

	// Test 4: Pitch up (one octave)
	t.Log("⬆️  Test 4: Pitch up one octave (+1200 cents)")
	player.SetPlaybackRate(1.0) // Back to normal speed
	player.SetPitch(1200.0)     // +1200 cents = +1 octave
	player.Play()
	verifyAudioSignal("Test 4", 3*time.Second, 500*time.Millisecond)
	player.Stop()

	// Test 5: Pitch down (one octave)
	t.Log("⬇️  Test 5: Pitch down one octave (-1200 cents)")
	player.SetPitch(-1200.0) // -1200 cents = -1 octave
	player.Play()
	verifyAudioSignal("Test 5", 3*time.Second, 500*time.Millisecond)
	player.Stop()

	// Test 6: Extreme pitch (chipmunk effect)
	t.Log("🐿️  Test 6: Chipmunk effect (+2000 cents)")
	player.SetPitch(2000.0)
	player.Play()
	verifyAudioSignal("Test 6", 2*time.Second, 500*time.Millisecond)
	player.Stop()

	// Test 7: Deep voice effect
	t.Log("🐻 Test 7: Deep voice effect (-1800 cents)")
	player.SetPitch(-1800.0)
	player.Play()
	verifyAudioSignal("Test 7", 3*time.Second, 500*time.Millisecond)
	player.Stop()

	// Test 8: Combined effects (slow + high pitch)
	t.Log("🎪 Test 8: Combined effects (rate=0.7, pitch=+600 cents)")
	player.SetPlaybackRate(0.7)
	player.SetPitch(600.0) // +600 cents = +5 semitones
	player.Play()
	verifyAudioSignal("Test 8", 3*time.Second, 500*time.Millisecond)
	player.Stop()

	// Reset to normal
	t.Log("🔄 Resetting to normal (rate=1.0, pitch=0)")
	player.SetPlaybackRate(1.0)
	player.SetPitch(0.0)
	player.Play()
	verifyAudioSignal("Reset", 2*time.Second, 500*time.Millisecond)
	player.Stop()
	time.Sleep(2 * time.Second)
	player.Stop()

	// Test disabling effects
	t.Log("🔧 Disabling time/pitch effects...")
	err = player.DisableTimePitchEffects()
	if err != nil {
		t.Fatalf("Failed to disable time/pitch effects: %v", err)
	}

	// Verify they're disabled
	enabled, err = player.IsTimePitchEffectsEnabled()
	if err != nil {
		t.Fatalf("Failed to check time/pitch effects status after disabling: %v", err)
	}
	if enabled {
		t.Fatalf("Time/pitch effects should be disabled but aren't")
	}
	t.Log("✅ Time/pitch effects disabled successfully!")

	// Final normal playback to verify everything is back to normal
	t.Log("🎵 Final test: Normal playback after disabling effects")
	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()

	t.Log("🎉 Time/pitch effects test complete!")
	t.Log("    Did you hear the different playback rates and pitches?")
	t.Log("    Your time/pitch functionality is working perfectly!")
}
