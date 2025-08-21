package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -L../../ -lmacaudio -Wl,-rpath,/Users/shaban/Code/macaudio
#include "../../native/macaudio.h"
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"
)

// AudioSegmentMetrics contains analysis results for a segment of audio
type AudioSegmentMetrics struct {
	RMS        float64   // Root Mean Square level of the audio
	FrameCount int       // Number of frames analyzed
	StartTime  float64   // Start time of the analyzed segment
	Duration   float64   // Duration of the analyzed segment
	Timestamp  time.Time // When the analysis was performed
}

// AudioPlayer represents an audio file player that can be connected to an engine
type AudioPlayer struct {
	ptr    *C.AudioPlayer
	engine *Engine // Reference to the engine this player belongs to
}

// FileInfo contains information about the loaded audio file
type FileInfo struct {
	SampleRate   float64
	ChannelCount int
	Format       string
	Duration     time.Duration
}

// NewPlayer creates a new audio player attached to the given engine
func (e *Engine) NewPlayer() (*AudioPlayer, error) {
	if e == nil || e.ptr == nil {
		return nil, errors.New("engine is nil")
	}

	// Get the native engine pointer
	enginePtr := e.GetNativeEngine()
	if enginePtr == nil {
		return nil, errors.New("failed to get native engine pointer")
	}

	result := C.audioplayer_new(enginePtr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	if result.result == nil {
		return nil, errors.New("player creation returned null pointer")
	}

	player := &AudioPlayer{
		ptr:    (*C.AudioPlayer)(result.result),
		engine: e,
	}

	return player, nil
}

// LoadFile loads an audio file for playback
// Supported formats: WAV, AIFF, MP3, AAC, M4A, FLAC (macOS 11+)
func (p *AudioPlayer) LoadFile(filePath string) error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	cPath := C.CString(filePath)
	defer C.free(unsafe.Pointer(cPath))

	result := C.audioplayer_load_file(p.ptr, cPath)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// Play starts playback of the loaded audio file from the beginning
func (p *AudioPlayer) Play() error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	result := C.audioplayer_play(p.ptr)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// PlayAt starts playback from a specific time position in seconds
func (p *AudioPlayer) PlayAt(timeSeconds float64) error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	if timeSeconds < 0.0 {
		return errors.New("time cannot be negative")
	}

	result := C.audioplayer_play_at_time(p.ptr, C.double(timeSeconds))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// Pause pauses playback (can be resumed with Play)
func (p *AudioPlayer) Pause() error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	result := C.audioplayer_pause(p.ptr)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// Stop stops playback completely
func (p *AudioPlayer) Stop() error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	result := C.audioplayer_stop(p.ptr)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// IsPlaying returns true if the player is currently playing audio
func (p *AudioPlayer) IsPlaying() (bool, error) {
	if p == nil || p.ptr == nil {
		return false, errors.New("player is nil")
	}

	var isPlaying C.bool
	result := C.audioplayer_is_playing(p.ptr, &isPlaying)
	if result != nil {
		return false, errors.New(C.GoString(result))
	}

	return bool(isPlaying), nil
}

// GetDuration returns the duration of the loaded audio file
func (p *AudioPlayer) GetDuration() (time.Duration, error) {
	if p == nil || p.ptr == nil {
		return 0, errors.New("player is nil")
	}

	var duration C.double
	result := C.audioplayer_get_duration(p.ptr, &duration)
	if result != nil {
		return 0, errors.New(C.GoString(result))
	}

	return time.Duration(float64(duration) * float64(time.Second)), nil
}

// GetCurrentTime returns the current playback position (approximation)
func (p *AudioPlayer) GetCurrentTime() (time.Duration, error) {
	if p == nil || p.ptr == nil {
		return 0, errors.New("player is nil")
	}

	var currentTime C.double
	result := C.audioplayer_get_current_time(p.ptr, &currentTime)
	if result != nil {
		return 0, errors.New(C.GoString(result))
	}

	return time.Duration(float64(currentTime) * float64(time.Second)), nil
}

// SeekTo seeks to a specific time position in the audio file
func (p *AudioPlayer) SeekTo(timeSeconds float64) error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	if timeSeconds < 0.0 {
		return errors.New("time cannot be negative")
	}

	result := C.audioplayer_seek_to_time(p.ptr, C.double(timeSeconds))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// SetVolume sets the player volume (0.0 to 1.0)
func (p *AudioPlayer) SetVolume(volume float32) error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}

	result := C.audioplayer_set_volume(p.ptr, C.float(volume))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// GetVolume returns the current player volume
func (p *AudioPlayer) GetVolume() (float32, error) {
	if p == nil || p.ptr == nil {
		return 0.0, errors.New("player is nil")
	}

	var volume C.float
	result := C.audioplayer_get_volume(p.ptr, &volume)
	if result != nil {
		return 0.0, errors.New(C.GoString(result))
	}

	return float32(volume), nil
}

// SetPan sets the stereo pan (-1.0 = left, 0.0 = center, 1.0 = right)
func (p *AudioPlayer) SetPan(pan float32) error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	if pan < -1.0 || pan > 1.0 {
		return errors.New("pan must be between -1.0 and 1.0")
	}

	result := C.audioplayer_set_pan(p.ptr, C.float(pan))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// GetPan returns the current stereo pan setting
func (p *AudioPlayer) GetPan() (float32, error) {
	if p == nil || p.ptr == nil {
		return 0.0, errors.New("player is nil")
	}

	var pan C.float
	result := C.audioplayer_get_pan(p.ptr, &pan)
	if result != nil {
		return 0.0, errors.New(C.GoString(result))
	}

	return float32(pan), nil
}

// SetPlaybackRate sets the playback rate (0.25 to 4.0, where 1.0 = normal speed)
// Note: Time/pitch effects must be enabled first with EnableTimePitchEffects()
func (p *AudioPlayer) SetPlaybackRate(rate float32) error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	if rate < 0.25 || rate > 4.0 {
		return errors.New("playback rate must be between 0.25 and 4.0")
	}

	result := C.audioplayer_set_playback_rate(p.ptr, C.float(rate))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// GetPlaybackRate returns the current playback rate
func (p *AudioPlayer) GetPlaybackRate() (float32, error) {
	if p == nil || p.ptr == nil {
		return 1.0, errors.New("player is nil")
	}

	var rate C.float
	result := C.audioplayer_get_playback_rate(p.ptr, &rate)
	if result != nil {
		return 1.0, errors.New(C.GoString(result))
	}

	return float32(rate), nil
}

// SetPitch sets the pitch in cents (-2400 to 2400, where 0 = no pitch change)
// Note: Time/pitch effects must be enabled first with EnableTimePitchEffects()
// Tip: 100 cents = 1 semitone, 1200 cents = 1 octave
func (p *AudioPlayer) SetPitch(pitchCents float32) error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	if pitchCents < -2400.0 || pitchCents > 2400.0 {
		return errors.New("pitch must be between -2400 and 2400 cents")
	}

	result := C.audioplayer_set_pitch(p.ptr, C.float(pitchCents))
	if result != nil {
		return errors.New(C.GoString(result))
	}

	return nil
}

// GetPitch returns the current pitch setting in cents
func (p *AudioPlayer) GetPitch() (float32, error) {
	if p == nil || p.ptr == nil {
		return 0.0, errors.New("player is nil")
	}

	var pitch C.float
	result := C.audioplayer_get_pitch(p.ptr, &pitch)
	if result != nil {
		return 0.0, errors.New(C.GoString(result))
	}

	return float32(pitch), nil
}

// EnableTimePitchEffects enables time stretching and pitch shifting capabilities
// This must be called before using SetPlaybackRate() or SetPitch()
//
// IMPORTANT: After calling this function, you need to restart the audio engine
// for the TimePitch effects to work properly:
//
//	engine.Stop()
//	time.Sleep(100 * time.Millisecond)  // Allow cleanup
//	engine.Start()
//
// Note: This only creates the TimePitch unit - you must call ConnectTo* methods
// to establish audio routing after restarting the engine.
func (p *AudioPlayer) EnableTimePitchEffects() error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	result := C.audioplayer_enable_time_pitch_effects(p.ptr)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	// TimePitch unit is now created but not connected
	// Caller must explicitly connect to desired destination after engine restart
	return nil
}

// DisableTimePitchEffects disables time stretching and pitch shifting
// This will reset playback rate to 1.0 and pitch to 0 cents
// Note: You may need to reconnect to your desired destination after disabling
func (p *AudioPlayer) DisableTimePitchEffects() error {
	if p == nil || p.ptr == nil {
		return errors.New("player is nil")
	}

	// Clean disconnect before disabling
	if err := p.disconnectFromCurrentDestination(); err != nil {
		// Log but don't fail - disconnection issues shouldn't block TimePitch disabling
	}

	result := C.audioplayer_disable_time_pitch_effects(p.ptr)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	// TimePitch unit is now disabled - caller must reconnect to desired destination
	return nil
}

// IsTimePitchEffectsEnabled returns true if time/pitch effects are currently enabled
func (p *AudioPlayer) IsTimePitchEffectsEnabled() (bool, error) {
	if p == nil || p.ptr == nil {
		return false, errors.New("player is nil")
	}

	var enabled C.bool
	result := C.audioplayer_is_time_pitch_effects_enabled(p.ptr, &enabled)
	if result != nil {
		return false, errors.New(C.GoString(result))
	}

	return bool(enabled), nil
}

// GetTimePitchNodePtr returns the TimePitch unit's node pointer for connecting to other nodes
func (p *AudioPlayer) GetTimePitchNodePtr() (unsafe.Pointer, error) {
	if p == nil || p.ptr == nil {
		return nil, errors.New("player is nil")
	}

	result := C.audioplayer_get_time_pitch_node_ptr(p.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return unsafe.Pointer(result.result), nil
}

// GetNodePtr returns the player's internal node pointer for connecting to other nodes
func (p *AudioPlayer) GetNodePtr() (unsafe.Pointer, error) {
	if p == nil || p.ptr == nil {
		return nil, errors.New("player is nil")
	}

	result := C.audioplayer_get_node_ptr(p.ptr)
	if result.error != nil {
		return nil, errors.New(C.GoString(result.error))
	}

	return unsafe.Pointer(result.result), nil
}

// GetFileInfo returns information about the loaded audio file
func (p *AudioPlayer) GetFileInfo() (*FileInfo, error) {
	if p == nil || p.ptr == nil {
		return nil, errors.New("player is nil")
	}

	var sampleRate C.double
	var channelCount C.int
	var format *C.char

	result := C.audioplayer_get_file_info(p.ptr, &sampleRate, &channelCount, &format)
	if result != nil {
		return nil, errors.New(C.GoString(result))
	}

	duration, err := p.GetDuration()
	if err != nil {
		duration = 0
	}

	info := &FileInfo{
		SampleRate:   float64(sampleRate),
		ChannelCount: int(channelCount),
		Format:       C.GoString(format),
		Duration:     duration,
	}

	return info, nil
}

// =====================================================
// CLEAN CONNECTION ARCHITECTURE
// =====================================================

// ConnectTo connects this player's output to any destination node
// This is the generic connection method with no assumptions about destinations
// It automatically handles TimePitch routing if enabled
func (p *AudioPlayer) ConnectTo(destinationNode unsafe.Pointer, outputBus, inputBus int) error {
	if p == nil || p.ptr == nil || p.engine == nil {
		return errors.New("player or engine is nil")
	}

	// Get our actual output node (player directly or TimePitch unit if enabled)
	outputNode, err := p.getEffectiveOutputNode()
	if err != nil {
		return err
	}

	// Clean disconnect from any existing destination first
	// Note: This is more thorough than the original implementation
	if err := p.disconnectFromCurrentDestination(); err != nil {
		// Log but don't fail - disconnection issues shouldn't block new connections
		// In production, you might want to log this
	}

	// Connect to new destination using engine's Connect method
	return p.engine.Connect(outputNode, destinationNode, outputBus, inputBus)
}

// ConnectToMixer connects to any mixer node (convenience method)
func (p *AudioPlayer) ConnectToMixer(mixerNode unsafe.Pointer, mixerInputBus int) error {
	return p.ConnectTo(mixerNode, 0, mixerInputBus)
}

// getEffectiveOutputNode returns the actual output node for this player
// If TimePitch is enabled, returns TimePitch unit; otherwise returns player node
func (p *AudioPlayer) getEffectiveOutputNode() (unsafe.Pointer, error) {
	timePitchEnabled, err := p.IsTimePitchEffectsEnabled()
	if err != nil {
		// If we can't determine TimePitch status, assume disabled
		timePitchEnabled = false
	}

	if timePitchEnabled {
		// Ensure TimePitch unit is properly connected to player
		if err := p.ensureTimePitchConnected(); err != nil {
			return nil, errors.New("failed to ensure TimePitch connection: " + err.Error())
		}
		return p.GetTimePitchNodePtr()
	}

	return p.GetNodePtr()
}

// ensureTimePitchConnected ensures Player -> TimePitch connection exists
// This is called automatically when needed
func (p *AudioPlayer) ensureTimePitchConnected() error {
	playerNodePtr, err := p.GetNodePtr()
	if err != nil {
		return err
	}

	timePitchNodePtr, err := p.GetTimePitchNodePtr()
	if err != nil {
		return err
	}

	// Check if connection already exists (to avoid redundant connections)
	// For now, we'll always reconnect to ensure proper routing
	// In production, you might want to add connection state tracking

	// Clean TimePitch input first
	p.engine.DisconnectNodeInput(timePitchNodePtr, 0)

	// Connect Player -> TimePitch with engine-compatible format
	engineFormat, err := p.engine.GetEngineFormat()
	if err != nil {
		// Fallback to nil format
		return p.engine.ConnectWithFormat(playerNodePtr, timePitchNodePtr, 0, 0, nil)
	}

	defer engineFormat.Destroy()
	return p.engine.ConnectWithFormat(playerNodePtr, timePitchNodePtr, 0, 0, engineFormat.GetPtr())
}

// disconnectFromCurrentDestination cleans up existing connections
// This provides more thorough cleanup than the original implementation
func (p *AudioPlayer) disconnectFromCurrentDestination() error {
	timePitchEnabled, _ := p.IsTimePitchEffectsEnabled()

	if timePitchEnabled {
		// If TimePitch is enabled, only disconnect TimePitch unit from its outputs
		// We preserve the Player->TimePitch connection since it should remain stable
		timePitchPtr, err := p.GetTimePitchNodePtr()
		if err != nil {
			return err
		}

		// Disconnect TimePitch from whatever destination it was connected to
		// NOTE: We do NOT disconnect the Player->TimePitch connection here
		if err := p.engine.DisconnectNodeOutput(timePitchPtr, 0); err != nil {
			// Non-fatal - the connection might not exist
		}

	} else {
		// If no TimePitch, disconnect player directly from its outputs
		playerNodePtr, err := p.GetNodePtr()
		if err != nil {
			return err
		}

		// Disconnect player from whatever it was connected to
		if err := p.engine.DisconnectNodeOutput(playerNodePtr, 0); err != nil {
			// Non-fatal - the connection might not exist
		}
	}

	return nil
}

// ConnectToMainMixer connects the player to the engine's main mixer for output
// This is now a convenience wrapper that uses the cleaner architecture
func (p *AudioPlayer) ConnectToMainMixer() error {
	mainMixerPtr, err := p.engine.MainMixerNode()
	if err != nil {
		return errors.New("failed to get main mixer: " + err.Error())
	}

	return p.ConnectToMixer(mainMixerPtr, 0)
}

// AnalyzeFileSegment analyzes a specific time segment of the loaded audio file
// This reads the raw audio data from the file (same data that gets played) and calculates metrics
// This is more reliable than tapping audio streams since it analyzes ground truth data
func (p *AudioPlayer) AnalyzeFileSegment(startTime, duration float64) (*AudioSegmentMetrics, error) {
	if p == nil || p.ptr == nil {
		return nil, errors.New("player is nil")
	}

	var rms C.double
	var frameCount C.int

	result := C.audioplayer_analyze_file_segment(p.ptr, C.double(startTime), C.double(duration), &rms, &frameCount)
	if result != nil {
		return nil, fmt.Errorf("analysis failed: %s", C.GoString(result))
	}

	return &AudioSegmentMetrics{
		RMS:        float64(rms),
		FrameCount: int(frameCount),
		StartTime:  startTime,
		Duration:   duration,
		Timestamp:  time.Now(),
	}, nil
}

// AnalyzeCurrentPlayback analyzes the audio data that should be playing at the current time
// This is useful for real-time monitoring during playback
func (p *AudioPlayer) AnalyzeCurrentPlayback(duration float64) (*AudioSegmentMetrics, error) {
	currentTime, err := p.GetCurrentTime()
	if err != nil {
		return nil, fmt.Errorf("failed to get current time: %w", err)
	}

	// Convert nanoseconds to seconds
	currentTimeSeconds := currentTime.Seconds()

	return p.AnalyzeFileSegment(currentTimeSeconds, duration)
}

// Destroy cleans up the player and frees resources
func (p *AudioPlayer) Destroy() {
	if p == nil || p.ptr == nil {
		return
	}

	C.audioplayer_destroy(p.ptr)
	p.ptr = nil
	p.engine = nil
}
