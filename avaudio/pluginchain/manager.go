package pluginchain

import (
	"fmt"
	"sort"
	"unsafe"

	"github.com/shaban/macaudio/plugins"
)

// ChainManager manages multiple named plugin chains for an audio engine
type ChainManager struct {
	chains    map[string]*PluginChain
	enginePtr unsafe.Pointer // Shared AVAudioEngine for all chains
}

// ManagerConfig holds configuration for creating a chain manager
type ManagerConfig struct {
	EnginePtr unsafe.Pointer // Shared AVAudioEngine pointer from engine package
}

// NewChainManager creates a new chain manager for managing multiple plugin chains
func NewChainManager(config ManagerConfig) *ChainManager {
	return &ChainManager{
		chains:    make(map[string]*PluginChain),
		enginePtr: config.EnginePtr,
	}
}

// CreateChain creates a new named plugin chain
func (cm *ChainManager) CreateChain(name string) (*PluginChain, error) {
	if name == "" {
		return nil, fmt.Errorf("chain name cannot be empty")
	}

	if _, exists := cm.chains[name]; exists {
		return nil, fmt.Errorf("chain '%s' already exists", name)
	}

	if cm.enginePtr == nil {
		return nil, fmt.Errorf("chain manager has no engine reference")
	}

	config := ChainConfig{
		Name:      name,
		EnginePtr: cm.enginePtr,
	}

	chain := NewPluginChain(config)
	cm.chains[name] = chain

	return chain, nil
}

// GetChain retrieves a plugin chain by name
func (cm *ChainManager) GetChain(name string) (*PluginChain, error) {
	chain, exists := cm.chains[name]
	if !exists {
		return nil, fmt.Errorf("chain '%s' not found", name)
	}
	return chain, nil
}

// DeleteChain removes a plugin chain by name
func (cm *ChainManager) DeleteChain(name string) error {
	chain, exists := cm.chains[name]
	if !exists {
		return fmt.Errorf("chain '%s' not found", name)
	}

	// Release chain resources
	chain.Release()

	// Remove from collection
	delete(cm.chains, name)

	return nil
}

// RenameChain changes the name of an existing chain
func (cm *ChainManager) RenameChain(oldName, newName string) error {
	if newName == "" {
		return fmt.Errorf("new chain name cannot be empty")
	}

	if oldName == newName {
		return nil // No-op
	}

	chain, exists := cm.chains[oldName]
	if !exists {
		return fmt.Errorf("chain '%s' not found", oldName)
	}

	if _, exists := cm.chains[newName]; exists {
		return fmt.Errorf("chain '%s' already exists", newName)
	}

	// Update the chain's internal name
	chain.SetName(newName)

	// Move in the map
	cm.chains[newName] = chain
	delete(cm.chains, oldName)

	return nil
}

// ListChains returns a sorted list of all chain names
func (cm *ChainManager) ListChains() []string {
	names := make([]string, 0, len(cm.chains))
	for name := range cm.chains {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetChainCount returns the total number of managed chains
func (cm *ChainManager) GetChainCount() int {
	return len(cm.chains)
}

// HasChain checks if a chain with the given name exists
func (cm *ChainManager) HasChain(name string) bool {
	_, exists := cm.chains[name]
	return exists
}

// GetAllChains returns a map of all chains (name -> chain)
func (cm *ChainManager) GetAllChains() map[string]*PluginChain {
	// Return a copy to prevent external modification
	result := make(map[string]*PluginChain)
	for name, chain := range cm.chains {
		result[name] = chain
	}
	return result
}

// CreateChainFromPluginInfos creates a chain and populates it with effects from plugin discovery
func (cm *ChainManager) CreateChainFromPluginInfos(name string, pluginInfos []plugins.PluginInfo) (*PluginChain, error) {
	chain, err := cm.CreateChain(name)
	if err != nil {
		return nil, err
	}

	// Add effects from plugin infos
	for i, pluginInfo := range pluginInfos {
		err := chain.AddEffectFromPluginInfo(pluginInfo)
		if err != nil {
			// If we fail partway through, clean up
			cm.DeleteChain(name)
			return nil, fmt.Errorf("failed to add effect %d (%s): %v", i, pluginInfo.Name, err)
		}
	}

	return chain, nil
}

// CloneChain creates a copy of an existing chain with a new name
func (cm *ChainManager) CloneChain(sourceName, targetName string) (*PluginChain, error) {
	if targetName == "" {
		return nil, fmt.Errorf("target chain name cannot be empty")
	}

	sourceChain, exists := cm.chains[sourceName]
	if !exists {
		return nil, fmt.Errorf("source chain '%s' not found", sourceName)
	}

	if _, exists := cm.chains[targetName]; exists {
		return nil, fmt.Errorf("target chain '%s' already exists", targetName)
	}

	// Create new chain
	targetChain, err := cm.CreateChain(targetName)
	if err != nil {
		return nil, err
	}

	// Copy all effects from source to target
	for i := 0; i < sourceChain.GetEffectCount(); i++ {
		_, plugin, err := sourceChain.GetEffectAt(i)
		if err != nil {
			// Clean up on failure
			cm.DeleteChain(targetName)
			return nil, fmt.Errorf("failed to get effect %d from source chain: %v", i, err)
		}

		err = targetChain.AddEffect(plugin)
		if err != nil {
			// Clean up on failure
			cm.DeleteChain(targetName)
			return nil, fmt.Errorf("failed to add effect %d to target chain: %v", i, err)
		}
	}

	return targetChain, nil
}

// ClearAllChains removes all chains and releases their resources
func (cm *ChainManager) ClearAllChains() error {
	var firstError error

	// Release all chains
	for name, chain := range cm.chains {
		err := chain.Clear()
		if err != nil && firstError == nil {
			firstError = fmt.Errorf("failed to clear chain '%s': %v", name, err)
		}
		chain.Release()
	}

	// Clear the map
	cm.chains = make(map[string]*PluginChain)

	return firstError
}

// GetChainsWithEffect returns chain names that contain a specific effect (by plugin name)
func (cm *ChainManager) GetChainsWithEffect(effectName string) []string {
	var result []string

	for chainName, chain := range cm.chains {
		effectNames := chain.GetEffectNames()
		for _, name := range effectNames {
			if name == effectName {
				result = append(result, chainName)
				break
			}
		}
	}

	sort.Strings(result)
	return result
}

// GetTotalEffectCount returns the total number of effects across all chains
func (cm *ChainManager) GetTotalEffectCount() int {
	total := 0
	for _, chain := range cm.chains {
		total += chain.GetEffectCount()
	}
	return total
}

// Summary returns a summary of all managed chains
func (cm *ChainManager) Summary() string {
	if len(cm.chains) == 0 {
		return "ChainManager: no chains"
	}

	totalEffects := cm.GetTotalEffectCount()
	return fmt.Sprintf("ChainManager: %d chains, %d total effects", len(cm.chains), totalEffects)
}

// GetChainsSummary returns detailed summary of all chains
func (cm *ChainManager) GetChainsSummary() map[string]string {
	result := make(map[string]string)
	for name, chain := range cm.chains {
		result[name] = chain.Summary()
	}
	return result
}

// Release releases all resources used by the chain manager
func (cm *ChainManager) Release() {
	cm.ClearAllChains()
}
