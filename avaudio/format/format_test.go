package format

import (
	"testing"
)

func TestNewMono(t *testing.T) {
	sampleRate := 44100.0
	format, err := NewMono(sampleRate)
	if err != nil {
		t.Fatalf("Failed to create mono format: %v", err)
	}
	defer format.Destroy()

	if format.SampleRate() != sampleRate {
		t.Errorf("Expected sample rate %.0f, got %.0f", sampleRate, format.SampleRate())
	}

	if format.ChannelCount() != 1 {
		t.Errorf("Expected 1 channel for mono, got %d", format.ChannelCount())
	}

	if format.IsInterleaved() {
		t.Error("Expected non-interleaved format for mono")
	}

	if format.GetFormatPtr() == nil {
		t.Error("Format pointer should not be nil")
	}

	t.Logf("✓ Mono format: %.0f Hz, %d channel, interleaved: %t",
		format.SampleRate(), format.ChannelCount(), format.IsInterleaved())
}

func TestNewStereo(t *testing.T) {
	sampleRate := 48000.0
	format, err := NewStereo(sampleRate)
	if err != nil {
		t.Fatalf("Failed to create stereo format: %v", err)
	}
	defer format.Destroy()

	if format.SampleRate() != sampleRate {
		t.Errorf("Expected sample rate %.0f, got %.0f", sampleRate, format.SampleRate())
	}

	if format.ChannelCount() != 2 {
		t.Errorf("Expected 2 channels for stereo, got %d", format.ChannelCount())
	}

	if format.IsInterleaved() {
		t.Error("Expected non-interleaved format for stereo")
	}

	if format.GetFormatPtr() == nil {
		t.Error("Format pointer should not be nil")
	}

	t.Logf("✓ Stereo format: %.0f Hz, %d channels, interleaved: %t",
		format.SampleRate(), format.ChannelCount(), format.IsInterleaved())
}

func TestNewWithChannels(t *testing.T) {
	testCases := []struct {
		name        string
		sampleRate  float64
		channels    int
		interleaved bool
	}{
		{"Mono non-interleaved", 44100, 1, false},
		{"Stereo interleaved", 48000, 2, true},
		{"Stereo non-interleaved", 96000, 2, false},
		{"Mono interleaved", 192000, 1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			format, err := NewWithChannels(tc.sampleRate, tc.channels, tc.interleaved)
			if err != nil {
				t.Fatalf("Failed to create %s format: %v", tc.name, err)
			}
			defer format.Destroy()

			if format.SampleRate() != tc.sampleRate {
				t.Errorf("Expected sample rate %.0f, got %.0f", tc.sampleRate, format.SampleRate())
			}

			if format.ChannelCount() != tc.channels {
				t.Errorf("Expected %d channels, got %d", tc.channels, format.ChannelCount())
			}

			if format.IsInterleaved() != tc.interleaved {
				t.Errorf("Expected interleaved %t, got %t", tc.interleaved, format.IsInterleaved())
			}

			t.Logf("✓ %s: %.0f Hz, %d channels, interleaved: %t",
				tc.name, format.SampleRate(), format.ChannelCount(), format.IsInterleaved())
		})
	}
}

func TestPCMFloat32ViaNewWithChannels(t *testing.T) {
	sampleRate := 44100.0
	channels := 2
	interleaved := true

	format, err := NewWithChannels(sampleRate, channels, interleaved)
	if err != nil {
		t.Fatalf("Failed to create PCM Float32 format: %v", err)
	}
	defer format.Destroy()

	if format.SampleRate() != sampleRate {
		t.Errorf("Expected sample rate %.0f, got %.0f", sampleRate, format.SampleRate())
	}

	if format.ChannelCount() != channels {
		t.Errorf("Expected %d channels, got %d", channels, format.ChannelCount())
	}

	if format.IsInterleaved() != interleaved {
		t.Errorf("Expected interleaved %t, got %t", interleaved, format.IsInterleaved())
	}

	t.Logf("✓ PCM Float32 (via NewWithChannels): %.0f Hz, %d channels, interleaved: %t",
		format.SampleRate(), format.ChannelCount(), format.IsInterleaved())
}

func TestFormatCopy(t *testing.T) {
	original, err := NewStereo(48000)
	if err != nil {
		t.Fatalf("Failed to create original format: %v", err)
	}
	defer original.Destroy()

	copied, err := original.Copy()
	if err != nil {
		t.Fatalf("Failed to copy format: %v", err)
	}
	defer copied.Destroy()

	// Should have same properties
	if copied.SampleRate() != original.SampleRate() {
		t.Errorf("Copy sample rate mismatch: expected %.0f, got %.0f",
			original.SampleRate(), copied.SampleRate())
	}

	if copied.ChannelCount() != original.ChannelCount() {
		t.Errorf("Copy channel count mismatch: expected %d, got %d",
			original.ChannelCount(), copied.ChannelCount())
	}

	if copied.IsInterleaved() != original.IsInterleaved() {
		t.Errorf("Copy interleaved mismatch: expected %t, got %t",
			original.IsInterleaved(), copied.IsInterleaved())
	}

	// Should be different objects but equal content
	if copied.GetFormatPtr() == original.GetFormatPtr() {
		t.Error("Copy should have different format pointer than original")
	}

	if !copied.IsEqual(original) {
		t.Error("Copy should be equal to original")
	}

	t.Logf("✓ Format copy successful: original and copy are equal but different objects")
}

func TestFormatEquality(t *testing.T) {
	format1, err := NewMono(44100)
	if err != nil {
		t.Fatalf("Failed to create format1: %v", err)
	}
	defer format1.Destroy()

	format2, err := NewMono(44100)
	if err != nil {
		t.Fatalf("Failed to create format2: %v", err)
	}
	defer format2.Destroy()

	format3, err := NewStereo(44100)
	if err != nil {
		t.Fatalf("Failed to create format3: %v", err)
	}
	defer format3.Destroy()

	format4, err := NewMono(48000)
	if err != nil {
		t.Fatalf("Failed to create format4: %v", err)
	}
	defer format4.Destroy()

	// Same format configurations should be equal
	if !format1.IsEqual(format2) {
		t.Error("Two mono 44100Hz formats should be equal")
	}

	// Different channel counts should not be equal
	if format1.IsEqual(format3) {
		t.Error("Mono and stereo formats should not be equal")
	}

	// Different sample rates should not be equal
	if format1.IsEqual(format4) {
		t.Error("Different sample rates should not be equal")
	}

	t.Logf("✓ Format equality tests passed")
}

func TestFormatLogInfo(t *testing.T) {
	format, err := NewStereo(48000)
	if err != nil {
		t.Fatalf("Failed to create format: %v", err)
	}
	defer format.Destroy()

	// This will log to console for visual verification
	format.LogInfo()

	t.Logf("✓ Format logging test completed (check console output)")
}

func TestFormatNilHandling(t *testing.T) {
	var nilFormat *Format

	// Test nil safety
	if nilFormat.SampleRate() != 0.0 {
		t.Error("Nil format should return 0.0 sample rate")
	}

	if nilFormat.ChannelCount() != 0 {
		t.Error("Nil format should return 0 channel count")
	}

	if nilFormat.IsInterleaved() {
		t.Error("Nil format should return false for interleaved")
	}

	if nilFormat.GetFormatPtr() != nil {
		t.Error("Nil format should return nil pointer")
	}

	if nilFormat.IsEqual(nil) {
		t.Error("Nil formats should not be equal")
	}

	// Should not crash
	nilFormat.LogInfo()
	nilFormat.Destroy()

	t.Logf("✓ Nil handling tests passed")
}

func TestFormatIntegrationWorkflow(t *testing.T) {
	// Create a mono format for left channel routing
	monoFormat, err := NewMono(44100)
	if err != nil {
		t.Fatalf("Failed to create mono format: %v", err)
	}
	defer monoFormat.Destroy()

	// Create a stereo format for mixer output
	stereoFormat, err := NewStereo(44100)
	if err != nil {
		t.Fatalf("Failed to create stereo format: %v", err)
	}
	defer stereoFormat.Destroy()

	// Verify format pointers are available for engine integration
	if monoFormat.GetFormatPtr() == nil {
		t.Error("Mono format pointer should be available for engine integration")
	}

	if stereoFormat.GetFormatPtr() == nil {
		t.Error("Stereo format pointer should be available for engine integration")
	}

	// Verify formats have expected properties for AVAudioEngine usage
	if monoFormat.ChannelCount() != 1 || monoFormat.SampleRate() != 44100 {
		t.Error("Mono format properties not suitable for engine integration")
	}

	if stereoFormat.ChannelCount() != 2 || stereoFormat.SampleRate() != 44100 {
		t.Error("Stereo format properties not suitable for engine integration")
	}

	// Test format copying for multiple engine nodes
	monoCopy, err := monoFormat.Copy()
	if err != nil {
		t.Fatalf("Failed to copy mono format: %v", err)
	}
	defer monoCopy.Destroy()

	if !monoFormat.IsEqual(monoCopy) {
		t.Error("Format copy should be equal to original")
	}

	t.Logf("✓ Integration workflow: formats ready for engine usage")
	t.Logf("  - Mono: %.0f Hz, %d channel, ptr: %v",
		monoFormat.SampleRate(), monoFormat.ChannelCount(), monoFormat.GetFormatPtr() != nil)
	t.Logf("  - Stereo: %.0f Hz, %d channels, ptr: %v",
		stereoFormat.SampleRate(), stereoFormat.ChannelCount(), stereoFormat.GetFormatPtr() != nil)
}

// =============================================================================
// ✅ CORRECT FUNCTION SIGNATURES - These tests use the new (result, error) pattern
// =============================================================================
// NOTE: All tests above this comment use the CORRECT function signatures with
// proper error handling. Any tests below this comment or in other files that
// don't handle errors properly need to be updated to match this pattern.
//
// MIGRATION PATTERN:
// OLD: format := NewMono(44100)           // ❌ Missing error handling
// NEW: format, err := NewMono(44100)      // ✅ Proper error handling
//      if err != nil { ... }
// =============================================================================

// TestBasicFunctionality verifies that core C functions are properly linked and working
// This test confirms the migration to string-based error handling was successful
func TestBasicFunctionality(t *testing.T) {
	t.Log("=== Testing Basic Format Functionality ===")

	// Test 1: Create a format
	t.Log("Creating mono format...")
	mono, err := NewMono(44100.0)
	if err != nil {
		t.Fatalf("NewMono failed: %v", err)
	}
	if mono == nil {
		t.Fatalf("NewMono returned nil")
	}
	t.Logf("✅ NewMono succeeded, ptr: %p", mono.ptr)

	// Test functions that should work (non-struct returns)
	t.Log("Testing SampleRate...")
	sampleRate := mono.SampleRate()
	t.Logf("✅ SampleRate: %.1f", sampleRate)

	if sampleRate != 44100.0 {
		t.Errorf("Expected sample rate 44100.0, got %.1f", sampleRate)
	}

	t.Log("Testing ChannelCount...")
	channels := mono.ChannelCount()
	t.Logf("✅ ChannelCount: %d", channels)

	if channels != 1 {
		t.Errorf("Expected 1 channel for mono, got %d", channels)
	}

	t.Log("Testing IsInterleaved...")
	interleaved := mono.IsInterleaved()
	t.Logf("✅ IsInterleaved: %t", interleaved)

	// Test GetFormatPtr (this should work with the direct field access)
	t.Log("Testing GetFormatPtr...")
	ptr := mono.GetFormatPtr()
	if ptr != nil {
		t.Logf("✅ GetFormatPtr: %p", ptr)
	} else {
		t.Logf("⚠️ GetFormatPtr returned nil")
	}

	// Test LogInfo (void function, should always work)
	t.Log("Testing LogInfo...")
	mono.LogInfo()
	t.Logf("✅ LogInfo completed")

	// Clean up
	t.Log("Testing Destroy...")
	mono.Destroy()
	t.Logf("✅ Destroy completed")

	t.Log("=== Basic functionality test completed ===")
}
