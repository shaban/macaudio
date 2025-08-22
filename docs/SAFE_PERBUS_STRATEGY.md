# Safe Per-Bus Strategy Implementation Status

## 🎯 **Current Status**

### ✅ **What's Working**
- **Native infrastructure**: Smart per-bus functions exist in `native/node.m`
- **Go compilation**: Basic structure compiles successfully  
- **Existing API preserved**: All current functionality unchanged
- **Test suite passes**: No regressions in existing features

### � **What We Discovered**
- **Complex array handling**: Go→C array conversion requires careful memory management
- **@autoreleasepool patterns**: Native code uses this extensively for memory safety
- **Existing per-connection API**: AVAudioMixingDestination already provides per-connection control

## 🛠️ **Safe Implementation Plan**

### **Phase 1: Connection Tracking Infrastructure** 
Add lightweight connection tracking to the Engine struct without breaking existing code:

```go
// Add to Engine struct (non-breaking addition)
type Engine struct {
    // ... existing fields ...
    connections map[uintptr][]unsafe.Pointer // mixer → sources mapping
    mu          sync.RWMutex                 // protects connections map
}
```

### **Phase 2: Track Connections in Existing Methods**
Update existing connection methods to track relationships:

```go
// In ConnectWithFormat, add tracking:
func (e *Engine) ConnectWithFormat(...) error {
    // ... existing connection logic ...
    
    // Track the connection (non-breaking addition)
    e.trackConnection(destPtr, sourcePtr, toBus)
    
    return nil
}
```

### **Phase 3: Smart Per-Bus Implementation**
Use existing per-connection control as the "true" per-bus mechanism:

```go
// Enhanced SetMixerVolumeForBus that actually works per-bus
func (e *Engine) SetMixerVolumeForBus(mixerPtr unsafe.Pointer, volume float32, inputBus int) error {
    // Get sources connected to this mixer  
    sources := e.getTrackedSources(mixerPtr)
    
    // Try per-connection control if we have sources
    if len(sources) > inputBus && sources[inputBus] != nil {
        return e.SetConnectionVolume(sources[inputBus], mixerPtr, inputBus, volume)
    }
    
    // Fall back to global control  
    return e.setGlobalMixerVolume(mixerPtr, volume)
}
```

## 📊 **Key Insights from Code Analysis**

### **Memory Management Patterns**
From existing `.m` files:
- `@autoreleasepool` used for object creation/manipulation
- `malloc/free` for C structs and arrays
- `calloc` for zero-initialized arrays (like matrix gains)
- Helper C functions for complex array operations

### **Array Handling Examples**
Found in `matrixmixer_*` functions:
```objectivec
// Pattern: Allocate, use, free
Float32* gains = (Float32*)calloc(count, sizeof(Float32));
// ... use gains ...
free(gains);
```

### **Per-Connection Control Already Exists**
The `SetConnectionVolume/Pan` methods already provide true per-input control via `AVAudioMixingDestination`. This is actually the "correct" way to achieve per-bus control!

## 🎯 **Revised Recommendation**

### **Option A: Use Existing Per-Connection API** ✅ **RECOMMENDED**
Instead of complex native arrays, leverage the existing per-connection control:

```go
// This already works and provides true per-bus control:
func (e *Engine) SetMixerVolumeForBus(mixerPtr, volume, bus) error {
    // Find source connected to this bus (via connection tracking)
    sourcePtr := e.getSourceForBus(mixerPtr, bus)
    if sourcePtr != nil {
        // Use existing per-connection control
        return e.SetConnectionVolume(sourcePtr, mixerPtr, bus, volume)  
    }
    // Fall back to global
    return e.setGlobalMixerVolume(mixerPtr, volume)
}
```

### **Option B: Implement Complex Native Arrays** 
Continue with the smart native functions, but requires:
- Proper `@autoreleasepool` usage
- Complex Go→C array conversion  
- Memory management safeguards

## 🚀 **Next Steps**

**I recommend Option A** because:
1. **Leverages existing working code** - SetConnectionVolume already provides per-bus control
2. **Simple connection tracking** - Just need to track which source connects to which bus
3. **Safe implementation** - No complex memory management
4. **Professional result** - True per-bus control using industry-standard AVAudioMixingDestination

**Would you like me to implement Option A with simple connection tracking?**

This would give you true per-bus control without the complexity of native array handling, while maintaining all the safety and backward compatibility you need.

## 💡 **The Real Solution**

**The "per-bus" control you want already exists** - it's called **per-connection control**! Each source→mixer connection can have individual volume/pan via `SetConnectionVolume/Pan`. We just need to track which source connects to which bus, then use the existing per-connection API.

**Result**: True per-bus control using existing, tested functionality! 🎵
