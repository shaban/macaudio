package engine

import (
	"testing"

	"github.com/shaban/macaudio/devices"
)

// testDeviceSetup is a helper function to get devices for testing
func testDeviceSetup(t *testing.T) (*devices.AudioDevice, *devices.AudioDevice) {
	allDevices, err := devices.GetAudio()
	if err != nil {
		t.Skip("No devices available for testing")
	}

	// Find an output device
	var outputDevice *devices.AudioDevice
	for i, device := range allDevices {
		if device.CanOutput() && len(device.SupportedSampleRates) > 0 {
			outputDevice = &allDevices[i]
			break
		}
	}

	if outputDevice == nil {
		t.Skip("No output devices available for testing")
	}

	// Find an input device
	var inputDevice *devices.AudioDevice
	for i, device := range allDevices {
		if device.CanInput() {
			inputDevice = &allDevices[i]
			break
		}
	}

	return outputDevice, inputDevice
}

// TestChannelTypeDetection verifies that channel types are correctly determined
// by the presence of options, and that options are never nil when created
func TestChannelTypeDetection(t *testing.T) {
	outputDevice, inputDevice := testDeviceSetup(t)

	engine, err := NewEngine(outputDevice, 0, 512)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	// Use inputDevice if we need input functionality
	if inputDevice == nil {
		t.Skip("No input devices available for testing")
	}

	t.Run("InputChannel", func(t *testing.T) {
		// Create input channel
		channel, err := engine.CreateInputChannel(inputDevice, 0)
		if err != nil {
			t.Fatalf("Failed to create input channel: %v", err)
		}

		// Verify channel type detection
		if !channel.IsInput() {
			t.Error("Expected IsInput() to return true")
		}
		if channel.IsPlayback() {
			t.Error("Expected IsPlayback() to return false")
		}

		// Verify InputOptions is never nil
		if channel.InputOptions == nil {
			t.Error("InputOptions should never be nil for input channel")
		}

		// Verify PlaybackOptions is nil
		if channel.PlaybackOptions != nil {
			t.Error("PlaybackOptions should be nil for input channel")
		}

		// Verify InputOptions has required fields
		if channel.InputOptions.Device == nil {
			t.Error("InputOptions.Device should not be nil")
		}
		if channel.InputOptions.PluginChain == nil {
			t.Error("InputOptions.PluginChain should not be nil (even if empty)")
		}
	})

	t.Run("PlaybackChannel", func(t *testing.T) {
		// Create playback channel
		channel, err := engine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
		if err != nil {
			t.Fatalf("Failed to create playback channel: %v", err)
		}

		// Verify channel type detection
		if channel.IsInput() {
			t.Error("Expected IsInput() to return false")
		}
		if !channel.IsPlayback() {
			t.Error("Expected IsPlayback() to return true")
		}

		// Verify PlaybackOptions is never nil
		if channel.PlaybackOptions == nil {
			t.Error("PlaybackOptions should never be nil for playback channel")
		}

		// Verify InputOptions is nil
		if channel.InputOptions != nil {
			t.Error("InputOptions should be nil for playback channel")
		}

		// Verify PlaybackOptions has required fields
		if channel.PlaybackOptions.FilePath == "" {
			t.Error("PlaybackOptions.FilePath should not be empty")
		}
		if channel.PlaybackOptions.Rate != 1.0 {
			t.Error("PlaybackOptions.Rate should default to 1.0")
		}
		if channel.PlaybackOptions.Pitch != 0.0 {
			t.Error("PlaybackOptions.Pitch should default to 0.0")
		}
	})

	t.Run("ChannelTypeInvariant", func(t *testing.T) {
		// Test that we can never have both options set or both nil
		// (This is enforced by our API, not runtime checks)

		// Manual construction to test invariant
		validInputChannel := &Channel{
			BusIndex: 0,
			Volume:   1.0,
			Pan:      0.0,
			InputOptions: &InputOptions{
				Device:       inputDevice,
				ChannelIndex: 0,
				PluginChain:  NewPluginChain(),
			},
			// PlaybackOptions is nil - this is valid
		}

		if !validInputChannel.IsInput() {
			t.Error("Valid input channel should be detected as input")
		}
		if validInputChannel.IsPlayback() {
			t.Error("Valid input channel should not be detected as playback")
		}

		validPlaybackChannel := &Channel{
			BusIndex: 1,
			Volume:   1.0,
			Pan:      0.0,
			// InputOptions is nil - this is valid
			PlaybackOptions: &PlaybackOptions{
				FilePath: "/path/to/file.wav",
				Rate:     1.0,
				Pitch:    0.0,
			},
		}

		if validPlaybackChannel.IsInput() {
			t.Error("Valid playback channel should not be detected as input")
		}
		if !validPlaybackChannel.IsPlayback() {
			t.Error("Valid playback channel should be detected as playback")
		}

		// Edge case: Both nil (uninitialized channel)
		emptyChannel := &Channel{
			BusIndex: 2,
			Volume:   1.0,
			Pan:      0.0,
			// Both options are nil
		}

		if emptyChannel.IsInput() {
			t.Error("Empty channel should not be detected as input")
		}
		if emptyChannel.IsPlayback() {
			t.Error("Empty channel should not be detected as playback")
		}
	})
}

// TestChannelCreationGuarantees verifies that our channel creation methods
// always initialize options and never leave them nil
func TestChannelCreationGuarantees(t *testing.T) {
	outputDevice, inputDevice := testDeviceSetup(t)

	engine, err := NewEngine(outputDevice, 0, 512)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Destroy()

	if inputDevice == nil {
		t.Skip("No input devices available for testing")
	}

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
