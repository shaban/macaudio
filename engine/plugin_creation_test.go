package engine

import (
	"testing"

	"github.com/shaban/macaudio/plugins"
)

func TestCreatePluginFromInfo(t *testing.T) {
	t.Run("ValidPlugin", func(t *testing.T) {
		// Get a real plugin to test with
		pluginInfos, err := plugins.List()
		if err != nil {
			t.Skip("No plugins available for testing")
		}
		if len(pluginInfos) == 0 {
			t.Skip("No plugins found")
		}

		// Use first available plugin
		pluginInfo := pluginInfos[0]

		enginePlugin, err := CreatePluginFromInfo(pluginInfo)
		if err != nil {
			t.Fatalf("CreatePluginFromInfo failed: %v", err)
		}

		if enginePlugin == nil {
			t.Fatal("CreatePluginFromInfo returned nil")
		}

		// Should either be installed (loaded successfully) or not installed (failed to load)
		// Both are valid outcomes
		t.Logf("Plugin %s: IsInstalled=%v", pluginInfo.Name, enginePlugin.IsInstalled)

		if enginePlugin.IsInstalled {
			if enginePlugin.Plugin == nil {
				t.Error("IsInstalled=true but Plugin=nil")
			}
		} else {
			// IsInstalled=false is OK - plugin exists but couldn't be loaded
			t.Logf("Plugin exists but couldn't be loaded (IsInstalled=false)")
		}
	})

	t.Run("InvalidPlugin", func(t *testing.T) {
		// Create a fake PluginInfo that doesn't exist
		fakePlugin := plugins.PluginInfo{
			Name:           "NonExistentPlugin",
			ManufacturerID: "FAKE",
			Type:           "fake",
			Subtype:        "test",
			Category:       "None",
		}

		enginePlugin, err := CreatePluginFromInfo(fakePlugin)

		// Should not return an error - instead IsInstalled should be false
		if err != nil {
			t.Errorf("CreatePluginFromInfo should not return error for invalid plugin, got: %v", err)
		}

		if enginePlugin == nil {
			t.Fatal("CreatePluginFromInfo returned nil")
		}

		// Should have IsInstalled=false for invalid plugin
		if enginePlugin.IsInstalled {
			t.Error("Invalid plugin should have IsInstalled=false")
		}

		if enginePlugin.Plugin != nil {
			t.Error("Invalid plugin should have Plugin=nil")
		}
	})
}
