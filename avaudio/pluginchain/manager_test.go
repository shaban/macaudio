package pluginchain

import (
	"testing"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/plugins"
)

func TestNewChainManager(t *testing.T) {
	// Create a test engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}

	manager := NewChainManager(config)

	if manager.GetChainCount() != 0 {
		t.Errorf("Expected 0 chains, got %d", manager.GetChainCount())
	}

	chains := manager.ListChains()
	if len(chains) != 0 {
		t.Errorf("Expected empty chain list, got %d chains", len(chains))
	}
}

func TestChainManagerBasicOperations(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create manager
	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Test CreateChain
	t.Run("CreateChain", func(t *testing.T) {
		chain, err := manager.CreateChain("Test Chain")
		if err != nil {
			t.Errorf("Failed to create chain: %v", err)
		}

		if chain.GetName() != "Test Chain" {
			t.Errorf("Expected chain name 'Test Chain', got '%s'", chain.GetName())
		}

		if manager.GetChainCount() != 1 {
			t.Errorf("Expected 1 chain, got %d", manager.GetChainCount())
		}

		if !manager.HasChain("Test Chain") {
			t.Error("Manager should have 'Test Chain'")
		}
	})

	// Test GetChain
	t.Run("GetChain", func(t *testing.T) {
		chain, err := manager.GetChain("Test Chain")
		if err != nil {
			t.Errorf("Failed to get chain: %v", err)
		}

		if chain.GetName() != "Test Chain" {
			t.Errorf("Expected chain name 'Test Chain', got '%s'", chain.GetName())
		}
	})

	// Test GetNonexistentChain
	t.Run("GetNonexistentChain", func(t *testing.T) {
		_, err := manager.GetChain("Nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent chain")
		}
	})

	// Test CreateDuplicateChain
	t.Run("CreateDuplicateChain", func(t *testing.T) {
		_, err := manager.CreateChain("Test Chain")
		if err == nil {
			t.Error("Expected error for duplicate chain name")
		}
	})

	// Test CreateEmptyName
	t.Run("CreateEmptyName", func(t *testing.T) {
		_, err := manager.CreateChain("")
		if err == nil {
			t.Error("Expected error for empty chain name")
		}
	})
}

func TestChainManagerMultipleChains(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create manager
	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Create multiple chains
	chainNames := []string{"Vocals", "Drums", "Bass", "Guitar"}

	for _, name := range chainNames {
		_, err := manager.CreateChain(name)
		if err != nil {
			t.Fatalf("Failed to create chain '%s': %v", name, err)
		}
	}

	// Test ListChains
	t.Run("ListChains", func(t *testing.T) {
		chains := manager.ListChains()
		if len(chains) != len(chainNames) {
			t.Errorf("Expected %d chains, got %d", len(chainNames), len(chains))
		}

		// Should be sorted
		expected := []string{"Bass", "Drums", "Guitar", "Vocals"}
		for i, name := range expected {
			if i >= len(chains) || chains[i] != name {
				t.Errorf("Expected chain %d to be '%s', got '%s'", i, name, chains[i])
			}
		}
	})

	// Test GetAllChains
	t.Run("GetAllChains", func(t *testing.T) {
		allChains := manager.GetAllChains()
		if len(allChains) != len(chainNames) {
			t.Errorf("Expected %d chains, got %d", len(chainNames), len(allChains))
		}

		for _, name := range chainNames {
			if chain, exists := allChains[name]; !exists {
				t.Errorf("Expected chain '%s' to exist", name)
			} else if chain.GetName() != name {
				t.Errorf("Expected chain name '%s', got '%s'", name, chain.GetName())
			}
		}
	})

	// Test HasChain
	t.Run("HasChain", func(t *testing.T) {
		for _, name := range chainNames {
			if !manager.HasChain(name) {
				t.Errorf("Expected to have chain '%s'", name)
			}
		}

		if manager.HasChain("Nonexistent") {
			t.Error("Should not have nonexistent chain")
		}
	})
}

func TestChainManagerRename(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create manager
	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Create chain
	_, err = manager.CreateChain("Old Name")
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}

	// Test RenameChain
	t.Run("RenameChain", func(t *testing.T) {
		err := manager.RenameChain("Old Name", "New Name")
		if err != nil {
			t.Errorf("Failed to rename chain: %v", err)
		}

		if manager.HasChain("Old Name") {
			t.Error("Old name should not exist after rename")
		}

		if !manager.HasChain("New Name") {
			t.Error("New name should exist after rename")
		}

		// Check that the chain's internal name was updated
		chain, err := manager.GetChain("New Name")
		if err != nil {
			t.Errorf("Failed to get renamed chain: %v", err)
		}

		if chain.GetName() != "New Name" {
			t.Errorf("Expected chain internal name 'New Name', got '%s'", chain.GetName())
		}
	})

	// Test rename to same name (no-op)
	t.Run("RenameSameName", func(t *testing.T) {
		err := manager.RenameChain("New Name", "New Name")
		if err != nil {
			t.Errorf("Expected no error for same name rename, got: %v", err)
		}
	})

	// Test rename to empty name
	t.Run("RenameEmptyName", func(t *testing.T) {
		err := manager.RenameChain("New Name", "")
		if err == nil {
			t.Error("Expected error for empty new name")
		}
	})

	// Test rename nonexistent chain
	t.Run("RenameNonexistent", func(t *testing.T) {
		err := manager.RenameChain("Nonexistent", "Some Name")
		if err == nil {
			t.Error("Expected error for renaming nonexistent chain")
		}
	})

	// Create another chain to test duplicate name
	manager.CreateChain("Another Chain")

	// Test rename to existing name
	t.Run("RenameToExisting", func(t *testing.T) {
		err := manager.RenameChain("New Name", "Another Chain")
		if err == nil {
			t.Error("Expected error for renaming to existing name")
		}
	})
}

func TestChainManagerDelete(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create manager
	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Create chains
	manager.CreateChain("Chain 1")
	manager.CreateChain("Chain 2")

	// Test DeleteChain
	t.Run("DeleteChain", func(t *testing.T) {
		originalCount := manager.GetChainCount()

		err := manager.DeleteChain("Chain 1")
		if err != nil {
			t.Errorf("Failed to delete chain: %v", err)
		}

		if manager.GetChainCount() != originalCount-1 {
			t.Errorf("Expected %d chains after delete, got %d", originalCount-1, manager.GetChainCount())
		}

		if manager.HasChain("Chain 1") {
			t.Error("Deleted chain should not exist")
		}

		if !manager.HasChain("Chain 2") {
			t.Error("Other chain should still exist")
		}
	})

	// Test delete nonexistent chain
	t.Run("DeleteNonexistent", func(t *testing.T) {
		err := manager.DeleteChain("Nonexistent")
		if err == nil {
			t.Error("Expected error for deleting nonexistent chain")
		}
	})

	// Test ClearAllChains
	t.Run("ClearAllChains", func(t *testing.T) {
		err := manager.ClearAllChains()
		if err != nil {
			t.Errorf("Failed to clear all chains: %v", err)
		}

		if manager.GetChainCount() != 0 {
			t.Errorf("Expected 0 chains after clear, got %d", manager.GetChainCount())
		}

		chains := manager.ListChains()
		if len(chains) != 0 {
			t.Errorf("Expected empty chain list after clear, got %d chains", len(chains))
		}
	})
}

func TestChainManagerWithEffects(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create manager
	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Get some plugins to work with
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) < 2 {
		t.Skip("Need at least 2 Apple AU effects for this test")
	}

	// Test CreateChainFromPluginInfos
	t.Run("CreateChainFromPluginInfos", func(t *testing.T) {
		testInfos := effectInfos[:2] // Take first 2 effects

		chain, err := manager.CreateChainFromPluginInfos("Effect Chain", testInfos)
		if err != nil {
			t.Errorf("Failed to create chain from plugin infos: %v", err)
		}

		if chain.GetEffectCount() != 2 {
			t.Errorf("Expected 2 effects in chain, got %d", chain.GetEffectCount())
		}

		if !manager.HasChain("Effect Chain") {
			t.Error("Manager should have created chain")
		}
	})

	// Test GetTotalEffectCount
	t.Run("GetTotalEffectCount", func(t *testing.T) {
		total := manager.GetTotalEffectCount()
		if total != 2 {
			t.Errorf("Expected 2 total effects, got %d", total)
		}
	})

	// Test GetChainsWithEffect
	t.Run("GetChainsWithEffect", func(t *testing.T) {
		chain, _ := manager.GetChain("Effect Chain")
		effectNames := chain.GetEffectNames()

		if len(effectNames) > 0 {
			chains := manager.GetChainsWithEffect(effectNames[0])
			if len(chains) != 1 || chains[0] != "Effect Chain" {
				t.Errorf("Expected ['Effect Chain'], got %v", chains)
			}
		}

		// Test with non-existent effect
		chains := manager.GetChainsWithEffect("Non-existent Effect")
		if len(chains) != 0 {
			t.Errorf("Expected empty result for non-existent effect, got %v", chains)
		}
	})
}

func TestChainManagerClone(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create manager
	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Get some plugins to work with
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) < 2 {
		t.Skip("Need at least 2 Apple AU effects for clone test")
	}

	// Create source chain with effects
	sourceChain, err := manager.CreateChainFromPluginInfos("Source", effectInfos[:2])
	if err != nil {
		t.Fatalf("Failed to create source chain: %v", err)
	}

	// Test CloneChain
	t.Run("CloneChain", func(t *testing.T) {
		targetChain, err := manager.CloneChain("Source", "Target")
		if err != nil {
			t.Errorf("Failed to clone chain: %v", err)
		}

		if targetChain.GetEffectCount() != sourceChain.GetEffectCount() {
			t.Errorf("Expected %d effects in cloned chain, got %d",
				sourceChain.GetEffectCount(), targetChain.GetEffectCount())
		}

		sourceNames := sourceChain.GetEffectNames()
		targetNames := targetChain.GetEffectNames()

		if len(sourceNames) != len(targetNames) {
			t.Errorf("Effect name lists should have same length")
		}

		for i, name := range sourceNames {
			if i >= len(targetNames) || targetNames[i] != name {
				t.Errorf("Expected effect %d to be '%s', got '%s'", i, name, targetNames[i])
			}
		}

		if !manager.HasChain("Target") {
			t.Error("Manager should have cloned chain")
		}
	})

	// Test clone nonexistent chain
	t.Run("CloneNonexistent", func(t *testing.T) {
		_, err := manager.CloneChain("Nonexistent", "Target2")
		if err == nil {
			t.Error("Expected error for cloning nonexistent chain")
		}
	})

	// Test clone to existing name
	t.Run("CloneToExisting", func(t *testing.T) {
		_, err := manager.CloneChain("Source", "Target")
		if err == nil {
			t.Error("Expected error for cloning to existing name")
		}
	})

	// Test clone to empty name
	t.Run("CloneToEmpty", func(t *testing.T) {
		_, err := manager.CloneChain("Source", "")
		if err == nil {
			t.Error("Expected error for cloning to empty name")
		}
	})
}

func TestChainManagerSummary(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create manager
	config := ManagerConfig{
		EnginePtr: eng.Ptr(),
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Test empty manager summary
	t.Run("EmptySummary", func(t *testing.T) {
		summary := manager.Summary()
		expected := "ChainManager: no chains"
		if summary != expected {
			t.Errorf("Expected '%s', got '%s'", expected, summary)
		}
	})

	// Create some chains
	manager.CreateChain("Chain 1")
	manager.CreateChain("Chain 2")

	// Test summary with chains
	t.Run("WithChainsSummary", func(t *testing.T) {
		summary := manager.Summary()
		expected := "ChainManager: 2 chains, 0 total effects"
		if summary != expected {
			t.Errorf("Expected '%s', got '%s'", expected, summary)
		}
	})

	// Test GetChainsSummary
	t.Run("GetChainsSummary", func(t *testing.T) {
		summaries := manager.GetChainsSummary()

		if len(summaries) != 2 {
			t.Errorf("Expected 2 chain summaries, got %d", len(summaries))
		}

		if _, exists := summaries["Chain 1"]; !exists {
			t.Error("Expected summary for 'Chain 1'")
		}

		if _, exists := summaries["Chain 2"]; !exists {
			t.Error("Expected summary for 'Chain 2'")
		}
	})
}

func TestChainManagerNilEngine(t *testing.T) {
	// Create manager with nil engine
	config := ManagerConfig{
		EnginePtr: nil,
	}
	manager := NewChainManager(config)
	defer manager.Release()

	// Test that CreateChain fails with nil engine
	t.Run("CreateChainNilEngine", func(t *testing.T) {
		_, err := manager.CreateChain("Test")
		if err == nil {
			t.Error("Expected error for creating chain with nil engine")
		}
	})
}
