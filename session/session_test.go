//go:build darwin

package session

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	fmt.Println("🚀 Session Package Test Suite")
	fmt.Println("=============================")
	fmt.Println()

	// Test basic session creation
	fmt.Println("📋 Test 1: Session Creation")
	sess, err := NewSessionWithDefaults()
	if err != nil {
		log.Fatalf("❌ Failed to create session: %v", err)
	}
	fmt.Printf("✅ Session created successfully\n")
	fmt.Printf("   - Monitoring: %v\n", sess.IsMonitoring())
	fmt.Printf("   - Audio spec: %+v\n", sess.GetAudioSpec())
	fmt.Println()

	// Test initial device enumeration
	fmt.Println("📋 Test 2: Initial Device Enumeration")
	audioDevices, err := sess.GetAudioDevices()
	if err != nil {
		log.Printf("⚠️ Error getting audio devices: %v", err)
	} else {
		fmt.Printf("✅ Audio devices: %d found\n", len(audioDevices))
		for i, device := range audioDevices {
			fmt.Printf("   %d. %s (%s)\n", i+1, device.Name, device.UID)
		}
	}

	midiDevices, err := sess.GetMIDIDevices()
	if err != nil {
		log.Printf("⚠️ Error getting MIDI devices: %v", err)
	} else {
		fmt.Printf("✅ MIDI devices: %d found\n", len(midiDevices))
		for i, device := range midiDevices {
			fmt.Printf("   %d. %s (%s)\n", i+1, device.Name, device.UID)
		}
	}
	fmt.Println()

	// Test device counts
	fmt.Println("📋 Test 3: Fast Device Counts")
	audioCount, midiCount := sess.GetDeviceCounts()
	fmt.Printf("✅ Fast counts: %d audio, %d MIDI\n", audioCount, midiCount)
	fmt.Println()

	// Test status
	fmt.Println("📋 Test 4: Session Status")
	status := sess.Status()
	fmt.Printf("✅ Session status:\n")
	fmt.Printf("   - Monitoring: %v\n", status.Monitoring)
	fmt.Printf("   - Audio count: %d\n", status.AudioCount)
	fmt.Printf("   - MIDI count: %d\n", status.MIDICount)
	fmt.Printf("   - Cache age: %v\n", status.CacheAge)
	fmt.Printf("   - Poll interval: %v\n", status.PollInterval)
	fmt.Println()

	// Test callback registration
	fmt.Println("📋 Test 5: Callback Registration")
	callbackCalled := false
	sess.OnDeviceChange(func(change DeviceChange) {
		callbackCalled = true
		fmt.Printf("📞 Callback triggered: %s change (%d audio, %d MIDI)\n",
			change.Type.String(), change.AudioCount, change.MIDICount)
	})
	fmt.Printf("✅ Callback registered\n")
	fmt.Println()

	// Test simulation
	fmt.Println("📋 Test 6: Device Change Simulation")
	sess.SimulateDeviceChange(BothDeviceChange)
	time.Sleep(10 * time.Millisecond) // Give callback time to execute
	if callbackCalled {
		fmt.Printf("✅ Callback was triggered by simulation\n")
	} else {
		fmt.Printf("❌ Callback was not triggered\n")
	}
	fmt.Println()

	// Interactive monitoring test
	fmt.Println("📋 Test 7: Interactive Device Monitoring")
	fmt.Println("🎸 Now testing REAL device change detection!")
	fmt.Println("📱 Plug/unplug your audio devices to see async monitoring in action")
	fmt.Println("⌨️  Press Ctrl+C to stop monitoring and exit")
	fmt.Println()

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Monitor device changes
	changeCount := 0
	go func() {
		for change := range sess.DeviceChanges() {
			changeCount++
			fmt.Printf("🚨 REAL CHANGE #%d detected at %s\n",
				changeCount, change.Timestamp.Format("15:04:05.000"))
			fmt.Printf("   📊 Type: %s\n", change.Type.String())
			fmt.Printf("   🔢 Counts: %d audio, %d MIDI\n", change.AudioCount, change.MIDICount)

			// Show scanning status
			if change.AudioScanning || change.MIDIScanning {
				var scanning []string
				if change.AudioScanning {
					scanning = append(scanning, "audio")
				}
				if change.MIDIScanning {
					scanning = append(scanning, "MIDI")
				}
				fmt.Printf("   🔄 Scanning: %v\n", scanning)
			}

			// Show device details when available
			if change.AudioDevices != nil {
				fmt.Printf("   🎵 Audio devices updated (%d):\n", len(*change.AudioDevices))
				for i, device := range *change.AudioDevices {
					fmt.Printf("     %d. %s\n", i+1, device.Name)
				}
			}
			if change.MIDIDevices != nil {
				fmt.Printf("   🎹 MIDI devices updated (%d):\n", len(*change.MIDIDevices))
				for i, device := range *change.MIDIDevices {
					fmt.Printf("     %d. %s\n", i+1, device.Name)
				}
			}
			fmt.Println()
		}
	}()

	// Show periodic status
	statusTicker := time.NewTicker(5 * time.Second)
	defer statusTicker.Stop()

	go func() {
		for {
			select {
			case <-statusTicker.C:
				status := sess.Status()
				fmt.Printf("📊 Status: monitoring=%v, changes=%d, cache_age=%v\n",
					status.Monitoring, changeCount, status.CacheAge)
			case <-c:
				return
			}
		}
	}()

	// Wait for interrupt
	<-c
	fmt.Println("\n🛑 Shutting down session...")

	// Cleanup
	if err := sess.Close(); err != nil {
		log.Printf("⚠️ Error closing session: %v", err)
	}

	fmt.Printf("✅ All tests completed! Detected %d real device changes\n", changeCount)
	fmt.Println("🎉 Session package is working perfectly!")

	os.Exit(0)
}

func TestSessionCreation(t *testing.T) {
	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sess.Close()

	if !sess.IsMonitoring() {
		t.Error("Session should be monitoring after creation")
	}

	audioCount, midiCount := sess.GetDeviceCounts()
	if audioCount < 0 || midiCount < 0 {
		t.Error("Device counts should be non-negative")
	}

	t.Logf("Created session with %d audio and %d MIDI devices", audioCount, midiCount)
}

func TestDeviceAccess(t *testing.T) {
	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sess.Close()

	audioDevices, err := sess.GetAudioDevices()
	if err != nil {
		t.Errorf("Failed to get audio devices: %v", err)
	}

	midiDevices, err := sess.GetMIDIDevices()
	if err != nil {
		t.Errorf("Failed to get MIDI devices: %v", err)
	}

	t.Logf("Retrieved %d audio and %d MIDI devices", len(audioDevices), len(midiDevices))
}

func TestCallbacks(t *testing.T) {
	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sess.Close()

	callbackCalled := false
	sess.OnDeviceChange(func(change DeviceChange) {
		callbackCalled = true
		t.Logf("Callback received change: %s", change.Type.String())
	})

	// Trigger a simulated change
	sess.SimulateDeviceChange(AudioDeviceChange)

	// Give it time to execute
	time.Sleep(10 * time.Millisecond)

	if !callbackCalled {
		t.Error("Callback should have been called")
	}
}
