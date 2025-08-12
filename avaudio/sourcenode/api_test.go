package sourcenode

import (
	"testing"
)

// Test the new simplified API
func TestNewTone(t *testing.T) {
	// Test tone generation
	toneNode, err := NewTone()
	if err != nil {
		t.Fatalf("Failed to create tone node: %v", err)
	}
	defer toneNode.Destroy()
	
	// Test parameter setting
	toneNode.SetFrequency(880.0) // A5
	toneNode.SetAmplitude(0.3)
	
	// Generate a small buffer
	buffer := toneNode.GenerateBuffer(100)
	if len(buffer) != 100 {
		t.Errorf("Expected buffer length 100, got %d", len(buffer))
	}
	
	// Should not be all zeros
	hasSound := false
	for _, sample := range buffer {
		if sample != 0.0 {
			hasSound = true
			break
		}
	}
	
	if !hasSound {
		t.Error("NewTone() should generate audio, but got silence")
	}
}

// Test silent vs tone nodes
func TestSilentVsTone(t *testing.T) {
	silentNode, err := NewSilent()
	if err != nil {
		t.Fatalf("Failed to create silent node: %v", err)
	}
	defer silentNode.Destroy()
	
	toneNode, err := NewTone()
	if err != nil {
		t.Fatalf("Failed to create tone node: %v", err)
	}
	defer toneNode.Destroy()
	
	// Both should be valid for integration
	if silentNode.GetNodePtr() == nil {
		t.Error("Silent node should have valid pointer")
	}
	
	if toneNode.GetNodePtr() == nil {
		t.Error("Tone node should have valid pointer")
	}
	
	// They should be different for actual audio generation
	silentBuffer := silentNode.GenerateBuffer(100)
	toneBuffer := toneNode.GenerateBuffer(100)
	
	// Silent buffer should be all zeros when generated manually
	// (Note: the audio callback will still produce silence for silent nodes)
	for i, sample := range silentBuffer {
		if sample != 0.0 {
			t.Errorf("Silent buffer should be zero at index %d, got %f", i, sample)
			break
		}
	}
	
	// Tone buffer should have some audio content
	hasAudio := false
	for _, sample := range toneBuffer {
		if sample != 0.0 {
			hasAudio = true
			break
		}
	}
	
	if !hasAudio {
		t.Error("Tone buffer should contain audio samples")
	}
}
