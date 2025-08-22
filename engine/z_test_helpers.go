package engine

import (
	"math/rand"
	"testing"
	"time"

	"github.com/shaban/macaudio/devices"
	"github.com/shaban/macaudio/plugins"
)

// TestEngineConfig holds configuration for creating test engines
type TestEngineConfig struct {
	MasterVolume float32
	SampleRate   int
	BufferSize   int
}

// DefaultTestEngineConfig returns a standard engine configuration for testing
func DefaultTestEngineConfig() TestEngineConfig {
	return TestEngineConfig{
		MasterVolume: 0.8,
		SampleRate:   0, // Use device default
		BufferSize:   512,
	}
}

// CreateTestEngine creates an engine with the given configuration and first available output device
func CreateTestEngine(t *testing.T, config TestEngineConfig) (*Engine, func()) {
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}

	outputDevices := audioDevices.Outputs()
	if len(outputDevices) == 0 {
		t.Skip("No output devices available")
	}

	engine, err := NewEngine(&outputDevices[0], config.SampleRate, config.BufferSize)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	engine.MasterVolume = config.MasterVolume

	cleanup := func() {
		engine.Destroy()
	}

	return engine, cleanup
}

// TestChannelConfig holds configuration for creating test channels
type TestChannelConfig struct {
	Volume      float32
	Pan         float32
	PluginCount int  // Number of plugins to add (0-N)
	UseRealFile bool // Use real system file vs fake path
}

// DefaultInputChannelConfig returns a standard input channel configuration
func DefaultInputChannelConfig() TestChannelConfig {
	return TestChannelConfig{
		Volume:      1.0,
		Pan:         0.0,
		PluginCount: 0,
		UseRealFile: true,
	}
}

// DefaultPlaybackChannelConfig returns a standard playback channel configuration
func DefaultPlaybackChannelConfig() TestChannelConfig {
	return TestChannelConfig{
		Volume:      0.8,
		Pan:         -0.2,
		PluginCount: 0, // No plugins on playback channels per MVP
		UseRealFile: true,
	}
}

// CreateTestInputChannel creates an input channel with the given configuration
func CreateTestInputChannel(t *testing.T, engine *Engine, config TestChannelConfig) *Channel {
	audioDevices, err := devices.GetAudio()
	if err != nil {
		t.Fatalf("Failed to get audio devices: %v", err)
	}

	inputDevices := audioDevices.Inputs()
	if len(inputDevices) == 0 {
		t.Skip("No input devices available")
	}

	device := inputDevices[0]
	channel := &Channel{
		InputOptions: &InputOptions{
			Device:       &device,
			ChannelIndex: 0,
		},
	}

	// For input channels (which may not have mixer nodes), we still want to validate parameters
	// Apply validation manually since SetVolume/SetPan may fail without mixer nodes
	if err := ValidateVolume(config.Volume); err != nil {
		t.Fatalf("Invalid test volume %v: %v", config.Volume, err)
	}
	channel.Volume = config.Volume

	if err := ValidatePan(config.Pan); err != nil {
		t.Fatalf("Invalid test pan %v: %v", config.Pan, err)
	}
	channel.Pan = config.Pan

	// Add plugins if requested
	if config.PluginCount > 0 {
		pluginInfos, err := plugins.List()
		if err != nil {
			t.Logf("Warning: Failed to list plugins: %v", err)
		} else if len(pluginInfos) > 0 {
			pluginChain := &PluginChain{}
			rand.Seed(time.Now().UnixNano())

			// Add up to requested number of plugins (limited by available plugins)
			actualCount := config.PluginCount
			if actualCount > len(pluginInfos) {
				actualCount = len(pluginInfos)
			}

			indices := rand.Perm(len(pluginInfos))[:actualCount]
			for _, idx := range indices {
				pluginInfo := pluginInfos[idx]
				if plugin, err := pluginInfo.Introspect(); err == nil {
					enginePlugin := EnginePlugin{
						IsInstalled: true,
						Plugin:      plugin,
						Bypassed:    false,
					}
					pluginChain.Plugins = append(pluginChain.Plugins, enginePlugin)
				}
			}
			channel.InputOptions.PluginChain = pluginChain
		}
	}

	engine.Channels = append(engine.Channels, channel)
	return channel
}

// CreateTestPlaybackChannel creates a playback channel with the given configuration
func CreateTestPlaybackChannel(t *testing.T, engine *Engine, config TestChannelConfig) *Channel {
	filePath := "/System/Library/Sounds/Ping.aiff"
	if !config.UseRealFile {
		filePath = "/fake/path/test.wav"
	}

	// Create channel via the proper API to test validation
	channel, err := engine.CreatePlaybackChannel(filePath)
	if err != nil {
		// For tests that expect creation to fail, this is acceptable
		if !config.UseRealFile {
			// Create minimal channel structure for testing
			channel = &Channel{
				PlaybackOptions: &PlaybackOptions{
					FilePath: filePath,
					Rate:     1.0,
					Pitch:    0.0,
				},
			}
			engine.Channels = append(engine.Channels, channel)
		} else {
			t.Fatalf("Failed to create playback channel: %v", err)
		}
	}

	// Apply volume and pan through validation methods (not direct assignment)
	if err := channel.SetVolume(config.Volume); err != nil {
		// If validation rejects the value, the channel still has the clamped/corrected value
		t.Logf("Volume %v validation: %v", config.Volume, err)
	}

	if err := channel.SetPan(config.Pan); err != nil {
		// If validation rejects the value, the channel still has the clamped/corrected value
		t.Logf("Pan %v validation: %v", config.Pan, err)
	}

	return channel
}

// TestDeviceSetup returns the first available input and output devices for testing
func TestDeviceSetup(t *testing.T) (*devices.AudioDevice, *devices.AudioDevice) {
	allDevices, err := devices.GetAudio()
	if err != nil {
		t.Skip("No devices available for testing")
	}

	var outputDevice *devices.AudioDevice
	for i, device := range allDevices {
		if device.CanOutput() && len(device.SupportedSampleRates) > 0 {
			outputDevice = &allDevices[i]
			break
		}
	}

	if outputDevice == nil {
		t.Skip("No output devices available for testing")
	}

	var inputDevice *devices.AudioDevice
	for i, device := range allDevices {
		if device.CanInput() {
			inputDevice = &allDevices[i]
			break
		}
	}

	return outputDevice, inputDevice
}

// ExpectedChannelState holds expected values for channel validation
type ExpectedChannelState struct {
	IsInput            bool
	IsPlayback         bool
	Volume             float32
	Pan                float32
	HasInputOptions    bool
	HasPlaybackOptions bool
	PluginCount        int
	FilePath           string // For playback channels
}

// ValidateChannelState validates that a channel matches expected state
func ValidateChannelState(t *testing.T, channel *Channel, expected ExpectedChannelState) {
	if channel.IsInput() != expected.IsInput {
		t.Errorf("Expected IsInput=%v, got %v", expected.IsInput, channel.IsInput())
	}

	if channel.IsPlayback() != expected.IsPlayback {
		t.Errorf("Expected IsPlayback=%v, got %v", expected.IsPlayback, channel.IsPlayback())
	}

	if channel.Volume != expected.Volume {
		t.Errorf("Expected Volume=%v, got %v", expected.Volume, channel.Volume)
	}

	if channel.Pan != expected.Pan {
		t.Errorf("Expected Pan=%v, got %v", expected.Pan, channel.Pan)
	}

	if (channel.InputOptions != nil) != expected.HasInputOptions {
		t.Errorf("Expected HasInputOptions=%v, got %v", expected.HasInputOptions, channel.InputOptions != nil)
	}

	if (channel.PlaybackOptions != nil) != expected.HasPlaybackOptions {
		t.Errorf("Expected HasPlaybackOptions=%v, got %v", expected.HasPlaybackOptions, channel.PlaybackOptions != nil)
	}

	if expected.HasInputOptions && channel.InputOptions != nil {
		if channel.InputOptions.PluginChain != nil {
			actualPluginCount := len(channel.InputOptions.PluginChain.Plugins)
			if actualPluginCount != expected.PluginCount {
				t.Errorf("Expected PluginCount=%v, got %v", expected.PluginCount, actualPluginCount)
			}
		} else if expected.PluginCount > 0 {
			t.Errorf("Expected PluginCount=%v, but PluginChain is nil", expected.PluginCount)
		}
	}

	if expected.HasPlaybackOptions && channel.PlaybackOptions != nil {
		if channel.PlaybackOptions.FilePath != expected.FilePath {
			t.Errorf("Expected FilePath=%v, got %v", expected.FilePath, channel.PlaybackOptions.FilePath)
		}
	}
}

// ErrorTestCase represents a test case for error handling validation
type ErrorTestCase struct {
	Name            string
	TestFunc        func() error
	WantErr         bool
	ExpectedMessage string // Optional: specific error message to check for
}

// ValidateErrorTestCase runs an error test case and validates the results
func ValidateErrorTestCase(t *testing.T, testCase ErrorTestCase) {
	err := testCase.TestFunc()

	if (err != nil) != testCase.WantErr {
		t.Errorf("Expected error=%v, got error=%v", testCase.WantErr, err)
	}

	if testCase.WantErr && err == nil {
		t.Errorf("Expected an error but got none")
	}

	if testCase.WantErr && err != nil {
		if testCase.ExpectedMessage != "" && err.Error() != testCase.ExpectedMessage {
			t.Errorf("Expected error message '%s', got '%s'", testCase.ExpectedMessage, err.Error())
		}
		t.Logf("✅ Correctly caught error: %v", err)
	}
}

// SerializationTestConfig holds configuration for serialization tests
type SerializationTestConfig struct {
	EngineConfig           TestEngineConfig
	InputChannelConfigs    []TestChannelConfig // Input channels to create
	PlaybackChannelConfigs []TestChannelConfig // Playback channels to create
	ExpectedChannelCount   int                 // Total expected channels after deserialization
}

// DefaultSerializationTestConfig returns a basic serialization test configuration
func DefaultSerializationTestConfig() SerializationTestConfig {
	return SerializationTestConfig{
		EngineConfig:           DefaultTestEngineConfig(),
		InputChannelConfigs:    []TestChannelConfig{DefaultInputChannelConfig()},
		PlaybackChannelConfigs: []TestChannelConfig{DefaultPlaybackChannelConfig()},
		ExpectedChannelCount:   2,
	}
}

// CreateComplexTestEngine creates an engine with multiple channels for serialization testing
func CreateComplexTestEngine(t *testing.T, config SerializationTestConfig) (*Engine, func()) {
	// Create basic engine
	engine, cleanup := CreateTestEngine(t, config.EngineConfig)

	// Add input channels if any input devices are available
	if len(config.InputChannelConfigs) > 0 {
		_, inputDevice := TestDeviceSetup(t)
		if inputDevice != nil {
			for _, channelConfig := range config.InputChannelConfigs {
				CreateTestInputChannel(t, engine, channelConfig)
			}
		} else {
			t.Log("Warning: No input devices available, skipping input channels")
		}
	}

	// Add playback channels
	for _, channelConfig := range config.PlaybackChannelConfigs {
		CreateTestPlaybackChannel(t, engine, channelConfig)
	}

	return engine, cleanup
}
