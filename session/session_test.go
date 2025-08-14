//go:build darwin

package session

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/shaban/macaudio/plugins"
)

func TestMain(m *testing.M) {
	// Default to non-interactive to support CI and focused unit tests
	if os.Getenv("SESSION_INTERACTIVE") != "1" {
		os.Exit(m.Run())
		return
	}

	// Interactive demonstration mode (run only when SESSION_INTERACTIVE=1)
	fmt.Println("🚀 Session Package Test Suite")
	fmt.Println("=============================")
	fmt.Println()

	fmt.Println("📋 Test 1: Session Creation")
	sess, err := NewSessionWithDefaults()
	if err != nil {
		log.Fatalf("❌ Failed to create session: %v", err)
	}
	fmt.Printf("✅ Session created successfully\n")
	fmt.Printf("   - Monitoring: %v\n", sess.IsMonitoring())
	fmt.Printf("   - Audio spec: %+v\n", sess.GetAudioSpec())
	fmt.Println()

	fmt.Println("📋 Test 2: Initial Device Enumeration")
	if audioDevices, err := sess.GetAudioDevices(); err == nil {
		fmt.Printf("✅ Audio devices: %d found\n", len(audioDevices))
	}
	if midiDevices, err := sess.GetMIDIDevices(); err == nil {
		fmt.Printf("✅ MIDI devices: %d found\n", len(midiDevices))
	}
	fmt.Println()

	fmt.Println("📋 Test 3: Fast Device Counts")
	audioCount, midiCount := sess.GetDeviceCounts()
	fmt.Printf("✅ Fast counts: %d audio, %d MIDI\n", audioCount, midiCount)
	fmt.Println()

	// Minimal callback check
	callbackCalled := false
	sess.OnDeviceChange(func(change DeviceChange) { callbackCalled = true })
	sess.SimulateDeviceChange(BothDeviceChange)
	time.Sleep(10 * time.Millisecond)
	_ = callbackCalled

	// Async plugin load (best-effort)
	done := make(chan struct{}, 1)
	sess.GetPluginsAsync(func(result PluginResult) { done <- struct{}{} })
	select {
	case <-done:
	case <-time.After(10 * time.Second):
	}

	// Interactive monitoring until Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	_ = sess.Close()
	os.Exit(0)
}

func TestSessionCreation(t *testing.T) {
	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sess.Close()

	if !sess.IsMonitoring() {
		t.Error("Session should be monitoring after creation")
	}

	audioCount, midiCount := sess.GetDeviceCounts()
	if audioCount < 0 || midiCount < 0 {
		t.Error("Device counts should be non-negative")
	}

	t.Logf("Created session with %d audio and %d MIDI devices", audioCount, midiCount)
}

func TestDeviceAccess(t *testing.T) {
	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sess.Close()

	audioDevices, err := sess.GetAudioDevices()
	if err != nil {
		t.Errorf("Failed to get audio devices: %v", err)
	}

	midiDevices, err := sess.GetMIDIDevices()
	if err != nil {
		t.Errorf("Failed to get MIDI devices: %v", err)
	}

	t.Logf("Retrieved %d audio and %d MIDI devices", len(audioDevices), len(midiDevices))
}

func TestCallbacks(t *testing.T) {
	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer sess.Close()

	var callbackCalled atomic.Bool
	sess.OnDeviceChange(func(change DeviceChange) {
		callbackCalled.Store(true)
		t.Logf("Callback received change: %s", change.Type.String())
	})

	// Trigger a simulated change
	sess.SimulateDeviceChange(AudioDeviceChange)

	// Give it time to execute
	time.Sleep(10 * time.Millisecond)

	if !callbackCalled.Load() {
		t.Error("Callback should have been called")
	}
}

// --- Plugin cache tests ---

// TestPluginCacheLifecycle verifies that cache is written on first scan and used on subsequent scans.
func TestPluginCacheLifecycle(t *testing.T) {
	// Use a temp cache dir to isolate from user environment
	tempDir, err := os.MkdirTemp("", "macaudio-cache-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Setenv("MACAUDIO_CACHE_DIR", tempDir)
	defer os.Unsetenv("MACAUDIO_CACHE_DIR")

	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer sess.Close()

	// 1) First request: expect full scan and cache write
	ch1 := make(chan PluginResult, 1)
	startFirst := time.Now()
	sess.GetPluginsAsync(func(r PluginResult) { ch1 <- r })
	var r1 PluginResult
	select {
	case r1 = <-ch1:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for first plugin result")
	}
	if !r1.Success {
		t.Fatalf("first plugin scan failed: %v", r1.Error)
	}
	if r1.CacheHit {
		t.Fatalf("expected cache miss on first scan, got cache hit")
	}
	if len(r1.Plugins) == 0 {
		t.Fatalf("expected plugins on first scan")
	}

	// Ensure cache file exists and is valid JSON (wait for async save)
	cachePath := tempDir + "/plugin_cache.json"
	var data []byte
	deadline := time.Now().Add(10 * time.Second)
	cacheWriteWaitStart := time.Now()
	for {
		data, err = os.ReadFile(cachePath)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("failed to read cache file: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	firstRunElapsed := time.Since(startFirst)
	cacheWriteWait := time.Since(cacheWriteWaitStart)
	t.Logf("First run: result.ScanTime=%v, end-to-end=%v, cache-write-wait=%v", r1.ScanTime, firstRunElapsed, cacheWriteWait)
	var cache struct {
		Version   string            `json:"version"`
		Plugins   []map[string]any  `json:"plugins"`
		QuickInfo map[string]string `json:"quick_info"`
		Timestamp string            `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("invalid cache JSON: %v", err)
	}
	if len(cache.Plugins) == 0 || len(cache.QuickInfo) == 0 {
		t.Fatalf("cache should contain plugins and quick_info")
	}

	// 2) Second request: expect cache hit
	ch2 := make(chan PluginResult, 1)
	startSecond := time.Now()
	sess.GetPluginsAsync(func(r PluginResult) { ch2 <- r })
	var r2 PluginResult
	select {
	case r2 = <-ch2:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for second plugin result")
	}
	if !r2.Success {
		t.Fatalf("second plugin scan failed: %v", r2.Error)
	}
	if !r2.CacheHit {
		t.Fatalf("expected cache hit on second scan")
	}
	if len(r2.Plugins) == 0 {
		t.Fatalf("expected plugins on cache hit")
	}
	cacheHitCallbackLatency := time.Since(startSecond)
	t.Logf("Cache hit: result.ScanTime=%v, callback-latency=%v", r2.ScanTime, cacheHitCallbackLatency)
}

// TestPluginReconciliation ensures changes between quick scan and cache trigger partial updates.
// Note: True system changes are hard to simulate; this test asserts path executes without error.
func TestPluginReconciliation_NoCrash(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "macaudio-cache-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Setenv("MACAUDIO_CACHE_DIR", tempDir)
	defer os.Unsetenv("MACAUDIO_CACHE_DIR")

	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer sess.Close()

	// Prime cache
	ch := make(chan PluginResult, 1)
	sess.GetPluginsAsync(func(r PluginResult) { ch <- r })
	select {
	case r := <-ch:
		if !r.Success {
			t.Fatalf("initial scan failed: %v", r.Error)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout priming plugin cache")
	}

	// Wait for cache file to be written
	cachePath := tempDir + "/plugin_cache.json"
	var raw []byte
	deadline := time.Now().Add(10 * time.Second)
	for {
		raw, err = os.ReadFile(cachePath)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("failed reading cache: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Mutate QuickInfo keys to force a detected change
	var cache map[string]any
	if err := json.Unmarshal(raw, &cache); err != nil {
		t.Fatalf("invalid cache JSON: %v", err)
	}
	if qi, ok := cache["quick_info"].(map[string]any); ok {
		qi["forced:change:key"] = "delta"
		cache["quick_info"] = qi
		buf, _ := json.Marshal(cache)
		_ = os.WriteFile(cachePath, buf, 0644)
	}

	// Request again; should not crash, result should be success
	ch2 := make(chan PluginResult, 1)
	sess.GetPluginsAsync(func(r PluginResult) { ch2 <- r })
	select {
	case r2 := <-ch2:
		if !r2.Success {
			t.Fatalf("reconcile scan failed: %v", r2.Error)
		}
		// We don't assert ChangedCount strictly; environment dependent.
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for reconcile result")
	}
}

// TestRefreshQuickCleansStaleDetails ensures that details files for removed or changed keys are cleaned up.
func TestRefreshQuickCleansStaleDetails(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "macaudio-cache-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Setenv("MACAUDIO_CACHE_DIR", tempDir)
	defer os.Unsetenv("MACAUDIO_CACHE_DIR")

	sess, err := NewSessionWithDefaults()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer sess.Close()

	// Ensure an index exists
	if _, err := sess.RefreshQuick(); err != nil {
		t.Fatalf("refresh quick failed: %v", err)
	}

	// Create a fake key, insert it into the in-memory index snapshot to simulate a previously-seen plugin
	fakeKey := "aufx:FAKE:ACME:Nonexistent Plugin"
	sess.idxMu.Lock()
	if sess.idxSnap == nil {
		sess.idxSnap = &indexFile{Version: indexVersion, Entries: map[string]indexEntry{}}
	}
	sess.idxSnap.Entries[fakeKey] = indexEntry{Key: fakeKey, Type: "aufx", Subtype: "FAKE", ManufacturerID: "ACME", Name: "Nonexistent Plugin", Category: "Effect", Checksum: "deadbeef", LastSeenAt: time.Now()}
	_ = saveIndex(sess.idxSnap)
	sess.idxMu.Unlock()

	// Write a details file for the fake key
	if err := writeDetails(fakeKey, "deadbeef", &plugins.Plugin{Name: "Nonexistent Plugin", ManufacturerID: "ACME", Type: "aufx", Subtype: "FAKE"}); err != nil {
		t.Fatalf("failed to write fake details: %v", err)
	}
	// Sanity: file should exist now
	_, detailsDir, err := getIndexPaths()
	if err != nil {
		t.Fatalf("getIndexPaths: %v", err)
	}
	path := filepath.Join(detailsDir, detailFileName(fakeKey))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected fake details file to exist: %v", err)
	}

	// Run RefreshQuick; since fakeKey won't be in current scan, it should be considered removed and cleaned
	diff, err := sess.RefreshQuick()
	if err != nil {
		t.Fatalf("refresh quick failed: %v", err)
	}
	// Ensure fakeKey is in Removed or Changed for visibility in logs
	t.Logf("RefreshQuick diff: %+v", diff)
	// Allow a brief moment for cleanup (though it's synchronous currently)
	time.Sleep(20 * time.Millisecond)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected fake details file to be deleted, got err=%v", err)
	}
}
