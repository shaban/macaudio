package analyze

import (
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
)

func TestAnalyzePackageBasic(t *testing.T) {
	t.Log("Testing basic analyze package functionality...")

	// Create engine and mixer nodes for testing
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	// Create two mixers to simulate input and output
	inputMixer, err := node.CreateMixer()
	if err != nil || inputMixer == nil {
		t.Fatal("Failed to create input mixer")
	}
	defer node.ReleaseMixer(inputMixer)

	outputMixer, err := node.CreateMixer()
	if err != nil || outputMixer == nil {
		t.Fatal("Failed to create output mixer")
	}
	defer node.ReleaseMixer(outputMixer)

	// Attach nodes to engine
	err = eng.Attach(inputMixer)
	if err != nil {
		t.Fatalf("Failed to attach input mixer: %v", err)
	}

	err = eng.Attach(outputMixer)
	if err != nil {
		t.Fatalf("Failed to attach output mixer: %v", err)
	}

	// Test signal path analysis
	config := DefaultAnalysisConfig()
	t.Logf("Using analysis config: %+v", config)

	analysis, err := VerifySignalPath(eng.Ptr(), inputMixer, outputMixer, config)
	if err != nil {
		t.Fatalf("Failed to analyze signal path: %v", err)
	}

	t.Logf("✓ Signal path analysis completed:")
	t.Logf("  - Input detected: %v (RMS: %.6f)", analysis.InputDetected, analysis.InputRMS)
	t.Logf("  - Output detected: %v (RMS: %.6f)", analysis.OutputDetected, analysis.OutputRMS)
	t.Logf("  - Signal integrity: %v", analysis.SignalIntegrity)
	t.Logf("  - Gain change: %.2f dB", analysis.GainChange)
	t.Logf("  - Latency: %v", analysis.Latency)

	// Validate analysis (expecting no signal since no audio source connected)
	err = ValidatePathAnalysis(analysis, false, config)
	if err != nil {
		t.Errorf("Path analysis validation failed: %v", err)
	} else {
		t.Log("✓ Path analysis validation passed")
	}
}

func TestAnalyzeMonoToStereo(t *testing.T) {
	t.Log("Testing mono→stereo analysis...")

	// Create engine and mixers
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	monoInput, err := node.CreateMixer()
	if err != nil || monoInput == nil {
		t.Fatal("Failed to create mono input mixer")
	}
	defer node.ReleaseMixer(monoInput)

	stereoOutput, err := node.CreateMixer()
	if err != nil || stereoOutput == nil {
		t.Fatal("Failed to create stereo output mixer")
	}
	defer node.ReleaseMixer(stereoOutput)

	// Attach nodes
	err = eng.Attach(monoInput)
	if err != nil {
		t.Fatalf("Failed to attach mono input: %v", err)
	}

	err = eng.Attach(stereoOutput)
	if err != nil {
		t.Fatalf("Failed to attach stereo output: %v", err)
	}

	// Test different pan positions
	panPositions := []float32{-1.0, -0.5, 0.0, 0.5, 1.0}

	for _, expectedPan := range panPositions {
		t.Run(fmt.Sprintf("Pan_%.1f", expectedPan), func(t *testing.T) {
			config := DefaultAnalysisConfig()
			config.SampleDuration = 50 * time.Millisecond // Shorter for multiple tests

			analysis, err := AnalyzeMonoToStereo(eng.Ptr(), monoInput, stereoOutput, expectedPan, config)
			if err != nil {
				t.Fatalf("Failed to analyze mono→stereo with pan %.1f: %v", expectedPan, err)
			}

			t.Logf("✓ Mono→stereo analysis for pan %.1f:", expectedPan)
			t.Logf("  - Left RMS: %.6f", analysis.LeftChannelRMS)
			t.Logf("  - Right RMS: %.6f", analysis.RightChannelRMS)
			t.Logf("  - Pan position: %.2f", analysis.PanPosition)
			t.Logf("  - Total RMS: %.6f", analysis.TotalRMS)
			t.Logf("  - Balance: %.2f", analysis.Balance)
			t.Logf("  - Stereo width: %.6f", analysis.StereoWidth)
			t.Logf("  - Mono compatible: %v", analysis.MonoCompatible)

			// Validate stereo analysis
			err = ValidateStereoAnalysis(analysis, expectedPan, config)
			if err != nil {
				t.Errorf("Stereo analysis validation failed for pan %.1f: %v", expectedPan, err)
			} else {
				t.Logf("✓ Stereo analysis validation passed for pan %.1f", expectedPan)
			}
		})
	}
}

func TestAnalyzePluginChain(t *testing.T) {
	t.Log("Testing plugin chain analysis...")

	// Create engine and mixers to simulate chain input/output
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	chainInput, err := node.CreateMixer()
	if err != nil || chainInput == nil {
		t.Fatal("Failed to create chain input mixer")
	}
	defer node.ReleaseMixer(chainInput)

	chainOutput, err := node.CreateMixer()
	if err != nil || chainOutput == nil {
		t.Fatal("Failed to create chain output mixer")
	}
	defer node.ReleaseMixer(chainOutput)

	// Attach nodes
	err = eng.Attach(chainInput)
	if err != nil {
		t.Fatalf("Failed to attach chain input: %v", err)
	}

	err = eng.Attach(chainOutput)
	if err != nil {
		t.Fatalf("Failed to attach chain output: %v", err)
	}

	// Analyze the "chain" (really just two unconnected mixers for now)
	config := DefaultAnalysisConfig()
	analysis, err := AnalyzePluginChain(eng.Ptr(), chainInput, chainOutput, config)
	if err != nil {
		t.Fatalf("Failed to analyze plugin chain: %v", err)
	}

	t.Log("✓ Plugin chain analysis completed:")
	t.Logf("  - Input RMS: %.6f", analysis.InputRMS)
	t.Logf("  - Output RMS: %.6f", analysis.OutputRMS)
	t.Logf("  - Gain change: %.2f dB", analysis.GainChange)
	t.Logf("  - Is processing: %v", analysis.IsProcessing)
	t.Logf("  - Frames in: %d", analysis.FramesIn)
	t.Logf("  - Frames out: %d", analysis.FramesOut)
	t.Logf("  - Latency frames: %d", analysis.LatencyFrames)

	// Validate chain analysis (expecting no processing since no effects)
	err = ValidateChainAnalysis(analysis, false, config)
	if err != nil {
		t.Errorf("Chain analysis validation failed: %v", err)
	} else {
		t.Log("✓ Chain analysis validation passed")
	}
}

func TestAnalyzeBusSends(t *testing.T) {
	t.Log("Testing bus sends analysis...")

	// Create engine and nodes
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	channelOutput, err := node.CreateMixer()
	if err != nil || channelOutput == nil {
		t.Fatal("Failed to create channel output mixer")
	}
	defer node.ReleaseMixer(channelOutput)

	// Create two bus inputs
	bus1Input, err := node.CreateMixer()
	if err != nil || bus1Input == nil {
		t.Fatal("Failed to create bus 1 input mixer")
	}
	defer node.ReleaseMixer(bus1Input)

	bus2Input, err := node.CreateMixer()
	if err != nil || bus2Input == nil {
		t.Fatal("Failed to create bus 2 input mixer")
	}
	defer node.ReleaseMixer(bus2Input)

	// Attach all nodes
	err = eng.Attach(channelOutput)
	if err != nil {
		t.Fatalf("Failed to attach channel output: %v", err)
	}

	err = eng.Attach(bus1Input)
	if err != nil {
		t.Fatalf("Failed to attach bus 1 input: %v", err)
	}

	err = eng.Attach(bus2Input)
	if err != nil {
		t.Fatalf("Failed to attach bus 2 input: %v", err)
	}

	// Define bus inputs and expected send levels
	busInputs := []unsafe.Pointer{bus1Input, bus2Input}
	expectedSendLevels := []float32{0.3, 0.2} // 30% and 20% sends

	// Analyze bus sends
	config := DefaultAnalysisConfig()
	analysis, err := AnalyzeBusSends(eng.Ptr(), channelOutput, busInputs, expectedSendLevels, config)
	if err != nil {
		t.Fatalf("Failed to analyze bus sends: %v", err)
	}

	t.Log("✓ Bus sends analysis completed:")
	t.Logf("  - Channel level: %.6f", analysis.ChannelLevel)
	t.Logf("  - Total send energy: %.6f", analysis.TotalSendEnergy)

	for i, level := range analysis.SendLevels {
		t.Logf("  - Bus %d level: %.6f (ratio: %.3f, efficiency: %.3f)",
			i, level, analysis.SendRatios[i], analysis.SendEfficiency[i])
	}

	// Basic validation
	if len(analysis.SendLevels) != len(expectedSendLevels) {
		t.Errorf("Expected %d send levels, got %d", len(expectedSendLevels), len(analysis.SendLevels))
	} else {
		t.Log("✓ Bus sends analysis has correct number of buses")
	}
}

func TestAnalysisConfigValidation(t *testing.T) {
	t.Log("Testing analysis configuration...")

	config := DefaultAnalysisConfig()

	// Test configuration values
	if config.SampleDuration <= 0 {
		t.Error("Sample duration should be positive")
	}
	if config.MinSignalLevel < 0 {
		t.Error("Minimum signal level should be non-negative")
	}
	if config.MaxLatency <= 0 {
		t.Error("Max latency should be positive")
	}
	if config.ToleranceDB < 0 {
		t.Error("Tolerance DB should be non-negative")
	}
	if config.PanTolerance < 0 || config.PanTolerance > 1 {
		t.Error("Pan tolerance should be between 0 and 1")
	}

	t.Logf("✓ Default config validation passed:")
	t.Logf("  - Sample duration: %v", config.SampleDuration)
	t.Logf("  - Min signal level: %.6f", config.MinSignalLevel)
	t.Logf("  - Max latency: %v", config.MaxLatency)
	t.Logf("  - Tolerance DB: %.1f", config.ToleranceDB)
	t.Logf("  - Pan tolerance: %.2f", config.PanTolerance)
}

func TestAnalyzeErrorHandling(t *testing.T) {
	t.Log("Testing analyze error handling...")

	config := DefaultAnalysisConfig()
	config.SampleDuration = 10 * time.Millisecond // Faster for error tests

	// Test with nil engine pointer (realistic error scenario)
	_, err := VerifySignalPath(nil, nil, nil, config)
	if err == nil {
		t.Error("Expected error with nil engine")
	} else {
		t.Logf("✓ Correctly rejected nil engine: %v", err)
	}

	// Test with nil mono input (realistic error scenario)
	_, err = AnalyzeMonoToStereo(nil, nil, nil, 0.0, config)
	if err == nil {
		t.Error("Expected error with nil parameters")
	} else {
		t.Logf("✓ Correctly rejected nil parameters: %v", err)
	}

	// Test with nil chain output (realistic error scenario)
	_, err = AnalyzePluginChain(nil, nil, nil, config)
	if err == nil {
		t.Error("Expected error with nil parameters")
	} else {
		t.Logf("✓ Correctly rejected nil parameters: %v", err)
	}

	// Test bus sends with mismatched arrays (realistic user error)
	busInputs := []unsafe.Pointer{nil} // Single nil pointer
	sendLevels := []float32{0.3, 0.2}  // Two levels - mismatch!
	_, err = AnalyzeBusSends(nil, nil, busInputs, sendLevels, config)
	if err == nil {
		t.Error("Expected error with mismatched bus inputs and send levels")
	} else {
		t.Logf("✓ Correctly rejected mismatched arrays: %v", err)
	}

	t.Log("✓ Error handling tests passed - using realistic error scenarios")
}
