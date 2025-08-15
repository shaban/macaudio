package testutil

import (
	"os"
	"testing"

	"github.com/shaban/macaudio/avaudio/engine"
	"github.com/shaban/macaudio/avaudio/node"
	"github.com/shaban/macaudio/engine/queue"
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
	if s.BufferSize > 256 {
		s.BufferSize = 256
	}
	return s
}

// MuteMainMixer sets the main mixer volume to 0.0 to avoid audible output during tests.
func MuteMainMixer(t *testing.T, eng *engine.Engine) {
	t.Helper()
	if eng == nil {
		t.Fatalf("engine is nil")
	}
	mm, err := eng.MainMixerNode()
	if err != nil || mm == nil {
		t.Fatalf("get main mixer: %v", err)
	}
	_ = node.SetMixerVolume(mm, 0.0, 0)
}

// MuteMainMixerNoT mutes the main mixer without requiring a testing.T.
// Returns any error encountered for callers to decide.
func MuteMainMixerNoT(eng *engine.Engine) error {
	if eng == nil {
		return nil
	}
	mm, err := eng.MainMixerNode()
	if err != nil || mm == nil {
		return err
	}
	return node.SetMixerVolume(mm, 0.0, 0)
}

// NewEngineForTest creates an Engine using SmallSpec tuned for tests,
// mutes the main mixer for quiet runs, and wires t.Cleanup to destroy it.
// If the engine cannot be created, the test is skipped.
func NewEngineForTest(t *testing.T) *engine.Engine {
	t.Helper()
	eng, err := engine.New(SmallSpec())
	if err != nil || eng == nil {
		t.Skipf("skipping: cannot create engine: %v", err)
		return nil
	}
	// Keep tests quiet by default
	MuteMainMixer(t, eng)
	t.Cleanup(func() { eng.Destroy() })
	return eng
}

// NewDispatcherForTest creates and starts a dispatcher bound to the given engine,
// and registers t.Cleanup to close it. Useful for dispatcher-backed routing tests.
func NewDispatcherForTest(t *testing.T, eng *engine.Engine) *queue.Dispatcher {
	t.Helper()
	if eng == nil {
		t.Fatalf("engine is nil")
	}
	disp := queue.NewDispatcher(eng, queue.New(32))
	disp.Start()
	t.Cleanup(func() { disp.Close() })
	return disp
}
