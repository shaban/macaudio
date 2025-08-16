package macaudio

import (
	"fmt"
	"sync"
	
	"github.com/shaban/macaudio/plugins"
)

// PluginBlueprint represents a plugin template that can be instantiated
type PluginBlueprint struct {
	Type           string  `json:"type"`
	Subtype        string  `json:"subtype"`
	ManufacturerID string  `json:"manufacturerID"`
	Name           string  `json:"name"`
	IsInstalled    bool    `json:"isInstalled"`
}

// PluginInstance represents an instantiated plugin in a chain
type PluginInstance struct {
	ID         string           `json:"id"`
	Blueprint  PluginBlueprint  `json:"blueprint"`
	Position   int              `json:"position"`
	IsActive   bool             `json:"isActive"`
	IsLoaded   bool             `json:"isLoaded"`
	Parameters map[string]float32 `json:"parameters"`
	
	// Internal state
	mu         sync.RWMutex
	plugin     *plugins.Plugin  // Full plugin data when loaded
}

// PluginChain manages a sequence of audio plugins for a channel
type PluginChain struct {
	mu        sync.RWMutex
	instances []*PluginInstance
	nextID    int
}

// PluginChainState represents the serializable state of a plugin chain
type PluginChainState struct {
	Instances []PluginInstanceState `json:"instances"`
}

// PluginInstanceState represents the serializable state of a plugin instance
type PluginInstanceState struct {
	ID         string              `json:"id"`
	Blueprint  PluginBlueprint     `json:"blueprint"`
	Position   int                 `json:"position"`
	IsActive   bool                `json:"isActive"`
	Parameters map[string]float32  `json:"parameters"`
}

// NewPluginChain creates a new empty plugin chain
func NewPluginChain() *PluginChain {
	return &PluginChain{
		instances: make([]*PluginInstance, 0),
		nextID:    1,
	}
}

// AddPlugin adds a plugin instance to the chain at the specified position
func (pc *PluginChain) AddPlugin(blueprint PluginBlueprint, position int) (*PluginInstance, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	// Validate position
	if position < 0 || position > len(pc.instances) {
		return nil, fmt.Errorf("invalid position %d for plugin chain", position)
	}
	
	// Create new instance
	instance := &PluginInstance{
		ID:         fmt.Sprintf("plugin_%d", pc.nextID),
		Blueprint:  blueprint,
		Position:   position,
		IsActive:   true,
		IsLoaded:   false,
		Parameters: make(map[string]float32),
	}
	pc.nextID++
	
	// Insert at position
	if position == len(pc.instances) {
		// Append to end
		pc.instances = append(pc.instances, instance)
	} else {
		// Insert at position
		pc.instances = append(pc.instances, nil)
		copy(pc.instances[position+1:], pc.instances[position:])
		pc.instances[position] = instance
	}
	
	// Update positions of subsequent plugins
	for i := position + 1; i < len(pc.instances); i++ {
		pc.instances[i].Position = i
	}
	
	// Try to load the plugin
	if err := instance.Load(); err != nil {
		// Plugin loading failed, but we still add it to the chain as inactive
		instance.IsActive = false
		instance.Blueprint.IsInstalled = false
	}
	
	return instance, nil
}

// RemovePlugin removes a plugin instance from the chain
func (pc *PluginChain) RemovePlugin(instanceID string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	for i, instance := range pc.instances {
		if instance.ID == instanceID {
			// Unload plugin before removing
			instance.Unload()
			
			// Remove from slice
			copy(pc.instances[i:], pc.instances[i+1:])
			pc.instances[len(pc.instances)-1] = nil
			pc.instances = pc.instances[:len(pc.instances)-1]
			
			// Update positions of subsequent plugins
			for j := i; j < len(pc.instances); j++ {
				pc.instances[j].Position = j
			}
			
			return nil
		}
	}
	
	return fmt.Errorf("plugin instance %s not found", instanceID)
}

// GetInstances returns a copy of all plugin instances
func (pc *PluginChain) GetInstances() []*PluginInstance {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	instances := make([]*PluginInstance, len(pc.instances))
	copy(instances, pc.instances)
	return instances
}

// GetInstance returns a specific plugin instance by ID
func (pc *PluginChain) GetInstance(instanceID string) (*PluginInstance, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	for _, instance := range pc.instances {
		if instance.ID == instanceID {
			return instance, true
		}
	}
	return nil, false
}

// GetState returns the serializable state of the plugin chain
func (pc *PluginChain) GetState() PluginChainState {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	states := make([]PluginInstanceState, len(pc.instances))
	for i, instance := range pc.instances {
		states[i] = instance.GetState()
	}
	
	return PluginChainState{
		Instances: states,
	}
}

// SetState restores the plugin chain from serializable state
func (pc *PluginChain) SetState(state PluginChainState) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	// Clear existing instances
	for _, instance := range pc.instances {
		instance.Unload()
	}
	pc.instances = make([]*PluginInstance, 0, len(state.Instances))
	
	// Restore instances from state
	for _, instanceState := range state.Instances {
		instance := &PluginInstance{
			ID:         instanceState.ID,
			Blueprint:  instanceState.Blueprint,
			Position:   instanceState.Position,
			IsActive:   instanceState.IsActive,
			IsLoaded:   false,
			Parameters: make(map[string]float32),
		}
		
		// Copy parameters
		for key, value := range instanceState.Parameters {
			instance.Parameters[key] = value
		}
		
		pc.instances = append(pc.instances, instance)
		
		// Try to load the plugin
		if instance.IsActive {
			if err := instance.Load(); err != nil {
				// Plugin loading failed
				instance.IsActive = false
				instance.Blueprint.IsInstalled = false
			}
		}
	}
	
	return nil
}

// Load attempts to load the plugin from the system
func (pi *PluginInstance) Load() error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	if pi.IsLoaded {
		return nil // Already loaded
	}
	
	// Create PluginInfo from blueprint
	info := plugins.PluginInfo{
		Name:           pi.Blueprint.Name,
		ManufacturerID: pi.Blueprint.ManufacturerID,
		Type:           pi.Blueprint.Type,
		Subtype:        pi.Blueprint.Subtype,
	}
	
	// Introspect to get full plugin data
	plugin, err := info.Introspect()
	if err != nil {
		return fmt.Errorf("failed to introspect plugin: %w", err)
	}
	
	pi.plugin = plugin
	pi.IsLoaded = true
	pi.Blueprint.IsInstalled = true
	
	return nil
}

// Unload unloads the plugin and releases resources
func (pi *PluginInstance) Unload() {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	if !pi.IsLoaded {
		return
	}
	
	pi.plugin = nil
	pi.IsLoaded = false
}

// GetPlugin returns the loaded plugin data (thread-safe)
func (pi *PluginInstance) GetPlugin() *plugins.Plugin {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.plugin
}

// SetParameter sets a plugin parameter value
func (pi *PluginInstance) SetParameter(name string, value float32) error {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	if !pi.IsLoaded {
		return fmt.Errorf("plugin not loaded")
	}
	
	// TODO: Validate parameter exists and range
	// TODO: Apply to actual plugin instance
	
	pi.Parameters[name] = value
	return nil
}

// GetParameter gets a plugin parameter value
func (pi *PluginInstance) GetParameter(name string) (float32, bool) {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	value, exists := pi.Parameters[name]
	return value, exists
}

// GetState returns the serializable state of the plugin instance
func (pi *PluginInstance) GetState() PluginInstanceState {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	
	// Copy parameters map
	params := make(map[string]float32)
	for k, v := range pi.Parameters {
		params[k] = v
	}
	
	return PluginInstanceState{
		ID:         pi.ID,
		Blueprint:  pi.Blueprint,
		Position:   pi.Position,
		IsActive:   pi.IsActive,
		Parameters: params,
	}
}
