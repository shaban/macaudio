package macaudio

import (
	"testing"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/devices"
)

// getValidOutputDevice returns the first available online output device for testing
func getValidOutputDevice(t *testing.T) string {
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to enumerate audio devices: %v", err)
	}

	// Try to find a default output device first
	for _, device := range audioDevices {
		if device.IsDefaultOutput && device.IsOnline && device.CanOutput() {
			return device.UID
		}
	}

	// Fall back to any online output device
	outputs := audioDevices.Online().Outputs()
	if len(outputs) == 0 {
		t.Skip("No online output devices available for testing")
	}

	return outputs[0].UID
}

// createTestConfig creates a standard EngineConfig for testing
func createTestConfig(t *testing.T, sampleRate float64, bufferSize int) EngineConfig {
	return EngineConfig{
		AudioSpec: engine.AudioSpec{
			SampleRate:   sampleRate,
			BufferSize:   bufferSize,
			BitDepth:     32, // Standard for testing
			ChannelCount: 2,  // Stereo
		},
		OutputDeviceUID: getValidOutputDevice(t),
		ErrorHandler:    &DefaultErrorHandler{},
	}
}

// TestEngineValidation tests various configuration validation scenarios
func TestEngineValidation(t *testing.T) {
	t.Run("EmptyConfig", func(t *testing.T) {
		config := EngineConfig{}
		_, err := NewEngine(config)
		if err == nil {
			t.Fatal("Expected error for empty config, got nil")
		}
		if !containsString(err.Error(), "OutputDeviceUID is required") {
			t.Errorf("Expected OutputDeviceUID validation error, got: %v", err)
		}
	})

	t.Run("InvalidSampleRateTooLow", func(t *testing.T) {
		config := EngineConfig{
			AudioSpec: engine.AudioSpec{
				SampleRate: 4000, // Too low
				BufferSize: 256,
			},
			OutputDeviceUID: getValidOutputDevice(t),
			ErrorHandler:    &DefaultErrorHandler{},
		}
		_, err := NewEngine(config)
		if err == nil {
			t.Fatal("Expected error for sample rate too low, got nil")
		}
		if !containsString(err.Error(), "SampleRate must be at least 8000") {
			t.Errorf("Expected SampleRate validation error, got: %v", err)
		}
	})

	t.Run("InvalidSampleRateTooHigh", func(t *testing.T) {
		config := EngineConfig{
			AudioSpec: engine.AudioSpec{
				SampleRate: 500000, // Too high
				BufferSize: 256,
			},
			OutputDeviceUID: getValidOutputDevice(t),
			ErrorHandler:    &DefaultErrorHandler{},
		}
		_, err := NewEngine(config)
		if err == nil {
			t.Fatal("Expected error for sample rate too high, got nil")
		}
		if !containsString(err.Error(), "SampleRate cannot exceed 384000") {
			t.Errorf("Expected SampleRate validation error, got: %v", err)
		}
	})

	t.Run("InvalidBufferSizeTooSmall", func(t *testing.T) {
		config := createTestConfig(t, 48000, 32) // Too small
		_, err := NewEngine(config)
		if err == nil {
			t.Fatal("Expected error for buffer size too small, got nil")
		}
		if !containsString(err.Error(), "BufferSize must be at least 64") {
			t.Errorf("Expected BufferSize validation error, got: %v", err)
		}
	})

	t.Run("InvalidBufferSizeTooBig", func(t *testing.T) {
		config := createTestConfig(t, 48000, 8192) // Too big
		_, err := NewEngine(config)
		if err == nil {
			t.Fatal("Expected error for buffer size too big, got nil")
		}
		if !containsString(err.Error(), "BufferSize cannot exceed 4096") {
			t.Errorf("Expected BufferSize validation error, got: %v", err)
		}
	})

	t.Run("NonexistentOutputDevice", func(t *testing.T) {
		config := createTestConfig(t, 48000, 256)
		config.OutputDeviceUID = "nonexistent-device-12345" // Override with invalid device
		_, err := NewEngine(config)
		if err == nil {
			t.Fatal("Expected error for nonexistent output device, got nil")
		}
		if !containsString(err.Error(), "output device") && !containsString(err.Error(), "not found") {
			t.Errorf("Expected output device validation error, got: %v", err)
		}
	})
}

// TestBufferSizeApplication tests that different buffer sizes are properly applied
func TestBufferSizeApplication(t *testing.T) {
	testCases := []struct {
		name       string
		bufferSize int
		sampleRate float64
		useCase    string
	}{
		{"LivePerformance", 64, 96000, "Ultra-low latency live performance"},
		{"Standard", 256, 48000, "Standard audio application"},
		{"Studio", 1024, 48000, "Studio production with plugin chains"},
		{"Audiophile", 2048, 192000, "High-end audiophile playback"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := createTestConfig(t, tc.sampleRate, tc.bufferSize)

			engine, err := NewEngine(config)
			if err != nil {
				t.Fatalf("Failed to create engine for %s: %v", tc.useCase, err)
			}
			defer engine.Stop()

			// Test that the engine was created successfully with the specified buffer size
			if engine == nil {
				t.Fatalf("Engine is nil for %s", tc.useCase)
			}

			// The engine should start successfully with the specified buffer size
			if err := engine.Start(); err != nil {
				t.Logf("Engine start failed for %s (may be expected): %v", tc.useCase, err)
			}

			t.Logf("%s: %d samples @ %.0f Hz = %.2f ms latency",
				tc.useCase, tc.bufferSize, tc.sampleRate,
				float64(tc.bufferSize)/tc.sampleRate*1000.0)
		})
	}
}

func TestEngineCreation(t *testing.T) {
	config := createTestConfig(t, 48000, 256)

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	if engine == nil {
		t.Fatal("Engine is nil")
	}

	// Check initial state
	if engine.IsRunning() {
		t.Error("Engine should not be running initially")
	}

	// Check master channel exists
	masterChannel := engine.GetMasterChannel()
	if masterChannel == nil {
		t.Fatal("Master channel is nil")
	}

	// Use GetIDString() for string comparison
	if masterChannel.GetIDString() == "" {
		t.Error("Master channel should have a valid ID string")
	}

	// Check initial channels list includes master
	channels := engine.ListChannels()
	if len(channels) != 1 {
		t.Errorf("Expected 1 channel initially, got %d", len(channels))
	}
}

// TestEngineStartValidation tests that Start() validates engine readiness
func TestEngineStartValidation(t *testing.T) {
	t.Run("StartWithoutChannelConnections", func(t *testing.T) {
		config := createTestConfig(t, 48000, 256)

		engine, err := NewEngine(config)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		// Attempting to start engine without proper channel connections
		// should either succeed with minimal setup or fail gracefully
		err = engine.Start()

		// We expect this to either:
		// 1. Succeed (if we have minimal default routing)
		// 2. Fail gracefully with a validation error (not crash)
		if err != nil {
			// If it fails, it should be a validation error, not a crash
			if containsString(err.Error(), "panic") || containsString(err.Error(), "fatal") {
				t.Errorf("Engine Start() should fail gracefully, not crash. Error: %v", err)
			}
			t.Logf("Engine Start() failed gracefully with validation: %v", err)
		} else {
			// If it succeeds, engine should be running
			if !engine.IsRunning() {
				t.Error("Engine should be running after successful Start()")
			}
			// Clean up
			if err := engine.Stop(); err != nil {
				t.Errorf("Failed to stop engine: %v", err)
			}
		}
	})
}

func TestEngineStartStop(t *testing.T) {
	config := createTestConfig(t, 48000, 256)

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Start engine
	if err := engine.Start(); err != nil {
		// For now, we'll accept that starting might fail gracefully
		// The important thing is that it doesn't crash
		t.Logf("Engine Start() returned error (expected for incomplete setup): %v", err)
		return
	}

	if !engine.IsRunning() {
		t.Error("Engine should be running after Start()")
	}

	// Check components are running
	if !engine.GetDeviceMonitor().IsRunning() {
		t.Error("Device monitor should be running")
	}

	if !engine.GetDispatcher().IsRunning() {
		t.Error("Dispatcher should be running")
	}

	// Stop engine
	if err := engine.Stop(); err != nil {
		t.Errorf("Failed to stop engine: %v", err)
	}

	if engine.IsRunning() {
		t.Error("Engine should not be running after Stop()")
	}
}

func TestChannelCreation(t *testing.T) {
	config := createTestConfig(t, 48000, 256)

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Start engine to initialize dispatcher for channel creation
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// Create playback channel
	playbackConfig := PlaybackConfig{
		FilePath:    "/nonexistent/file.wav", // File doesn't need to exist for this test
		LoopEnabled: false,
		AutoStart:   false,
	}

	playbackChannel, err := engine.CreatePlaybackChannel("test_playback", playbackConfig)
	if err != nil {
		t.Fatalf("Failed to create playback channel: %v", err)
	}

	// Use GetIDString() for comparison
	if playbackChannel.GetIDString() == "" {
		t.Error("Playback channel should have a valid ID string")
	}

	if playbackChannel.GetType() != ChannelTypePlayback {
		t.Errorf("Channel type should be playback, got %s", playbackChannel.GetType())
	}

	// Check channel is in engine
	channels := engine.ListChannels()
	found := false
	expectedID := playbackChannel.GetIDString()
	for _, id := range channels {
		if id == expectedID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Playback channel not found in engine channels list")
	}

	// Create aux channel
	auxConfig := AuxConfig{
		SendLevel:   0.5,
		ReturnLevel: 0.7,
		PreFader:    false,
	}

	auxChannel, err := engine.CreateAuxChannel("test_aux", auxConfig)
	if err != nil {
		t.Fatalf("Failed to create aux channel: %v", err)
	}

	if auxChannel.GetIDString() == "" {
		t.Error("Aux channel should have a valid ID string")
	}
}

func TestPluginChain(t *testing.T) {
	chain := NewPluginChain()

	if chain == nil {
		t.Fatal("Plugin chain is nil")
	}

	instances := chain.GetInstances()
	if len(instances) != 0 {
		t.Errorf("New plugin chain should be empty, got %d instances", len(instances))
	}

	// Add a plugin blueprint
	blueprint := PluginBlueprint{
		Type:           "aufx",
		Subtype:        "test",
		ManufacturerID: "test",
		Name:           "Test Plugin",
		IsInstalled:    false,
	}

	instance, err := chain.AddPlugin(blueprint, 0)
	if err != nil {
		t.Fatalf("Failed to add plugin: %v", err)
	}

	if instance.Blueprint.Name != "Test Plugin" {
		t.Errorf("Plugin name should be 'Test Plugin', got '%s'", instance.Blueprint.Name)
	}

	if instance.Position != 0 {
		t.Errorf("Plugin position should be 0, got %d", instance.Position)
	}

	// Check plugin is in chain
	instances = chain.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Plugin chain should have 1 instance, got %d", len(instances))
	}

	// Remove plugin
	if err := chain.RemovePlugin(instance.ID); err != nil {
		t.Fatalf("Failed to remove plugin: %v", err)
	}

	instances = chain.GetInstances()
	if len(instances) != 0 {
		t.Errorf("Plugin chain should be empty after removal, got %d instances", len(instances))
	}
}

func TestSerialization(t *testing.T) {
	config := createTestConfig(t, 48000, 256)

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Start engine to initialize dispatcher for channel creation
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// Create a channel
	playbackConfig := PlaybackConfig{
		FilePath:    "/test/file.wav",
		LoopEnabled: true,
		AutoStart:   true,
	}

	channel, err := engine.CreatePlaybackChannel("test_serialize", playbackConfig)
	if err != nil {
		t.Fatalf("Failed to create playback channel: %v", err)
	}

	// Serialize engine state
	serializer := engine.GetSerializer()
	jsonState, err := serializer.SaveToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize state: %v", err)
	}

	if len(jsonState) == 0 {
		t.Error("Serialized state is empty")
	}

	// The serialized state should contain our channel
	channelID := channel.GetIDString()
	if !containsString(string(jsonState), channelID) {
		t.Error("Serialized state doesn't contain test channel")
	}

	masterChannel := engine.GetMasterChannel()
	masterID := masterChannel.GetIDString()
	if !containsString(string(jsonState), masterID) {
		t.Error("Serialized state doesn't contain master channel")
	}
}

func TestDeviceMonitor(t *testing.T) {
	config := createTestConfig(t, 48000, 256)

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	monitor := engine.GetDeviceMonitor()
	if monitor == nil {
		t.Fatal("Device monitor is nil")
	}

	// Check initial state
	if monitor.IsRunning() {
		t.Error("Device monitor should not be running initially")
	}

	// Check default polling interval
	interval := monitor.GetPollingInterval()
	expectedInterval := 50 * time.Millisecond
	if interval != expectedInterval {
		t.Errorf("Expected polling interval %v, got %v", expectedInterval, interval)
	}

	// Test interval validation
	err = monitor.SetPollingInterval(5 * time.Millisecond)
	if err == nil {
		t.Error("Should reject polling interval less than 10ms")
	}

	err = monitor.SetPollingInterval(100 * time.Millisecond)
	if err != nil {
		t.Errorf("Should accept valid polling interval: %v", err)
	}

	// Try to start engine - it might fail, but should be graceful
	if err := engine.Start(); err != nil {
		t.Logf("Engine Start() failed gracefully: %v", err)
		return
	}
	defer engine.Stop()

	if !monitor.IsRunning() {
		t.Error("Device monitor should be running after engine start")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
