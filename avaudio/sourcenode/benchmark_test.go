package sourcenode

import (
	"testing"
)

const (
	// Audio buffer sizes for testing - realistic audio callback sizes
	SmallBuffer  = 256   // ~5.8ms at 44.1kHz
	NormalBuffer = 1024  // ~23.2ms at 44.1kHz  
	LargeBuffer  = 4096  // ~92.9ms at 44.1kHz
)

// ============================================================================
// OBJECTIVE-C BENCHMARKS
// ============================================================================

func BenchmarkSineGeneration_Small(b *testing.B) {
	benchmarkSineGeneration(b, SmallBuffer)
}

func BenchmarkSineGeneration_Normal(b *testing.B) {
	benchmarkSineGeneration(b, NormalBuffer)
}

func BenchmarkSineGeneration_Large(b *testing.B) {
	benchmarkSineGeneration(b, LargeBuffer)
}

func benchmarkSineGeneration(b *testing.B, frameCount int) {
	sourceNode, err := NewTone() // Objective-C generation
	if err != nil {
		b.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()
	
	sourceNode.SetFrequency(440.0)
	sourceNode.SetAmplitude(0.5)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		buffer := sourceNode.GenerateBuffer(frameCount)
		_ = buffer // Prevent optimization
	}
}

// ============================================================================
// END OBJECTIVE-C BENCHMARKS
// ============================================================================

// ============================================================================
// OVERHEAD BENCHMARKS
// ============================================================================

// Benchmark the CGO call overhead
func BenchmarkCGOCallOverhead(b *testing.B) {
	sourceNode, err := NewTone()
	if err != nil {
		b.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Just call the C function with minimal work
		sourceNode.SetFrequency(440.0)
	}
}

// Benchmark memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		buffer := make([]float32, NormalBuffer)
		_ = buffer
	}
}

// ============================================================================
// END OVERHEAD BENCHMARKS
// ============================================================================

// ============================================================================
// VALIDATION TESTS
// ============================================================================

// Test that tone generation produces expected output
func TestToneGeneration(t *testing.T) {
	toneNode, err := NewTone()
	if err != nil {
		t.Fatalf("Failed to create tone source node: %v", err)
	}
	defer toneNode.Destroy()
	
	// Set parameters
	freq, amp := 440.0, 0.5
	toneNode.SetFrequency(freq)
	toneNode.SetAmplitude(amp)
	
	// Generate buffer
	frameCount := 100
	buffer := toneNode.GenerateBuffer(frameCount)
	
	if len(buffer) != frameCount {
		t.Fatalf("Buffer length mismatch: expected=%d, got=%d", frameCount, len(buffer))
	}
	
	// Check that we're not generating silence
	hasNonZero := false
	for _, sample := range buffer {
		if sample != 0.0 {
			hasNonZero = true
			break
		}
	}
	
	if !hasNonZero {
		t.Error("Tone generation produced only silence")
	}
	
	// Check amplitude bounds
	for i, sample := range buffer {
		if sample < -1.0 || sample > 1.0 {
			t.Errorf("Sample %d out of bounds: %f", i, sample)
			break
		}
	}
}

// Test performance under realistic audio callback constraints
func BenchmarkRealtimeConstraints(b *testing.B) {
	// Simulate realistic audio callback: 1024 samples at 44.1kHz = ~23ms budget
	frameCount := 1024
	
	sourceNode, err := NewTone()
	if err != nil {
		b.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()
	
	sourceNode.SetFrequency(440.0)
	sourceNode.SetAmplitude(0.5)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		buffer := sourceNode.GenerateBuffer(frameCount)
		_ = buffer
	}
}

// ============================================================================
// END VALIDATION TESTS
// ============================================================================
