package session

import (
	aveng "github.com/shaban/macaudio/avaudio/engine"
)

// MapLatencyToBuffer maps a LatencyClass to a suggested buffer size in frames.
// These are opinionated defaults tuned for Core v1.
func MapLatencyToBuffer(c LatencyClass) int {
	switch c {
	case LatencyLow:
		return 128
	case LatencyHigh:
		return 1024
	case LatencyMedium:
		fallthrough
	default:
		return 256
	}
}

// ResolveEngineSpec converts a Session AudioSpec into an avaudio engine.AudioSpec.
// Rules:
// - If BufferSize is set (>0), it overrides LatencyHint mapping.
// - SampleRate uses PreferredSampleRate when >0, else avaudio's default.
// - ChannelCount defaults to 2 (stereo) and BitDepth to 32-bit float.
func ResolveEngineSpec(s AudioSpec) aveng.AudioSpec {
	// Start with avaudio defaults
	eff := aveng.DefaultAudioSpec()

	if s.PreferredSampleRate > 0 {
		eff.SampleRate = s.PreferredSampleRate
	}

	if s.BufferSize > 0 {
		eff.BufferSize = s.BufferSize
	} else {
		eff.BufferSize = MapLatencyToBuffer(s.LatencyHint)
	}

	// Keep defaults for bit depth and channels (engines run 32f stereo)
	return eff
}
