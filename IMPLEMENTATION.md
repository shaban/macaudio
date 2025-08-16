# MacAudio Engine Implementation Specification

## Implementation Overview

This document provides detailed implementation specifications for the MacAudio engine based on the complete architecture specification. The implementation is driven by the architecture, not by existing code.

**Implementation Philosophy**: Build according to specification - existing code that doesn't align will be replaced or eliminated.

## Phase 1: Core Engine Implementation

### 1.1 Engine Core Structure

```go
// /engine/engine.go
package engine

import (
    "github.com/google/uuid"
    "github.com/shaban/macaudio/devices"
    "github.com/shaban/macaudio/plugins"
    "github.com/shaban/macaudio/avaudio/engine"
)

type Engine struct {
    ID          uuid.UUID                `json:"id"`
    Name        string                   `json:"name"`
    Spec        EngineSpec              `json:"spec"`
    Channels    map[uuid.UUID]Channel   `json:"channels"`
    Master      *MasterChannel          `json:"master"`
    
    // Runtime state (not serialized)
    avEngine    *engine.Engine          `json:"-"`
    dispatcher  *Dispatcher             `json:"-"`
    running     bool                    `json:"-"`
}

type EngineSpec struct {
    BufferSize int `json:"bufferSize"` // 64, 128, 256, 512, 1024 frames only
}

type Channel interface {
    ID() uuid.UUID
    Name() string
    Type() ChannelType
    
    // Lifecycle
    Initialize(avEngine *engine.Engine, dispatcher *Dispatcher) error
    Start() error
    Stop() error
    Release() error
    
    // Controls
    SetVolume(volume float32) error
    GetVolume() float32
    SetMute(mute bool) error
    IsMuted() bool
    
    // Serialization
    Serialize() ([]byte, error)
}

type ChannelType string
const (
    AudioInputChannelType ChannelType = "audio_input"
    MidiInputChannelType  ChannelType = "midi_input"
    PlaybackChannelType   ChannelType = "playback"
    AuxChannelType        ChannelType = "aux"
)
```

**📚 DOCUMENTATION REQUIREMENT**: Engine lifecycle and state management patterns - non-obvious threading model with dispatcher pattern.

### 1.2 Channel Implementation Hierarchy

```go
// /engine/channel/base.go
package channel

type BaseChannel struct {
    id          uuid.UUID
    name        string
    channelType ChannelType
    
    // Audio controls
    volume      float32
    mute        bool
    
    // AVFoundation integration
    avEngine    *engine.Engine
    dispatcher  *Dispatcher
    outputMixer unsafe.Pointer // AVAudioMixerNode for this channel
    
    // Routing
    auxSends    []AuxSend
    
    // State
    initialized bool
    running     bool
    mu          sync.RWMutex
}

type AuxSend struct {
    TargetAux uuid.UUID `json:"targetAux"`
    Level     float32   `json:"level"`     // 0.0-1.0
    PreFader  bool      `json:"preFader"` // before or after channel volume
}

func NewBaseChannel(id uuid.UUID, name string, channelType ChannelType) *BaseChannel {
    return &BaseChannel{
        id:          id,
        name:        name,
        channelType: channelType,
        volume:      1.0,
        mute:        false,
        auxSends:    make([]AuxSend, 0),
    }
}
```

### 1.3 AudioInputChannel Implementation

```go
// /engine/channel/audio_input.go
package channel

type AudioInputChannel struct {
    *BaseChannel
    
    // Device binding
    DeviceUID   string              `json:"deviceUID"`   // Apple's native device UID
    
    // Audio processing
    PluginChain *pluginchain.PluginChain `json:"pluginChain,omitempty"`
    Pan         float32             `json:"pan"`         // -1.0 to 1.0
    
    // Runtime state
    inputNode   unsafe.Pointer      `json:"-"` // AVAudioInputNode
    deviceOnline bool               `json:"-"`
}

func NewAudioInputChannel(name, deviceUID string) *AudioInputChannel {
    return &AudioInputChannel{
        BaseChannel: NewBaseChannel(uuid.New(), name, AudioInputChannelType),
        DeviceUID:   deviceUID,
        Pan:         0.0,
    }
}

func (c *AudioInputChannel) Initialize(avEngine *engine.Engine, dispatcher *Dispatcher) error {
    // Device validation against devices package using helper method
    audioDevices, err := devices.GetAudio()
    if err != nil {
        return fmt.Errorf("failed to get audio devices: %w", err)
    }
    
    device := audioDevices.ByUID(c.DeviceUID)  // Helper method needed in devices package
    if device == nil {
        return fmt.Errorf("audio device %s not found", c.DeviceUID)
    }
    
    c.deviceOnline = device.IsOnline
    if !c.deviceOnline {
        return fmt.Errorf("audio device %s is offline", c.DeviceUID)
    }
    
    // Implementation continues with AVAudioEngine setup...
    return nil
}
```

**📚 DOCUMENTATION REQUIREMENT**: Device UID binding and offline device handling - this is unintuitive behavior that needs clear documentation.

### 1.4 MidiInputChannel Implementation

```go
// /engine/channel/midi_input.go
package channel

type MidiInputChannel struct {
    *BaseChannel
    
    // MIDI device binding  
    DeviceUID   string              `json:"deviceUID"`   // MIDI device UID
    
    // Audio generation (MIDI→Audio)
    PluginChain *pluginchain.PluginChain `json:"pluginChain"` // MUST contain AU instrument
    Pan         float32             `json:"pan"`         // -1.0 to 1.0
    
    // Runtime state
    midiSynth   unsafe.Pointer      `json:"-"` // AVAudioUnitMIDISynth
    deviceOnline bool               `json:"-"`
}

func NewMidiInputChannel(name, deviceUID string) *MidiInputChannel {
    return &MidiInputChannel{
        BaseChannel: NewBaseChannel(uuid.New(), name, MidiInputChannelType),
        DeviceUID:   deviceUID,
        Pan:         0.0,
    }
}

func (c *MidiInputChannel) Initialize(avEngine *engine.Engine, dispatcher *Dispatcher) error {
    // Validate MIDI device exists and is online using helper method
    midiDevices, err := devices.GetMIDI()
    if err != nil {
        return fmt.Errorf("failed to get MIDI devices: %w", err)
    }
    
    device := midiDevices.ByUID(c.DeviceUID)  // Helper method needed in devices package
    if device == nil || !device.IsInput {
        return fmt.Errorf("MIDI input device %s not found", c.DeviceUID)
    }
    
    c.deviceOnline = device.IsOnline
    if !c.deviceOnline {
        return fmt.Errorf("MIDI device %s is offline", c.DeviceUID)
    }
    
    // 2. Create AVAudioUnitMIDISynth for audio generation
    // 3. Set up signal chain: MIDI Device → midiSynth → [PluginChain] → outputMixer
    // 4. Load default DLS soundbank or handle empty plugin chain
    
    return nil
}

func (c *MidiInputChannel) LoadSoundbank(dlsPath string) error {
    // Load DLS soundbank into AVAudioUnitMIDISynth
    // This is required for audio generation from MIDI
    return nil
}
```

**📚 DOCUMENTATION REQUIREMENT**: MIDI synthesis requires soundbank loading - this is a critical step that's often overlooked and will cause "no audio" issues.

### 1.5 PlaybackChannel Implementation (Enhanced)

```go
// /engine/channel/playback.go
package channel

type PlaybackChannel struct {
    *BaseChannel
    
    // File source
    FilePath    string      `json:"filePath"`
    
    // Playback controls
    // Note: No Pan - preserve stereo imaging
    
    // Time/Pitch manipulation (AVAudioEngine built-in)
    PlaybackRate float32    `json:"playbackRate"` // 0.5 = half speed, 2.0 = double
    PitchShift   float32    `json:"pitchShift"`   // -12 to +12 semitones
    
    // Loop support
    CanLoop     bool        `json:"canLoop"`
    
    // Metronome support (embedded)
    Metronome   *Metronome  `json:"metronome,omitempty"`
    
    // Runtime state
    playerNode  unsafe.Pointer `json:"-"` // AVAudioPlayerNode
    audioFile   unsafe.Pointer `json:"-"` // AVAudioFile
    timePitch   unsafe.Pointer `json:"-"` // AVAudioUnitTimePitch
}

type Metronome struct {
    BPM     int      `json:"bpm"`
    Ramping *Ramping `json:"ramping,omitempty"`
}

type Ramping struct {
    From    int  `json:"from"`    // Start BPM
    To      int  `json:"to"`      // Target BPM
    After   int  `json:"after"`   // After X bars
    AndBack bool `json:"andBack"` // Return to From after reaching To
}

func (c *PlaybackChannel) Initialize(avEngine *engine.Engine, dispatcher *Dispatcher) error {
    // 1. Load audio file from FilePath
    // 2. Create AVAudioPlayerNode
    // 3. Create AVAudioUnitTimePitch for time/pitch processing
    // 4. Set up signal chain: playerNode → timePitch → outputMixer
    // 5. Configure loop settings if CanLoop is true
    
    if c.FilePath == "" {
        return fmt.Errorf("file path is required for playback channel")
    }
    
    // Validate file exists and is readable
    if _, err := os.Stat(c.FilePath); os.IsNotExist(err) {
        return fmt.Errorf("audio file not found: %s", c.FilePath)
    }
    
    return nil
}

func (c *PlaybackChannel) Play() error {
    // Start playback with current settings
    return nil
}

func (c *PlaybackChannel) Pause() error {
    // Pause playback (can be resumed)
    return nil
}

func (c *PlaybackChannel) Stop() error {
    // Stop and reset to beginning
    return nil
}
```

**📚 DOCUMENTATION REQUIREMENT**: Asset-based signal generation approach and metronome implementation using audio files + pitch shift - this is a non-standard approach that needs explanation.

### 1.6 AuxChannel Implementation

```go
// /engine/channel/aux.go
package channel

type AuxChannel struct {
    *BaseChannel
    
    // Effects processing only
    PluginChain *pluginchain.PluginChain `json:"pluginChain,omitempty"`
    
    // Note: No Pan - receives pre-positioned sends
    // Note: No AuxSends - only routes to Master
    
    // Runtime state
    inputMixer  unsafe.Pointer `json:"-"` // AVAudioMixerNode for receiving sends
}

func NewAuxChannel(name string) *AuxChannel {
    return &AuxChannel{
        BaseChannel: NewBaseChannel(uuid.New(), name, AuxChannelType),
    }
}

func (c *AuxChannel) Initialize(avEngine *engine.Engine, dispatcher *Dispatcher) error {
    // 1. Create input mixer node for receiving aux sends from other channels
    // 2. Set up signal chain: inputMixer → [PluginChain] → outputMixer
    // 3. Connect outputMixer to master
    
    return nil
}

// Delete removes this aux channel and cleans up all references
func (c *AuxChannel) Delete(engine *Engine) error {
    // CRITICAL: Remove all AuxSend references from sending channels
    for _, channel := range engine.Channels {
        if sendingChannel, ok := channel.(AuxSendCapable); ok {
            sendingChannel.RemoveAuxSend(c.ID())
        }
    }
    
    // Then perform standard channel cleanup
    return c.Release()
}

// AuxSendCapable interface for channels that can send to aux
type AuxSendCapable interface {
    RemoveAuxSend(auxChannelID uuid.UUID)
}
```

### 1.7 MasterChannel Implementation

```go
// /engine/master/master.go  
package master

type MasterChannel struct {
    ID              uuid.UUID           `json:"id"`
    Volume          float32             `json:"volume"`          // 0.0-1.0
    Mute            bool                `json:"mute"`
    PluginChain     *pluginchain.PluginChain `json:"pluginChain,omitempty"`
    OutputDevice    OutputDevice        `json:"outputDevice"`
    MeteringEnabled bool                `json:"meteringEnabled"`
    
    // Runtime state
    avEngine        *engine.Engine      `json:"-"`
    mainMixerNode   unsafe.Pointer      `json:"-"` // AVAudioEngine.mainMixerNode
    outputNode      unsafe.Pointer      `json:"-"` // AVAudioEngine.outputNode
    meterTap        *tap.Tap           `json:"-"`
}

type OutputDevice struct {
    DeviceUID string `json:"deviceUID"` // Apple's native output device UID
}

func NewMasterChannel() *MasterChannel {
    return &MasterChannel{
        ID:              uuid.New(),
        Volume:          1.0,
        Mute:            false,
        MeteringEnabled: false,
    }
}

func (m *MasterChannel) Initialize(avEngine *engine.Engine) error {
    // 1. Get mainMixerNode from AVAudioEngine (automatically created)
    // 2. Get outputNode from AVAudioEngine
    // 3. Set up master plugin chain if present
    // 4. Configure output device
    // 5. Set up metering tap if enabled
    
    return nil
}

func (m *MasterChannel) EnableMetering(enable bool) error {
    // Install or remove tap on mainMixerNode for level monitoring
    return nil
}

func (m *MasterChannel) GetMeterLevel() (float64, error) {
    // Return current RMS level from meter tap
    if m.meterTap == nil {
        return 0.0, fmt.Errorf("metering not enabled")
    }
    
    metrics, err := m.meterTap.GetMetrics()
    if err != nil {
        return 0.0, err
    }
    
    return metrics.RMS, nil
}

// Delete is not allowed - MasterChannel cannot be deleted
func (m *MasterChannel) Delete() error {
    return fmt.Errorf("master channel cannot be deleted - only removed when entire engine is destroyed")
}
```

**📚 DOCUMENTATION REQUIREMENT**: Master channel metering and AVAudioEngine.mainMixerNode integration - this automatic node creation is AVFoundation-specific behavior.

## Phase 2: Device Integration Layer

### 2.1 Device Monitoring System

```go
// /engine/device/monitor.go
package device

import (
    "context"
    "sync"
    "time"
    "github.com/shaban/macaudio/devices"
)

type DeviceMonitor struct {
    pollInterval    time.Duration
    onDeviceChange  func(ChangeEvent)
    
    // Adaptive polling configuration
    baseInterval     time.Duration  // Base polling interval (50ms)
    maxInterval      time.Duration  // Max interval when no changes (200ms)
    currentInterval  time.Duration  // Current adaptive interval
    lastChangeTime   time.Time      // Last time devices changed
    noChangeCount    int            // Consecutive polls with no changes
    
    // Performance tracking
    averageCheckTime time.Duration  // Running average of check times
    maxCheckTime     time.Duration  // Maximum check time observed
    checkCount       int64          // Total number of checks performed
    targetCheckTime  time.Duration  // Target check time (50μs)
    
    lastAudioCount  int
    lastMidiCount   int
    
    ctx     context.Context
    cancel  context.CancelFunc
    mu      sync.RWMutex
}

type ChangeEvent struct {
    Type        ChangeType
    AudioCount  int
    MidiCount   int
    Timestamp   time.Time
}

type ChangeType string
const (
    DeviceAdded   ChangeType = "added"
    DeviceRemoved ChangeType = "removed"
)

func NewDeviceMonitor(pollInterval time.Duration, onChange func(ChangeEvent)) *DeviceMonitor {
    ctx, cancel := context.WithCancel(context.Background())
    return &DeviceMonitor{
        pollInterval:     pollInterval,
        baseInterval:     pollInterval,              // 50ms base
        maxInterval:      200 * time.Millisecond,   // 200ms max for power efficiency
        currentInterval:  pollInterval,
        targetCheckTime:  50 * time.Microsecond,    // 50μs target
        onDeviceChange:   onChange,
        ctx:             ctx,
        cancel:          cancel,
        lastChangeTime:  time.Now(),
    }
}

func (m *DeviceMonitor) Start() error {
    // Initialize baseline counts
    audioCount, midiCount, err := devices.GetDeviceCounts()
    if err != nil {
        return err
    }
    
    m.mu.Lock()
    m.lastAudioCount = audioCount
    m.lastMidiCount = midiCount
    m.mu.Unlock()
    
    // Start polling goroutine
    go m.pollLoop()
    
    return nil
}

func (m *DeviceMonitor) pollLoop() {
    // Use dynamic ticker that can adjust interval for adaptive polling
    currentInterval := m.pollInterval
    ticker := time.NewTicker(currentInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-m.ctx.Done():
            return
        case <-ticker.C:
            // Check if polling interval changed (adaptive behavior)
            m.mu.RLock()
            newInterval := m.currentInterval
            m.mu.RUnlock()
            
            // Reset ticker if interval changed
            if newInterval != currentInterval {
                ticker.Stop()
                ticker = time.NewTicker(newInterval)
                currentInterval = newInterval
            }
            
            m.checkForChanges()
        }
    }
}

func (m *DeviceMonitor) checkForChanges() {
    start := time.Now()
    
    audioCount, midiCount, err := devices.GetDeviceCounts()
    if err != nil {
        // Log error but continue polling
        return
    }
    
    // Update performance tracking
    elapsed := time.Since(start)
    m.updatePerformanceStats(elapsed)
    
    m.mu.Lock()
    audioChanged := audioCount != m.lastAudioCount
    midiChanged := midiCount != m.lastMidiCount
    m.mu.Unlock()
    
    if audioChanged || midiChanged {
        // Changes detected - reset to fast polling
        m.adaptiveSpeedup()
        
        changeType := DeviceAdded
        if audioCount < m.lastAudioCount || midiCount < m.lastMidiCount {
            changeType = DeviceRemoved
        }
        
        event := ChangeEvent{
            Type:       changeType,
            AudioCount: audioCount,
            MidiCount:  midiCount,
            Timestamp:  time.Now(),
        }
        
        // CRITICAL: This callback notifies the consuming app of device changes
        // The app can then check which specific devices changed and update UI
        if m.onDeviceChange != nil {
            m.onDeviceChange(event)
        }
        
        m.mu.Lock()
        m.lastAudioCount = audioCount
        m.lastMidiCount = midiCount
        m.mu.Unlock()
    } else {
        // No changes - gradually increase interval for power efficiency
        m.adaptiveSlowdown()
    }
}

// updatePerformanceStats tracks device check performance with exponential moving average
func (m *DeviceMonitor) updatePerformanceStats(elapsed time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.checkCount++
    
    // Update running average using exponential moving average (alpha = 0.1)
    if m.checkCount == 1 {
        m.averageCheckTime = elapsed
    } else {
        m.averageCheckTime = time.Duration(float64(m.averageCheckTime)*0.9 + float64(elapsed)*0.1)
    }
    
    // Track maximum observed check time
    if elapsed > m.maxCheckTime {
        m.maxCheckTime = elapsed
    }
    
    // Alert if we exceed target performance (50μs)
    if elapsed > m.targetCheckTime {
        // Log performance warning - device check exceeded target time
    }
}

// adaptiveSlowdown gradually increases polling interval when no changes detected
func (m *DeviceMonitor) adaptiveSlowdown() {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.noChangeCount++
    
    // After 10 consecutive checks with no changes, start slowing down
    if m.noChangeCount > 10 {
        // Gradually increase interval up to maxInterval (200ms)
        newInterval := time.Duration(float64(m.currentInterval) * 1.1)
        if newInterval > m.maxInterval {
            newInterval = m.maxInterval
        }
        m.currentInterval = newInterval
        m.pollInterval = newInterval
    }
}

// adaptiveSpeedup resets to fast polling when changes are detected
func (m *DeviceMonitor) adaptiveSpeedup() {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.noChangeCount = 0
    m.lastChangeTime = time.Now()
    m.currentInterval = m.baseInterval
    m.pollInterval = m.baseInterval
}
```

**📚 DOCUMENTATION REQUIREMENT**: Adaptive device polling (50ms→200ms) with performance monitoring and power efficiency optimization - this polling strategy directly impacts both system responsiveness and battery life.

**Performance Characteristics:**
- **Base Interval**: 50ms for responsive device change detection
- **Adaptive Range**: 50ms→200ms based on system stability  
- **Target Check Time**: 50μs per device enumeration call
- **CPU Usage**: ~0.024% when stable (200ms intervals)
- **Responsiveness**: Device changes detected within 50-200ms
- **Power Efficiency**: Automatically reduces polling when no changes detected

**Adaptive Behavior:**
- Starts at 50ms intervals for responsive detection
- After 10 consecutive checks with no changes, gradually increases interval
- Scales up to 200ms maximum for power efficiency when system is stable
- Immediately returns to 50ms when device changes are detected
- Maintains performance statistics for monitoring and debugging

### 2.2 Plugin Chain Management

```go
// /engine/pluginchain/chain.go
package pluginchain

type PluginChain struct {
    ID        uuid.UUID         `json:"id"`
    Name      string            `json:"name"`
    Instances []*PluginInstance `json:"instances"`
}

type PluginInstance struct {
    ID          uuid.UUID `json:"id"`
    *plugins.Plugin        `json:"plugin"` // Embedded from plugins package
    Bypassed    bool      `json:"bypassed"`
    IsInstalled bool      `json:"isInstalled"` // false on deserialization failure
    
    // Runtime state
    avUnit      unsafe.Pointer `json:"-"` // AVAudioUnit
}

func NewPluginChain(name string) *PluginChain {
    return &PluginChain{
        ID:        uuid.New(),
        Name:      name,
        Instances: make([]*PluginInstance, 0),
    }
}

func (pc *PluginChain) AddPlugin(pluginInfo plugins.PluginInfo) (*PluginInstance, error) {
    // 1. Introspect plugin to get full details
    plugin, err := pluginInfo.Introspect()
    if err != nil {
        return nil, fmt.Errorf("failed to introspect plugin: %w", err)
    }
    
    // 2. Create instance
    instance := &PluginInstance{
        ID:          uuid.New(),
        Plugin:      plugin,
        Bypassed:    false,
        IsInstalled: false, // Will be set to true after successful AVAudioUnit creation
    }
    
    // 3. Add to chain
    pc.Instances = append(pc.Instances, instance)
    
    return instance, nil
}

func (pc *PluginChain) InstallInEngine(avEngine *engine.Engine, dispatcher *Dispatcher) error {
    // 1. Create AVAudioUnit for each plugin instance
    // 2. Connect plugins in series
    // 3. Mark instances as installed
    // 4. Handle bypass state for each instance
    
    for _, instance := range pc.Instances {
        if err := pc.createAVUnit(instance); err != nil {
            // Mark as failed but continue with other plugins
            instance.IsInstalled = false
            continue
        }
        instance.IsInstalled = true
    }
    
    return nil
}

func (pc *PluginChain) createAVUnit(instance *PluginInstance) error {
    // Convert plugin info to AVAudioUnit
    // This requires mapping from plugins.Plugin to AudioComponentDescription
    return nil
}
```

**📚 DOCUMENTATION REQUIREMENT**: Plugin chain loading behavior and failure handling - IsInstalled field indicates deserialization failures that need clear documentation.

## Phase 3: Serialization and State Management

### 3.1 Engine Serialization

```go
// /engine/serialization.go
package engine

import (
    "encoding/json"
    "fmt"
    "time"
)

type SerializedEngine struct {
    Engine  *Engine   `json:"engine"`
    Version string    `json:"version"`
    Created time.Time `json:"created"`
    Updated time.Time `json:"updated"`
}

func (e *Engine) Serialize() ([]byte, error) {
    serialized := SerializedEngine{
        Engine:  e,
        Version: "1.0.0",
        Created: time.Now(), // This should be set once when engine is created
        Updated: time.Now(),
    }
    
    return json.MarshalIndent(serialized, "", "  ")
}

func DeserializeEngine(data []byte) (*Engine, error) {
    var serialized SerializedEngine
    if err := json.Unmarshal(data, &serialized); err != nil {
        return nil, fmt.Errorf("failed to deserialize engine: %w", err)
    }
    
    // Version compatibility check
    if serialized.Version != "1.0.0" {
        return nil, fmt.Errorf("unsupported engine version: %s", serialized.Version)
    }
    
    engine := serialized.Engine
    
    // Validate deserialized engine
    if err := validateEngine(engine); err != nil {
        return nil, fmt.Errorf("invalid engine state: %w", err)
    }
    
    return engine, nil
}

func validateEngine(e *Engine) error {
    // 1. Validate buffer size is supported
    validBufferSizes := []int{64, 128, 256, 512, 1024}
    valid := false
    for _, size := range validBufferSizes {
        if e.Spec.BufferSize == size {
            valid = true
            break
        }
    }
    if !valid {
        return fmt.Errorf("invalid buffer size: %d", e.Spec.BufferSize)
    }
    
    // 2. Validate all channels have valid UUIDs
    for uuid, channel := range e.Channels {
        if channel.ID() != uuid {
            return fmt.Errorf("channel UUID mismatch: map key %s != channel ID %s", uuid, channel.ID())
        }
    }
    
    // 3. Validate device UIDs exist (will mark channels as offline if not)
    // This is done during Initialize(), not during deserialization
    
    return nil
}
```

## Phase 4: Error Handling and Recovery

### 4.1 Error Types and Handling

```go
// /engine/errors.go
package engine

type EngineError struct {
    Type    ErrorType
    Message string
    Channel uuid.UUID // Optional - if error is channel-specific
    Device  string    // Optional - if error is device-specific
    Plugin  uuid.UUID // Optional - if error is plugin-specific
}

type ErrorType string
const (
    DeviceOfflineError    ErrorType = "device_offline"
    DeviceRemovedError    ErrorType = "device_removed"
    PluginLoadError       ErrorType = "plugin_load_failed"
    PluginCrashError      ErrorType = "plugin_crash"
    BufferUnderrunError   ErrorType = "buffer_underrun"
    EngineStartError      ErrorType = "engine_start_failed"
)

func (e EngineError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

type ErrorHandler interface {
    HandleError(EngineError) ErrorAction
}

type ErrorAction string
const (
    ContinueAction ErrorAction = "continue"    // Continue operation, log error
    StopChannel    ErrorAction = "stop_channel" // Stop specific channel
    StopEngine     ErrorAction = "stop_engine"  // Stop entire engine
    NotifyApp      ErrorAction = "notify_app"   // Notify consuming app
)

// Default error handler - consuming apps can provide their own
type DefaultErrorHandler struct {
    OnError func(EngineError)
}

func (h *DefaultErrorHandler) HandleError(err EngineError) ErrorAction {
    if h.OnError != nil {
        h.OnError(err)
    }
    
    switch err.Type {
    case DeviceOfflineError, DeviceRemovedError:
        return StopChannel
    case PluginLoadError:
        return ContinueAction // Continue without plugin
    case PluginCrashError:
        return ContinueAction // Plugin bypass automatically handled
    case EngineStartError:
        return StopEngine
    default:
        return NotifyApp
    }
}
```

**📚 DOCUMENTATION REQUIREMENT**: Error handling philosophy and default behaviors - the "library detects, app handles" principle needs clear explanation with examples.

## Phase 5: Integration Points

### 5.1 Dispatcher Pattern (Queue System)

```go
// /engine/dispatcher/dispatcher.go  
package dispatcher

type Dispatcher struct {
    queue      chan Operation
    avEngine   *engine.Engine
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

type Operation struct {
    Type     OperationType
    Params   map[string]interface{}
    Callback chan error
}

type OperationType string
const (
    AttachNode    OperationType = "attach_node"
    DetachNode    OperationType = "detach_node"
    ConnectNodes  OperationType = "connect_nodes"
    SetParameter  OperationType = "set_parameter"
    BypassPlugin  OperationType = "bypass_plugin"
)

func NewDispatcher(avEngine *engine.Engine) *Dispatcher {
    ctx, cancel := context.WithCancel(context.Background())
    return &Dispatcher{
        queue:    make(chan Operation, 100),
        avEngine: avEngine,
        ctx:      ctx,
        cancel:   cancel,
    }
}

func (d *Dispatcher) Start() {
    d.wg.Add(1)
    go d.processOperations()
}

func (d *Dispatcher) processOperations() {
    defer d.wg.Done()
    
    for {
        select {
        case <-d.ctx.Done():
            return
        case op := <-d.queue:
            err := d.executeOperation(op)
            if op.Callback != nil {
                op.Callback <- err
                close(op.Callback)
            }
        }
    }
}
```

**📚 DOCUMENTATION REQUIREMENT**: Dispatcher pattern for sub-300ms glitch-free topology changes - this serialization approach is critical for real-time audio.

## Implementation Priorities and Dependencies

### Phase 1 Dependencies
- ✅ `devices` package (complete)
- ✅ `plugins` package (complete) 
- ✅ `avaudio/engine` package (complete)
- 🔄 `avaudio/tap` package (needs integration)
- ❌ **Helper methods needed in devices package**: `AudioDevices.ByUID()` and `MidiDevices.ByUID()`
- ❌ New `engine` package structure
- ❌ Channel implementations with AuxSend cleanup
- ❌ Master channel (deletion protected)
- ❌ Serialization system

### Phase 2 Dependencies  
- ⚡ Device monitoring system
- ⚡ Plugin chain management
- ⚡ Error handling framework

### Phase 3 Dependencies
- ⚡ Dispatcher implementation
- ⚡ Integration testing
- ⚡ Performance validation

## Critical Documentation Requirements Summary

1. **Engine Lifecycle and Threading Model** - Non-obvious dispatcher pattern
2. **Device UID Binding and Offline Handling** - Unintuitive failure modes  
3. **MIDI Synthesis and Soundbank Loading** - Critical for audio generation
4. **Asset-Based Signal Generation** - Non-standard approach
5. **Master Channel and AVAudioEngine Integration** - Platform-specific behavior
6. **50ms Device Polling Performance** - Architecture-critical timing
7. **Plugin Chain Loading and Failure Handling** - Complex state management
8. **Error Handling Philosophy** - Clear responsibility boundaries
9. **Dispatcher Pattern for Real-Time Changes** - Critical for glitch-free operation

---

**Implementation Status**: Ready to begin Phase 1 development based on pure architecture specification.
