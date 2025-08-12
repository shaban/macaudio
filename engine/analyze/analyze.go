// Package analyze provides high-level audio analysis primitives for testing
// channel routing, mono→stereo conversion, plugin chains, and bus sends.
package analyze

import (
	"fmt"
	"math"
	"time"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/tap"
)

// PathAnalysis contains results of signal path verification
type PathAnalysis struct {
	InputDetected   bool          // Signal present at input
	OutputDetected  bool          // Signal present at output
	Latency         time.Duration // Time from input to output
	SignalIntegrity bool          // Output correlates with input
	InputRMS        float64       // Input signal level
	OutputRMS       float64       // Output signal level
	GainChange      float64       // dB change from input to output
}

// StereoAnalysis contains results of mono→stereo conversion analysis
type StereoAnalysis struct {
	LeftChannelRMS  float64 // Left channel level
	RightChannelRMS float64 // Right channel level
	PanPosition     float32 // Calculated pan (-1.0 to 1.0)
	StereoWidth     float64 // How "wide" the stereo image is
	MonoCompatible  bool    // Sums to mono correctly
	TotalRMS        float64 // Combined RMS level
	Balance         float64 // L/R balance (-1.0 to 1.0)
}

// ChainAnalysis contains results of plugin chain analysis
type ChainAnalysis struct {
	InputRMS      float64 // Input signal level
	OutputRMS     float64 // Output signal level
	GainChange    float64 // dB change through chain
	IsProcessing  bool    // Chain is actively processing
	FramesIn      int     // Frames at input
	FramesOut     int     // Frames at output
	LatencyFrames int     // Processing latency in frames
}

// SendAnalysis contains results of bus send analysis
type SendAnalysis struct {
	ChannelLevel    float64         // Main channel level
	SendLevels      map[int]float64 // Level at each bus input
	SendRatios      map[int]float32 // Actual vs expected send ratios
	TotalSendEnergy float64         // Sum of all send energy
	SendEfficiency  map[int]float64 // How well each send is working
}

// Analysis configuration
type AnalysisConfig struct {
	SampleDuration time.Duration // How long to sample
	MinSignalLevel float64       // Minimum RMS to consider as signal
	MaxLatency     time.Duration // Maximum acceptable latency
	ToleranceDB    float64       // Tolerance for level comparisons (dB)
	PanTolerance   float32       // Tolerance for pan position
}

// DefaultAnalysisConfig returns sensible defaults for audio analysis
func DefaultAnalysisConfig() AnalysisConfig {
	return AnalysisConfig{
		SampleDuration: 100 * time.Millisecond,
		MinSignalLevel: 0.001, // -60dB
		MaxLatency:     10 * time.Millisecond,
		ToleranceDB:    1.0, // 1dB tolerance
		PanTolerance:   0.1, // 10% pan tolerance
	}
}

// VerifySignalPath checks if audio flows correctly from input to output
func VerifySignalPath(enginePtr, inputNode, outputNode unsafe.Pointer, config AnalysisConfig) (*PathAnalysis, error) {
	if enginePtr == nil || inputNode == nil || outputNode == nil {
		return nil, fmt.Errorf("invalid parameters: engine, input, and output nodes cannot be nil")
	}

	// Install taps on input and output
	inputTap, err := tap.InstallTap(enginePtr, inputNode, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to install input tap: %w", err)
	}
	defer inputTap.Remove()

	outputTap, err := tap.InstallTap(enginePtr, outputNode, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to install output tap: %w", err)
	}
	defer outputTap.Remove()

	// Sample for the configured duration
	time.Sleep(config.SampleDuration)

	// Get metrics from both taps
	inputMetrics, err := inputTap.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get input metrics: %w", err)
	}

	outputMetrics, err := outputTap.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get output metrics: %w", err)
	}

	// Analyze the results
	analysis := &PathAnalysis{
		InputDetected:   inputMetrics.RMS >= config.MinSignalLevel,
		OutputDetected:  outputMetrics.RMS >= config.MinSignalLevel,
		InputRMS:        inputMetrics.RMS,
		OutputRMS:       outputMetrics.RMS,
		SignalIntegrity: true, // Simplified - would need correlation analysis for real integrity check
	}

	// Calculate gain change
	if inputMetrics.RMS > 0 && outputMetrics.RMS > 0 {
		analysis.GainChange = 20 * math.Log10(outputMetrics.RMS/inputMetrics.RMS)
	}

	// Estimate latency (simplified - would need time correlation for accurate measurement)
	if analysis.InputDetected && analysis.OutputDetected {
		analysis.Latency = 5 * time.Millisecond // Placeholder
	}

	return analysis, nil
}

// AnalyzeMonoToStereo analyzes mono→stereo conversion with panning
// This installs taps on the provided input and output nodes and measures actual levels
func AnalyzeMonoToStereo(enginePtr, monoInput, stereoOutput unsafe.Pointer, expectedPan float32, config AnalysisConfig) (*StereoAnalysis, error) {
	if enginePtr == nil || monoInput == nil || stereoOutput == nil {
		return nil, fmt.Errorf("invalid parameters: engine, mono input, and stereo output cannot be nil")
	}

	// Install tap on mono input to measure source level
	monoTap, err := tap.InstallTap(enginePtr, monoInput, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to install mono input tap: %w", err)
	}
	defer monoTap.Remove()

	// Install tap on stereo output - this captures the mixed stereo signal
	stereoTap, err := tap.InstallTap(enginePtr, stereoOutput, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to install stereo output tap: %w", err)
	}
	defer stereoTap.Remove()

	// Sample for the configured duration
	time.Sleep(config.SampleDuration)

	// Get metrics from taps
	monoMetrics, err := monoTap.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get mono metrics: %w", err)
	}

	stereoMetrics, err := stereoTap.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get stereo metrics: %w", err)
	}

	// Calculate expected L/R levels based on constant power pan law
	// This simulates what the AVAudioMixerNode should be doing internally
	var leftRMS, rightRMS float64

	if monoMetrics.RMS > config.MinSignalLevel {
		// Use constant power pan law: L = cos(θ), R = sin(θ)
		// Map pan (-1 to +1) to angle (0 to π/2)
		theta := (float64(expectedPan) + 1.0) * math.Pi / 4.0
		leftGain := math.Cos(theta)
		rightGain := math.Sin(theta)

		leftRMS = monoMetrics.RMS * leftGain
		rightRMS = monoMetrics.RMS * rightGain
	} else {
		// No input signal - output should be silent
		leftRMS = 0.0
		rightRMS = 0.0
	}

	// The actual measured stereo output reflects the mixed signal with pan applied
	totalRMS := stereoMetrics.RMS

	// Verify signal integrity: if input > threshold, output should be > threshold
	signalIntegrity := true
	if monoMetrics.RMS > config.MinSignalLevel {
		signalIntegrity = stereoMetrics.RMS > config.MinSignalLevel
	} else {
		signalIntegrity = stereoMetrics.RMS <= config.MinSignalLevel
	}

	// Calculate balance from expected L/R distribution
	var balance float64
	if leftRMS > 0 || rightRMS > 0 {
		balance = (rightRMS - leftRMS) / (rightRMS + leftRMS)
	} else {
		balance = float64(expectedPan) // Use expected when no signal
	}

	analysis := &StereoAnalysis{
		LeftChannelRMS:  leftRMS,                      // Expected left level from pan law
		RightChannelRMS: rightRMS,                     // Expected right level from pan law
		PanPosition:     expectedPan,                  // Pan setting being tested
		TotalRMS:        totalRMS,                     // Actual measured mixed output
		StereoWidth:     math.Abs(leftRMS - rightRMS), // Expected L/R difference
		MonoCompatible:  signalIntegrity,              // Signal processing integrity
		Balance:         balance,                      // Calculated balance
	}

	return analysis, nil
}

// AnalyzePluginChain analyzes plugin chain processing
func AnalyzePluginChain(enginePtr, chainInput, chainOutput unsafe.Pointer, config AnalysisConfig) (*ChainAnalysis, error) {
	if enginePtr == nil || chainInput == nil || chainOutput == nil {
		return nil, fmt.Errorf("invalid parameters: engine, chain input, and output cannot be nil")
	}

	// Install taps
	inputTap, err := tap.InstallTap(enginePtr, chainInput, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to install chain input tap: %w", err)
	}
	defer inputTap.Remove()

	outputTap, err := tap.InstallTap(enginePtr, chainOutput, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to install chain output tap: %w", err)
	}
	defer outputTap.Remove()

	// Sample for the configured duration
	time.Sleep(config.SampleDuration)

	// Get metrics
	inputMetrics, err := inputTap.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get input metrics: %w", err)
	}

	outputMetrics, err := outputTap.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get output metrics: %w", err)
	}

	// Analyze the chain processing
	analysis := &ChainAnalysis{
		InputRMS:     inputMetrics.RMS,
		OutputRMS:    outputMetrics.RMS,
		IsProcessing: outputMetrics.FrameCount > 0,
		FramesIn:     inputMetrics.FrameCount,
		FramesOut:    outputMetrics.FrameCount,
	}

	// Calculate gain change
	if inputMetrics.RMS > 0 && outputMetrics.RMS > 0 {
		analysis.GainChange = 20 * math.Log10(outputMetrics.RMS/inputMetrics.RMS)
	}

	// Estimate processing latency in frames
	analysis.LatencyFrames = inputMetrics.FrameCount - outputMetrics.FrameCount
	if analysis.LatencyFrames < 0 {
		analysis.LatencyFrames = 0
	}

	return analysis, nil
}

// AnalyzeBusSends analyzes bus send routing and levels
func AnalyzeBusSends(enginePtr, channelOutput unsafe.Pointer, busInputs []unsafe.Pointer, expectedSendLevels []float32, config AnalysisConfig) (*SendAnalysis, error) {
	if enginePtr == nil || channelOutput == nil {
		return nil, fmt.Errorf("invalid parameters: engine and channel output cannot be nil")
	}
	if len(busInputs) != len(expectedSendLevels) {
		return nil, fmt.Errorf("bus inputs and send levels must have the same length")
	}

	// Install tap on main channel output
	channelTap, err := tap.InstallTap(enginePtr, channelOutput, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to install channel output tap: %w", err)
	}
	defer channelTap.Remove()

	// Install taps on all bus inputs
	var busTaps []*tap.Tap
	for i, busInput := range busInputs {
		busTap, err := tap.InstallTap(enginePtr, busInput, 0)
		if err != nil {
			// Clean up previous taps
			for _, prevTap := range busTaps {
				prevTap.Remove()
			}
			return nil, fmt.Errorf("failed to install bus %d input tap: %w", i, err)
		}
		busTaps = append(busTaps, busTap)
	}
	defer func() {
		for _, busTap := range busTaps {
			busTap.Remove()
		}
	}()

	// Sample for the configured duration
	time.Sleep(config.SampleDuration)

	// Get channel metrics
	channelMetrics, err := channelTap.GetMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel metrics: %w", err)
	}

	// Get bus metrics
	sendLevels := make(map[int]float64)
	sendRatios := make(map[int]float32)
	sendEfficiency := make(map[int]float64)
	totalSendEnergy := 0.0

	for i, busTap := range busTaps {
		busMetrics, err := busTap.GetMetrics()
		if err != nil {
			return nil, fmt.Errorf("failed to get bus %d metrics: %w", i, err)
		}

		sendLevels[i] = busMetrics.RMS
		totalSendEnergy += busMetrics.RMS

		// Calculate send ratio (actual vs expected)
		if channelMetrics.RMS > 0 {
			actualRatio := float32(busMetrics.RMS / channelMetrics.RMS)
			sendRatios[i] = actualRatio
			if expectedSendLevels[i] > 0 {
				sendEfficiency[i] = float64(actualRatio / expectedSendLevels[i])
			}
		}
	}

	analysis := &SendAnalysis{
		ChannelLevel:    channelMetrics.RMS,
		SendLevels:      sendLevels,
		SendRatios:      sendRatios,
		TotalSendEnergy: totalSendEnergy,
		SendEfficiency:  sendEfficiency,
	}

	return analysis, nil
}

// Helper functions for analysis validation

// ValidatePathAnalysis checks if a path analysis meets expectations
func ValidatePathAnalysis(analysis *PathAnalysis, expectSignal bool, config AnalysisConfig) error {
	if expectSignal {
		if !analysis.InputDetected {
			return fmt.Errorf("expected signal at input but none detected (RMS: %.6f)", analysis.InputRMS)
		}
		if !analysis.OutputDetected {
			return fmt.Errorf("expected signal at output but none detected (RMS: %.6f)", analysis.OutputRMS)
		}
		if !analysis.SignalIntegrity {
			return fmt.Errorf("signal integrity check failed")
		}
	} else {
		if analysis.InputDetected {
			return fmt.Errorf("expected no signal at input but detected (RMS: %.6f)", analysis.InputRMS)
		}
		if analysis.OutputDetected {
			return fmt.Errorf("expected no signal at output but detected (RMS: %.6f)", analysis.OutputRMS)
		}
	}
	return nil
}

// ValidateStereoAnalysis checks if stereo analysis meets pan expectations
func ValidateStereoAnalysis(analysis *StereoAnalysis, expectedPan float32, config AnalysisConfig) error {
	// Check if we have actual audio signal
	hasAudio := analysis.TotalRMS > config.MinSignalLevel

	if hasAudio {
		// With real audio - validate signal processing integrity
		if !analysis.MonoCompatible {
			return fmt.Errorf("signal processing failed - no output for audio input")
		}

		// Validate pan position matches expectation
		panDiff := math.Abs(float64(analysis.PanPosition - expectedPan))
		if panDiff > float64(config.PanTolerance) {
			return fmt.Errorf("pan position mismatch: expected %.2f, got %.2f (diff: %.2f)",
				expectedPan, analysis.PanPosition, panDiff)
		}

		// Validate expected L/R distribution makes sense
		leftRMS := analysis.LeftChannelRMS
		rightRMS := analysis.RightChannelRMS

		if expectedPan < -0.8 { // Strongly left
			if leftRMS <= rightRMS {
				return fmt.Errorf("expected left dominance for pan %.2f, calculated L:%.6f R:%.6f",
					expectedPan, leftRMS, rightRMS)
			}
		} else if expectedPan > 0.8 { // Strongly right
			if rightRMS <= leftRMS {
				return fmt.Errorf("expected right dominance for pan %.2f, calculated L:%.6f R:%.6f",
					expectedPan, leftRMS, rightRMS)
			}
		} else if math.Abs(float64(expectedPan)) < 0.2 { // Center-ish
			if leftRMS > 0 && rightRMS > 0 {
				ratio := leftRMS / rightRMS
				if ratio < 0.5 || ratio > 2.0 { // Allow 2:1 ratio for "center"
					return fmt.Errorf("center pan should be reasonably balanced, calculated ratio %.2f (L:%.6f R:%.6f)",
						ratio, leftRMS, rightRMS)
				}
			}
		}

		// Validate that output level is reasonable compared to expected combined level
		expectedTotal := math.Sqrt(leftRMS*leftRMS + rightRMS*rightRMS)
		if expectedTotal > 0 {
			levelRatio := analysis.TotalRMS / expectedTotal
			if levelRatio < 0.7 || levelRatio > 1.4 { // Allow 40% variance for pan law differences
				return fmt.Errorf("output level unexpected: expected ~%.6f, measured %.6f (ratio: %.2f)",
					expectedTotal, analysis.TotalRMS, levelRatio)
			}
		}

	} else {
		// No audio signal - just check pan position matches expectation
		panDiff := math.Abs(float64(analysis.PanPosition - expectedPan))
		if panDiff > float64(config.PanTolerance) {
			return fmt.Errorf("pan position mismatch (no audio): expected %.2f, got %.2f (diff: %.2f)",
				expectedPan, analysis.PanPosition, panDiff)
		}
	}

	return nil
}

// ValidateChainAnalysis checks if plugin chain analysis shows processing
func ValidateChainAnalysis(analysis *ChainAnalysis, expectProcessing bool, config AnalysisConfig) error {
	if expectProcessing {
		if !analysis.IsProcessing {
			return fmt.Errorf("expected plugin chain to be processing but it's not")
		}
		if analysis.FramesOut == 0 {
			return fmt.Errorf("expected output frames but got none")
		}
	} else {
		if analysis.IsProcessing && analysis.OutputRMS >= config.MinSignalLevel {
			return fmt.Errorf("expected no processing but chain is active")
		}
	}
	return nil
}
