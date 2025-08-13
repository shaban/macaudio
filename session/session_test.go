//go:build darwin && cgo

package session

import (
	"sync"
	"testing"
	"time"
	"unsafe"
)

// Mock engine for testing (actual implementation would use real AVAudioEngine)
type MockEngine struct{}

func TestSetEngineAndGetStatus(t *testing.T) {
	// Test with nil engine pointer (should not crash)
	var enginePtr unsafe.Pointer = nil

	// Test spec
	spec := AudioSpec{
		SampleRate:   44100.0,
		ChannelCount: 2,
		BitDepth:     16,
		BufferSize:   512,
	}

	// Set engine - this should handle nil gracefully
	SetEngine(enginePtr, spec)

	// Get status
	status := GetEngineStatus()

	// Verify spec was stored
	if status.AudioSpec.SampleRate != 44100.0 {
		t.Errorf("Expected sample rate 44100, got %.1f", status.AudioSpec.SampleRate)
	}
	if status.AudioSpec.ChannelCount != 2 {
		t.Errorf("Expected 2 channels, got %d", status.AudioSpec.ChannelCount)
	}
	if status.AudioSpec.BitDepth != 16 {
		t.Errorf("Expected 16 bit depth, got %d", status.AudioSpec.BitDepth)
	}
	if status.AudioSpec.BufferSize != 512 {
		t.Errorf("Expected 512 buffer size, got %d", status.AudioSpec.BufferSize)
	}

	// Clean up
	Cleanup()
}

func TestConfigurationChangeCallback(t *testing.T) {
	callbackTriggered := false

	// Set callback
	SetConfigurationChangeCallback(func() {
		callbackTriggered = true
	})

	// Simulate configuration change
	configurationChanged()

	// Give it a moment to trigger
	time.Sleep(10 * time.Millisecond)

	if !callbackTriggered {
		t.Error("Expected callback to be triggered")
	}

	// Clean up
	Cleanup()
}

func TestLastConfigChangeTime(t *testing.T) {
	before := time.Now()

	// Simulate configuration change
	configurationChanged()

	after := time.Now()

	// Check that lastConfigChange was updated
	status := GetEngineStatus()

	if status.LastConfigChange.Before(before) || status.LastConfigChange.After(after) {
		t.Errorf("LastConfigChange %v should be between %v and %v",
			status.LastConfigChange, before, after)
	}
}

func TestHotplugSimulation(t *testing.T) {
	t.Log("🔥 Testing hotplug simulation with real AVAudioEngine")

	var wg sync.WaitGroup
	wg.Add(1)

	callbackTriggered := false
	callback := func() {
		t.Log("📞 Configuration change callback received from simulated hotplug!")
		callbackTriggered = true
		wg.Done()
	}

	// Set up monitoring with nil engine - this will enable global monitoring
	SetEngine(nil, AudioSpec{
		SampleRate:   48000,
		ChannelCount: 2,
		BitDepth:     32,
		BufferSize:   512,
	})
	defer Cleanup()

	// Set up callback
	SetConfigurationChangeCallback(callback)

	// Simulate hotplug - will create temporary engine and post notification
	t.Log("🔌 Simulating hotplug event...")
	SimulateHotplug(nil) // Creates temporary engine and posts notification

	// Wait for callback with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if !callbackTriggered {
			t.Error("WaitGroup completed but callback flag not set")
		}
		t.Log("✅ Hotplug simulation successful - callback triggered!")
	case <-time.After(2 * time.Second):
		t.Fatal("❌ Timeout waiting for hotplug callback")
	}
}

func BenchmarkGetEngineStatus(b *testing.B) {
	// Set up mock engine
	mockEngine := &MockEngine{}
	enginePtr := unsafe.Pointer(mockEngine)
	spec := AudioSpec{SampleRate: 44100.0, ChannelCount: 2, BitDepth: 16, BufferSize: 512}
	SetEngine(enginePtr, spec)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		GetEngineStatus()
	}

	Cleanup()
}
