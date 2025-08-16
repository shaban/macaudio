package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shaban/macaudio"
	"github.com/shaban/macaudio/devices"
)

func main() {
	fmt.Println("MacAudio Engine v1.0 - Architecture Demonstration")
	fmt.Println("==================================================")
	
	// Create engine configuration
	config := macaudio.EngineConfig{
		BufferSize:   512,
		SampleRate:   44100.0,
		ErrorHandler: &macaudio.DefaultErrorHandler{},
	}
	
	// Optionally bind to default audio devices
	if audioDevices, err := devices.GetAudio(); err == nil && len(audioDevices) > 0 {
		// Use the first available output device
		for _, device := range audioDevices {
			if device.CanOutput() && device.IsOnline {
				config.AudioDeviceUID = device.UID
				fmt.Printf("Binding to audio device: %s (%s)\n", device.Name, device.UID)
				break
			}
		}
	}
	
	// Create and start engine
	engine, err := macaudio.NewEngine(config)
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}
	
	fmt.Println("Starting engine...")
	if err := engine.Start(); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}
	
	defer func() {
		fmt.Println("Stopping engine...")
		if err := engine.Stop(); err != nil {
			log.Printf("Error stopping engine: %v", err)
		}
	}()
	
	// Demonstrate channel creation
	fmt.Println("\nCreating channels...")
	
	// Create a playback channel
	playbackConfig := macaudio.PlaybackConfig{
		FilePath:    "/System/Library/Sounds/Ping.aiff", // Use system sound
		LoopEnabled: false,
		AutoStart:   false,
	}
	
	playbackChannel, err := engine.CreatePlaybackChannel("demo_playback", playbackConfig)
	if err != nil {
		log.Printf("Failed to create playback channel: %v", err)
	} else {
		fmt.Printf("Created playback channel: %s\n", playbackChannel.GetID())
	}
	
	// Create an auxiliary channel
	auxConfig := macaudio.AuxConfig{
		SendLevel:   0.5,
		ReturnLevel: 0.7,
		PreFader:    false,
	}
	
	auxChannel, err := engine.CreateAuxChannel("demo_aux", auxConfig)
	if err != nil {
		log.Printf("Failed to create aux channel: %v", err)
	} else {
		fmt.Printf("Created aux channel: %s\n", auxChannel.GetID())
	}
	
	// Demonstrate plugin chain (if plugins are available)
	fmt.Println("\nDemonstrating plugin chain...")
	if playbackChannel != nil {
		// This would work if we had plugins installed
		blueprint := macaudio.PluginBlueprint{
			Type:           "aufx",
			Subtype:        "rvb2",
			ManufacturerID: "appl",
			Name:           "ChromaVerb",
			IsInstalled:    false, // Will be set by Load()
		}
		
		instance, err := playbackChannel.AddPlugin(blueprint, 0)
		if err != nil {
			fmt.Printf("Plugin loading failed (expected): %v\n", err)
		} else {
			fmt.Printf("Added plugin: %s\n", instance.ID)
		}
	}
	
	// Demonstrate serialization
	fmt.Println("\nDemonstrating state serialization...")
	serializer := engine.GetSerializer()
	
	jsonState, err := serializer.SaveToJSON()
	if err != nil {
		log.Printf("Failed to serialize state: %v", err)
	} else {
		fmt.Printf("Engine state serialized (%d bytes)\n", len(jsonState))
		// Uncomment to see full state:
		// fmt.Printf("State JSON:\n%s\n", jsonState)
	}
	
	// Show engine status
	fmt.Println("\nEngine Status:")
	fmt.Printf("- Running: %v\n", engine.IsRunning())
	fmt.Printf("- Channels: %v\n", engine.ListChannels())
	fmt.Printf("- Master Volume: %.2f\n", func() float32 {
		vol, _ := engine.GetMasterChannel().GetMasterVolume()
		return vol
	}())
	
	// Show device monitoring
	monitor := engine.GetDeviceMonitor()
	avgTime, maxTime, checkCount := monitor.GetPerformanceStats()
	fmt.Printf("- Device Monitor: %v\n", monitor.IsRunning())
	fmt.Printf("- Polling Interval: %v\n", monitor.GetPollingInterval())
	fmt.Printf("- Monitor Performance: avg=%v, max=%v, checks=%d\n", avgTime, maxTime, checkCount)
	
	// Show dispatcher performance
	dispatcher := engine.GetDispatcher()
	lastDuration, maxDuration := dispatcher.GetPerformanceStats()
	fmt.Printf("- Dispatcher Performance: last=%v, max_target=%v\n", lastDuration, maxDuration)
	
	// Setup graceful shutdown
	fmt.Println("\nEngine running. Press Ctrl+C to stop...")
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Run for a bit to show monitoring
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(1 * time.Second)
			if !engine.IsRunning() {
				return
			}
			fmt.Printf("Engine heartbeat %d/5...\n", i+1)
		}
	}()
	
	<-sigChan
	fmt.Println("\nShutdown signal received.")
}
