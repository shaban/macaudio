package engine

import (
	"fmt"
	"testing"
)

func TestNewEngine(t *testing.T) {
	as := new(AudioSpec)
	as.BitDepth = 32
	as.BufferSize = 512
	as.ChannelCount = 2
	as.SampleRate = 96000
	engine, err := New(*as)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}
}

// Test basic engine lifecycle
func TestEngineLifecycle(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Initially should not be running
	if engine.IsRunning() {
		t.Error("New engine should not be running")
	}

	// Start the engine
	engine.Prepare()
	if err := engine.Start(); err != nil {
		t.Fatal("Failed to start engine:", err)
	}

	// Should be running now
	if !engine.IsRunning() {
		t.Error("Engine should be running after start")
	}

	// Pause and check
	engine.Pause()
	// Note: Paused engines might still report as "running" in AVFoundation

	// Stop the engine
	engine.Stop()
	if engine.IsRunning() {
		t.Error("Engine should not be running after stop")
	}

	// Reset should be safe to call
	engine.Reset()
}

// Test getting basic nodes from the engine
func TestEngineNodes(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Test output node
	outputNode, err := engine.OutputNode()
	if err != nil {
		t.Fatal("Failed to get output node:", err)
	}
	if outputNode == nil {
		t.Error("Output node should not be nil")
	}

	// Test input node
	inputNode, err := engine.InputNode()
	if err != nil {
		t.Fatal("Failed to get input node:", err)
	}
	if inputNode == nil {
		t.Error("Input node should not be nil")
	}

	// Test main mixer node
	mainMixer, err := engine.MainMixerNode()
	if err != nil {
		t.Fatal("Failed to get main mixer node:", err)
	}
	if mainMixer == nil {
		t.Error("Main mixer node should not be nil")
	}
}

// Test creating and managing mixer nodes
func TestMixerNodeCreation(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Create a new mixer node
	mixerNode, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatal("Failed to create mixer node:", err)
	}
	if mixerNode == nil {
		t.Error("Mixer node should not be nil")
	}

	// Test volume control on the mixer
	testVolume := float32(0.7)
	if err := engine.SetMixerVolume(mixerNode, testVolume); err != nil {
		t.Fatal("Failed to set mixer volume:", err)
	}

	// Get the volume back
	volume, err := engine.GetMixerVolume(mixerNode)
	if err != nil {
		t.Fatal("Failed to get mixer volume:", err)
	}

	// Volume should be approximately what we set (allowing for floating point precision)
	if volume < testVolume-0.01 || volume > testVolume+0.01 {
		t.Errorf("Expected volume ~%.2f, got %.2f", testVolume, volume)
	}
}

// Test node attachment and detachment
func TestNodeAttachDetach(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Create a mixer node to attach/detach
	mixerNode, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatal("Failed to create mixer node:", err)
	}

	// Detach it (it should already be attached from CreateMixerNode)
	if err := engine.Detach(mixerNode); err != nil {
		t.Fatal("Failed to detach mixer node:", err)
	}

	// Re-attach it
	if err := engine.Attach(mixerNode); err != nil {
		t.Fatal("Failed to attach mixer node:", err)
	}
}

// Test node connections
func TestNodeConnections(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Get main mixer and output node
	mainMixer, err := engine.MainMixerNode()
	if err != nil {
		t.Fatal("Failed to get main mixer:", err)
	}

	_, err = engine.OutputNode()
	if err != nil {
		t.Fatal("Failed to get output node:", err)
	}

	// Create a custom mixer
	customMixer, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatal("Failed to create custom mixer:", err)
	}

	// Test connection with automatic format
	if err := engine.Connect(customMixer, mainMixer, 0, 0); err != nil {
		t.Fatal("Failed to connect custom mixer to main mixer:", err)
	}

	// Test disconnection
	if err := engine.DisconnectNodeInput(mainMixer, 0); err != nil {
		t.Fatal("Failed to disconnect main mixer input:", err)
	}

	// Test connection with explicit format
	if err := engine.ConnectWithFormat(customMixer, mainMixer, 0, 0, nil); err != nil {
		t.Fatal("Failed to connect with format:", err)
	}
}

// Test mixer pan control
func TestMixerPan(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Test different pan values
	testPans := []float32{-1.0, -0.5, 0.0, 0.5, 1.0}

	for _, pan := range testPans {
		engine.SetMixerPan(pan)
		// Note: There's no GetMixerPan in the current API, so we just test that SetMixerPan doesn't crash
		t.Logf("Set mixer pan to %.1f", pan)
	}
}

// Test buffer size changes
func TestBufferSize(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Test various buffer sizes
	bufferSizes := []int{64, 128, 256, 512, 1024, 2048}

	for _, size := range bufferSizes {
		if err := engine.SetBufferSize(size); err != nil {
			t.Errorf("Failed to set buffer size to %d: %v", size, err)
		}

		// Check that the spec was updated
		if engine.GetSpec().BufferSize != size {
			t.Errorf("Expected buffer size %d, got %d", size, engine.GetSpec().BufferSize)
		}
	}

	// Test invalid buffer size
	if err := engine.SetBufferSize(-1); err == nil {
		t.Error("Setting negative buffer size should fail")
	}
}

// Test engine with different audio specs
func TestDifferentAudioSpecs(t *testing.T) {
	testSpecs := []AudioSpec{
		{SampleRate: 44100, BufferSize: 256, BitDepth: 16, ChannelCount: 1},  // Mono, CD quality
		{SampleRate: 48000, BufferSize: 512, BitDepth: 24, ChannelCount: 2},  // Stereo, professional
		{SampleRate: 96000, BufferSize: 1024, BitDepth: 32, ChannelCount: 2}, // High-res stereo
	}

	for i, spec := range testSpecs {
		t.Run(fmt.Sprintf("Spec%d", i+1), func(t *testing.T) {
			engine, err := New(spec)
			if err != nil {
				t.Fatal("Failed to create engine with spec:", err)
			}
			defer engine.Destroy()

			// Verify the spec was set
			gotSpec := engine.GetSpec()
			if gotSpec.SampleRate != spec.SampleRate {
				t.Errorf("Expected sample rate %.0f, got %.0f", spec.SampleRate, gotSpec.SampleRate)
			}
			if gotSpec.ChannelCount != spec.ChannelCount {
				t.Errorf("Expected %d channels, got %d", spec.ChannelCount, gotSpec.ChannelCount)
			}
		})
	}
}

// Test the native engine pointer
func TestNativeEnginePointer(t *testing.T) {
	spec := DefaultAudioSpec()
	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// Test getting the native engine pointer
	nativePtr := engine.GetNativeEngine()
	if nativePtr == nil {
		t.Error("Native engine pointer should not be nil")
	}

	// Test the Ptr() method too
	ptr := engine.Ptr()
	if ptr == nil {
		t.Error("Ptr() should not return nil")
	}

	// They should be the same
	if nativePtr != ptr {
		t.Error("GetNativeEngine() and Ptr() should return the same pointer")
	}
}

// Test comprehensive engine workflow
func TestEngineWorkflow(t *testing.T) {
	spec := AudioSpec{
		SampleRate:   48000,
		BufferSize:   512,
		BitDepth:     32,
		ChannelCount: 2,
	}

	engine, err := New(spec)
	if err != nil {
		t.Fatal("Failed to create engine:", err)
	}
	defer engine.Destroy()

	// 1. Get all the basic nodes
	_, err = engine.OutputNode()
	if err != nil {
		t.Fatal("Failed to get output node:", err)
	}

	mainMixer, err := engine.MainMixerNode()
	if err != nil {
		t.Fatal("Failed to get main mixer:", err)
	}

	_, err = engine.InputNode()
	if err != nil {
		t.Fatal("Failed to get input node:", err)
	}

	// 2. Create some custom mixers
	channelMixer1, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatal("Failed to create channel mixer 1:", err)
	}

	channelMixer2, err := engine.CreateMixerNode()
	if err != nil {
		t.Fatal("Failed to create channel mixer 2:", err)
	}

	// 3. Set up a routing chain: channelMixer1 -> channelMixer2 -> mainMixer -> output
	// Note: The main mixer is typically already connected to output, so we don't need to do that
	// Note: Input node requires special configuration, so we'll skip direct connection for this test

	// Connect first mixer to second mixer
	if err := engine.Connect(channelMixer1, channelMixer2, 0, 0); err != nil {
		t.Fatal("Failed to connect channel mixer 1 to 2:", err)
	}

	// Connect second mixer to main mixer
	if err := engine.Connect(channelMixer2, mainMixer, 0, 0); err != nil {
		t.Fatal("Failed to connect channel mixer 2 to main mixer:", err)
	}

	// 4. Set some volume levels
	if err := engine.SetMixerVolume(channelMixer1, 0.8); err != nil {
		t.Fatal("Failed to set channel mixer 1 volume:", err)
	}

	if err := engine.SetMixerVolume(channelMixer2, 0.6); err != nil {
		t.Fatal("Failed to set channel mixer 2 volume:", err)
	}

	// 5. Set pan on main mixer
	engine.SetMixerPan(0.2) // Slightly to the right

	// 6. Start the engine and verify it works
	engine.Prepare()
	if err := engine.Start(); err != nil {
		t.Fatal("Failed to start configured engine:", err)
	}

	if !engine.IsRunning() {
		t.Error("Engine should be running after start")
	}

	// 7. Test volume retrieval
	vol1, err := engine.GetMixerVolume(channelMixer1)
	if err != nil {
		t.Fatal("Failed to get channel mixer 1 volume:", err)
	}
	t.Logf("Channel mixer 1 volume: %.2f", vol1)

	vol2, err := engine.GetMixerVolume(channelMixer2)
	if err != nil {
		t.Fatal("Failed to get channel mixer 2 volume:", err)
	}
	t.Logf("Channel mixer 2 volume: %.2f", vol2)

	// 8. Test runtime buffer size change
	if err := engine.SetBufferSize(1024); err != nil {
		t.Fatal("Failed to change buffer size at runtime:", err)
	}

	// 9. Clean shutdown
	engine.Stop()
	if engine.IsRunning() {
		t.Error("Engine should not be running after stop")
	}

	t.Log("Complete engine workflow test passed!")
}
