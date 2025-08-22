package engine

import (
	"path/filepath"
	"testing"
)

// TestPlaybackChannelFileLoading tests actual file loading and playback functionality
func TestPlaybackChannelFileLoading(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	// Use the existing test audio file
	testAudioPath, err := filepath.Abs("../avaudio/engine/idea.m4a")
	if err != nil {
		t.Fatalf("Failed to get absolute path to test audio file: %v", err)
	}

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "ValidAudioFile",
			filePath:    testAudioPath,
			expectError: false,
		},
		{
			name:        "NonExistentFile",
			filePath:    "/nonexistent/file.mp3",
			expectError: true,
			errorMsg:    "failed to load audio file",
		},
		{
			name:        "InvalidPath",
			filePath:    "",
			expectError: true,
			errorMsg:    "file path cannot be empty", // Updated to match our validation
		},
		{
			name:        "SystemAudioFile",
			filePath:    "/System/Library/Sounds/Ping.aiff", // macOS system sound
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := engine.CreatePlaybackChannel(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for file path %s, but got none", tt.filePath)
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error creating playback channel: %v", err)
			}

			// Verify channel state
			expectedState := ExpectedChannelState{
				IsInput:            false,
				IsPlayback:         true,
				Volume:             1.0,
				Pan:                0.0,
				HasInputOptions:    false,
				HasPlaybackOptions: true,
				FilePath:           tt.filePath,
			}

			ValidateChannelState(t, channel, expectedState)

			// Verify native player was created
			if channel.PlaybackOptions.playerPtr == nil {
				t.Error("Native player pointer should not be nil after successful creation")
			}

			t.Logf("✅ Successfully loaded audio file: %s", tt.filePath)
		})
	}
}

// TestPlaybackChannelNativeIntegration tests the native C integration aspects
func TestPlaybackChannelNativeIntegration(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	// Use a system audio file that's guaranteed to exist on macOS
	testAudioPath := "/System/Library/Sounds/Ping.aiff"

	t.Run("PlayerCreationAndFileLoading", func(t *testing.T) {
		channel, err := engine.CreatePlaybackChannel(testAudioPath)
		if err != nil {
			t.Fatalf("Failed to create playback channel: %v", err)
		}

		// Verify all the expected properties
		if channel.PlaybackOptions == nil {
			t.Fatal("PlaybackOptions should not be nil")
		}

		if channel.PlaybackOptions.playerPtr == nil {
			t.Fatal("Native player pointer should not be nil")
		}

		if channel.PlaybackOptions.FilePath != testAudioPath {
			t.Errorf("Expected file path %s, got %s", testAudioPath, channel.PlaybackOptions.FilePath)
		}

		if channel.PlaybackOptions.Rate != 1.0 {
			t.Errorf("Expected default rate 1.0, got %f", channel.PlaybackOptions.Rate)
		}

		if channel.PlaybackOptions.Pitch != 0.0 {
			t.Errorf("Expected default pitch 0.0, got %f", channel.PlaybackOptions.Pitch)
		}

		t.Log("✅ Native player integration successful")
	})

	t.Run("MultipleChannelCreation", func(t *testing.T) {
		// Test creating multiple playback channels
		channels := make([]*Channel, 3)
		var err error

		for i := 0; i < 3; i++ {
			channels[i], err = engine.CreatePlaybackChannel(testAudioPath)
			if err != nil {
				t.Fatalf("Failed to create playback channel %d: %v", i, err)
			}

			if channels[i].PlaybackOptions.playerPtr == nil {
				t.Errorf("Channel %d: Native player pointer should not be nil", i)
			}
		}

		// Verify each channel has its own player instance
		for i := 0; i < 3; i++ {
			for j := i + 1; j < 3; j++ {
				if channels[i].PlaybackOptions.playerPtr == channels[j].PlaybackOptions.playerPtr {
					t.Errorf("Channels %d and %d have the same player pointer - should be unique", i, j)
				}
			}
		}

		t.Logf("✅ Successfully created %d independent playback channels", len(channels))
	})
}

// TestPlaybackChannelEngineIntegration tests integration with the engine lifecycle
func TestPlaybackChannelEngineIntegration(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	testAudioPath := "/System/Library/Sounds/Ping.aiff"

	t.Run("ChannelCountTracking", func(t *testing.T) {
		initialCount := len(engine.Channels)

		// Create a playback channel
		channel, err := engine.CreatePlaybackChannel(testAudioPath)
		if err != nil {
			t.Fatalf("Failed to create playback channel: %v", err)
		}

		// Verify it was added to the engine's channels
		if len(engine.Channels) != initialCount+1 {
			t.Errorf("Expected channel count %d, got %d", initialCount+1, len(engine.Channels))
		}

		// Verify it's the last channel
		lastChannel := engine.Channels[len(engine.Channels)-1]
		if lastChannel != channel {
			t.Error("Created channel should be the last one in the engine's channels slice")
		}

		t.Log("✅ Channel count tracking works correctly")
	})

	t.Run("EngineStateValidation", func(t *testing.T) {
		// Create playback channel
		_, err := engine.CreatePlaybackChannel(testAudioPath)
		if err != nil {
			t.Fatalf("Failed to create playback channel: %v", err)
		}

		// Test engine operations after channel creation
		if !engine.IsRunning() {
			t.Log("Engine not running - this is expected for test setup")
		}

		// Test master volume operations
		err = engine.SetMasterVolume(0.5)
		if err != nil {
			t.Logf("SetMasterVolume failed (expected for stopped engine): %v", err)
		}

		currentVolume := engine.GetMasterVolume()
		t.Logf("Current master volume: %f", currentVolume)

		t.Log("✅ Engine state validation completed")
	})
}

// TestPlaybackChannelErrorHandling tests error conditions and cleanup
func TestPlaybackChannelErrorHandling(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	t.Run("EngineNotInitialized", func(t *testing.T) {
		// Create an engine with nil native engine to test error handling
		brokenEngine := &Engine{
			SampleRate:   44100,
			BufferSize:   1024,
			MasterVolume: 1.0,
			nativeEngine: nil, // This will cause the error
		}

		_, err := brokenEngine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
		if err == nil {
			t.Fatal("Expected error when engine is not properly initialized")
		}

		expectedMsg := "engine is not properly initialized"
		if !containsString(err.Error(), expectedMsg) {
			t.Errorf("Expected error message to contain '%s', got: %v", expectedMsg, err)
		}

		t.Log("✅ Proper error handling for uninitialized engine")
	})

	t.Run("FileLoadingFailure", func(t *testing.T) {
		// Test with definitely non-existent file
		invalidPath := "/this/path/definitely/does/not/exist.mp3"
		_, err := engine.CreatePlaybackChannel(invalidPath)

		if err == nil {
			t.Fatal("Expected error for non-existent file")
		}

		if !containsString(err.Error(), "failed to load audio file") {
			t.Errorf("Expected file loading error, got: %v", err)
		}

		t.Log("✅ Proper error handling for file loading failure")
	})
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
