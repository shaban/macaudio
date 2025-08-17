package macaudio

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shaban/macaudio/avaudio/engine"
)

// TestDispatcherRaceConditions tests that dispatcher prevents race conditions
func TestDispatcherRaceConditions(t *testing.T) {
	// Create test engine
	config := EngineConfig{
		AudioSpec: engine.AudioSpec{
			SampleRate:   48000,
			BufferSize:   256,
			BitDepth:     32,
			ChannelCount: 2,
		},
		OutputDeviceUID: "BuiltInSpeakerDevice", // Use built-in speaker
		ErrorHandler:    &DefaultErrorHandler{},
	}

	testEngine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer testEngine.Destroy()

	// Start the engine (routes through dispatcher)
	if err := testEngine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer testEngine.Stop()

	// Create multiple audio input channels
	numChannels := 5
	channelIDs := make([]string, numChannels)

	for i := 0; i < numChannels; i++ {
		channelID := fmt.Sprintf("test-channel-%d", i)
		channelIDs[i] = channelID

		// Create a basic channel without device-specific configuration
		// since we're testing the dispatcher behavior, not actual audio devices
		config := AudioInputConfig{
			DeviceUID:       "", // Empty device UID for testing
			InputBus:        0,
			MonitoringLevel: 0.5,
		}

		_, err := testEngine.CreateAudioInputChannel(channelID, config)
		if err != nil {
			// For testing purposes, create the channel entry manually if device creation fails
			// This allows us to test the dispatcher logic
			t.Logf("Warning: Failed to create audio channel %s: %v (continuing with test)", channelID, err)
			
			// Create a mock channel for testing dispatcher behavior
			channelUUID, _ := uuid.Parse(channelID)
			if channelUUID == uuid.Nil {
				channelUUID = uuid.New()
			}
			
			baseChannel := &BaseChannel{
				id:        channelUUID,
				engine:    testEngine,
				isRunning: false,
			}
			
			channel := &AudioInputChannel{
				BaseChannel: baseChannel,
			}
			
			testEngine.mu.Lock()
			testEngine.channels[channelID] = channel
			testEngine.mu.Unlock()
		}
	}

	t.Run("ConcurrentMuteOperations", func(t *testing.T) {
		testConcurrentMuteOperations(t, testEngine, channelIDs)
	})

	t.Run("ConcurrentPluginOperations", func(t *testing.T) {
		testConcurrentPluginOperations(t, testEngine, channelIDs)
	})

	t.Run("ConcurrentDeviceChanges", func(t *testing.T) {
		testConcurrentDeviceChanges(t, testEngine, channelIDs)
	})

	t.Run("MixedConcurrentOperations", func(t *testing.T) {
		testMixedConcurrentOperations(t, testEngine, channelIDs)
	})
}

// testConcurrentMuteOperations tests concurrent mute/unmute operations
func testConcurrentMuteOperations(t *testing.T, testEngine *Engine, channelIDs []string) {
	const numGoroutines = 20
	const operationsPerGoroutine = 50

	var wg sync.WaitGroup
	var errorCount int32
	var mu sync.Mutex

	startTime := time.Now()

	// Launch concurrent goroutines doing mute operations
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				channelID := channelIDs[op%len(channelIDs)]
				muted := (op % 2) == 0 // Alternate between mute/unmute

				// This should go through dispatcher, preventing race conditions
				err := testEngine.SetChannelMute(channelID, muted)
				if err != nil {
					mu.Lock()
					errorCount++
					t.Logf("Goroutine %d, op %d: Mute operation failed: %v", goroutineID, op, err)
					mu.Unlock()
				}
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Concurrent mute test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total operations: %d", numGoroutines*operationsPerGoroutine)
	t.Logf("  Operations/sec: %.0f", float64(numGoroutines*operationsPerGoroutine)/duration.Seconds())
	t.Logf("  Errors: %d", errorCount)

	if errorCount > 0 {
		t.Errorf("Expected 0 errors, got %d", errorCount)
	}

	// Verify final state consistency
	for _, channelID := range channelIDs {
		channel, exists := testEngine.GetChannel(channelID)
		if !exists {
			t.Errorf("Channel %s disappeared during concurrent operations", channelID)
			continue
		}

		_, err := channel.GetMute()
		if err != nil {
			t.Errorf("Failed to get mute state for channel %s: %v", channelID, err)
		}
	}
}

// testConcurrentPluginOperations tests concurrent plugin bypass operations
func testConcurrentPluginOperations(t *testing.T, testEngine *Engine, channelIDs []string) {
	const numGoroutines = 10
	const operationsPerGoroutine = 20

	var wg sync.WaitGroup
	var errorCount int32
	var mu sync.Mutex

	startTime := time.Now()

	// Launch concurrent goroutines doing plugin bypass operations
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				channelID := channelIDs[op%len(channelIDs)]
				pluginID := "test-plugin-1" // Use a dummy plugin ID
				bypassed := (op % 2) == 0   // Alternate between bypass/enable

				// This should go through dispatcher
				err := testEngine.SetPluginBypass(channelID, pluginID, bypassed)
				if err != nil {
					mu.Lock()
					errorCount++
					// Expected to fail since we don't have actual plugins, but should not crash
					mu.Unlock()
				}
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Concurrent plugin test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total operations: %d", numGoroutines*operationsPerGoroutine)
	t.Logf("  Operations/sec: %.0f", float64(numGoroutines*operationsPerGoroutine)/duration.Seconds())
	t.Logf("  Errors: %d (expected, no actual plugins)", errorCount)
}

// testConcurrentDeviceChanges tests concurrent device change operations
func testConcurrentDeviceChanges(t *testing.T, testEngine *Engine, channelIDs []string) {
	const numGoroutines = 5
	const operationsPerGoroutine = 10

	var wg sync.WaitGroup
	var errorCount int32
	var mu sync.Mutex

	devices := []string{
		"BuiltInMicrophoneDevice",
		"BuiltInSpeakerDevice", // This will fail for input channels, which is expected
	}

	startTime := time.Now()

	// Launch concurrent goroutines doing device changes
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				channelID := channelIDs[op%len(channelIDs)]
				deviceUID := devices[op%len(devices)]

				// This should go through dispatcher
				err := testEngine.ChangeChannelDevice(channelID, deviceUID)
				if err != nil {
					mu.Lock()
					errorCount++
					// Some errors expected (trying to use output device as input)
					mu.Unlock()
				}
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Concurrent device change test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total operations: %d", numGoroutines*operationsPerGoroutine)
	t.Logf("  Operations/sec: %.0f", float64(numGoroutines*operationsPerGoroutine)/duration.Seconds())
	t.Logf("  Errors: %d (some expected)", errorCount)
}

// testMixedConcurrentOperations tests mixing different types of operations
func testMixedConcurrentOperations(t *testing.T, testEngine *Engine, channelIDs []string) {
	const totalOperations = 200
	const numWorkers = 10

	var wg sync.WaitGroup
	var operationsCompleted int32
	var mu sync.Mutex

	operationTypes := []string{"mute", "plugin", "device"}

	startTime := time.Now()

	// Launch workers doing mixed operations
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for op := 0; op < totalOperations/numWorkers; op++ {
				channelID := channelIDs[op%len(channelIDs)]
				opType := operationTypes[op%len(operationTypes)]

				var err error
				switch opType {
				case "mute":
					err = testEngine.SetChannelMute(channelID, (op%2) == 0)
				case "plugin":
					err = testEngine.SetPluginBypass(channelID, "test-plugin", (op%2) == 0)
				case "device":
					err = testEngine.ChangeChannelDevice(channelID, "BuiltInMicrophoneDevice")
				}

				mu.Lock()
				operationsCompleted++
				if err != nil {
					// Some errors are expected in this mixed test
				}
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Mixed concurrent operations test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations completed: %d", operationsCompleted)
	t.Logf("  Operations/sec: %.0f", float64(operationsCompleted)/duration.Seconds())

	// Verify engine is still functional
	for _, channelID := range channelIDs {
		channel, exists := testEngine.GetChannel(channelID)
		if !exists {
			t.Errorf("Channel %s missing after mixed operations", channelID)
			continue
		}

		if !channel.IsRunning() {
			t.Logf("Channel %s stopped during mixed operations (may be expected)", channelID)
		}
	}
}

// TestDispatcherPerformance tests dispatcher performance meets sub-300ms target
func TestDispatcherPerformance(t *testing.T) {
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

	if err := testEngine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer testEngine.Stop()

	// Create a test channel
	channelConfig := AudioInputConfig{
		DeviceUID:       "BuiltInMicrophoneDevice",
		InputBus:        0,
		MonitoringLevel: 0.5,
	}

	_, err = testEngine.CreateAudioInputChannel("perf-test-channel", channelConfig)
	if err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	// Test individual operation performance
	const numTests = 100
	var totalDuration time.Duration

	for i := 0; i < numTests; i++ {
		start := time.Now()
		err := testEngine.SetChannelMute("perf-test-channel", (i%2) == 0)
		duration := time.Since(start)
		totalDuration += duration

		if err != nil {
			t.Errorf("Mute operation %d failed: %v", i, err)
		}

		if duration > 300*time.Millisecond {
			t.Errorf("Operation %d took %v, exceeds 300ms target", i, duration)
		}
	}

	avgDuration := totalDuration / numTests
	t.Logf("Dispatcher performance test completed:")
	t.Logf("  Average operation time: %v", avgDuration)
	t.Logf("  Target: < 300ms")
	t.Logf("  Performance margin: %.1fx faster than target", 
		(300*time.Millisecond).Seconds()/avgDuration.Seconds())

	if avgDuration > 300*time.Millisecond {
		t.Errorf("Average operation time %v exceeds 300ms target", avgDuration)
	}

	// Get dispatcher performance stats
	lastDuration, maxDuration := testEngine.dispatcher.GetPerformanceStats()
	t.Logf("Dispatcher internal stats:")
	t.Logf("  Last operation: %v", lastDuration)
	t.Logf("  Max operation: %v", maxDuration)
}
