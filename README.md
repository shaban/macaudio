# macaudio - macOS Audio/MIDI Device Library

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
    
    "macaudio/devices"
)

func main() {
    // Enable JSON logging for debugging (optional)
    devices.SetJSONLogging(true)
    
    // Get all audio devices
    audioDevices, err := devices.GetAllAudioDevices()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d audio devices\n", len(audioDevices))
    
    // Filter for input devices only
    inputDevices := audioDevices.Inputs()
    fmt.Printf("Input devices: %d\n", len(inputDevices))
    
    // Get all MIDI devices  
    midiDevices, err := devices.GetAllMIDIDevices()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d MIDI endpoints\n", len(midiDevices))
    
    // Filter for output MIDI devices
    midiOutputs := midiDevices.Outputs()
    fmt.Printf("MIDI outputs: %d\n", len(midiOutputs))
}
```

## Installation

```bash
go get github.com/shaban/macaudio
```

## Audio Device Example

```go
import "macaudio/devices"

// Get audio devices with rich capabilities
audioDevices, err := devices.GetAllAudioDevices()
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
import "macaudio/devices"

// Get MIDI devices with complete hierarchy
midiDevices, err := devices.GetAllMIDIDevices()
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
bossMIDI := midiDevices.ByManufacturer("BOSS Corporation")
katanaMIDI := midiDevices.ByModel("KATANA")
inputDevices := midiDevices.Inputs()
outputDevices := midiDevices.Outputs()
```

## Device Filtering

Both audio and MIDI devices support comprehensive filtering:

```go
// Audio device filters
audioDevices := devices.GetAllAudioDevices()
inputs := audioDevices.Inputs()           // Input capable devices
outputs := audioDevices.Outputs()         // Output capable devices  
inputOutput := audioDevices.InputOutput() // Bidirectional devices
builtin := audioDevices.ByType("builtin") // Built-in devices
usb := audioDevices.ByType("usb")         // USB devices

// MIDI device filters
midiDevices := devices.GetAllMIDIDevices()
inputs := midiDevices.Inputs()                    // Input endpoints
outputs := midiDevices.Outputs()                 // Output endpoints
boss := midiDevices.ByManufacturer("BOSS")       // BOSS devices
katana := midiDevices.ByModel("KATANA")          // KATANA models
online := midiDevices.Online()                   // Online devices
```

## JSON Logging

Enable detailed JSON logging for debugging:

```go
import "macaudio/devices"

// Enable JSON logging to see raw device data
devices.SetJSONLogging(true)

audioDevices, _ := devices.GetAllAudioDevices()
// Outputs: 🔍 Audio Devices JSON: {"success":true,"devices":[...],"deviceCount":5}

midiDevices, _ := devices.GetAllMIDIDevices()  
// Outputs: 🔍 MIDI Devices JSON: {"success":true,"devices":[...],"deviceCount":8}

// Disable for production
devices.SetJSONLogging(false)
```

## Testing

Use the included Makefile for comprehensive testing:

```bash
# Test all devices (recommended)
make test-all

# Test specific device types
make test-audio
make test-midi

# Test with clean build
make test-clean

# Show library information
make info

# Show all available commands
make help
```

Or use Go directly:

```bash
# Test audio devices
go test -v ./devices -run TestGetAudioDevices

# Test MIDI devices  
go test -v ./devices -run TestGetAllMIDIDevices

# Run all tests
go test -v ./devices
```

## Requirements

- macOS 10.9+ (Core Audio/MIDI frameworks)
- Go 1.23+ with CGO enabled
- Xcode command line tools

## Package Structure

```
macaudio/                          # Root package
├── LICENSE                        # GNU AGPL v3 License
├── README.md                      # This file
├── go.mod                         # Module: macaudio
├── Makefile                       # Build and test commands
└── devices/                       # Device enumeration package
    ├── devices.go                 # Main API
    ├── devices_test.go            # Audio device tests
    ├── midi_test.go               # MIDI device tests
    ├── unified_test.go            # Combined tests
    └── native/
        └── devices.m              # Core Audio/MIDI implementation
```

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

## Real Device Examples

The library has been tested with real hardware:

### MIDI Devices
- **BOSS KATANA**: Guitar amplifier with MIDI control (manufacturer: "BOSS Corporation", model: "KATANA")
- **Nektar SE61**: MIDI keyboard controller (manufacturer: "Nektar", model: "SE61")  
- **Mooer Audio Steep II**: Audio interface with MIDI (manufacturer: "Mooer Audio", model: "Steep II")
- **Apple IAC Driver**: Virtual MIDI buses (manufacturer: "Apple Inc.", model: "IAC Driver")

### Audio Devices
- **USB Audio Interfaces**: Complete sample rate and bit depth enumeration
- **Built-in Audio**: Mac built-in speakers and headphone outputs
- **HDMI Audio**: External displays with audio capabilities
- **Background Music**: Virtual audio routing devices

## Contributing

Contributions are welcome! Please ensure all tests pass:

```bash
make test-all
```

## License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Go Version**: 1.23+
- **Platform**: macOS 10.9+
- **Architecture**: x86_64, ARM64 (Apple Silicon)
- **Dependencies**: Core Audio, Core MIDI frameworks (system-provided)
