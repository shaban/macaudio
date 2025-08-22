# devices package

Unified Core Audio and Core MIDI enumeration with a silent-by-default Go API and optional JSON logging.

## API overview

- GetAudio() → AudioDevices: unified audio devices with input/output channels, sample rates, bit depths, defaults, type and transport.
- GetMIDI() → MIDIDevices: unified MIDI endpoints (input/output), with device/entity/display names, manufacturer/model, SysEx speed.
- GetAudioDeviceCount(), GetMIDIDeviceCount(), GetDeviceCounts(): fast-path counters for hotplug polling (~tens of microseconds) without full enumeration.

## Fast-path counts (hotplug polling)

Use the count functions to poll for changes quickly:

```go
ac, mc, _ := devices.GetDeviceCounts()
// If counts changed, do a focused refresh.
```

Notes:
- Counts are fast because they avoid building full structures.
- On discrepancy, call GetAudio/GetMIDI to reconcile.

Example polling loop with debounce:

```go
prevA, prevM := -1, -1
for range time.Tick(250 * time.Millisecond) {
  a, m, err := devices.GetDeviceCounts()
  if err != nil { continue }
  if a != prevA || m != prevM {
    prevA, prevM = a, m
    // Reconcile: fetch details only when counts change
    audio, _ := devices.GetAudio()
    midi, _ := devices.GetMIDI()
    _ = audio; _ = midi
  }
}
```

## JSON logging

Enable raw JSON logging from the native layer, optionally redirected to a file:

```go
f, _ := os.Create("devices_scan.jsonl")
defer f.Close()
devices.SetJSONLogWriter(f)
devices.SetJSONLogging(true)

audio, _ := devices.GetAudio() // Emits: AudioDevices: {...}
midi,  _ := devices.GetMIDI()  // Emits: MIDIDevices: {...}

// Tip: The labels in the log make it easy to grep separate Audio vs MIDI records.
```

## Data contracts

Audio JSON envelope:
```json
{
  "success": true,
  "devices": [ { "name": "...", "uid": "...", "deviceId": 123, ... } ],
  "deviceCount": 5,
  "totalDevicesScanned": 7
}
```

MIDI JSON envelope:
```json
{
  "success": true,
  "devices": [ { "name": "...", "uid": "midi_123_Bus 1", "isInput": true, ... } ],
  "deviceCount": 8,
  "totalDevicesScanned": 3
}
```

## Semantics and IDs

- Audio device UID: CoreAudio UID when available; otherwise a stable fallback of `device_<AudioDeviceID>`.
- MIDI UID: based on CoreMIDI UniqueID plus endpoint name to disambiguate endpoints of the same device.

## Filtering

The returned slices provide convenience filters:

- Audio: `Inputs()`, `Outputs()`, `InputOutput()`, `Online()`, `ByType("builtin|usb|aggregate|...")`
- MIDI: `Inputs()`, `Outputs()`, `InputOutput()`, `Online()`, `ByManufacturer()`, `ByModel()`

## Hotplug notes

- For polling, use the fast-path counts at a modest interval and debounce changes.
- Avoid handling CoreAudio/MIDI callbacks on real-time threads directly; dispatch to non-RT threads before calling into Go.
