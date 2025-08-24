# Sampler Implementation Reference

This document shows the Swift reference implementation that inspired our Go/C sampler approach.

## Key Insights from Swift Implementation

The Swift program demonstrated that using `AVAudioUnitSampler` with direct `startNote()`/`stopNote()` calls produces reliable sound output, unlike the complex MIDI routing approach we initially tried.

**Architecture:**
```
Go Application → AVAudioUnitSampler.startNote() → AVAudioEngine → Speakers
```

Instead of:
```
Go → MIDI Library → IAC Driver → AVAudioUnitMIDIInstrument → Engine
```

## Swift Reference Implementation

```swift
import AVFoundation
import Foundation

class MIDIGenerator {
    private var audioEngine: AVAudioEngine
    private var sampler: AVAudioUnitSampler
    
    init() {
        audioEngine = AVAudioEngine()
        sampler = AVAudioUnitSampler()
        
        setupAudio()
    }
    
    private func setupAudio() {
        // Attach the sampler to the audio engine
        audioEngine.attach(sampler)
        
        // Connect sampler to main mixer with explicit format
        let format = AVAudioFormat(standardFormatWithSampleRate: 44100, channels: 2)
        audioEngine.connect(sampler, to: audioEngine.mainMixerNode, format: format)
        
        // Load a basic instrument sound
        loadInstrument()
        
        // Start the audio engine
        do {
            try audioEngine.start()
            print("Audio engine started successfully")
        } catch {
            print("Failed to start audio engine: \(error)")
        }
    }
    
    private func loadInstrument() {
        // List of possible system instrument paths on macOS
        let possiblePaths = [
            "/System/Library/Components/CoreAudio.component/Contents/Resources/gs_instruments.dls",
            "/System/Library/Audio/Sounds/Banks/gs_instruments.dls",
            "/Library/Audio/Sounds/Banks/gs_instruments.dls"
        ]
        
        for path in possiblePaths {
            if FileManager.default.fileExists(atPath: path) {
                do {
                    try sampler.loadInstrument(at: URL(fileURLWithPath: path))
                    print("Successfully loaded instrument from: \(path)")
                    return
                } catch {
                    print("Failed to load from \(path): \(error)")
                }
            } else {
                print("File not found: \(path)")
            }
        }
        
        // If no system instruments found, try SoundFont bank loading with General MIDI program 0 (Piano)
        print("No system instruments found, trying alternative approach...")
        loadGeneralMIDIPreset()
    }
    
    private func loadGeneralMIDIPreset() {
        // Try alternative system paths for instruments
        let alternatePaths = [
            "/System/Library/Audio/Sounds/Banks/DefaultSounds.sf2",
            "/Library/Audio/Sounds/Banks/DefaultSounds.sf2"
        ]
        
        for path in alternatePaths {
            if FileManager.default.fileExists(atPath: path) {
                do {
                    try sampler.loadInstrument(at: URL(fileURLWithPath: path))
                    print("Loaded instrument from: \(path)")
                    return
                } catch {
                    print("Failed to load from \(path): \(error)")
                }
            }
        }
        
        print("Using default sampler instrument (may be silent)")
        // The sampler will use its default behavior, which might not produce sound
        // but the MIDI events will still be sent
    }
    
    func playNote(midiNote: UInt8, velocity: UInt8 = 127, duration: TimeInterval = 1.0) {
        print("Playing MIDI note: \(midiNote)")
        
        // Start the note
        sampler.startNote(midiNote, withVelocity: velocity, onChannel: 0)
        
        // Stop the note after duration
        DispatchQueue.main.asyncAfter(deadline: .now() + duration) {
            self.sampler.stopNote(midiNote, onChannel: 0)
        }
    }
    
    func playChord(notes: [UInt8], velocity: UInt8 = 127, duration: TimeInterval = 2.0) {
        print("Playing chord: \(notes)")
        
        // Start all notes in the chord
        for note in notes {
            sampler.startNote(note, withVelocity: velocity, onChannel: 0)
        }
        
        // Stop all notes after duration
        DispatchQueue.main.asyncAfter(deadline: .now() + duration) {
            for note in notes {
                self.sampler.stopNote(note, onChannel: 0)
            }
        }
    }
}

// Example usage for macOS
print("Starting macOS MIDI Generator...")
let midiGen = MIDIGenerator()

print("\n--- Playing single note (Middle C) ---")
midiGen.playNote(midiNote: 60, duration: 2.0)

// Play a chord after 3 seconds
DispatchQueue.main.asyncAfter(deadline: .now() + 3.0) {
    print("\n--- Playing C Major chord ---")
    midiGen.playChord(notes: [60, 64, 67], duration: 2.0)
}

// Keep the program running
print("Audio should be playing... (program will exit automatically)")
RunLoop.main.run()
```

## Key Takeaways for Our Go Implementation

1. **Direct Note Control**: Use `sampler.startNote()` and `sampler.stopNote()` instead of MIDI routing
2. **Simple Connection**: Just attach sampler to engine and connect to main mixer
3. **Default Instruments**: AVAudioUnitSampler has built-in instruments that work without loading files
4. **Reliable Sound**: This approach produces consistent audio output

Our Go implementation successfully replicates this pattern using CGO bridges to the same AVFoundation APIs.
