package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/shaban/macaudio/devices"
	"github.com/shaban/macaudio/engine"
)

func main() {
	// Initialize the audio engine
	audioDevices, err := devices.GetAudio()
	if err != nil {
		log.Fatalf("Failed to get audio devices: %v", err)
	}
	outputDevices := audioDevices.Outputs()
	fmt.Println("Using device:", outputDevices[0].Name)

	engine, err := engine.NewEngine(&outputDevices[0], 0, 64)
	if err != nil {
		log.Fatalf("Failed to create audio engine: %v", err)
	}
	defer engine.Destroy()

	// Create a playback channel
	channel, err := engine.CreatePlaybackChannel("./engine/idea.m4a")
	if err != nil {
		log.Fatalf("Failed to create playback channel: %v", err)
	}

	err = engine.Start()
	if err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	fmt.Println("Audio engine started successfully!")

	// Start playback
	err = channel.Play()
	if err != nil {
		log.Fatalf("Failed to start playback: %v", err)
	}

	fmt.Println("Audio file is now playing!")
	fmt.Println("Interactive mixer controls:")
	fmt.Println("  v <0-100>     - Set volume (e.g., 'v 40' for 40%)")
	fmt.Println("  p <-100-100>  - Set pan (e.g., 'p -50' for left, 'p 50' for right)")
	fmt.Println("  r <25-200>    - Set playback rate (e.g., 'r 50' for 0.5x speed, 'r 125' for 1.25x speed)")
	fmt.Println("  t <-12-12>    - Set pitch (e.g., 't -5' for -5 semitones, 't 7' for +7 semitones)")
	fmt.Println("  status        - Show current volume, pan, rate, and pitch")
	fmt.Println("  quit          - Exit")
	fmt.Print("\n> ")

	// Interactive command loop
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if command == "" {
			fmt.Print("> ")
			continue
		}

		parts := strings.Fields(command)
		if len(parts) == 0 {
			fmt.Print("> ")
			continue
		}

		switch parts[0] {
		case "v", "volume":
			if len(parts) != 2 {
				fmt.Println("Usage: v <0-100> (e.g., 'v 40' for 40% volume)")
			} else if volumePercent, err := strconv.Atoi(parts[1]); err != nil {
				fmt.Println("Error: Volume must be a number between 0-100")
			} else if volumePercent < 0 || volumePercent > 100 {
				fmt.Println("Error: Volume must be between 0-100")
			} else {
				volume := float32(volumePercent) / 100.0
				err := channel.SetVolume(volume)
				if err != nil {
					fmt.Printf("Error setting volume: %v\n", err)
				} else {
					fmt.Printf("Volume set to %d%% (%.2f)\n", volumePercent, volume)
				}
			}

		case "p", "pan":
			if len(parts) != 2 {
				fmt.Println("Usage: p <-100-100> (e.g., 'p -50' for left, 'p 0' for center, 'p 50' for right)")
			} else if panPercent, err := strconv.Atoi(parts[1]); err != nil {
				fmt.Println("Error: Pan must be a number between -100 and 100")
			} else if panPercent < -100 || panPercent > 100 {
				fmt.Println("Error: Pan must be between -100 and 100")
			} else {
				pan := float32(panPercent) / 100.0
				err := channel.SetPan(pan)
				if err != nil {
					fmt.Printf("Error setting pan: %v\n", err)
				} else {
					direction := "center"
					if panPercent < 0 {
						direction = "left"
					} else if panPercent > 0 {
						direction = "right"
					}
					fmt.Printf("Pan set to %d%% %s (%.2f)\n", panPercent, direction, pan)
				}
			}

		case "r", "rate":
			if len(parts) != 2 {
				fmt.Println("Usage: r <25-200> (e.g., 'r 50' for 0.5x speed, 'r 125' for 1.25x speed)")
			} else if ratePercent, err := strconv.Atoi(parts[1]); err != nil {
				fmt.Println("Error: Rate must be a number between 25-200")
			} else if ratePercent < 25 || ratePercent > 200 {
				fmt.Println("Error: Rate must be between 25 (0.25x) and 200 (2.0x)")
			} else {
				rate := float32(ratePercent) / 100.0
				err := channel.SetPlaybackRate(rate)
				if err != nil {
					fmt.Printf("Error setting rate: %v\n", err)
				} else {
					fmt.Printf("Playback rate set to %d%% (%.2fx speed)\n", ratePercent, rate)
				}
			}

		case "t", "pitch":
			if len(parts) != 2 {
				fmt.Println("Usage: t <-12-12> (e.g., 't -5' for -5 semitones, 't 7' for +7 semitones)")
			} else if pitchSemitones, err := strconv.Atoi(parts[1]); err != nil {
				fmt.Println("Error: Pitch must be a number between -12 and 12")
			} else if pitchSemitones < -12 || pitchSemitones > 12 {
				fmt.Println("Error: Pitch must be between -12 and +12 semitones")
			} else {
				pitch := float32(pitchSemitones)
				err := channel.SetPitch(pitch)
				if err != nil {
					fmt.Printf("Error setting pitch: %v\n", err)
				} else {
					sign := ""
					if pitchSemitones > 0 {
						sign = "+"
					}
					fmt.Printf("Pitch set to %s%d semitones\n", sign, pitchSemitones)
				}
			}

		case "status", "s":
			volume, err := channel.GetVolume()
			if err != nil {
				fmt.Printf("Error getting volume: %v\n", err)
			} else {
				fmt.Printf("Current volume: %.0f%% (%.2f)\n", volume*100, volume)
			}

			pan, err := channel.GetPan()
			if err != nil {
				fmt.Printf("Error getting pan: %v\n", err)
			} else {
				direction := "center"
				if pan < 0 {
					direction = "left"
				} else if pan > 0 {
					direction = "right"
				}
				fmt.Printf("Current pan: %.0f%% %s (%.2f)\n", pan*100, direction, pan)
			}

			rate, err := channel.GetPlaybackRate()
			if err != nil {
				fmt.Printf("Error getting rate: %v\n", err)
			} else {
				fmt.Printf("Current rate: %.0f%% (%.2fx speed)\n", rate*100, rate)
			}

			pitch, err := channel.GetPitch()
			if err != nil {
				fmt.Printf("Error getting pitch: %v\n", err)
			} else {
				sign := ""
				if pitch > 0 {
					sign = "+"
				}
				fmt.Printf("Current pitch: %s%.0f semitones\n", sign, pitch)
			}

		case "quit", "exit", "q":
			fmt.Println("Stopping audio engine...")
			return

		default:
			fmt.Printf("Unknown command: %s\n", parts[0])
			fmt.Println("Available commands: v (volume), p (pan), r (rate), t (pitch), status, quit")
		}

		fmt.Print("> ")
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}

	fmt.Println("Stopping audio engine...")
}
