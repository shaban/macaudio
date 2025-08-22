package engine

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/shaban/macaudio/devices"
	"github.com/shaban/macaudio/plugins"
)

func TestPluginInstanceSeparation(t *testing.T) {
	// Get real devices
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}

	inputDevices := audioDevices.Inputs().Online()
	if len(inputDevices) == 0 {
		t.Skip("No online input devices found")
	}

	// Get a plugin with parameters to test with
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}

	// Find a plugin with writable parameters
	var testPluginInfo *plugins.PluginInfo
	for _, info := range pluginInfos {
		plugin, err := info.Introspect()
		if err != nil {
			continue
		}
		if len(plugin.GetWritableParameters()) > 0 {
			testPluginInfo = &info
			break
		}
	}

	if testPluginInfo == nil {
		t.Skip("No plugins with writable parameters found")
	}

	t.Logf("Testing with plugin: %s", testPluginInfo.Name)

	// Introspect the same plugin twice
	plugin1, err := testPluginInfo.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugin (first time): %v", err)
	}

	plugin2, err := testPluginInfo.Introspect()
	if err != nil {
		t.Fatalf("Failed to introspect plugin (second time): %v", err)
	}

	// Find a writable parameter to modify
	var targetParamIndex int = -1
	for i, param := range plugin1.Parameters {
		if param.IsWritable && param.MinValue != param.MaxValue {
			targetParamIndex = i
			break
		}
	}

	if targetParamIndex == -1 {
		t.Skip("No modifiable parameters found in plugin")
	}

	originalValue := plugin1.Parameters[targetParamIndex].CurrentValue
	t.Logf("Original parameter value: %f", originalValue)

	// Modify the parameter in plugin1
	newValue := plugin1.Parameters[targetParamIndex].MinValue +
		(plugin1.Parameters[targetParamIndex].MaxValue-plugin1.Parameters[targetParamIndex].MinValue)*0.5
	plugin1.Parameters[targetParamIndex].CurrentValue = newValue
	t.Logf("Modified plugin1 parameter to: %f", newValue)

	// Check if plugin2's parameter value is affected
	plugin2Value := plugin2.Parameters[targetParamIndex].CurrentValue
	t.Logf("Plugin2 parameter value: %f", plugin2Value)

	// Test: Are they independent?
	if plugin1.Parameters[targetParamIndex].CurrentValue == plugin2.Parameters[targetParamIndex].CurrentValue {
		t.Logf("⚠️  SHARED STATE: Both plugins have the same parameter value after modification")
		t.Logf("This means plugins share parameter state - need instance separation")
	} else {
		t.Logf("✅ INDEPENDENT STATE: Plugins have different parameter values")
		t.Logf("Plugin instance separation is working correctly")
	}

	// Create an engine with both plugins in a chain to test serialization
	engine := &Engine{}

	// Set up an input channel with a plugin chain containing both plugins
	channel := &Channel{
		Volume: 1.0,
		Pan:    0.0,
		InputOptions: &InputOptions{
			Device:       &inputDevices[0], // Use real device from devices.GetAudio()
			ChannelIndex: 0,
			PluginChain: &PluginChain{
				Plugins: []EnginePlugin{
					{
						IsInstalled: true,
						Plugin:      plugin1,
						Bypassed:    false,
					},
					{
						IsInstalled: true,
						Plugin:      plugin2,
						Bypassed:    false,
					},
				},
			},
		},
	}

	// Add channel to engine using append
	engine.Channels = append(engine.Channels, channel)

	// Serialize the engine
	jsonData, err := json.MarshalIndent(engine, "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize engine: %v", err)
	}

	t.Logf("Serialized engine with duplicate plugins:")

	// Parse back to verify parameter values are preserved
	var deserializedEngine Engine
	if err := json.Unmarshal(jsonData, &deserializedEngine); err != nil {
		t.Fatalf("Failed to deserialize engine: %v", err)
	}

	// Find the first channel with InputOptions and PluginChain
	var found bool
	for _, ch := range deserializedEngine.Channels {
		if ch != nil && ch.InputOptions != nil && ch.InputOptions.PluginChain != nil &&
			len(ch.InputOptions.PluginChain.Plugins) == 2 {
			plugin1Deserialized := ch.InputOptions.PluginChain.Plugins[0].Plugin
			plugin2Deserialized := ch.InputOptions.PluginChain.Plugins[1].Plugin

			if plugin1Deserialized == nil || plugin2Deserialized == nil {
				t.Fatalf("Plugins not properly deserialized")
			}

			if len(plugin1Deserialized.Parameters) <= targetParamIndex ||
				len(plugin2Deserialized.Parameters) <= targetParamIndex {
				t.Fatalf("Target parameter index %d not found in deserialized plugins", targetParamIndex)
			}

			param1Val := plugin1Deserialized.Parameters[targetParamIndex].CurrentValue
			param2Val := plugin2Deserialized.Parameters[targetParamIndex].CurrentValue

			t.Logf("After serialization roundtrip:")
			t.Logf("  Plugin 1 parameter: %f", param1Val)
			t.Logf("  Plugin 2 parameter: %f", param2Val)

			if param1Val != param2Val {
				t.Logf("✅ SUCCESS: Different parameter values preserved through serialization")
			} else {
				t.Logf("⚠️  WARNING: Parameter values are identical after serialization")
			}

			// Verify the modified value is preserved (allow for float precision differences)
			if math.Abs(float64(param1Val-newValue)) < 0.0001 {
				t.Logf("✅ Modified parameter value correctly preserved: %f", param1Val)
			} else {
				t.Errorf("❌ Modified parameter value lost: expected %f, got %f", newValue, param1Val)
			}

			// Verify the original value is preserved
			if math.Abs(float64(param2Val-originalValue)) < 0.0001 {
				t.Logf("✅ Original parameter value correctly preserved: %f", param2Val)
			} else {
				t.Errorf("❌ Original parameter value changed: expected %f, got %f", originalValue, param2Val)
			}

			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Deserialized engine does not contain expected channel with plugins")
	}
}
