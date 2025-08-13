package sourcenode

import (
	"testing"
)

func TestSourceNode_New(t *testing.T) {
	sourceNode, err := NewSilent() // Use silent version for compatibility
	if err != nil {
		t.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()

	if sourceNode == nil {
		t.Fatal("Source node is nil")
	}

	if sourceNode.ptr == nil {
		t.Fatal("Source node ptr is nil")
	}
}

func TestSourceNode_GetNodePtr(t *testing.T) {
	sourceNode, err := NewSilent()
	if err != nil {
		t.Fatalf("Failed to create source node: %v", err)
	}
	defer sourceNode.Destroy()

	// Should return valid pointer for valid source node
	nodePtr, err := sourceNode.GetNodePtr()
	if err != nil {
		t.Fatalf("Failed to get node pointer: %v", err)
	}
	if nodePtr == nil {
		t.Error("GetNodePtr should not return nil for valid source node")
	}
}

func TestSourceNode_GetNodePtr_Nil(t *testing.T) {
	// Test nil source node
	var sourceNode *SourceNode
	_, err := sourceNode.GetNodePtr()
	if err == nil {
		t.Error("GetNodePtr should return error for nil source node")
	}

	// Test source node with nil ptr
	sourceNode = &SourceNode{ptr: nil}
	_, err = sourceNode.GetNodePtr()
	if err == nil {
		t.Error("GetNodePtr should return error for source node with nil ptr")
	}
}

func TestSourceNode_Destroy(t *testing.T) {
	sourceNode, err := NewSilent()
	if err != nil {
		t.Fatalf("Failed to create source node: %v", err)
	}

	// Destroy should work on valid source node
	err = sourceNode.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy source node: %v", err)
	}

	// After destroy, ptr should be nil (tests our resource cleanup)
	if sourceNode.ptr != nil {
		t.Error("Expected source node ptr to be nil after Destroy()")
	}

	// Multiple destroys should return error
	err = sourceNode.Destroy()
	if err == nil {
		t.Error("Expected error when destroying already destroyed source node")
	}

	// Should return error
	_, err = sourceNode.GetNodePtr()
	if err == nil {
		t.Error("Destroyed source node should return error for GetNodePtr()")
	}
}

func TestSourceNode_DestroyNil(t *testing.T) {
	var sourceNode *SourceNode

	// Should handle nil gracefully
	sourceNode.Destroy()

	// Should also handle source node with nil ptr
	sourceNode = &SourceNode{ptr: nil}
	sourceNode.Destroy()
}

func TestSourceNode_Bridge_Solidity(t *testing.T) {
	// Create multiple source nodes to test bridge stability
	const numNodes = 10
	sourceNodes := make([]*SourceNode, numNodes)

	// Create all nodes
	for i := 0; i < numNodes; i++ {
		var err error
		sourceNodes[i], err = NewSilent()
		if err != nil {
			t.Fatalf("Failed to create source node %d: %v", i, err)
		}

		// Each should have a valid pointer
		nodePtr, err := sourceNodes[i].GetNodePtr()
		if err != nil {
			t.Errorf("Source node %d failed to get pointer: %v", i, err)
		} else if nodePtr == nil {
			t.Errorf("Source node %d has nil pointer", i)
		}
	}

	// Destroy all nodes
	for i, node := range sourceNodes {
		err := node.Destroy()
		if err != nil {
			t.Errorf("Failed to destroy source node %d: %v", i, err)
		}

		// After destroy, should return error
		_, err = node.GetNodePtr()
		if err == nil {
			t.Errorf("Source node %d should return error after destroy", i)
		}
	}

	t.Logf("Successfully created and destroyed %d source nodes", numNodes)
}
