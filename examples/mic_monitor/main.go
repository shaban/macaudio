// Package main provides a minimal working example of the MacAudio engine
// This example creates a live microphone monitor - mic input → speakers output
// Use this to validate the complete signal path and test tap functionality
package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shaban/macaudio"
	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/tap"
	"github.com/shaban/macaudio/devices"
)

func main() {
	fmt.Println("🎤 MacAudio Live Microphone Monitor")
	fmt.Println("===================================")
	fmt.Println("")

	// Get available audio devices
	fmt.Println("📱 Scanning audio devices...")
	audioDevices, err := devices.GetAudio()
	if err != nil {
		fmt.Printf("❌ Failed to get audio devices: %v\n", err)
		return
	}

	// Find and display input devices
	inputDevices := audioDevices.Inputs().Online()
	if len(inputDevices) == 0 {
		fmt.Println("❌ No audio input devices found")
		return
	}

	fmt.Println("🎤 Available Input Devices:")
	for i, dev := range inputDevices {
		defaultStr := ""
		if dev.IsDefaultInput {
			defaultStr = " [DEFAULT]"
		}
		fmt.Printf("  %d. %s (%s)%s\n", i+1, dev.Name, dev.DeviceType, defaultStr)
		fmt.Printf("     UID: %s, Channels: %d\n", dev.UID, dev.InputChannelCount)
	}

	// Find and display output devices
	outputDevices := audioDevices.Outputs().Online()
	if len(outputDevices) == 0 {
		fmt.Println("❌ No audio output devices found")
		return
	}

	fmt.Println("\n🔊 Available Output Devices:")
	for i, dev := range outputDevices {
		defaultStr := ""
		if dev.IsDefaultOutput {
			defaultStr = " [DEFAULT]"
		}
		fmt.Printf("  %d. %s (%s)%s\n", i+1, dev.Name, dev.DeviceType, defaultStr)
		fmt.Printf("     UID: %s, Channels: %d\n", dev.UID, dev.OutputChannelCount)
	}

	// Select input device (default to first default device)
	var selectedInput *devices.AudioDevice
	for _, dev := range inputDevices {
		if dev.IsDefaultInput {
			selectedInput = &dev
			break
		}
	}
	if selectedInput == nil {
		selectedInput = &inputDevices[0]
	}

	// Select output device (default to first default device)
	var selectedOutput *devices.AudioDevice
	for _, dev := range outputDevices {
		if dev.IsDefaultOutput {
			selectedOutput = &dev
			break
		}
	}
	if selectedOutput == nil {
		selectedOutput = &outputDevices[0]
	}

	fmt.Printf("\n✅ Using Input: %s\n", selectedInput.Name)
	fmt.Printf("✅ Using Output: %s\n", selectedOutput.Name)

	// Create engine configuration
	fmt.Println("\n🔧 Creating audio engine...")
	engineConfig := macaudio.EngineConfig{
		AudioSpec: engine.AudioSpec{
			SampleRate:   48000.0, // Professional sample rate
			BufferSize:   256,     // Low latency for live monitoring
			BitDepth:     32,      // High quality
			ChannelCount: 2,       // Stereo
		},
		OutputDeviceUID: selectedOutput.UID,
		ErrorHandler:    &macaudio.DefaultErrorHandler{},
	}

	audioEngine, err := macaudio.NewEngine(engineConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create engine: %v\n", err)
		return
	}
	defer func() {
		fmt.Println("🛑 Shutting down audio engine...")
		if audioEngine.IsRunning() {
			audioEngine.Stop()
		}
		audioEngine.Destroy()
		fmt.Println("✅ Engine shutdown complete")
	}()

	// Create input channel for microphone
	fmt.Println("🎤 Creating microphone input channel...")
	inputConfig := macaudio.AudioInputConfig{
		DeviceUID:       selectedInput.UID,
		InputBus:        0, // First input channel
		MonitoringLevel: 0.8, // Enable monitoring at 80%
	}

	inputChannel, err := audioEngine.CreateAudioInputChannel("mic-input", inputConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create input channel: %v\n", err)
		return
	}

	// Configure signal path
	fmt.Println("🔗 Configuring signal path...")
	
	// Set input volume to safe level
	if err := inputChannel.SetVolume(0.6); err != nil {
		fmt.Printf("⚠️ Failed to set input volume: %v\n", err)
	}

	// Set master volume to safe level for monitoring
	masterChannel := audioEngine.GetMasterChannel()
	if masterChannel == nil {
		fmt.Println("❌ Master channel not available")
		return
	}

	if err := masterChannel.SetMasterVolume(0.3); err != nil {
		fmt.Printf("⚠️ Failed to set master volume: %v\n", err)
	}

	// Ensure channel is not muted
	if err := inputChannel.SetMute(false); err != nil {
		fmt.Printf("⚠️ Failed to unmute input: %v\n", err)
	}

	fmt.Println("✅ Signal path configured: Mic(60%) → Master(30%)")

	// Start master channel to establish main mixer → output connection
	fmt.Println("\n🔗 Starting master channel...")
	if err := masterChannel.Start(); err != nil {
		fmt.Printf("⚠️ Failed to start master channel: %v\n", err)
	}

	// Start the engine
	fmt.Println("\n🚀 Starting audio engine...")
	if err := audioEngine.Start(); err != nil {
		fmt.Printf("❌ Failed to start engine: %v\n", err)
		return
	}

	if !audioEngine.IsRunning() {
		fmt.Println("❌ Engine not running after start")
		return
	}

	fmt.Println("✅ Audio engine running!")
	
	// Install audio tap on input channel for RMS monitoring
	var inputTap *tap.Tap
	fmt.Println("🔍 Installing audio tap for signal monitoring...")
	
	// Get the native engine and input node pointers
	enginePtr := audioEngine.GetNativeEngine()
	inputNodePtr := inputChannel.GetInputNode() // Tap the input node directly
	
	if enginePtr != nil && inputNodePtr != nil {
		var err error
		inputTap, err = tap.InstallTapWithKey(enginePtr, inputNodePtr, 0, "mic_input_monitor")
		if err != nil {
			fmt.Printf("⚠️ Failed to install input tap: %v\n", err)
		} else {
			fmt.Println("✅ Input audio tap installed - RMS monitoring active")
		}
	} else {
		fmt.Println("⚠️ Unable to install input tap - engine or node pointer unavailable")
	}

	// TODO: Add output tap on main mixer to debug volume controls
	// This would show the final processed signal going to speakers
	fmt.Println("")

	// Display current status
	fmt.Println("📊 Current Audio Status:")
	inputVol, _ := inputChannel.GetVolume()
	masterVol, _ := masterChannel.GetMasterVolume()
	inputMuted, _ := inputChannel.GetMute()
	
	fmt.Printf("  🎤 Input Volume: %.0f%%\n", inputVol*100)
	fmt.Printf("  🔊 Master Volume: %.0f%%\n", masterVol*100)
	fmt.Printf("  🔇 Input Muted: %v\n", inputMuted)

	// Interactive control loop
	fmt.Println("")
	fmt.Println("🎛️  Interactive Controls:")
	fmt.Println("  'i <volume>'  - Set input volume (0-100)")
	fmt.Println("  'm <volume>'  - Set master volume (0-100)")  
	fmt.Println("  'mute'       - Toggle input mute")
	fmt.Println("  'status'     - Show current settings")
	fmt.Println("  'tap'        - Show tap data (if available)")
	fmt.Println("  'quit'       - Exit")
	fmt.Println("")
	fmt.Println("🔊 You should now hear microphone input through your speakers!")
	fmt.Printf("   (Be careful of feedback - keep volume low or use headphones)\n")
	fmt.Println("")

	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print("macaudio> ")
		if !scanner.Scan() {
			break
		}
		
		command := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(command)
		
		if len(parts) == 0 {
			continue
		}
		
		switch strings.ToLower(parts[0]) {
		case "quit", "exit", "q":
			fmt.Println("👋 Goodbye!")
			return
			
		case "i", "input":
			if len(parts) < 2 {
				fmt.Println("Usage: i <volume> (0-100)")
				continue
			}
			vol, err := strconv.Atoi(parts[1])
			if err != nil || vol < 0 || vol > 100 {
				fmt.Println("Invalid volume. Use 0-100")
				continue
			}
			
			if err := inputChannel.SetVolume(float32(vol) / 100.0); err != nil {
				fmt.Printf("❌ Failed to set input volume: %v\n", err)
			} else {
				fmt.Printf("✅ Input volume set to %d%%\n", vol)
			}
			
		case "m", "master":
			if len(parts) < 2 {
				fmt.Println("Usage: m <volume> (0-100)")
				continue
			}
			vol, err := strconv.Atoi(parts[1])
			if err != nil || vol < 0 || vol > 100 {
				fmt.Println("Invalid volume. Use 0-100")
				continue
			}
			
			if err := masterChannel.SetMasterVolume(float32(vol) / 100.0); err != nil {
				fmt.Printf("❌ Failed to set master volume: %v\n", err)
			} else {
				fmt.Printf("✅ Master volume set to %d%%\n", vol)
			}
			
		case "mute":
			currentMute, _ := inputChannel.GetMute()
			newMute := !currentMute
			
			if err := inputChannel.SetMute(newMute); err != nil {
				fmt.Printf("❌ Failed to toggle mute: %v\n", err)
			} else {
				if newMute {
					fmt.Println("🔇 Input muted")
				} else {
					fmt.Println("🔊 Input unmuted")
				}
			}
			
		case "status":
			inputVol, _ := inputChannel.GetVolume()
			masterVol, _ := masterChannel.GetMasterVolume()
			inputMuted, _ := inputChannel.GetMute()
			
			fmt.Println("📊 Current Status:")
			fmt.Printf("  🎤 Input Volume: %.0f%%\n", inputVol*100)
			fmt.Printf("  🔊 Master Volume: %.0f%%\n", masterVol*100)
			fmt.Printf("  🔇 Input Muted: %v\n", inputMuted)
			fmt.Printf("  🚀 Engine Running: %v\n", audioEngine.IsRunning())
			
		case "tap":
			if inputTap != nil && inputTap.IsInstalled() {
				// Show real-time tap data for 3 seconds
				fmt.Println("📊 Live Audio Tap Data (3 seconds):")
				fmt.Println("  RMS Level  | Frame Count | Status")
				fmt.Println("  -----------|-------------|--------")
				
				start := time.Now()
				for time.Since(start) < 3*time.Second {
					metrics, err := inputTap.GetMetrics()
					if err != nil {
						fmt.Printf("  Error: %v\n", err)
						break
					}
					
					// Convert RMS to dB for more readable display
					var rmsDb string
					if metrics.RMS > 0.0001 { // Avoid log(0)
						rmsDbVal := 20 * math.Log10(metrics.RMS)
						if rmsDbVal > -60 {
							rmsDb = fmt.Sprintf("%.1f dB", rmsDbVal)
						} else {
							rmsDb = "< -60 dB"
						}
					} else {
						rmsDb = "Silent"
					}
					
					// Create simple visual bar
					barLength := int(metrics.RMS * 50) // Scale to 50 chars max
					if barLength > 50 {
						barLength = 50
					}
					bar := strings.Repeat("█", barLength) + strings.Repeat("░", 50-barLength)
					
					fmt.Printf("\r  %-9s | %11d | %s [%s]", 
						rmsDb, metrics.FrameCount, "Active", bar)
					
					time.Sleep(100 * time.Millisecond)
				}
				fmt.Println("\n📊 Tap monitoring complete")
			} else {
				fmt.Println("📊 Tap Data:")
				fmt.Println("  ❌ No audio tap installed")
				fmt.Println("  This could be due to:")
				fmt.Println("    • Engine not running")
				fmt.Println("    • Input channel not connected")
				fmt.Println("    • Native pointer unavailable")
			}
			
		case "help", "h":
			fmt.Println("Available commands:")
			fmt.Println("  i <volume>   - Set input volume (0-100)")
			fmt.Println("  m <volume>   - Set master volume (0-100)")
			fmt.Println("  mute         - Toggle input mute")
			fmt.Println("  status       - Show current settings")
			fmt.Println("  tap          - Show tap data")
			fmt.Println("  quit         - Exit")
			
		default:
			fmt.Printf("Unknown command: %s (type 'help' for available commands)\n", parts[0])
		}
	}
}
