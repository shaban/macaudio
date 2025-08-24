package engine

/*
#include "../native/macaudio.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

// =============================================================================
// Public API - Channel Management
// =============================================================================

// CreatePlaybackChannel creates a playback channel for an audio file
func (e *Engine) CreatePlaybackChannel(filePath string) (*Channel, error) {
	// Check if engine is properly initialized
	if e.nativeEngine == nil {
		return nil, errors.New("engine is not properly initialized")
	}

	// Validate file path
	if err := ValidateFilePath(filePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// TODO: Validate file format and size (200MB limit)
	channel := &Channel{
		Volume: 1.0,
		Pan:    0.0,
		PlaybackOptions: &PlaybackOptions{
			FilePath: filePath,
			Rate:     1.0, // Normal playback rate
			Pitch:    0.0, // No pitch shift
		},
	}

	// Create native player using the C API
	result := C.audioplayer_new(unsafe.Pointer(e.nativeEngine.engine))
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	// Store the native player pointer
	channel.PlaybackOptions.playerPtr = result.result

	// Load the audio file into the player
	cFilePath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cFilePath))

	playerPtr := (*C.AudioPlayer)(channel.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_load_file(playerPtr, cFilePath)
	if errorStr != nil {
		// Clean up the player if file loading fails
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to load audio file: " + C.GoString(errorStr))
	}

	// Enable time/pitch effects by default
	errorStr = C.audioplayer_enable_time_pitch_effects(playerPtr)
	if errorStr != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to enable time/pitch effects: " + C.GoString(errorStr))
	}

	// Connect the player to the engine's audio graph
	// Get the player node
	nodeResult := C.audioplayer_get_node_ptr(playerPtr)
	if nodeResult.error != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to get player node: " + C.GoString(nodeResult.error))
	}

	// Get the time/pitch node
	timePitchResult := C.audioplayer_get_time_pitch_node_ptr(playerPtr)
	if timePitchResult.error != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to get time/pitch node: " + C.GoString(timePitchResult.error))
	}

	// Create a dedicated mixer node for this channel
	channelMixerResult := C.audioengine_create_mixer_node(e.nativeEngine)
	if channelMixerResult.error != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to create channel mixer: " + C.GoString(channelMixerResult.error))
	}

	// Store the mixer node pointer in the channel
	channel.mixerNodePtr = channelMixerResult.result

	// Attach all nodes to the engine
	errorStr = C.audioengine_attach(e.nativeEngine, nodeResult.result)
	if errorStr != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to attach player to engine: " + C.GoString(errorStr))
	}

	errorStr = C.audioengine_attach(e.nativeEngine, timePitchResult.result)
	if errorStr != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to attach time/pitch unit to engine: " + C.GoString(errorStr))
	}

	errorStr = C.audioengine_attach(e.nativeEngine, channelMixerResult.result)
	if errorStr != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to attach channel mixer to engine: " + C.GoString(errorStr))
	}

	// Connect audio graph: Player → TimePitch → ChannelMixer
	errorStr = C.audioengine_connect(e.nativeEngine, nodeResult.result, timePitchResult.result, 0, 0)
	if errorStr != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to connect player to time/pitch unit: " + C.GoString(errorStr))
	}

	errorStr = C.audioengine_connect(e.nativeEngine, timePitchResult.result, channelMixerResult.result, 0, 0)
	if errorStr != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to connect time/pitch unit to channel mixer: " + C.GoString(errorStr))
	}

	// Get the main mixer node
	mainMixerResult := C.audioengine_main_mixer_node(e.nativeEngine)
	if mainMixerResult.error != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to get main mixer: " + C.GoString(mainMixerResult.error))
	}

	// Allocate a unique input bus for this channel on the main mixer
	busIndex, err := e.AllocateBusForChannel(channel)
	if err != nil {
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to allocate bus for channel: " + err.Error())
	}

	// Debug: Log the bus allocation
	// Note: This will be visible in test logs to confirm unique bus assignment
	fmt.Printf("🎛️  ALLOCATED: Channel (mixer=%p) → Main Mixer Bus %d\n", channel.mixerNodePtr, busIndex)

	// Connect channel mixer to main mixer (channel mixer output bus 0 → main mixer allocated input bus)
	errorStr = C.audioengine_connect(e.nativeEngine, channelMixerResult.result, mainMixerResult.result, 0, C.int(busIndex))
	if errorStr != nil {
		// Free the allocated bus if connection fails
		e.FreeBusForChannel(channel)
		C.audioplayer_destroy(playerPtr)
		return nil, errors.New("failed to connect channel mixer to main mixer: " + C.GoString(errorStr))
	}

	// Enable time/pitch effects for this player by default
	errorStr = C.audioplayer_enable_time_pitch_effects(playerPtr)
	if errorStr != nil {
		// This is not fatal - continue without time/pitch effects
		// Some audio formats or systems might not support it
		// But we'll log it for debugging
	}

	e.Channels = append(e.Channels, channel)
	return channel, nil
}

// PlayChannel starts playback for a playback channel
func (c *Channel) Play() error {
	if !c.IsPlayback() {
		return errors.New("channel is not a playback channel")
	}

	if c.PlaybackOptions.playerPtr == nil {
		return errors.New("no native player available")
	}

	playerPtr := (*C.AudioPlayer)(c.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_play(playerPtr)
	if errorStr != nil {
		return errors.New("failed to start playback: " + C.GoString(errorStr))
	}

	return nil
}

// EnableTimePitchEffects enables time/pitch processing for this playback channel
func (c *Channel) EnableTimePitchEffects() error {
	if !c.IsPlayback() {
		return errors.New("channel is not a playback channel")
	}

	if c.PlaybackOptions.playerPtr == nil {
		return errors.New("no native player available")
	}

	playerPtr := (*C.AudioPlayer)(c.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_enable_time_pitch_effects(playerPtr)
	if errorStr != nil {
		return errors.New("failed to enable time/pitch effects: " + C.GoString(errorStr))
	}

	return nil
}

// DisableTimePitchEffects disables time/pitch processing for this playback channel
func (c *Channel) DisableTimePitchEffects() error {
	if !c.IsPlayback() {
		return errors.New("channel is not a playback channel")
	}

	if c.PlaybackOptions.playerPtr == nil {
		return errors.New("no native player available")
	}

	playerPtr := (*C.AudioPlayer)(c.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_disable_time_pitch_effects(playerPtr)
	if errorStr != nil {
		return errors.New("failed to disable time/pitch effects: " + C.GoString(errorStr))
	}

	return nil
}

// SetPlaybackRate sets the playback rate (0.25x to 1.25x, normal = 1.0)
func (c *Channel) SetPlaybackRate(rate float32) error {
	if !c.IsPlayback() {
		return errors.New("channel is not a playback channel")
	}

	if c.PlaybackOptions.playerPtr == nil {
		return errors.New("no native player available")
	}

	// Validate the rate parameter
	if err := ValidateRate(rate); err != nil {
		return err
	}

	// Update cached value with validated value
	c.PlaybackOptions.Rate = rate

	playerPtr := (*C.AudioPlayer)(c.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_set_playback_rate(playerPtr, C.float(rate))
	if errorStr != nil {
		return errors.New("failed to set playback rate: " + C.GoString(errorStr))
	}

	return nil
}

// GetPlaybackRate returns the current playback rate
func (c *Channel) GetPlaybackRate() (float32, error) {
	if !c.IsPlayback() {
		return 0.0, errors.New("channel is not a playback channel")
	}

	if c.PlaybackOptions.playerPtr == nil {
		return 0.0, errors.New("no native player available")
	}

	var rate C.float
	playerPtr := (*C.AudioPlayer)(c.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_get_playback_rate(playerPtr, &rate)
	if errorStr != nil {
		return 0.0, errors.New("failed to get playback rate: " + C.GoString(errorStr))
	}

	// Update cached value
	c.PlaybackOptions.Rate = float32(rate)
	return float32(rate), nil
}

// SetPitch sets the pitch shift in semitones (-12 to +12, normal = 0)
func (c *Channel) SetPitch(pitch float32) error {
	if !c.IsPlayback() {
		return errors.New("channel is not a playback channel")
	}

	if c.PlaybackOptions.playerPtr == nil {
		return errors.New("no native player available")
	}

	// Validate the pitch parameter
	if err := ValidatePitch(pitch); err != nil {
		return err
	}

	// Update cached value with validated value
	c.PlaybackOptions.Pitch = pitch

	// Convert semitones to cents (1 semitone = 100 cents)
	pitchInCents := pitch * 100.0

	playerPtr := (*C.AudioPlayer)(c.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_set_pitch(playerPtr, C.float(pitchInCents))
	if errorStr != nil {
		return errors.New("failed to set pitch: " + C.GoString(errorStr))
	}

	return nil
}

// GetPitch returns the current pitch shift in semitones
func (c *Channel) GetPitch() (float32, error) {
	if !c.IsPlayback() {
		return 0.0, errors.New("channel is not a playback channel")
	}

	if c.PlaybackOptions.playerPtr == nil {
		return 0.0, errors.New("no native player available")
	}

	var pitchInCents C.float
	playerPtr := (*C.AudioPlayer)(c.PlaybackOptions.playerPtr)
	errorStr := C.audioplayer_get_pitch(playerPtr, &pitchInCents)
	if errorStr != nil {
		return 0.0, errors.New("failed to get pitch: " + C.GoString(errorStr))
	}

	// Convert cents back to semitones (1 semitone = 100 cents)
	pitchInSemitones := float32(pitchInCents) / 100.0

	// Update cached value
	c.PlaybackOptions.Pitch = pitchInSemitones
	return pitchInSemitones, nil
}
