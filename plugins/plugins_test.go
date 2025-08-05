package plugins

import (
	"testing"
	"time"
)

func TestGetPlugins(t *testing.T) {
	t.Log("Testing AU plugin enumeration...")

	// Test basic plugin enumeration
	plugins, err := GetPlugins()
	if err != nil {
		t.Fatalf("Failed to get plugins: %v", err)
	}

	t.Logf("Found %d plugins total", len(plugins))

	if len(plugins) == 0 {
		t.Log("⚠️  No plugins found - this might be expected on some systems")
		return
	}

	// Test that we have some plugins with parameters
	pluginsWithParams := plugins.WithParameters()
	t.Logf("Plugins with parameters: %d", len(pluginsWithParams))

	if len(pluginsWithParams) == 0 {
		t.Log("⚠️  No plugins with parameters found")
		return
	}

	// Test basic filtering
	applePlugins := plugins.ByManufacturer("appl")
	t.Logf("Apple plugins: %d", len(applePlugins))

	effectPlugins := plugins.ByType("aufx")
	t.Logf("Effect plugins (aufx): %d", len(effectPlugins))

	indexedPlugins := plugins.WithIndexedParameters()
	t.Logf("Plugins with indexed parameters: %d", len(indexedPlugins))

	t.Log("✅ Plugin enumeration test completed successfully!")
}

func TestPluginStructure(t *testing.T) {
	t.Log("Testing plugin data structure...")

	plugins, err := GetPlugins()
	if err != nil {
		t.Fatalf("Failed to get plugins: %v", err)
	}

	if len(plugins) == 0 {
		t.Skip("No plugins available for structure testing")
	}

	// Find a plugin with parameters for detailed testing
	var testPlugin *Plugin
	for _, plugin := range plugins {
		if len(plugin.Parameters) > 0 {
			testPlugin = &plugin
			break
		}
	}

	if testPlugin == nil {
		t.Skip("No plugins with parameters available for structure testing")
	}

	t.Logf("Testing plugin: %s", testPlugin.Name)

	// Test required fields
	if testPlugin.Name == "" {
		t.Error("Plugin name is empty")
	}
	if testPlugin.ManufacturerID == "" {
		t.Error("Plugin manufacturer ID is empty")
	}
	if testPlugin.Type == "" {
		t.Error("Plugin type is empty")
	}
	if testPlugin.Subtype == "" {
		t.Error("Plugin subtype is empty")
	}

	// Test parameter structure
	if len(testPlugin.Parameters) > 0 {
		param := testPlugin.Parameters[0]
		t.Logf("Testing parameter: %s", param.DisplayName)

		if param.DisplayName == "" {
			t.Error("Parameter display name is empty")
		}
		if param.Identifier == "" {
			t.Error("Parameter identifier is empty")
		}
		if param.Unit == "" {
			t.Error("Parameter unit is empty")
		}

		// Test parameter methods
		writableParams := testPlugin.GetWritableParameters()
		t.Logf("Writable parameters: %d", len(writableParams))

		indexedParams := testPlugin.GetIndexedParameters()
		t.Logf("Indexed parameters: %d", len(indexedParams))

		rampableParams := testPlugin.GetRampableParameters()
		t.Logf("Rampable parameters: %d", len(rampableParams))
	}

	t.Log("✅ Plugin structure test completed successfully!")
}

func TestPluginFiltering(t *testing.T) {
	t.Log("Testing plugin filtering methods...")

	plugins, err := GetPlugins()
	if err != nil {
		t.Fatalf("Failed to get plugins: %v", err)
	}

	if len(plugins) == 0 {
		t.Skip("No plugins available for filtering tests")
	}

	// Test manufacturer filtering
	applePlugins := plugins.ByManufacturer("appl")
	t.Logf("Apple plugins: %d", len(applePlugins))

	// Verify all returned plugins are actually from Apple
	for _, plugin := range applePlugins {
		if plugin.ManufacturerID != "appl" {
			t.Errorf("Expected Apple plugin, got manufacturer: %s", plugin.ManufacturerID)
		}
	}

	// Test type filtering
	effectPlugins := plugins.ByType("aufx")
	t.Logf("Effect plugins: %d", len(effectPlugins))

	for _, plugin := range effectPlugins {
		if plugin.Type != "aufx" {
			t.Errorf("Expected effect plugin, got type: %s", plugin.Type)
		}
	}

	// Test parameter filtering
	pluginsWithParams := plugins.WithParameters()
	for _, plugin := range pluginsWithParams {
		if len(plugin.Parameters) == 0 {
			t.Errorf("Plugin %s should have parameters but has none", plugin.Name)
		}
	}

	// Test indexed parameter filtering
	indexedPlugins := plugins.WithIndexedParameters()
	for _, plugin := range indexedPlugins {
		hasIndexed := false
		for _, param := range plugin.Parameters {
			if len(param.IndexedValues) > 0 {
				hasIndexed = true
				break
			}
		}
		if !hasIndexed {
			t.Errorf("Plugin %s should have indexed parameters but has none", plugin.Name)
		}
	}

	t.Log("✅ Plugin filtering test completed successfully!")
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

func TestGetPluginsWithTimeout(t *testing.T) {
	t.Log("Testing plugin enumeration with custom timeout...")

	// Test with a short timeout (5 seconds)
	plugins, err := GetPluginsWithTimeout(5.0)
	if err != nil {
		t.Logf("Short timeout failed (expected): %v", err)
		// This might fail due to timeout, which is acceptable
	} else {
		t.Logf("Found %d plugins with 5-second timeout", len(plugins))
	}

	// Test with a reasonable timeout (30 seconds)
	plugins, err = GetPluginsWithTimeout(30.0)
	if err != nil {
		t.Fatalf("Failed to get plugins with 30-second timeout: %v", err)
	}

	t.Logf("Found %d plugins with 30-second timeout", len(plugins))

	t.Log("✅ Timeout test completed successfully!")
}

func TestGetPluginList(t *testing.T) {
	t.Log("Testing quick AudioUnit plugin enumeration...")

	// Test quick scan
	pluginInfos, err := GetPluginList()
	if err != nil {
		t.Fatalf("Failed to get plugin list: %v", err)
	}

	t.Logf("Quick scan found %d plugins total", len(pluginInfos))

	if len(pluginInfos) == 0 {
		t.Log("⚠️  No plugins found - this might be expected on some systems")
		return
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

	// Test data integrity
	for i, plugin := range pluginInfos {
		if i >= 5 { // Just test first 5 for brevity
			break
		}

		t.Logf("Plugin %d: %s (%s %s %s)", i+1, plugin.Name, plugin.Type, plugin.Subtype, plugin.ManufacturerID)

		// Test required fields
		if plugin.Name == "" {
			t.Errorf("Plugin %d has empty name", i+1)
		}
		if plugin.ManufacturerID == "" {
			t.Errorf("Plugin %d has empty manufacturer ID", i+1)
		}
		if plugin.Type == "" {
			t.Errorf("Plugin %d has empty type", i+1)
		}
		if plugin.Subtype == "" {
			t.Errorf("Plugin %d has empty subtype", i+1)
		}
	}

	t.Log("✅ Quick plugin list test completed successfully!")
}

func TestQuickScanPerformance(t *testing.T) {
	t.Log("Testing quick scan performance...")

	// Measure quick scan time
	start := time.Now()
	pluginInfos, err := GetPluginList()
	quickScanDuration := time.Since(start)

	if err != nil {
		t.Fatalf("Quick scan failed: %v", err)
	}

	t.Logf("Quick scan: %d plugins in %v", len(pluginInfos), quickScanDuration)

	// Quick scan should be much faster than full introspection
	if quickScanDuration > 5*time.Second {
		t.Errorf("Quick scan took too long: %v (should be under 5 seconds)", quickScanDuration)
	}

	// Compare with full introspection (just measure, don't wait for completion)
	start = time.Now()
	fullPlugins, err := GetPluginsWithTimeout(2.0) // Short timeout
	shortIntrospectionDuration := time.Since(start)

	t.Logf("Partial introspection: %d plugins in %v", len(fullPlugins), shortIntrospectionDuration)

	// Quick scan should be much faster
	if quickScanDuration >= shortIntrospectionDuration {
		t.Logf("⚠️  Quick scan (%v) not significantly faster than introspection (%v)",
			quickScanDuration, shortIntrospectionDuration)
	} else {
		speedup := float64(shortIntrospectionDuration) / float64(quickScanDuration)
		t.Logf("Quick scan is %.1fx faster than introspection", speedup)
	}

	t.Log("✅ Performance test completed successfully!")
}
