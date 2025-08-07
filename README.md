# macaudio - macOS Audio/MIDI Device & AudioUnit Plugin Library

A silent, Go library for enumerating macOS Core Audio and Core MIDI devices, and introspecting AudioUnit plugins with configurable JSON logging.

## Features

- **Complete Audio Device Enumeration**: Get all audio devices with input/output capabilities, sample rates, bit depths, device types, and transport types
- **Advanced MIDI Device Hierarchy**: Full 3-level MIDI enumeration (devices → entities → endpoints) with manufacturer details, display names, and SysEx capabilities
- **AudioUnit Plugin Introspection**: Enumerate and introspect AudioUnit plugins with full parameter metadata and filtering capabilities
- **Method-Based Plugin API**: Modern Go API with both synchronous method-based and function-based introspection
- **Silent Library Design**: No unwanted logging output by default - perfect for production use
- **Configurable JSON Logging**: Enable detailed JSON logging for debugging and development
- **Unified Structure**: Audio devices, MIDI devices, and plugins all follow consistent error handling patterns
- **Rich Filtering Methods**: Built-in filters for device capabilities, plugin types, and status

## Quick Start

### Device Discovery
```go
package main

import (
    "fmt"
    "log"
    
    "github.com/shaban/macaudio/devices"
)

func main() {
    // Enable JSON logging for debugging (optional)
    devices.SetJSONLogging(true)
    
    // Get all audio devices
    audioDevices, err := devices.GetAudio()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d audio devices\n", len(audioDevices))
    
    // Filter for input devices only
    inputDevices := audioDevices.Inputs()
    fmt.Printf("Input devices: %d\n", len(inputDevices))
    
    // Get all MIDI devices  
    midiDevices, err := devices.GetMIDI()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d MIDI endpoints\n", len(midiDevices))
    
    // Filter for output MIDI devices
    midiOutputs := midiDevices.Outputs()
    fmt.Printf("MIDI outputs: %d\n", len(midiOutputs))
}
```

### Plugin Introspection
```go
package main

import (
    "fmt"
    "log"
    
    "github.com/shaban/macaudio/plugins"
)

func main() {
    // Get all AudioUnit plugins
    auPlugins, err := plugins.GetAU()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d AudioUnit plugins\n", len(auPlugins))
    
    // Filter for instruments only
    instruments := auPlugins.Instruments()
    fmt.Printf("Instruments: %d\n", len(instruments))
    
    // Get parameter details for the first plugin
    if len(auPlugins) > 0 {
        params, err := auPlugins[0].GetParameters()
        if err == nil {
            fmt.Printf("Plugin '%s' has %d parameters\n", 
                auPlugins[0].Name, len(params))
        }
    }
}
```

## Installation

```bash
go get github.com/shaban/macaudio
```

## Audio Device Example

```go
import "github.com/shaban/macaudio/devices"

// Get audio devices with rich capabilities
audioDevices, err := devices.GetAudio()
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
midiDevices, err := devices.GetMIDI()
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

## AudioUnit Plugin Example

```go
import "github.com/shaban/macaudio/plugins"

// Get all AudioUnit plugins with quick scan
pluginInfos, err := plugins.List()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d AudioUnit plugins\n", len(pluginInfos))

// Filter examples
effectPlugins := pluginInfos.ByType("aufx")              // Effects
instrumentPlugins := pluginInfos.ByType("aumu")          // Instruments
applePlugins := pluginInfos.ByManufacturer("appl")       // Apple plugins
compressorPlugins := pluginInfos.ByName("compressor")    // Name search

// Method-based introspection (recommended)
for _, info := range effectPlugins[:3] { // First 3 effects
    plugin, err := info.Introspect()
    if err != nil {
        continue
    }
    
    fmt.Printf("Plugin: %s\n", plugin.Name)
    fmt.Printf("  Type: %s (%s)\n", plugin.Type, plugin.Category)
    fmt.Printf("  Manufacturer: %s\n", plugin.ManufacturerID)
    fmt.Printf("  Parameters: %d\n", len(plugin.Parameters))
    
    // Parameter details
    for _, param := range plugin.Parameters[:min(3, len(plugin.Parameters))] {
        fmt.Printf("    %s: %.2f (%.2f-%.2f) %s\n", 
            param.DisplayName, param.CurrentValue, 
            param.MinValue, param.MaxValue, param.Unit)
    }
}

// Batch introspection for multiple plugins
instrumentPlugins = instrumentPlugins[:2] // Limit for example
instruments, err := instrumentPlugins.Introspect()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Introspected %d instruments with full parameter data\n", len(instruments))
```

## Plugin Introspection API

The plugins package provides both method-based and function-based APIs:

### Method-Based API (Recommended)

```go
// Single plugin introspection
pluginInfo := pluginInfos[0]
plugin, err := pluginInfo.Introspect()
if err != nil {
    log.Fatal(err)
}

// Multiple plugin introspection
selectedPlugins := pluginInfos.ByCategory("Effect")[:5]
plugins, err := selectedPlugins.Introspect()
if err != nil {
    log.Fatal(err)
}
```

### Function-Based API

```go
// Introspect by plugin identifiers
plugins, err := plugins.Introspect("aufx", "dcmp", "appl") // Apple Compressor
if err != nil {
    log.Fatal(err)
}

// Get all plugins (empty parameters = all)
allPlugins, err := plugins.Introspect("", "", "")
if err != nil {
    log.Fatal(err)
}
```

## Plugin Filtering and Analysis

Rich filtering methods for both plugin lists and full plugin data:

```go
// Plugin info filtering (quick scan data)
pluginInfos, _ := plugins.List()

effectPlugins := pluginInfos.ByType("aufx")                    // Audio effects
instrumentPlugins := pluginInfos.ByType("aumu")                // Instruments  
musicEffectPlugins := pluginInfos.ByType("aumf")               // Music effects
applePlugins := pluginInfos.ByManufacturer("appl")             // Apple plugins
izotopePlugins := pluginInfos.ByManufacturer("iZtp")           // iZotope plugins
delayPlugins := pluginInfos.ByName("delay")                    // Name contains "delay"
effectCategory := pluginInfos.ByCategory("Effect")             // Effect category

// Full plugin filtering (after introspection)
allPlugins, _ := plugins.Introspect("", "", "")

pluginsWithParams := allPlugins.WithParameters()                // Has parameters
indexedPlugins := allPlugins.WithIndexedParameters()            // Has dropdown/list params
appleInstruments := allPlugins.ByManufacturer("appl").ByType("aumu")

// Parameter analysis for individual plugins
for _, plugin := range allPlugins {
    writableParams := plugin.GetWritableParameters()            // User-controllable
    rampableParams := plugin.GetRampableParameters()            // Automatable
    booleanParams := plugin.GetParametersByUnit("Boolean")      // On/off switches
    indexedParams := plugin.GetIndexedParameters()              // Dropdowns/lists
    
    fmt.Printf("%s: %d writable, %d automatable, %d boolean, %d indexed\n",
        plugin.Name, len(writableParams), len(rampableParams), 
        len(booleanParams), len(indexedParams))
}
```

## Plugin Parameter Details

AudioUnit parameters include comprehensive metadata:

```go
plugin, _ := pluginInfo.Introspect()

for _, param := range plugin.Parameters {
    fmt.Printf("Parameter: %s\n", param.DisplayName)
    fmt.Printf("  Value: %.3f (default: %.3f)\n", param.CurrentValue, param.DefaultValue)
    fmt.Printf("  Range: %.3f to %.3f\n", param.MinValue, param.MaxValue)
    fmt.Printf("  Unit: %s\n", param.Unit)                     // Hz, dB, Percent, etc.
    fmt.Printf("  Address: %d\n", param.Address)               // For automation
    fmt.Printf("  Writable: %v\n", param.IsWritable)           // User controllable
    fmt.Printf("  Automatable: %v\n", param.CanRamp)           // DAW automation
    fmt.Printf("  Raw Flags: 0x%X\n", param.RawFlags)          // AudioUnit flags
    
    // Indexed parameters (dropdowns, lists)
    if len(param.IndexedValues) > 0 {
        fmt.Printf("  Options: %v\n", param.IndexedValues)
        fmt.Printf("  Current Option: %s\n", param.IndexedValues[int(param.CurrentValue)])
    }
}
```

## Plugin Categories and Types

AudioUnit plugins are organized by type and category:

### Plugin Types
- **`aufx`**: Audio Effects (reverb, delay, EQ, compressor)
- **`aumu`**: Instruments (synths, samplers, drum machines)
- **`aumf`**: Music Effects (guitar amps, vocal processors)
- **`aumx`**: Mixers (channel strips, spatial audio)
- **`augn`**: Generators (tone generators, noise generators)
- **`auou`**: Output Units (audio interfaces, system output)
- **`aufc`**: Format Converters (sample rate, time stretching)

### Common Categories
- **Effect**: Standard audio effects
- **Instrument**: Software instruments
- **Music Effect**: Specialized music processing
- **Mixer**: Audio mixing and routing
- **Generator**: Audio/tone generation
- **Output**: Audio output and interfaces

```go
// Category-based workflow
pluginInfos, _ := plugins.List()

// Audio production workflow
effects := pluginInfos.ByCategory("Effect")
instruments := pluginInfos.ByCategory("Instrument")
processors := pluginInfos.ByCategory("Music Effect")

fmt.Printf("Available: %d effects, %d instruments, %d processors\n",
    len(effects), len(instruments), len(processors))

// Type-based filtering
audioEffects := pluginInfos.ByType("aufx")        // Standard effects
softSynths := pluginInfos.ByType("aumu")          // Software instruments
guitarAmps := pluginInfos.ByType("aumf")          // Guitar/music effects
```

## Plugin JSON Logging

Enable detailed JSON logging for debugging plugin introspection:

```go
import "github.com/shaban/macaudio/plugins"

// Enable JSON logging
plugins.SetJSONLogging(true)

pluginInfos, _ := plugins.List()
// Outputs: 🔍 Plugin List JSON: {"success":true,"plugins":[...],"pluginCount":152}

plugin, _ := pluginInfos[0].Introspect()
// Outputs: 🔍 IntrospectWithTimeout JSON: {"success":true,"plugins":[...],"pluginCount":1}

// Disable for production
plugins.SetJSONLogging(false)
```

## Device Compatibility

The library provides powerful utility methods to find compatible audio settings between devices:

### Audio Device Compatibility Methods

```go
import "macaudio/devices"

// CommonSampleRates finds sample rates supported by both devices
func (device AudioDevice) CommonSampleRates(other AudioDevice) []int

// CommonBitDepths finds bit depths supported by both devices  
func (device AudioDevice) CommonBitDepths(other AudioDevice) []int
```

### Basic Usage

```go
// Get devices
audioDevices, _ := devices.GetAudio()
inputDevice := audioDevices.Inputs()[0]   // Audio interface
outputDevice := audioDevices.Outputs()[0] // Speakers/headphones

// Find compatible sample rates and bit depths
commonRates := inputDevice.CommonSampleRates(outputDevice)
commonDepths := inputDevice.CommonBitDepths(outputDevice)

fmt.Printf("Compatible sample rates: %v\n", commonRates)     // [44100, 48000]
fmt.Printf("Compatible bit depths: %v\n", commonDepths)     // [24, 32]

// Check if devices are compatible
if len(commonRates) == 0 || len(commonDepths) == 0 {
    fmt.Println("⚠️  Devices are not compatible")
} else {
    fmt.Printf("✅ Found %d compatible configurations\n", len(commonRates)*len(commonDepths))
}
```

### UI Integration Examples

Perfect for dynamic UI updates when users change device selections:

```go
// Sample Rate Dropdown Population
func updateSampleRateOptions(input, output AudioDevice) {
    compatibleRates := input.CommonSampleRates(output)
    
    sampleRateSelect.Clear()
    for _, rate := range compatibleRates {
        sampleRateSelect.AddOption(fmt.Sprintf("%d Hz", rate))
    }
    
    if len(compatibleRates) == 0 {
        sampleRateSelect.AddOption("No compatible rates")
        sampleRateSelect.Disable()
    }
}

// Bit Depth Dropdown Population  
func updateBitDepthOptions(input, output AudioDevice) {
    compatibleDepths := input.CommonBitDepths(output)
    
    bitDepthSelect.Clear()
    for _, depth := range compatibleDepths {
        bitDepthSelect.AddOption(fmt.Sprintf("%d-bit", depth))
    }
}
```

### Real-world Use Cases

```go
// DAW/Audio Software: User changes input device
func onInputDeviceChanged(newInput AudioDevice, currentOutput AudioDevice) {
    availableRates := newInput.CommonSampleRates(currentOutput)
    availableDepths := newInput.CommonBitDepths(currentOutput)
    
    // Update UI to show only compatible options
    updateSampleRateDropdown(availableRates)
    updateBitDepthDropdown(availableDepths)
    
    // Auto-select best option
    if len(availableRates) > 0 {
        selectBestSampleRate(availableRates) // e.g., highest available
    }
}

// Pro Audio: Validate routing before connecting
func validateAudioRoute(source, destination AudioDevice) error {
    commonRates := source.CommonSampleRates(destination)
    commonDepths := source.CommonBitDepths(destination)
    
    if len(commonRates) == 0 {
        return fmt.Errorf("no compatible sample rates between %s and %s", 
            source.Name, destination.Name)
    }
    
    if len(commonDepths) == 0 {
        return fmt.Errorf("no compatible bit depths between %s and %s", 
            source.Name, destination.Name)
    }
    
    return nil // Devices are compatible
}

// Audio Interface Setup: Find optimal settings
func findOptimalSettings(devices []AudioDevice) (int, int) {
    if len(devices) < 2 {
        return 0, 0
    }
    
    // Start with first device's capabilities
    commonRates := devices[0].SupportedSampleRates
    commonDepths := devices[0].SupportedBitDepths
    
    // Find intersection across all devices
    for i := 1; i < len(devices); i++ {
        commonRates = intersectRates(commonRates, devices[i].SupportedSampleRates)
        commonDepths = intersectDepths(commonDepths, devices[i].SupportedBitDepths)
    }
    
    // Return highest quality settings
    bestRate := findHighest(commonRates)   // e.g., 96000
    bestDepth := findHighest(commonDepths) // e.g., 32
    
    return bestRate, bestDepth
}
```

### Edge Cases Handled

The utility methods gracefully handle all edge cases:

```go
// Empty arrays - returns empty slice
emptyDevice := AudioDevice{SupportedSampleRates: []int{}}
result := device1.CommonSampleRates(emptyDevice) // Returns: []int{}

// No intersection - returns empty slice  
device1 := AudioDevice{SupportedSampleRates: []int{44100, 48000}}
device2 := AudioDevice{SupportedSampleRates: []int{96000, 192000}}
result := device1.CommonSampleRates(device2) // Returns: []int{}

// Order preservation - maintains first device's order
device1 := AudioDevice{SupportedSampleRates: []int{96000, 44100, 48000}}
device2 := AudioDevice{SupportedSampleRates: []int{44100, 48000, 96000}}
result := device1.CommonSampleRates(device2) // Returns: [96000, 44100, 48000]
```

## Device Filtering

Both audio and MIDI devices support comprehensive filtering:

```go
// Audio device filters
audioDevices := devices.GetAudio()
inputs := audioDevices.Inputs()           // Input capable devices
outputs := audioDevices.Outputs()         // Output capable devices  
inputOutput := audioDevices.InputOutput() // Bidirectional devices
builtin := audioDevices.ByType("builtin") // Built-in devices
usb := audioDevices.ByType("usb")         // USB devices

// MIDI device filters
midiDevices := devices.GetMIDI()
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

audioDevices, _ := devices.GetAudio()
// Outputs: 🔍 Audio Devices JSON: {"success":true,"devices":[...],"deviceCount":5}

midiDevices, _ := devices.GetMIDI()  
// Outputs: 🔍 MIDI Devices JSON: {"success":true,"devices":[...],"deviceCount":8}

// Disable for production
devices.SetJSONLogging(false)
```

## Testing

Use the included Makefile for comprehensive testing:

```bash
# Test all devices (recommended)
make test-devices

# Test specific device types (from devices directory)
cd devices && make test-audio
cd devices && make test-midi

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
go test -v ./devices -run TestGetAudio

# Test MIDI devices  
go test -v ./devices -run TestGetMIDI

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
├── go.mod                         # Module: github.com/shaban/macaudio
├── Makefile                       # Build and test commands
├── devices/                       # Device enumeration package
│   ├── devices.go                 # Main API
│   ├── devices_test.go            # Audio device tests
│   ├── midi_test.go               # MIDI device tests
│   ├── unified_test.go            # Combined tests
│   └── native/
│       └── devices.m              # Core Audio/MIDI implementation
└── plugins/                       # AudioUnit plugin package
    ├── plugins.go                 # Main API
    ├── plugins_test.go            # Plugin enumeration tests
    ├── method_test.go             # Method-based API tests
    └── native/
        └── plugins.m              # AudioUnit introspection implementation
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
make test-devices
```

## License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Support

- **Go Version**: 1.23+
- **Platform**: macOS 10.9+
- **Architecture**: x86_64, ARM64 (Apple Silicon)
- **Dependencies**: Core Audio, Core MIDI frameworks (system-provided)
