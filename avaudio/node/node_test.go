package node

import (
	"testing"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/sourcenode"
)

func TestNodeHelperFunctions(t *testing.T) {
	// Create a simple source node (no tone generation needed for testing helpers)
	sourceNode, err := sourcenode.NewSilent()
	if err != nil {
		t.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()

	// Get the node pointer (upcast to AVAudioNode*)
	nodePtr, err := sourceNode.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get node pointer: %v", err)
	}
	if nodePtr == nil {
		t.Fatal("Source node pointer is nil")
	}

	t.Logf("✓ Created source node and got node pointer")

	// Test GetNumberOfInputs - AVAudioSourceNode should have 0 inputs
	numInputs, err := GetNumberOfInputs(nodePtr)
	if err != nil {
		t.Fatalf("Failed to get number of inputs: %v", err)
	}
	t.Logf("✓ Number of inputs: %d", numInputs)

	if numInputs != 0 {
		t.Errorf("Expected 0 inputs for source node, got %d", numInputs)
	}

	// Test GetNumberOfOutputs - AVAudioSourceNode should have 1 output
	numOutputs, err := GetNumberOfOutputs(nodePtr)
	if err != nil {
		t.Fatalf("Failed to get number of outputs: %v", err)
	}
	t.Logf("✓ Number of outputs: %d", numOutputs)

	if numOutputs != 1 {
		t.Errorf("Expected 1 output for source node, got %d", numOutputs)
	}

	// Test IsInstalledOnEngine - should be false since not attached
	installed, err := IsInstalledOnEngine(nodePtr)
	if err != nil {
		t.Fatalf("Failed to check if installed on engine: %v", err)
	}
	t.Logf("✓ Installed on engine: %t", installed)

	if installed {
		t.Error("Expected node to not be installed on engine")
	}

	// Test LogInfo - should not crash
	t.Logf("✓ Logging node info:")
	err = LogInfo(nodePtr)
	if err != nil {
		t.Fatalf("Failed to log node info: %v", err)
	}
}

func TestNodeHelperValidation(t *testing.T) {
	sourceNode, err := sourcenode.NewSilent()
	if err != nil {
		t.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()

	nodePtr, err := sourceNode.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get node pointer: %v", err)
	}

	// Test bus validation - source nodes have 0 inputs, 1 output

	// Input bus validation - should fail for any bus (source has no inputs)
	err = ValidateInputBus(nodePtr, 0)
	if err == nil {
		t.Error("Expected error for input bus 0 on source node (has no inputs)")
	} else {
		t.Logf("✓ Input bus validation correctly failed: %v", err)
	}

	// Output bus validation - bus 0 should be valid
	err = ValidateOutputBus(nodePtr, 0)
	if err != nil {
		t.Errorf("Expected output bus 0 to be valid, got error: %v", err)
	} else {
		t.Logf("✓ Output bus 0 validation passed")
	}

	// Output bus validation - bus 1 should be invalid
	err = ValidateOutputBus(nodePtr, 1)
	if err == nil {
		t.Error("Expected error for output bus 1 (only has 1 output, bus 0)")
	} else {
		t.Logf("✓ Output bus validation correctly failed for bus 1: %v", err)
	}

	// Negative bus numbers should be invalid
	err = ValidateInputBus(nodePtr, -1)
	if err == nil {
		t.Error("Expected error for negative input bus number")
	} else {
		t.Logf("✓ Negative input bus correctly rejected")
	}

	err = ValidateOutputBus(nodePtr, -1)
	if err == nil {
		t.Error("Expected error for negative output bus number")
	} else {
		t.Logf("✓ Negative output bus correctly rejected")
	}
}

func TestNodeHelperNilSafety(t *testing.T) {
	// Test all functions with nil pointer - should return errors
	var nilPtr unsafe.Pointer = nil

	// Should return errors for nil pointer
	numInputs, err := GetNumberOfInputs(nilPtr)
	if err == nil {
		t.Error("Expected error for nil pointer")
	} else {
		t.Logf("✓ GetNumberOfInputs with nil pointer correctly rejected: %v", err)
	}
	if numInputs != 0 {
		t.Errorf("Expected 0 inputs for nil pointer, got %d", numInputs)
	}

	numOutputs, err := GetNumberOfOutputs(nilPtr)
	if err == nil {
		t.Error("Expected error for nil pointer")
	} else {
		t.Logf("✓ GetNumberOfOutputs with nil pointer correctly rejected: %v", err)
	}
	if numOutputs != 0 {
		t.Errorf("Expected 0 outputs for nil pointer, got %d", numOutputs)
	}

	installed, err := IsInstalledOnEngine(nilPtr)
	if err == nil {
		t.Error("Expected error for nil pointer")
	} else {
		t.Logf("✓ IsInstalledOnEngine with nil pointer correctly rejected: %v", err)
	}
	if installed {
		t.Error("Expected false for nil pointer installation check")
	}

	// LogInfo should return error for nil pointer
	err = LogInfo(nilPtr)
	if err == nil {
		t.Error("Expected error for nil pointer")
	} else {
		t.Logf("✓ LogInfo with nil pointer correctly rejected: %v", err)
	}

	formatPtr, err := GetInputFormatForBus(nilPtr, 0)
	if err == nil {
		t.Error("Expected error for nil node pointer")
	} else {
		t.Logf("✓ GetInputFormatForBus with nil pointer correctly rejected: %v", err)
	}
	if formatPtr != nil {
		t.Error("Expected nil format for nil node pointer")
	}

	formatPtr, err = GetOutputFormatForBus(nilPtr, 0)
	if err == nil {
		t.Error("Expected error for nil node pointer")
	} else {
		t.Logf("✓ GetOutputFormatForBus with nil pointer correctly rejected: %v", err)
	}
	if formatPtr != nil {
		t.Error("Expected nil format for nil node pointer")
	}

	// Validation should return errors
	err = ValidateInputBus(nilPtr, 0)
	if err == nil {
		t.Error("Expected error for nil pointer input bus validation")
	} else {
		t.Logf("✓ Nil pointer input validation error: %v", err)
	}

	err = ValidateOutputBus(nilPtr, 0)
	if err == nil {
		t.Error("Expected error for nil pointer output bus validation")
	} else {
		t.Logf("✓ Nil pointer output validation error: %v", err)
	}

	t.Logf("✓ All nil safety tests passed")
}

func TestGetFormatForBus(t *testing.T) {
	sourceNode, err := sourcenode.NewSilent()
	if err != nil {
		t.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()

	nodePtr, err := sourceNode.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get node pointer: %v", err)
	}

	// Test getting output format (source nodes have output but no input)
	formatPtr, err := GetOutputFormatForBus(nodePtr, 0)
	if err != nil {
		// Format might not be available until connected to engine - check if it's a reasonable error
		t.Logf("Output format not available (normal for unconnected node): %v", err)
	} else {
		t.Logf("✓ Got output format pointer: %p", formatPtr)
	}

	// Test getting input format - should return error for source nodes (no inputs)
	inputFormatPtr, err := GetInputFormatForBus(nodePtr, 0)
	if err == nil {
		t.Error("Expected error for source node input format (has no inputs)")
	} else {
		t.Logf("✓ Input format correctly rejected for source node: %v", err)
	}
	if inputFormatPtr != nil {
		t.Error("Expected nil input format for source node (has no inputs)")
	}

	// Test invalid bus numbers - should return errors
	invalidFormatPtr, err := GetOutputFormatForBus(nodePtr, 999)
	if err == nil {
		t.Error("Expected error for invalid bus number")
	} else {
		t.Logf("✓ Invalid output bus correctly rejected: %v", err)
	}
	if invalidFormatPtr != nil {
		t.Error("Expected nil format for invalid bus number")
	}

	invalidInputFormatPtr, err := GetInputFormatForBus(nodePtr, 999)
	if err == nil {
		t.Error("Expected error for invalid input bus number")
	} else {
		t.Logf("✓ Invalid input bus correctly rejected: %v", err)
	}
	if invalidInputFormatPtr != nil {
		t.Error("Expected nil format for invalid input bus number")
	}
}

// Tests for AVAudioMixerNode functionality

func TestCreateMixer(t *testing.T) {
	// Test mixer creation
	mixerPtr, err := CreateMixer()
	if err != nil {
		t.Fatalf("CreateMixer failed: %v", err)
	}
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil pointer")
	}
	defer func() {
		if err := ReleaseMixer(mixerPtr); err != nil {
			t.Logf("Warning: Failed to release mixer: %v", err)
		}
	}()

	// Test basic node properties
	inputs, err := GetNumberOfInputs(mixerPtr)
	if err != nil {
		t.Fatalf("Failed to get number of inputs: %v", err)
	}
	outputs, err := GetNumberOfOutputs(mixerPtr)
	if err != nil {
		t.Fatalf("Failed to get number of outputs: %v", err)
	}

	t.Logf("✓ Mixer has %d inputs and %d outputs", inputs, outputs)

	// AVAudioMixerNode should have 1 output
	if outputs != 1 {
		t.Errorf("Expected 1 output, got %d", outputs)
	}

	// Test that it's not yet installed on an engine
	installed, err := IsInstalledOnEngine(mixerPtr)
	if err != nil {
		t.Fatalf("Failed to check if installed on engine: %v", err)
	}
	if installed {
		t.Error("Mixer should not be installed on engine initially")
	}

	t.Logf("✓ Mixer creation and basic properties test passed")
}

func TestMixerVolumeAndPan(t *testing.T) {
	mixerPtr, err := CreateMixer()
	if err != nil {
		t.Fatalf("CreateMixer failed: %v", err)
	}
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil pointer")
	}
	defer func() {
		if err := ReleaseMixer(mixerPtr); err != nil {
			t.Logf("Warning: Failed to release mixer: %v", err)
		}
	}()

	// Test setting volume
	testVolume := float32(0.75)
	err = SetMixerVolume(mixerPtr, testVolume, 0)
	if err != nil {
		t.Fatalf("Failed to set mixer volume: %v", err)
	}

	// Test getting volume
	volume, err := GetMixerVolume(mixerPtr, 0)
	if err != nil {
		t.Fatalf("Failed to get mixer volume: %v", err)
	}

	t.Logf("✓ Set volume: %.2f, Got volume: %.2f", testVolume, volume)

	// Test setting pan
	testPan := float32(-0.5) // Left
	err = SetMixerPan(mixerPtr, testPan, 0)
	if err != nil {
		t.Fatalf("Failed to set mixer pan: %v", err)
	}

	// Test getting pan
	pan, err := GetMixerPan(mixerPtr, 0)
	if err != nil {
		t.Fatalf("Failed to get mixer pan: %v", err)
	}

	t.Logf("✓ Set pan: %.2f, Got pan: %.2f", testPan, pan)
}

func TestMixerVolumeValidation(t *testing.T) {
	mixerPtr, err := CreateMixer()
	if err != nil {
		t.Fatalf("CreateMixer failed: %v", err)
	}
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil pointer")
	}
	defer func() {
		if err := ReleaseMixer(mixerPtr); err != nil {
			t.Logf("Warning: Failed to release mixer: %v", err)
		}
	}()

	// Test invalid volume values
	err = SetMixerVolume(mixerPtr, -0.1, 0)
	if err == nil {
		t.Error("Expected error for negative volume")
	} else {
		t.Logf("✓ Negative volume correctly rejected: %v", err)
	}

	err = SetMixerVolume(mixerPtr, 1.1, 0)
	if err == nil {
		t.Error("Expected error for volume > 1.0")
	} else {
		t.Logf("✓ Volume > 1.0 correctly rejected: %v", err)
	}
}

func TestMixerPanValidation(t *testing.T) {
	mixerPtr, err := CreateMixer()
	if err != nil {
		t.Fatalf("CreateMixer failed: %v", err)
	}
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil pointer")
	}
	defer func() {
		if err := ReleaseMixer(mixerPtr); err != nil {
			t.Logf("Warning: Failed to release mixer: %v", err)
		}
	}()

	// Test invalid pan values
	err = SetMixerPan(mixerPtr, -1.1, 0)
	if err == nil {
		t.Error("Expected error for pan < -1.0")
	} else {
		t.Logf("✓ Pan < -1.0 correctly rejected: %v", err)
	}

	err = SetMixerPan(mixerPtr, 1.1, 0)
	if err == nil {
		t.Error("Expected error for pan > 1.0")
	} else {
		t.Logf("✓ Pan > 1.0 correctly rejected: %v", err)
	}
}

func TestMixerLogInfo(t *testing.T) {
	mixerPtr, err := CreateMixer()
	if err != nil {
		t.Fatalf("CreateMixer failed: %v", err)
	}
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil pointer")
	}
	defer func() {
		if err := ReleaseMixer(mixerPtr); err != nil {
			t.Logf("Warning: Failed to release mixer: %v", err)
		}
	}()

	// This should log mixer information to console
	t.Logf("✓ Logging mixer info:")
	err = LogInfo(mixerPtr)
	if err != nil {
		t.Fatalf("Failed to log mixer info: %v", err)
	}
}

func TestMixerNilPointerHandling(t *testing.T) {
	// Test all mixer functions handle nil pointers gracefully

	// ReleaseMixer should handle nil pointers gracefully
	err := ReleaseMixer(nil)
	if err == nil {
		t.Error("Expected error for nil mixer pointer")
	} else {
		t.Logf("✓ ReleaseMixer(nil) correctly rejected: %v", err)
	}

	err = SetMixerVolume(nil, 0.5, 0)
	if err == nil {
		t.Error("Expected error for nil mixer pointer")
	} else {
		t.Logf("✓ SetMixerVolume with nil pointer correctly rejected: %v", err)
	}

	err = SetMixerPan(nil, 0.0, 0)
	if err == nil {
		t.Error("Expected error for nil mixer pointer")
	} else {
		t.Logf("✓ SetMixerPan with nil pointer correctly rejected: %v", err)
	}

	_, err = GetMixerVolume(nil, 0)
	if err == nil {
		t.Error("Expected error for nil mixer pointer")
	} else {
		t.Logf("✓ GetMixerVolume with nil pointer correctly rejected: %v", err)
	}

	_, err = GetMixerPan(nil, 0)
	if err == nil {
		t.Error("Expected error for nil mixer pointer")
	} else {
		t.Logf("✓ GetMixerPan with nil pointer correctly rejected: %v", err)
	}
}
