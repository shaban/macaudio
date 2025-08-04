//go:build darwin && cgo

package devices

import (
	"reflect"
	"testing"
)

func TestAudioDeviceCompatibility(t *testing.T) {
	t.Log("Testing audio device compatibility methods...")

	// Create test devices with different capabilities
	device1 := AudioDevice{
		Device: Device{
			Name:     "Test Device 1",
			UID:      "test1",
			IsOnline: true,
		},
		SupportedSampleRates: []int{44100, 48000, 96000, 192000},
		SupportedBitDepths:   []int{16, 24, 32},
		InputChannelCount:    2,
		OutputChannelCount:   2,
	}

	device2 := AudioDevice{
		Device: Device{
			Name:     "Test Device 2", 
			UID:      "test2",
			IsOnline: true,
		},
		SupportedSampleRates: []int{48000, 96000, 176400, 192000},
		SupportedBitDepths:   []int{24, 32},
		InputChannelCount:    0,
		OutputChannelCount:   8,
	}

	device3 := AudioDevice{
		Device: Device{
			Name:     "Test Device 3",
			UID:      "test3", 
			IsOnline: true,
		},
		SupportedSampleRates: []int{44100, 48000},
		SupportedBitDepths:   []int{16, 24},
		InputChannelCount:    2,
		OutputChannelCount:   0,
	}

	// Test CommonSampleRates
	t.Run("CommonSampleRates", func(t *testing.T) {
		// Test intersection between device1 and device2
		common12 := device1.CommonSampleRates(device2)
		expected12 := []int{48000, 96000, 192000}
		if !reflect.DeepEqual(common12, expected12) {
			t.Errorf("Device1-Device2 common sample rates: expected %v, got %v", expected12, common12)
		}

		// Test intersection between device1 and device3
		common13 := device1.CommonSampleRates(device3)
		expected13 := []int{44100, 48000}
		if !reflect.DeepEqual(common13, expected13) {
			t.Errorf("Device1-Device3 common sample rates: expected %v, got %v", expected13, common13)
		}

		// Test intersection between device2 and device3
		common23 := device2.CommonSampleRates(device3)
		expected23 := []int{48000}
		if !reflect.DeepEqual(common23, expected23) {
			t.Errorf("Device2-Device3 common sample rates: expected %v, got %v", expected23, common23)
		}

		// Test symmetry (should be same regardless of order)
		common21 := device2.CommonSampleRates(device1)
		// Note: order follows first device, so different order but same values
		expectedRates := map[int]bool{48000: true, 96000: true, 192000: true}
		for _, rate := range common21 {
			if !expectedRates[rate] {
				t.Errorf("Unexpected rate in reverse order test: %d", rate)
			}
		}
		if len(common21) != 3 {
			t.Errorf("Expected 3 common rates in reverse order, got %d", len(common21))
		}
	})

	// Test CommonBitDepths
	t.Run("CommonBitDepths", func(t *testing.T) {
		// Test intersection between device1 and device2
		common12 := device1.CommonBitDepths(device2)
		expected12 := []int{24, 32}
		if !reflect.DeepEqual(common12, expected12) {
			t.Errorf("Device1-Device2 common bit depths: expected %v, got %v", expected12, common12)
		}

		// Test intersection between device1 and device3
		common13 := device1.CommonBitDepths(device3)
		expected13 := []int{16, 24}
		if !reflect.DeepEqual(common13, expected13) {
			t.Errorf("Device1-Device3 common bit depths: expected %v, got %v", expected13, common13)
		}

		// Test intersection between device2 and device3
		common23 := device2.CommonBitDepths(device3)
		expected23 := []int{24}
		if !reflect.DeepEqual(common23, expected23) {
			t.Errorf("Device2-Device3 common bit depths: expected %v, got %v", expected23, common23)
		}
	})

	// Test edge cases
	t.Run("EdgeCases", func(t *testing.T) {
		// Device with no sample rates
		emptyDevice := AudioDevice{
			Device:               Device{Name: "Empty", UID: "empty", IsOnline: true},
			SupportedSampleRates: []int{},
			SupportedBitDepths:   []int{},
		}

		// Test with empty device
		common := device1.CommonSampleRates(emptyDevice)
		if len(common) != 0 {
			t.Errorf("Expected no common rates with empty device, got %v", common)
		}

		common = emptyDevice.CommonSampleRates(device1)
		if len(common) != 0 {
			t.Errorf("Expected no common rates from empty device, got %v", common)
		}

		// Test bit depths with empty device
		commonDepths := device1.CommonBitDepths(emptyDevice)
		if len(commonDepths) != 0 {
			t.Errorf("Expected no common depths with empty device, got %v", commonDepths)
		}
	})

	// Test no intersection case
	t.Run("NoIntersection", func(t *testing.T) {
		deviceA := AudioDevice{
			Device:               Device{Name: "Device A", UID: "a", IsOnline: true},
			SupportedSampleRates: []int{44100, 88200},
			SupportedBitDepths:   []int{16},
		}

		deviceB := AudioDevice{
			Device:               Device{Name: "Device B", UID: "b", IsOnline: true},
			SupportedSampleRates: []int{48000, 96000},
			SupportedBitDepths:   []int{24, 32},
		}

		common := deviceA.CommonSampleRates(deviceB)
		if len(common) != 0 {
			t.Errorf("Expected no common rates, got %v", common)
		}

		commonDepths := deviceA.CommonBitDepths(deviceB)
		if len(commonDepths) != 0 {
			t.Errorf("Expected no common depths, got %v", commonDepths)
		}
	})

	t.Log("✅ All compatibility tests passed!")
}

func TestAudioDeviceCompatibilityWithRealDevices(t *testing.T) {
	t.Log("Testing compatibility methods with real audio devices...")

	// Get real devices for integration testing
	audioDevices, err := GetAllAudioDevices()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}

	if len(audioDevices) < 2 {
		t.Skip("Need at least 2 audio devices for compatibility testing")
	}

	// Test with first two real devices
	device1 := audioDevices[0]
	device2 := audioDevices[1]

	t.Run("RealDeviceCommonRates", func(t *testing.T) {
		common := device1.CommonSampleRates(device2)
		
		// Log the results for manual verification
		t.Logf("Device 1: %s - Sample rates: %v", device1.Name, device1.SupportedSampleRates)
		t.Logf("Device 2: %s - Sample rates: %v", device2.Name, device2.SupportedSampleRates)
		t.Logf("Common sample rates: %v", common)

		// Verify each common rate exists in both devices
		for _, rate := range common {
			found1 := false
			for _, r1 := range device1.SupportedSampleRates {
				if r1 == rate {
					found1 = true
					break
				}
			}
			
			found2 := false
			for _, r2 := range device2.SupportedSampleRates {
				if r2 == rate {
					found2 = true
					break
				}
			}

			if !found1 {
				t.Errorf("Common rate %d not found in device1 (%s)", rate, device1.Name)
			}
			if !found2 {
				t.Errorf("Common rate %d not found in device2 (%s)", rate, device2.Name)
			}
		}
	})

	t.Run("RealDeviceCommonDepths", func(t *testing.T) {
		common := device1.CommonBitDepths(device2)
		
		// Log the results for manual verification
		t.Logf("Device 1: %s - Bit depths: %v", device1.Name, device1.SupportedBitDepths)
		t.Logf("Device 2: %s - Bit depths: %v", device2.Name, device2.SupportedBitDepths)
		t.Logf("Common bit depths: %v", common)

		// Verify each common depth exists in both devices
		for _, depth := range common {
			found1 := false
			for _, d1 := range device1.SupportedBitDepths {
				if d1 == depth {
					found1 = true
					break
				}
			}
			
			found2 := false
			for _, d2 := range device2.SupportedBitDepths {
				if d2 == depth {
					found2 = true
					break
				}
			}

			if !found1 {
				t.Errorf("Common depth %d not found in device1 (%s)", depth, device1.Name)
			}
			if !found2 {
				t.Errorf("Common depth %d not found in device2 (%s)", depth, device2.Name)
			}
		}
	})

	// Test practical use case: find best input/output pair
	t.Run("PracticalUseCase", func(t *testing.T) {
		inputs := audioDevices.Inputs()
		outputs := audioDevices.Outputs()

		if len(inputs) == 0 || len(outputs) == 0 {
			t.Skip("Need both input and output devices for practical test")
		}

		inputDevice := inputs[0]
		outputDevice := outputs[0]

		// Find common capabilities for audio routing
		commonRates := inputDevice.CommonSampleRates(outputDevice)
		commonDepths := inputDevice.CommonBitDepths(outputDevice)

		t.Logf("Audio routing compatibility:")
		t.Logf("  Input: %s (%d channels)", inputDevice.Name, inputDevice.InputChannelCount)
		t.Logf("  Output: %s (%d channels)", outputDevice.Name, outputDevice.OutputChannelCount)
		t.Logf("  Common sample rates: %v", commonRates)
		t.Logf("  Common bit depths: %v", commonDepths)

		if len(commonRates) == 0 {
			t.Log("⚠️  No common sample rates found - these devices can't work together")
		} else {
			t.Logf("✅ Found %d compatible sample rates", len(commonRates))
		}

		if len(commonDepths) == 0 {
			t.Log("⚠️  No common bit depths found - these devices can't work together")
		} else {
			t.Logf("✅ Found %d compatible bit depths", len(commonDepths))
		}
	})

	t.Log("✅ All real device compatibility tests completed!")
}
