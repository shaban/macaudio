package tap

import (
	"testing"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
)

func TestTapBasicFunctionality(t *testing.T) {
	t.Log("Testing basic tap functionality...")

	// Create a real engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	// Create a mixer node to tap
	mixerPtr := node.CreateMixer()
	if mixerPtr == nil {
		t.Fatal("Failed to create mixer node")
	}
	defer node.ReleaseMixer(mixerPtr)

	// Attach the mixer to the engine
	err = eng.Attach(mixerPtr)
	if err != nil {
		t.Fatalf("Failed to attach mixer to engine: %v", err)
	}

	// Test installing a tap
	tap, err := InstallTap(eng.Ptr(), mixerPtr, 0)
	if err != nil {
		t.Fatalf("Failed to install tap: %v", err)
	}

	t.Log("✓ Successfully installed tap")

	// Test getting tap info
	info, err := tap.GetInfo()
	if err != nil {
		t.Fatalf("Failed to get tap info: %v", err)
	}

	t.Logf("✓ Got tap info - Sample Rate: %.2f Hz, Channels: %d", info.SampleRate, info.ChannelCount)

	// Test getting metrics
	metrics, err := tap.GetMetrics()
	if err != nil {
		t.Fatalf("Failed to get tap metrics: %v", err)
	}

	t.Logf("✓ Got metrics - RMS: %.2f, Frame Count: %d", metrics.RMS, metrics.FrameCount)

	// Test removing the tap
	err = tap.Remove()
	if err != nil {
		t.Fatalf("Failed to remove tap: %v", err)
	}

	t.Log("✓ Successfully removed tap")
}

func TestTapInstallErrors(t *testing.T) {
	t.Log("Testing tap error handling...")

	// Test with nil engine pointer
	_, err := InstallTap(nil, unsafe.Pointer(uintptr(0x123)), 0)
	if err == nil {
		t.Error("Expected error with nil engine pointer")
	}
	t.Log("✓ Correctly rejected nil engine pointer")

	// Test with nil node pointer
	_, err = InstallTap(unsafe.Pointer(uintptr(0x123)), nil, 0)
	if err == nil {
		t.Error("Expected error with nil node pointer")
	}
	t.Log("✓ Correctly rejected nil node pointer")

	// Test with negative bus index
	_, err = InstallTap(unsafe.Pointer(uintptr(0x123)), unsafe.Pointer(uintptr(0x456)), -1)
	if err == nil {
		t.Error("Expected error with negative bus index")
	}
	t.Log("✓ Correctly rejected negative bus index")
}

func TestTapCount(t *testing.T) {
	t.Log("Testing tap count functions...")

	// Get initial count
	initialCount := GetActiveTapCount()
	t.Logf("Initial active tap count: %d", initialCount)

	// Remove all taps to start clean
	RemoveAllTaps()

	// Verify count is reset
	count := GetActiveTapCount()
	t.Logf("✓ Tap count after RemoveAllTaps(): %d", count)
}
