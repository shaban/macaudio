# MacAudio Engine MVP Design Document

**Version**: 1.1  
**Date**: August 21, 2025  
**Status**: Implemented - Clean API Architecture

## Architecture Overview

MacAudio provides a streamlined 8-channel audio mixing engine built on AVAudioEngine, designed for simplicity and performance. The architecture eliminates complex send/return routing in favor of clean, direct signal paths suitable for live performance, recording, and basic production.

## Core Components

### 1. Engine
- **Purpose**: Central coordinator managing parameter tree, routing, and lifecycle
- **Channel Management**: Fixed-length array[8] of channels (not slice)
- **Channel Allocation**: Automatic bus assignment (array index = mixer bus)
- **Master Controls**: Master volume only (no master pan)
- **State Serialization**: Engine IS the parameter tree - direct JSON serialization
- **Device Management**: Optional input/output device assignments

### 2. Unified Channel Architecture
- **Channel Types**: Input or Playback (determined by presence of InputOptions/PlaybackOptions)
- **Base Properties**: Volume, Pan, BusIndex, Type
- **Type-Specific Options**: Optional InputOptions or PlaybackOptions (nil when not applicable)
- **Clean Design**: Eliminates interface complexity with clear struct composition

#### Input Channel (Audio Device Input)
- **Source**: Audio device channel (mono)  
- **Configuration**: InputOptions containing Device, ChannelIndex, PluginChain
- **Plugin Chain**: Up to 8 AudioUnit effects in series with individual bypass
- **Device Selection**: Via devices library with capability validation
- **Format Handling**: Auto-conversion via AVAudioEngine (non-nil format)
- **Zero-Latency Monitoring**: Supported via hardware capabilities + high sample rate/low buffer

#### Playback Channel (Audio File)
- **Source**: Audio file (stereo)
- **Configuration**: PlaybackOptions containing FilePath, Rate, Pitch
- **TimePitch Unit**: Built-in with performance priority
- **Rate Range**: 0.25x to 1.25x
- **Pitch Range**: ±12 semitones  
- **File Formats**: Full AVAudioEngine supported formats only (no shallow support)
- **Loading Strategy**: Whole file to memory
- **File Size Limit**: 200MB threshold (audiophile album size)

### 3. Enhanced Plugin Management
- **Plugin Architecture**: EnginePlugin with embedded *plugins.Plugin pointer
- **Instance Separation**: Automatic parameter isolation through Go value copying semantics
- **Parameters**: Direct access to plugins.Parameter slice with CurrentValue fields
- **Installation State**: IsInstalled flag for persistence handling when plugins are unavailable
- **Type Safety**: Parameter API uses proper float32 types with bounds validation
- **Bypass Control**: Individual plugin bypass per EnginePlugin instance

### 4. Main Mixing Node
- **Input Buses**: 8 buses for channel routing
- **Per-Bus Controls**: Independent volume/pan per input bus
- **Output**: Direct to speaker/audio device (no send/return)

## Channel Architecture

### Unified Channel Structure
```go
type Channel struct {
    // Base channel properties
    BusIndex int     `json:"busIndex"`
    Volume   float32 `json:"volume"`
    Pan      float32 `json:"pan"`

    // Optional type-specific data (nil when not applicable)
    PlaybackOptions *PlaybackOptions `json:"playbackOptions,omitempty"`
    InputOptions    *InputOptions    `json:"inputOptions,omitempty"`
}

// Channel type detection methods
func (c *Channel) IsInput() bool {
    return c.InputOptions != nil
}

func (c *Channel) IsPlayback() bool {
    return c.PlaybackOptions != nil
}

type PlaybackOptions struct {
    FilePath string  `json:"filePath"`
    Rate     float32 `json:"rate"`  // 0.25x to 1.25x
    Pitch    float32 `json:"pitch"` // ±12 semitones
}

type InputOptions struct {
    Device       *devices.AudioDevice `json:"device"`       // Complete device info with capabilities
    ChannelIndex int                  `json:"channelIndex"`
    PluginChain  *PluginChain         `json:"pluginChain"`
}
```

### Plugin Instance Architecture
```go
type EnginePlugin struct {
    IsInstalled bool            `json:"isInstalled"` // false when plugin unavailable
    *plugins.Plugin             `json:"plugin"`      // embedded with independent parameters
    Bypassed    bool            `json:"bypassed"`    // Individual bypass control
}
```

### Engine Structure
```go
type Engine struct {
    Channels     [8]*Channel `json:"channels"`  // Fixed array, not slice
    MasterVolume float32     `json:"masterVolume"`
    
    // Device assignments
    InputDevice  *devices.AudioDevice `json:"inputDevice,omitempty"`
    OutputDevice *devices.AudioDevice `json:"outputDevice,omitempty"`
    // ...
}
```

### Channel Types & Signal Flow
- **Input Channel**: Has InputOptions (PlaybackOptions = nil) - detected via `channel.IsInput()`
- **Playback Channel**: Has PlaybackOptions (InputOptions = nil) - detected via `channel.IsPlayback()`  
- **Unallocated**: nil channel slot

### Channel Type Detection
Channel type is determined by the presence of options structs, eliminating redundant type fields:
```go
// Clean type detection - single source of truth
if channel.IsInput() {
    // Handle input channel - InputOptions guaranteed to be non-nil
    device := channel.InputOptions.Device
    pluginChain := channel.InputOptions.PluginChain
}

if channel.IsPlayback() {
    // Handle playback channel - PlaybackOptions guaranteed to be non-nil
    filePath := channel.PlaybackOptions.FilePath
    rate := channel.PlaybackOptions.Rate
}
```

### Signal Flow
```
Input Channel:    Device -> Plugin Chain -> MainMixer[Bus0-7] -> Speaker
Playback Channel: File -> TimePitch -> MainMixer[Bus0-7] -> Speaker
```

### Plugin Instance Separation
- **Introspection**: Each plugins.Introspect() returns independent *Plugin pointer
- **Parameter Isolation**: Parameters []Parameter slice copied by value
- **Current Values**: Each parameter has independent CurrentValue field
- **Persistence**: IsInstalled flag handles unavailable plugins gracefully

## Parameter Tree Structure

### Direct Engine Serialization (Engine IS the Parameter Tree)
```json
{
  "channels": [
    {
      "busIndex": 0,
      "volume": 0.8,
      "pan": -0.3,
      "inputOptions": {
        "device": {
          "uid": "BuiltInMicrophoneDevice",
          "name": "MacBook Pro Microphone",
          "channels": [
            {
              "channelNumber": 1,
              "channelName": "Channel 1",
              "channelLabel": "Left"
            }
          ],
          "sampleRates": [44100, 48000, 88200, 96000],
          "manufacturer": "Apple Inc."
        },
        "channelIndex": 0,
        "pluginChain": {
          "plugins": [
            {
              "isInstalled": true,
              "bypassed": false,
              "plugin": {
                "type": "aufx",
                "subtype": "comp", 
                "manufacturer": "appl",
                "name": "Compressor",
                "parameters": [
                  {
                    "identifier": "threshold",
                    "displayName": "Threshold",
                    "currentValue": -18.0,
                    "minValue": -40.0,
                    "maxValue": 0.0,
                    "isWritable": true
                  },
                  {
                    "identifier": "ratio", 
                    "displayName": "Ratio",
                    "currentValue": 4.0,
                    "minValue": 1.0,
                    "maxValue": 20.0,
                    "isWritable": true
                  }
                ]
              }
            }
          ]
        }
      }
    },
    {
      "busIndex": 1, 
      "volume": 0.7,
      "pan": 0.1,
      "playbackOptions": {
        "filePath": "/path/to/audio.m4a",
        "rate": 1.0,
        "pitch": 0.0
      }
    },
    null, null, null, null, null, null
  ],
  "masterVolume": 0.85,
  "sampleRate": 48000,
  "bufferSize": 512,
  "inputDevice": {
    "uid": "BuiltInMicrophoneDevice",
    "name": "MacBook Pro Microphone"
  },
  "outputDevice": {
    "uid": "BuiltInSpeakerDevice", 
    "name": "MacBook Pro Speakers"
  }
}
```

### Key Architectural Benefits
- **No Custom Marshaling**: Engine struct directly serializes to JSON
- **Single Source of Truth**: Channel type determined by presence of options (no redundant fields)
- **Impossible Invalid States**: API design prevents conflicting channel type indicators
- **Clean Type Detection**: `channel.IsInput()` and `channel.IsPlayback()` methods for clear logic
- **Automatic Instance Separation**: Plugin parameters maintain independent state
- **Persistence Resilience**: IsInstalled flag handles missing plugins gracefully  
- **Type Safety**: Direct float32 parameter access with bounds validation
- **Guaranteed Non-Nil Options**: Channel creation methods ensure options are never nil when needed

## Implementation Details

### File Format Support
- **Supported**: M4A, MP3, WAV, AIFF, CAF (full AVAudioEngine support)
- **Validation**: Pre-check format compatibility before loading
- **Rejection**: Shallow support formats require user conversion

### Device Integration
- **Selection**: Use comprehensive devices library for enumeration
- **Capability Validation**: Check channel count, sample rates, bit depths
- **Format Matching**: Ensure device compatibility with engine settings

### Enhanced Plugin Management
- **Plugin Discovery**: Use plugins.List() and Introspect() for real AudioUnit plugins
- **Instance Creation**: Each Introspect() call creates independent plugin instances
- **Parameter Access**: Direct access to plugins.Parameter slice with CurrentValue fields  
- **Bounds Validation**: SetPluginParameter enforces MinValue/MaxValue constraints
- **Persistence**: IsInstalled flag for graceful handling of unavailable plugins
- **API Flexibility**: Set/get parameters by Identifier or DisplayName

### Error Handling Philosophy
- **Engine Errors**: Pass through AVAudioEngine errors directly
- **Device Disconnection**: Callback notification, let app handle  
- **Plugin Unavailability**: Mark IsInstalled=false, let app handle gracefully
- **File Loading**: Notify failures, support file switching
- **Resource Limits**: App responsibility, not engine
- **Parameter Bounds**: Engine validates, rejects out-of-bounds values

### Performance Priorities
- **TimePitch**: Performance over quality for real-time use
- **Memory**: Whole file loading with size limits
- **Plugin Processing**: Efficient chain processing with bypass
- **Zero-Latency**: Hardware monitoring + optimized buffer sizes

## MVP Scope Limitations

### Intentionally Excluded
- **Send/Return Routing**: No auxiliary sends or parallel processing
- **Master Pan**: Master volume only
- **Mix Buses**: Direct channel-to-main routing only  
- **Plugin Preset Management**: Full parameter state captured in engine serialization
- **Session Metadata**: App-level concern
- **Dynamic Channel Count**: Fixed 8-channel architecture
- **Plugin Chains on Playback**: Currently input channels only (can be added later)

### Architecture Strengths
- **Single Source of Truth**: Channel type determined by options presence, no redundant fields
- **Impossible Invalid States**: API design prevents conflicting channel type information  
- **Clean Type Detection**: Clear `IsInput()` and `IsPlayback()` methods for type checking
- **Guaranteed Initialization**: Channel creation ensures options are never nil when needed
- **Instance Separation**: Automatic plugin parameter isolation
- **Persistence Resilience**: Graceful handling of unavailable plugins
- **Clean Serialization**: Engine IS the parameter tree - no custom marshaling
- **Type Safety**: Direct float32 parameter access with validation

### Future Considerations
- **Plugin Chains on Playback**: Could extend PlaybackOptions to include PluginChain
- **Streaming Playback**: Currently memory-only file loading
- **Advanced TimePitch**: Currently performance-focused
- **Send/Return**: Would require architectural redesign
- **Dynamic Plugin Loading**: Current persistence model handles availability gracefully

## Technical Foundation

### Native Integration
- **Library**: Unified libmacaudio.dylib
- **Language**: Objective-C native code with Go wrapper
- **Framework**: AVAudioEngine with AVAudioMixerNode core

### Testing Strategy
- **Unit Tests**: Component-level validation with real plugins and devices
- **Integration Tests**: Full signal path verification  
- **Plugin Instance Tests**: Verify parameter separation and serialization
- **Serialization Tests**: Roundtrip testing with comprehensive real data
- **Device Tests**: Real hardware compatibility
- **Performance Tests**: Resource usage validation
- **Persistence Tests**: Plugin availability handling

### Dependencies
- **Devices Package**: Audio device enumeration and capabilities
- **Plugins Package**: AudioUnit discovery and introspection
- **Native Library**: Core AVAudioEngine functionality

## Design Principles

1. **Simplicity Over Features**: Clean architecture beats complex routing
2. **Single Source of Truth**: Channel type determined by options presence, no redundant fields
3. **Impossible Invalid States**: API design prevents conflicting type indicators
4. **Direct Serialization**: Engine IS the parameter tree - no custom marshaling needed
5. **Instance Separation**: Automatic plugin parameter isolation through Go semantics
6. **Type Safety**: Direct float32 access with bounds validation over interface{} maps
7. **Persistence Resilience**: Graceful handling of unavailable plugins via IsInstalled flag
8. **Guaranteed Initialization**: Options structs are never nil when channels are created
9. **Performance Over Quality**: Real-time use prioritized
10. **Fixed Over Dynamic**: Predictable resource usage
11. **Validation Over Recovery**: Fail fast with clear errors

---

This MVP provides a robust foundation for audio mixing applications with a clean, unified channel architecture and sophisticated plugin management. The design achieves automatic parameter instance separation, graceful persistence handling, and direct JSON serialization while maintaining architectural clarity and implementation simplicity. The fixed 8-channel design with optional type-specific configurations supports real-world use cases from live performance to basic production without sacrificing maintainability.

````
