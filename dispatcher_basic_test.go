package macaudio

import (
	"testing"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
)

// TestDispatcherBasic tests basic dispatcher functionality
func TestDispatcherBasic(t *testing.T) {
	// Create test engine
	config := EngineConfig{
		AudioSpec: engine.AudioSpec{
			SampleRate:   48000,
			BufferSize:   256,
			BitDepth:     32,
			ChannelCount: 2,
		},
		OutputDeviceUID: "BuiltInSpeakerDevice",
		ErrorHandler:    &DefaultErrorHandler{},
	}

	testEngine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Destroy()

	dispatcher := testEngine.dispatcher

	// Test basic start/stop
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}

	if !dispatcher.IsRunning() {
		t.Error("Dispatcher should be running")
	}

	// Submit a simple operation with timeout
	operation := DispatcherOperation{
		Type: OpSetMute,
		Data: SetMuteData{
			ChannelID: "nonexistent-channel",
			Muted:     true,
		},
		Response: make(chan DispatcherResult, 1),
	}

	t.Log("Submitting test operation...")
	
	// Submit operation with timeout
	select {
	case dispatcher.operations <- operation:
		t.Log("Operation submitted successfully")
		
		// Wait for response with timeout
		select {
		case result := <-operation.Response:
			t.Logf("Operation completed with result: success=%t, error=%v", result.Success, result.Error)
		case <-time.After(1 * time.Second):
			t.Error("Operation timed out waiting for response")
		}
		
	case <-time.After(1 * time.Second):
		t.Error("Timed out submitting operation")
	}

	// Test performance stats
	lastDuration, maxDuration := dispatcher.GetPerformanceStats()
	t.Logf("Performance stats - Last: %v, Max: %v", lastDuration, maxDuration)

	// Clean shutdown
	if err := dispatcher.Stop(); err != nil {
		t.Errorf("Failed to stop dispatcher: %v", err)
	}

	t.Log("Basic dispatcher test completed")
}
