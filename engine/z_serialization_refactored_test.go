package engine

import (
	"encoding/json"
	"testing"
)

// TestBasicSerializationRoundtrip demonstrates focused serialization testing with helpers
func TestBasicSerializationRoundtrip(t *testing.T) {
	tests := []struct {
		name   string
		config SerializationTestConfig
	}{
		{
			name:   "MinimalEngine",
			config: DefaultSerializationTestConfig(),
		},
		{
			name: "EngineWithMultipleChannels",
			config: SerializationTestConfig{
				EngineConfig: TestEngineConfig{
					MasterVolume: 0.6,
					SampleRate:   0, // Use device default
					BufferSize:   256,
				},
				InputChannelConfigs: []TestChannelConfig{
					{
						Volume:      0.8,
						Pan:         -0.2,
						PluginCount: 1,
						UseRealFile: true,
					},
				},
				PlaybackChannelConfigs: []TestChannelConfig{
					{
						Volume:      0.7,
						Pan:         0.1,
						PluginCount: 0,
						UseRealFile: true,
					},
					{
						Volume:      0.9,
						Pan:         -0.4,
						PluginCount: 0,
						UseRealFile: false,
					},
				},
				ExpectedChannelCount: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create complex engine with specified configuration
			originalEngine, cleanup := CreateComplexTestEngine(t, tt.config)
			defer cleanup()

			// Serialize to JSON
			jsonData, err := json.MarshalIndent(originalEngine, "", "  ")
			if err != nil {
				t.Fatalf("Failed to serialize engine: %v", err)
			}

			t.Logf("Serialized JSON size: %d bytes", len(jsonData))

			// Deserialize from JSON
			var deserializedEngine Engine
			if err := json.Unmarshal(jsonData, &deserializedEngine); err != nil {
				t.Fatalf("Failed to deserialize engine: %v", err)
			}

			// Validate deserialized engine state
			validateDeserializedEngine(t, originalEngine, &deserializedEngine, tt.config)
		})
	}
}

// validateDeserializedEngine compares original and deserialized engines
func validateDeserializedEngine(t *testing.T, original, deserialized *Engine, config SerializationTestConfig) {
	// Validate basic engine properties
	if deserialized.MasterVolume != original.MasterVolume {
		t.Errorf("MasterVolume mismatch: expected %v, got %v", original.MasterVolume, deserialized.MasterVolume)
	}

	// Validate channel count
	actualChannelCount := len(deserialized.Channels)
	if actualChannelCount != len(original.Channels) {
		t.Errorf("Channel count mismatch: expected %v, got %v", len(original.Channels), actualChannelCount)
	}

	// Validate individual channels using our helper
	for i, channel := range deserialized.Channels {
		if i >= len(original.Channels) {
			t.Errorf("Unexpected extra channel at index %d", i)
			continue
		}

		originalChannel := original.Channels[i]

		expectedState := ExpectedChannelState{
			IsInput:            originalChannel.IsInput(),
			IsPlayback:         originalChannel.IsPlayback(),
			Volume:             originalChannel.Volume,
			Pan:                originalChannel.Pan,
			HasInputOptions:    originalChannel.InputOptions != nil,
			HasPlaybackOptions: originalChannel.PlaybackOptions != nil,
		}

		// Set plugin count and file path expectations
		if originalChannel.InputOptions != nil && originalChannel.InputOptions.PluginChain != nil {
			expectedState.PluginCount = len(originalChannel.InputOptions.PluginChain.Plugins)
		}
		if originalChannel.PlaybackOptions != nil {
			expectedState.FilePath = originalChannel.PlaybackOptions.FilePath
		}

		ValidateChannelState(t, channel, expectedState)
	}
}

// TestSerializationEdgeCases tests specific serialization edge cases
func TestSerializationEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) (*Engine, func())
		validator func(t *testing.T, original, deserialized *Engine)
	}{
		{
			name: "EmptyEngine",
			setupFunc: func(t *testing.T) (*Engine, func()) {
				config := TestEngineConfig{
					MasterVolume: 1.0,
					SampleRate:   0,
					BufferSize:   512,
				}
				return CreateTestEngine(t, config)
			},
			validator: func(t *testing.T, original, deserialized *Engine) {
				if len(deserialized.Channels) != 0 {
					t.Errorf("Expected empty channel list, got %d channels", len(deserialized.Channels))
				}
				if deserialized.MasterVolume != 1.0 {
					t.Errorf("Expected MasterVolume=1.0, got %v", deserialized.MasterVolume)
				}
			},
		},
		{
			name: "ExtremeValues",
			setupFunc: func(t *testing.T) (*Engine, func()) {
				config := TestEngineConfig{
					MasterVolume: 0.0, // Minimum volume
					SampleRate:   0,
					BufferSize:   512,
				}
				engine, cleanup := CreateTestEngine(t, config)

				// Add playback channel with extreme pan value
				channelConfig := TestChannelConfig{
					Volume:      0.01, // Very low volume
					Pan:         1.0,  // Full right pan
					PluginCount: 0,
					UseRealFile: true,
				}
				CreateTestPlaybackChannel(t, engine, channelConfig)

				return engine, cleanup
			},
			validator: func(t *testing.T, original, deserialized *Engine) {
				if deserialized.MasterVolume != 0.0 {
					t.Errorf("Expected MasterVolume=0.0, got %v", deserialized.MasterVolume)
				}
				if len(deserialized.Channels) != 1 {
					t.Errorf("Expected 1 channel, got %d", len(deserialized.Channels))
				}
				if len(deserialized.Channels) > 0 {
					ch := deserialized.Channels[0]
					if ch.Volume != 0.01 {
						t.Errorf("Expected channel Volume=0.01, got %v", ch.Volume)
					}
					if ch.Pan != 1.0 {
						t.Errorf("Expected channel Pan=1.0, got %v", ch.Pan)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original, cleanup := tt.setupFunc(t)
			defer cleanup()

			// Serialize
			jsonData, err := json.MarshalIndent(original, "", "  ")
			if err != nil {
				t.Fatalf("Failed to serialize engine: %v", err)
			}

			// Deserialize
			var deserialized Engine
			if err := json.Unmarshal(jsonData, &deserialized); err != nil {
				t.Fatalf("Failed to deserialize engine: %v", err)
			}

			// Custom validation for this test case
			tt.validator(t, original, &deserialized)
		})
	}
}
