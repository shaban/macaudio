//go:build darwin

package session

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shaban/macaudio/devices"
)

// Session provides unified macOS audio device management with async monitoring
type Session struct {
	// Cached device data (protected by RWMutex)
	audioDevices devices.AudioDevices
	midiDevices  devices.MIDIDevices
	lastUpdate   time.Time
	deviceMutex  sync.RWMutex

	// Atomic counters for lock-free count comparison
	audioCount int64
	midiCount  int64

	// Change notifications
	deviceChanges chan DeviceChange
	callbacks     []ChangeCallback
	callbackMutex sync.RWMutex

	// Configuration
	audioSpec    AudioSpec
	pollInterval time.Duration

	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	monitoring int64 // atomic bool
}

// AudioSpec defines the audio configuration
type AudioSpec struct {
	SampleRate   float64 `json:"sample_rate"`
	ChannelCount int     `json:"channel_count"`
	BitDepth     int     `json:"bit_depth"`
	BufferSize   int     `json:"buffer_size"`
}

// DeviceChange represents a device change event with async scan status
type DeviceChange struct {
	Type          ChangeType            `json:"type"`
	Timestamp     time.Time             `json:"timestamp"`
	AudioCount    int                   `json:"audio_count"`
	MIDICount     int                   `json:"midi_count"`
	AudioDevices  *devices.AudioDevices `json:"audio_devices,omitempty"`
	MIDIDevices   *devices.MIDIDevices  `json:"midi_devices,omitempty"`
	AudioScanning bool                  `json:"audio_scanning"`
	MIDIScanning  bool                  `json:"midi_scanning"`
}

type ChangeType int

const (
	AudioDeviceChange ChangeType = iota
	MIDIDeviceChange
	BothDeviceChange
)

func (ct ChangeType) String() string {
	switch ct {
	case AudioDeviceChange:
		return "audio"
	case MIDIDeviceChange:
		return "midi"
	case BothDeviceChange:
		return "both"
	default:
		return "unknown"
	}
}

// ChangeCallback for callback-style notifications
type ChangeCallback func(DeviceChange)

// Default audio configuration
var DefaultAudioSpec = AudioSpec{
	SampleRate:   48000,
	ChannelCount: 2,
	BitDepth:     32,
	BufferSize:   512,
}

// NewSession creates a new audio session with fast async monitoring
func NewSession(spec AudioSpec) (*Session, error) {
	ctx, cancel := context.WithCancel(context.Background())

	session := &Session{
		deviceChanges: make(chan DeviceChange, 10),
		audioSpec:     spec,
		pollInterval:  50 * time.Millisecond, // Fast count-based polling
		ctx:           ctx,
		cancel:        cancel,
	}

	// Initial device enumeration and count setup
	if err := session.refreshDevicesSync(); err != nil {
		cancel()
		return nil, err
	}

	// Start async monitoring
	go session.monitorDevices()
	atomic.StoreInt64(&session.monitoring, 1)

	return session, nil
}

// NewSessionWithDefaults creates a session with default audio spec
func NewSessionWithDefaults() (*Session, error) {
	return NewSession(DefaultAudioSpec)
}

// GetAudioDevices returns cached audio devices (thread-safe)
func (s *Session) GetAudioDevices() (devices.AudioDevices, error) {
	s.deviceMutex.RLock()
	defer s.deviceMutex.RUnlock()
	return s.audioDevices, nil
}

// GetMIDIDevices returns cached MIDI devices (thread-safe)
func (s *Session) GetMIDIDevices() (devices.MIDIDevices, error) {
	s.deviceMutex.RLock()
	defer s.deviceMutex.RUnlock()
	return s.midiDevices, nil
}

// DeviceChanges returns the change notification channel
func (s *Session) DeviceChanges() <-chan DeviceChange {
	return s.deviceChanges
}

// OnDeviceChange registers a callback for device changes (thread-safe)
func (s *Session) OnDeviceChange(callback ChangeCallback) {
	s.callbackMutex.Lock()
	s.callbacks = append(s.callbacks, callback)
	s.callbackMutex.Unlock()
}

// GetDeviceCounts returns current device counts (atomic read)
func (s *Session) GetDeviceCounts() (audioCount, midiCount int) {
	return int(atomic.LoadInt64(&s.audioCount)), int(atomic.LoadInt64(&s.midiCount))
}

// IsMonitoring returns true if the session is actively monitoring
func (s *Session) IsMonitoring() bool {
	return atomic.LoadInt64(&s.monitoring) == 1
}

// GetAudioSpec returns the current audio specification
func (s *Session) GetAudioSpec() AudioSpec {
	return s.audioSpec
}

// Status returns comprehensive session status
func (s *Session) Status() SessionStatus {
	s.deviceMutex.RLock()
	defer s.deviceMutex.RUnlock()

	audioCount, midiCount := s.GetDeviceCounts()
	return SessionStatus{
		Monitoring:   s.IsMonitoring(),
		AudioSpec:    s.audioSpec,
		AudioCount:   audioCount,
		MIDICount:    midiCount,
		LastUpdate:   s.lastUpdate,
		CacheAge:     time.Since(s.lastUpdate),
		PollInterval: s.pollInterval,
	}
}

type SessionStatus struct {
	Monitoring   bool          `json:"monitoring"`
	AudioSpec    AudioSpec     `json:"audio_spec"`
	AudioCount   int           `json:"audio_count"`
	MIDICount    int           `json:"midi_count"`
	LastUpdate   time.Time     `json:"last_update"`
	CacheAge     time.Duration `json:"cache_age"`
	PollInterval time.Duration `json:"poll_interval"`
}

// Core monitoring loop with fast count-based detection
func (s *Session) monitorDevices() {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkForChangesAsync()
		}
	}
}

// Fast change detection with async scanning
func (s *Session) checkForChangesAsync() {
	// Ultra-fast count check (~50µs)
	newAudioCount, newMIDICount, err := devices.GetDeviceCounts()
	if err != nil {
		return // Skip this cycle on error
	}

	// Atomic compare (lock-free)
	oldAudioCount := atomic.LoadInt64(&s.audioCount)
	oldMIDICount := atomic.LoadInt64(&s.midiCount)

	audioChanged := int64(newAudioCount) != oldAudioCount
	midiChanged := int64(newMIDICount) != oldMIDICount

	if !audioChanged && !midiChanged {
		return // No changes detected
	}

	// Update counts immediately
	atomic.StoreInt64(&s.audioCount, int64(newAudioCount))
	atomic.StoreInt64(&s.midiCount, int64(newMIDICount))

	// Determine change type
	var changeType ChangeType
	if audioChanged && midiChanged {
		changeType = BothDeviceChange
	} else if audioChanged {
		changeType = AudioDeviceChange
	} else {
		changeType = MIDIDeviceChange
	}

	// Create immediate notification with counts and scanning flags
	change := DeviceChange{
		Type:          changeType,
		Timestamp:     time.Now(),
		AudioCount:    newAudioCount,
		MIDICount:     newMIDICount,
		AudioScanning: audioChanged,
		MIDIScanning:  midiChanged,
	}

	// Immediate notification (counts available immediately)
	s.notifyChange(change)

	// Start independent async scans
	if audioChanged {
		go s.scanAudioDevicesAsync(change)
	}
	if midiChanged {
		go s.scanMIDIDevicesAsync(change)
	}
}

// Async audio device enumeration
func (s *Session) scanAudioDevicesAsync(initialChange DeviceChange) {
	audioDevices, err := devices.GetAudio()
	if err != nil {
		return // Skip notification on error
	}

	// Update cache
	s.deviceMutex.Lock()
	s.audioDevices = audioDevices
	s.lastUpdate = time.Now()
	s.deviceMutex.Unlock()

	// Create completion notification
	change := DeviceChange{
		Type:          initialChange.Type,
		Timestamp:     time.Now(),
		AudioCount:    initialChange.AudioCount,
		MIDICount:     initialChange.MIDICount,
		AudioDevices:  &audioDevices,
		AudioScanning: false, // Scan complete
		MIDIScanning:  initialChange.MIDIScanning,
	}

	s.notifyChange(change)
}

// Async MIDI device enumeration
func (s *Session) scanMIDIDevicesAsync(initialChange DeviceChange) {
	midiDevices, err := devices.GetMIDI()
	if err != nil {
		return // Skip notification on error
	}

	// Update cache
	s.deviceMutex.Lock()
	s.midiDevices = midiDevices
	s.lastUpdate = time.Now()
	s.deviceMutex.Unlock()

	// Create completion notification
	change := DeviceChange{
		Type:          initialChange.Type,
		Timestamp:     time.Now(),
		AudioCount:    initialChange.AudioCount,
		MIDICount:     initialChange.MIDICount,
		MIDIDevices:   &midiDevices,
		AudioScanning: initialChange.AudioScanning,
		MIDIScanning:  false, // Scan complete
	}

	s.notifyChange(change)
}

// Thread-safe change notification
func (s *Session) notifyChange(change DeviceChange) {
	// Non-blocking channel notification
	select {
	case s.deviceChanges <- change:
	case <-time.After(1 * time.Millisecond):
		// Channel full, skip this notification
	}

	// Non-blocking callback notifications
	s.callbackMutex.RLock()
	callbacks := make([]ChangeCallback, len(s.callbacks))
	copy(callbacks, s.callbacks)
	s.callbackMutex.RUnlock()

	for _, callback := range callbacks {
		go callback(change)
	}
}

// Synchronous device refresh for initialization
func (s *Session) refreshDevicesSync() error {
	// Get initial device lists
	audioDevices, err := devices.GetAudio()
	if err != nil {
		return err
	}

	midiDevices, err := devices.GetMIDI()
	if err != nil {
		return err
	}

	// Update cache and counts
	s.deviceMutex.Lock()
	s.audioDevices = audioDevices
	s.midiDevices = midiDevices
	s.lastUpdate = time.Now()
	s.deviceMutex.Unlock()

	atomic.StoreInt64(&s.audioCount, int64(len(audioDevices)))
	atomic.StoreInt64(&s.midiCount, int64(len(midiDevices)))

	return nil
}

// ForceRefresh triggers immediate synchronous device refresh
func (s *Session) ForceRefresh() error {
	return s.refreshDevicesSync()
}

// SimulateDeviceChange sends a fake change notification for testing
func (s *Session) SimulateDeviceChange(changeType ChangeType) {
	audioCount, midiCount := s.GetDeviceCounts()
	change := DeviceChange{
		Type:          changeType,
		Timestamp:     time.Now(),
		AudioCount:    audioCount,
		MIDICount:     midiCount,
		AudioScanning: changeType == AudioDeviceChange || changeType == BothDeviceChange,
		MIDIScanning:  changeType == MIDIDeviceChange || changeType == BothDeviceChange,
	}

	s.notifyChange(change)
}

// Close stops monitoring and cleans up resources
func (s *Session) Close() error {
	atomic.StoreInt64(&s.monitoring, 0)
	s.cancel()
	close(s.deviceChanges)
	return nil
}
