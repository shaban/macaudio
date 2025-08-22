package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -L.. -lmacaudio -Wl,-rpath,..
#include "../native/macaudio.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/shaban/macaudio/devices"
)

// =============================================================================
// Public API - Channel Management
// =============================================================================

// GetChannelID returns a unique identifier for the channel based on its mixer node pointer
func (c *Channel) GetChannelID() uintptr {
	return uintptr(c.mixerNodePtr)
}

// AllocateBusForChannel assigns a unique input bus on the main mixer for this channel
func (e *Engine) AllocateBusForChannel(channel *Channel) (int, error) {
	channelID := channel.GetChannelID()

	// Check if channel already has a bus allocated
	if busIndex, exists := e.busAllocation[channelID]; exists {
		return busIndex, nil
	}

	// Check if we have available buses
	if e.nextAvailableBus >= e.maxBuses {
		return -1, errors.New("no available buses - maximum channels reached")
	}

	// Allocate the next available bus
	busIndex := e.nextAvailableBus
	e.busAllocation[channelID] = busIndex
	e.nextAvailableBus++

	return busIndex, nil
}

// FreeBusForChannel releases the bus allocated to this channel
func (e *Engine) FreeBusForChannel(channel *Channel) error {
	channelID := channel.GetChannelID()

	busIndex, exists := e.busAllocation[channelID]
	if !exists {
		return errors.New("channel does not have an allocated bus")
	}

	// Remove the allocation
	delete(e.busAllocation, channelID)

	// If this was the highest allocated bus, we can reuse it next
	if busIndex == e.nextAvailableBus-1 {
		e.nextAvailableBus--
	}

	return nil
}

// GetChannelBus returns the bus index allocated to this channel
func (e *Engine) GetChannelBus(channel *Channel) (int, error) {
	channelID := channel.GetChannelID()

	busIndex, exists := e.busAllocation[channelID]
	if !exists {
		return -1, errors.New("channel does not have an allocated bus")
	}

	return busIndex, nil
}

// Channel represents a unified channel that can be input or playback
// BusIndex removed: channels no longer care about their index in the slice
type Channel struct {
	// Base channel properties
	Volume float32 `json:"volume"`
	Pan    float32 `json:"pan"`

	// Optional type-specific data (nil when not applicable)
	PlaybackOptions *PlaybackOptions `json:"playbackOptions,omitempty"`
	InputOptions    *InputOptions    `json:"inputOptions,omitempty"`

	// Internal mixing node for this channel (not serialized)
	mixerNodePtr unsafe.Pointer `json:"-"`
}

// PlaybackOptions contains playback-specific configuration
type PlaybackOptions struct {
	FilePath string  `json:"filePath"`
	Rate     float32 `json:"rate"`  // 0.25x to 1.25x
	Pitch    float32 `json:"pitch"` // ±12 semitones

	// Native player instance (not serialized)
	playerPtr unsafe.Pointer `json:"-"`
}

// InputOptions contains input-specific configuration
type InputOptions struct {
	Device       *devices.AudioDevice `json:"device"`       // Complete device info with capabilities
	ChannelIndex int                  `json:"channelIndex"` // Channel index on the device
	PluginChain  *PluginChain         `json:"pluginChain"`  // Effects chain
}

// IsInput returns true if this is an input channel
func (c *Channel) IsInput() bool {
	return c.InputOptions != nil
}

// IsPlayback returns true if this is a playback channel
func (c *Channel) IsPlayback() bool {
	return c.PlaybackOptions != nil
}

// SetVolume sets the volume for this channel (0.0 to 1.0)
func (c *Channel) SetVolume(volume float32) error {
	if c.mixerNodePtr == nil {
		return errors.New("no mixer node available for this channel")
	}

	// Validate the volume parameter
	if err := ValidateVolume(volume); err != nil {
		return err
	}

	// Update the stored volume with validated value
	c.Volume = volume

	// Set volume on the channel's mixer node (input bus 0)
	errorStr := C.audiomixer_set_volume(c.mixerNodePtr, C.float(volume), 0)
	if errorStr != nil {
		return errors.New("failed to set channel volume: " + C.GoString(errorStr))
	}

	return nil
}

// GetVolume returns the current volume for this channel
func (c *Channel) GetVolume() (float32, error) {
	if c.mixerNodePtr == nil {
		return 0.0, errors.New("no mixer node available for this channel")
	}

	var volume C.float
	errorStr := C.audiomixer_get_volume(c.mixerNodePtr, 0, &volume)
	if errorStr != nil {
		return 0.0, errors.New("failed to get channel volume: " + C.GoString(errorStr))
	}

	// Update cached value
	c.Volume = float32(volume)
	return float32(volume), nil
}

// SetPan sets the pan for this channel (-1.0 to 1.0)
func (c *Channel) SetPan(pan float32) error {
	if c.mixerNodePtr == nil {
		return errors.New("no mixer node available for this channel")
	}

	// Validate the pan parameter
	if err := ValidatePan(pan); err != nil {
		return err
	}

	// Update the stored pan with validated value
	c.Pan = pan

	// Set pan on the channel's mixer node (input bus 0)
	errorStr := C.audiomixer_set_pan(c.mixerNodePtr, C.float(pan), 0)
	if errorStr != nil {
		return errors.New("failed to set channel pan: " + C.GoString(errorStr))
	}

	return nil
}

// GetPan returns the current pan for this channel
func (c *Channel) GetPan() (float32, error) {
	if c.mixerNodePtr == nil {
		return 0.0, errors.New("no mixer node available for this channel")
	}

	var pan C.float
	errorStr := C.audiomixer_get_pan(c.mixerNodePtr, 0, &pan)
	if errorStr != nil {
		return 0.0, errors.New("failed to get channel pan: " + C.GoString(errorStr))
	}

	// Update cached value
	c.Pan = float32(pan)
	return float32(pan), nil
}

// DestroyChannel removes a channel and frees its bus
func (e *Engine) DestroyChannel(index int) error {
	if index < 0 || index >= len(e.Channels) {
		return errors.New("invalid channel index")
	}

	if e.Channels[index] == nil {
		return errors.New("channel slot already empty")
	}

	channel := e.Channels[index]

	// Free the bus allocated to this channel
	err := e.FreeBusForChannel(channel)
	if err != nil {
		// Log the warning but don't fail the destruction
		// The channel might not have had a bus allocated yet
	}

	// TODO: Disconnect channel from mixer bus
	// TODO: Clean up channel resources (playerPtr, mixerNodePtr, etc.)

	// Remove channel from slice
	e.Channels = append(e.Channels[:index], e.Channels[index+1:]...)
	return nil
}
