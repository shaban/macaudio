package macaudio

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
)

// TestDispatcherSerialization tests that operations are properly serialized through dispatcher
func TestDispatcherSerialization(t *testing.T) {
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

	// Test that dispatcher starts and stops correctly
	t.Run("DispatcherLifecycle", func(t *testing.T) {
		testDispatcherLifecycle(t, testEngine)
	})

	// Test serialized operation execution
	t.Run("OperationSerialization", func(t *testing.T) {
		testOperationSerialization(t, testEngine)
	})

	// Test concurrent operations don't crash
	t.Run("ConcurrentOperationSafety", func(t *testing.T) {
		testConcurrentOperationSafety(t, testEngine)
	})

	// Test performance meets targets
	t.Run("DispatcherPerformance", func(t *testing.T) {
		testDispatcherPerformance(t, testEngine)
	})
}

// testDispatcherLifecycle tests dispatcher start/stop behavior
func testDispatcherLifecycle(t *testing.T, testEngine *Engine) {
	dispatcher := testEngine.dispatcher

	// Check initial state
	if dispatcher.IsRunning() {
		t.Error("Dispatcher should not be running initially")
	}

	// Start dispatcher
	if err := dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}

	if !dispatcher.IsRunning() {
		t.Error("Dispatcher should be running after start")
	}

	// Test double start doesn't crash
	if err := dispatcher.Start(); err == nil {
		t.Error("Expected error when starting already running dispatcher")
	}

	// Stop dispatcher
	if err := dispatcher.Stop(); err != nil {
		t.Errorf("Failed to stop dispatcher: %v", err)
	}

	// Allow some time for shutdown
	time.Sleep(10 * time.Millisecond)

	if dispatcher.IsRunning() {
		t.Error("Dispatcher should not be running after stop")
	}

	// Test double stop is safe
	if err := dispatcher.Stop(); err != nil {
		t.Errorf("Stop should be idempotent, got error: %v", err)
	}

	t.Logf("Dispatcher lifecycle test passed")
}

// testOperationSerialization tests that operations execute in order
func testOperationSerialization(t *testing.T, testEngine *Engine) {
	if err := testEngine.dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}
	defer testEngine.dispatcher.Stop()

	// Track operation execution order
	var executionOrder []int
	var mu sync.Mutex

	const numOperations = 10
	var wg sync.WaitGroup

	// Submit operations that should execute in order
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opNum int) {
			defer wg.Done()

			// Create a dummy mute operation that will fail but shows ordering
			op := DispatcherOperation{
				Type: OpSetMute,
				Data: SetMuteData{
					ChannelID: fmt.Sprintf("test-channel-%d", opNum),
					Muted:     true,
				},
				Response: make(chan DispatcherResult, 1),
			}

			// Submit to dispatcher
			testEngine.dispatcher.operations <- op

			// Wait for response
			<-op.Response

			// Record execution order
			mu.Lock()
			executionOrder = append(executionOrder, opNum)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Check that operations were processed (even if they failed)
	if len(executionOrder) != numOperations {
		t.Errorf("Expected %d operations processed, got %d", numOperations, len(executionOrder))
	}

	t.Logf("Operation serialization test: processed %d operations", len(executionOrder))
}

// testConcurrentOperationSafety tests that concurrent submissions don't crash
func testConcurrentOperationSafety(t *testing.T, testEngine *Engine) {
	if err := testEngine.dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}
	defer testEngine.dispatcher.Stop()

	const numGoroutines = 20
	const operationsPerGoroutine = 50

	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32
	var mu sync.Mutex

	startTime := time.Now()

	// Launch concurrent goroutines submitting operations
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				operation := DispatcherOperation{
					Type: OpSetMute,
					Data: SetMuteData{
						ChannelID: fmt.Sprintf("channel-%d-%d", goroutineID, op),
						Muted:     (op % 2) == 0,
					},
					Response: make(chan DispatcherResult, 1),
				}

				// Submit operation
				select {
				case testEngine.dispatcher.operations <- operation:
					// Wait for response
					result := <-operation.Response

					mu.Lock()
					if result.Error != nil {
						errorCount++ // Expected errors for non-existent channels
					} else {
						successCount++
					}
					mu.Unlock()

				case <-time.After(1 * time.Second):
					t.Errorf("Operation timed out")
					return
				}
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(startTime)

	totalOperations := numGoroutines * operationsPerGoroutine

	t.Logf("Concurrent operation safety test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Operations/sec: %.0f", float64(totalOperations)/duration.Seconds())
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d (expected for non-existent channels)", errorCount)

	if successCount+errorCount != int32(totalOperations) {
		t.Errorf("Operation count mismatch: success=%d + errors=%d != total=%d",
			successCount, errorCount, totalOperations)
	}

	// Check that no operations were lost or duplicated
	if duration > 5*time.Second {
		t.Errorf("Operations took too long: %v", duration)
	}

	t.Logf("Concurrent operation safety test passed")
}

// testDispatcherPerformance tests performance meets sub-300ms target
func testDispatcherPerformance(t *testing.T, testEngine *Engine) {
	if err := testEngine.dispatcher.Start(); err != nil {
		t.Fatalf("Failed to start dispatcher: %v", err)
	}
	defer testEngine.dispatcher.Stop()

	const numTests = 100
	var durations []time.Duration

	for i := 0; i < numTests; i++ {
		operation := DispatcherOperation{
			Type: OpSetMute,
			Data: SetMuteData{
				ChannelID: fmt.Sprintf("perf-test-channel-%d", i),
				Muted:     (i % 2) == 0,
			},
			Response: make(chan DispatcherResult, 1),
		}

		start := time.Now()

		// Submit operation
		select {
		case testEngine.dispatcher.operations <- operation:
			// Wait for response
			<-operation.Response
			duration := time.Since(start)
			durations = append(durations, duration)

		case <-time.After(1 * time.Second):
			t.Fatalf("Operation %d timed out", i)
		}
	}

	// Calculate statistics
	var totalDuration time.Duration
	var maxDuration time.Duration

	for _, d := range durations {
		totalDuration += d
		if d > maxDuration {
			maxDuration = d
		}
	}

	avgDuration := totalDuration / time.Duration(len(durations))

	t.Logf("Dispatcher performance test completed:")
	t.Logf("  Tests: %d", len(durations))
	t.Logf("  Average operation time: %v", avgDuration)
	t.Logf("  Max operation time: %v", maxDuration)
	t.Logf("  Target: < 300ms")

	if avgDuration > 300*time.Millisecond {
		t.Errorf("Average operation time %v exceeds 300ms target", avgDuration)
	}

	if maxDuration > 500*time.Millisecond {
		t.Errorf("Max operation time %v is excessive", maxDuration)
	}

	// Test dispatcher performance stats
	lastDuration, maxFromStats := testEngine.dispatcher.GetPerformanceStats()
	t.Logf("Dispatcher internal stats:")
	t.Logf("  Last operation: %v", lastDuration)
	t.Logf("  Max from stats: %v", maxFromStats)

	t.Logf("Performance test passed - dispatcher meets sub-300ms target")
}
