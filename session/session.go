//go:build darwin && cgo

package session

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AVFoundation
#include "native/session.h"
*/
import "C"
import (
	"sync"
	"time"
	"unsafe"
)

// AudioSpec - our configuration request
type AudioSpec struct {
	SampleRate   float64 `json:"sample_rate"`
	ChannelCount int     `json:"channel_count"`
	BitDepth     int     `json:"bit_depth"`
	BufferSize   int     `json:"buffer_size"`
}

// EngineStatus - minimal performance info
type EngineStatus struct {
	AudioSpec        AudioSpec `json:"audio_spec"`         // What we configured
	LastConfigChange time.Time `json:"last_config_change"` // When config changed
}

// Global state (minimal)
var (
	enginePtr        unsafe.Pointer
	currentSpec      AudioSpec
	lastConfigChange time.Time
	configMutex      sync.RWMutex
	changeCallback   func() // Notify consuming app
)

// SetEngine - register engine for monitoring
func SetEngine(engine unsafe.Pointer, spec AudioSpec) {
	configMutex.Lock()
	defer configMutex.Unlock()

	enginePtr = engine
	currentSpec = spec

	// Set up configuration change monitoring
	C.macaudio_setup_config_monitoring(engine)
}

// SetConfigurationChangeCallback - app provides callback for topology changes
func SetConfigurationChangeCallback(callback func()) {
	configMutex.Lock()
	defer configMutex.Unlock()

	changeCallback = callback
}

// GetEngineStatus - check performance
func GetEngineStatus() EngineStatus {
	configMutex.RLock()
	defer configMutex.RUnlock()

	return EngineStatus{
		AudioSpec:        currentSpec,
		LastConfigChange: lastConfigChange,
	}
}

// Cleanup - remove monitoring
func Cleanup() {
	configMutex.Lock()
	defer configMutex.Unlock()

	C.macaudio_cleanup_config_monitoring()
	enginePtr = nil
	changeCallback = nil
}

// SimulateHotplug - test function to simulate configuration change events
func SimulateHotplug(engine unsafe.Pointer) {
	C.macaudio_simulate_hotplug(engine)
}

// Configuration change callback (called from native)
//
//export configurationChanged
func configurationChanged() {
	configMutex.Lock()
	lastConfigChange = time.Now()
	callback := changeCallback
	configMutex.Unlock()

	// Notify consuming app
	if callback != nil {
		callback()
	}
}
