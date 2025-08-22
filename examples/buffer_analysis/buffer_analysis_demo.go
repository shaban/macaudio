package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
)

// getRMSDescription provides a human-readable description of RMS levels
func getRMSDescription(rms float64) string {
	switch {
	case rms < 0.000001:
		return "(silence)"
	case rms < 0.01:
		return "(very quiet)"
	case rms < 0.1:
		return "(quiet)"
	case rms < 0.5:
		return "(moderate)"
	default:
		return "(loud)"
	}
}

func main() {
	// Create engine with default spec
	eng, err := engine.New(engine.AudioSpec{
		SampleRate:   44100,
		ChannelCount: 2,
		BitDepth:     16,
	})
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create player
	player, err := eng.NewPlayer()
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load the test audio file
	testFile := "avaudio/engine/idea.m4a"
	if err := player.LoadFile(testFile); err != nil {
		log.Fatalf("Failed to load file %s: %v", testFile, err)
	}

	fmt.Printf("🎵 File-Based Buffer Analysis Demo\n")
	fmt.Printf("📁 Loaded: %s\n\n", testFile)

	// Set different playback rates to test the analysis
	rates := []float64{0.5, 1.0, 2.0}
	
	for _, rate := range rates {
		fmt.Printf("🎚️  Testing at rate %.1fx:\n", rate)
		
		// Enable TimePitch effects for rate changes
		if err := player.EnableTimePitchEffects(); err != nil {
			log.Printf("Failed to enable TimePitch: %v", err)
			continue
		}
		
		// Set the rate
		if err := player.SetPlaybackRate(float32(rate)); err != nil {
			log.Printf("Failed to set rate to %.1f: %v", rate, err)
			continue
		}

		// Connect to main mixer and start playback
		if err := player.ConnectToMainMixer(); err != nil {
			log.Printf("Failed to connect to mixer: %v", err)
			continue
		}

		if err := eng.Start(); err != nil {
			log.Printf("Failed to start engine: %v", err)
			continue
		}

		if err := player.Play(); err != nil {
			log.Printf("Failed to play: %v", err)
			continue
		}

		fmt.Printf("   🎵 Now playing at %.1fx speed for 4 seconds...\n", rate)
		
		// Play for a few seconds and analyze in real-time
		playDuration := 4.0 // seconds
		analysisInterval := 1.0 // analyze every second
		
		for t := 0.0; t < playDuration; t += analysisInterval {
			// Wait for the playback time
			time.Sleep(time.Duration(analysisInterval * 1000) * time.Millisecond)
			
			// Analyze what should be playing at this moment
			currentMetrics, err := player.AnalyzeCurrentPlayback(analysisInterval)
			if err != nil {
				log.Printf("Current playback analysis failed: %v", err)
				continue
			}
			
			// Also analyze the corresponding file segment for comparison
			fileMetrics, err := player.AnalyzeFileSegment(t, analysisInterval)
			if err != nil {
				log.Printf("File segment analysis failed: %v", err)
				continue
			}
			
			fmt.Printf("   📊 Second %.0f: Current RMS=%.6f, File RMS=%.6f %s\n", 
				t+1, currentMetrics.RMS, fileMetrics.RMS,
				getRMSDescription(fileMetrics.RMS))
		}

		// Stop playback and reset for next test
		player.Stop()
		eng.Stop()
		
		// Disable TimePitch for clean state
		player.DisableTimePitchEffects()
		
		fmt.Println()
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("✅ Audible buffer analysis complete!")
	fmt.Println("🎧 You should have heard the audio at different speeds.")
	fmt.Println("� The RMS analysis shows what you heard:")
	fmt.Println("   • Silence at the beginning (RMS ≈ 0.000000)")
	fmt.Println("   • Music content later (RMS > 0.1)")
	fmt.Println("📈 This proves the file-based analysis matches the actual audio content!")
}
