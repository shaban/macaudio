package input

import (
	"fmt"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
	"github.com/shaban/macaudio/engine/channel"
)

// MonoToStereoChannel represents a mono input that converts to stereo output
// with configurable panning, volume control, and plugin chain processing.
type MonoToStereoChannel struct {
	*channel.BaseChannel
	pan float32 // Pan position: -1.0 (left) to +1.0 (right)
}

// MonoToStereoConfig contains configuration for creating a MonoToStereoChannel
type MonoToStereoConfig struct {
	Name       string         // Channel name
	Engine     *engine.Engine // High-level engine (contains everything we need)
	InitialPan float32        // Initial pan position (-1.0 to +1.0)
}

// NewMonoToStereo creates a new mono-to-stereo input channel
func NewMonoToStereo(config MonoToStereoConfig) (*MonoToStereoChannel, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("channel name cannot be empty")
	}
	if config.Engine == nil {
		return nil, fmt.Errorf("engine cannot be nil")
	}

	// Validate pan range
	if config.InitialPan < -1.0 || config.InitialPan > 1.0 {
		return nil, fmt.Errorf("pan must be between -1.0 and +1.0, got %.2f", config.InitialPan)
	}

	// Create base channel configuration (derive low-level pointer from high-level engine)
	baseConfig := channel.BaseChannelConfig{
		Name:           config.Name,
		EnginePtr:      config.Engine.Ptr(), // Derive from high-level engine
		EngineInstance: config.Engine,
	}

	// Create base channel (uses a mixer node internally)
	baseChannel, err := channel.NewBaseChannel(baseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create base channel: %w", err)
	}

	// Create the mono-to-stereo channel
	monoChannel := &MonoToStereoChannel{
		BaseChannel: baseChannel,
		pan:         config.InitialPan,
	}

	// Set initial pan
	err = monoChannel.SetPan(config.InitialPan)
	if err != nil {
		monoChannel.Release()
		return nil, fmt.Errorf("failed to set initial pan: %w", err)
	}

	return monoChannel, nil
}

// SetPan sets the pan position for the channel
// -1.0 = full left, 0.0 = center, +1.0 = full right
func (m *MonoToStereoChannel) SetPan(pan float32) error {
	if pan < -1.0 || pan > 1.0 {
		return fmt.Errorf("pan must be between -1.0 and +1.0, got %.2f", pan)
	}

	// Set the pan on the underlying mixer node
	err := node.SetMixerPan(m.GetOutputNode(), pan, 0)
	if err != nil {
		return fmt.Errorf("failed to set mixer pan: %w", err)
	}

	m.pan = pan
	return nil
}

// GetPan returns the current pan position
func (m *MonoToStereoChannel) GetPan() float32 {
	return m.pan
}

// SetPanLeft sets the channel to full left (-1.0)
func (m *MonoToStereoChannel) SetPanLeft() error {
	return m.SetPan(-1.0)
}

// SetPanRight sets the channel to full right (+1.0)
func (m *MonoToStereoChannel) SetPanRight() error {
	return m.SetPan(1.0)
}

// SetPanCenter sets the channel to center (0.0)
func (m *MonoToStereoChannel) SetPanCenter() error {
	return m.SetPan(0.0)
}

// Summary returns a detailed string representation of the channel state
func (m *MonoToStereoChannel) Summary() string {
	baseSummary := m.BaseChannel.Summary()
	return fmt.Sprintf("%s, Pan: %.2f", baseSummary, m.pan)
}
