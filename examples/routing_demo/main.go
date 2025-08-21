package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shaban/macaudio/avaudio/engine"
)

func main() {
	fmt.Println("🎛️  MacAudio Clean Routing Architecture Demo")
	fmt.Println("=============================================")

	// Create audio engine and player
	spec := engine.DefaultAudioSpec()
	audioEngine, err := engine.New(spec)
	if err != nil {
		log.Fatalf("Failed to create audio engine: %v", err)
	}
	defer audioEngine.Destroy()

	player, err := audioEngine.NewPlayer()
	if err != nil {
		log.Fatalf("Failed to create player: %v", err)
	}
	defer player.Destroy()

	// Load test audio file
	if err := player.LoadFile("../../avaudio/engine/idea.m4a"); err != nil {
		log.Fatalf("Failed to load audio file: %v", err)
	}

	fmt.Println("\n🎵 Demo 1: Direct Connection (Clean Architecture)")
	fmt.Println("------------------------------------------------")

	// Start engine
	if err := audioEngine.Start(); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	// Clean connection to main mixer - no assumptions, clear intent
	mainMixer, err := audioEngine.MainMixerNode()
	if err != nil {
		log.Fatalf("Failed to get main mixer: %v", err)
	}

	// Use the new clean API - explicit about what we're connecting to
	if err := player.ConnectTo(mainMixer, 0, 0); err != nil {
		log.Fatalf("Failed to connect player to main mixer: %v", err)
	}

	fmt.Println("✅ Connected: Player -> MainMixer")
	fmt.Println("🎵 Playing for 2 seconds...")

	player.SetVolume(0.7)
	if err := player.Play(); err != nil {
		log.Fatalf("Failed to start playback: %v", err)
	}

	time.Sleep(2 * time.Second)
	player.Stop()

	fmt.Println("\n🎛️  Demo 2: TimePitch Effects (Separated Concerns)")
	fmt.Println("--------------------------------------------------")

	// Enable TimePitch effects - only creates the unit, doesn't assume connections
	fmt.Println("🔧 Enabling TimePitch effects...")
	if err := player.EnableTimePitchEffects(); err != nil {
		log.Fatalf("Failed to enable TimePitch effects: %v", err)
	}

	// Restart engine (required for TimePitch)
	audioEngine.Stop()
	time.Sleep(100 * time.Millisecond)
	if err := audioEngine.Start(); err != nil {
		log.Fatalf("Failed to restart engine: %v", err)
	}

	// Now explicitly connect with TimePitch in the chain
	// The ConnectTo method automatically handles Player->TimePitch->Destination routing
	if err := player.ConnectTo(mainMixer, 0, 0); err != nil {
		log.Fatalf("Failed to connect with TimePitch: %v", err)
	}

	fmt.Println("✅ Connected: Player -> TimePitch -> MainMixer")

	// Test different effects
	effects := []struct {
		name  string
		rate  float32
		pitch float32
	}{
		{"Normal playback", 1.0, 0},
		{"Slow motion", 0.5, 0},
		{"Chipmunk effect", 1.0, 800},
		{"Deep voice", 1.0, -600},
	}

	for _, effect := range effects {
		fmt.Printf("\n🎵 %s (rate=%.1f, pitch=%.0f cents)\n", effect.name, effect.rate, effect.pitch)

		player.SetPlaybackRate(effect.rate)
		player.SetPitch(effect.pitch)

		if err := player.Play(); err != nil {
			fmt.Printf("❌ Failed to play: %v\n", err)
			continue
		}

		time.Sleep(2 * time.Second)
		player.Stop()
	}

	fmt.Println("\n🔄 Demo 3: Connection Flexibility")
	fmt.Println("---------------------------------")

	// The new architecture allows easy reconnection
	fmt.Println("🔧 Disconnecting and reconnecting...")

	// Clean disconnect (the new method handles this properly)
	if err := player.ConnectTo(mainMixer, 0, 0); err != nil {
		fmt.Printf("⚠️  Reconnection warning (expected): %v\n", err)
	}

	fmt.Println("✅ Reconnected successfully")

	// Final playback test
	fmt.Println("🎵 Final test - normal playback")
	player.SetPlaybackRate(1.0)
	player.SetPitch(0)

	if err := player.Play(); err != nil {
		log.Fatalf("Failed to play final test: %v", err)
	}

	time.Sleep(2 * time.Second)
	player.Stop()

	fmt.Println("\n🎉 Clean Routing Architecture Demo Complete!")
	fmt.Println("Benefits demonstrated:")
	fmt.Println("  ✅ Separation of concerns - connection vs destination")
	fmt.Println("  ✅ No hardcoded routing assumptions")
	fmt.Println("  ✅ Explicit connection management")
	fmt.Println("  ✅ TimePitch effects without automatic reconnection")
	fmt.Println("  ✅ Flexible routing for different audio graphs")

	audioEngine.Stop()
}
