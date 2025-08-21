package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/shaban/macaudio/devices"
	"github.com/shaban/macaudio/plugins"
)

// TestEngineSerializationRoundtrip tests complete serialization roundtrip with real devices and plugins
func TestEngineSerializationRoundtrip(t *testing.T) {
	// Create engine with real device and plugin data
	originalEngine := createEngineWithRealData(t)

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(originalEngine, "", "  ")
	if err != nil {
		t.Fatalf("Failed to serialize engine: %v", err)
	}

	// Output JSON to stdout if environment variable is set (pure JSON only)
	if dumpJSON := os.Getenv("MACAUDIO_DUMP_JSON"); dumpJSON != "" {
		fmt.Println(string(jsonData))
		return // Skip further logging/testing when dumping JSON
	}

	t.Logf("Serialized JSON size: %d bytes", len(jsonData))

	// Log some details about what we're testing
	logEngineDetails(t, originalEngine)

	// Deserialize from JSON
	var deserializedEngine Engine
	if err := json.Unmarshal(jsonData, &deserializedEngine); err != nil {
		t.Fatalf("Failed to deserialize engine: %v", err)
	}

	// Verify roundtrip accuracy
	compareEngines(t, originalEngine, &deserializedEngine)
}

// createEngineWithRealData creates an engine populated with real system data
func createEngineWithRealData(t *testing.T) *Engine {
	engine := &Engine{
		MasterVolume: 0.85,
		SampleRate:   48000,
	}

	// Get real audio devices
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Logf("Warning: Could not get audio devices: %v", err)
		return engine // Return basic engine if devices unavailable
	}

	// Get real plugins
	pluginInfos, err := plugins.List()
	if err != nil {
		t.Logf("Warning: Could not get plugins: %v", err)
		return engine // Return basic engine if plugins unavailable
	}

	// Set up a few channels with real data
	setupChannelWithInputDevice(engine, 0, audioDevices.Inputs().Online(), pluginInfos.ByType("aufx"))
	setupChannelWithPlaybackFile(engine, 1)
	setupChannelWithInputDevice(engine, 2, audioDevices.Inputs().Online(), pluginInfos.ByType("aumu"))
	setupChannelWithInputDevice(engine, 3, audioDevices.Inputs().Online(), pluginInfos.ByManufacturer("NDSP"))

	return engine
}

// setupChannelWithInputDevice sets up an input channel with real device and plugins
func setupChannelWithInputDevice(engine *Engine, channelIndex int, inputDevices devices.AudioDevices, effectPlugins plugins.PluginInfos) {
	if len(inputDevices) == 0 {
		return // Skip if no input devices
	}

	// Use first available input device
	device := inputDevices[0]

	channel := &Channel{
		BusIndex: channelIndex,
		Volume:   0.75,
		Pan:      0.0,
		InputOptions: &InputOptions{
			Device:       &device, // Embed complete device info
			ChannelIndex: 0,
		},
	}

	// Add plugin chain if effects available
	if len(effectPlugins) > 0 {
		pluginChain := &PluginChain{}

		// Add first available effect plugin
		if plugin, err := effectPlugins[0].Introspect(); err == nil {
			enginePlugin := EnginePlugin{
				IsInstalled: true,
				Plugin:      plugin, // Complete introspected plugin with parameter schema
				Bypassed:    false,
			}

			// Set some realistic current values using the introspected parameter metadata
			for i := range plugin.Parameters {
				param := &plugin.Parameters[i] // Get pointer to modify in place
				if param.IsWritable && i < 5 { // Limit to 5 parameters for testing
					// Set a reasonable value within the parameter's range
					if param.CurrentValue == 0 && param.DefaultValue != 0 {
						param.CurrentValue = param.DefaultValue
					} else if param.CurrentValue == 0 {
						// Calculate a reasonable value within the parameter's range
						param.CurrentValue = param.MinValue + (param.MaxValue-param.MinValue)*0.3
					}
				}
			}

			pluginChain.Plugins = append(pluginChain.Plugins, enginePlugin)
		}

		channel.InputOptions.PluginChain = pluginChain
	}

	engine.Channels[channelIndex] = channel
}

// setupChannelWithPlaybackFile sets up a playback channel (no plugins per MVP)
func setupChannelWithPlaybackFile(engine *Engine, channelIndex int) {
	channel := &Channel{
		BusIndex: channelIndex,
		Volume:   0.65,
		Pan:      -0.3,
		PlaybackOptions: &PlaybackOptions{
			FilePath: "/System/Library/Sounds/Ping.aiff", // System sound that should exist
			Rate:     1.0,
			Pitch:    0.0,
		},
	}

	// Note: Per MVP, plugin chains are input channels only
	engine.Channels[channelIndex] = channel
}

// compareEngines performs deep comparison between original and deserialized engines
func compareEngines(t *testing.T, original, deserialized *Engine) {
	if original.MasterVolume != deserialized.MasterVolume {
		t.Errorf("MasterVolume mismatch: original=%.3f, deserialized=%.3f",
			original.MasterVolume, deserialized.MasterVolume)
	}

	if original.SampleRate != deserialized.SampleRate {
		t.Errorf("SampleRate mismatch: original=%d, deserialized=%d",
			original.SampleRate, deserialized.SampleRate)
	}

	// Compare channels
	for i := 0; i < 8; i++ {
		origChannel := original.Channels[i]
		deserChannel := deserialized.Channels[i]

		if (origChannel == nil) != (deserChannel == nil) {
			t.Errorf("Channel[%d] nil mismatch: original=%v, deserialized=%v",
				i, origChannel == nil, deserChannel == nil)
			continue
		}

		if origChannel == nil {
			continue // Both nil, OK
		}

		compareChannels(t, i, origChannel, deserChannel)
	}
}

// compareChannels compares individual channel data
func compareChannels(t *testing.T, index int, original, deserialized *Channel) {
	// Check channel type by presence of options
	originalIsInput := original.IsInput()
	deserializedIsInput := deserialized.IsInput()

	if originalIsInput != deserializedIsInput {
		t.Errorf("Channel[%d] type mismatch: original is input=%t, deserialized is input=%t",
			index, originalIsInput, deserializedIsInput)
	}

	if original.BusIndex != deserialized.BusIndex {
		t.Errorf("Channel[%d] BusIndex mismatch: original=%d, deserialized=%d",
			index, original.BusIndex, deserialized.BusIndex)
	}

	if original.Volume != deserialized.Volume {
		t.Errorf("Channel[%d] Volume mismatch: original=%.3f, deserialized=%.3f",
			index, original.Volume, deserialized.Volume)
	}

	if original.Pan != deserialized.Pan {
		t.Errorf("Channel[%d] Pan mismatch: original=%.3f, deserialized=%.3f",
			index, original.Pan, deserialized.Pan)
	}

	// Compare InputOptions
	if (original.InputOptions == nil) != (deserialized.InputOptions == nil) {
		t.Errorf("Channel[%d] InputOptions nil mismatch", index)
		return
	}

	if original.InputOptions != nil {
		if original.InputOptions.Device != nil && deserialized.InputOptions.Device != nil {
			if original.InputOptions.Device.UID != deserialized.InputOptions.Device.UID {
				t.Errorf("Channel[%d] InputOptions.Device.UID mismatch: original=%s, deserialized=%s",
					index, original.InputOptions.Device.UID, deserialized.InputOptions.Device.UID)
			}
			if original.InputOptions.Device.Name != deserialized.InputOptions.Device.Name {
				t.Errorf("Channel[%d] InputOptions.Device.Name mismatch: original=%s, deserialized=%s",
					index, original.InputOptions.Device.Name, deserialized.InputOptions.Device.Name)
			}
		} else if (original.InputOptions.Device == nil) != (deserialized.InputOptions.Device == nil) {
			t.Errorf("Channel[%d] InputOptions.Device nil mismatch", index)
		}

		if original.InputOptions.ChannelIndex != deserialized.InputOptions.ChannelIndex {
			t.Errorf("Channel[%d] InputOptions.ChannelIndex mismatch: original=%d, deserialized=%d",
				index, original.InputOptions.ChannelIndex, deserialized.InputOptions.ChannelIndex)
		}

		// Compare plugin chains
		comparePluginChains(t, index, original.InputOptions.PluginChain, deserialized.InputOptions.PluginChain)
	}

	// Compare PlaybackOptions
	if (original.PlaybackOptions == nil) != (deserialized.PlaybackOptions == nil) {
		t.Errorf("Channel[%d] PlaybackOptions nil mismatch", index)
		return
	}

	if original.PlaybackOptions != nil {
		if original.PlaybackOptions.FilePath != deserialized.PlaybackOptions.FilePath {
			t.Errorf("Channel[%d] PlaybackOptions.FilePath mismatch: original=%s, deserialized=%s",
				index, original.PlaybackOptions.FilePath, deserialized.PlaybackOptions.FilePath)
		}
		if original.PlaybackOptions.Rate != deserialized.PlaybackOptions.Rate {
			t.Errorf("Channel[%d] PlaybackOptions.Rate mismatch: original=%.3f, deserialized=%.3f",
				index, original.PlaybackOptions.Rate, deserialized.PlaybackOptions.Rate)
		}
		if original.PlaybackOptions.Pitch != deserialized.PlaybackOptions.Pitch {
			t.Errorf("Channel[%d] PlaybackOptions.Pitch mismatch: original=%.3f, deserialized=%.3f",
				index, original.PlaybackOptions.Pitch, deserialized.PlaybackOptions.Pitch)
		}
	}
}

// comparePluginChains compares plugin chains for serialization accuracy
func comparePluginChains(t *testing.T, channelIndex int, original, deserialized *PluginChain) {
	if (original == nil) != (deserialized == nil) {
		t.Errorf("Channel[%d] PluginChain nil mismatch: original=%v, deserialized=%v",
			channelIndex, original == nil, deserialized == nil)
		return
	}

	if original == nil {
		return // Both nil, OK
	}

	if len(original.Plugins) != len(deserialized.Plugins) {
		t.Errorf("Channel[%d] PluginChain length mismatch: original=%d, deserialized=%d",
			channelIndex, len(original.Plugins), len(deserialized.Plugins))
		return
	}

	// Compare each plugin in the chain
	for i := range original.Plugins {
		origPlugin := &original.Plugins[i]
		deserPlugin := &deserialized.Plugins[i]

		if origPlugin.Bypassed != deserPlugin.Bypassed {
			t.Errorf("Channel[%d] Plugin[%d] Bypassed mismatch: original=%v, deserialized=%v",
				channelIndex, i, origPlugin.Bypassed, deserPlugin.Bypassed)
		}

		// Compare parameter slices - check if plugins have same parameter count
		if origPlugin.Plugin != nil && deserPlugin.Plugin != nil {
			if len(origPlugin.Plugin.Parameters) != len(deserPlugin.Plugin.Parameters) {
				t.Errorf("Channel[%d] Plugin[%d] Parameters length mismatch: original=%d, deserialized=%d",
					channelIndex, i, len(origPlugin.Plugin.Parameters), len(deserPlugin.Plugin.Parameters))
				continue
			}

			// Check each parameter value by index
			for paramIndex, origParam := range origPlugin.Plugin.Parameters {
				deserParam := deserPlugin.Plugin.Parameters[paramIndex]

				if origParam.Identifier != deserParam.Identifier {
					t.Errorf("Channel[%d] Plugin[%d] Parameter[%d] identifier mismatch: original=%s, deserialized=%s",
						channelIndex, i, paramIndex, origParam.Identifier, deserParam.Identifier)
					continue
				}

				// Compare current values (accounting for float precision)
				if !parametersEqual(origParam.CurrentValue, deserParam.CurrentValue) {
					t.Errorf("Channel[%d] Plugin[%d] Parameter[%d] '%s' value mismatch: original=%v, deserialized=%v",
						channelIndex, i, paramIndex, origParam.Identifier, origParam.CurrentValue, deserParam.CurrentValue)
				}
			}
		}
	}
}

// parametersEqual compares parameter values accounting for JSON marshaling/unmarshaling type changes
func parametersEqual(a, b interface{}) bool {
	// JSON unmarshaling converts all numbers to float64
	switch aVal := a.(type) {
	case float32:
		if bVal, ok := b.(float64); ok {
			return float64(aVal) == bVal
		}
	case float64:
		if bVal, ok := b.(float64); ok {
			return aVal == bVal
		}
	case int:
		if bVal, ok := b.(float64); ok {
			return float64(aVal) == bVal
		}
	default:
		return a == b
	}
	return a == b
}

// logEngineDetails logs information about what's being tested
func logEngineDetails(t *testing.T, engine *Engine) {
	activeChannels := 0
	totalPlugins := 0
	inputDevices := make(map[string]bool)

	for i, channel := range engine.Channels {
		if channel != nil {
			activeChannels++
			channelType := "unknown"
			if channel.IsInput() {
				channelType = "input"
			} else if channel.IsPlayback() {
				channelType = "playback"
			}
			t.Logf("Channel[%d]: Type=%s, Volume=%.2f, Pan=%.2f",
				i, channelType, channel.Volume, channel.Pan)

			if channel.InputOptions != nil {
				if channel.InputOptions.Device != nil {
					inputDevices[channel.InputOptions.Device.UID] = true
				}
				if channel.InputOptions.PluginChain != nil {
					pluginCount := len(channel.InputOptions.PluginChain.Plugins)
					totalPlugins += pluginCount
					deviceName := "unknown"
					if channel.InputOptions.Device != nil {
						deviceName = channel.InputOptions.Device.Name
					}
					t.Logf("  → Input from device %s, %d plugins",
						deviceName, pluginCount)
				}
			}

			if channel.PlaybackOptions != nil {
				t.Logf("  → Playback file: %s (rate=%.2f, pitch=%.2f)",
					channel.PlaybackOptions.FilePath, channel.PlaybackOptions.Rate, channel.PlaybackOptions.Pitch)
			}
		}
	}

	t.Logf("Summary: %d active channels, %d total plugins, %d unique input devices",
		activeChannels, totalPlugins, len(inputDevices))
}
