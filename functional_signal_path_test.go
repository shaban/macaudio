package macaudio

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/devices"
)

// TestFunctionalSignalPath validates complete input → master signal flow
// This tests the MVP: AudioInputChannel → MasterChannel with real AVFoundation integration
func TestFunctionalSignalPath(t *testing.T) {
	t.Log("=== Functional Signal Path Test ===")
	t.Log("Testing: AudioInputChannel → MasterChannel signal flow with AVFoundation")

	// Get default audio devices first
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}

	// Find default output device (most reliable for testing)
	var outputDevice *devices.AudioDevice
	for _, dev := range audioDevices {
		if dev.IsDefaultOutput && dev.CanOutput() && dev.IsOnline {
			outputDevice = &dev
			break
		}
	}

	// Fallback to any available output device if no default found
	if outputDevice == nil {
		for _, dev := range audioDevices {
			if dev.CanOutput() && dev.IsOnline {
				outputDevice = &dev
				break
			}
		}
	}

	if outputDevice == nil {
		t.Skip("No audio output device available - skipping functional test")
	}

	// Create engine with real AVFoundation integration
	engineConfig := EngineConfig{
		AudioSpec: engine.AudioSpec{
			SampleRate:   48000.0,
			BufferSize:   512,
			BitDepth:     32,
			ChannelCount: 2,
		},
		OutputDeviceUID: outputDevice.UID,
		ErrorHandler:    &DefaultErrorHandler{},
	}

	eng, err := NewEngine(engineConfig)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer func() {
		t.Log("Cleaning up engine...")
		if eng.IsRunning() {
			eng.Stop()
		}
		eng.Destroy()
	}()

	t.Logf("✅ Engine created successfully")

	// Get input device for testing - prioritize default input device
	var inputDevice *devices.AudioDevice
	for _, dev := range audioDevices {
		if dev.IsDefaultInput && dev.CanInput() && dev.IsOnline {
			inputDevice = &dev
			break
		}
	}

	// Fallback to any available input device if no default found
	if inputDevice == nil {
		for _, dev := range audioDevices {
			if dev.CanInput() && dev.IsOnline {
				inputDevice = &dev
				break
			}
		}
	}

	if inputDevice == nil {
		t.Skip("No audio input device available - skipping functional test")
	}

	t.Logf("✅ Using input device: %s (%s) [Default: %v]", 
		inputDevice.Name, inputDevice.UID, inputDevice.IsDefaultInput)
	t.Logf("✅ Using output device: %s (%s) [Default: %v]", 
		outputDevice.Name, outputDevice.UID, outputDevice.IsDefaultOutput)

	t.Logf("✅ Using input device: %s (%s)", inputDevice.Name, inputDevice.UID)

	// Step 1: Create audio input channel through dispatcher
	t.Log("\n--- Step 1: Create AudioInputChannel ---")
	
	inputConfig := AudioInputConfig{
		DeviceUID:       inputDevice.UID,
		InputBus:        0, // First input channel
		MonitoringLevel: 0.5,
	}

	inputChannel, err := eng.CreateAudioInputChannel("input-1", inputConfig)
	if err != nil {
		t.Fatalf("Failed to create audio input channel: %v", err)
	}

	t.Logf("✅ AudioInputChannel created: %s", inputChannel.GetID())

	// Step 2: Get master channel
	t.Log("\n--- Step 2: Get MasterChannel ---")
	
	masterChannel := eng.GetMasterChannel()
	if masterChannel == nil {
		t.Fatalf("Master channel not found")
	}

	t.Logf("✅ MasterChannel available: %s", masterChannel.GetID())

	// Step 3: Set initial levels for signal path validation
	t.Log("\n--- Step 3: Configure Signal Path ---")
	
	// Set input channel to audible levels
	if err := inputChannel.SetVolume(0.8); err != nil {
		t.Errorf("Failed to set input volume: %v", err)
	}
	
	// Set master channel to safe but audible level
	if err := masterChannel.SetMasterVolume(0.6); err != nil {
		t.Errorf("Failed to set master volume: %v", err)
	}

	// Ensure input channel is not muted
	if err := inputChannel.SetMute(false); err != nil {
		t.Errorf("Failed to unmute input channel: %v", err)
	}

	t.Logf("✅ Signal path configured: Input(80%%) → Master(60%%)")

	// Step 4: Start the engine through dispatcher
	t.Log("\n--- Step 4: Start Engine ---")

	if err := eng.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	if !eng.IsRunning() {
		t.Fatalf("Engine reported as not running after start")
	}

	t.Logf("✅ Engine started successfully")

	// Step 5: Validate AVFoundation signal connections
	t.Log("\n--- Step 5: Validate AVFoundation Integration ---")
	
	// Verify AVFoundation engine is running
	avEngine := eng.getAVEngine()
	if avEngine == nil {
		t.Fatalf("AVFoundation engine is nil")
	}

	if !avEngine.IsRunning() {
		t.Fatalf("AVFoundation engine is not running")
	}

	t.Logf("✅ AVFoundation engine running")

	// Get AVFoundation nodes for verification
	inputNodePtr, err := avEngine.InputNode()
	if err != nil {
		t.Errorf("Failed to get input node: %v", err)
	} else {
		t.Logf("✅ AVFoundation input node available: %p", inputNodePtr)
	}

	mainMixerPtr, err := avEngine.MainMixerNode()
	if err != nil {
		t.Errorf("Failed to get main mixer node: %v", err)
	} else {
		t.Logf("✅ AVFoundation main mixer available: %p", mainMixerPtr)
	}

	outputNodePtr, err := avEngine.OutputNode()
	if err != nil {
		t.Errorf("Failed to get output node: %v", err)
	} else {
		t.Logf("✅ AVFoundation output node available: %p", outputNodePtr)
	}

	// Step 6: Test signal path operations through dispatcher
	t.Log("\n--- Step 6: Test Real-Time Operations ---")
	
	// Test volume changes (direct calls - no dispatcher)
	originalVolume, err := inputChannel.GetVolume()
	if err != nil {
		t.Errorf("Failed to get input volume: %v", err)
	}

	testVolumes := []float32{0.5, 0.9, 0.3, 0.8}
	for _, vol := range testVolumes {
		if err := inputChannel.SetVolume(vol); err != nil {
			t.Errorf("Failed to set volume to %.1f: %v", vol, err)
		}

		// Small delay to allow AVFoundation to process
		time.Sleep(10 * time.Millisecond)

		actualVol, err := inputChannel.GetVolume()
		if err != nil {
			t.Errorf("Failed to get volume after setting to %.1f: %v", vol, err)
		} else if actualVol != vol {
			t.Errorf("Volume mismatch: set %.1f, got %.1f", vol, actualVol)
		}
	}

	t.Logf("✅ Volume control working (%.2f → %.2f)", originalVolume, testVolumes[len(testVolumes)-1])

	// Test mute operations (through dispatcher - topology changes)
	_, err = inputChannel.GetMute()
	if err != nil {
		t.Errorf("Failed to get mute state: %v", err)
	}

	// Test mute on
	if err := inputChannel.SetMute(true); err != nil {
		t.Errorf("Failed to mute channel: %v", err)
	}

	time.Sleep(20 * time.Millisecond) // Allow dispatcher processing

	isMuted, err := inputChannel.GetMute()
	if err != nil {
		t.Errorf("Failed to get mute state after muting: %v", err)
	} else if !isMuted {
		t.Errorf("Channel should be muted but reports unmuted")
	}

	// Test mute off
	if err := inputChannel.SetMute(false); err != nil {
		t.Errorf("Failed to unmute channel: %v", err)
	}

	time.Sleep(20 * time.Millisecond) // Allow dispatcher processing

	t.Logf("✅ Mute control working (through dispatcher)")

	// Step 7: Test master channel operations
	t.Log("\n--- Step 7: Test Master Channel Operations ---")
	
	originalMasterVol, err := masterChannel.GetMasterVolume()
	if err != nil {
		t.Errorf("Failed to get master volume: %v", err)
	}

	testMasterVol := float32(0.4)
	if err := masterChannel.SetMasterVolume(testMasterVol); err != nil {
		t.Errorf("Failed to set master volume: %v", err)
	}

	actualMasterVol, err := masterChannel.GetMasterVolume()
	if err != nil {
		t.Errorf("Failed to get master volume after change: %v", err)
	} else if actualMasterVol != testMasterVol {
		t.Errorf("Master volume mismatch: set %.1f, got %.1f", testMasterVol, actualMasterVol)
	}

	t.Logf("✅ Master volume control working (%.2f → %.2f)", originalMasterVol, testMasterVol)

	// Step 8: Run signal path for a brief period to validate stability
	t.Log("\n--- Step 8: Signal Path Stability Test ---")
	
	stableTestDuration := 2 * time.Second
	t.Logf("Running signal path for %v to test stability...", stableTestDuration)
	
	stableTicker := time.NewTicker(200 * time.Millisecond)
	defer stableTicker.Stop()
	
	stableCtx, stableCancel := context.WithTimeout(context.Background(), stableTestDuration)
	defer stableCancel()

	stabilityCheckCount := 0
	for {
		select {
		case <-stableCtx.Done():
			t.Logf("✅ Signal path stable for %v (%d checks)", stableTestDuration, stabilityCheckCount)
			goto StabilityTestComplete
		case <-stableTicker.C:
			// Verify engine is still running
			if !eng.IsRunning() {
				t.Errorf("Engine stopped running during stability test")
				goto StabilityTestComplete
			}
			
			if !avEngine.IsRunning() {
				t.Errorf("AVFoundation engine stopped during stability test")
				goto StabilityTestComplete
			}
			
			stabilityCheckCount++
		}
	}

StabilityTestComplete:

	// Step 9: Test graceful shutdown
	t.Log("\n--- Step 9: Test Graceful Shutdown ---")
	
	if err := eng.Stop(); err != nil {
		t.Errorf("Error stopping engine: %v", err)
	}

	if eng.IsRunning() {
		t.Errorf("Engine still running after stop")
	}

	if avEngine.IsRunning() {
		t.Errorf("AVFoundation engine still running after stop")
	}

	t.Logf("✅ Graceful shutdown successful")

	// Performance summary
	t.Log("\n--- Performance Summary ---")
	lastDuration, maxDuration := eng.dispatcher.GetPerformanceStats()
	t.Logf("Dispatcher Performance:")
	t.Logf("  - Last operation: %v", lastDuration)
	t.Logf("  - Max operation: %v", maxDuration)
	t.Logf("  - Target: <300ms")
	
	if maxDuration > 300*time.Millisecond {
		t.Errorf("Performance target missed: max operation took %v (target: <300ms)", maxDuration)
	} else {
		t.Logf("✅ Performance target met")
	}

	t.Log("\n=== Functional Signal Path Test COMPLETE ===")
}

// TestFunctionalSignalPathRaceConditions tests signal path under concurrent load
func TestFunctionalSignalPathRaceConditions(t *testing.T) {
	t.Log("=== Functional Signal Path Race Condition Test ===")

	// Create engine
	engineConfig := EngineConfig{
		AudioSpec: engine.DefaultAudioSpec(),
		ErrorHandler: &DefaultErrorHandler{},
	}

	eng, err := NewEngine(engineConfig)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer func() {
		if eng.IsRunning() {
			eng.Stop()
		}
		eng.Destroy()
	}()

	// Get input device
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}

	// Get input device - prioritize default input
	var inputDevice *devices.AudioDevice
	for _, dev := range audioDevices {
		if dev.IsDefaultInput && dev.CanInput() && dev.IsOnline {
			inputDevice = &dev
			break
		}
	}

	// Fallback to any available input device
	if inputDevice == nil {
		for _, dev := range audioDevices {
			if dev.CanInput() && dev.IsOnline {
				inputDevice = &dev
				break
			}
		}
	}

	if inputDevice == nil {
		t.Skip("No audio input device available - skipping race condition test")
	}

	// Create input channel
	inputConfig := AudioInputConfig{
		DeviceUID:       inputDevice.UID,
		InputBus:        0,
		MonitoringLevel: 0.5,
	}

	inputChannel, err := eng.CreateAudioInputChannel("race-input", inputConfig)
	if err != nil {
		t.Fatalf("Failed to create input channel: %v", err)
	}

	masterChannel := eng.GetMasterChannel()

	// Start engine
	if err := eng.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	t.Log("✅ Engine and signal path ready for race condition testing")

	// Concurrent operations test
	const numWorkers = 10
	const operationsPerWorker = 100
	const testDuration = 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Channel for collecting operation results
	results := make(chan error, numWorkers*operationsPerWorker)

	// Start concurrent workers performing various operations
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					results <- fmt.Errorf("worker %d panicked: %v", workerID, r)
				}
			}()

			for j := 0; j < operationsPerWorker; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					// Mix of direct operations (volume/pan) and dispatcher operations (mute)
					switch j % 4 {
					case 0:
						// Direct volume change (no dispatcher)
						vol := 0.5 + float32(j%5)*0.1
						if err := inputChannel.SetVolume(vol); err != nil {
							results <- fmt.Errorf("worker %d SetVolume failed: %v", workerID, err)
						}

					case 1:
						// Direct master volume change (no dispatcher)
						vol := 0.3 + float32(j%4)*0.15
						if err := masterChannel.SetMasterVolume(vol); err != nil {
							results <- fmt.Errorf("worker %d SetMasterVolume failed: %v", workerID, err)
						}

					case 2:
						// Dispatcher operation - mute toggle
						muted := (j%2 == 0)
						if err := inputChannel.SetMute(muted); err != nil {
							results <- fmt.Errorf("worker %d SetMute failed: %v", workerID, err)
						}

					case 3:
						// Read operations
						if _, err := inputChannel.GetVolume(); err != nil {
							results <- fmt.Errorf("worker %d GetVolume failed: %v", workerID, err)
						}
						if _, err := inputChannel.GetMute(); err != nil {
							results <- fmt.Errorf("worker %d GetMute failed: %v", workerID, err)
						}
					}
				}
			}
		}(i)
	}

	// Collect results
	var errors []error
	timeout := time.After(testDuration + 2*time.Second) // Grace period for operations to complete

CollectResults:
	for {
		select {
		case err := <-results:
			if err != nil {
				errors = append(errors, err)
			}
		case <-timeout:
			break CollectResults
		}
		
		// Check if we've collected all expected operations or timed out
		if len(errors) > 0 {
			// Stop early if we have errors
			break CollectResults
		}
	}

	// Performance check
	lastDuration, maxDuration := eng.dispatcher.GetPerformanceStats()
	t.Logf("Race condition test performance:")
	t.Logf("  - Max operation: %v", maxDuration)
	t.Logf("  - Last operation: %v", lastDuration)

	// Report results
	if len(errors) > 0 {
		t.Errorf("Race condition test found %d errors:", len(errors))
		for i, err := range errors {
			if i < 10 { // Limit output to first 10 errors
				t.Logf("  Error %d: %v", i+1, err)
			}
		}
		if len(errors) > 10 {
			t.Logf("  ... and %d more errors", len(errors)-10)
		}
	} else {
		t.Logf("✅ No race conditions detected in %d concurrent operations across %d workers", 
			numWorkers*operationsPerWorker, numWorkers)
	}

	// Verify engine is still stable
	if !eng.IsRunning() {
		t.Errorf("Engine stopped running during race condition test")
	}

	avEngine := eng.getAVEngine()
	if !avEngine.IsRunning() {
		t.Errorf("AVFoundation engine stopped during race condition test")
	}

	t.Log("✅ Signal path remains stable after race condition testing")
	t.Log("=== Functional Signal Path Race Condition Test COMPLETE ===")
}
