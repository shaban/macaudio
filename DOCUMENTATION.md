# MacAudio Engine Documentation Specification

## Documentation Strategy

This document identifies documentation requirements with far-reaching implications or non-intuitive behaviors that must be thoroughly documented to prevent common mistakes and architectural misunderstandings.

## Critical Documentation Requirements

### 1. Engine Lifecycle and Threading Model ⚠️ CRITICAL

**Why Critical**: The dispatcher pattern for audio engine operations is non-obvious and critical for real-time performance.

**Documentation Scope**:
- AVAudioEngine thread-safety requirements  
- Dispatcher queue serialization for topology changes
- Sub-300ms glitch-free operation guarantees
- When to use direct calls vs. dispatcher queue
- **AuxSend cleanup when deleting AuxChannels**

**Example Issues Without Documentation**:
- Deadlocks from calling AVAudioEngine methods on wrong thread
- Audio glitches from unserialized topology changes
- Race conditions in plugin bypass operations

```markdown
# Engine Threading Model

MacAudio uses a strict threading model to ensure real-time audio performance:

## Core Principle: Serialized Topology Changes
ALL topology changes (attach/detach nodes, connect/disconnect, plugin bypass) 
MUST be queued through the Dispatcher to maintain sub-300ms glitch-free operation.

## Direct vs Dispatched Operations

### Use Direct Calls For:
- Parameter changes on existing plugins
- Volume/pan/mute controls  
- Metering data retrieval
- State queries (IsMuted, GetVolume, etc.)

### Use Dispatcher Queue For:
- Attaching/detaching nodes
- Connecting/disconnecting audio paths
- Plugin bypass/enable operations
- Device changes
- Plugin chain modifications
- **Channel deletion (includes AuxSend cleanup)**

## Example:
```go
// ❌ WRONG - Will cause audio glitches
engine.Attach(pluginNode)
engine.Connect(inputNode, pluginNode, 0, 0)

// ✅ CORRECT - Queued for glitch-free execution  
dispatcher.QueueAttach(pluginNode)
dispatcher.QueueConnect(inputNode, pluginNode, 0, 0)
```

## AuxSend Cleanup Requirement:
```go
// When deleting an AuxChannel, ALL AuxSends must be removed first
// ❌ WRONG - Will leave dangling references
engine.DeleteChannel(auxChannelID)

// ✅ CORRECT - AuxChannel.Delete() handles cleanup automatically
auxChannel.Delete(engine) // Removes all AuxSend references from sending channels
```
```

### 2. Device UID Binding and Offline Device Handling ⚠️ CRITICAL

**Why Critical**: Device failures are common and the offline device behavior is unintuitive.

**Documentation Scope**:
- Apple's device UID persistence across reboots
- Device offline detection and channel behavior
- No automatic device switching policy
- Fast-path polling implementation details

**Example Issues Without Documentation**:
- Confusion about why channels stop working when devices are unplugged
- Expectation of automatic device switching
- Not understanding device UID persistence

```markdown
# Device Management and Failure Handling

## Device UID Binding
MacAudio uses Apple's native device UIDs for persistent device identification:
- UIDs persist across system reboots
- UIDs remain valid even when device is temporarily offline
- UIDs are tied to specific hardware, not device names

## Offline Device Behavior
When a bound device goes offline:
1. Channel immediately stops processing audio
2. Channel.Initialize() returns error on next engine start
3. No automatic switching to alternative devices
4. Consuming app is notified via error callback

## Device Monitoring
- **Adaptive polling**: Starts at 50ms intervals, scales to 200ms when stable
- **Performance target**: ~50μs execution time per poll (achieved ~48μs average)  
- **CPU efficiency**: ~0.024% CPU usage when stable (200ms intervals)
- **Power optimization**: Automatically reduces polling frequency during idle periods
- **Responsiveness**: Device changes detected within 50-200ms for excellent UX

## Adaptive Polling Behavior:
1. **Fast Response Phase**: 50ms intervals when system is active or changes detected
2. **Stability Detection**: After 10 consecutive polls with no changes  
3. **Power Efficient Phase**: Gradually increases interval up to 200ms maximum
4. **Change Detection**: Immediately returns to 50ms on any device count change
5. **Performance Monitoring**: Tracks average/max check times with alerts for >50μs

## Device Monitoring Configuration:
```go
type DeviceMonitorConfig struct {
    BaseInterval   time.Duration  // 50ms - responsive polling when active
    MaxInterval    time.Duration  // 200ms - power-efficient polling when stable  
    TargetTime     time.Duration  // 50μs - performance target per check
    DebounceCount  int           // 10 - polls before slowing down
}
```

## Fast-path Implementation Details:
- Uses `devices.GetDeviceCounts()` for O(1) change detection
- Full device enumeration (`devices.GetAudio()`) only triggered on count changes
- Exponential moving average (α=0.1) for performance tracking
- Thread-safe statistics collection for monitoring and debugging

## Device Change Notification Mechanism:
```go
monitor := device.NewDeviceMonitor(50*time.Millisecond, func(event ChangeEvent) {
    // This callback is invoked when device counts change
    if event.Type == device.DeviceRemoved {
        // Check which specific devices went offline
        currentDevices, _ := devices.GetAudio()
        for channelID, channel := range engine.Channels {
            if audioChannel, ok := channel.(*AudioInputChannel); ok {
                deviceStillExists := currentDevices.ByUID(audioChannel.DeviceUID) != nil
                if !deviceStillExists {
                    // Device removed - notify UI, stop channel, etc.
                    handleDeviceRemoval(channelID, audioChannel.DeviceUID)
                }
            }
        }
    }
})
monitor.Start()
```

## Why No Automatic Switching?
- Security: Prevents unauthorized device access
- Privacy: User explicitly chooses audio devices
- Predictability: No surprising device changes during sessions
```

### 3. MIDI Synthesis and Soundbank Requirements ⚠️ CRITICAL

**Why Critical**: MIDI channels produce no audio without proper soundbank loading - common beginner mistake.

**Documentation Scope**:
- AVAudioUnitMIDISynth requires DLS soundbank
- Default soundbank loading strategies  
- Why MIDI channels might be silent
- DLS file format and sourcing

**Example Issues Without Documentation**:
- "MIDI channel not working" - actually needs soundbank
- Confusion about MIDI vs. audio generation
- Not understanding DLS soundbank requirements

```markdown
# MIDI Input Channels and Audio Generation

## Critical Requirement: Soundbank Loading
MIDI channels REQUIRE a loaded soundbank to generate audio:

```go
midiChannel := NewMidiInputChannel("Piano", midiDeviceUID)
// ❌ This will be SILENT - no soundbank loaded

// ✅ Load soundbank for audio generation
err := midiChannel.LoadSoundbank("/path/to/soundbank.dls")
```

## Why MIDI Channels Can Be Silent:
1. **No Soundbank Loaded** (most common)
2. Plugin chain is empty (valid for advanced users)  
3. MIDI device is offline
4. MIDI device not sending data

## Soundbank Options:
- System DLS files (macOS includes basic soundbank)
- Third-party DLS/SF2 files
- Plugin-based synthesis (AU instruments in chain)

## DLS vs. Plugin Synthesis:
- DLS: Quick setup, basic sounds, low CPU
- Plugins: High quality, CPU intensive, complex setup
```

### 4. Asset-Based Signal Generation Approach ⚠️ NON-STANDARD

**Why Critical**: This approach differs from typical DAW signal generation and needs explanation.

**Documentation Scope**:
- Why audio files instead of real-time synthesis
- Test tone generation strategy (440Hz + pitch shift)
- Metronome implementation using audio files
- Performance and reliability benefits

```markdown
# Signal Generation: Asset-Based Approach

MacAudio uses pre-recorded audio files with real-time processing instead of 
mathematical signal generation. This is a deliberate architectural choice.

## Why Audio Files Instead of Math?
- **Reliability**: No synthesis bugs or edge cases
- **Performance**: AVAudioEngine optimized for file playback
- **Quality**: Professional test tones and metronome sounds
- **Simplicity**: Eliminates complex DSP code

## Test Tone Implementation:
```go
// Instead of: sin(2π * frequency * time)
testChannel := NewPlaybackChannel("Test Tone")
testChannel.FilePath = "/path/to/440hz_sine.wav"
testChannel.PitchShift = 12  // +1 octave = 880Hz
```

## Metronome Implementation:
```go
metronomeChannel := NewPlaybackChannel("Metronome")  
metronomeChannel.FilePath = "/path/to/click.wav"
metronomeChannel.CanLoop = true
metronomeChannel.Metronome = &Metronome{BPM: 120}
// Tempo changes via pitch shift for audio cues
```

## Asset Requirements:
- 440Hz sine wave (reference tone)
- High-quality click sound (metronome)
- Standard sample rates (44.1kHz, 48kHz)
- Mono files preferred for processing efficiency
```

### 5. Plugin Chain Loading and Failure Resilience ⚠️ COMPLEX

**Why Critical**: Plugin loading can fail silently and the IsInstalled field behavior is unintuitive.

**Documentation Scope**:
- Plugin loading failure scenarios  
- IsInstalled field meaning and usage
- Graceful degradation strategies
- Parameter persistence across failures

```markdown
# Plugin Chain Management and Failure Handling

## Plugin Loading Failure Scenarios:
1. Plugin files deleted/moved since serialization
2. Plugin version incompatibilities  
3. System architecture changes (Intel → Apple Silicon)
4. AudioUnit instantiation timeouts
5. Corrupted plugin state

## IsInstalled Field Behavior:
```go
type PluginInstance struct {
    ID          uuid.UUID
    Plugin      *plugins.Plugin  // Always contains plugin metadata
    IsInstalled bool            // Runtime state: AVAudioUnit creation success
    Bypassed    bool            // User control: bypass even if installed
}
```

- `IsInstalled = false`: Plugin failed to load, no audio processing
- `IsInstalled = true, Bypassed = true`: Plugin loaded but bypassed  
- `IsInstalled = true, Bypassed = false`: Plugin active and processing

## Graceful Degradation:
Engine continues running even when plugins fail to load:
- Failed plugins are skipped in signal chain
- Parameter values are preserved for future sessions
- User interface can show failed plugins with "reload" option
- Audio flows through chain bypassing failed plugins

## Best Practices:
- Check IsInstalled before UI parameter controls
- **Provide visual indication of failed plugins** (grayed out, warning icon, "Failed to Load" status)
- **Show IsInstalled=false state prominently** so users can identify and resolve plugin issues
- Provide "reload plugin" functionality  
- **Offer plugin replacement/deletion options** for persistently failed plugins
- Save plugin state frequently
- Test plugin chains after major system updates
- **Include plugin health indicators** in session management UI
```

### 6. Master Channel and AVAudioEngine Integration ⚠️ PLATFORM-SPECIFIC

**Why Critical**: AVAudioEngine automatically creates mainMixerNode but behavior is platform-specific, and MasterChannel deletion must be prevented.

**Documentation Scope**:
- Automatic mainMixerNode creation by AVAudioEngine
- Metering enablement requirements
- Output device selection implications
- Master bus effects routing
- **MasterChannel deletion protection**

```markdown
# Master Channel and AVAudioEngine Integration

## Automatic Node Creation:
AVAudioEngine automatically creates these nodes:
- `mainMixerNode`: Receives all channel outputs
- `outputNode`: Connects to system audio output
- Connection: `mainMixerNode → outputNode` (automatic)

```go
// These nodes are created automatically:
engine := engine.New()
mainMixer := engine.MainMixerNode() // Already exists
outputNode := engine.OutputNode()   // Already exists
```

## Metering Setup:
Metering is NOT enabled by default:
```go
// ❌ This returns 0.0 - no metering enabled
level := masterChannel.GetMeterLevel()

// ✅ Enable metering first
masterChannel.EnableMetering(true)
level := masterChannel.GetMeterLevel() // Now returns actual levels
```

## Master Bus Effects:
Plugin chains on master affect ALL audio:
- Insert effects: EQ, compression, limiting  
- No send effects on master (would create feedback)
- Processing order: channels → master plugins → output

## Output Device Binding:
- Master channel controls which physical output is used
- Changes require engine restart (AVAudioEngine limitation)
- Device changes should be queued through dispatcher

## Master Channel Protection:
```go
// ❌ This will fail - master cannot be deleted
err := engine.DeleteChannel(masterChannel.ID) 
// Returns: "master channel cannot be deleted - only removed when entire engine is destroyed"

// ✅ Master channel lifecycle tied to engine
engine.Destroy() // This destroys master channel along with entire engine
```
```

### 7. Error Handling Philosophy and Boundaries ⚠️ ARCHITECTURAL

**Why Critical**: The "library detects, app handles" philosophy needs clear explanation to prevent architectural mistakes.

**Documentation Scope**:
- Library vs. application responsibility boundaries
- Error detection capabilities and limitations  
- Recovery strategy recommendations
- Callback-based error reporting

```markdown
# Error Handling Philosophy

## Core Principle: Library Detects, App Handles
MacAudio detects errors but does NOT automatically recover. This is intentional.

## Library Responsibilities:
- Detect device offline/removal events
- Identify plugin loading failures  
- Report buffer underruns and performance issues
- Validate configuration parameters
- Provide error callbacks to application

## Application Responsibilities:
- Decide recovery strategies (retry, switch devices, notify user)
- Implement UI for error states and user choices
- Handle device selection and reselection
- Manage plugin reload attempts
- Save/restore engine state

## Why This Division?
- **User Agency**: Only user should decide device switching
- **UI Context**: App knows best how to present errors to user
- **Policy Flexibility**: Different apps need different recovery behaviors
- **Security**: Automatic device switching could be privacy violation

## Example Error Handling:
```go
engine.SetErrorHandler(&MyErrorHandler{
    OnDeviceOffline: func(channelID uuid.UUID, deviceUID string) {
        // Show user dialog: "Microphone disconnected. Select new device?"
        // App provides device selection UI
        // App calls engine.RebindDevice(channelID, newDeviceUID)
    },
    OnPluginCrash: func(pluginID uuid.UUID) {
        // Log error, continue without plugin
        // Optionally show notification to user
        // Do NOT automatically restart plugin
    },
})
```

## Error Recovery Patterns:
1. **Device Issues**: Stop processing, notify user, await device reselection
2. **Plugin Issues**: Continue without plugin, log error, allow manual retry
3. **Performance Issues**: Log warning, continue operation, suggest buffer size increase
4. **Configuration Issues**: Reject invalid config, provide specific error message
```

## Documentation Structure

### Primary Documentation Files

1. **`README.md`** - Quick start and overview
2. **`ARCHITECTURE.md`** - Complete architecture specification (already exists)
3. **`IMPLEMENTATION.md`** - Implementation specification (already exists)
4. **`THREADING.md`** - Threading model and dispatcher pattern details
5. **`DEVICES.md`** - Device management and failure handling
6. **`PLUGINS.md`** - Plugin system and chain management
7. **`ERRORS.md`** - Error handling philosophy and examples
8. **`EXAMPLES/`** - Working code examples for common scenarios

### API Documentation Requirements

Each public type and method must include:
- **Purpose**: What it does
- **Thread Safety**: Safe to call from any thread? Requires dispatcher?
- **Error Conditions**: When it fails and why
- **Performance**: CPU/memory implications
- **Dependencies**: What must be initialized first
- **Examples**: Working code samples

### Example Code Requirements

Must include working examples for:
- Basic engine setup and teardown
- Each channel type configuration
- Device binding and failure recovery
- Plugin chain creation and management
- Serialization and deserialization
- Error handling patterns
- Performance monitoring
- Threading model usage

## Documentation Quality Standards

### 1. Accuracy Requirements
- All examples must compile and run
- Performance claims must be measurable
- Thread safety claims must be tested
- Error conditions must be reproducible

### 2. Clarity Requirements  
- No assumed knowledge of AVFoundation internals
- Step-by-step procedures for complex operations
- Clear distinction between library and app responsibilities
- Explicit documentation of unintuitive behaviors

### 3. Completeness Requirements
- Cover all public APIs
- Document all error types and recovery patterns
- Explain all configuration options and their implications
- Provide migration guides for version changes

---

**Documentation Priority**: Focus on the 7 critical areas identified above - these prevent the most common architectural mistakes and integration problems.
