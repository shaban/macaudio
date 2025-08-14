package testutil

import (
    "testing"
    "time"
    "unsafe"

    "github.com/shaban/macaudio/avaudio/engine"
    "github.com/shaban/macaudio/avaudio/tap"
)

// AssertRMSAbove installs a temporary tap and asserts RMS exceeds threshold within timeout.
func AssertRMSAbove(t *testing.T, eng *engine.Engine, nodePtr unsafe.Pointer, bus int, minRMS float64, timeout time.Duration) {
    t.Helper()
    if eng == nil || eng.Ptr() == nil { t.Fatalf("engine is nil") }
    if nodePtr == nil { t.Fatalf("nodePtr is nil") }
    tp, err := tap.InstallTap(eng.Ptr(), nodePtr, bus)
    if err != nil { t.Fatalf("install tap: %v", err) }
    defer tp.Remove()
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        m, err := tp.GetMetrics()
        if err == nil && m.RMS >= minRMS && m.FrameCount > 0 {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatalf("signal below threshold: wanted >= %.6f within %s", minRMS, timeout)
}
