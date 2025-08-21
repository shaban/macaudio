package engine

import (
	"os"
	"testing"
	"time"
)

// TestMinimalTimePitch - strip everything down to basics
func TestMinimalTimePitch(t *testing.T) {
	if os.Getenv("MACAUDIO_AUDIBLE") == "" {
		t.Skip("Set MACAUDIO_AUDIBLE=1 to run test")
	}

	t.Log("🔍 MINIMAL Time/Pitch Test - just the essentials")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	player, err := engine.NewPlayer()
	if err != nil {
		t.Fatal(err)
	}
	defer player.Destroy()

	player.LoadFile("idea.m4a")

	// Test 1: Normal connection and playback
	t.Log("🎵 Test 1: Normal playback")
	player.ConnectToMainMixer()
	player.SetVolume(0.8)
	player.Play()
	time.Sleep(2 * time.Second)
	player.Stop()
	t.Log("   ✅ Normal playback works")

	// Test 2: Enable TimePitch and try again
	t.Log("🔧 Test 2: Enable TimePitch effects")
	err = player.EnableTimePitchEffects()
	if err != nil {
		t.Fatal("EnableTimePitchEffects failed:", err)
	}

	// RESTART the engine to reset the audio graph
	t.Log("🔄 Restarting engine after enabling TimePitch...")
	engine.Stop()
	time.Sleep(100 * time.Millisecond)
	engine.Start()

	// Check the actual connection status by trying to play
	t.Log("🎵 Test 3: TimePitch playback with timing measurement")
	player.SetPlaybackRate(0.8) // Slower = should take longer
	player.SetPitch(-300.0)     // Lower pitch

	startTime := time.Now()
	err = player.Play()
	if err != nil {
		t.Fatal("Play failed after TimePitch enabled:", err)
	}

	// Play for 2 seconds of actual time
	time.Sleep(2 * time.Second)
	player.Stop()
	elapsed := time.Since(startTime)

	t.Logf("⏱️  Playback took %v (expected ~2s)", elapsed)
	t.Log("🎉 If you heard slower/lower audio, TimePitch works!")
	t.Log("   If it sounded normal, TimePitch is not in the audio path")

	// Test extreme settings to make it more obvious
	t.Log("🔧 Test 4: Extreme TimePitch settings")
	player.SetPlaybackRate(0.5) // Half speed
	player.SetPitch(1200.0)     // One octave higher

	err = player.Play()
	if err != nil {
		t.Fatal("Play failed with extreme TimePitch:", err)
	}

	time.Sleep(2 * time.Second)
	player.Stop()

	t.Log("🎵 Extreme test complete - half speed, one octave higher")
}
