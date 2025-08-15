package unit

import (
	"testing"

	"github.com/shaban/macaudio/plugins"
)

// TestMain sets up and tears down test resources
func TestMain(m *testing.M) {
	// Enable JSON logging for plugin discovery
	plugins.SetJSONLogging(true)

	// Run tests
	m.Run()
}

// TestCreateEffectFromPlugin tests effect creation using plugins package introspection
func TestCreateEffectFromPlugin(t *testing.T) {
	t.Log("Testing effect creation using plugins package...")

	// Get Apple plugins
	pluginList, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	applePlugins := pluginList.ByManufacturer("appl")
	if len(applePlugins) == 0 {
		t.Skip("No Apple plugins found")
	}

	// Find an effect plugin
	var effectPluginInfo plugins.PluginInfo
	found := false
	for _, plugin := range applePlugins {
		if plugin.Category == "Effect" {
			effectPluginInfo = plugin
			found = true
			break
		}
	}

	if !found {
		t.Skip("No Apple effect plugins found")
	}

	t.Logf("Found effect plugin: %s (Type: %s, Subtype: %s)",
		effectPluginInfo.Name, effectPluginInfo.Type, effectPluginInfo.Subtype)

	// Get full plugin details with parameters
	fullPlugin, err := effectPluginInfo.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugin: %v", err)
	}
	t.Logf("Plugin has %d parameters", len(fullPlugin.Parameters))

	// Create effect using the new API
	effect, err := CreateEffect(fullPlugin)
	if err != nil {
		t.Fatalf("Failed to create effect: %v", err)
	}
	defer effect.Release()

	// Verify effect properties
	plugin := effect.GetPlugin()
	if plugin.Name != fullPlugin.Name {
		t.Errorf("Expected plugin name %s, got %s", fullPlugin.Name, plugin.Name)
	}

	ptr := effect.Ptr()
	if ptr == nil {
		t.Error("Effect pointer is nil")
	}

	t.Log("✅ Effect created successfully")
}

// TestParameterControl tests parameter control using plugins package parameter metadata
func TestParameterControl(t *testing.T) {
	t.Log("Testing parameter control using plugins package...")

	// Get Apple plugins
	pluginList, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	applePlugins := pluginList.ByManufacturer("appl")
	if len(applePlugins) == 0 {
		t.Skip("No Apple plugins found")
	}

	// Find reverb plugin specifically for parameter testing
	var reverbPluginInfo plugins.PluginInfo
	found := false
	for _, plugin := range applePlugins {
		if plugin.Category == "Effect" && (plugin.Name == "AUMatrixReverb" || plugin.Name == "Reverb") {
			reverbPluginInfo = plugin
			found = true
			break
		}
	}

	if !found {
		// Fall back to any effect plugin
		for _, plugin := range applePlugins {
			if plugin.Category == "Effect" {
				reverbPluginInfo = plugin
				found = true
				break
			}
		}
	}

	if !found {
		t.Skip("No Apple effect plugins found")
	}

	// Get full plugin details with parameters
	fullPlugin, err := reverbPluginInfo.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugin: %v", err)
	}
	if len(fullPlugin.Parameters) == 0 {
		t.Skip("Plugin has no parameters")
	}

	t.Logf("Testing with plugin: %s (%d parameters)", fullPlugin.Name, len(fullPlugin.Parameters))

	// Create effect
	effect, err := CreateEffect(fullPlugin)
	if err != nil {
		t.Fatalf("Failed to create effect: %v", err)
	}
	defer effect.Release()

	// Test parameter control with the first parameter
	param := fullPlugin.Parameters[0]
	t.Logf("Testing parameter: %s (Address: %d, Range: %.3f-%.3f, Default: %.3f)",
		param.DisplayName, param.Address, param.MinValue, param.MaxValue, param.DefaultValue)

	// Calculate test value (mid-range)
	testValue := param.MinValue + (param.MaxValue-param.MinValue)*0.5

	// Set parameter
	err = effect.SetParameter(param, testValue)
	if err != nil {
		t.Errorf("Failed to set parameter: %v", err)
	} else {
		t.Logf("✅ Successfully set parameter to %.3f", testValue)
	}

	// Get parameter back
	gotValue, err := effect.GetParameter(param)
	if err != nil {
		t.Errorf("Failed to get parameter: %v", err)
	} else {
		t.Logf("✅ Got parameter value: %.3f", gotValue)
	}
}

// TestEffectLifecycle tests effect lifecycle (create and release)
func TestEffectLifecycle(t *testing.T) {
	t.Log("Testing effect lifecycle...")

	// Get Apple plugins
	pluginList, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	applePlugins := pluginList.ByManufacturer("appl")
	if len(applePlugins) == 0 {
		t.Skip("No Apple plugins found")
	}

	// Find an effect plugin
	var effectPluginInfo plugins.PluginInfo
	found := false
	for _, plugin := range applePlugins {
		if plugin.Category == "Effect" {
			effectPluginInfo = plugin
			found = true
			break
		}
	}

	if !found {
		t.Skip("No Apple effect plugins found")
	}

	// Get full plugin details
	fullPlugin, err := effectPluginInfo.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugin: %v", err)
	}

	// Create effect
	effect, err := CreateEffect(fullPlugin)
	if err != nil {
		t.Fatalf("Failed to create effect: %v", err)
	}

	// Verify it's working
	if effect.Ptr() == nil {
		t.Fatal("Effect pointer is nil")
	}

	// Release effect
	effect.Release()

	// Verify operations fail after release
	if len(fullPlugin.Parameters) > 0 {
		param := fullPlugin.Parameters[0]
		err = effect.SetParameter(param, 0.5)
		if err == nil {
			t.Error("Expected error when setting parameter on released effect")
		} else {
			t.Logf("✅ Correctly got error after release: %v", err)
		}
	}

	t.Log("✅ Effect lifecycle test completed")
}

func TestEffectBypass(t *testing.T) {
	// Discover Apple AU effects
	pluginList, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}
	effects := pluginList.ByManufacturer("appl").ByType("aufx")
	if len(effects) == 0 {
		t.Skip("No Apple AU effects found for bypass test")
	}

	// Introspect first effect
	plg, err := effects[0].Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugin: %v", err)
	}

	eff, err := CreateEffect(plg)
	if err != nil {
		t.Fatalf("Failed to create effect: %v", err)
	}
	defer eff.Release()

	// Toggle bypass on
	if err := eff.SetBypass(true); err != nil {
		t.Skipf("Bypass not supported or failed to set: %v", err)
	}

	isOn, err := eff.IsBypassed()
	if err != nil {
		t.Skipf("Failed to query bypass state: %v", err)
	}
	if !isOn {
		t.Errorf("Expected bypass ON after SetBypass(true)")
	}

	// Toggle bypass off
	if err := eff.SetBypass(false); err != nil {
		t.Errorf("Failed to set bypass off: %v", err)
	}
	isOn, err = eff.IsBypassed()
	if err != nil {
		t.Errorf("Failed to query bypass state: %v", err)
	}
	if isOn {
		t.Errorf("Expected bypass OFF after SetBypass(false)")
	}
}
