// Package channel: Bus abstraction for simple mix buses backed by AVAudioMixerNode.
package channel

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
)

// Bus wraps an AVAudioMixerNode to act as a simple mix bus with input allocation.
// It manages a dedicated mixer node, attaches it to the engine, and tracks the
// next free input index for convenience.
type Bus struct {
	name      string
	eng       *engine.Engine
	mixer     unsafe.Pointer
	mu        sync.Mutex
	nextInput int
	inputs    map[int]unsafe.Pointer // input index -> source node pointer
}

// NewBus creates and attaches a new mixer-backed bus.
func NewBus(eng *engine.Engine, name string) (*Bus, error) {
	if eng == nil {
		return nil, fmt.Errorf("engine instance cannot be nil")
	}
	m, err := node.CreateMixer()
	if err != nil || m == nil {
		return nil, fmt.Errorf("create bus mixer: %v", err)
	}
	if err := eng.Attach(m); err != nil {
		_ = node.ReleaseMixer(m)
		return nil, fmt.Errorf("attach bus mixer: %w", err)
	}
	return &Bus{name: name, eng: eng, mixer: m, inputs: make(map[int]unsafe.Pointer)}, nil
}

// Ptr returns the underlying mixer node pointer.
func (b *Bus) Ptr() unsafe.Pointer { return b.mixer }

// Name returns the bus name.
func (b *Bus) Name() string { return b.name }

// NextInput returns the next free input index and increments the counter.
func (b *Bus) NextInput() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx := b.nextInput
	b.nextInput++
	return idx
}

// ConnectChannel connects a channel's output to the bus at the next free input.
// Returns the input index used.
func (b *Bus) ConnectChannel(ch Channel) (int, error) {
	if b == nil || b.mixer == nil || b.eng == nil {
		return -1, fmt.Errorf("bus not initialized")
	}
	if ch == nil {
		return -1, fmt.Errorf("channel cannot be nil")
	}
	src := ch.GetOutputNode()
	if src == nil {
		return -1, fmt.Errorf("channel output node is nil")
	}
	// Ensure nodes are attached as needed
	if installed, err := node.IsInstalledOnEngine(src); err == nil && !installed {
		if err := b.eng.Attach(src); err != nil {
			return -1, fmt.Errorf("attach source: %w", err)
		}
	}
	// Allocate input and connect
	to := b.NextInput()
	if err := b.eng.Connect(src, b.mixer, 0, to); err != nil {
		return -1, fmt.Errorf("connect channel->bus: %w", err)
	}
	b.mu.Lock()
	b.inputs[to] = src
	b.mu.Unlock()
	return to, nil
}

// DisconnectInput disconnects a specific input bus on the bus mixer.
func (b *Bus) DisconnectInput(input int) error {
	if b == nil || b.mixer == nil || b.eng == nil {
		return fmt.Errorf("bus not initialized")
	}
	err := b.eng.DisconnectNodeInput(b.mixer, input)
	b.mu.Lock()
	delete(b.inputs, input)
	b.mu.Unlock()
	return err
}

// Release detaches and releases the bus mixer.
func (b *Bus) Release() {
	if b == nil || b.mixer == nil {
		return
	}
	_ = node.ReleaseMixer(b.mixer)
	b.mixer = nil
}

// MasterBus provides a view of the engine's main mixer as a Bus-like target.
// It does not own the main mixer lifetime.
type MasterBus struct {
	eng   *engine.Engine
	mixer unsafe.Pointer
}

// NewMasterBus fetches the engine's main mixer and wraps it.
func NewMasterBus(eng *engine.Engine) (*MasterBus, error) {
	if eng == nil {
		return nil, fmt.Errorf("engine instance cannot be nil")
	}
	mm, err := eng.MainMixerNode()
	if err != nil || mm == nil {
		return nil, fmt.Errorf("main mixer: %v", err)
	}
	return &MasterBus{eng: eng, mixer: mm}, nil
}

// Ptr returns the mixer pointer for MasterBus.
func (m *MasterBus) Ptr() unsafe.Pointer { return m.mixer }

// SetInputLevel sets the gain for the given input bus on the bus mixer.
// Note: current native bridge applies volume at the mixer level; per-input control
// is emulated by dedicating this bus to a small number of sources.
func (b *Bus) SetInputLevel(input int, level float32) error {
	if b == nil || b.mixer == nil {
		return fmt.Errorf("bus not initialized")
	}
	b.mu.Lock()
	src := b.inputs[input]
	b.mu.Unlock()
	if src != nil {
		if err := node.SetConnectionInputVolume(src, b.mixer, input, level); err == nil {
			return nil
		}
	}
	// Fallback to mixer-level setter if per-connection not available
	return node.SetMixerVolume(b.mixer, level, input)
}

// GetInputLevel reads the gain for the given input bus on the bus mixer.
func (b *Bus) GetInputLevel(input int) (float32, error) {
	if b == nil || b.mixer == nil {
		return 0, fmt.Errorf("bus not initialized")
	}
	b.mu.Lock()
	src := b.inputs[input]
	b.mu.Unlock()
	if src != nil {
		if v, err := node.GetConnectionInputVolume(src, b.mixer, input); err == nil {
			return v, nil
		}
	}
	return node.GetMixerVolume(b.mixer, input)
}

// SetInputPan sets the pan for the given input bus on the bus mixer.
// Note: current native bridge applies pan at the mixer level; true per-input
// requires native support. This is a best-effort shim.
func (b *Bus) SetInputPan(input int, pan float32) error {
	if b == nil || b.mixer == nil {
		return fmt.Errorf("bus not initialized")
	}
	b.mu.Lock()
	src := b.inputs[input]
	b.mu.Unlock()
	if src != nil {
		if err := node.SetConnectionInputPan(src, b.mixer, input, pan); err == nil {
			return nil
		}
	}
	return node.SetMixerPan(b.mixer, pan, input)
}

// GetInputPan reads the pan for the given input bus on the bus mixer.
func (b *Bus) GetInputPan(input int) (float32, error) {
	if b == nil || b.mixer == nil {
		return 0, fmt.Errorf("bus not initialized")
	}
	b.mu.Lock()
	src := b.inputs[input]
	b.mu.Unlock()
	if src != nil {
		if v, err := node.GetConnectionInputPan(src, b.mixer, input); err == nil {
			return v, nil
		}
	}
	return node.GetMixerPan(b.mixer, input)
}
