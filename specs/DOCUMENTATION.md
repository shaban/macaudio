# MacAudio Engine Documentation Specification

## Documentation Strategy

This document identifies documentation requirements with far-reaching implications or non-intuitive behaviors that must be thoroughly documented to prevent common mistakes and architectural misunderstandings.

## Critical Documentation Requirements

### 1. Programmatic Engine Initialization ⚠️ CRITICAL

**Why Critical**: The programmatic initialization approach requires clear developer guidance to prevent common mistakes and ensure proper usage patterns.

**Documentation Scope**:
- Engine lifecycle states and validation
- Incremental channel creation workflow
- Meaningful error messages and resolution steps
- Serialization/deserialization at any initialization stage
- Common initialization patterns and best practices

**Example Issues Without Documentation**:
- Confusion about when engine can be started
- Not understanding why Start() fails with specific channels
- Improper usage of serialization/deserialization workflow
- Missing device validation steps

```markdown
# Programmatic Engine Initialization

MacAudio uses a **programmatic initialization** approach that allows building engines incrementally while providing specific validation feedback.

## Core Initialization Pattern

### Step 1: Create Engine with Minimal Configuration
```go
engine, err := macaudio.NewEngine(macaudio.EngineConfig{
    BufferSize:      512,                    // Required: 64, 128, 256, 512, or 1024
    OutputDeviceUID: "BuiltInSpeakerDevice", // Required: must be online
    ErrorHandler:    &MyErrorHandler{},     // Optional: defaults provided
})
if err != nil {
    // Handle configuration errors:
    // - Invalid buffer size
    // - Output device not found or offline
    // - Device enumeration failure
}
```

### Step 2: Add Channels Incrementally
```go
// Add audio input channel
audioChannel, err := engine.CreateAudioInputChannel("microphone", AudioInputConfig{
    DeviceUID: "USB-Audio-Interface",
    Volume:    0.8,
    Pan:       0.0,
})
if err != nil {
    // Handle channel creation errors:
    // - Device not found
    // - Device offline
    // - Invalid configuration
}

// Add MIDI input channel  
midiChannel, err := engine.CreateMidiInputChannel("piano", MidiInputConfig{
    DeviceUID: "Digital-Piano-MIDI",
})
if err != nil {
    // Handle MIDI channel errors
}

// Load soundbank for MIDI channel (REQUIRED for audio output)
err = midiChannel.LoadSoundbank("/System/Library/Audio/Sounds/Banks/default.dls")
if err != nil {
    // Handle soundbank loading errors
}
```

### Step 3: Start Engine with Validation
```go
if err := engine.Start(); err != nil {
    // Engine provides specific error guidance:
    switch {
    case strings.Contains(err.Error(), "no audio channels"):
        // Create at least one AudioInputChannel, MidiInputChannel, or PlaybackChannel
    case strings.Contains(err.Error(), "requires soundbank"):
        // Load soundbank for MIDI channels
    case strings.Contains(err.Error(), "device offline"):
        // Check device connections and online status
    case strings.Contains(err.Error(), "incomplete audio graph"):
        // Verify all channels are properly configured
    default:
        // Handle other initialization errors
    }
}
```

## Engine Lifecycle States

```go
type EngineInitState int
const (
    EngineCreated     // AVFoundation engine created, master channel ready
    MasterReady       // Master channel initialized (automatic)
    ChannelsReady     // At least one audio channel exists
    AudioGraphReady   // Complete audio path validated, ready to start
    EngineRunning     // AVFoundation engine started successfully
)
```

### State Transition Rules
- **EngineCreated → MasterReady**: Automatic (master channel created in NewEngine)
- **MasterReady → ChannelsReady**: When first audio channel is added
- **ChannelsReady → AudioGraphReady**: When all channels are ready (devices online, soundbanks loaded)
- **AudioGraphReady → EngineRunning**: When Start() is successfully called

## Common Initialization Patterns

### Simple Audio Input Setup
```go
engine, _ := macaudio.NewEngine(macaudio.EngineConfig{
    BufferSize:      256,
    OutputDeviceUID: "BuiltInSpeakerDevice",
})

audioChannel, _ := engine.CreateAudioInputChannel("mic", AudioInputConfig{
    DeviceUID: "BuiltInMicrophone",
})

// Ready to start - minimal audio path exists
engine.Start()
```

### MIDI Instrument Setup
```go
engine, _ := macaudio.NewEngine(config)

midiChannel, _ := engine.CreateMidiInputChannel("piano", MidiInputConfig{
    DeviceUID: "Digital-Piano",
})

// CRITICAL: Must load soundbank for audio output
midiChannel.LoadSoundbank("/path/to/piano.dls")

// Now ready to start
engine.Start()
```

### Multi-Channel Setup
```go
engine, _ := macaudio.NewEngine(config)

// Add multiple input sources
micChannel, _ := engine.CreateAudioInputChannel("microphone", audioConfig)
guitarChannel, _ := engine.CreateAudioInputChannel("guitar", guitarConfig)
pianoChannel, _ := engine.CreateMidiInputChannel("piano", pianoConfig)

// Add aux channel for reverb
reverbAux, _ := engine.CreateAuxChannel("reverb", auxConfig)

// Configure aux sends
micChannel.AddAuxSend(reverbAux.ID(), 0.3, false) // 30% post-fader send

// Load MIDI soundbank
pianoChannel.LoadSoundbank("/path/to/piano.dls")

// Start with complete setup
engine.Start()
```

## Serialization Workflow

### Save Engine State (Any Initialization Stage)
```go
// Can serialize at any point - even before Start()
engineData, err := engine.Serialize()
if err != nil {
    // Handle serialization error
}

// Save to file
err = os.WriteFile("session.json", engineData, 0644)
```

### Restore Engine State
```go
// Load from file
sessionData, err := os.ReadFile("session.json")
if err != nil {
    // Handle file error
}

// Deserialize engine
engine, err := macaudio.DeserializeEngine(sessionData)
if err != nil {
    // Handle deserialization errors:
    // - Invalid JSON format
    // - Unknown plugin references
    // - Invalid device UIDs
}

// Check if engine is ready to start
if engine.CanStart() {
    engine.Start()
} else {
    // Handle devices that went offline since serialization
    offlineChannels := engine.GetOfflineChannels()
    for _, channel := range offlineChannels {
        // Prompt user to reselect devices or disable channels
    }
}
```

## Error Handling During Initialization

### Device-Related Errors
```go
// Output device validation (in NewEngine)
if err := engine.ValidateOutputDevice(); err != nil {
    // Specific errors:
    // - "output device with UID 'xyz' not found"
    // - "output device 'xyz' is not online"
    
    // Resolution: Use devices.GetAudio() to list available devices
    devices, _ := devices.GetAudio()
    for _, device := range devices {
        if device.IsOnline {
            fmt.Printf("Available: %s (%s)\n", device.Name, device.UID)
        }
    }
}

// Input device validation (per channel)
if err := channel.ValidateInputDevice(); err != nil {
    // Handle offline input devices
    // User can choose to continue without this channel
}
```

### MIDI Channel Errors
```go
// MIDI channels require soundbank for audio output
if err := midiChannel.LoadSoundbank(path); err != nil {
    // Common errors:
    // - "soundbank file not found"
    // - "invalid soundbank format"
    // - "soundbank loading failed"
    
    // Provide fallback soundbank
    systemSoundbank := "/System/Library/Audio/Sounds/Banks/default.dls"
    midiChannel.LoadSoundbank(systemSoundbank)
}
```

### Plugin-Related Errors
```go
// Plugin loading during deserialization
if err := pluginChain.LoadPlugins(); err != nil {
    // Handle plugin failures:
    // - Plugin not installed
    // - Plugin version incompatible
    // - Plugin loading timeout
    
    // Get failed plugins
    failedPlugins := pluginChain.GetFailedPlugins()
    for _, plugin := range failedPlugins {
        fmt.Printf("Failed to load: %s (%s)\n", plugin.Name, plugin.Error)
        // Option to remove failed plugin or find replacement
    }
}
```

## Best Practices

### 1. Always Validate Configuration
```go
// Check configuration before engine creation
if err := macaudio.ValidateConfig(config); err != nil {
    // Fix configuration issues before NewEngine()
}
```

### 2. Handle Device Changes Gracefully
```go
// Set up device change notifications
engine.SetDeviceChangeHandler(func(event DeviceChangeEvent) {
    if event.Type == DeviceOffline {
        // Notify user, provide device reselection UI
    }
})
```

### 3. Provide User Feedback
```go
// Show initialization progress
engine.SetInitializationCallback(func(state EngineInitState, progress float32) {
    switch state {
    case ChannelsReady:
        fmt.Printf("Channels created: %.0f%%\n", progress*100)
    case AudioGraphReady:
        fmt.Printf("Audio graph validated: %.0f%%\n", progress*100)
    }
})
```

### 4. Graceful Degradation
```go
// Continue with partial setup if some devices are offline
engine.SetIgnoreOfflineDevices(true)

// Start engine even if some channels fail
if err := engine.Start(); err != nil {
    // Check if any channels are working
    workingChannels := engine.GetWorkingChannels()
    if len(workingChannels) > 0 {
        fmt.Printf("Started with %d working channels\n", len(workingChannels))
    }
}
```
```

### 1. Engine Lifecycle and AVFoundation Startup Sequence ⚠️ CRITICAL

**Why Critical**: The AVFoundation engine startup sequence is non-obvious and critical for preventing runtime crashes.

**Documentation Scope**:
- Exact startup sequence: Prepare → Initialize Channels → Create Basic Routing → Start
- AVAudioEngine requirements: "inputNode != nullptr || outputNode != nullptr"
- Node sharing strategy for device efficiency
- When to use direct calls vs. dispatcher queue
- **AuxSend cleanup when deleting AuxChannels**

**Example Issues Without Documentation**:
- Engine startup failures with "inputNode != nullptr || outputNode != nullptr"
- Memory leaks from unshared input nodes
- Race conditions in AuxChannel deletion
- Audio glitches from unserialized topology changes

```markdown
# Engine Lifecycle and AVFoundation Integration

## Critical Startup Sequence

MacAudio requires a specific startup sequence to satisfy AVFoundation requirements:

### Phase 1: AVFoundation Engine Creation
```go
// ✅ CORRECT - Create but don't start yet
avEngine, err := engine.New(audioSpec)
avEngine.Prepare() // Prepare resources but don't start audio
```

### Phase 2: Master Channel Initialization
```go
// ✅ CORRECT - Initialize master channel first
engine.Master.Initialize(avEngine) // Creates mainMixer → output connection
```

### Phase 3: Channel Initialization
```go
// ✅ CORRECT - Initialize all channels
for _, channel := range engine.Channels {
    channel.Initialize(avEngine, dispatcher) // Creates input → mixer connections
}
```

### Phase 4: Basic Routing (if needed)
```go
// ✅ CORRECT - Ensure minimum connections exist
if onlyMasterChannelExists {
    engine.createBasicRouting() // input → mainMixer connection
}
```

### Phase 5: AVFoundation Engine Start
```go
// ✅ CORRECT - Start only after complete audio graph
avEngine.Start() // Now safe - all connections exist
```

## Node Sharing Strategy

Multiple AudioInputChannels using the same device share input nodes:

```go
// ✅ CORRECT - Shared input nodes
inputNode := engine.getOrCreateInputNode("device-uid", 0)
// Multiple channels can use the same inputNode safely
```

## Why This Sequence Matters:
1. **AVFoundation Requirement**: Engine needs complete audio graph before starting
2. **Resource Efficiency**: Input nodes are shared among channels
3. **Thread Safety**: All topology changes are serialized through dispatcher
4. **Error Prevention**: Prevents "no input/output nodes" runtime crashes
```

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
