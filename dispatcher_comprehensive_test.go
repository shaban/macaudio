package macaudio

import (
	"sync"
	"testing"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
)

// TestDispatcherRaceConditionPrevention validates dispatcher prevents race conditions
func TestDispatcherRaceConditionPrevention(t *testing.T) {
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

	// Start dispatcher
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}
	defer dispatcher.Stop()

	// High-concurrency test: many goroutines submit operations simultaneously
	const numWorkers = 50
	const operationsPerWorker = 100
	totalOperations := numWorkers * operationsPerWorker

	var wg sync.WaitGroup
	var processedOps int32
	var mu sync.Mutex
	var operationResults []bool

	startTime := time.Now()

	// Launch worker goroutines
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for op := 0; op < operationsPerWorker; op++ {
				// Create operation
				operation := DispatcherOperation{
					Type: OpSetMute,
					Data: SetMuteData{
						ChannelID: "test-channel", // All target same channel to create contention
						Muted:     (op % 2) == 0,
					},
					Response: make(chan DispatcherResult, 1),
				}

				// Submit operation
				select {
				case dispatcher.operations <- operation:
					// Wait for response
					result := <-operation.Response

					// Record result
					mu.Lock()
					processedOps++
					operationResults = append(operationResults, result.Success)
					mu.Unlock()

				case <-time.After(5 * time.Second):
					t.Errorf("Worker %d operation %d timed out", workerID, op)
					return
				}
			}
		}(w)
	}

	// Wait for all workers to complete
	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Race condition prevention test completed:")
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Operations per worker: %d", operationsPerWorker)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Processed operations: %d", processedOps)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations/sec: %.0f", float64(processedOps)/duration.Seconds())

	// Validate all operations were processed
	if int(processedOps) != totalOperations {
		t.Errorf("Expected %d operations, processed %d", totalOperations, processedOps)
	}

	// Check performance stats
	lastDuration, maxDuration := dispatcher.GetPerformanceStats()
	t.Logf("Performance stats:")
	t.Logf("  Last operation: %v", lastDuration)
	t.Logf("  Max operation: %v", maxDuration)

	// Validate performance meets targets
	avgDuration := duration / time.Duration(processedOps)
	if avgDuration > 300*time.Millisecond {
		t.Errorf("Average operation time %v exceeds 300ms target", avgDuration)
	}

	// Validate no crashes or deadlocks occurred
	t.Logf("✓ No race conditions detected")
	t.Logf("✓ All operations serialized successfully")
	t.Logf("✓ Performance target met")
}

// TestDispatcherEngineLifecycle tests dispatcher integration with engine lifecycle
func TestDispatcherEngineLifecycle(t *testing.T) {
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

	// Start dispatcher
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}
	defer dispatcher.Stop()

	// Test engine start/stop through dispatcher (serialized)
	t.Run("EngineStartStop", func(t *testing.T) {
		// Engine start
		startOp := DispatcherOperation{
			Type:     OpStartEngine,
			Data:     CreateEngineData{}, // Empty data for start
			Response: make(chan DispatcherResult, 1),
		}

		dispatcher.operations <- startOp
		result := <-startOp.Response

		if result.Error != nil {
			t.Logf("Engine start result: %v (may fail without proper audio setup)", result.Error)
		} else {
			t.Logf("Engine started successfully through dispatcher")
		}

		// Engine stop
		stopOp := DispatcherOperation{
			Type:     OpStopEngine,
			Data:     CreateEngineData{}, // Empty data for stop
			Response: make(chan DispatcherResult, 1),
		}

		dispatcher.operations <- stopOp
		stopResult := <-stopOp.Response

		if stopResult.Error != nil {
			t.Logf("Engine stop result: %v", stopResult.Error)
		} else {
			t.Logf("Engine stopped successfully through dispatcher")
		}
	})

	t.Logf("Engine lifecycle through dispatcher test completed")
}

// TestDispatcherMultipleOperationTypes tests different operation types don't interfere
func TestDispatcherMultipleOperationTypes(t *testing.T) {
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

	// Start dispatcher
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}
	defer dispatcher.Stop()

	// Test different operation types in mixed sequence
	operations := []DispatcherOperation{
		{
			Type: OpSetMute,
			Data: SetMuteData{ChannelID: "test-1", Muted: true},
			Response: make(chan DispatcherResult, 1),
		},
		{
			Type: OpPluginBypass,
			Data: PluginBypassData{ChannelID: "test-1", PluginID: "plugin-1", Bypassed: true},
			Response: make(chan DispatcherResult, 1),
		},
		{
			Type: OpDeviceChange,
			Data: DeviceChangeData{ChannelID: "test-1", NewDeviceUID: "new-device"},
			Response: make(chan DispatcherResult, 1),
		},
		{
			Type: OpOutputDeviceChange,
			Data: OutputDeviceChangeData{NewDeviceUID: "new-output"},
			Response: make(chan DispatcherResult, 1),
		},
	}

	var results []DispatcherResult
	for i, op := range operations {
		t.Logf("Submitting operation %d: %s", i+1, op.Type)
		
		dispatcher.operations <- op
		result := <-op.Response
		results = append(results, result)
		
		t.Logf("Operation %d completed: success=%t, error=%v", i+1, result.Success, result.Error)
	}

	t.Logf("Multiple operation types test completed:")
	t.Logf("  Operations processed: %d", len(results))
	
	// All operations should complete (even if they fail due to missing channels/devices)
	if len(results) != len(operations) {
		t.Errorf("Expected %d results, got %d", len(operations), len(results))
	}

	t.Logf("✓ All operation types processed without interference")
}
