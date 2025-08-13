// Package tap provides audio tap functionality for monitoring and testing audio signals.
// It allows installing taps on AVAudioNode buses to capture audio metrics and data
// for analysis, monitoring, and debugging purposes.
package tap

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#include "native/tap.m"
#include <stdlib.h>

// Function declarations - now using string keys instead of void* pointers
const char* tap_install(void* enginePtr, void* nodePtr, int busIndex, const char* tapKey);
const char* tap_remove(const char* tapKey);
const char* tap_get_info(const char* tapKey, TapInfo* info);
const char* tap_get_rms(const char* tapKey, double* result);
const char* tap_get_frame_count(const char* tapKey, int* result);
const char* tap_remove_all(void);
const char* tap_get_active_count(int* result);
*/
import "C"
import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"
	"unsafe"
)

// TapInfo contains information about an installed audio tap
type TapInfo struct {
	TapID        unsafe.Pointer
	NodePtr      unsafe.Pointer
	BusIndex     int
	IsActive     bool
	SampleRate   float64
	ChannelCount int
}

// TapMetrics contains current metrics from an audio tap
type TapMetrics struct {
	RMS        float64   // Root Mean Square level
	FrameCount int       // Number of frames in last buffer
	LastUpdate time.Time // When metrics were last updated
}

// Global tap registry (Go side owns the bookkeeping)
var (
	tapRegistry = make(map[string]*Tap)
	tapMutex    sync.RWMutex
)

// Tap represents an audio tap for monitoring signal flow
type Tap struct {
	key       string         // Human-readable identifier like "test_output_bus0"
	enginePtr unsafe.Pointer // AVAudioEngine pointer
	nodePtr   unsafe.Pointer // AVAudioNode pointer
	busIndex  int            // Bus index for the tap
	installed bool           // Whether tap is currently installed
}

// isValidTapKey validates that a tap key contains only safe characters
func isValidTapKey(key string) bool {
	// Allow alphanumeric, underscore, hyphen
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return validPattern.MatchString(key)
}

// InstallTapWithKey installs a tap with a specific string key for identification
func InstallTapWithKey(enginePtr, nodePtr unsafe.Pointer, busIndex int, key string) (*Tap, error) {
	if enginePtr == nil {
		return nil, fmt.Errorf("engine pointer cannot be nil")
	}
	if nodePtr == nil {
		return nil, fmt.Errorf("node pointer cannot be nil")
	}
	if busIndex < 0 {
		return nil, fmt.Errorf("bus index must be non-negative")
	}
	if key == "" || !isValidTapKey(key) {
		return nil, fmt.Errorf("invalid tap key: must be non-empty alphanumeric with underscore/hyphen")
	}

	tapMutex.Lock()
	defer tapMutex.Unlock()

	// STRICT: Check for key collision
	if _, exists := tapRegistry[key]; exists {
		return nil, fmt.Errorf("🚨 TAP KEY COLLISION: '%s' already exists - remove existing tap first", key)
	}

	tap := &Tap{
		key:       key,
		enginePtr: enginePtr,
		nodePtr:   nodePtr,
		busIndex:  busIndex,
		installed: false,
	}

	// Convert key to C string
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	// Install the tap using string key
	errorStr := C.tap_install(enginePtr, nodePtr, C.int(busIndex), cKey)
	if errorStr != nil {
		return nil, errors.New(C.GoString(errorStr))
	}

	tap.installed = true
	tapRegistry[key] = tap
	return tap, nil
}

// InstallTap installs a tap with an auto-generated key (for compatibility)
func InstallTap(enginePtr, nodePtr unsafe.Pointer, busIndex int) (*Tap, error) {
	// Generate a unique key based on pointers and timestamp
	key := fmt.Sprintf("tap_node%p_bus%d_%d", nodePtr, busIndex, time.Now().UnixNano())
	return InstallTapWithKey(enginePtr, nodePtr, busIndex, key)
}

// Remove removes the tap from the audio node
func (t *Tap) Remove() error {
	if !t.installed {
		return fmt.Errorf("tap is not installed")
	}

	tapMutex.Lock()
	defer tapMutex.Unlock()

	// Convert key to C string
	cKey := C.CString(t.key)
	defer C.free(unsafe.Pointer(cKey))

	errorStr := C.tap_remove(cKey)
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}

	t.installed = false
	delete(tapRegistry, t.key)
	return nil
}

// GetInfo returns information about the tap
func (t *Tap) GetInfo() (*TapInfo, error) {
	if !t.installed {
		return nil, fmt.Errorf("tap is not installed")
	}

	// Convert key to C string
	cKey := C.CString(t.key)
	defer C.free(unsafe.Pointer(cKey))

	var info C.TapInfo
	errorStr := C.tap_get_info(cKey, &info)
	if errorStr != nil {
		return nil, errors.New(C.GoString(errorStr))
	}

	return &TapInfo{
		TapID:        unsafe.Pointer(info.tapPtr),
		NodePtr:      unsafe.Pointer(info.nodePtr),
		BusIndex:     int(info.busIndex),
		IsActive:     bool(info.isActive),
		SampleRate:   float64(info.sampleRate),
		ChannelCount: int(info.channelCount),
	}, nil
}

// GetMetrics returns current audio metrics from the tap
func (t *Tap) GetMetrics() (*TapMetrics, error) {
	if !t.installed {
		return nil, fmt.Errorf("tap is not installed")
	}

	// Convert key to C string
	cKey := C.CString(t.key)
	defer C.free(unsafe.Pointer(cKey))

	var rms C.double
	errorStr := C.tap_get_rms(cKey, &rms)
	if errorStr != nil {
		return nil, errors.New(C.GoString(errorStr))
	}

	var frameCount C.int
	errorStr = C.tap_get_frame_count(cKey, &frameCount)
	if errorStr != nil {
		return nil, errors.New(C.GoString(errorStr))
	}

	return &TapMetrics{
		RMS:        float64(rms),
		FrameCount: int(frameCount),
		LastUpdate: time.Now(),
	}, nil
}

// IsInstalled returns true if the tap is currently installed
func (t *Tap) IsInstalled() bool {
	return t.installed
}

// GetBusIndex returns the bus index being tapped
func (t *Tap) GetBusIndex() int {
	return t.busIndex
}

// GetNodePtr returns the node pointer being tapped
func (t *Tap) GetNodePtr() unsafe.Pointer {
	return t.nodePtr
}

// WaitForActivity waits for audio activity on the tap with a timeout
func (t *Tap) WaitForActivity(timeout time.Duration, minRMS float64) (bool, error) {
	if !t.installed {
		return false, fmt.Errorf("tap is not installed")
	}

	start := time.Now()
	for time.Since(start) < timeout {
		metrics, err := t.GetMetrics()
		if err != nil {
			return false, err
		}

		if metrics.RMS >= minRMS && metrics.FrameCount > 0 {
			return true, nil
		}

		time.Sleep(10 * time.Millisecond) // Small delay between checks
	}

	return false, nil // Timeout reached without activity
}

// Package-level functions

// RemoveAllTaps removes all active taps (useful for cleanup)
func RemoveAllTaps() error {
	errorStr := C.tap_remove_all()
	if errorStr != nil {
		return errors.New(C.GoString(errorStr))
	}
	return nil
}

// GetActiveTapCount returns the number of currently active taps
func GetActiveTapCount() (int, error) {
	var result C.int
	errorStr := C.tap_get_active_count(&result)
	if errorStr != nil {
		return 0, errors.New(C.GoString(errorStr))
	}
	return int(result), nil
}

// WaitForSignal is a utility function to wait for audio signal on any tap
func WaitForSignal(taps []*Tap, timeout time.Duration, minRMS float64) (*Tap, error) {
	start := time.Now()

	for time.Since(start) < timeout {
		for _, tap := range taps {
			if !tap.IsInstalled() {
				continue
			}

			metrics, err := tap.GetMetrics()
			if err != nil {
				continue
			}

			if metrics.RMS >= minRMS && metrics.FrameCount > 0 {
				return tap, nil
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for signal")
}
