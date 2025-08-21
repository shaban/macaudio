package engine

/*
#cgo CFLAGS: -I../../
#cgo LDFLAGS: -L../../ -lmacaudio

#include <stdlib.h>
#include "../../native/macaudio.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// TapMetrics contains current metrics from an audio tap
type TapMetrics struct {
	RMS        float64   // Root Mean Square level
	FrameCount int       // Number of frames in last buffer
	LastUpdate time.Time // When metrics were last updated
}

// TapInfo contains information about an installed audio tap
type TapInfo struct {
	TapID        unsafe.Pointer
	NodePtr      unsafe.Pointer
	BusIndex     int
	IsActive     bool
	SampleRate   float64
	ChannelCount int
}

// Tap represents an audio tap for monitoring signal flow (unified dylib version)
type Tap struct {
	key       string         // Human-readable identifier like "test_output_bus0"
	enginePtr unsafe.Pointer // AVAudioEngine pointer
	nodePtr   unsafe.Pointer // AVAudioNode pointer
	busIndex  int            // Bus index for the tap
	installed bool           // Whether tap is currently installed
}

// Global tap registry for Go-side bookkeeping
var (
	tapRegistry = make(map[string]*Tap)
	tapMutex    sync.RWMutex
)

// InstallTapWithKey installs a tap with a specific string key for identification
func InstallTapWithKey(enginePtr, nodePtr unsafe.Pointer, busIndex int, key string) (*Tap, error) {
	tapMutex.Lock()
	defer tapMutex.Unlock()

	// Check if key is already in use
	if existing, exists := tapRegistry[key]; exists && existing.installed {
		return nil, fmt.Errorf("tap key '%s' is already in use", key)
	}

	// Convert key to C string
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	// Call unified dylib function
	result := C.tap_install(enginePtr, nodePtr, C.int(busIndex), cKey)
	if result != nil {
		return nil, errors.New(C.GoString(result))
	}

	// Create and register the tap
	tap := &Tap{
		key:       key,
		enginePtr: enginePtr,
		nodePtr:   nodePtr,
		busIndex:  busIndex,
		installed: true,
	}

	tapRegistry[key] = tap
	return tap, nil
}

// InstallTap installs a tap with an auto-generated key (for compatibility)
func InstallTap(enginePtr, nodePtr unsafe.Pointer, busIndex int) (*Tap, error) {
	key := fmt.Sprintf("tap_%p_bus%d_%d", nodePtr, busIndex, time.Now().UnixNano())
	return InstallTapWithKey(enginePtr, nodePtr, busIndex, key)
}

// Remove removes the tap and cleans up resources
func (t *Tap) Remove() error {
	tapMutex.Lock()
	defer tapMutex.Unlock()

	if !t.installed {
		return fmt.Errorf("tap is not installed")
	}

	// Convert key to C string
	cKey := C.CString(t.key)
	defer C.free(unsafe.Pointer(cKey))

	// Call unified dylib function
	result := C.tap_remove(cKey)
	if result != nil {
		return errors.New(C.GoString(result))
	}

	t.installed = false
	delete(tapRegistry, t.key)
	return nil
}

// GetMetrics returns current audio metrics from the tap
func (t *Tap) GetMetrics() (*TapMetrics, error) {
	if !t.installed {
		return nil, fmt.Errorf("tap is not installed")
	}

	// Convert key to C string
	cKey := C.CString(t.key)
	defer C.free(unsafe.Pointer(cKey))

	// Get RMS
	var rms C.double
	result := C.tap_get_rms(cKey, &rms)
	if result != nil {
		return nil, errors.New(C.GoString(result))
	}

	// Get frame count
	var frameCount C.int
	result = C.tap_get_frame_count(cKey, &frameCount)
	if result != nil {
		return nil, errors.New(C.GoString(result))
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

// GetKey returns the tap's key identifier
func (t *Tap) GetKey() string {
	return t.key
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

// RemoveAllTaps removes all active taps (useful for cleanup)
func RemoveAllTaps() error {
	result := C.tap_remove_all()
	if result != nil {
		return errors.New(C.GoString(result))
	}

	// Clear Go-side registry
	tapMutex.Lock()
	defer tapMutex.Unlock()
	for _, tap := range tapRegistry {
		tap.installed = false
	}
	tapRegistry = make(map[string]*Tap)

	return nil
}

// GetActiveTapCount returns the number of currently active taps
func GetActiveTapCount() (int, error) {
	var count C.int
	result := C.tap_get_active_count(&count)
	if result != nil {
		return 0, errors.New(C.GoString(result))
	}
	return int(count), nil
}
