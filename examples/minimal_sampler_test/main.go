package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shaban/macaudio/devices"
	"github.com/shaban/macaudio/engine"
)

func main() {
	fmt.Println("🎹 Minimal Sampler Test")
	fmt.Println("=======================")

	// Step 1: Create audio engine
	fmt.Println("🔧 Creating audio engine...")
	audioDevices, err := devices.GetAudio()
	if err != nil {
		log.Fatalf("❌ Failed to get audio devices: %v", err)
	}

	var outputDevice *devices.AudioDevice
	for _, device := range audioDevices {
		if device.CanOutput() {
			outputDevice = &device
			break
		}
	}

	if outputDevice == nil {
		log.Fatalf("❌ No audio output device found")
	}

	audioEngine, err := engine.NewEngine(outputDevice, 0, 512)
	if err != nil {
		log.Fatalf("❌ Failed to create engine: %v", err)
	}
	defer audioEngine.Destroy()

	// Step 2: Create sampler channel
	fmt.Println("🎹 Creating sampler channel...")
	samplerChannel, err := audioEngine.CreateSamplerChannel()
	if err != nil {
		log.Fatalf("❌ Failed to create sampler channel: %v", err)
	}

	// Step 3: Start engine
	fmt.Println("🚀 Starting audio engine...")
	audioEngine.Prepare()
	err = audioEngine.Start()
	if err != nil {
		log.Fatalf("❌ Failed to start engine: %v", err)
	}
	defer audioEngine.Stop()

	fmt.Printf("✅ Sampler channel created and engine started!\n")
	fmt.Printf("   Channel is sampler: %t\n", samplerChannel.IsSampler())

	// Step 4: Test single note
	fmt.Println("\n🎵 Playing test note (Middle C for 2 seconds)...")
	err = samplerChannel.StartNote(60, 100) // Middle C, velocity 100
	if err != nil {
		log.Printf("⚠️ Failed to start note: %v", err)
	} else {
		fmt.Println("✅ Note started")
	}

	time.Sleep(2 * time.Second)

	err = samplerChannel.StopNote(60)
	if err != nil {
		log.Printf("⚠️ Failed to stop note: %v", err)
	} else {
		fmt.Println("✅ Note stopped")
	}

	// Step 5: Test convenience function
	fmt.Println("\n🎼 Playing melody using PlayNote()...")
	notes := []int{60, 64, 67, 72} // C-E-G-C
	for i, note := range notes {
		fmt.Printf("   Playing note %d (MIDI %d)...\n", i+1, note)
		err = samplerChannel.PlayNote(note, 90, 500*time.Millisecond)
		if err != nil {
			log.Printf("⚠️ Failed to play note: %v", err)
		}
		time.Sleep(600 * time.Millisecond) // Slight overlap
	}

	fmt.Println("\n📊 Test Summary:")
	fmt.Println("   ✅ Engine created and started")
	fmt.Println("   ✅ Sampler channel created") 
	fmt.Println("   ✅ Notes triggered (may be silent - needs instrument loading)")
	fmt.Println("\n❓ Did you hear any sound?")
	fmt.Println("   If NO: Sampler needs instrument file (.dls/.sf2)")
	fmt.Println("   If YES: 🎉 Basic sampler functionality works!")

	time.Sleep(2 * time.Second)
}
