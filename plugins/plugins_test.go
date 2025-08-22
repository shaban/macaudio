package plugins

import (
	"os"
	"path/filepath"
	"strings"
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

	// Test a known suite case if present and verify suite vs single behavior
	found := false
	for _, info := range pluginInfos {
		if info.Name == "MS ADPCM Decoder" {
			found = true
			t.Logf("Found candidate plugin suite triplet: %s:%s:%s (Name=%q)",
				info.Type, info.Subtype, info.ManufacturerID, info.Name)

			// Suite: expect one or more entries
			suite, err := info.IntrospectSuite()
			if err != nil {
				t.Fatalf("IntrospectSuite error: %v", err)
			}
			if len(suite) == 0 {
				t.Fatalf("IntrospectSuite returned 0 plugins for triplet %s:%s:%s",
					info.Type, info.Subtype, info.ManufacturerID)
			}
			t.Logf("IntrospectSuite returned %d plugin(s)", len(suite))

			// Single: exact name must return exactly one
			single, err := info.Introspect()
			if err != nil {
				t.Fatalf("Introspect(single) error: %v", err)
			}
			if single == nil || single.Name != info.Name {
				t.Fatalf("Introspect(single) returned unexpected plugin: got %v", single)
			}
			t.Logf("Introspect(single) returned %q with %d parameters", single.Name, len(single.Parameters))

			// Not found: mutate name to something impossible
			bogus := info
			bogus.Name = info.Name + "__DOES_NOT_EXIST__"
			if _, err := bogus.Introspect(); err == nil {
				t.Fatalf("Expected error for non-existent plugin name, got nil")
			} else {
				t.Logf("Introspect(non-existent) correctly errored: %v", err)
			}
			break
		}
	}
	if !found {
		t.Logf("MS ADPCM Decoder plugin not found in %d total plugins (skipping suite/single checks)", len(pluginInfos))
		for _, info := range pluginInfos {
			if strings.Contains(strings.ToLower(info.Name), "adpcm") ||
				strings.Contains(strings.ToLower(info.Name), "decoder") {
				t.Logf("   - %s (%s:%s:%s)", info.Name, info.Type, info.Subtype, info.ManufacturerID)
			}
		}
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

	// Keep JSON logging disabled to avoid noisy test output
	originalState := enableJSONLogging
	defer func() { enableJSONLogging = originalState }()
	SetJSONLogging(false)

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

	// Then, do full introspection of all plugins (single-plugin per info)
	t.Log("Step 2: Full introspection (single) for all plugins...")
	plugins, err := pluginInfos.Introspect()
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

// TestFullScanJSONToFile performs a full scan and introspection, logging JSON to a file for manual inspection.
func TestFullScanJSONToFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "plugins_full_scan.jsonl")
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("create log file: %v", err)
	}
	defer f.Close()

	// Redirect JSON logs to file and enable logging
	prev := enableJSONLogging
	SetJSONLogWriter(f)
	SetJSONLogging(true)
	defer func() {
		SetJSONLogWriter(nil)
		SetJSONLogging(prev)
	}()

	t.Logf("Writing JSON logs to %s", outPath)

	// Step 1: Quick scan (will emit a JSON record labeled QuickScan)
	infos, err := List()
	if err != nil {
		t.Fatalf("quick scan failed: %v", err)
	}
	t.Logf("Quick scan found %d plugins", len(infos))

	// Step 2: Introspect each plugin (each emits a JSON record labeled Introspect[...])
	// Keep runtime reasonable but thorough; we’ll do all for accuracy.
	_, err = PluginInfos(infos).Introspect()
	if err != nil {
		t.Fatalf("full introspection failed: %v", err)
	}

	// Flush to disk
	if err := f.Sync(); err != nil {
		t.Fatalf("sync log file: %v", err)
	}

	fi, statErr := f.Stat()
	if statErr != nil {
		t.Fatalf("stat log file: %v", statErr)
	}
	if fi.Size() == 0 {
		t.Fatalf("expected JSON logs in %s, got empty file", outPath)
	}

	t.Logf("Full scan JSON written (%d bytes)", fi.Size())
}
