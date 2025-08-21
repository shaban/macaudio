package engine

import (
	"errors"

	"github.com/shaban/macaudio/plugins"
)

// PluginChain manages a series of AudioUnit effects
type PluginChain struct {
	Plugins []EnginePlugin `json:"plugins"`
}

// EnginePlugin represents an AudioUnit effect in the chain with engine-specific state
type EnginePlugin struct {
	IsInstalled     bool            `json:"isInstalled"` // false when plugin no longer available on system
	*plugins.Plugin `json:"plugin"` // embedded plugin with independent parameter values
	Bypassed        bool            `json:"bypassed"` // Individual bypass control
}

// NewPluginChain creates an empty plugin processing chain
func NewPluginChain() *PluginChain {
	return &PluginChain{
		Plugins: make([]EnginePlugin, 0, 8), // Pre-allocate for up to 8 plugins
	}
}

// =============================================================================
// Plugin Chain Management
// =============================================================================

// AddPlugin adds an AudioUnit plugin to the end of the chain
func (pc *PluginChain) AddPlugin(plugin EnginePlugin) error {
	if len(pc.Plugins) >= 8 {
		return errors.New("plugin chain full (maximum 8 plugins)")
	}

	// TODO: Validate plugin exists using plugins package
	// TODO: Initialize plugin with default parameters
	// TODO: Connect plugin to audio chain

	pc.Plugins = append(pc.Plugins, plugin)
	return nil
}

// RemovePlugin removes a plugin from the chain by index
func (pc *PluginChain) RemovePlugin(index int) error {
	if index < 0 || index >= len(pc.Plugins) {
		return errors.New("invalid plugin index")
	}

	// TODO: Cleanup plugin resources
	// TODO: Disconnect plugin from audio chain

	pc.Plugins = append(pc.Plugins[:index], pc.Plugins[index+1:]...)
	return nil
}

// SetPluginBypassed enables or disables a plugin in the chain
func (pc *PluginChain) SetPluginBypassed(index int, bypassed bool) error {
	if index < 0 || index >= len(pc.Plugins) {
		return errors.New("invalid plugin index")
	}

	pc.Plugins[index].Bypassed = bypassed

	// TODO: Apply bypass state to actual AudioUnit

	return nil
}

// GetPlugin returns a plugin by index
func (pc *PluginChain) GetPlugin(index int) (*EnginePlugin, error) {
	if index < 0 || index >= len(pc.Plugins) {
		return nil, errors.New("invalid plugin index")
	}

	return &pc.Plugins[index], nil
}

// GetPluginCount returns the number of plugins in the chain
func (pc *PluginChain) GetPluginCount() int {
	return len(pc.Plugins)
}

// ClearPlugins removes all plugins from the chain
func (pc *PluginChain) ClearPlugins() error {
	// TODO: Cleanup all plugin resources

	pc.Plugins = pc.Plugins[:0] // Clear slice but keep capacity
	return nil
}

// ReorderPlugin moves a plugin to a different position in the chain
func (pc *PluginChain) ReorderPlugin(fromIndex, toIndex int) error {
	if fromIndex < 0 || fromIndex >= len(pc.Plugins) {
		return errors.New("invalid from index")
	}
	if toIndex < 0 || toIndex >= len(pc.Plugins) {
		return errors.New("invalid to index")
	}
	if fromIndex == toIndex {
		return nil // No-op
	}

	// TODO: Update audio chain connections for new order

	// Move the plugin
	plugin := pc.Plugins[fromIndex]
	pc.Plugins = append(pc.Plugins[:fromIndex], pc.Plugins[fromIndex+1:]...)

	// Insert at new position
	if toIndex > fromIndex {
		toIndex-- // Adjust for removal
	}

	pc.Plugins = append(pc.Plugins[:toIndex], append([]EnginePlugin{plugin}, pc.Plugins[toIndex:]...)...)

	return nil
}

// =============================================================================
// Plugin Factory Methods
// =============================================================================

// NewEnginePlugin creates a new EnginePlugin from a plugins.Plugin
func NewEnginePlugin(plugin *plugins.Plugin) *EnginePlugin {
	return &EnginePlugin{
		IsInstalled: true, // Assume installed since we just introspected it
		Plugin:      plugin,
		Bypassed:    false,
	}
}

// CreatePluginFromInfo creates an EnginePlugin from a PluginInfo
// Uses PluginInfo.Introspect() to get the actual plugin with full 4-tuple lookup
// Sets IsInstalled=false if introspection fails or doesn't yield exactly 1 plugin
func CreatePluginFromInfo(pluginInfo plugins.PluginInfo) (*EnginePlugin, error) {
	// Use the plugin's introspect method which handles the 4-tuple lookup
	plugin, err := pluginInfo.Introspect()
	if err != nil {
		// Introspection failed - plugin exists in list but can't be loaded
		return &EnginePlugin{
			IsInstalled: false,
			Plugin:      nil,
			Bypassed:    false,
		}, nil // Return success with IsInstalled=false, not an error
	}

	// Successfully introspected - plugin is available and loaded
	return &EnginePlugin{
		IsInstalled: true,
		Plugin:      plugin,
		Bypassed:    false,
	}, nil
}

// =============================================================================
// Plugin Parameter Management
// =============================================================================

// SetPluginParameter sets a parameter value for a specific plugin by parameter address or identifier
func (pc *PluginChain) SetPluginParameter(pluginIndex int, paramIdentifier string, value float32) error {
	if pluginIndex < 0 || pluginIndex >= len(pc.Plugins) {
		return errors.New("invalid plugin index")
	}

	plugin := &pc.Plugins[pluginIndex]
	if plugin.Plugin == nil {
		return errors.New("plugin not initialized")
	}

	// Find parameter by identifier or display name
	for i := range plugin.Plugin.Parameters {
		param := &plugin.Plugin.Parameters[i]
		if param.Identifier == paramIdentifier || param.DisplayName == paramIdentifier {
			// Validate value is within bounds
			if value < param.MinValue || value > param.MaxValue {
				return errors.New("parameter value out of bounds")
			}

			// Set the current value
			param.CurrentValue = value

			// TODO: Apply parameter to actual AudioUnit

			return nil
		}
	}

	return errors.New("parameter not found")
}

// GetPluginParameter gets a parameter value for a specific plugin by identifier
func (pc *PluginChain) GetPluginParameter(pluginIndex int, paramIdentifier string) (float32, error) {
	if pluginIndex < 0 || pluginIndex >= len(pc.Plugins) {
		return 0, errors.New("invalid plugin index")
	}

	plugin := &pc.Plugins[pluginIndex]
	if plugin.Plugin == nil {
		return 0, errors.New("plugin not initialized")
	}

	// Find parameter by identifier or display name
	for _, param := range plugin.Plugin.Parameters {
		if param.Identifier == paramIdentifier || param.DisplayName == paramIdentifier {
			return param.CurrentValue, nil
		}
	}

	return 0, errors.New("parameter not found")
}

// GetPluginParameterNames returns all parameter identifiers for a specific plugin
func (pc *PluginChain) GetPluginParameterNames(pluginIndex int) ([]string, error) {
	if pluginIndex < 0 || pluginIndex >= len(pc.Plugins) {
		return nil, errors.New("invalid plugin index")
	}

	plugin := &pc.Plugins[pluginIndex]
	if plugin.Plugin == nil {
		return nil, errors.New("plugin not initialized")
	}

	names := make([]string, 0, len(plugin.Plugin.Parameters))
	for _, param := range plugin.Plugin.Parameters {
		names = append(names, param.Identifier)
	}

	return names, nil
}
