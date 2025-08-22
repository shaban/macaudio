# Testing guide

This project mixes pure-Go logic with AVFoundation graph operations. To keep CI fast and stable:

- Fast unit tests: default. Avoid real-time waits and hardware assumptions. Use small buffer sizes and taps/metrics.
- Slow/integration/audible: opt-in only. Gate behind an env var. Skip on CI by default.

## Tiers

- Unit: deterministic, no audio hardware, run with `go test ./...`.
- Integration: builds graphs, starts the engine briefly, uses taps to assert signal presence; keep sub-200ms waits.
- Manual audible: requires speakers; only runs when `MACAUDIO_AUDIBLE=1`.

## Conventions

- Use `internal/testutil`:
  - `testutil.SmallSpec()` for faster buffers.
  - `testutil.SkipUnlessEnv(t, "MACAUDIO_AUDIBLE", "1")` to gate audible tests.
  - `testutil.IsCI()` to conditionally skip heavy paths.
- Prefer `t.Helper()`, short sleeps, and `t.Skip()` over flaky assertions.
- Prefer taps and RMS checks over long tones; cap sample durations to <= 100ms where possible.

## Examples

```go
// Gate an audible/manual test
testutil.SkipUnlessEnv(t, "MACAUDIO_AUDIBLE", "1")
```

```go
// Use smaller buffers for speed
eng, _ := engine.New(testutil.SmallSpec())
```
