package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -L.. -lmacaudio -Wl,-rpath,..
#include "../native/macaudio.h"
#include <stdlib.h>
*/
import "C"
import (
	"github.com/shaban/macaudio/devices"
)

// =============================================================================
// Public API - Input Channel Management
// =============================================================================

// CreateInputChannel creates an input channel connected to an audio device
func (e *Engine) CreateInputChannel(device *devices.AudioDevice, channelIndex int) (*Channel, error) {
	// TODO: Validate channelIndex is within device's channel count
	channel := &Channel{
		Volume: 1.0,
		Pan:    0.0,
		InputOptions: &InputOptions{
			Device:       device,
			ChannelIndex: channelIndex,
			PluginChain:  NewPluginChain(),
		},
	}

	e.Channels = append(e.Channels, channel)
	return channel, nil
}

// CreateMIDIInputChannel creates an input channel connected to a MIDI device
func (e *Engine) CreateMIDIInputChannel(midiDevice *devices.MIDIDevice, midiChannel int) (*Channel, error) {
	// TODO: Validate midiChannel is within valid MIDI range (1-16 or 0-15)
	channel := &Channel{
		Volume: 1.0,
		Pan:    0.0,
		InputOptions: &InputOptions{
			MidiDevice:   midiDevice,
			ChannelIndex: midiChannel,
			PluginChain:  NewPluginChain(),
		},
	}

	e.Channels = append(e.Channels, channel)
	return channel, nil
}
