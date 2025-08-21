package engine

import (
	"testing"

	"github.com/shaban/macaudio/plugins"
)

func TestPluginParameterAPI(t *testing.T) {
	// Get a plugin with writable parameters
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	var testPlugin *plugins.Plugin
	for _, info := range pluginInfos {
		plugin, err := info.Introspect()
		if err != nil {
			continue
		}
		if len(plugin.GetWritableParameters()) > 0 {
			testPlugin = plugin
			break
		}
	}

	if testPlugin == nil {
		t.Skip("No plugins with writable parameters found")
	}

	// Create a plugin chain and add the plugin
	chain := NewPluginChain()
	enginePlugin := NewEnginePlugin(testPlugin)

	if err := chain.AddPlugin(*enginePlugin); err != nil {
		t.Fatalf("Failed to add plugin: %v", err)
	}

	// Test parameter management
	writableParams := testPlugin.GetWritableParameters()
	if len(writableParams) > 0 {
		param := writableParams[0]
		t.Logf("Testing parameter: %s (identifier: %s)", param.DisplayName, param.Identifier)
		t.Logf("  Range: %f to %f, Default: %f, Current: %f",
			param.MinValue, param.MaxValue, param.DefaultValue, param.CurrentValue)

		// Test setting parameter by identifier
		newValue := param.MinValue + (param.MaxValue-param.MinValue)*0.7
		err := chain.SetPluginParameter(0, param.Identifier, newValue)
		if err != nil {
			t.Fatalf("Failed to set parameter by identifier: %v", err)
		}

		// Test getting parameter
		retrievedValue, err := chain.GetPluginParameter(0, param.Identifier)
		if err != nil {
			t.Fatalf("Failed to get parameter: %v", err)
		}

		if retrievedValue != newValue {
			t.Errorf("Parameter value mismatch: set %f, got %f", newValue, retrievedValue)
		}

		t.Logf("✅ Parameter set/get by identifier works: %f", retrievedValue)

		// Test setting parameter by display name
		newValue2 := param.MinValue + (param.MaxValue-param.MinValue)*0.2
		err = chain.SetPluginParameter(0, param.DisplayName, newValue2)
		if err != nil {
			t.Fatalf("Failed to set parameter by display name: %v", err)
		}

		retrievedValue2, err := chain.GetPluginParameter(0, param.DisplayName)
		if err != nil {
			t.Fatalf("Failed to get parameter by display name: %v", err)
		}

		if retrievedValue2 != newValue2 {
			t.Errorf("Parameter value mismatch: set %f, got %f", newValue2, retrievedValue2)
		}

		t.Logf("✅ Parameter set/get by display name works: %f", retrievedValue2)

		// Test parameter bounds checking
		err = chain.SetPluginParameter(0, param.Identifier, param.MaxValue+1.0)
		if err == nil {
			t.Error("Expected error when setting parameter above max value")
		} else {
			t.Logf("✅ Bounds checking works: %v", err)
		}
	}

	// Test parameter name enumeration
	names, err := chain.GetPluginParameterNames(0)
	if err != nil {
		t.Fatalf("Failed to get parameter names: %v", err)
	}

	t.Logf("Plugin has %d parameters:", len(names))
	for i, name := range names {
		if i < 5 { // Only log first 5 to avoid spam
			t.Logf("  [%d] %s", i, name)
		}
	}

	if len(names) != len(testPlugin.Parameters) {
		t.Errorf("Parameter names count mismatch: expected %d, got %d",
			len(testPlugin.Parameters), len(names))
	}
}
