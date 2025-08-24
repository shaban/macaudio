package engine

/*
#include "../native/macaudio.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"time"
)

// CreateSamplerChannel creates a sampler channel that can play notes directly
func (e *Engine) CreateSamplerChannel() (*Channel, error) {
	// Create native sampler
	samplerResult := C.audiosampler_create(e.nativeEngine.engine)
	if samplerResult.error != nil {
		return nil, errors.New("Failed to create sampler: " + C.GoString(samplerResult.error))
	}

	// Get main mixer
	mixerResult := C.audioengine_main_mixer_node(e.nativeEngine)
	if mixerResult.error != nil {
		C.audiosampler_destroy((*C.AudioSampler)(samplerResult.result))
		return nil, errors.New("Failed to get main mixer: " + C.GoString(mixerResult.error))
	}

	// Create channel
	channel := &Channel{
		Volume: 1.0,
		Pan:    0.0,
		SamplerOptions: &SamplerOptions{
			samplerPtr: samplerResult.result,
		},
	}

	// Allocate bus for this channel
	busIndex, err := e.AllocateBusForChannel(channel)
	if err != nil {
		C.audiosampler_destroy((*C.AudioSampler)(samplerResult.result))
		return nil, err
	}

	// Connect sampler to main mixer
	connectError := C.audiosampler_connect_to_mixer((*C.AudioSampler)(samplerResult.result), mixerResult.result, C.int(busIndex))
	if connectError != nil {
		C.audiosampler_destroy((*C.AudioSampler)(samplerResult.result))
		e.FreeBusForChannel(channel)
		return nil, errors.New("Failed to connect sampler to mixer: " + C.GoString(connectError))
	}

	// Add to engine's channel list
	e.Channels = append(e.Channels, channel)

	return channel, nil
}

// StartNote starts playing a note on the sampler channel
func (c *Channel) StartNote(note int, velocity int) error {
	if !c.IsSampler() {
		return errors.New("not a sampler channel")
	}

	if c.SamplerOptions.samplerPtr == nil {
		return errors.New("sampler not initialized")
	}

	// Use MIDI channel 0 for simplicity
	errorStr := C.audiosampler_start_note((*C.AudioSampler)(c.SamplerOptions.samplerPtr), C.int(note), C.int(velocity), C.int(0))
	if errorStr != nil {
		return errors.New("Failed to start note: " + C.GoString(errorStr))
	}

	return nil
}

// StopNote stops playing a note on the sampler channel
func (c *Channel) StopNote(note int) error {
	if !c.IsSampler() {
		return errors.New("not a sampler channel")
	}

	if c.SamplerOptions.samplerPtr == nil {
		return errors.New("sampler not initialized")
	}

	// Use MIDI channel 0 for simplicity
	errorStr := C.audiosampler_stop_note((*C.AudioSampler)(c.SamplerOptions.samplerPtr), C.int(note), C.int(0))
	if errorStr != nil {
		return errors.New("Failed to stop note: " + C.GoString(errorStr))
	}

	return nil
}

// PlayNote plays a note for a specific duration (convenience function)
func (c *Channel) PlayNote(note int, velocity int, duration time.Duration) error {
	err := c.StartNote(note, velocity)
	if err != nil {
		return err
	}

	// Schedule note stop
	time.AfterFunc(duration, func() {
		c.StopNote(note) // Ignore error in background
	})

	return nil
}
