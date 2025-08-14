//go:build darwin

package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shaban/macaudio/devices"
	"github.com/shaban/macaudio/plugins"
)

// Library configuration - easily changeable for future expansion
const (
	LibraryName = "macaudio" // Change to "audio" when going cross-platform
)

// Default plugin introspection timeouts (seconds)
const (
	defaultPresetLoadingTimeout   = 0.15
	defaultProcessUpdateTimeout   = 0.05
	defaultTotalIntrospectTimeout = 2.0
)

// Cache TTL
const pluginCacheTTL = 24 * time.Hour

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

	// Plugin management
	cachedPlugins   []*plugins.Plugin  // Full plugin data
	cachedQuickInfo map[string]string  // Quick lookup for change detection
	pluginCallbacks []PluginCallback   // Plugin callbacks
	pluginRequests  chan PluginRequest // Async plugin requests
	pluginMutex     sync.RWMutex       // Plugin cache protection

	// Configuration
	audioSpec    AudioSpec
	pollInterval time.Duration

	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	monitoring int64 // atomic bool

	// Cache index in-memory snapshot (for fast QuickPlugins)
	idxMu   sync.RWMutex
	idxSnap *indexFile

	// single-flight dedupe for Plugin() calls
	inflightMu sync.Mutex
	inflight   map[string]*inflightCall

	// optional metrics hook
	hook MetricsHook
}

// LatencyClass is a coarse latency preference that maps to buffer sizes.
type LatencyClass string

const (
	LatencyLow    LatencyClass = "low"    // prioritize minimal latency (smaller buffers)
	LatencyMedium LatencyClass = "medium" // balanced default
	LatencyHigh   LatencyClass = "high"   // prioritize stability (larger buffers)
)

// AudioSpec captures session-level audio preferences.
// Note:
//  - PreferredSampleRate is a target; actual device/sample rate may differ.
//  - BufferSize is a hint and may be adjusted at runtime.
//  - ChannelCount and BitDepth reflect legacy/global settings; engines typically run 32-bit float stereo internally.
//    Deprecated: these are not enforced globally and may be removed in a future release.
type AudioSpec struct {
	// Preferred target sample rate for the session; devices may override.
	PreferredSampleRate float64      `json:"preferred_sample_rate,omitempty"`
	// Coarse latency preference; maps to buffer sizes per backend.
	LatencyHint         LatencyClass `json:"latency_hint,omitempty"`

	// Deprecated: Global channel count is not a session invariant. Use per-node formats.
	ChannelCount int `json:"channel_count,omitempty"`
	// Deprecated: Engines use 32-bit float internally; keep for I/O contexts only.
	BitDepth     int `json:"bit_depth,omitempty"`

	// Optional explicit buffer size hint (frames). Overrides LatencyHint if set > 0.
	BufferSize   int `json:"buffer_size,omitempty"`
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

// Plugin-related types
type PluginCache struct {
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Plugins   []*plugins.Plugin `json:"plugins"`    // Full introspected plugins
	QuickInfo map[string]string `json:"quick_info"` // For change detection
}

type PluginCallback func(PluginResult)

type PluginRequest struct {
	ID        string         `json:"id"`
	Callback  PluginCallback `json:"-"`
	Timestamp time.Time      `json:"timestamp"`
}

type PluginResult struct {
	RequestID    string            `json:"request_id"`
	Success      bool              `json:"success"`
	Error        string            `json:"error,omitempty"`
	Plugins      []*plugins.Plugin `json:"plugins,omitempty"`
	CacheHit     bool              `json:"cache_hit"`
	ScanTime     time.Duration     `json:"scan_time"`
	ChangedCount int               `json:"changed_count"`
	Timestamp    time.Time         `json:"timestamp"`
}

// Default audio configuration
var DefaultAudioSpec = AudioSpec{
	PreferredSampleRate: 48000,
	LatencyHint:         LatencyMedium,
	// Legacy fields retained for compatibility; not strictly enforced.
	ChannelCount: 2,
	BitDepth:     32,
	BufferSize:   512,
}

// NewSession creates a new audio session with fast async monitoring
func NewSession(spec AudioSpec) (*Session, error) {
	ctx, cancel := context.WithCancel(context.Background())

	session := &Session{
		deviceChanges:  make(chan DeviceChange, 10),
		pluginRequests: make(chan PluginRequest, 10), // Plugin request queue
		audioSpec:      spec,
		pollInterval:   50 * time.Millisecond, // Fast count-based polling
		ctx:            ctx,
		cancel:         cancel,
		inflight:       make(map[string]*inflightCall),
	}

	// Initial device enumeration and count setup
	if err := session.refreshDevicesSync(); err != nil {
		cancel()
		return nil, err
	}

	// Load index snapshot (do not block on details)
	if idx, err := loadIndex(); err == nil {
		session.idxMu.Lock()
		session.idxSnap = idx
		session.idxMu.Unlock()
	}

	// Start async monitoring and plugin processing
	go session.monitorDevices()
	go session.processPluginRequests()
	atomic.StoreInt64(&session.monitoring, 1)

	// Configure reasonable plugin introspection timeouts globally
	// These can be adjusted later via dedicated setters if needed
	plugins.SetPresetLoadingTimeout(defaultPresetLoadingTimeout)
	plugins.SetProcessUpdateTimeout(defaultProcessUpdateTimeout)
	plugins.SetTotalTimeout(defaultTotalIntrospectTimeout)

	return session, nil
}

// SetMetricsHook sets an optional metrics hook. Passing nil disables metrics callbacks.
func (s *Session) SetMetricsHook(h MetricsHook) { s.hook = h }

// Options configure advanced behaviors at session construction time.
// Use this to tune plugin introspection timeouts or warm specific plugins on startup.
type Options struct {
	// Override default plugin introspection timeouts (seconds); <=0 keeps defaults
	PresetLoadingTimeout   float64
	ProcessUpdateTimeout   float64
	TotalIntrospectTimeout float64
	// When true, run a quick scan to populate index on start
	RefreshQuickOnStart bool
	// Warm predicate and concurrency; applied after quick refresh if set
	WarmSelector   func(plugins.PluginInfo) bool
	WarmConcurrency int
}

// NewSessionWithOptions creates a session with advanced options.
func NewSessionWithOptions(spec AudioSpec, opt Options) (*Session, error) {
	s, err := NewSession(spec)
	if err != nil { return nil, err }
	// Apply timeouts if provided
	if opt.PresetLoadingTimeout > 0 { plugins.SetPresetLoadingTimeout(opt.PresetLoadingTimeout) }
	if opt.ProcessUpdateTimeout > 0 { plugins.SetProcessUpdateTimeout(opt.ProcessUpdateTimeout) }
	if opt.TotalIntrospectTimeout > 0 { plugins.SetTotalTimeout(opt.TotalIntrospectTimeout) }
	// Optionally refresh quick index and warm details asynchronously
	if opt.RefreshQuickOnStart {
		go func() {
			if _, err := s.RefreshQuick(); err != nil {
				// best-effort; ignore
			}
			if opt.WarmSelector != nil {
				_ = s.Warm(opt.WarmSelector, opt.WarmConcurrency)
			}
		}()
	}
	return s, nil
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

// GetPluginsAsync - consumer requests plugins and gets callback when ready
func (s *Session) GetPluginsAsync(callback PluginCallback) string {
	requestID := fmt.Sprintf("plugin_%d", time.Now().UnixNano())

	request := PluginRequest{
		ID:        requestID,
		Callback:  callback,
		Timestamp: time.Now(),
	}

	// Non-blocking request queue
	select {
	case s.pluginRequests <- request:
		// Request queued successfully
	default:
		// Queue full - immediate error callback
		go callback(PluginResult{
			RequestID: requestID,
			Success:   false,
			Error:     "plugin request queue full",
			Timestamp: time.Now(),
		})
	}

	return requestID
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

// getPluginCacheDir returns the Mac-native cache directory
func getPluginCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Allow override for tests and custom setups
	if override := os.Getenv("MACAUDIO_CACHE_DIR"); override != "" {
		if err := os.MkdirAll(override, 0755); err != nil {
			return "", fmt.Errorf("failed to create override cache directory: %w", err)
		}
		return override, nil
	}

	// Mac-specific default: ~/Library/Application Support/macaudio/
	cacheDir := filepath.Join(home, "Library", "Application Support", LibraryName)
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return cacheDir, nil
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

// Plugin processing methods

// processPluginRequests handles async plugin requests
func (s *Session) processPluginRequests() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case request := <-s.pluginRequests:
			go s.handlePluginRequest(request)
		}
	}
}

// handlePluginRequest processes a single plugin request
func (s *Session) handlePluginRequest(request PluginRequest) {
	startTime := time.Now()

	// Step 1: Load cache from disk if exists
	cachedPlugins, cachedQuickInfo, err := s.loadFullPluginCache()
	if err != nil {
		// Cache load error - treat as no cache
		cachedPlugins = nil
		cachedQuickInfo = nil
	}

	// Step 2: Quick scan current state
	currentInfos, err := plugins.List()
	if err != nil {
		request.Callback(PluginResult{
			RequestID: request.ID,
			Success:   false,
			Error:     fmt.Sprintf("plugin quick scan failed: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// Step 3: Decide what to do
	if cachedPlugins == nil {
		// No cache - do full scan
		s.doFullPluginScan(request, currentInfos, startTime)
	} else {
		// Have cache - reconcile with current state
		changes := s.findPluginChanges(cachedQuickInfo, currentInfos)
		if len(changes) == 0 {
			// No changes - return cached data
			request.Callback(PluginResult{
				RequestID: request.ID,
				Success:   true,
				Plugins:   cachedPlugins,
				CacheHit:  true,
				ScanTime:  time.Since(startTime),
				Timestamp: time.Now(),
			})
		} else {
			// Changes found - update cache
			s.updatePluginCache(request, cachedPlugins, cachedQuickInfo, currentInfos, changes, startTime)
		}
	}
}

// doFullPluginScan performs complete plugin introspection
func (s *Session) doFullPluginScan(request PluginRequest, infos plugins.PluginInfos, startTime time.Time) {
	// Introspect all plugins
	allPlugins, err := infos.Introspect()
	if err != nil {
		request.Callback(PluginResult{
			RequestID: request.ID,
			Success:   false,
			Error:     fmt.Sprintf("plugin introspection failed: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// Build cache
	quickInfo := s.buildQuickLookup(infos)
	cache := &PluginCache{
		Version:   "1.0",
		Timestamp: time.Now(),
		Plugins:   allPlugins,
		QuickInfo: quickInfo,
	}

	// Update session cache
	s.pluginMutex.Lock()
	s.cachedPlugins = allPlugins
	s.cachedQuickInfo = quickInfo
	s.pluginMutex.Unlock()

	// Save cache to disk asynchronously
	go s.savePluginCache(cache)

	// Success callback
	request.Callback(PluginResult{
		RequestID:    request.ID,
		Success:      true,
		Plugins:      allPlugins,
		CacheHit:     false,
		ScanTime:     time.Since(startTime),
		ChangedCount: len(allPlugins),
		Timestamp:    time.Now(),
	})
}

// Helper methods for plugin management

// buildQuickLookup creates change detection map from plugin infos
func (s *Session) buildQuickLookup(infos plugins.PluginInfos) map[string]string {
	lookup := make(map[string]string)
	for _, info := range infos {
		// Use the full quadruplet as the unique key: type:subtype:manufacturerID:name
		key := fmt.Sprintf("%s:%s:%s:%s", info.Type, info.Subtype, info.ManufacturerID, info.Name)
		// Strong checksum across quick info fields for reliable change detection
		sumInput := fmt.Sprintf("%s|%s|%s|%s|%s", info.Type, info.Subtype, info.ManufacturerID, info.Name, info.Category)
		h := sha256.Sum256([]byte(sumInput))
		checksum := hex.EncodeToString(h[:])
		lookup[key] = checksum
	}
	return lookup
}

// findPluginChanges compares cache with current quick scan
func (s *Session) findPluginChanges(cachedQuickInfo map[string]string, current plugins.PluginInfos) []string {
	var changedKeys []string

	// Build current lookup
	currentLookup := s.buildQuickLookup(current)

	// Find additions and modifications
	for key, checksum := range currentLookup {
		if cachedChecksum, exists := cachedQuickInfo[key]; !exists || cachedChecksum != checksum {
			changedKeys = append(changedKeys, key)
		}
	}

	return changedKeys
}

// updatePluginCache handles partial cache updates
func (s *Session) updatePluginCache(request PluginRequest, cachedPlugins []*plugins.Plugin, cachedQuickInfo map[string]string, current plugins.PluginInfos, changes []string, startTime time.Time) {
	// Create lookup maps
	currentLookup := make(map[string]plugins.PluginInfo)
	for _, info := range current {
		key := fmt.Sprintf("%s:%s:%s:%s", info.Type, info.Subtype, info.ManufacturerID, info.Name)
		currentLookup[key] = info
	}

	cachePluginMap := make(map[string]*plugins.Plugin)
	for _, plugin := range cachedPlugins {
		key := fmt.Sprintf("%s:%s:%s:%s", plugin.Type, plugin.Subtype, plugin.ManufacturerID, plugin.Name)
		cachePluginMap[key] = plugin
	}

	// Build updated plugin list
	var updatedPlugins []*plugins.Plugin
	changedCount := 0

	for _, info := range current {
		key := fmt.Sprintf("%s:%s:%s:%s", info.Type, info.Subtype, info.ManufacturerID, info.Name)

		if s.contains(changes, key) {
			// This plugin changed - introspect it
			plugin, err := info.Introspect()
			if err != nil {
				// Skip failed introspections but continue
				continue
			}
			updatedPlugins = append(updatedPlugins, plugin)
			changedCount++
		} else {
			// Plugin unchanged - use cached version
			if cachedPlugin, exists := cachePluginMap[key]; exists {
				updatedPlugins = append(updatedPlugins, cachedPlugin)
			}
		}
	}

	// Build updated cache
	updatedQuickInfo := s.buildQuickLookup(current)
	cache := &PluginCache{
		Version:   "1.0",
		Timestamp: time.Now(),
		Plugins:   updatedPlugins,
		QuickInfo: updatedQuickInfo,
	}

	// Update session cache
	s.pluginMutex.Lock()
	s.cachedPlugins = updatedPlugins
	s.cachedQuickInfo = updatedQuickInfo
	s.pluginMutex.Unlock()

	// Save cache to disk asynchronously
	go s.savePluginCache(cache)

	// Success callback
	request.Callback(PluginResult{
		RequestID:    request.ID,
		Success:      true,
		Plugins:      updatedPlugins,
		CacheHit:     false,
		ScanTime:     time.Since(startTime),
		ChangedCount: changedCount,
		Timestamp:    time.Now(),
	})
}

// contains checks if a slice contains a string
func (s *Session) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// loadFullPluginCache loads the complete cache from disk
func (s *Session) loadFullPluginCache() ([]*plugins.Plugin, map[string]string, error) {
	cacheDir, err := getPluginCacheDir()
	if err != nil {
		return nil, nil, err
	}

	cachePath := filepath.Join(cacheDir, "plugin_cache.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, nil, err // No cache file
	}

	var cache PluginCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, nil, err // Corrupted cache
	}

	// Validate cache version
	if cache.Version != "1.0" {
		return nil, nil, fmt.Errorf("unsupported plugin cache version: %s", cache.Version)
	}

	// Enforce TTL
	if time.Since(cache.Timestamp) > pluginCacheTTL {
		return nil, nil, fmt.Errorf("plugin cache expired")
	}

	return cache.Plugins, cache.QuickInfo, nil
}

// savePluginCache saves the cache to disk
func (s *Session) savePluginCache(cache *PluginCache) {
	cacheDir, err := getPluginCacheDir()
	if err != nil {
		return
	}

	cachePath := filepath.Join(cacheDir, "plugin_cache.json")
	data, err := json.Marshal(cache)
	if err != nil {
		return
	}

	os.WriteFile(cachePath, data, 0644)
}

// Close stops monitoring and cleans up resources
func (s *Session) Close() error {
	atomic.StoreInt64(&s.monitoring, 0)
	s.cancel()
	// Intentionally do not close s.deviceChanges to avoid send-on-closed panics.
	// Consumers should stop reading when context is canceled or the session is closed.
	return nil
}

// QuickPlugins returns the cached quick index; runs a quick scan when empty/outdated and persists it.
func (s *Session) QuickPlugins() (plugins.PluginInfos, error) {
	start := time.Now()
	if s.hook != nil { s.hook.OnQuickScanStart() }
	s.idxMu.RLock()
	idx := s.idxSnap
	s.idxMu.RUnlock()

	if idx != nil && len(idx.Entries) > 0 {
		infos := make(plugins.PluginInfos, 0, len(idx.Entries))
		for _, e := range idx.Entries {
			infos = append(infos, plugins.PluginInfo{
				Name: e.Name, ManufacturerID: e.ManufacturerID, Type: e.Type, Subtype: e.Subtype, Category: e.Category,
			})
		}
		if s.hook != nil { s.hook.OnQuickScanDone(time.Since(start), len(infos), false) }
		return infos, nil
	}

	// Populate via quick scan
	infos, err := plugins.List()
	if err != nil {
		if s.hook != nil { s.hook.OnQuickScanDone(time.Since(start), 0, true) }
		return nil, err
	}
	// Persist index
	newIdx := &indexFile{Version: indexVersion, UpdatedAt: time.Now(), Entries: map[string]indexEntry{}}
	for _, info := range infos {
		key := quadKey(info.Type, info.Subtype, info.ManufacturerID, info.Name)
		newIdx.Entries[key] = indexEntry{
			Key: key, Type: info.Type, Subtype: info.Subtype, ManufacturerID: info.ManufacturerID,
			Name: info.Name, Category: info.Category, Checksum: checksumQuick(info), LastSeenAt: time.Now(),
		}
	}
	_ = saveIndex(newIdx) // best-effort
	s.idxMu.Lock()
	s.idxSnap = newIdx
	s.idxMu.Unlock()
	if s.hook != nil { s.hook.OnQuickScanDone(time.Since(start), len(infos), true) }
	return infos, nil
}

// Plugin returns full details for the given quadruplet, using lazy cached details when available.
func (s *Session) Plugin(t, st, man, name string) (*plugins.Plugin, error) {
	key := quadKey(t, st, man, name)
	// Single-flight dedupe: join in-flight call for the same key
	if p, joined, err := s.joinInFlight(key); joined {
		return p, err
	}
	defer s.finishInFlight(key)
	// Try index to get checksum
	s.idxMu.RLock()
	idx := s.idxSnap
	s.idxMu.RUnlock()
	var wantChecksum string
	if idx != nil {
		if e, ok := idx.Entries[key]; ok {
			wantChecksum = e.Checksum
		}
	}
	if wantChecksum != "" {
		if p, chk, err := readDetails(key); err == nil && chk == wantChecksum {
			if s.hook != nil { s.hook.OnCacheHit(key) }
			s.setInFlightResult(key, p, nil)
			return p, nil
		}
	}
	if s.hook != nil { s.hook.OnCacheMiss(key) }
	// Introspect single
	if s.hook != nil { s.hook.OnDetailsFetchStart(key) }
	t0 := time.Now()
	infos, err := plugins.List()
	if err != nil {
		if s.hook != nil { s.hook.OnDetailsFetchDone(key, time.Since(t0), false) }
		s.setInFlightResult(key, nil, err)
		return nil, err
	}
	var target *plugins.PluginInfo
	for _, i := range infos {
		if i.Type == t && i.Subtype == st && i.ManufacturerID == man && i.Name == name {
			ii := i
			target = &ii
			break
		}
	}
	if target == nil {
	err := fmt.Errorf("plugin not found: %s", key)
	if s.hook != nil { s.hook.OnDetailsFetchDone(key, time.Since(t0), false) }
		s.setInFlightResult(key, nil, err)
		return nil, err
	}
	p, err := target.Introspect()
	if err != nil {
	if s.hook != nil { s.hook.OnDetailsFetchDone(key, time.Since(t0), false) }
		s.setInFlightResult(key, nil, err)
		return nil, err
	}
	// Persist details and refresh index entry
	chk := checksumQuick(*target)
	_ = writeDetails(key, chk, p)
	s.idxMu.Lock()
	if s.idxSnap == nil {
		s.idxSnap = &indexFile{Version: indexVersion, Entries: map[string]indexEntry{}}
	}
	s.idxSnap.Entries[key] = indexEntry{Key: key, Type: t, Subtype: st, ManufacturerID: man, Name: name, Category: target.Category, Checksum: chk, LastSeenAt: time.Now()}
	_ = saveIndex(s.idxSnap)
	s.idxMu.Unlock()
	if s.hook != nil { s.hook.OnDetailsFetchDone(key, time.Since(t0), true) }
	s.setInFlightResult(key, p, nil)
	return p, nil
}

// inflightCall tracks waiting goroutines for Plugin() of a key
type inflightCall struct {
	done chan struct{}
	p    *plugins.Plugin
	err  error
}

// joinInFlight registers/join an in-flight call. If already running, waits and returns its result.
func (s *Session) joinInFlight(key string) (*plugins.Plugin, bool, error) {
	s.inflightMu.Lock()
	if s.inflight == nil { s.inflight = make(map[string]*inflightCall) }
	if c, ok := s.inflight[key]; ok {
		// another call in-flight; wait
		done := c.done
		s.inflightMu.Unlock()
		<-done
		return c.p, true, c.err
	}
	// create a new in-flight entry for this caller to publish later
	c := &inflightCall{done: make(chan struct{})}
	s.inflight[key] = c
	s.inflightMu.Unlock()
	return nil, false, nil
}

// finishInFlight publishes the result to waiters and clears the entry.
func (s *Session) finishInFlight(key string) {
	s.inflightMu.Lock()
	c, ok := s.inflight[key]
	if ok {
		close(c.done)
		delete(s.inflight, key)
	}
	s.inflightMu.Unlock()
}

// setInFlightResult sets the result for a key so waiters receive it upon finish.
func (s *Session) setInFlightResult(key string, p *plugins.Plugin, err error) {
	s.inflightMu.Lock()
	if c, ok := s.inflight[key]; ok {
		c.p, c.err = p, err
	}
	s.inflightMu.Unlock()
}

// RefreshQuick re-runs quick scan, reconciles the index, and returns a simple diff summary.
type QuickDiff struct{ Added, Removed, Changed []string }

func (s *Session) RefreshQuick() (QuickDiff, error) {
	if s.hook != nil { s.hook.OnQuickScanStart() }
	t0 := time.Now()
	infos, err := plugins.List()
	if err != nil {
		if s.hook != nil { s.hook.OnQuickScanDone(time.Since(t0), 0, true) }
		return QuickDiff{}, err
	}
	// Build new index map
	newIdx := &indexFile{Version: indexVersion, UpdatedAt: time.Now(), Entries: map[string]indexEntry{}}
	for _, info := range infos {
		key := quadKey(info.Type, info.Subtype, info.ManufacturerID, info.Name)
		newIdx.Entries[key] = indexEntry{
			Key: key, Type: info.Type, Subtype: info.Subtype, ManufacturerID: info.ManufacturerID,
			Name: info.Name, Category: info.Category, Checksum: checksumQuick(info), LastSeenAt: time.Now(),
		}
	}
	// Diff
	s.idxMu.RLock()
	old := s.idxSnap
	s.idxMu.RUnlock()
	diff := QuickDiff{}
	if old != nil {
		for k, ov := range old.Entries {
			nv, ok := newIdx.Entries[k]
			if !ok {
				diff.Removed = append(diff.Removed, k)
			}
			if ok && ov.Checksum != nv.Checksum {
				diff.Changed = append(diff.Changed, k)
			}
		}
	}
	for k := range newIdx.Entries {
		if old == nil {
			diff.Added = append(diff.Added, k)
			continue
		}
		if _, ok := old.Entries[k]; !ok {
			diff.Added = append(diff.Added, k)
		}
	}
	// Save and swap snapshot
	_ = saveIndex(newIdx)
	s.idxMu.Lock()
	s.idxSnap = newIdx
	s.idxMu.Unlock()
	// Cleanup stale details for removed or changed keys (best-effort)
	for _, k := range append(diff.Removed, diff.Changed...) {
		_ = deleteDetails(k)
	}
	if s.hook != nil {
		s.hook.OnQuickScanDone(time.Since(t0), len(infos), true)
		s.hook.OnRefreshQuickDiff(len(diff.Added), len(diff.Removed), len(diff.Changed), time.Since(t0))
	}
	return diff, nil
}

// Warm introspects details for a subset defined by a selector and saves them to cache.
func (s *Session) Warm(selector func(plugins.PluginInfo) bool, concurrency int) error {
	if concurrency <= 0 {
		concurrency = 2
	}
	infos, err := s.QuickPlugins()
	if err != nil {
		return err
	}
	total := 0
	for _, info := range infos { if selector == nil || selector(info) { total++ } }
	completed := 0
	if s.hook != nil { s.hook.OnWarmProgress(total, completed) }
	sem := make(chan struct{}, concurrency)
	errCh := make(chan error, concurrency)
	for _, info := range infos {
		if selector != nil && !selector(info) {
			continue
		}
		info := info
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			if _, err := s.Plugin(info.Type, info.Subtype, info.ManufacturerID, info.Name); err != nil {
				errCh <- err
			}
			completed++
			if s.hook != nil { s.hook.OnWarmProgress(total, completed) }
		}()
	}
	for i := 0; i < cap(sem); i++ {
		sem <- struct{}{}
	}
	close(errCh)
	// Best-effort: return first error if any
	for e := range errCh {
		if e != nil {
			return e
		}
	}
	return nil
}
