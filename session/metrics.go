//go:build darwin

package session

import "time"

// MetricsHook allows callers to observe key events and durations in the session.
// Implementers can log, aggregate metrics, or emit traces. All methods are optional.
type MetricsHook interface {
    // Quick scan lifecycle (List() of PluginInfos)
    OnQuickScanStart()
    OnQuickScanDone(duration time.Duration, count int, scanned bool)

    // Details fetch lifecycle for a specific key (quadruplet)
    OnDetailsFetchStart(key string)
    OnDetailsFetchDone(key string, duration time.Duration, success bool)

    // Cache signals for details
    OnCacheHit(key string)
    OnCacheMiss(key string)

    // RefreshQuick diff summary
    OnRefreshQuickDiff(added, removed, changed int, duration time.Duration)

    // Warm progress updates
    OnWarmProgress(total, completed int)
}
