package engine

import (
	"testing"
)

// TestVolumeConstraints tests volume validation and edge cases
func TestVolumeConstraints(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	tests := []struct {
		name        string
		setVolume   float32
		expectError bool
		expectedVol float32 // What we expect GetMasterVolume to return
		description string
	}{
		{
			name:        "ValidMinVolume",
			setVolume:   0.0,
			expectError: false,
			expectedVol: 0.0,
			description: "Minimum valid volume (silence)",
		},
		{
			name:        "ValidMaxVolume",
			setVolume:   1.0,
			expectError: false,
			expectedVol: 1.0,
			description: "Maximum valid volume (unity gain)",
		},
		{
			name:        "ValidMidVolume",
			setVolume:   0.5,
			expectError: false,
			expectedVol: 0.5,
			description: "Mid-range volume",
		},
		{
			name:        "NegativeVolume",
			setVolume:   -0.1,
			expectError: true, // ✅ Native code validates this!
			expectedVol: 0.0,  // Volume should remain unchanged on error
			description: "Negative volume - correctly rejected",
		},
		{
			name:        "VolumeAboveOne",
			setVolume:   1.5,
			expectError: true, // ✅ Native code validates this!
			expectedVol: 0.0,  // Volume should remain unchanged on error
			description: "Volume above unity gain - correctly rejected",
		},
		{
			name:        "ExtremeNegativeVolume",
			setVolume:   -100.0,
			expectError: true, // ✅ Native code validates this!
			expectedVol: 0.0,  // Volume should remain unchanged on error
			description: "Extreme negative volume - correctly rejected",
		},
		{
			name:        "ExtremePositiveVolume",
			setVolume:   100.0,
			expectError: true, // ✅ Native code validates this!
			expectedVol: 0.0,  // Volume should remain unchanged on error
			description: "Extreme positive volume - correctly rejected",
		},
		{
			name:        "VerySmallVolume",
			setVolume:   0.001,
			expectError: false,
			expectedVol: 0.001,
			description: "Very quiet volume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			// Reset to a known state before each test
			if err := engine.SetMasterVolume(0.5); err != nil {
				t.Fatalf("Failed to reset volume: %v", err)
			}

			// Test setting volume
			err := engine.SetMasterVolume(tt.setVolume)
			if (err != nil) != tt.expectError {
				t.Errorf("SetMasterVolume(%v) error = %v, wantErr %v", tt.setVolume, err, tt.expectError)
			}

			// Always check what the volume actually is now
			actualVolume := engine.GetMasterVolume()
			if !tt.expectError {
				// If we expected success, the volume should be what we set
				if actualVolume != tt.expectedVol {
					t.Errorf("GetMasterVolume() = %v, want %v", actualVolume, tt.expectedVol)
				}
			} else {
				// If we expected an error, the volume should remain at 0.5 (our reset value)
				if actualVolume != 0.5 {
					t.Errorf("GetMasterVolume() = %v, expected it to remain at 0.5 after error", actualVolume)
				}
				t.Logf("✅ Volume correctly remained at %v after rejecting invalid value %v", actualVolume, tt.setVolume)
			}
		})
	}
}

// TestPanConstraints tests pan validation and edge cases
func TestPanConstraints(t *testing.T) {
	_, inputDevice := TestDeviceSetup(t)
	if inputDevice == nil {
		t.Skip("No input devices available for testing")
	}

	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	tests := []struct {
		name        string
		pan         float32
		expectValid bool
		description string
	}{
		{
			name:        "CenterPan",
			pan:         0.0,
			expectValid: true,
			description: "Center pan (no stereo shift)",
		},
		{
			name:        "FullLeft",
			pan:         -1.0,
			expectValid: true,
			description: "Full left pan",
		},
		{
			name:        "FullRight",
			pan:         1.0,
			expectValid: true,
			description: "Full right pan",
		},
		{
			name:        "PartialLeft",
			pan:         -0.5,
			expectValid: true,
			description: "Partial left pan",
		},
		{
			name:        "PartialRight",
			pan:         0.5,
			expectValid: true,
			description: "Partial right pan",
		},
		{
			name:        "BeyondLeft",
			pan:         -1.5,
			expectValid: false, // TODO: Should be invalid
			description: "Pan beyond full left - should be clamped or rejected",
		},
		{
			name:        "BeyondRight",
			pan:         1.5,
			expectValid: false, // TODO: Should be invalid
			description: "Pan beyond full right - should be clamped or rejected",
		},
		{
			name:        "ExtremePan",
			pan:         -100.0,
			expectValid: false, // TODO: Should be invalid
			description: "Extreme pan value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			if tt.expectValid {
				// Test valid values - should succeed in channel creation and setting
				inputConfig := TestChannelConfig{
					Volume:      1.0,
					Pan:         tt.pan,
					PluginCount: 0,
					UseRealFile: true,
				}

				inputChannel := CreateTestInputChannel(t, engine, inputConfig)

				// Verify the pan value was set correctly
				if inputChannel.Pan != tt.pan {
					t.Errorf("Expected pan %v to be accepted, but got %v", tt.pan, inputChannel.Pan)
				}

				// Test with playback channel too
				playbackConfig := TestChannelConfig{
					Volume:      1.0,
					Pan:         tt.pan,
					PluginCount: 0,
					UseRealFile: true,
				}

				playbackChannel := CreateTestPlaybackChannel(t, engine, playbackConfig)

				if playbackChannel.Pan != tt.pan {
					t.Errorf("Expected pan %v to be accepted, but got %v", tt.pan, playbackChannel.Pan)
				}
			} else {
				// Test invalid values - should be rejected by validation

				// First create a valid channel
				validConfig := TestChannelConfig{
					Volume:      1.0,
					Pan:         0.0, // Start with valid pan
					PluginCount: 0,
					UseRealFile: true,
				}

				inputChannel := CreateTestInputChannel(t, engine, validConfig)
				playbackChannel := CreateTestPlaybackChannel(t, engine, validConfig)

				// Now try to set invalid pan values - should be rejected
				originalPan := inputChannel.Pan

				err := inputChannel.SetPan(tt.pan)
				if err == nil {
					t.Errorf("⚠️  Pan %v should have been rejected but was accepted", tt.pan)
				} else {
					t.Logf("✅ Pan %v correctly rejected: %v", tt.pan, err)
				}

				// Verify pan remained unchanged after rejection
				if inputChannel.Pan != originalPan {
					t.Errorf("Pan should have remained %v after rejection, but got %v", originalPan, inputChannel.Pan)
				}

				// Test playback channel too
				originalPlaybackPan := playbackChannel.Pan
				err = playbackChannel.SetPan(tt.pan)
				if err == nil {
					t.Errorf("⚠️  Pan %v should have been rejected but was accepted", tt.pan)
				} else {
					t.Logf("✅ Pan %v correctly rejected: %v", tt.pan, err)
				}

				if playbackChannel.Pan != originalPlaybackPan {
					t.Errorf("Pan should have remained %v after rejection, but got %v", originalPlaybackPan, playbackChannel.Pan)
				}
			}
		})
	}
}

// TestChannelVolumeConstraints tests individual channel volume constraints
func TestChannelVolumeConstraints(t *testing.T) {
	_, inputDevice := TestDeviceSetup(t)
	if inputDevice == nil {
		t.Skip("No input devices available for testing")
	}

	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	tests := []struct {
		name        string
		volume      float32
		expectValid bool
		description string
	}{
		{
			name:        "UnityGain",
			volume:      1.0,
			expectValid: true,
			description: "Unity gain (no amplification)",
		},
		{
			name:        "Silence",
			volume:      0.0,
			expectValid: true,
			description: "Complete silence",
		},
		{
			name:        "HalfVolume",
			volume:      0.5,
			expectValid: true,
			description: "Half volume",
		},
		{
			name:        "QuietVolume",
			volume:      0.1,
			expectValid: true,
			description: "Very quiet",
		},
		{
			name:        "NegativeVolume",
			volume:      -0.5,
			expectValid: false, // TODO: Should be invalid
			description: "Negative volume - phase inversion?",
		},
		{
			name:        "AmplifiedVolume",
			volume:      2.0,
			expectValid: false, // TODO: Could cause clipping
			description: "2x amplification - potential clipping",
		},
		{
			name:        "ExtremeAmplification",
			volume:      10.0,
			expectValid: false, // TODO: Dangerous
			description: "10x amplification - very dangerous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			if tt.expectValid {
				// Test valid values - should succeed in channel creation and setting
				inputConfig := TestChannelConfig{
					Volume:      tt.volume,
					Pan:         0.0,
					PluginCount: 0,
					UseRealFile: true,
				}

				inputChannel := CreateTestInputChannel(t, engine, inputConfig)

				// Verify the volume value was set correctly
				if inputChannel.Volume != tt.volume {
					t.Errorf("Expected volume %v to be accepted, but got %v", tt.volume, inputChannel.Volume)
				}

				// Test with playback channel too
				playbackConfig := TestChannelConfig{
					Volume:      tt.volume,
					Pan:         0.0,
					PluginCount: 0,
					UseRealFile: true,
				}

				playbackChannel := CreateTestPlaybackChannel(t, engine, playbackConfig)

				if playbackChannel.Volume != tt.volume {
					t.Errorf("Expected volume %v to be accepted, but got %v", tt.volume, playbackChannel.Volume)
				}
			} else {
				// Test invalid values - should be rejected by validation

				// First create a valid channel
				validConfig := TestChannelConfig{
					Volume:      1.0, // Start with valid volume
					Pan:         0.0,
					PluginCount: 0,
					UseRealFile: true,
				}

				inputChannel := CreateTestInputChannel(t, engine, validConfig)
				playbackChannel := CreateTestPlaybackChannel(t, engine, validConfig)

				// Now try to set invalid volume values - should be rejected
				originalVolume := inputChannel.Volume

				err := inputChannel.SetVolume(tt.volume)
				if err == nil {
					t.Errorf("⚠️  Volume %v should have been rejected but was accepted", tt.volume)
				} else {
					t.Logf("✅ Volume %v correctly rejected: %v", tt.volume, err)
				}

				// Verify volume remained unchanged after rejection
				if inputChannel.Volume != originalVolume {
					t.Errorf("Volume should have remained %v after rejection, but got %v", originalVolume, inputChannel.Volume)
				}

				// Test playback channel too
				originalPlaybackVolume := playbackChannel.Volume
				err = playbackChannel.SetVolume(tt.volume)
				if err == nil {
					t.Errorf("⚠️  Volume %v should have been rejected but was accepted", tt.volume)
				} else {
					t.Logf("✅ Volume %v correctly rejected: %v", tt.volume, err)
				}

				if playbackChannel.Volume != originalPlaybackVolume {
					t.Errorf("Volume should have remained %v after rejection, but got %v", originalPlaybackVolume, playbackChannel.Volume)
				}
			}
		})
	}
}

// TestOtherConstraints tests other potential missing validations
func TestOtherConstraints(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	t.Run("BufferSizeConstraints", func(t *testing.T) {
		// TODO: Are there constraints on buffer sizes?
		// Common valid sizes: 64, 128, 256, 512, 1024, 2048
		// Invalid sizes: 0, 1, 3, negative values, extremely large values

		validBufferSizes := []int{64, 128, 256, 512, 1024, 2048}
		invalidBufferSizes := []int{0, -1, 1, 3, 7, 100000}

		t.Logf("Valid buffer sizes we should support: %v", validBufferSizes)
		t.Logf("Invalid buffer sizes we should reject: %v", invalidBufferSizes)

		// Currently no validation exists for buffer sizes
		t.Log("⚠️  No buffer size validation is currently implemented")
	})

	t.Run("SampleRateConstraints", func(t *testing.T) {
		// TODO: Are there constraints on sample rates?
		// Common rates: 44100, 48000, 88200, 96000, 176400, 192000
		// Invalid rates: 0, negative, extremely high/low values

		validSampleRates := []int{44100, 48000, 88200, 96000}
		invalidSampleRates := []int{0, -1, 100, 1000000}

		t.Logf("Valid sample rates we should support: %v", validSampleRates)
		t.Logf("Invalid sample rates we should reject: %v", invalidSampleRates)

		// Currently no validation exists for sample rates (except 0 = device default)
		t.Log("⚠️  No sample rate validation is currently implemented")
	})

	t.Run("ChannelIndexConstraints", func(t *testing.T) {
		// TODO: Are there constraints on channel indices?
		// Valid: 0, 1 (for stereo), maybe more for multi-channel devices
		// Invalid: negative values, extremely high values

		t.Log("⚠️  No channel index validation is currently implemented")
		// We use channel index 0 in our tests, but don't validate bounds
	})

	t.Run("FilePathValidation", func(t *testing.T) {
		// Test playback channels with various file paths
		testPaths := []struct {
			path        string
			shouldWork  bool
			description string
		}{
			{
				path:        "/System/Library/Sounds/Ping.aiff",
				shouldWork:  true,
				description: "Valid system sound file",
			},
			{
				path:        "/nonexistent/path/file.wav",
				shouldWork:  false,
				description: "Nonexistent file path",
			},
			{
				path:        "",
				shouldWork:  false,
				description: "Empty file path",
			},
			{
				path:        "/etc/passwd",
				shouldWork:  false,
				description: "Non-audio file",
			},
		}

		for _, tt := range testPaths {
			t.Run(tt.description, func(t *testing.T) {
				config := TestChannelConfig{
					Volume:      1.0,
					Pan:         0.0,
					PluginCount: 0,
					UseRealFile: false, // Use the provided path directly
				}

				// Create channel manually to test the specific path
				channel := &Channel{
					Volume: config.Volume,
					Pan:    config.Pan,
					PlaybackOptions: &PlaybackOptions{
						FilePath: tt.path,
						Rate:     1.0,
						Pitch:    0.0,
					},
				}

				engine.Channels = append(engine.Channels, channel)

				// Currently no validation happens during channel creation
				// Validation might happen during engine start
				t.Logf("Created channel with path: %s", tt.path)

				if !tt.shouldWork {
					t.Logf("⚠️  Invalid path %s was accepted - validation may happen at runtime", tt.path)
				}
			})
		}
	})
}
