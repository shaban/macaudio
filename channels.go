package macaudio

// Channel represents the common interface for all audio channel types
type Channel interface {
	// Identity and lifecycle
	GetID() string
	GetType() ChannelType
	Start() error
	Stop() error
	IsRunning() bool
	
	// Connections
	ConnectTo(target Channel, bus int) error
	DisconnectFrom(target Channel, bus int) error
	GetConnections() []Connection
	
	// Plugin chain management
	GetPluginChain() *PluginChain
	AddPlugin(blueprint PluginBlueprint, position int) (*PluginInstance, error)
	RemovePlugin(instanceID string) error
	
	// Audio processing
	SetVolume(volume float32) error
	GetVolume() (float32, error)
	SetPan(pan float32) error
	GetPan() (float32, error)
	SetMute(muted bool) error
	GetMute() (bool, error)
	
	// Serialization for state persistence
	GetState() ChannelState
	SetState(state ChannelState) error
}

// ChannelType represents the different types of audio channels
type ChannelType string

const (
	ChannelTypeAudioInput ChannelType = "audio_input"
	ChannelTypeMidiInput  ChannelType = "midi_input"
	ChannelTypePlayback   ChannelType = "playback"
	ChannelTypeAux        ChannelType = "aux"
	ChannelTypeMaster     ChannelType = "master"
)

// Connection represents a connection between channels
type Connection struct {
	SourceChannel string
	TargetChannel string
	SourceBus     int
	TargetBus     int
}

// ChannelState represents the serializable state of a channel
type ChannelState struct {
	ID          string            `json:"id"`
	Type        ChannelType       `json:"type"`
	Volume      float32           `json:"volume"`
	Pan         float32           `json:"pan"`
	Muted       bool              `json:"muted"`
	Connections []Connection      `json:"connections"`
	PluginChain PluginChainState  `json:"pluginChain"`
	Config      map[string]interface{} `json:"config,omitempty"`
}
