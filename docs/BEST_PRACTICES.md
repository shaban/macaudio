# AVAudioEngine Best Practices & Quirks

This document contains hard-learned lessons about AVAudioEngine's undocumented requirements and quirks that can save hours of debugging.

## 🔄 Engine Restart Requirements

### **ALWAYS Requires Engine Restart**
These operations **will not work** without restarting the engine:

```go
// TimePitch Effects
player.EnableTimePitchEffects()
engine.Stop()
time.Sleep(100 * time.Millisecond)  // Let engine settle
engine.Start()

// Audio Unit Changes  
player.AddEffect(effectUnit)
engine.Stop()
time.Sleep(100 * time.Millisecond)
engine.Start()

// Major Format Changes
engine.SetSampleRate(96000)  // From 48000
engine.Stop()
time.Sleep(100 * time.Millisecond)
engine.Start()
```

### **SOMETIMES Requires Engine Restart**
These may work without restart but are more reliable with restart:

- Output device changes (officially not required, but often needed)
- Large buffer size changes
- Complex routing topology changes
- Adding nodes to a running engine

### **NEVER Requires Engine Restart**
These work safely on a running engine:

- Volume/pan adjustments: `player.SetVolume(0.8)`
- Play/stop/pause operations: `player.Play()`  
- Real-time effect parameters: `eq.SetGain(band, gain)`
- Simple connect/disconnect operations

## ⚡ Connection State Prerequisites

### **Engine Startup Requirements**

AVAudioEngine **must** have a complete audio graph before starting:

```go
// ✅ CORRECT - Complete graph first, then start
engine := NewEngine()
player := engine.NewPlayer()
mixer := engine.MainMixerNode()
engine.Connect(player, mixer, 0, 0)  // Complete graph exists
engine.Start()  // Now safe - "inputNode != nullptr || outputNode != nullptr"

// ❌ WRONG - Incomplete graph
engine := NewEngine()
engine.Start()  // Will fail! No complete audio path
engine.Connect(player, mixer, 0, 0)  // Too late
```

### **Connection Order Matters**

Always follow this sequence:

1. **Create all nodes first**
2. **Connect in signal flow order**: Source → Effects → Destination  
3. **Start engine only after complete graph exists**

```go
// ✅ CORRECT Order
player := engine.NewPlayer()           // 1. Create nodes
timePitch := player.EnableTimePitch()  
mixer := engine.MainMixerNode()

engine.Connect(player, timePitch, 0, 0)    // 2. Connect signal flow
engine.Connect(timePitch, mixer, 0, 0)     
engine.Start()                             // 3. Start with complete graph
```

## 🔌 Node Type Behaviors

### **Players (Source Nodes)**
```go
// ✅ Players have OUTPUT buses only
playerPtr := player.GetNodePtr()
engine.Connect(playerPtr, mixer, 0, 0)  // Connect player OUTPUT to mixer INPUT

// ❌ NEVER do this - players don't have inputs!
engine.DisconnectNodeInput(playerPtr, 0)  // Will throw exception!
```

### **Mixers (Processing Nodes)**
```go
// ✅ Mixers have BOTH input and output buses
mixerPtr := engine.MainMixerNode()
engine.Connect(source, mixerPtr, 0, 0)        // Connect to mixer INPUT
engine.Connect(mixerPtr, output, 0, 0)        // Connect from mixer OUTPUT
engine.DisconnectNodeInput(mixerPtr, 0)       // Safe - mixer has inputs
```

### **Effects (Unit Nodes)**
```go
// ✅ Effects have BOTH input and output buses
effectPtr := player.GetTimePitchNodePtr()
engine.Connect(source, effectPtr, 0, 0)       // Connect to effect INPUT
engine.Connect(effectPtr, dest, 0, 0)         // Connect from effect OUTPUT
engine.DisconnectNodeInput(effectPtr, 0)      // Safe - effect has inputs
```

## 🧵 Threading Rules

### **Main Thread Required**
- Engine creation and destruction
- Node attachment/detachment  
- Complex routing changes
- Device enumeration

```go
// ✅ Main thread operations
dispatch_sync(dispatch_get_main_queue(), ^{
    engine := NewEngine()
    engine.AttachNode(customNode)
    engine.DetachNode(customNode)
    engine.Destroy()
});
```

### **Any Thread OK**
- Parameter changes (volume, pan, effects)
- Playback control (play/stop/pause)
- Simple queries (volume, status)

```go
// ✅ Background thread safe
go func() {
    player.SetVolume(0.8)
    player.Play()
    volume := player.GetVolume()
}()
```

### **Background Thread Recommended**
- File loading operations
- Long-running operations  
- Audio processing callbacks

## 🎵 Format Compatibility 

### **Sample Rate Handling**
```go
// ✅ BEST PRACTICE - Use engine's native format
engineFormat := engine.GetEngineFormat()  // Usually 48kHz
engine.ConnectWithFormat(source, dest, 0, 0, engineFormat)

// ⚠️ RISKY - Sample rate mismatches often work but with quality loss
// 44.1kHz file → 48kHz engine = automatic conversion (quality loss)
```

### **Channel Count Rules**
- **Mono → Stereo**: Usually works (duplicates to both channels)
- **Stereo → Mono**: Requires explicit handling
- **Multi-channel**: Very finicky, often requires exact matches

### **Bit Depth**
- Usually handled automatically by AVAudioEngine
- 24-bit can be problematic on some devices
- Stick to 16-bit or 32-bit float for maximum compatibility

## ⚠️ Common Exception Patterns

### **"Player Started When in Disconnected State"**
```go
// ❌ CAUSE - Wrong disconnect pattern
engine.DisconnectNodeInput(playerPtr, 0)  // Players don't have inputs!
player.Play()  // Exception!

// ✅ FIX - Don't disconnect player inputs
// Players are sources - only disconnect their destinations
engine.DisconnectNodeInput(mixerPtr, inputBus)  // Disconnect mixer's input instead
```

**Special Case: TimePitch Effects**
- TimePitch routing changes can trigger this warning even with correct connections
- The warning appears due to AVAudioEngine's internal state detection lag
- Audio functionality works correctly despite the warning
- Consider this a cosmetic issue, not a functional problem

```go
// This may produce warnings but works correctly:
player.EnableTimePitchEffects()
engine.Restart()  // Required after TimePitch changes
player.Play()     // May warn but plays correctly
```

### **"Required Condition is False"**
```go
// ❌ CAUSE - Node lifecycle mismatch  
engine.DetachNode(nodePtr)  
engine.Connect(nodePtr, other, 0, 0)  // Connecting detached node!

// ✅ FIX - Proper lifecycle management
engine.Connect(nodePtr, other, 0, 0)   // Connect first
engine.DetachNode(nodePtr)             // Then detach if needed
```

### **"InputNode != nullptr || OutputNode != nullptr"**
```go
// ❌ CAUSE - Starting engine with no complete audio paths
engine.Start()  // No connections exist yet!

// ✅ FIX - Ensure complete audio graph exists
engine.Connect(input, output, 0, 0)    // Create at least one complete path
engine.Start()                         // Now safe
```

## 🛠️ Safe Patterns We've Learned

### **TimePitch Effects Pattern**
```go
// ✅ SAFE TimePitch pattern
player.EnableTimePitchEffects()

// CRITICAL: Always restart after enabling TimePitch
engine.Stop()
time.Sleep(100 * time.Millisecond)  // Let AVAudioEngine settle
engine.Start()

player.SetPlaybackRate(0.8)  // Now works
player.SetPitch(200.0)       // Now works
player.Play()                // Now works
```

### **Connection Cleanup Pattern**
```go
// ✅ SAFE disconnection pattern
func SafeDisconnectPlayer(engine *Engine, playerPtr, mixerPtr unsafe.Pointer, inputBus int) {
    // Don't disconnect player inputs - they don't exist!
    // Instead, disconnect the mixer's input where the player is connected
    engine.DisconnectNodeInput(mixerPtr, inputBus)
}
```

### **Engine Lifecycle Pattern**
```go
// ✅ SAFE engine lifecycle
func SafeEngineSetup() *Engine {
    // 1. Create engine
    engine := NewEngine(DefaultSpec())
    
    // 2. Create and connect nodes BEFORE starting
    player := engine.NewPlayer()
    mixer := engine.MainMixerNode()
    engine.Connect(player.GetNodePtr(), mixer, 0, 0)
    
    // 3. Start only with complete graph
    engine.Start()
    
    return engine
}

func SafeEngineShutdown(engine *Engine) {
    // 1. Stop engine first
    engine.Stop()
    
    // 2. Brief pause for cleanup
    time.Sleep(50 * time.Millisecond)
    
    // 3. Destroy (automatically disconnects and detaches)
    engine.Destroy()
}
```

### **Format-Safe Connection Pattern**
```go
// ✅ SAFE format handling
func SafeConnect(engine *Engine, source, dest unsafe.Pointer, srcBus, destBus int) error {
    // Use engine's native format for consistency
    engineFormat, err := engine.GetEngineFormat()
    if err != nil {
        // Fallback to nil format (auto-conversion)
        return engine.ConnectWithFormat(source, dest, srcBus, destBus, nil)
    }
    defer engineFormat.Destroy()
    
    // Use explicit format for quality
    return engine.ConnectWithFormat(source, dest, srcBus, destBus, engineFormat.GetPtr())
}
```

## 🎯 Key Takeaways

1. **Always create complete audio graphs before starting the engine**
2. **Players are sources - they don't have input buses to disconnect**
3. **TimePitch and Audio Units always require engine restart**
4. **Use engine's native format for best compatibility**
5. **When in doubt, restart the engine - it's safer than fighting AVAudioEngine's state machine**

## 📖 Related Documentation

- [TimePitch Effects Guide](TIMEPITCH.md) - Detailed TimePitch usage
- [Engine Architecture](../specs/DOCUMENTATION.md) - Threading and lifecycle details
- [Format Handling](FORMAT_CONSOLIDATION.md) - Audio format best practices

---

*This guide is based on real-world experience with AVAudioEngine's undocumented behaviors. When Apple's documentation conflicts with this guide, trust this guide - it's based on what actually works in production.*
