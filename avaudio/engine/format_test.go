package engine

import (
	"testing"
)

// TestFormatIntegration tests the consolidated format functionality
func TestFormatIntegration(t *testing.T) {
	t.Log("🔧 Testing format integration into engine package")

	// Create an engine for format creation
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Test 1: Create mono format
	t.Log("📼 Test 1: Creating mono format")
	monoFormat, err := engine.NewMonoFormat(44100)
	if err != nil {
		t.Fatal("Failed to create mono format:", err)
	}
	defer monoFormat.Destroy()

	if monoFormat.SampleRate() != 44100 {
		t.Errorf("Expected sample rate 44100, got %.0f", monoFormat.SampleRate())
	}

	if monoFormat.ChannelCount() != 1 {
		t.Errorf("Expected 1 channel, got %d", monoFormat.ChannelCount())
	}
	t.Log("   ✅ Mono format: 44100 Hz, 1 channel")

	// Test 2: Create stereo format
	t.Log("📼 Test 2: Creating stereo format")
	stereoFormat, err := engine.NewStereoFormat(48000)
	if err != nil {
		t.Fatal("Failed to create stereo format:", err)
	}
	defer stereoFormat.Destroy()

	if stereoFormat.SampleRate() != 48000 {
		t.Errorf("Expected sample rate 48000, got %.0f", stereoFormat.SampleRate())
	}

	if stereoFormat.ChannelCount() != 2 {
		t.Errorf("Expected 2 channels, got %d", stereoFormat.ChannelCount())
	}
	t.Log("   ✅ Stereo format: 48000 Hz, 2 channels")

	// Test 3: Create format with channels (stereo with explicit interleaving)
	t.Log("📼 Test 3: Creating explicit stereo format with interleaving control")
	interleavedStereo, err := engine.NewFormatWithChannels(48000, 2, true) // Interleaved stereo
	if err != nil {
		t.Fatal("Failed to create interleaved stereo format:", err)
	}
	defer interleavedStereo.Destroy()

	if interleavedStereo.SampleRate() != 48000 {
		t.Errorf("Expected sample rate 48000, got %.0f", interleavedStereo.SampleRate())
	}

	if interleavedStereo.ChannelCount() != 2 {
		t.Errorf("Expected 2 channels, got %d", interleavedStereo.ChannelCount())
	}

	if !interleavedStereo.IsInterleaved() {
		t.Error("Expected interleaved format, got non-interleaved")
	}
	t.Log("   ✅ Interleaved stereo format: 48000 Hz, 2 channels, interleaved")

	// Test 4: Create format from enhanced spec (mono example)
	t.Log("📼 Test 4: Creating mono format from EnhancedAudioSpec")
	enhancedSpec := EnhancedAudioSpec{
		SampleRate:   22050,
		BufferSize:   1024,  // This will be used for ToSpec() but not format creation
		BitDepth:     16,    // This will be used for ToSpec() but not format creation
		ChannelCount: 1,     // Mono
		Interleaved:  false, // Doesn't matter for mono, but let's be explicit
	}

	specFormat, err := engine.NewFormat(enhancedSpec)
	if err != nil {
		t.Fatal("Failed to create format from spec:", err)
	}
	defer specFormat.Destroy()

	if specFormat.SampleRate() != 22050 {
		t.Errorf("Expected sample rate 22050, got %.0f", specFormat.SampleRate())
	}

	if specFormat.ChannelCount() != 1 {
		t.Errorf("Expected 1 channel, got %d", specFormat.ChannelCount())
	}
	t.Log("   ✅ Spec-based mono format: 22050 Hz, 1 channel")

	// Test 5: Format comparison
	t.Log("📼 Test 5: Testing format comparison")
	format1, err := engine.NewStereoFormat(44100)
	if err != nil {
		t.Fatal("Failed to create format1:", err)
	}
	defer format1.Destroy()

	format2, err := engine.NewStereoFormat(44100)
	if err != nil {
		t.Fatal("Failed to create format2:", err)
	}
	defer format2.Destroy()

	format3, err := engine.NewStereoFormat(48000)
	if err != nil {
		t.Fatal("Failed to create format3:", err)
	}
	defer format3.Destroy()

	if !format1.IsEqual(format2) {
		t.Error("Expected format1 to equal format2 (same specs)")
	}

	if format1.IsEqual(format3) {
		t.Error("Expected format1 to NOT equal format3 (different sample rates)")
	}
	t.Log("   ✅ Format comparison working correctly")

	// Test 6: ToSpec conversion
	t.Log("📼 Test 6: Testing ToSpec conversion")
	extractedSpec := stereoFormat.ToSpec()
	if extractedSpec.SampleRate != 48000 {
		t.Errorf("Expected extracted sample rate 48000, got %.0f", extractedSpec.SampleRate)
	}
	if extractedSpec.ChannelCount != 2 {
		t.Errorf("Expected extracted channel count 2, got %d", extractedSpec.ChannelCount)
	}
	t.Log("   ✅ ToSpec conversion working")

	// Test 7: Engine format
	t.Log("📼 Test 7: Testing engine-compatible format")
	engineFormat, err := engine.GetEngineFormat()
	if err != nil {
		t.Fatal("Failed to create engine format:", err)
	}
	defer engineFormat.Destroy()

	expectedSpec := engine.GetSpec()
	if engineFormat.SampleRate() != expectedSpec.SampleRate {
		t.Errorf("Expected engine format sample rate %.0f, got %.0f",
			expectedSpec.SampleRate, engineFormat.SampleRate())
	}
	if engineFormat.ChannelCount() != expectedSpec.ChannelCount {
		t.Errorf("Expected engine format channel count %d, got %d",
			expectedSpec.ChannelCount, engineFormat.ChannelCount())
	}
	t.Log("   ✅ Engine-compatible format created")

	// Test 8: GetPtr functionality
	t.Log("📼 Test 8: Testing GetPtr for unsafe.Pointer compatibility")
	ptr := monoFormat.GetPtr()
	if ptr == nil {
		t.Error("Expected non-nil unsafe.Pointer from GetPtr()")
	}
	t.Log("   ✅ GetPtr returns valid pointer for compatibility")

	t.Log("🎉 Format integration test complete!")
}

// TestCommonFormatShortcuts tests the convenience methods for common formats
func TestCommonFormatShortcuts(t *testing.T) {
	t.Log("🎯 Testing common format shortcuts")

	engine, err := New(DefaultAudioSpec())
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Test standard stereo (most common)
	t.Log("📼 Test 1: Standard stereo format")
	stereo, err := engine.NewStandardStereoFormat()
	if err != nil {
		t.Fatal("Failed to create standard stereo:", err)
	}
	defer stereo.Destroy()

	if stereo.SampleRate() != 48000 || stereo.ChannelCount() != 2 {
		t.Errorf("Expected 48kHz stereo, got %.0f Hz %d channels",
			stereo.SampleRate(), stereo.ChannelCount())
	}
	t.Log("   ✅ Standard stereo: 48kHz, 2 channels")

	// Test standard mono
	t.Log("📼 Test 2: Standard mono format")
	mono, err := engine.NewStandardMonoFormat()
	if err != nil {
		t.Fatal("Failed to create standard mono:", err)
	}
	defer mono.Destroy()

	if mono.SampleRate() != 48000 || mono.ChannelCount() != 1 {
		t.Errorf("Expected 48kHz mono, got %.0f Hz %d channels",
			mono.SampleRate(), mono.ChannelCount())
	}
	t.Log("   ✅ Standard mono: 48kHz, 1 channel")

	// Test CD audio format
	t.Log("📼 Test 3: CD audio format")
	cd, err := engine.NewCDAudioFormat()
	if err != nil {
		t.Fatal("Failed to create CD format:", err)
	}
	defer cd.Destroy()

	if cd.SampleRate() != 44100 || cd.ChannelCount() != 2 {
		t.Errorf("Expected 44.1kHz stereo, got %.0f Hz %d channels",
			cd.SampleRate(), cd.ChannelCount())
	}
	t.Log("   ✅ CD format: 44.1kHz, 2 channels")

	// Test interleaved stereo
	t.Log("📼 Test 4: Interleaved stereo format")
	interleaved, err := engine.NewInterleavedStereoFormat(48000)
	if err != nil {
		t.Fatal("Failed to create interleaved stereo:", err)
	}
	defer interleaved.Destroy()

	if !interleaved.IsInterleaved() {
		t.Error("Expected interleaved format")
	}
	if interleaved.ChannelCount() != 2 {
		t.Errorf("Expected 2 channels, got %d", interleaved.ChannelCount())
	}
	t.Log("   ✅ Interleaved stereo: 48kHz, 2 channels, interleaved")

	t.Log("🎯 Common format shortcuts test complete!")
}

// TestFormatWithConnections tests format usage with engine connections
func TestFormatWithConnections(t *testing.T) {
	t.Log("🔗 Testing format integration with engine connections")

	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	engine.Start()
	defer engine.Stop()

	// Create mixer nodes
	mixer1, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatal("Failed to create mixer1:", err)
	}

	mixer2, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatal("Failed to create mixer2:", err)
	}

	mainMixer, err := engine.MainMixerNode()
	if err != nil {
		t.Fatal("Failed to get main mixer:", err)
	}

	// Test ConnectWithSpec
	t.Log("🔗 Test 1: ConnectWithSpec")
	enhancedSpec := EnhancedAudioSpec{
		SampleRate:   48000,
		ChannelCount: 2,
		Interleaved:  false,
	}

	err = engine.ConnectWithSpec(mixer1, mixer2, 0, 0, enhancedSpec)
	if err != nil {
		t.Fatal("Failed to connect with spec:", err)
	}
	t.Log("   ✅ ConnectWithSpec successful")

	// Test ConnectWithTypedFormat
	t.Log("🔗 Test 2: ConnectWithTypedFormat")
	format, err := engine.NewStereoFormat(48000)
	if err != nil {
		t.Fatal("Failed to create format:", err)
	}
	defer format.Destroy()

	err = engine.ConnectWithTypedFormat(mixer2, mainMixer, 0, 0, format)
	if err != nil {
		t.Fatal("Failed to connect with typed format:", err)
	}
	t.Log("   ✅ ConnectWithTypedFormat successful")

	t.Log("🎉 Format connection test complete!")
}
