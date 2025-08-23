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

// TestCreateMIDIInputChannel tests the new CreateMIDIInputChannel method
func TestCreateMIDIInputChannel(t *testing.T) {
	// Create test engine
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}
	if len(audioDevices) == 0 {
		t.Skip("No audio devices available for testing")
	}

	outputDevice := &audioDevices[0]
	engine, err := NewEngine(outputDevice, 0, 512)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Create test MIDI device
	testMIDIDevice := &devices.MIDIDevice{
		Device: devices.Device{
			Name:     "Test MIDI Controller",
			UID:      "test-midi-uid",
			IsOnline: true,
		},
		DeviceName:      "Virtual MIDI",
		Manufacturer:    "TestCorp",
		Model:           "TestController",
		InputEndpointID: 123,
		IsInput:         true,
	}

	// Test creating MIDI input channel
	channel, err := engine.CreateMIDIInputChannel(testMIDIDevice, 1)
	if err != nil {
		t.Fatalf("Failed to create MIDI input channel: %v", err)
	}

	// Validate the channel
	if channel == nil {
		t.Fatal("Created channel is nil")
	}

	if !channel.IsMIDIInput() {
		t.Error("Channel should be detected as MIDI input")
	}

	if channel.IsAudioInput() {
		t.Error("Channel should not be detected as audio input")
	}

	if channel.IsPlayback() {
		t.Error("Channel should not be detected as playback")
	}

	// Validate the channel configuration
	if channel.Volume != 1.0 {
		t.Errorf("Expected default volume 1.0, got %f", channel.Volume)
	}

	if channel.Pan != 0.0 {
		t.Errorf("Expected default pan 0.0, got %f", channel.Pan)
	}

	if channel.InputOptions == nil {
		t.Fatal("InputOptions should not be nil")
	}

	if channel.InputOptions.MidiDevice != testMIDIDevice {
		t.Error("MIDI device not properly set")
	}

	if channel.InputOptions.ChannelIndex != 1 {
		t.Errorf("Expected MIDI channel 1, got %d", channel.InputOptions.ChannelIndex)
	}

	// Validate engine state
	if len(engine.Channels) != 1 {
		t.Errorf("Expected 1 channel in engine, got %d", len(engine.Channels))
	}

	if engine.Channels[0] != channel {
		t.Error("Channel not properly added to engine")
	}

	t.Logf("✅ Successfully created MIDI input channel for device %s on MIDI channel %d",
		testMIDIDevice.Name, channel.InputOptions.ChannelIndex)
}

// TestMixedChannelTypes tests creating different channel types in the same engine
func TestMixedChannelTypes(t *testing.T) {
	// Create test engine
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}
	if len(audioDevices) == 0 {
		t.Skip("No audio devices available for testing")
	}

	outputDevice := &audioDevices[0]
	engine, err := NewEngine(outputDevice, 0, 512)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// 1. Create a playback channel
	playbackChannel, err := engine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
	if err != nil {
		t.Fatalf("Failed to create playback channel: %v", err)
	}

	// 2. Create an audio input channel (if we have input capability)
	var audioInputChannel *Channel
	for _, device := range audioDevices {
		if device.CanInput() {
			audioInputChannel, err = engine.CreateInputChannel(&device, 0)
			if err != nil {
				t.Fatalf("Failed to create audio input channel: %v", err)
			}
			break
		}
	}

	// 3. Create a MIDI input channel
	testMIDIDevice := &devices.MIDIDevice{
		Device: devices.Device{
			Name:     "Virtual MIDI",
			UID:      "test-midi-uid",
			IsOnline: true,
		},
		DeviceName:   "Test MIDI",
		Manufacturer: "TestCorp",
		IsInput:      true,
	}

	midiInputChannel, err := engine.CreateMIDIInputChannel(testMIDIDevice, 1)
	if err != nil {
		t.Fatalf("Failed to create MIDI input channel: %v", err)
	}

	// Validate we have the right number of channels
	expectedChannels := 2 // playback + MIDI
	if audioInputChannel != nil {
		expectedChannels = 3 // playback + audio input + MIDI
	}

	if len(engine.Channels) != expectedChannels {
		t.Fatalf("Expected %d channels, got %d", expectedChannels, len(engine.Channels))
	}

	// Validate each channel type
	if !playbackChannel.IsPlayback() {
		t.Error("First channel should be playback type")
	}

	if audioInputChannel != nil && !audioInputChannel.IsAudioInput() {
		t.Error("Second channel should be audio input type")
	}

	if !midiInputChannel.IsMIDIInput() {
		t.Error("Last channel should be MIDI input type")
	}

	// Test serialization with mixed channel types
	jsonData, err := engine.SerializeState()
	if err != nil {
		t.Fatalf("Failed to serialize engine with mixed channel types: %v", err)
	}

	// Test deserialization
	var restoredEngine Engine
	err = restoredEngine.DeserializeState(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize engine with mixed channel types: %v", err)
	}

	// Validate restored state
	if len(restoredEngine.Channels) != expectedChannels {
		t.Errorf("After deserialization: expected %d channels, got %d", expectedChannels, len(restoredEngine.Channels))
	}

	t.Logf("✅ Successfully created and serialized engine with %d mixed channel types", expectedChannels)
}
