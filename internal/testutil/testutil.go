package testutil

import (
    "os"
    "testing"

    "github.com/shaban/macaudio/avaudio/engine"
    "github.com/shaban/macaudio/avaudio/node"
)

// SkipUnlessEnv skips the test unless the given env var equals the wanted value.
func SkipUnlessEnv(t *testing.T, key, want string) {
    t.Helper()
    if os.Getenv(key) != want {
        t.Skipf("skipped: set %s=%s to run", key, want)
    }
}

// IsCI reports whether running under common CI environments.
func IsCI() bool {
    if os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true" {
        return true
    }
    return false
}

// SmallSpec returns a default AudioSpec tuned for faster tests.
func SmallSpec() engine.AudioSpec {
    s := engine.DefaultAudioSpec()
    if s.BufferSize > 256 { s.BufferSize = 256 }
    return s
}

// MuteMainMixer sets the main mixer volume to 0.0 to avoid audible output during tests.
func MuteMainMixer(t *testing.T, eng *engine.Engine) {
    t.Helper()
    if eng == nil { t.Fatalf("engine is nil") }
    mm, err := eng.MainMixerNode()
    if err != nil || mm == nil { t.Fatalf("get main mixer: %v", err) }
    _ = node.SetMixerVolume(mm, 0.0, 0)
}

// MuteMainMixerNoT mutes the main mixer without requiring a testing.T.
// Returns any error encountered for callers to decide.
func MuteMainMixerNoT(eng *engine.Engine) error {
    if eng == nil { return nil }
    mm, err := eng.MainMixerNode()
    if err != nil || mm == nil { return err }
    return node.SetMixerVolume(mm, 0.0, 0)
}
