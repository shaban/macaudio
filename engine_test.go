package macaudio

import (
	"testing"
	"time"
)

func TestEngineCreation(t *testing.T) {
	config := EngineConfig{
		BufferSize:   256,
		SampleRate:   48000.0,
		ErrorHandler: &DefaultErrorHandler{},
	}
	
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	if engine == nil {
		t.Fatal("Engine is nil")
	}
	
	// Check initial state
	if engine.IsRunning() {
		t.Error("Engine should not be running initially")
	}
	
	// Check master channel exists
	masterChannel := engine.GetMasterChannel()
	if masterChannel == nil {
		t.Fatal("Master channel is nil")
	}
	
	if masterChannel.GetID() != "master" {
		t.Errorf("Master channel ID should be 'master', got '%s'", masterChannel.GetID())
	}
	
	// Check initial channels list includes master
	channels := engine.ListChannels()
	if len(channels) != 1 {
		t.Errorf("Expected 1 channel initially, got %d", len(channels))
	}
	
	if channels[0] != "master" {
		t.Errorf("Expected master channel in list, got %v", channels)
	}
}

func TestEngineStartStop(t *testing.T) {
	config := EngineConfig{
		BufferSize:   256,
		SampleRate:   48000.0,
		ErrorHandler: &DefaultErrorHandler{},
	}
	
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	// Start engine
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	
	if !engine.IsRunning() {
		t.Error("Engine should be running after Start()")
	}
	
	// Check components are running
	if !engine.GetDeviceMonitor().IsRunning() {
		t.Error("Device monitor should be running")
	}
	
	if !engine.GetDispatcher().IsRunning() {
		t.Error("Dispatcher should be running")
	}
	
	// Stop engine
	if err := engine.Stop(); err != nil {
		t.Errorf("Failed to stop engine: %v", err)
	}
	
	if engine.IsRunning() {
		t.Error("Engine should not be running after Stop()")
	}
}

func TestChannelCreation(t *testing.T) {
	config := EngineConfig{
		BufferSize:   256,
		SampleRate:   48000.0,
		ErrorHandler: &DefaultErrorHandler{},
	}
	
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()
	
	// Create playback channel
	playbackConfig := PlaybackConfig{
		FilePath:    "/nonexistent/file.wav", // File doesn't need to exist for this test
		LoopEnabled: false,
		AutoStart:   false,
	}
	
	playbackChannel, err := engine.CreatePlaybackChannel("test_playback", playbackConfig)
	if err != nil {
		t.Fatalf("Failed to create playback channel: %v", err)
	}
	
	if playbackChannel.GetID() != "test_playback" {
		t.Errorf("Playback channel ID should be 'test_playback', got '%s'", playbackChannel.GetID())
	}
	
	if playbackChannel.GetType() != ChannelTypePlayback {
		t.Errorf("Channel type should be playback, got %s", playbackChannel.GetType())
	}
	
	// Check channel is in engine
	channels := engine.ListChannels()
	found := false
	for _, id := range channels {
		if id == "test_playback" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Playback channel not found in engine channels list")
	}
	
	// Create aux channel
	auxConfig := AuxConfig{
		SendLevel:   0.5,
		ReturnLevel: 0.7,
		PreFader:    false,
	}
	
	auxChannel, err := engine.CreateAuxChannel("test_aux", auxConfig)
	if err != nil {
		t.Fatalf("Failed to create aux channel: %v", err)
	}
	
	if auxChannel.GetID() != "test_aux" {
		t.Errorf("Aux channel ID should be 'test_aux', got '%s'", auxChannel.GetID())
	}
}

func TestPluginChain(t *testing.T) {
	chain := NewPluginChain()
	
	if chain == nil {
		t.Fatal("Plugin chain is nil")
	}
	
	instances := chain.GetInstances()
	if len(instances) != 0 {
		t.Errorf("New plugin chain should be empty, got %d instances", len(instances))
	}
	
	// Add a plugin blueprint
	blueprint := PluginBlueprint{
		Type:           "aufx",
		Subtype:        "test",
		ManufacturerID: "test",
		Name:           "Test Plugin",
		IsInstalled:    false,
	}
	
	instance, err := chain.AddPlugin(blueprint, 0)
	if err != nil {
		t.Fatalf("Failed to add plugin: %v", err)
	}
	
	if instance.Blueprint.Name != "Test Plugin" {
		t.Errorf("Plugin name should be 'Test Plugin', got '%s'", instance.Blueprint.Name)
	}
	
	if instance.Position != 0 {
		t.Errorf("Plugin position should be 0, got %d", instance.Position)
	}
	
	// Check plugin is in chain
	instances = chain.GetInstances()
	if len(instances) != 1 {
		t.Errorf("Plugin chain should have 1 instance, got %d", len(instances))
	}
	
	// Remove plugin
	if err := chain.RemovePlugin(instance.ID); err != nil {
		t.Fatalf("Failed to remove plugin: %v", err)
	}
	
	instances = chain.GetInstances()
	if len(instances) != 0 {
		t.Errorf("Plugin chain should be empty after removal, got %d instances", len(instances))
	}
}

func TestSerialization(t *testing.T) {
	config := EngineConfig{
		BufferSize:   256,
		SampleRate:   48000.0,
		ErrorHandler: &DefaultErrorHandler{},
	}
	
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()
	
	// Create a channel
	playbackConfig := PlaybackConfig{
		FilePath:    "/test/file.wav",
		LoopEnabled: true,
		AutoStart:   true,
	}
	
	_, err = engine.CreatePlaybackChannel("test_serialize", playbackConfig)
	if err != nil {
		t.Fatalf("Failed to create playback channel: %v", err)
	}
	
	// Serialize engine state
	serializer := engine.GetSerializer()
	jsonState, err := serializer.SaveToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize state: %v", err)
	}
	
	if len(jsonState) == 0 {
		t.Error("Serialized state is empty")
	}
	
	// The serialized state should contain our channel
	if !containsString(jsonState, "test_serialize") {
		t.Error("Serialized state doesn't contain test channel")
	}
	
	if !containsString(jsonState, "master") {
		t.Error("Serialized state doesn't contain master channel")
	}
}

func TestDeviceMonitor(t *testing.T) {
	config := EngineConfig{
		BufferSize:   256,
		SampleRate:   48000.0,
		ErrorHandler: &DefaultErrorHandler{},
	}
	
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	monitor := engine.GetDeviceMonitor()
	if monitor == nil {
		t.Fatal("Device monitor is nil")
	}
	
	// Check initial state
	if monitor.IsRunning() {
		t.Error("Device monitor should not be running initially")
	}
	
	// Check default polling interval
	interval := monitor.GetPollingInterval()
	expectedInterval := 50 * time.Millisecond
	if interval != expectedInterval {
		t.Errorf("Expected polling interval %v, got %v", expectedInterval, interval)
	}
	
	// Test interval validation
	err = monitor.SetPollingInterval(5 * time.Millisecond)
	if err == nil {
		t.Error("Should reject polling interval less than 10ms")
	}
	
	err = monitor.SetPollingInterval(100 * time.Millisecond)
	if err != nil {
		t.Errorf("Should accept valid polling interval: %v", err)
	}
	
	// Start engine (which starts monitor)
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()
	
	if !monitor.IsRunning() {
		t.Error("Device monitor should be running after engine start")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
