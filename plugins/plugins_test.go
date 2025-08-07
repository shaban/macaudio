package plugins

import (
	"testing"
	"time"
)

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestJSONLogging(t *testing.T) {
	t.Log("Testing JSON logging functionality...")

	// Test that JSON logging can be enabled/disabled
	originalState := enableJSONLogging
	defer func() {
		enableJSONLogging = originalState
	}()

	// Test enabling JSON logging
	SetJSONLogging(true)
	if !enableJSONLogging {
		t.Error("JSON logging should be enabled")
	}

	// Test disabling JSON logging
	SetJSONLogging(false)
	if enableJSONLogging {
		t.Error("JSON logging should be disabled")
	}

	t.Log("✅ JSON logging test completed successfully!")
}

func TestList(t *testing.T) {
	t.Log("Testing quick AudioUnit plugin enumeration...")

	// Test quick scan
	pluginInfos, err := List()
	if err != nil {
		t.Fatalf("Failed to get plugin list: %v", err)
	}

	t.Logf("Quick scan found %d plugins total", len(pluginInfos))

	if len(pluginInfos) == 0 {
		t.Skip("No plugins available for testing")
	}

	// Test basic filtering
	applePlugins := pluginInfos.ByManufacturer("appl")
	t.Logf("Apple plugins: %d", len(applePlugins))

	effectPlugins := pluginInfos.ByType("aufx")
	t.Logf("Effect plugins (aufx): %d", len(effectPlugins))

	instrumentPlugins := pluginInfos.ByType("aumu")
	t.Logf("Instrument plugins (aumu): %d", len(instrumentPlugins))

	// Test name filtering
	compressorPlugins := pluginInfos.ByName("compressor")
	t.Logf("Plugins with 'compressor' in name: %d", len(compressorPlugins))

	// Test category filtering
	effectsByCategory := pluginInfos.ByCategory("Effect")
	t.Logf("Effect plugins (by category): %d", len(effectsByCategory))

	instrumentsByCategory := pluginInfos.ByCategory("Instrument")
	t.Logf("Instrument plugins (by category): %d", len(instrumentsByCategory))

	// Test plugin info structure
	for i, plugin := range pluginInfos[:5] { // Test first 5
		t.Logf("Plugin %d: %s (%s %s %s) [%s]",
			i+1, plugin.Name, plugin.Type, plugin.Subtype, plugin.ManufacturerID, plugin.Category)

		// Validate required fields
		if plugin.Name == "" {
			t.Errorf("Plugin %d has empty name", i+1)
		}
		if plugin.Type == "" {
			t.Errorf("Plugin %d has empty type", i+1)
		}
		if plugin.Subtype == "" {
			t.Errorf("Plugin %d has empty subtype", i+1)
		}
		if plugin.ManufacturerID == "" {
			t.Errorf("Plugin %d has empty manufacturer ID", i+1)
		}
		if plugin.Category == "" {
			t.Errorf("Plugin %d has empty category", i+1)
		}
	}

	t.Log("✅ Quick plugin list test completed successfully!")
}

func TestQuickScanPerformance(t *testing.T) {
	t.Log("Testing quick scan performance...")

	// Measure quick scan time
	start := time.Now()
	pluginInfos, err := List()
	quickScanDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Quick scan failed: %v", err)
	}

	t.Logf("Quick scan: %d plugins in %v", len(pluginInfos), quickScanDuration)

	// Quick scan should be much faster than full introspection
	if quickScanDuration > 5*time.Second {
		t.Errorf("Quick scan took too long: %v (should be under 5 seconds)", quickScanDuration)
	}

	t.Log("✅ Performance test completed successfully!")
}

func TestAllPlugins(t *testing.T) {
	// Test to ensure quick scan and full introspection deliver the same count
	t.Log("Testing all plugins: comparing quick scan vs full introspection...")

	// Enable JSON logging to capture the raw JSON output
	originalState := enableJSONLogging
	defer func() {
		enableJSONLogging = originalState
	}()
	SetJSONLogging(true)

	// First, do a quick scan to get the baseline count
	t.Log("Step 1: Quick scan to get plugin count...")
	pluginInfos, err := List()
	if err != nil {
		t.Fatalf("Quick scan failed: %v", err)
	}
	quickScanCount := len(pluginInfos)
	t.Logf("Quick scan found %d plugins", quickScanCount)

	if quickScanCount == 0 {
		t.Skip("No plugins found")
		return
	}

	// Then, do full introspection of all plugins
	t.Log("Step 2: Full introspection of all plugins...")
	plugins, err := Introspect("", "", "") // All plugins
	if err != nil {
		t.Fatalf("Full introspection failed: %v", err)
	}
	introspectionCount := len(plugins)
	t.Logf("Full introspection found %d plugins", introspectionCount)

	// Compare counts - they should match
	if quickScanCount != introspectionCount {
		t.Errorf("Plugin count mismatch: Quick scan found %d, Full introspection found %d",
			quickScanCount, introspectionCount)
	} else {
		t.Logf("✅ Success! Both methods found the same number of plugins: %d", quickScanCount)
	}

	// Validate that introspected plugins have parameter data
	pluginsWithParams := 0
	totalParams := 0
	for _, plugin := range plugins {
		if len(plugin.Parameters) > 0 {
			pluginsWithParams++
			totalParams += len(plugin.Parameters)
		}
	}

	t.Logf("Plugins with parameters: %d/%d", pluginsWithParams, introspectionCount)
	t.Logf("Total parameters across all plugins: %d", totalParams)

	// Log some sample plugins for verification
	sampleCount := min(3, len(plugins))
	t.Logf("Sample of introspected plugins:")
	for i := 0; i < sampleCount; i++ {
		plugin := plugins[i]
		t.Logf("  %d. %s - %d parameters", i+1, plugin.Name, len(plugin.Parameters))
	}

	t.Log("✅ All plugins test completed successfully!")
}

/*func TestIntrospectFromInfo(t *testing.T) {
	t.Log("Testing IntrospectFromInfo helper function...")

	// Get the list of plugins first
	plugins, err := List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	if len(plugins) == 0 {
		t.Skip("No plugins found")
		return
	}

	// Find Neural DSP plugin if available, otherwise use the first plugin
	var testPlugin PluginInfo
	found := false
	for _, plugin := range plugins {
		if plugin.ManufacturerID == "NDSP" {
			testPlugin = plugin
			found = true
			t.Logf("Found Neural DSP plugin: %s", plugin.Name)
			break
		}
	}

	if !found {
		testPlugin = plugins[0]
		t.Logf("Neural DSP not found, using: %s (%s:%s:%s)",
			testPlugin.Name, testPlugin.Type, testPlugin.Subtype, testPlugin.ManufacturerID)
	}

	// Test the helper function
	result, err := IntrospectFromInfo(testPlugin)
	if err != nil {
		t.Fatalf("IntrospectFromInfo failed: %v", err)
	}

	t.Logf("✅ Helper function worked! Plugin: %s, Parameters: %d",
		result.Name, len(result.Parameters))

	// Compare with direct function call
	directResult, err := Introspect(testPlugin.Type, testPlugin.Subtype, testPlugin.ManufacturerID)
	if err != nil {
		t.Fatalf("Direct Introspect failed: %v", err)
	}

	// Compare key fields (can't compare structs with slices directly)
	if result.Name == directResult.Name &&
		result.Category == directResult.Category &&
		len(result.Parameters) == len(directResult.Parameters) {
		t.Log("✅ Helper function returns equivalent result to direct function")
	} else {
		t.Error("Helper function and direct function returned different results")
		t.Logf("Helper: %s, %s, %d params", result.Name, result.Category, len(result.Parameters))
		t.Logf("Direct: %s, %s, %d params", directResult.Name, directResult.Category, len(directResult.Parameters))
	}
}*/
