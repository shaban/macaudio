package pluginchain

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AudioUnit.h>

// Connect effects in series using AVAudioEngine
bool connect_effects(void* enginePtr, void** effectPtrs, int effectCount) {
    if (!enginePtr || !effectPtrs || effectCount <= 0) return false;

    AVAudioEngine* engine = (__bridge AVAudioEngine*)enginePtr;

    @try {
        // First, disconnect all effects that are currently attached to prevent conflicts
        for (int i = 0; i < effectCount; i++) {
            AVAudioNode* effect = (__bridge AVAudioNode*)effectPtrs[i];
            if (!effect) {
                NSLog(@"Invalid effect at index %d", i);
                return false;
            }

            // Disconnect the node if it's already connected
            if ([engine.attachedNodes containsObject:effect]) {
                [engine disconnectNodeInput:effect bus:0];
                [engine disconnectNodeOutput:effect bus:0];
            }
        }

        // Then, attach all effects to the engine if not already attached
        for (int i = 0; i < effectCount; i++) {
            AVAudioNode* effect = (__bridge AVAudioNode*)effectPtrs[i];

            // Check if already attached to avoid duplicate attachment
            if (![engine.attachedNodes containsObject:effect]) {
                [engine attachNode:effect];
            }
        }

        // Finally, connect effects in series: effect[0] -> effect[1] -> ... -> effect[n-1]
        for (int i = 0; i < effectCount - 1; i++) {
            AVAudioNode* sourceEffect = (__bridge AVAudioNode*)effectPtrs[i];
            AVAudioNode* destinationEffect = (__bridge AVAudioNode*)effectPtrs[i + 1];

            if (!sourceEffect || !destinationEffect) {
                NSLog(@"Invalid effect at index %d or %d", i, i + 1);
                return false;
            }

            // Connect source to destination
            [engine connect:sourceEffect to:destinationEffect format:nil];
        }

        return true;
    }
    @catch (NSException* exception) {
        NSLog(@"Exception connecting effects: %@", exception);
        return false;
    }
}

// Get the last effect's audio node for external routing
void* get_effect_audio_node(void* effectPtr) {
    if (!effectPtr) return NULL;

    // Effects are already AVAudioNode instances, so we just return the pointer
    return effectPtr;
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/unit"
	"github.com/shaban/macaudio/plugins"
)

// PluginChain represents a reorderable chain of audio effects
type PluginChain struct {
	name      string
	effects   []*unit.Effect
	plugins   []plugins.Plugin
	enginePtr unsafe.Pointer // Reference to AVAudioEngine for connections
}

// ChainConfig holds configuration for creating a plugin chain
type ChainConfig struct {
	Name      string
	EnginePtr unsafe.Pointer // AVAudioEngine pointer from engine package
}

// NewPluginChain creates a new empty plugin chain
func NewPluginChain(config ChainConfig) *PluginChain {
	return &PluginChain{
		name:      config.Name,
		effects:   make([]*unit.Effect, 0),
		plugins:   make([]plugins.Plugin, 0),
		enginePtr: config.EnginePtr,
	}
}

// AddEffect adds an effect to the end of the chain using plugin discovery
func (pc *PluginChain) AddEffect(plugin plugins.Plugin) error {
	if pc.enginePtr == nil {
		return fmt.Errorf("chain %s has no engine reference", pc.name)
	}

	// Create the effect using our unit package
	effect, err := unit.CreateEffect(plugin)
	if err != nil {
		return fmt.Errorf("failed to create effect %s: %v", plugin.Name, err)
	}

	// Add to our Go-side bookkeeping
	pc.effects = append(pc.effects, effect)
	pc.plugins = append(pc.plugins, plugin)

	// Update native connections
	return pc.updateConnections()
}

// AddEffectFromPluginInfo is a convenience method for adding effects from plugin discovery
func (pc *PluginChain) AddEffectFromPluginInfo(pluginInfo plugins.PluginInfo) error {
	// Use plugins package introspection to get full plugin details
	plugin, err := pluginInfo.Introspect()
	if err != nil {
		return fmt.Errorf("failed to introspect plugin %s: %v", pluginInfo.Name, err)
	}

	return pc.AddEffect(plugin)
}

// InsertEffect inserts an effect at the specified index
func (pc *PluginChain) InsertEffect(index int, plugin plugins.Plugin) error {
	if index < 0 || index > len(pc.effects) {
		return fmt.Errorf("invalid index %d for chain of length %d", index, len(pc.effects))
	}

	if pc.enginePtr == nil {
		return fmt.Errorf("chain %s has no engine reference", pc.name)
	}

	// Create the effect
	effect, err := unit.CreateEffect(plugin)
	if err != nil {
		return fmt.Errorf("failed to create effect %s: %v", plugin.Name, err)
	}

	// Insert into slices at the specified index
	pc.effects = append(pc.effects[:index], append([]*unit.Effect{effect}, pc.effects[index:]...)...)
	pc.plugins = append(pc.plugins[:index], append([]plugins.Plugin{plugin}, pc.plugins[index:]...)...)

	// Update native connections
	return pc.updateConnections()
}

// RemoveEffect removes an effect at the specified index
func (pc *PluginChain) RemoveEffect(index int) error {
	if index < 0 || index >= len(pc.effects) {
		return fmt.Errorf("invalid index %d for chain of length %d", index, len(pc.effects))
	}

	// Release the effect resources
	pc.effects[index].Release()

	// Remove from slices
	pc.effects = append(pc.effects[:index], pc.effects[index+1:]...)
	pc.plugins = append(pc.plugins[:index], pc.plugins[index+1:]...)

	// Update native connections
	return pc.updateConnections()
}

// MoveEffect moves an effect from one position to another
func (pc *PluginChain) MoveEffect(fromIndex, toIndex int) error {
	if fromIndex < 0 || fromIndex >= len(pc.effects) {
		return fmt.Errorf("invalid fromIndex %d for chain of length %d", fromIndex, len(pc.effects))
	}
	if toIndex < 0 || toIndex >= len(pc.effects) {
		return fmt.Errorf("invalid toIndex %d for chain of length %d", toIndex, len(pc.effects))
	}
	if fromIndex == toIndex {
		return nil // No-op
	}

	// Store the items to move
	effect := pc.effects[fromIndex]
	plugin := pc.plugins[fromIndex]

	// Remove from current position
	pc.effects = append(pc.effects[:fromIndex], pc.effects[fromIndex+1:]...)
	pc.plugins = append(pc.plugins[:fromIndex], pc.plugins[fromIndex+1:]...)

	// For moving forward, we need to insert at the original toIndex position
	// but since we removed an element, the actual insert index is toIndex (not toIndex-1)
	insertIndex := toIndex
	if toIndex > fromIndex {
		// We removed an element before toIndex, but we want to end up at original toIndex
		// So we insert at toIndex-1 in the new shortened array, which puts it at original toIndex
		insertIndex = toIndex
	}

	// Insert at calculated position
	pc.effects = append(pc.effects[:insertIndex], append([]*unit.Effect{effect}, pc.effects[insertIndex:]...)...)
	pc.plugins = append(pc.plugins[:insertIndex], append([]plugins.Plugin{plugin}, pc.plugins[insertIndex:]...)...)

	// Update native connections
	return pc.updateConnections()
}

// SwapEffects swaps two effects in the chain
func (pc *PluginChain) SwapEffects(index1, index2 int) error {
	if index1 < 0 || index1 >= len(pc.effects) {
		return fmt.Errorf("invalid index1 %d for chain of length %d", index1, len(pc.effects))
	}
	if index2 < 0 || index2 >= len(pc.effects) {
		return fmt.Errorf("invalid index2 %d for chain of length %d", index2, len(pc.effects))
	}
	if index1 == index2 {
		return nil // No-op
	}

	// Swap in both slices
	pc.effects[index1], pc.effects[index2] = pc.effects[index2], pc.effects[index1]
	pc.plugins[index1], pc.plugins[index2] = pc.plugins[index2], pc.plugins[index1]

	// Update native connections
	return pc.updateConnections()
}

// SetParameter sets a parameter on a specific effect in the chain
func (pc *PluginChain) SetParameter(effectIndex int, param plugins.Parameter, value float32) error {
	if effectIndex < 0 || effectIndex >= len(pc.effects) {
		return fmt.Errorf("invalid effect index %d for chain of length %d", effectIndex, len(pc.effects))
	}

	return pc.effects[effectIndex].SetParameter(param, value)
}

// GetParameter gets a parameter value from a specific effect in the chain
func (pc *PluginChain) GetParameter(effectIndex int, param plugins.Parameter) (float32, error) {
	if effectIndex < 0 || effectIndex >= len(pc.effects) {
		return 0, fmt.Errorf("invalid effect index %d for chain of length %d", effectIndex, len(pc.effects))
	}

	return pc.effects[effectIndex].GetParameter(param)
}

// updateConnections updates the native AVAudioEngine connections for the chain
func (pc *PluginChain) updateConnections() error {
	if len(pc.effects) == 0 {
		return nil // Empty chain, nothing to connect
	}

	if pc.enginePtr == nil {
		return fmt.Errorf("chain %s has no engine reference", pc.name)
	}

	// Build array of effect pointers for native code
	effectPtrs := make([]unsafe.Pointer, len(pc.effects))
	for i, effect := range pc.effects {
		effectPtrs[i] = effect.Ptr()
	}

	// Use native function to connect effects in series
	success := bool(C.connect_effects(pc.enginePtr, &effectPtrs[0], C.int(len(effectPtrs))))
	if !success {
		return fmt.Errorf("failed to update native connections for chain %s", pc.name)
	}

	return nil
}

// GetInputNode returns the first effect in the chain for external routing
func (pc *PluginChain) GetInputNode() unsafe.Pointer {
	if len(pc.effects) == 0 {
		return nil
	}
	return C.get_effect_audio_node(pc.effects[0].Ptr())
}

// GetOutputNode returns the last effect in the chain for external routing
func (pc *PluginChain) GetOutputNode() unsafe.Pointer {
	if len(pc.effects) == 0 {
		return nil
	}
	return C.get_effect_audio_node(pc.effects[len(pc.effects)-1].Ptr())
}

// GetEffectCount returns the number of effects in the chain
func (pc *PluginChain) GetEffectCount() int {
	return len(pc.effects)
}

// GetEffectAt returns the effect and plugin at the specified index
func (pc *PluginChain) GetEffectAt(index int) (*unit.Effect, plugins.Plugin, error) {
	if index < 0 || index >= len(pc.effects) {
		return nil, plugins.Plugin{}, fmt.Errorf("invalid index %d for chain of length %d", index, len(pc.effects))
	}
	return pc.effects[index], pc.plugins[index], nil
}

// GetName returns the chain name
func (pc *PluginChain) GetName() string {
	return pc.name
}

// SetName updates the chain name
func (pc *PluginChain) SetName(name string) {
	pc.name = name
}

// IsEmpty returns true if the chain has no effects
func (pc *PluginChain) IsEmpty() bool {
	return len(pc.effects) == 0
}

// Clear removes all effects from the chain
func (pc *PluginChain) Clear() error {
	// Release all effects
	for _, effect := range pc.effects {
		effect.Release()
	}

	// Clear slices
	pc.effects = pc.effects[:0]
	pc.plugins = pc.plugins[:0]

	return nil
}

// Release releases all resources used by the chain
func (pc *PluginChain) Release() {
	pc.Clear()
}

// Summary returns a brief summary of the chain
func (pc *PluginChain) Summary() string {
	if len(pc.effects) == 0 {
		return fmt.Sprintf("Chain '%s': empty", pc.name)
	}

	effectNames := make([]string, len(pc.plugins))
	for i, plugin := range pc.plugins {
		effectNames[i] = plugin.Name
	}

	return fmt.Sprintf("Chain '%s': %d effects [%s]", pc.name, len(pc.effects),
		joinStrings(effectNames, " -> "))
}

// GetEffectNames returns a slice of effect names in chain order
func (pc *PluginChain) GetEffectNames() []string {
	names := make([]string, len(pc.plugins))
	for i, plugin := range pc.plugins {
		names[i] = plugin.Name
	}
	return names
}

// Helper function to join strings
func joinStrings(strings []string, separator string) string {
	if len(strings) == 0 {
		return ""
	}
	if len(strings) == 1 {
		return strings[0]
	}

	result := strings[0]
	for i := 1; i < len(strings); i++ {
		result += separator + strings[i]
	}
	return result
}
