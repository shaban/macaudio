package macaudio

import (
	"fmt"
	"sync"
	"time"

	"github.com/shaban/macaudio/devices"
)

// DeviceMonitor handles device change detection and hotplug events
type DeviceMonitor struct {
	engine           *Engine
	mu               sync.RWMutex
	isRunning        bool
	pollingInterval  time.Duration
	
	// Adaptive polling
	baseInterval     time.Duration  // Base polling interval (50ms)
	maxInterval      time.Duration  // Max interval when no changes (200ms)
	currentInterval  time.Duration  // Current adaptive interval
	lastChangeTime   time.Time      // Last time devices changed
	noChangeCount    int            // Consecutive polls with no changes
	
	// Device state tracking
	lastAudioCount   int
	lastMidiCount    int
	
	// Performance tracking
	averageCheckTime time.Duration
	maxCheckTime     time.Duration
	checkCount       int64
	
	// Callbacks for device events
	onAudioDeviceAdded    func(device devices.AudioDevice)
	onAudioDeviceRemoved  func(deviceUID string)
	onMidiDeviceAdded     func(device devices.MIDIDevice)
	onMidiDeviceRemoved   func(deviceUID string)
	onDeviceStatusChanged func(deviceUID string, isOnline bool)
}

// NewDeviceMonitor creates a new device monitor
func NewDeviceMonitor(engine *Engine) *DeviceMonitor {
	return &DeviceMonitor{
		engine:           engine,
		pollingInterval:  50 * time.Millisecond, // 50ms as specified
		baseInterval:     50 * time.Millisecond,
		maxInterval:      200 * time.Millisecond,
		currentInterval:  50 * time.Millisecond,
		lastChangeTime:   time.Now(),
	}
}

// Start begins device monitoring with 50ms polling
func (dm *DeviceMonitor) Start() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	if dm.isRunning {
		return fmt.Errorf("device monitor is already running")
	}
	
	// Get initial device counts
	audioCount, midiCount, err := devices.GetDeviceCounts()
	if err != nil {
		return fmt.Errorf("failed to get initial device counts: %w", err)
	}
	
	dm.lastAudioCount = audioCount
	dm.lastMidiCount = midiCount
	dm.isRunning = true
	
	// Start monitoring goroutine
	go dm.monitorLoop()
	
	return nil
}

// Stop halts device monitoring
func (dm *DeviceMonitor) Stop() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	if !dm.isRunning {
		return nil // Already stopped
	}
	
	dm.isRunning = false
	return nil
}

// IsRunning returns whether device monitoring is active
func (dm *DeviceMonitor) IsRunning() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.isRunning
}

// SetCallbacks configures device event callbacks
func (dm *DeviceMonitor) SetCallbacks(
	onAudioAdded func(devices.AudioDevice),
	onAudioRemoved func(string),
	onMidiAdded func(devices.MIDIDevice),
	onMidiRemoved func(string),
	onStatusChanged func(string, bool),
) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	dm.onAudioDeviceAdded = onAudioAdded
	dm.onAudioDeviceRemoved = onAudioRemoved
	dm.onMidiDeviceAdded = onMidiAdded
	dm.onMidiDeviceRemoved = onMidiRemoved
	dm.onDeviceStatusChanged = onStatusChanged
}

// GetPollingInterval returns the current polling interval
func (dm *DeviceMonitor) GetPollingInterval() time.Duration {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.pollingInterval
}

// SetPollingInterval updates the polling interval (minimum 10ms)
func (dm *DeviceMonitor) SetPollingInterval(interval time.Duration) error {
	if interval < 10*time.Millisecond {
		return fmt.Errorf("polling interval cannot be less than 10ms")
	}
	
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.pollingInterval = interval
	
	return nil
}

// monitorLoop runs the device monitoring loop
func (dm *DeviceMonitor) monitorLoop() {
	// Use dynamic ticker that can adjust interval
	currentInterval := dm.pollingInterval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-dm.engine.ctx.Done():
			return
		case <-ticker.C:
			if !dm.IsRunning() {
				return
			}
			
			// Check if polling interval changed
			dm.mu.RLock()
			newInterval := dm.pollingInterval
			dm.mu.RUnlock()
			
			// Reset ticker if interval changed
			if newInterval != currentInterval {
				ticker.Stop()
				ticker = time.NewTicker(newInterval)
				currentInterval = newInterval
			}
			
			// Perform device check
			dm.checkDevices()
		}
	}
}

// checkDevices performs fast device change detection
func (dm *DeviceMonitor) checkDevices() {
	start := time.Now()
	
	// Fast count-based detection first
	audioCount, midiCount, err := devices.GetDeviceCounts()
	if err != nil {
		dm.engine.errorHandler.HandleError(fmt.Errorf("device count check failed: %w", err))
		return
	}
	
	// Check for changes
	audioChanged := audioCount != dm.lastAudioCount
	midiChanged := midiCount != dm.lastMidiCount
	
	// Update performance tracking
	elapsed := time.Since(start)
	dm.updatePerformanceStats(elapsed)
	
	if !audioChanged && !midiChanged {
		// No changes - increase interval gradually for power efficiency
		dm.adaptiveSlowdown()
		return
	}
	
	// Changes detected - reset to fast polling
	dm.adaptiveSpeedup()
	
	// Update counts
	dm.lastAudioCount = audioCount
	dm.lastMidiCount = midiCount
	
	// Perform detailed enumeration for changed device types
	if audioChanged {
		dm.handleAudioDeviceChange()
	}
	
	if midiChanged {
		dm.handleMidiDeviceChange()
	}
}

// updatePerformanceStats tracks device check performance
func (dm *DeviceMonitor) updatePerformanceStats(elapsed time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	dm.checkCount++
	
	// Update running average (simple exponential moving average)
	if dm.checkCount == 1 {
		dm.averageCheckTime = elapsed
	} else {
		// EMA with alpha = 0.1 (gives more weight to recent samples)
		dm.averageCheckTime = time.Duration(float64(dm.averageCheckTime)*0.9 + float64(elapsed)*0.1)
	}
	
	// Track maximum
	if elapsed > dm.maxCheckTime {
		dm.maxCheckTime = elapsed
	}
	
	// Log only if we significantly exceed our target runtime (200μs instead of 50μs) 
	// to reduce noise during normal operation
	if elapsed > 200*time.Microsecond {
		dm.engine.errorHandler.HandleError(
			fmt.Errorf("device check took %v, target is 50μs", elapsed))
	}
}

// adaptiveSlowdown gradually increases polling interval when no changes detected
func (dm *DeviceMonitor) adaptiveSlowdown() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	dm.noChangeCount++
	
	// After 10 consecutive checks with no changes, start slowing down
	if dm.noChangeCount > 10 {
		// Gradually increase interval up to maxInterval
		newInterval := time.Duration(float64(dm.currentInterval) * 1.1)
		if newInterval > dm.maxInterval {
			newInterval = dm.maxInterval
		}
		dm.currentInterval = newInterval
		dm.pollingInterval = newInterval
	}
}

// adaptiveSpeedup resets to fast polling when changes are detected
func (dm *DeviceMonitor) adaptiveSpeedup() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	dm.noChangeCount = 0
	dm.lastChangeTime = time.Now()
	dm.currentInterval = dm.baseInterval
	dm.pollingInterval = dm.baseInterval
}

// GetPerformanceStats returns device monitoring performance statistics
func (dm *DeviceMonitor) GetPerformanceStats() (avgTime, maxTime time.Duration, checkCount int64) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.averageCheckTime, dm.maxCheckTime, dm.checkCount
}

// handleAudioDeviceChange processes audio device changes
func (dm *DeviceMonitor) handleAudioDeviceChange() {
	audioDevices, err := devices.GetAudio()
	if err != nil {
		dm.engine.errorHandler.HandleError(fmt.Errorf("audio device enumeration failed: %w", err))
		return
	}
	
	// TODO: Compare with previous device list to determine added/removed devices
	// For now, we'll just trigger callbacks if they exist
	
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	if dm.onAudioDeviceAdded != nil {
		for _, device := range audioDevices {
			// This is a simplified implementation - in practice we'd track previous state
			if device.IsOnline {
				dm.onAudioDeviceAdded(device)
			}
		}
	}
}

// handleMidiDeviceChange processes MIDI device changes
func (dm *DeviceMonitor) handleMidiDeviceChange() {
	midiDevices, err := devices.GetMIDI()
	if err != nil {
		dm.engine.errorHandler.HandleError(fmt.Errorf("MIDI device enumeration failed: %w", err))
		return
	}
	
	// TODO: Compare with previous device list to determine added/removed devices
	// For now, we'll just trigger callbacks if they exist
	
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	if dm.onMidiDeviceAdded != nil {
		for _, device := range midiDevices {
			// This is a simplified implementation - in practice we'd track previous state
			if device.IsOnline {
				dm.onMidiDeviceAdded(device)
			}
		}
	}
}

// ForceDeviceCheck triggers an immediate device check (useful for testing)
func (dm *DeviceMonitor) ForceDeviceCheck() {
	if dm.IsRunning() {
		dm.checkDevices()
	}
}
