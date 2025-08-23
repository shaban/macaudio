package engine

import (
	"encoding/json"
	"testing"

	"github.com/shaban/macaudio/devices"
)

// TestChannelSerialization tests JSON serialization for all channel types
func TestChannelSerialization(t *testing.T) {
	tests := []struct {
		name    string
		channel *Channel
	}{
		{
			name: "Playback Channel",
			channel: &Channel{
				Volume: 0.8,
				Pan:    -0.2,
				PlaybackOptions: &PlaybackOptions{
					FilePath: "/path/to/audio.wav",
					Rate:     1.0,
					Pitch:    0.0,
				},
			},
		},
		{
			name: "Audio Input Channel",
			channel: &Channel{
				Volume: 0.9,
				Pan:    0.1,
				InputOptions: &InputOptions{
					Device: &devices.AudioDevice{
						Device: devices.Device{
							Name:     "Test Audio Device",
							UID:      "test-audio-uid",
							IsOnline: true,
						},
						DeviceID:          42,
						InputChannelCount: 2,
					},
					ChannelIndex: 1,
				},
			},
		},
		{
			name: "MIDI Input Channel",
			channel: &Channel{
				Volume: 1.0,
				Pan:    0.0,
				InputOptions: &InputOptions{
					MidiDevice: &devices.MIDIDevice{
						Device: devices.Device{
							Name:     "Test MIDI Device",
							UID:      "test-midi-uid",
							IsOnline: true,
						},
						DeviceName:      "Test MIDI Controller",
						Manufacturer:    "TestCorp",
						Model:           "TestModel",
						InputEndpointID: 123,
						IsInput:         true,
					},
					ChannelIndex: 1, // MIDI channel 1
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			jsonData, err := json.Marshal(tt.channel)
			if err != nil {
				t.Fatalf("Failed to marshal channel: %v", err)
			}

			// Test deserialization
			var restored Channel
			err = json.Unmarshal(jsonData, &restored)
			if err != nil {
				t.Fatalf("Failed to unmarshal channel: %v", err)
			}

			// Basic validation
			if restored.Volume != tt.channel.Volume {
				t.Errorf("Volume mismatch: got %f, want %f", restored.Volume, tt.channel.Volume)
			}
			if restored.Pan != tt.channel.Pan {
				t.Errorf("Pan mismatch: got %f, want %f", restored.Pan, tt.channel.Pan)
			}

			// Type-specific validation
			if tt.channel.IsPlayback() {
				if !restored.IsPlayback() {
					t.Error("Restored channel should be playback type")
				}
				if restored.PlaybackOptions.FilePath != tt.channel.PlaybackOptions.FilePath {
					t.Errorf("FilePath mismatch: got %s, want %s",
						restored.PlaybackOptions.FilePath, tt.channel.PlaybackOptions.FilePath)
				}
			}

			if tt.channel.IsAudioInput() {
				if !restored.IsAudioInput() {
					t.Error("Restored channel should be audio input type")
				}
				if restored.InputOptions.Device.Name != tt.channel.InputOptions.Device.Name {
					t.Errorf("Audio device name mismatch: got %s, want %s",
						restored.InputOptions.Device.Name, tt.channel.InputOptions.Device.Name)
				}
			}

			if tt.channel.IsMIDIInput() {
				if !restored.IsMIDIInput() {
					t.Error("Restored channel should be MIDI input type")
				}
				if restored.InputOptions.MidiDevice.Name != tt.channel.InputOptions.MidiDevice.Name {
					t.Errorf("MIDI device name mismatch: got %s, want %s",
						restored.InputOptions.MidiDevice.Name, tt.channel.InputOptions.MidiDevice.Name)
				}
			}

			t.Logf("Successfully serialized/deserialized %s: %s", tt.name, string(jsonData))
		})
	}
}

// TestUnifiedChannelTypeDetection tests the helper methods for channel type detection with MIDI support
func TestUnifiedChannelTypeDetection(t *testing.T) {
	// Playback channel
	playbackCh := &Channel{
		PlaybackOptions: &PlaybackOptions{FilePath: "test.wav"},
	}
	if !playbackCh.IsPlayback() {
		t.Error("Should detect playback channel")
	}
	if playbackCh.IsInput() {
		t.Error("Playback channel should not be input")
	}
	if playbackCh.IsAudioInput() {
		t.Error("Playback channel should not be audio input")
	}
	if playbackCh.IsMIDIInput() {
		t.Error("Playback channel should not be MIDI input")
	}

	// Audio input channel
	audioInputCh := &Channel{
		InputOptions: &InputOptions{
			Device: &devices.AudioDevice{
				Device: devices.Device{Name: "Test Audio"},
			},
		},
	}
	if audioInputCh.IsPlayback() {
		t.Error("Audio input channel should not be playback")
	}
	if !audioInputCh.IsInput() {
		t.Error("Should detect input channel")
	}
	if !audioInputCh.IsAudioInput() {
		t.Error("Should detect audio input channel")
	}
	if audioInputCh.IsMIDIInput() {
		t.Error("Audio input channel should not be MIDI input")
	}

	// MIDI input channel
	midiInputCh := &Channel{
		InputOptions: &InputOptions{
			MidiDevice: &devices.MIDIDevice{
				Device: devices.Device{Name: "Test MIDI"},
			},
		},
	}
	if midiInputCh.IsPlayback() {
		t.Error("MIDI input channel should not be playback")
	}
	if !midiInputCh.IsInput() {
		t.Error("Should detect input channel")
	}
	if midiInputCh.IsAudioInput() {
		t.Error("MIDI input channel should not be audio input")
	}
	if !midiInputCh.IsMIDIInput() {
		t.Error("Should detect MIDI input channel")
	}
}
