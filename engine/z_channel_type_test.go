package engine

import (
	"testing"
)

// TestChannelTypeDetection verifies that channel types are correctly determined
// by the presence of options, and that options are never nil when created
func TestChannelTypeDetection(t *testing.T) {
	_, inputDevice := TestDeviceSetup(t)
	if inputDevice == nil {
		t.Skip("No input devices available for testing")
	}

	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	tests := []struct {
		name          string
		setupChannel  func() (*Channel, error)
		expectedState ExpectedChannelState
	}{
		{
			name: "InputChannel",
			setupChannel: func() (*Channel, error) {
				return engine.CreateInputChannel(inputDevice, 0)
			},
			expectedState: ExpectedChannelState{
				IsInput:            true,
				IsPlayback:         false,
				Volume:             1.0, // Default volume for input channels
				Pan:                0.0, // Default pan
				HasInputOptions:    true,
				HasPlaybackOptions: false,
				PluginCount:        0, // No plugins by default
			},
		},
		{
			name: "PlaybackChannel",
			setupChannel: func() (*Channel, error) {
				return engine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
			},
			expectedState: ExpectedChannelState{
				IsInput:            false,
				IsPlayback:         true,
				Volume:             1.0, // Default volume for playback channels
				Pan:                0.0, // Default pan
				HasInputOptions:    false,
				HasPlaybackOptions: true,
				PluginCount:        0, // Playback channels don't support plugins per MVP
				FilePath:           "/System/Library/Sounds/Ping.aiff",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := tt.setupChannel()
			if err != nil {
				t.Fatalf("Failed to create channel: %v", err)
			}

			ValidateChannelState(t, channel, tt.expectedState)

			// Additional specific validations for channel creation guarantees
			if tt.expectedState.HasInputOptions {
				if channel.InputOptions.Device == nil {
					t.Error("InputOptions.Device should not be nil")
				}
				if channel.InputOptions.PluginChain == nil {
					t.Error("InputOptions.PluginChain should not be nil (even if empty)")
				}
			}

			if tt.expectedState.HasPlaybackOptions {
				if channel.PlaybackOptions.Rate != 1.0 {
					t.Error("PlaybackOptions.Rate should default to 1.0")
				}
				if channel.PlaybackOptions.Pitch != 0.0 {
					t.Error("PlaybackOptions.Pitch should default to 0.0")
				}
			}
		})
	}
}

// TestChannelTypeInvariant tests manual channel construction edge cases
func TestChannelTypeInvariant(t *testing.T) {
	_, inputDevice := TestDeviceSetup(t)
	if inputDevice == nil {
		t.Skip("No input devices available for testing")
	}

	tests := []struct {
		name          string
		channel       *Channel
		expectedState ExpectedChannelState
	}{
		{
			name: "ValidInputChannel",
			channel: &Channel{
				Volume: 1.0,
				Pan:    0.0,
				InputOptions: &InputOptions{
					Device:       inputDevice,
					ChannelIndex: 0,
					PluginChain:  NewPluginChain(),
				},
				// PlaybackOptions is nil - this is valid
			},
			expectedState: ExpectedChannelState{
				IsInput:            true,
				IsPlayback:         false,
				Volume:             1.0,
				Pan:                0.0,
				HasInputOptions:    true,
				HasPlaybackOptions: false,
				PluginCount:        0,
			},
		},
		{
			name: "ValidPlaybackChannel",
			channel: &Channel{
				Volume: 1.0,
				Pan:    0.0,
				// InputOptions is nil - this is valid
				PlaybackOptions: &PlaybackOptions{
					FilePath: "/path/to/file.wav",
					Rate:     1.0,
					Pitch:    0.0,
				},
			},
			expectedState: ExpectedChannelState{
				IsInput:            false,
				IsPlayback:         true,
				Volume:             1.0,
				Pan:                0.0,
				HasInputOptions:    false,
				HasPlaybackOptions: true,
				PluginCount:        0,
				FilePath:           "/path/to/file.wav",
			},
		},
		{
			name: "EmptyChannel",
			channel: &Channel{
				Volume: 1.0,
				Pan:    0.0,
				// Both options are nil
			},
			expectedState: ExpectedChannelState{
				IsInput:            false,
				IsPlayback:         false,
				Volume:             1.0,
				Pan:                0.0,
				HasInputOptions:    false,
				HasPlaybackOptions: false,
				PluginCount:        0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ValidateChannelState(t, tt.channel, tt.expectedState)
		})
	}
}

// TestChannelCreationGuarantees verifies that our channel creation methods
// always initialize options and never leave them nil
func TestChannelCreationGuarantees(t *testing.T) {
	_, inputDevice := TestDeviceSetup(t)
	if inputDevice == nil {
		t.Skip("No input devices available for testing")
	}

	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	t.Run("InputChannelGuarantees", func(t *testing.T) {
		channel, err := engine.CreateInputChannel(inputDevice, 0)
		if err != nil {
			t.Fatalf("Failed to create input channel: %v", err)
		}

		// These are GUARANTEES our API provides
		if channel.InputOptions == nil {
			t.Fatal("GUARANTEE VIOLATED: InputOptions must never be nil for input channels")
		}
		if channel.InputOptions.Device == nil {
			t.Fatal("GUARANTEE VIOLATED: InputOptions.Device must never be nil")
		}
		if channel.InputOptions.PluginChain == nil {
			t.Fatal("GUARANTEE VIOLATED: InputOptions.PluginChain must never be nil (can be empty)")
		}
		if channel.PlaybackOptions != nil {
			t.Fatal("GUARANTEE VIOLATED: PlaybackOptions must be nil for input channels")
		}
	})

	t.Run("PlaybackChannelGuarantees", func(t *testing.T) {
		channel, err := engine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
		if err != nil {
			t.Fatalf("Failed to create playback channel: %v", err)
		}

		// These are GUARANTEES our API provides
		if channel.PlaybackOptions == nil {
			t.Fatal("GUARANTEE VIOLATED: PlaybackOptions must never be nil for playback channels")
		}
		if channel.PlaybackOptions.FilePath == "" {
			t.Fatal("GUARANTEE VIOLATED: PlaybackOptions.FilePath must not be empty")
		}
		if channel.InputOptions != nil {
			t.Fatal("GUARANTEE VIOLATED: InputOptions must be nil for playback channels")
		}
	})
}
