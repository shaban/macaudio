package spec

import (
    avengine "github.com/shaban/macaudio/avaudio/engine"
    sess "github.com/shaban/macaudio/session"
)

// Resolve maps session-level preferences to avaudio engine specs.
func Resolve(s sess.AudioSpec) avengine.AudioSpec {
    targetRate := s.PreferredSampleRate
    if targetRate <= 0 {
        targetRate = 48000
    }

    // Map latency hint to default buffer size, unless explicit BufferSize set
    buf := s.BufferSize
    if buf <= 0 {
        switch s.LatencyHint {
        case sess.LatencyLow:
            buf = 256
        case sess.LatencyHigh:
            buf = 1024
        default:
            buf = 512
        }
    }

    // Engines work in 32-bit float stereo typically; keep legacy overrides if provided
    ch := s.ChannelCount
    if ch <= 0 { ch = 2 }
    bd := s.BitDepth
    if bd <= 0 { bd = 32 }

    return avengine.AudioSpec{
        SampleRate:   targetRate,
        BufferSize:   buf,
        BitDepth:     bd,
        ChannelCount: ch,
    }
}
