package plugins

import (
	"testing"
)

func TestMethodBasedAPI(t *testing.T) {
	t.Log("Testing method-based introspection API...")

	// Get the list of all plugins first
	pluginInfos, err := List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	if len(pluginInfos) == 0 {
		t.Skip("No plugins available for testing")
	}

	t.Logf("Found %d total plugins", len(pluginInfos))

	// Test 1: Single plugin introspection using method
	firstPlugin := pluginInfos[0]
	t.Logf("Testing single plugin introspection: %s", firstPlugin.Name)

	plugin, err := firstPlugin.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect single plugin: %v", err)
	}

	t.Logf("✅ Single plugin introspected: %s with %d parameters", plugin.Name, len(plugin.Parameters))

	// Verify the plugin data matches
	if plugin.Name != firstPlugin.Name {
		t.Errorf("Plugin name mismatch: expected %s, got %s", firstPlugin.Name, plugin.Name)
	}
	if plugin.Type != firstPlugin.Type {
		t.Errorf("Plugin type mismatch: expected %s, got %s", firstPlugin.Type, plugin.Type)
	}

	// Test 2: Multiple plugin introspection using method
	// Take a smaller subset for testing
	testPlugins := pluginInfos[:min(3, len(pluginInfos))]
	t.Logf("Testing multiple plugin introspection with %d plugins", len(testPlugins))

	plugins, err := testPlugins.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect multiple plugins: %v", err)
	}

	if len(plugins) != len(testPlugins) {
		t.Errorf("Plugin count mismatch: expected %d, got %d", len(testPlugins), len(plugins))
	}

	t.Logf("✅ Multiple plugins introspected: %d plugins", len(plugins))

	// Test 3: Compare method-based with function-based
	t.Log("Comparing method-based API with function-based API...")

	// Use function-based approach
	funcResults, err := Introspect(firstPlugin.Type, firstPlugin.Subtype, firstPlugin.ManufacturerID)
	if err != nil {
		t.Fatalf("Function-based introspection failed: %v", err)
	}

	// Should get same result
	if len(funcResults) != 1 {
		t.Errorf("Expected 1 plugin from function call, got %d", len(funcResults))
	} else {
		funcPlugin := funcResults[0]
		if funcPlugin.Name != plugin.Name || funcPlugin.Type != plugin.Type {
			t.Error("Method-based and function-based results don't match")
		} else {
			t.Log("✅ Method-based and function-based APIs return consistent results")
		}
	}

	// Test 4: Test filtering with methods
	t.Log("Testing filtering combined with introspection...")

	applePlugins := pluginInfos.ByManufacturer("appl")
	if len(applePlugins) > 0 {
		// Take first Apple plugin
		applePlugin, err := applePlugins[0].Introspect()
		if err != nil {
			t.Fatalf("Failed to introspect Apple plugin: %v", err)
		}
		t.Logf("✅ Apple plugin introspected: %s", applePlugin.Name)
	}

	t.Log("✅ Method-based API test completed successfully!")
}
