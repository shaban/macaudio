package main

import (
	"fmt"
	"log"

	"github.com/shaban/macaudio/engine"
	"github.com/shaban/macaudio/plugins"
)

// PluginCreationDemo demonstrates the correct way to create plugins
func main() {
	fmt.Println("🎵 MacAudio Plugin Creation Demo")
	fmt.Println("=====================================")

	// Step 1: Get list of available plugins for UI/programmatic selection
	fmt.Println("📋 Getting available plugins...")
	pluginInfos, err := plugins.List()
	if err != nil {
		log.Fatalf("Failed to list plugins: %v", err)
	}

	fmt.Printf("Found %d plugins\n\n", len(pluginInfos))

	// Step 2: Find a specific plugin (e.g., for a UI selection or programmatic choice)
	// Let's find a time-based effect
	timeEffects := pluginInfos.ByType("aufx").ByCategory("Effect")
	if len(timeEffects) == 0 {
		fmt.Println("⚠️  No time effects found, using first available plugin")
		if len(pluginInfos) == 0 {
			fmt.Println("❌ No plugins available")
			return
		}
		timeEffects = pluginInfos[:1]
	}

	selectedPlugin := timeEffects[0]
	fmt.Printf("🎛️  Selected plugin: %s by %s\n", selectedPlugin.Name, selectedPlugin.ManufacturerID)
	fmt.Printf("   Type: %s, Subtype: %s, Category: %s\n",
		selectedPlugin.Type, selectedPlugin.Subtype, selectedPlugin.Category)

	// Step 3: Create EnginePlugin from PluginInfo (the correct approach!)
	fmt.Println("\n🔧 Creating EnginePlugin...")
	enginePlugin, err := engine.CreatePluginFromInfo(selectedPlugin)
	if err != nil {
		log.Fatalf("Failed to create plugin: %v", err)
	}

	// Step 4: Check if plugin is properly installed and usable
	fmt.Printf("✅ EnginePlugin created successfully!\n")
	fmt.Printf("   IsInstalled: %v\n", enginePlugin.IsInstalled)
	fmt.Printf("   Bypassed: %v\n", enginePlugin.Bypassed)

	if enginePlugin.IsInstalled && enginePlugin.Plugin != nil {
		fmt.Printf("   Plugin loaded with %d parameters\n", len(enginePlugin.Plugin.Parameters))

		// Show first few parameters as example
		for i, param := range enginePlugin.Plugin.Parameters {
			if i >= 3 { // Show max 3 parameters
				fmt.Printf("   ... and %d more parameters\n", len(enginePlugin.Plugin.Parameters)-3)
				break
			}
			fmt.Printf("   [%d] %s (%.2f - %.2f, default: %.2f)\n",
				i, param.DisplayName, param.MinValue, param.MaxValue, param.DefaultValue)
		}
	} else {
		fmt.Println("   ⚠️  Plugin exists in system but failed to load (IsInstalled=false)")
	}

	fmt.Println("\n🎉 Demo completed!")
	fmt.Println("\nKey points:")
	fmt.Println("  • Use plugins.List() to get PluginInfo objects for UI")
	fmt.Println("  • Pass PluginInfo to engine.CreatePluginFromInfo()")
	fmt.Println("  • Check IsInstalled flag to see if plugin loaded successfully")
	fmt.Println("  • PluginInfo.Introspect() uses the full 4-tuple for exact matching")
}
