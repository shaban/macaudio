package channel

import (
	"testing"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
	"github.com/shaban/macaudio/avaudio/pluginchain"
	"github.com/shaban/macaudio/internal/testutil"
	"github.com/shaban/macaudio/plugins"
)

// Mock implementations for testing

// mockChannel implements the Channel interface for testing sends
type mockChannel struct {
	name     string
	released bool
}

// TestPhaseInvertToggle verifies that enabling/disabling phase invert rewires safely and is idempotent.
func TestPhaseInvertToggle(t *testing.T) {
	spec := testutil.SmallSpec()
	eng, err := engine.New(spec)
	if err != nil {
		t.Fatalf("engine new: %v", err)
	}
	defer eng.Destroy()

	ch, err := NewBaseChannel(BaseChannelConfig{Name: "phase", EnginePtr: eng.Ptr(), EngineInstance: eng})
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}

	// No effects: toggling invert should be idempotent and not error
	if err := ch.SetPhaseInvert(true); err != nil {
		t.Fatalf("enable invert: %v", err)
	}
	if err := ch.SetPhaseInvert(true); err != nil {
		t.Fatalf("idempotent enable invert: %v", err)
	}
	if !ch.IsPhaseInverted() {
		t.Fatalf("expected phase inverted flag true")
	}
	if err := ch.SetPhaseInvert(false); err != nil {
		t.Fatalf("disable invert: %v", err)
	}
	if ch.IsPhaseInverted() {
		t.Fatalf("expected phase inverted flag false")
	}

	// Add a no-op effect (use first available plugin from plugins package if supported), else skip wiring test
	// For now, just call ConnectPluginChainToMixer and ensure it doesn't error with current state
	if err := ch.ConnectPluginChainToMixer(); err != nil {
		t.Fatalf("connect chain to mixer: %v", err)
	}

	// Toggle invert with non-empty chain path (chain may still be empty; method is safe regardless)
	if err := ch.SetPhaseInvert(true); err != nil {
		t.Fatalf("enable invert (post-connect): %v", err)
	}
	if err := ch.SetPhaseInvert(false); err != nil {
		t.Fatalf("disable invert (post-connect): %v", err)
	}

	// Ensure release cleans up without panic
	ch.Release()
}

func (mc *mockChannel) GetName() string                                             { return mc.name }
func (mc *mockChannel) SetName(name string)                                         { mc.name = name }
func (mc *mockChannel) SetVolume(volume float32) error                              { return nil }
func (mc *mockChannel) GetVolume() (float32, error)                                 { return 0.8, nil }
func (mc *mockChannel) SetMute(muted bool) error                                    { return nil }
func (mc *mockChannel) GetMute() (bool, error)                                      { return false, nil }
func (mc *mockChannel) SetPan(pan float32) error                                    { return nil }
func (mc *mockChannel) GetPan() (float32, error)                                    { return 0.0, nil }
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

func TestChannel_ConnectPluginChainToMixer_WithSingleAppleEffect(t *testing.T) {
	// Create engine
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	// Build channel
	channel, err := NewBaseChannel(BaseChannelConfig{
		Name:           "FX Test",
		EnginePtr:      eng.Ptr(),
		EngineInstance: eng,
	})
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}
	defer channel.Release()

	// Find a fast Apple stock effect (AUBandpass preferred)
	infos, err := plugins.List()
	if err != nil {
		t.Skipf("Skipping: quick plugin list failed: %v", err)
	}
	var pick *plugins.PluginInfo
	for _, i := range infos {
		if i.ManufacturerID == "appl" && i.Type == "aufx" && i.Name == "Apple: AUBandpass" && i.Subtype == "bpas" {
			ii := i
			pick = &ii
			break
		}
	}
	if pick == nil {
		// fallback: any Apple effect
		for _, i := range infos {
			if i.ManufacturerID == "appl" && i.Type == "aufx" {
				ii := i
				pick = &ii
				break
			}
		}
	}
	if pick == nil {
		t.Skip("No suitable Apple stock effect found; skipping")
	}

	// Add effect and connect chain to mixer
	if err := channel.AddEffectFromPluginInfo(*pick); err != nil {
		t.Fatalf("AddEffectFromPluginInfo failed: %v", err)
	}
	if err := channel.ConnectPluginChainToMixer(); err != nil {
		t.Fatalf("ConnectPluginChainToMixer failed: %v", err)
	}
}

func TestChannel_MasterRouting_Smoke(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create AVAudioEngine for testing")
	}
	defer eng.Destroy()

	ch, err := NewBaseChannel(BaseChannelConfig{
		Name:           "Master Smoke",
		EnginePtr:      eng.Ptr(),
		EngineInstance: eng,
	})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Release()

	// Connect to master
	if err := ch.ConnectToMaster(eng); err != nil {
		t.Fatalf("connect to master: %v", err)
	}
	if !ch.IsConnectedToMaster() {
		t.Fatalf("expected connectedToMaster = true")
	}

	// Disconnect
	if err := ch.DisconnectFromMaster(eng); err != nil {
		t.Fatalf("disconnect from master: %v", err)
	}
	if ch.IsConnectedToMaster() {
		t.Fatalf("expected connectedToMaster = false")
	}

	// Reconnect
	if err := ch.ConnectToMaster(eng); err != nil {
		t.Fatalf("reconnect to master: %v", err)
	}
	if !ch.IsConnectedToMaster() {
		t.Fatalf("expected connectedToMaster = true after reconnect")
	}
}

func TestBaseChannel_VolumeAndPan(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine")
	}
	defer eng.Destroy()

	ch, err := NewBaseChannel(BaseChannelConfig{
		Name:           "MixCtl",
		EnginePtr:      eng.Ptr(),
		EngineInstance: eng,
	})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Release()

	// Volume set/get
	if err := ch.SetVolume(0.42); err != nil {
		t.Fatalf("set vol: %v", err)
	}
	if v, err := ch.GetVolume(); err != nil {
		t.Fatalf("get vol: %v", err)
	} else if v == 0.0 {
		t.Fatalf("unexpected zero volume")
	}

	// Pan set/get
	if err := ch.SetPan(-0.5); err != nil {
		t.Fatalf("set pan: %v", err)
	}
	if p, err := ch.GetPan(); err != nil {
		t.Fatalf("get pan: %v", err)
	} else if p < -1.01 || p > 1.01 {
		t.Fatalf("pan out of range: %v", p)
	}
}

func TestChannel_Sends_PreAndPostFader_Smoke(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine")
	}
	defer eng.Destroy()

	// Create a destination bus mixer
	busMixer, err := node.CreateMixer()
	if err != nil || busMixer == nil {
		t.Fatalf("create bus mixer: %v", err)
	}
	defer node.ReleaseMixer(busMixer)

	// Attach bus mixer to engine for receiving connections
	if err := eng.Attach(busMixer); err != nil {
		t.Fatalf("attach bus: %v", err)
	}

	// Build a channel
	ch, err := NewBaseChannel(BaseChannelConfig{
		Name:           "SendChan",
		EnginePtr:      eng.Ptr(),
		EngineInstance: eng,
	})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Release()

	// Post-fader send
	if err := ch.CreateSendWithMode("post", &mockChannel{name: "dst"}, 1.0, PostFader); err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := ch.ConnectSendToBus(eng, "post", busMixer, 0); err != nil {
		t.Fatalf("connect post: %v", err)
	}

	// Pre-fader send (with empty chain, will use mixer as source)
	if err := ch.CreateSendWithMode("pre", &mockChannel{name: "dst"}, 1.0, PreFader); err != nil {
		t.Fatalf("create pre: %v", err)
	}
	if err := ch.ConnectSendToBus(eng, "pre", busMixer, 0); err != nil {
		t.Fatalf("connect pre: %v", err)
	}

	// Disconnect bus input to clean up
	if err := eng.DisconnectNodeInput(busMixer, 0); err != nil {
		t.Fatalf("disconnect bus: %v", err)
	}
}

func TestChannel_Idempotent_Master_And_Send_Rewire(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine")
	}
	defer eng.Destroy()

	// Create bus A and B
	busA, err := node.CreateMixer()
	if err != nil || busA == nil {
		t.Fatalf("create busA: %v", err)
	}
	defer node.ReleaseMixer(busA)
	if err := eng.Attach(busA); err != nil {
		t.Fatalf("attach busA: %v", err)
	}

	busB, err := node.CreateMixer()
	if err != nil || busB == nil {
		t.Fatalf("create busB: %v", err)
	}
	defer node.ReleaseMixer(busB)
	if err := eng.Attach(busB); err != nil {
		t.Fatalf("attach busB: %v", err)
	}

	ch, err := NewBaseChannel(BaseChannelConfig{Name: "Idem", EnginePtr: eng.Ptr(), EngineInstance: eng})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Release()

	// Master idempotency
	if err := ch.ConnectToMaster(eng); err != nil {
		t.Fatalf("connect master: %v", err)
	}
	if err := ch.ConnectToMaster(eng); err != nil {
		t.Fatalf("connect master again should be OK: %v", err)
	}
	if err := ch.DisconnectFromMaster(eng); err != nil {
		t.Fatalf("disconnect master: %v", err)
	}
	if err := ch.DisconnectFromMaster(eng); err != nil {
		t.Fatalf("disconnect master again should be OK: %v", err)
	}

	// Send rewire idempotency
	if err := ch.CreateSendWithMode("aux", &mockChannel{name: "dst"}, 0.5, PostFader); err != nil {
		t.Fatalf("create send: %v", err)
	}
	if err := ch.ConnectSendToBus(eng, "aux", busA, 0); err != nil {
		t.Fatalf("connect busA: %v", err)
	}
	// idempotent re-connect to same target
	if err := ch.ConnectSendToBus(eng, "aux", busA, 0); err != nil {
		t.Fatalf("reconnect same bus should be OK: %v", err)
	}
	// rewire to busB
	if err := ch.ConnectSendToBus(eng, "aux", busB, 0); err != nil {
		t.Fatalf("rewire to busB: %v", err)
	}
}

func TestBus_Create_Connect_Disconnect(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine")
	}
	defer eng.Destroy()

	bus, err := NewBus(eng, "AuxBus")
	if err != nil {
		t.Fatalf("new bus: %v", err)
	}
	defer bus.Release()

	ch, err := NewBaseChannel(BaseChannelConfig{Name: "Src", EnginePtr: eng.Ptr(), EngineInstance: eng})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Release()

	inputIdx, err := bus.ConnectChannel(ch)
	if err != nil {
		t.Fatalf("connect channel->bus: %v", err)
	}
	if inputIdx < 0 {
		t.Fatalf("invalid input idx: %d", inputIdx)
	}

	if err := bus.DisconnectInput(inputIdx); err != nil {
		t.Fatalf("disconnect: %v", err)
	}
}

func TestBus_InputLevelAndPan(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine")
	}
	defer eng.Destroy()

	bus, err := NewBus(eng, "AuxBus2")
	if err != nil {
		t.Fatalf("new bus: %v", err)
	}
	defer bus.Release()

	// Set and get input 0 level/pan (best-effort with current native bridge)
	if err := bus.SetInputLevel(0, 0.6); err != nil {
		t.Fatalf("set level: %v", err)
	}
	if v, err := bus.GetInputLevel(0); err != nil {
		t.Fatalf("get level: %v", err)
	} else if v == 0.0 {
		t.Fatalf("unexpected zero level")
	}

	if err := bus.SetInputPan(0, 0.25); err != nil {
		t.Fatalf("set pan: %v", err)
	}
	if p, err := bus.GetInputPan(0); err != nil {
		t.Fatalf("get pan: %v", err)
	} else if p < -1.01 || p > 1.01 {
		t.Fatalf("pan out of range: %v", p)
	}
}

func TestChannel_Send_LevelAndMute_Control(t *testing.T) {
	eng, err := engine.New(engine.DefaultAudioSpec())
	if err != nil || eng == nil {
		t.Skip("Cannot create engine")
	}
	defer eng.Destroy()

	// Destination bus mixer
	busMixer, err := node.CreateMixer()
	if err != nil || busMixer == nil {
		t.Fatalf("create bus mixer: %v", err)
	}
	defer node.ReleaseMixer(busMixer)
	if err := eng.Attach(busMixer); err != nil {
		t.Fatalf("attach bus: %v", err)
	}

	// Channel
	ch, err := NewBaseChannel(BaseChannelConfig{Name: "SendLevelMute", EnginePtr: eng.Ptr(), EngineInstance: eng})
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Release()

	// Create and connect send
	if err := ch.CreateSendWithMode("aux", &mockChannel{name: "dst"}, 0.7, PostFader); err != nil {
		t.Fatalf("create send: %v", err)
	}
	if err := ch.ConnectSendToBus(eng, "aux", busMixer, 0); err != nil {
		t.Fatalf("connect send: %v", err)
	}

	// Adjust level down and mute/unmute; ensure no errors
	if err := ch.SetSendLevel("aux", 0.3); err != nil {
		t.Fatalf("set level: %v", err)
	}
	if err := ch.SetSendMute("aux", true); err != nil {
		t.Fatalf("mute send: %v", err)
	}
	if err := ch.SetSendMute("aux", false); err != nil {
		t.Fatalf("unmute send: %v", err)
	}

	// Disconnect send wires cleanly
	if err := ch.DisconnectSend(eng, "aux"); err != nil {
		t.Fatalf("disconnect send: %v", err)
	}
}
