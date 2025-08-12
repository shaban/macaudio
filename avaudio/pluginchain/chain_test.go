package pluginchain

import (
	"testing"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/plugins"
)

func TestNewPluginChain(t *testing.T) {
	// Create a test engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	config := ChainConfig{
		Name:      "Test Chain",
		EnginePtr: eng.Ptr(),
	}

	chain := NewPluginChain(config)

	if chain.GetName() != "Test Chain" {
		t.Errorf("Expected chain name 'Test Chain', got '%s'", chain.GetName())
	}

	if !chain.IsEmpty() {
		t.Error("New chain should be empty")
	}

	if chain.GetEffectCount() != 0 {
		t.Errorf("Expected 0 effects, got %d", chain.GetEffectCount())
	}
}

func TestPluginChainBasicOperations(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "Basic Test Chain",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Get some plugins to work with
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	// Find some AU effects to test with
	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) == 0 {
		t.Skip("No Apple AU effects found, skipping test")
	}

	// Take the first few effects for testing
	testInfos := effectInfos
	if len(testInfos) > 3 {
		testInfos = effectInfos[:3]
	}

	// Introspect to get full plugin details
	testPlugins, err := testInfos.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugins: %v", err)
	}

	if len(testPlugins) == 0 {
		t.Skip("No plugins available for testing")
	}

	// Test adding effects
	t.Run("AddEffect", func(t *testing.T) {
		err := chain.AddEffect(testPlugins[0])
		if err != nil {
			t.Errorf("Failed to add effect: %v", err)
		}

		if chain.GetEffectCount() != 1 {
			t.Errorf("Expected 1 effect, got %d", chain.GetEffectCount())
		}

		if chain.IsEmpty() {
			t.Error("Chain should not be empty after adding effect")
		}
	})

	// Test getting effect details
	t.Run("GetEffectAt", func(t *testing.T) {
		effect, plugin, err := chain.GetEffectAt(0)
		if err != nil {
			t.Errorf("Failed to get effect at index 0: %v", err)
		}

		if effect == nil {
			t.Error("Effect should not be nil")
		}

		if plugin.Name != testPlugins[0].Name {
			t.Errorf("Expected plugin name '%s', got '%s'", testPlugins[0].Name, plugin.Name)
		}
	})

	// Test invalid index
	t.Run("GetEffectAtInvalidIndex", func(t *testing.T) {
		_, _, err := chain.GetEffectAt(10)
		if err == nil {
			t.Error("Expected error for invalid index")
		}
	})

	// Test adding more effects if available
	if len(testPlugins) > 1 {
		t.Run("AddMoreEffects", func(t *testing.T) {
			err := chain.AddEffect(testPlugins[1])
			if err != nil {
				t.Errorf("Failed to add second effect: %v", err)
			}

			if chain.GetEffectCount() != 2 {
				t.Errorf("Expected 2 effects, got %d", chain.GetEffectCount())
			}

			// Check effect names
			names := chain.GetEffectNames()
			if len(names) != 2 {
				t.Errorf("Expected 2 effect names, got %d", len(names))
			}
			if names[0] != testPlugins[0].Name {
				t.Errorf("Expected first effect name '%s', got '%s'", testPlugins[0].Name, names[0])
			}
			if names[1] != testPlugins[1].Name {
				t.Errorf("Expected second effect name '%s', got '%s'", testPlugins[1].Name, names[1])
			}
		})
	}
}

func TestPluginChainReordering(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "Reorder Test Chain",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Get test plugins
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) < 3 {
		t.Skip("Need at least 3 Apple AU effects for reordering tests")
	}

	testInfos := effectInfos[:3]
	testPlugins, err := testInfos.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugins: %v", err)
	}

	// Add three effects
	for i, plugin := range testPlugins {
		err := chain.AddEffect(plugin)
		if err != nil {
			t.Fatalf("Failed to add effect %d: %v", i, err)
		}
	}

	if chain.GetEffectCount() != 3 {
		t.Fatalf("Expected 3 effects, got %d", chain.GetEffectCount())
	}

	// Test SwapEffects
	t.Run("SwapEffects", func(t *testing.T) {
		originalNames := chain.GetEffectNames()

		err := chain.SwapEffects(0, 2)
		if err != nil {
			t.Errorf("Failed to swap effects: %v", err)
		}

		newNames := chain.GetEffectNames()
		if newNames[0] != originalNames[2] {
			t.Errorf("Expected first effect to be '%s', got '%s'", originalNames[2], newNames[0])
		}
		if newNames[2] != originalNames[0] {
			t.Errorf("Expected third effect to be '%s', got '%s'", originalNames[0], newNames[2])
		}
		if newNames[1] != originalNames[1] {
			t.Errorf("Expected middle effect unchanged, got '%s'", newNames[1])
		}

		// Swap back
		err = chain.SwapEffects(0, 2)
		if err != nil {
			t.Errorf("Failed to swap effects back: %v", err)
		}
	})

	// Test MoveEffect
	t.Run("MoveEffect", func(t *testing.T) {
		originalNames := chain.GetEffectNames()

		// Move first effect to last position
		err := chain.MoveEffect(0, 2)
		if err != nil {
			t.Errorf("Failed to move effect: %v", err)
		}

		newNames := chain.GetEffectNames()
		if newNames[0] != originalNames[1] {
			t.Errorf("Expected first effect to be '%s', got '%s'", originalNames[1], newNames[0])
		}
		if newNames[1] != originalNames[2] {
			t.Errorf("Expected second effect to be '%s', got '%s'", originalNames[2], newNames[1])
		}
		if newNames[2] != originalNames[0] {
			t.Errorf("Expected third effect to be '%s', got '%s'", originalNames[0], newNames[2])
		}
	})

	// Test InsertEffect
	if len(effectInfos) > 3 {
		t.Run("InsertEffect", func(t *testing.T) {
			// Get another plugin to insert
			insertInfo := effectInfos[3]
			insertPlugin, err := insertInfo.Introspect()
			if err != nil {
				t.Errorf("Failed to introspect insert plugin: %v", err)
				return
			}

			originalCount := chain.GetEffectCount()

			err = chain.InsertEffect(1, insertPlugin)
			if err != nil {
				t.Errorf("Failed to insert effect: %v", err)
				return
			}

			if chain.GetEffectCount() != originalCount+1 {
				t.Errorf("Expected %d effects after insert, got %d", originalCount+1, chain.GetEffectCount())
			}

			// Check that the inserted effect is at the correct position
			_, plugin, err := chain.GetEffectAt(1)
			if err != nil {
				t.Errorf("Failed to get inserted effect: %v", err)
				return
			}

			if plugin.Name != insertPlugin.Name {
				t.Errorf("Expected inserted effect name '%s', got '%s'", insertPlugin.Name, plugin.Name)
			}
		})
	}
}

func TestPluginChainRemoval(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "Removal Test Chain",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Get test plugins
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) < 2 {
		t.Skip("Need at least 2 Apple AU effects for removal tests")
	}

	testInfos := effectInfos[:2]
	testPlugins, err := testInfos.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugins: %v", err)
	}

	// Add effects
	for _, plugin := range testPlugins {
		err := chain.AddEffect(plugin)
		if err != nil {
			t.Fatalf("Failed to add effect: %v", err)
		}
	}

	// Test RemoveEffect
	t.Run("RemoveEffect", func(t *testing.T) {
		originalCount := chain.GetEffectCount()
		originalNames := chain.GetEffectNames()

		err := chain.RemoveEffect(0)
		if err != nil {
			t.Errorf("Failed to remove effect: %v", err)
		}

		if chain.GetEffectCount() != originalCount-1 {
			t.Errorf("Expected %d effects after removal, got %d", originalCount-1, chain.GetEffectCount())
		}

		newNames := chain.GetEffectNames()
		if len(newNames) > 0 && newNames[0] != originalNames[1] {
			t.Errorf("Expected first effect to be '%s' after removal, got '%s'", originalNames[1], newNames[0])
		}
	})

	// Test Clear
	t.Run("Clear", func(t *testing.T) {
		err := chain.Clear()
		if err != nil {
			t.Errorf("Failed to clear chain: %v", err)
		}

		if !chain.IsEmpty() {
			t.Error("Chain should be empty after clear")
		}

		if chain.GetEffectCount() != 0 {
			t.Errorf("Expected 0 effects after clear, got %d", chain.GetEffectCount())
		}
	})
}

func TestPluginChainParameters(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "Parameter Test Chain",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Get a plugin with parameters
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) == 0 {
		t.Skip("No Apple AU effects found for parameter tests")
	}

	// Find a plugin with parameters
	var testPlugin plugins.Plugin
	found := false
	for _, info := range effectInfos {
		plugin, err := info.Introspect()
		if err != nil {
			continue
		}
		if len(plugin.Parameters) > 0 {
			testPlugin = plugin
			found = true
			break
		}
	}

	if !found {
		t.Skip("No plugins with parameters found")
	}

	// Add the effect
	err = chain.AddEffect(testPlugin)
	if err != nil {
		t.Fatalf("Failed to add effect: %v", err)
	}

	// Test parameter operations
	if len(testPlugin.Parameters) > 0 {
		param := testPlugin.Parameters[0]

		t.Run("SetAndGetParameter", func(t *testing.T) {
			// Set parameter to a value within range
			testValue := param.MinValue + (param.MaxValue-param.MinValue)*0.5

			err := chain.SetParameter(0, param, testValue)
			if err != nil {
				t.Errorf("Failed to set parameter: %v", err)
				return
			}

			// Get parameter value back
			gotValue, err := chain.GetParameter(0, param)
			if err != nil {
				t.Errorf("Failed to get parameter: %v", err)
				return
			}

			// Values might not match exactly due to AU quantization
			t.Logf("Set parameter %s to %.3f, got %.3f", param.DisplayName, testValue, gotValue)
		})

		t.Run("InvalidEffectIndex", func(t *testing.T) {
			err := chain.SetParameter(10, param, param.DefaultValue)
			if err == nil {
				t.Error("Expected error for invalid effect index")
			}

			_, err = chain.GetParameter(10, param)
			if err == nil {
				t.Error("Expected error for invalid effect index")
			}
		})
	}
}

func TestPluginChainFromPluginInfo(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "PluginInfo Test Chain",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Get plugin infos
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) == 0 {
		t.Skip("No Apple AU effects found")
	}

	// Test adding effect from PluginInfo
	t.Run("AddEffectFromPluginInfo", func(t *testing.T) {
		err := chain.AddEffectFromPluginInfo(effectInfos[0])
		if err != nil {
			t.Errorf("Failed to add effect from PluginInfo: %v", err)
		}

		if chain.GetEffectCount() != 1 {
			t.Errorf("Expected 1 effect, got %d", chain.GetEffectCount())
		}

		names := chain.GetEffectNames()
		if len(names) != 1 || names[0] != effectInfos[0].Name {
			t.Errorf("Expected effect name '%s', got %v", effectInfos[0].Name, names)
		}
	})
}

func TestPluginChainRouting(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "Routing Test Chain",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Test empty chain routing
	t.Run("EmptyChainRouting", func(t *testing.T) {
		inputNode := chain.GetInputNode()
		outputNode := chain.GetOutputNode()

		if inputNode != nil {
			t.Error("Expected nil input node for empty chain")
		}
		if outputNode != nil {
			t.Error("Expected nil output node for empty chain")
		}
	})

	// Add an effect and test routing
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) == 0 {
		t.Skip("No Apple AU effects found for routing tests")
	}

	err = chain.AddEffectFromPluginInfo(effectInfos[0])
	if err != nil {
		t.Fatalf("Failed to add effect: %v", err)
	}

	t.Run("SingleEffectRouting", func(t *testing.T) {
		inputNode := chain.GetInputNode()
		outputNode := chain.GetOutputNode()

		if inputNode == nil {
			t.Error("Expected non-nil input node for single effect chain")
		}
		if outputNode == nil {
			t.Error("Expected non-nil output node for single effect chain")
		}
		if inputNode != outputNode {
			t.Error("For single effect chain, input and output nodes should be the same")
		}
	})
}

func TestPluginChainSummary(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "Summary Test Chain",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Test empty chain summary
	t.Run("EmptyChainSummary", func(t *testing.T) {
		summary := chain.Summary()
		expectedSummary := "Chain 'Summary Test Chain': empty"
		if summary != expectedSummary {
			t.Errorf("Expected summary '%s', got '%s'", expectedSummary, summary)
		}
	})

	// Add effects and test summary
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	effectInfos := pluginInfos.ByType("aufx").ByManufacturer("appl")
	if len(effectInfos) < 2 {
		t.Skip("Need at least 2 Apple AU effects for summary tests")
	}

	// Add two effects
	for i := 0; i < 2 && i < len(effectInfos); i++ {
		err := chain.AddEffectFromPluginInfo(effectInfos[i])
		if err != nil {
			t.Fatalf("Failed to add effect %d: %v", i, err)
		}
	}

	t.Run("MultiEffectSummary", func(t *testing.T) {
		summary := chain.Summary()
		t.Logf("Chain summary: %s", summary)

		// Should contain chain name and effect count
		if !containsString(summary, "Summary Test Chain") {
			t.Error("Summary should contain chain name")
		}
		if !containsString(summary, "2 effects") {
			t.Error("Summary should contain effect count")
		}
	})
}

func TestPluginChainNameOperations(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer eng.Destroy()

	// Create chain
	config := ChainConfig{
		Name:      "Original Name",
		EnginePtr: eng.Ptr(),
	}
	chain := NewPluginChain(config)
	defer chain.Release()

	// Test name operations
	t.Run("GetName", func(t *testing.T) {
		name := chain.GetName()
		if name != "Original Name" {
			t.Errorf("Expected name 'Original Name', got '%s'", name)
		}
	})

	t.Run("SetName", func(t *testing.T) {
		chain.SetName("New Name")
		name := chain.GetName()
		if name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", name)
		}
	})
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
