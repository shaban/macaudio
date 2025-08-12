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
	nodePtr := sourceNode.GetNodePtr()
	if nodePtr == nil {
		t.Fatal("Source node pointer is nil")
	}

	t.Logf("✓ Created source node and got node pointer")

	// Test GetNumberOfInputs - AVAudioSourceNode should have 0 inputs
	numInputs := GetNumberOfInputs(nodePtr)
	t.Logf("✓ Number of inputs: %d", numInputs)

	if numInputs != 0 {
		t.Errorf("Expected 0 inputs for source node, got %d", numInputs)
	}

	// Test GetNumberOfOutputs - AVAudioSourceNode should have 1 output
	numOutputs := GetNumberOfOutputs(nodePtr)
	t.Logf("✓ Number of outputs: %d", numOutputs)

	if numOutputs != 1 {
		t.Errorf("Expected 1 output for source node, got %d", numOutputs)
	}

	// Test IsInstalledOnEngine - should be false since not attached
	installed := IsInstalledOnEngine(nodePtr)
	t.Logf("✓ Installed on engine: %t", installed)

	if installed {
		t.Error("Expected node to not be installed on engine")
	}

	// Test LogInfo - should not crash
	t.Logf("✓ Logging node info:")
	LogInfo(nodePtr)
}

func TestNodeHelperValidation(t *testing.T) {
	sourceNode, err := sourcenode.NewSilent()
	if err != nil {
		t.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()

	nodePtr := sourceNode.GetNodePtr()

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
	// Test all functions with nil pointer - should not crash
	var nilPtr unsafe.Pointer = nil

	// Should return safe defaults
	numInputs := GetNumberOfInputs(nilPtr)
	if numInputs != 0 {
		t.Errorf("Expected 0 inputs for nil pointer, got %d", numInputs)
	}

	numOutputs := GetNumberOfOutputs(nilPtr)
	if numOutputs != 0 {
		t.Errorf("Expected 0 outputs for nil pointer, got %d", numOutputs)
	}

	installed := IsInstalledOnEngine(nilPtr)
	if installed {
		t.Error("Expected false for nil pointer installation check")
	}

	// These should not crash
	LogInfo(nilPtr)

	formatPtr := GetInputFormatForBus(nilPtr, 0)
	if formatPtr != nil {
		t.Error("Expected nil format for nil node pointer")
	}

	formatPtr = GetOutputFormatForBus(nilPtr, 0)
	if formatPtr != nil {
		t.Error("Expected nil format for nil node pointer")
	}

	// Validation should return errors
	err := ValidateInputBus(nilPtr, 0)
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

	nodePtr := sourceNode.GetNodePtr()

	// Test getting output format (source nodes have output but no input)
	formatPtr := GetOutputFormatForBus(nodePtr, 0)

	// Format might be nil until connected to engine - that's normal
	if formatPtr == nil {
		t.Logf("✓ Output format is nil (normal for unconnected node)")
	} else {
		t.Logf("✓ Got output format pointer: %p", formatPtr)
	}

	// Test getting input format - should always be nil for source nodes
	inputFormatPtr := GetInputFormatForBus(nodePtr, 0)
	if inputFormatPtr != nil {
		t.Error("Expected nil input format for source node (has no inputs)")
	} else {
		t.Logf("✓ Input format correctly nil for source node")
	}

	// Test invalid bus numbers - should return nil
	invalidFormatPtr := GetOutputFormatForBus(nodePtr, 999)
	if invalidFormatPtr != nil {
		t.Error("Expected nil format for invalid bus number")
	} else {
		t.Logf("✓ Invalid output bus correctly returns nil format")
	}

	invalidInputFormatPtr := GetInputFormatForBus(nodePtr, 999)
	if invalidInputFormatPtr != nil {
		t.Error("Expected nil format for invalid input bus number")
	} else {
		t.Logf("✓ Invalid input bus correctly returns nil format")
	}
}

// Tests for AVAudioMixerNode functionality

func TestCreateMixer(t *testing.T) {
	// Test mixer creation
	mixerPtr := CreateMixer()
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil")
	}
	defer ReleaseMixer(mixerPtr)

	// Test basic node properties
	inputs := GetNumberOfInputs(mixerPtr)
	outputs := GetNumberOfOutputs(mixerPtr)

	t.Logf("✓ Mixer has %d inputs and %d outputs", inputs, outputs)

	// AVAudioMixerNode should have 1 output
	if outputs != 1 {
		t.Errorf("Expected 1 output, got %d", outputs)
	}

	// Test that it's not yet installed on an engine
	if IsInstalledOnEngine(mixerPtr) {
		t.Error("Mixer should not be installed on engine initially")
	}

	t.Logf("✓ Mixer creation and basic properties test passed")
}

func TestMixerVolumeAndPan(t *testing.T) {
	mixerPtr := CreateMixer()
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil")
	}
	defer ReleaseMixer(mixerPtr)

	// Test setting volume
	testVolume := float32(0.75)
	err := SetMixerVolume(mixerPtr, testVolume, 0)
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
	mixerPtr := CreateMixer()
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil")
	}
	defer ReleaseMixer(mixerPtr)

	// Test invalid volume values
	err := SetMixerVolume(mixerPtr, -0.1, 0)
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
	mixerPtr := CreateMixer()
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil")
	}
	defer ReleaseMixer(mixerPtr)

	// Test invalid pan values
	err := SetMixerPan(mixerPtr, -1.1, 0)
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
	mixerPtr := CreateMixer()
	if mixerPtr == nil {
		t.Fatal("CreateMixer returned nil")
	}
	defer ReleaseMixer(mixerPtr)

	// This should log mixer information to console
	t.Logf("✓ Logging mixer info:")
	LogInfo(mixerPtr)
}

func TestMixerNilPointerHandling(t *testing.T) {
	// Test all mixer functions handle nil pointers gracefully

	// Mixer functions
	ReleaseMixer(nil) // Should not crash
	t.Logf("✓ ReleaseMixer(nil) handled gracefully")

	err := SetMixerVolume(nil, 0.5, 0)
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
