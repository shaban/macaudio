package macaudio

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// EngineState represents the complete serializable state of the engine
type EngineState struct {
	Version        string                 `json:"version"`
	Configuration  EngineConfig           `json:"configuration"`
	Channels       map[string]ChannelState `json:"channels"`
	Connections    []Connection           `json:"connections"`
	Timestamp      int64                  `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Serializer handles engine state persistence and restoration
type Serializer struct {
	engine  *Engine
	mu      sync.RWMutex
	version string
}

// NewSerializer creates a new serializer
func NewSerializer(engine *Engine) *Serializer {
	return &Serializer{
		engine:  engine,
		version: "1.0.0", // Engine state format version
	}
}

// GetState captures the complete engine state
func (s *Serializer) GetState() EngineState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Get all channel states
	channels := make(map[string]ChannelState)
	allConnections := make([]Connection, 0)
	
	for id, channel := range s.engine.channels {
		state := channel.GetState()
		channels[id] = state
		
		// Collect all connections
		for _, conn := range state.Connections {
			allConnections = append(allConnections, conn)
		}
	}
	
	return EngineState{
		Version:       s.version,
		Configuration: s.engine.GetConfiguration(),
		Channels:      channels,
		Connections:   allConnections,
		Timestamp:     0, // TODO: Add actual timestamp
		Metadata:      make(map[string]interface{}),
	}
}

// SetState restores the engine from the given state
func (s *Serializer) SetState(state EngineState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Version compatibility check
	if state.Version != s.version {
		return fmt.Errorf("incompatible state version: got %s, expected %s", 
			state.Version, s.version)
	}
	
	// Clear existing channels (except master)
	for id := range s.engine.channels {
		if id != "master" {
			if err := s.engine.removeChannel(id); err != nil {
				return fmt.Errorf("failed to remove channel %s during state restore: %w", id, err)
			}
		}
	}
	
	// Restore channels
	for id, channelState := range state.Channels {
		if id == "master" {
			// Update master channel state
			if err := s.engine.masterChannel.SetState(channelState); err != nil {
				return fmt.Errorf("failed to restore master channel state: %w", err)
			}
			continue
		}
		
		// Create new channel based on type
		channel, err := s.createChannelFromState(id, channelState)
		if err != nil {
			return fmt.Errorf("failed to create channel %s from state: %w", id, err)
		}
		
		// Add to engine
		if err := s.engine.addChannel(channel); err != nil {
			return fmt.Errorf("failed to add restored channel %s: %w", id, err)
		}
		
		// Restore channel state
		if err := channel.SetState(channelState); err != nil {
			return fmt.Errorf("failed to restore state for channel %s: %w", id, err)
		}
	}
	
	// Restore connections (handled by channel state restoration)
	
	return nil
}

// SaveToWriter saves the engine state to a writer (JSON format)
func (s *Serializer) SaveToWriter(writer io.Writer) error {
	state := s.GetState()
	
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ") // Pretty print
	
	if err := encoder.Encode(state); err != nil {
		return fmt.Errorf("failed to encode engine state: %w", err)
	}
	
	return nil
}

// LoadFromReader loads engine state from a reader (JSON format)
func (s *Serializer) LoadFromReader(reader io.Reader) error {
	var state EngineState
	
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&state); err != nil {
		return fmt.Errorf("failed to decode engine state: %w", err)
	}
	
	return s.SetState(state)
}

// SaveToJSON returns the engine state as JSON string
func (s *Serializer) SaveToJSON() (string, error) {
	state := s.GetState()
	
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal engine state: %w", err)
	}
	
	return string(data), nil
}

// LoadFromJSON restores engine state from JSON string
func (s *Serializer) LoadFromJSON(jsonData string) error {
	var state EngineState
	
	if err := json.Unmarshal([]byte(jsonData), &state); err != nil {
		return fmt.Errorf("failed to unmarshal engine state: %w", err)
	}
	
	return s.SetState(state)
}

// createChannelFromState creates a channel instance from serialized state
func (s *Serializer) createChannelFromState(id string, state ChannelState) (Channel, error) {
	switch state.Type {
	case ChannelTypeAudioInput:
		// Extract audio input config from state.Config
		config := AudioInputConfig{}
		if state.Config != nil {
			if deviceUID, ok := state.Config["deviceUID"].(string); ok {
				config.DeviceUID = deviceUID
			}
			if inputBus, ok := state.Config["inputBus"].(float64); ok {
				config.InputBus = int(inputBus)
			}
			if monitoringLevel, ok := state.Config["monitoringLevel"].(float64); ok {
				config.MonitoringLevel = float32(monitoringLevel)
			}
		}
		return NewAudioInputChannel(id, config, s.engine)
		
	case ChannelTypeMidiInput:
		// Extract MIDI input config from state.Config
		config := MidiInputConfig{}
		if state.Config != nil {
			if deviceUID, ok := state.Config["deviceUID"].(string); ok {
				config.DeviceUID = deviceUID
			}
			if channel, ok := state.Config["channel"].(float64); ok {
				config.Channel = int(channel)
			}
		}
		return NewMidiInputChannel(id, config, s.engine)
		
	case ChannelTypePlayback:
		// Extract playback config from state.Config
		config := PlaybackConfig{}
		if state.Config != nil {
			if filePath, ok := state.Config["filePath"].(string); ok {
				config.FilePath = filePath
			}
			if loopEnabled, ok := state.Config["loopEnabled"].(bool); ok {
				config.LoopEnabled = loopEnabled
			}
			if autoStart, ok := state.Config["autoStart"].(bool); ok {
				config.AutoStart = autoStart
			}
			if fadeIn, ok := state.Config["fadeIn"].(float64); ok {
				config.FadeIn = float32(fadeIn)
			}
			if fadeOut, ok := state.Config["fadeOut"].(float64); ok {
				config.FadeOut = float32(fadeOut)
			}
		}
		return NewPlaybackChannel(id, config, s.engine)
		
	case ChannelTypeAux:
		// Extract aux config from state.Config
		config := AuxConfig{}
		if state.Config != nil {
			if sendLevel, ok := state.Config["sendLevel"].(float64); ok {
				config.SendLevel = float32(sendLevel)
			}
			if returnLevel, ok := state.Config["returnLevel"].(float64); ok {
				config.ReturnLevel = float32(returnLevel)
			}
			if preFader, ok := state.Config["preFader"].(bool); ok {
				config.PreFader = preFader
			}
		}
		return NewAuxChannel(id, config, s.engine)
		
	default:
		return nil, fmt.Errorf("unknown channel type: %s", state.Type)
	}
}

// GetVersion returns the current serializer version
func (s *Serializer) GetVersion() string {
	return s.version
}

// IsCompatible checks if a state version is compatible with current serializer
func (s *Serializer) IsCompatible(version string) bool {
	// For now, only exact version match
	// In the future, this could handle backward compatibility
	return version == s.version
}

// ValidateState validates the integrity of an engine state
func (s *Serializer) ValidateState(state EngineState) error {
	// Version check
	if !s.IsCompatible(state.Version) {
		return fmt.Errorf("incompatible state version: %s", state.Version)
	}
	
	// Master channel must exist
	if _, exists := state.Channels["master"]; !exists {
		return fmt.Errorf("master channel missing from state")
	}
	
	// Validate channel references in connections
	channelIDs := make(map[string]bool)
	for id := range state.Channels {
		channelIDs[id] = true
	}
	
	for _, conn := range state.Connections {
		if !channelIDs[conn.SourceChannel] {
			return fmt.Errorf("connection references unknown source channel: %s", conn.SourceChannel)
		}
		if !channelIDs[conn.TargetChannel] {
			return fmt.Errorf("connection references unknown target channel: %s", conn.TargetChannel)
		}
	}
	
	return nil
}
