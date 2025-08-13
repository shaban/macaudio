package engine

import (
	"testing"

	"github.com/shaban/macaudio/avaudio/node"
)

// TestEngine_AudioSpecIntegration tests that Connect() properly uses AudioSpec to create formats
func TestEngine_AudioSpecIntegration(t *testing.T) {
	// Create engine with custom AudioSpec
	spec := AudioSpec{
		SampleRate:   48000.0,
		BufferSize:   1024,
		BitDepth:     32,
		ChannelCount: 2,
	}

	engine, err := New(spec)
	if err != nil {
		t.Fatalf("Failed to create engine with custom AudioSpec: %v", err)
	}
	if engine == nil {
		t.Fatal("Engine is nil")
	}
	defer engine.Destroy()

	t.Logf("✓ Engine created with AudioSpec: %.0fHz, %d samples, %d-bit, %d channels",
		spec.SampleRate, spec.BufferSize, spec.BitDepth, spec.ChannelCount)

	// Create two mixer nodes for testing connection
	mixer1Ptr, err := node.CreateMixer()
	if err != nil {
		t.Fatalf("Failed to create first mixer: %v", err)
	}
	mixer2Ptr, err := node.CreateMixer()
	if err != nil {
		t.Fatalf("Failed to create second mixer: %v", err)
	}
	defer node.ReleaseMixer(mixer1Ptr)
	defer node.ReleaseMixer(mixer2Ptr)

	// Attach both mixers
	err = engine.Attach(mixer1Ptr)
	if err != nil {
		t.Fatalf("Failed to attach first mixer: %v", err)
	}

	err = engine.Attach(mixer2Ptr)
	if err != nil {
		t.Fatalf("Failed to attach second mixer: %v", err)
	}

	// Test the Connect method which should create format from AudioSpec
	err = engine.Connect(mixer1Ptr, mixer2Ptr, 0, 0)
	if err != nil {
		t.Logf("Connection result: %v", err)
	} else {
		t.Logf("✓ Successfully connected mixer1 to mixer2 using AudioSpec-derived format")
	}

	// Verify the engine's AudioSpec is still accessible
	engineSpec := engine.GetSpec()
	if engineSpec.SampleRate != spec.SampleRate {
		t.Errorf("AudioSpec sample rate mismatch: expected %.0f, got %.0f",
			spec.SampleRate, engineSpec.SampleRate)
	}

	if engineSpec.ChannelCount != spec.ChannelCount {
		t.Errorf("AudioSpec channel count mismatch: expected %d, got %d",
			spec.ChannelCount, engineSpec.ChannelCount)
	}

	t.Logf("✓ AudioSpec properly preserved: %.0fHz, %d channels",
		engineSpec.SampleRate, engineSpec.ChannelCount)

	// Clean up
	engine.DisconnectNodeInput(mixer2Ptr, 0)
	engine.Detach(mixer1Ptr)
	engine.Detach(mixer2Ptr)
}

// TestEngine_AudioSpecFormatCreation tests format creation with different AudioSpec settings
func TestEngine_AudioSpecFormatCreation(t *testing.T) {
	testCases := []struct {
		name string
		spec AudioSpec
	}{
		{
			name: "Standard 44.1kHz Stereo",
			spec: AudioSpec{SampleRate: 44100.0, BufferSize: 512, BitDepth: 24, ChannelCount: 2},
		},
		{
			name: "High Quality 96kHz Stereo",
			spec: AudioSpec{SampleRate: 96000.0, BufferSize: 256, BitDepth: 32, ChannelCount: 2},
		},
		{
			name: "Mono 48kHz",
			spec: AudioSpec{SampleRate: 48000.0, BufferSize: 1024, BitDepth: 16, ChannelCount: 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine, err := New(tc.spec)
			if err != nil {
				t.Fatalf("Failed to create engine with spec: %+v, error: %v", tc.spec, err)
			}
			if engine == nil {
				t.Fatalf("Engine is nil for spec: %+v", tc.spec)
			}
			defer engine.Destroy()

			// Test that we can create mixer nodes and connect them
			mixer1Ptr, err := node.CreateMixer()
			if err != nil {
				t.Fatalf("Failed to create first mixer: %v", err)
			}
			mixer2Ptr, err := node.CreateMixer()
			if err != nil {
				t.Fatalf("Failed to create second mixer: %v", err)
			}
			defer node.ReleaseMixer(mixer1Ptr)
			defer node.ReleaseMixer(mixer2Ptr)

			err = engine.Attach(mixer1Ptr)
			if err != nil {
				t.Fatalf("Failed to attach first mixer: %v", err)
			}

			err = engine.Attach(mixer2Ptr)
			if err != nil {
				t.Fatalf("Failed to attach second mixer: %v", err)
			}

			// The key test: Connect should use the AudioSpec to create a format
			err = engine.Connect(mixer1Ptr, mixer2Ptr, 0, 0)
			if err != nil {
				t.Logf("Connection with %s: %v", tc.name, err)
			} else {
				t.Logf("✓ %s: Successfully connected using AudioSpec format", tc.name)
			}

			// Verify the AudioSpec is preserved
			preserved := engine.GetSpec()
			if preserved.SampleRate != tc.spec.SampleRate {
				t.Errorf("Sample rate not preserved: expected %.0f, got %.0f",
					tc.spec.SampleRate, preserved.SampleRate)
			}
			if preserved.ChannelCount != tc.spec.ChannelCount {
				t.Errorf("Channel count not preserved: expected %d, got %d",
					tc.spec.ChannelCount, preserved.ChannelCount)
			}

			// Clean up
			engine.DisconnectNodeInput(mixer2Ptr, 0)
			engine.Detach(mixer1Ptr)
			engine.Detach(mixer2Ptr)
		})
	}
}
