package channel

import (
	"testing"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/pluginchain"
	"github.com/shaban/macaudio/plugins"
)

// Mock implementations for testing

// mockChannel implements the Channel interface for testing sends
type mockChannel struct {
	name     string
	released bool
}

func (mc *mockChannel) GetName() string                                             { return mc.name }
func (mc *mockChannel) SetName(name string)                                         { mc.name = name }
func (mc *mockChannel) SetVolume(volume float32) error                              { return nil }
func (mc *mockChannel) GetVolume() (float32, error)                                 { return 0.8, nil }
func (mc *mockChannel) SetMute(muted bool) error                                    { return nil }
func (mc *mockChannel) GetMute() (bool, error)                                      { return false, nil }
func (mc *mockChannel) GetPluginChain() *pluginchain.PluginChain                    { return nil }
func (mc *mockChannel) AddEffect(plugin *plugins.Plugin) error                      { return nil }
func (mc *mockChannel) AddEffectFromPluginInfo(pluginInfo plugins.PluginInfo) error { return nil }
func (mc *mockChannel) GetInputNode() unsafe.Pointer                                { return nil }
func (mc *mockChannel) GetOutputNode() unsafe.Pointer                               { return nil }
func (mc *mockChannel) Release()                                                    { mc.released = true }
func (mc *mockChannel) IsReleased() bool                                            { return mc.released }
func (mc *mockChannel) Summary() string                                             { return "Mock Channel: " + mc.name }

func TestNewBaseChannel(t *testing.T) {
	// Create a real engine for valid test cases
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine for testing")
	}
	defer eng.Destroy()

	tests := []struct {
		name      string
		config    BaseChannelConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "ValidConfig",
			config: BaseChannelConfig{
				Name:           "Test Channel",
				EnginePtr:      eng.Ptr(), // Real engine pointer
				EngineInstance: eng,       // Real engine instance
			},
			expectErr: false,
		},
		{
			name: "EmptyName",
			config: BaseChannelConfig{
				Name:           "",
				EnginePtr:      eng.Ptr(), // Real engine pointer
				EngineInstance: eng,       // Real engine instance
			},
			expectErr: true,
			errMsg:    "channel name cannot be empty",
		},
		{
			name: "NilEnginePtr",
			config: BaseChannelConfig{
				Name:           "Test Channel",
				EnginePtr:      nil, // Test nil pointer
				EngineInstance: eng, // Valid instance
			},
			expectErr: true,
			errMsg:    "engine pointer cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := NewBaseChannel(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if err.Error() != tt.errMsg {
					t.Fatalf("Expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if channel == nil {
				t.Fatal("Expected non-nil channel")
			}

			// Test basic properties
			if channel.GetName() != tt.config.Name {
				t.Errorf("Expected name '%s', got '%s'", tt.config.Name, channel.GetName())
			}

			if channel.IsReleased() {
				t.Error("New channel should not be released")
			}

			// Test plugin chain creation
			pluginChain := channel.GetPluginChain()
			if pluginChain == nil {
				t.Error("Plugin chain should not be nil")
			}

			// Test summary
			summary := channel.Summary()
			if summary == "" {
				t.Error("Summary should not be empty")
			}
			t.Logf("Channel summary: %s", summary)

			// Clean up
			channel.Release()
			if !channel.IsReleased() {
				t.Error("Channel should be released after Release() call")
			}
		})
	}
}

func TestBaseChannelNaming(t *testing.T) {
	// Create channel with real engine for testing
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine for testing")
	}
	defer eng.Destroy()

	config := BaseChannelConfig{
		Name:           "Original Name",
		EnginePtr:      eng.Ptr(),
		EngineInstance: eng,
	}

	channel, err := NewBaseChannel(config)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}
	defer channel.Release()

	// Test initial name
	if channel.GetName() != "Original Name" {
		t.Errorf("Expected name 'Original Name', got '%s'", channel.GetName())
	}

	// Test name change
	channel.SetName("New Name")
	if channel.GetName() != "New Name" {
		t.Errorf("Expected name 'New Name', got '%s'", channel.GetName())
	}

	// Test that plugin chain name also updates
	pluginChain := channel.GetPluginChain()
	if pluginChain != nil {
		expectedChainName := "New Name Chain"
		if pluginChain.GetName() != expectedChainName {
			t.Errorf("Expected plugin chain name '%s', got '%s'", expectedChainName, pluginChain.GetName())
		}
	}
}

func TestBaseChannelRelease(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine for testing")
	}
	defer eng.Destroy()

	config := BaseChannelConfig{
		Name:           "Release Test Channel",
		EnginePtr:      eng.Ptr(),
		EngineInstance: eng,
	}

	channel, err := NewBaseChannel(config)
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Test operations before release
	if channel.IsReleased() {
		t.Error("Channel should not be released initially")
	}

	// Test basic operations work before release
	pluginChain := channel.GetPluginChain()
	if pluginChain == nil {
		t.Error("Plugin chain should be available before release")
	}

	// Release the channel
	channel.Release()

	if !channel.IsReleased() {
		t.Error("Channel should be released after Release() call")
	}

	// Test double release (should be safe)
	channel.Release() // Should not panic or error
}

func TestBaseChannelWithRealEngine(t *testing.T) {
	t.Log("Testing BaseChannel with real AVAudioEngine...")

	// Create a real engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	config := BaseChannelConfig{
		Name:           "Real Engine Test",
		EnginePtr:      eng.Ptr(),
		EngineInstance: eng,
	}

	channel, err := NewBaseChannel(config)
	if err != nil {
		t.Fatalf("Failed to create channel with real engine: %v", err)
	}
	defer channel.Release()

	// Test that input and output nodes are created
	inputNode := channel.GetInputNode()
	outputNode := channel.GetOutputNode()

	if inputNode == nil {
		t.Error("Input node should not be nil")
	}

	if outputNode == nil {
		t.Error("Output node should not be nil")
	}

	// Initially, with no effects, input and output should be the same (output mixer)
	if inputNode != outputNode {
		t.Log("Input and output nodes differ (this is expected if plugin chain is not empty)")
	}

	t.Logf("✓ Channel created successfully with input node: %p, output node: %p", inputNode, outputNode)
	t.Logf("✓ Channel summary: %s", channel.Summary())

	// With empty plugin chain, ConnectPluginChainToMixer should be a no-op
	if err := channel.ConnectPluginChainToMixer(); err != nil {
		t.Fatalf("ConnectPluginChainToMixer on empty chain should not error: %v", err)
	}
}
