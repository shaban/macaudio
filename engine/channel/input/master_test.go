package input

import (
	"fmt"
	"testing"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/sourcenode"
	"github.com/shaban/macaudio/engine/analyze"
	"github.com/shaban/macaudio/internal/testutil"
)

func TestMonoToStereoMasterConnection(t *testing.T) {
	t.Log("Testing MonoToStereoChannel master connection/disconnection...")

	eng, err := engine.New(testutil.SmallSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	// Create a mono-to-stereo channel
	config := MonoToStereoConfig{
		Name:       "Master Connection Test",
		Engine:     eng,
		InitialPan: 0.0,
	}

	monoChannel, err := NewMonoToStereo(config)
	if err != nil {
		t.Fatalf("Failed to create mono channel: %v", err)
	}
	defer monoChannel.Release()

	// Initially not connected to master
	if monoChannel.IsConnectedToMaster() {
		t.Error("Channel should not be connected to master initially")
	}

	t.Log("✓ Channel initially not connected to master")

	// Attach the channel to the engine before connecting
	err = eng.Attach(monoChannel.GetOutputNode())
	if err != nil {
		t.Fatalf("Failed to attach channel to engine: %v", err)
	}

	// Test ConnectToMaster
	err = monoChannel.ConnectToMaster(eng)
	if err != nil {
		t.Fatalf("Failed to connect channel to master: %v", err)
	}

	// Should now be connected
	if !monoChannel.IsConnectedToMaster() {
		t.Error("Channel should be connected to master after ConnectToMaster()")
	}

	t.Log("✓ Successfully connected channel to master")

	// Test that connecting again is idempotent (no error)
	err = monoChannel.ConnectToMaster(eng)
	if err != nil {
		t.Errorf("Idempotent connect should not error: %v", err)
	} else {
		t.Log("✓ Duplicate connect is idempotent (no-op)")
	}

	// Test DisconnectFromMaster
	err = monoChannel.DisconnectFromMaster(eng)
	if err != nil {
		t.Fatalf("Failed to disconnect channel from master: %v", err)
	}

	// Should no longer be connected
	if monoChannel.IsConnectedToMaster() {
		t.Error("Channel should not be connected to master after DisconnectFromMaster()")
	}

	t.Log("✓ Successfully disconnected channel from master")

	// Test that disconnecting again is idempotent (no error)
	err = monoChannel.DisconnectFromMaster(eng)
	if err != nil {
		t.Errorf("Idempotent disconnect should not error: %v", err)
	} else {
		t.Log("✓ Duplicate disconnect is idempotent (no-op)")
	}
}

func TestMasterConnectionWithRealAudio(t *testing.T) {
	t.Log("Testing master connection with real audio signal...")

	eng, err := engine.New(testutil.SmallSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	// Create channel and tone generator
	config := MonoToStereoConfig{
		Name:       "Real Audio Master Test",
		Engine:     eng,
		InitialPan: 0.0,
	}

	monoChannel, err := NewMonoToStereo(config)
	if err != nil {
		t.Fatalf("Failed to create mono channel: %v", err)
	}
	defer monoChannel.Release()

	// Create tone generator
	toneNode, err := sourcenode.NewTone()
	if err != nil {
		t.Fatalf("Failed to create tone node: %v", err)
	}
	defer toneNode.Destroy()

	toneNodePtr, err := toneNode.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get tone node pointer: %v", err)
	}
	if toneNodePtr == nil {
		t.Fatal("Tone node pointer is nil")
	}

	// Attach nodes to engine
	err = eng.Attach(toneNodePtr)
	if err != nil {
		t.Fatalf("Failed to attach tone node: %v", err)
	}

	err = eng.Attach(monoChannel.GetOutputNode())
	if err != nil {
		t.Fatalf("Failed to attach channel: %v", err)
	}

	// Connect tone generator to channel input
	err = eng.Connect(toneNodePtr, monoChannel.GetInputNode(), 0, 0)
	if err != nil {
		t.Fatalf("Failed to connect source to channel: %v", err)
	}

	// Connect channel to master using the new method
	err = monoChannel.ConnectToMaster(eng)
	if err != nil {
		t.Fatalf("Failed to connect channel to master: %v", err)
	}

	t.Log("✓ Connected channel to master output")

	// Start engine and test audio processing (mute main mixer to avoid audible output)
	testutil.MuteMainMixer(t, eng)
	err = eng.Start()
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer func() {
		if eng.IsRunning() {
			eng.Stop()
			t.Log("✓ Engine stopped")
		}
	}()

	// Quick tap-based signal check
	mm, _ := eng.MainMixerNode()
	testutil.AssertRMSAbove(t, eng, mm, 0, 0.0005, 200*time.Millisecond)

	// Verify audio is flowing through the master connection
	analysisConfig := analyze.DefaultAnalysisConfig()
	analysisConfig.SampleDuration = 100 * time.Millisecond

	stereoAnalysis, err := analyze.AnalyzeMonoToStereo(
		eng.Ptr(),
		toneNodePtr,
		monoChannel.GetOutputNode(),
		0.0, // Center pan
		analysisConfig,
	)
	if err != nil {
		t.Fatalf("Failed to analyze audio: %v", err)
	}

	if stereoAnalysis.TotalRMS > 0.001 {
		t.Logf("✓ Real audio detected through master connection (RMS: %.6f)", stereoAnalysis.TotalRMS)
	} else {
		t.Logf("⚠ Low audio signal (RMS: %.6f) - may need audio hardware", stereoAnalysis.TotalRMS)
	}

	// Test disconnection while engine is running
	t.Log("Testing disconnection while engine is running...")

	err = monoChannel.DisconnectFromMaster(eng)
	if err != nil {
		t.Fatalf("Failed to disconnect from master while running: %v", err)
	}

	t.Log("✓ Successfully disconnected from master while engine running")

	// Reconnect to test dynamic routing
	err = monoChannel.ConnectToMaster(eng)
	if err != nil {
		t.Fatalf("Failed to reconnect to master: %v", err)
	}

	t.Log("✓ Successfully reconnected to master (dynamic routing)")

	// This demonstrates the use case: user can dynamically route channels
	// for performance optimization without stopping the engine
}

func TestMultipleChannelMasterConnections(t *testing.T) {
	t.Log("Testing multiple channels connecting to master...")

	eng, err := engine.New(testutil.SmallSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	// Create multiple channels
	channels := make([]*MonoToStereoChannel, 3)
	for i := 0; i < 3; i++ {
		config := MonoToStereoConfig{
			Name:       fmt.Sprintf("Multi Channel %d", i+1),
			Engine:     eng,
			InitialPan: float32(i - 1), // -1.0, 0.0, 1.0 pan positions
		}

		channel, err := NewMonoToStereo(config)
		if err != nil {
			t.Fatalf("Failed to create channel %d: %v", i, err)
		}
		defer channel.Release()

		channels[i] = channel

		// Attach each channel to the engine
		err = eng.Attach(channel.GetOutputNode())
		if err != nil {
			t.Fatalf("Failed to attach channel %d to engine: %v", i, err)
		}
	}

	// Connect all channels to master
	for i, channel := range channels {
		err = channel.ConnectToMaster(eng)
		if err != nil {
			t.Fatalf("Failed to connect channel %d to master: %v", i, err)
		}

		if !channel.IsConnectedToMaster() {
			t.Errorf("Channel %d not properly connected to master", i)
		}
	}

	t.Log("✓ All channels connected to master")

	// Selectively disconnect channels (use case: user optimizing mix)
	err = channels[1].DisconnectFromMaster(eng) // Disconnect middle channel
	if err != nil {
		t.Fatalf("Failed to disconnect channel 1: %v", err)
	}

	// Verify states
	if !channels[0].IsConnectedToMaster() {
		t.Error("Channel 0 should still be connected")
	}
	if channels[1].IsConnectedToMaster() {
		t.Error("Channel 1 should be disconnected")
	}
	if !channels[2].IsConnectedToMaster() {
		t.Error("Channel 2 should still be connected")
	}

	t.Log("✓ Selective disconnection works correctly")

	// Reconnect for full mix
	err = channels[1].ConnectToMaster(eng)
	if err != nil {
		t.Fatalf("Failed to reconnect channel 1: %v", err)
	}

	// All should be connected again
	for i, channel := range channels {
		if !channel.IsConnectedToMaster() {
			t.Errorf("Channel %d should be reconnected", i)
		}
	}

	t.Log("✓ Full mix restored - demonstrates flexible routing for performance optimization")
}
