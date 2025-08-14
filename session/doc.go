//go:build darwin

// Package session provides a high-level, macOS-only orchestration layer for:
//   - Device monitoring (audio + MIDI) with fast, non-blocking change detection
//   - Plugin discovery with a two-tier cache (quick index + per-plugin details)
//   - Convenience APIs for quick scans, lazy details, and optional warm-up
//
// It is intentionally opinionated around performance and responsiveness:
//   - Device change detection favors atomic count polling, then fan-out async scans
//   - Plugin discovery keeps a quick index in index.json and full details in
//     details/<hash>.json, enabling millisecond startup with cache hits
//   - Single-flight deduplication prevents duplicate details introspections
//
// Consumers can attach a MetricsHook to observe timings and cache behavior.
package session
