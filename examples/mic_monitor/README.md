# MacAudio Live Microphone Monitor Example

This example demonstrates the core MacAudio engine functionality with a **live microphone monitor** - real-time audio input processing and output.

## What it does

- **Real Audio Signal Path**: Microphone → Processing → Speakers
- **Interactive Control**: Adjust volumes, mute, and monitor settings in real-time
- **Device Selection**: Automatically uses default audio devices or shows available options
- **Performance Validation**: Tests the complete audio engine under real conditions

## Features Demonstrated

### ✅ Core Engine Functionality
- Engine creation and lifecycle management
- AudioInputChannel with real microphone input
- MasterChannel with system audio output
- Volume controls with AVFoundation integration

### ✅ Live Audio Processing  
- Low-latency audio monitoring (256 sample buffer)
- Professional sample rate (48kHz)
- Real-time volume adjustment
- Mute functionality through dispatcher

### ✅ Device Integration
- Audio device enumeration and selection
- Input/output device mapping
- Default device detection

## Usage

```bash
# Build the example
cd examples/mic_monitor
go build

# Run the monitor
./mic_monitor
```

### Interactive Commands

Once running, you can control the audio in real-time:

```
macaudio> i 75          # Set input volume to 75%
macaudio> m 40          # Set master volume to 40%
macaudio> mute          # Toggle input mute
macaudio> status        # Show current settings
macaudio> quit          # Exit
```

## Safety Notes

⚠️ **Audio Feedback Warning**: This creates a live audio loop from microphone to speakers.

- **Use headphones** to prevent feedback
- **Keep volumes low** when using speakers
- **Start with low volumes** and increase gradually

## What You'll Experience

### Expected Results ✅
- Immediate microphone audio through speakers/headphones
- Responsive volume controls (changes apply instantly)
- Clean audio without dropouts or distortion
- Stable operation during control changes

### Performance Indicators ✅
- Low latency audio monitoring
- Real-time control responsiveness  
- No audio glitches during volume changes
- Stable engine operation

## Technical Validation

This example validates:

1. **Complete Signal Path**: End-to-end audio flow
2. **AVFoundation Integration**: Real native audio processing
3. **Dispatcher Architecture**: Thread-safe control operations
4. **Device Management**: Proper audio device handling
5. **Memory Management**: Clean resource cleanup

## Extending the Example

Future enhancements could include:
- Real-time audio level meters (tap functionality)
- Frequency analysis and visualization  
- Audio effects processing
- Multi-channel input support
- MIDI control integration

---

**This example demonstrates that MacAudio provides a solid foundation for professional audio applications with real-time performance and reliable operation.**
