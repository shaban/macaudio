package engine

import (
	"testing"
	"time"
)

// TestMultiChannelPlayback tests multiple playback channels playing simultaneously
func TestMultiChannelPlayback(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	// Use system audio files for reliable testing
	testFiles := []string{
		"/System/Library/Sounds/Ping.aiff",
		"/System/Library/Sounds/Ping.aiff", // Same file, different channels
		"/System/Library/Sounds/Ping.aiff",
	}

	channels := make([]*Channel, len(testFiles))
	var err error

	t.Run("CreateMultipleChannels", func(t *testing.T) {
		for i, filePath := range testFiles {
			channels[i], err = engine.CreatePlaybackChannel(filePath)
			if err != nil {
				t.Fatalf("Failed to create channel %d: %v", i, err)
			}

			// Verify each channel has unique mixer node
			if channels[i].mixerNodePtr == nil {
				t.Errorf("Channel %d has nil mixer node", i)
			}

			// Verify unique player instances
			if channels[i].PlaybackOptions.playerPtr == nil {
				t.Errorf("Channel %d has nil player", i)
			}
		}

		// Verify all mixer nodes are unique
		for i := 0; i < len(channels); i++ {
			for j := i + 1; j < len(channels); j++ {
				if channels[i].mixerNodePtr == channels[j].mixerNodePtr {
					t.Errorf("Channels %d and %d have the same mixer node - should be unique", i, j)
				}
				if channels[i].PlaybackOptions.playerPtr == channels[j].PlaybackOptions.playerPtr {
					t.Errorf("Channels %d and %d have the same player - should be unique", i, j)
				}
			}
		}

		t.Logf("✅ Successfully created %d independent channels", len(channels))
	})

	t.Run("StartEngineWithMultipleChannels", func(t *testing.T) {
		err = engine.Start()
		if err != nil {
			t.Fatalf("Failed to start engine with multiple channels: %v", err)
		}

		if !engine.IsRunning() {
			t.Fatal("Engine should be running after successful start")
		}

		t.Log("✅ Engine started successfully with multiple channels connected")
	})

	t.Run("ChannelIsolationTesting", func(t *testing.T) {
		// Set different volumes and pans for each channel
		testConfigs := []struct {
			volume float32
			pan    float32
		}{
			{volume: 1.0, pan: 0.0},  // Channel 0: Full volume, center
			{volume: 0.5, pan: -0.5}, // Channel 1: Half volume, left
			{volume: 0.75, pan: 0.5}, // Channel 2: 75% volume, right
		}

		for i, config := range testConfigs {
			if i >= len(channels) {
				break
			}

			// Set volume
			err = channels[i].SetVolume(config.volume)
			if err != nil {
				t.Errorf("Failed to set volume for channel %d: %v", i, err)
				continue
			}

			// Set pan
			err = channels[i].SetPan(config.pan)
			if err != nil {
				t.Errorf("Failed to set pan for channel %d: %v", i, err)
				continue
			}

			// Verify settings
			actualVolume, err := channels[i].GetVolume()
			if err != nil {
				t.Errorf("Failed to get volume for channel %d: %v", i, err)
			} else if abs(actualVolume-config.volume) > 0.01 {
				t.Errorf("Channel %d volume mismatch: expected %.2f, got %.2f", i, config.volume, actualVolume)
			}

			actualPan, err := channels[i].GetPan()
			if err != nil {
				t.Errorf("Failed to get pan for channel %d: %v", i, err)
			} else if abs(actualPan-config.pan) > 0.01 {
				t.Errorf("Channel %d pan mismatch: expected %.2f, got %.2f", i, config.pan, actualPan)
			}

			t.Logf("Channel %d configured: volume=%.2f, pan=%.2f", i, config.volume, config.pan)
		}

		// Verify other channels weren't affected
		for i := 0; i < len(channels); i++ {
			volume, err := channels[i].GetVolume()
			if err != nil {
				t.Errorf("Failed to read volume for channel %d after other channel changes: %v", i, err)
				continue
			}

			expectedVolume := testConfigs[i].volume
			if abs(volume-expectedVolume) > 0.01 {
				t.Errorf("Channel isolation failed: channel %d volume changed unexpectedly to %.2f (expected %.2f)",
					i, volume, expectedVolume)
			}
		}

		t.Log("✅ Channel isolation verified - changes to one channel don't affect others")
	})

	t.Run("SimultaneousPlayback", func(t *testing.T) {
		// Start playback on all channels
		for i, channel := range channels {
			err = channel.Play()
			if err != nil {
				t.Errorf("Failed to start playback on channel %d: %v", i, err)
			} else {
				t.Logf("Started playback on channel %d", i)
			}
		}

		// Let them play briefly
		time.Sleep(100 * time.Millisecond)

		// All should be playing
		for i, channel := range channels {
			// Note: We could add IsPlaying() method to channels if needed
			// For now, just verify the channel state is consistent
			if !channel.IsPlayback() {
				t.Errorf("Channel %d should be a playback channel", i)
			}
		}

		t.Log("✅ All channels started playback successfully")
	})
}

// TestChannelCapacity tests behavior with many channels
func TestChannelCapacity(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	maxChannels := 8 // Test with reasonable number
	channels := make([]*Channel, 0, maxChannels)

	t.Run("CreateManyChannels", func(t *testing.T) {
		for i := 0; i < maxChannels; i++ {
			channel, err := engine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
			if err != nil {
				t.Fatalf("Failed to create channel %d: %v", i, err)
			}
			channels = append(channels, channel)
			t.Logf("Created channel %d", i)
		}

		if len(engine.Channels) != maxChannels {
			t.Errorf("Engine should have %d channels, got %d", maxChannels, len(engine.Channels))
		}

		t.Logf("✅ Successfully created %d channels", maxChannels)
	})

	t.Run("StartEngineWithManyChannels", func(t *testing.T) {
		err := engine.Start()
		if err != nil {
			t.Fatalf("Failed to start engine with %d channels: %v", maxChannels, err)
		}

		t.Logf("✅ Engine started with %d channels", maxChannels)
	})

	t.Run("ConfigureAllChannels", func(t *testing.T) {
		for i, channel := range channels {
			// Set unique volume and pan for each channel
			volume := 0.5 + (float32(i) * 0.1) // 0.5 to 1.2
			if volume > 1.0 {
				volume = 1.0
			}
			pan := -1.0 + (float32(i) * 0.25) // -1.0 to 0.75

			err := channel.SetVolume(volume)
			if err != nil {
				t.Errorf("Failed to set volume on channel %d: %v", i, err)
			}

			err = channel.SetPan(pan)
			if err != nil {
				t.Errorf("Failed to set pan on channel %d: %v", i, err)
			}
		}

		t.Logf("✅ Configured %d channels with unique settings", len(channels))
	})
}

// TestChannelCleanup tests proper resource cleanup when destroying channels
func TestChannelCleanup(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	t.Run("CreateAndDestroyChannels", func(t *testing.T) {
		initialChannelCount := len(engine.Channels)

		// Create some channels
		channels := make([]*Channel, 3)
		for i := 0; i < 3; i++ {
			var err error
			channels[i], err = engine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
			if err != nil {
				t.Fatalf("Failed to create channel %d: %v", i, err)
			}
		}

		if len(engine.Channels) != initialChannelCount+3 {
			t.Errorf("Expected %d channels, got %d", initialChannelCount+3, len(engine.Channels))
		}

		// Start engine to ensure nodes are connected
		err := engine.Start()
		if err != nil {
			t.Fatalf("Failed to start engine: %v", err)
		}

		// Destroy middle channel
		err = engine.DestroyChannel(initialChannelCount + 1) // Middle channel
		if err != nil {
			t.Errorf("Failed to destroy channel: %v", err)
		}

		if len(engine.Channels) != initialChannelCount+2 {
			t.Errorf("Expected %d channels after destroy, got %d", initialChannelCount+2, len(engine.Channels))
		}

		t.Log("✅ Channel destruction works correctly")
	})
}

// Helper function for float comparison
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
