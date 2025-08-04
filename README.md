# goauv3 - macOS Audio/MIDI Device Library

A silent, Go library for enumerating macOS Core Audio and Core MIDI devices with configurable JSON logging.

## Features

- **Complete Audio Device Enumeration**: Get all audio devices with input/output capabilities, sample rates, bit depths, device types, and transport types
- **Advanced MIDI Device Hierarchy**: Full 3-level MIDI enumeration (devices → entities → endpoints) with manufacturer details, display names, and SysEx capabilities  
- **Silent Library Design**: No unwanted logging output by default - perfect for production use
- **Configurable JSON Logging**: Enable detailed JSON logging for debugging and development
- **Unified Device Structure**: Both audio and MIDI devices follow consistent error handling patterns
- **Rich Filtering Methods**: Built-in filters for device capabilities, types, and status

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/yourusername/goauv3"
)

func main() {
    // Enable JSON logging for debugging (optional)
    auv3.SetJSONLogging(true)
    
    // Get all audio devices
    audioDevices, err := auv3.GetAllAudioDevices()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d audio devices\n", len(audioDevices))
    
    // Filter for input devices only
    inputDevices := audioDevices.Inputs()
    fmt.Printf("Input devices: %d\n", len(inputDevices))
    
    // Get all MIDI devices  
    midiDevices, err := auv3.GetAllMIDIDevices()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d MIDI endpoints\n", len(midiDevices))
    
    // Filter for output MIDI devices
    midiOutputs := midiDevices.Outputs()
    fmt.Printf("MIDI outputs: %d\n", len(midiOutputs))
}
```

## Audio Device Example

```go
// Get audio devices with rich capabilities
audioDevices, err := auv3.GetAllAudioDevices()
if err != nil {
    log.Fatal(err)
}

for _, device := range audioDevices {
    fmt.Printf("Device: %s\n", device.Name)
    fmt.Printf("  Type: %s (%s transport)\n", device.DeviceType, device.TransportType)
    fmt.Printf("  Input channels: %d\n", device.InputChannelCount)
    fmt.Printf("  Output channels: %d\n", device.OutputChannelCount)
    fmt.Printf("  Sample rates: %v\n", device.SupportedSampleRates)
    fmt.Printf("  Bit depths: %v\n", device.SupportedBitDepths)
    fmt.Printf("  Default input: %v\n", device.IsDefaultInput)
    fmt.Printf("  Default output: %v\n", device.IsDefaultOutput)
    fmt.Printf("  Online: %v\n", device.IsOnline)
}

// Filter examples
usbDevices := audioDevices.ByType("usb")
inputOutputDevices := audioDevices.InputOutput()
onlineDevices := audioDevices.Online()
```

## MIDI Device Example

```go
// Get MIDI devices with complete hierarchy
midiDevices, err := auv3.GetAllMIDIDevices()
if err != nil {
    log.Fatal(err)
}

for _, device := range midiDevices {
    fmt.Printf("MIDI Device: %s\n", device.Name)
    fmt.Printf("  Display Name: %s\n", device.DisplayName)
    fmt.Printf("  Device: %s\n", device.DeviceName)
    fmt.Printf("  Manufacturer: %s\n", device.Manufacturer)
    fmt.Printf("  Model: %s\n", device.Model)
    fmt.Printf("  Entity: %s\n", device.EntityName)
    fmt.Printf("  SysEx Speed: %d bytes/sec\n", device.SysExSpeed)
    fmt.Printf("  Input: %v (ID: %d)\n", device.IsInput, device.InputEndpointID)
    fmt.Printf("  Output: %v (ID: %d)\n", device.IsOutput, device.OutputEndpointID)
    fmt.Printf("  Online: %v\n", device.IsOnline)
}

// Filter examples
midiInputs := midiDevices.Inputs()
midiOutputs := midiDevices.Outputs()
onlineMidi := midiDevices.Online()
```

## Device Filtering Methods

### Audio Devices

```go
audioDevices, _ := auv3.GetAllAudioDevices()

// Capability filters
inputs := audioDevices.Inputs()           // Can capture audio
outputs := audioDevices.Outputs()         // Can play audio  
inputOutput := audioDevices.InputOutput() // Can do both

// Type filters
usbDevices := audioDevices.ByType("usb")
builtinDevices := audioDevices.ByType("builtin")
bluetoothDevices := audioDevices.ByType("bluetooth")

// Status filters
onlineDevices := audioDevices.Online()
```

### MIDI Devices

```go
midiDevices, _ := auv3.GetAllMIDIDevices()

// Capability filters
inputs := midiDevices.Inputs()           // Can receive MIDI
outputs := midiDevices.Outputs()         // Can send MIDI
inputOutput := midiDevices.InputOutput() // Can do both

// Manufacturer filters
rolandDevices := midiDevices.ByManufacturer("Roland")
appleDevices := midiDevices.ByManufacturer("Apple")

// Status filters
onlineDevices := midiDevices.Online()
```

## JSON Logging

Enable detailed JSON logging for debugging:

```go
// Enable JSON logging to see raw device data
auv3.SetJSONLogging(true)

audioDevices, _ := auv3.GetAllAudioDevices()
// Outputs: 🔍 Audio Devices JSON: {"success":true,"devices":[...],"deviceCount":5}

midiDevices, _ := auv3.GetAllMIDIDevices()  
// Outputs: 🔍 MIDI Devices JSON: {"success":true,"devices":[...],"deviceCount":8}

// Disable for production
auv3.SetJSONLogging(false)
```

## Testing

Run the comprehensive test suite:

```bash
# Test audio devices
go test -run TestGetAllAudioDevices -v

# Test MIDI devices  
go test -run TestGetAllMIDIDevices -v

# Test device filtering
go test -run TestMIDIDeviceFiltering -v

# Run all tests
go test -v
```

## Requirements

- macOS 10.9+ (Core Audio/MIDI frameworks)
- Go 1.16+ with CGO enabled
- Xcode command line tools

## Architecture

This library implements a **silent library design pattern**:

- **Objective-C Layer**: Silent Core Audio/MIDI enumeration functions that return structured JSON
- **Go Layer**: Configurable JSON logging and rich device structures with filtering methods
- **Error Handling**: Consistent success/error JSON responses with proper Go error propagation

## Device Hierarchy

### Audio Devices
- Direct Core Audio device enumeration
- Input/output channel detection
- Sample rate and bit depth discovery  
- Transport type identification (USB, built-in, Bluetooth, etc.)

### MIDI Devices (3-Level Hierarchy)
1. **Devices**: Top-level MIDI devices (e.g., "KATANA", "IAC Driver")
2. **Entities**: Logical groupings within devices (e.g., "MIDI Port", "Bus 1") 
3. **Endpoints**: Actual input/output endpoints with capabilities

Each endpoint includes:
- Manufacturer and model information
- Display names (user-friendly names)
- SysEx transfer speeds
- Unique endpoint IDs for Core MIDI operations
