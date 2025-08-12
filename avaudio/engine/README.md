# AVAudioEngine 1:1 Mapping

This package provides a pure 1:1 mapping to macOS AVAudioEngine primitives via CGO.

## Design Philosophy

This is the **lowest level** of our audio engine architecture:
- `macaudio/avaudio/engine` - Pure 1:1 AVAudioEngine primitives (this package)
- `macaudio/engine` - Higher-level abstractions built on top

## Key Features

- **Exact AVAudioEngine mapping**: No Go-specific abstractions, just direct access
- **Proper error handling**: AVAudioEngine exceptions are caught and converted to Go errors
- **Resource management**: Proper cleanup with `Destroy()` method
- **Nil safety**: All methods handle nil receivers gracefully

## Test Philosophy

Our tests focus on **error handling validation** rather than forcing success:

- `TestEngine_StartWithoutNodes` - Verifies that starting an engine without connected nodes returns an error (not a crash)
- `TestEngine_Destroy` - Tests proper resource cleanup and multiple destroys
- `TestEngine_DestroyNil` - Ensures nil receivers don't crash

This approach lets us catch AVAudioEngine's real constraints early and expose them properly to higher-level code.

## AVAudioEngine Constraints Exposed

1. **Start requires nodes**: Cannot start an engine without input or output nodes connected
2. **Threading**: All operations happen on the main thread (AVAudioEngine requirement)
3. **Resource lifecycle**: Proper cleanup required to avoid memory leaks

## Usage

```go
// Create engine
engine, err := New()
if err != nil {
    // Handle creation failure
}

// Check constraints before starting
if err := engine.Start(); err != nil {
    // Handle start failure (e.g., no nodes connected)
}

// Always cleanup
defer engine.Destroy()
```

## Next Steps

This foundation enables building higher-level abstractions in `macaudio/engine` that:
- Handle node connection automatically
- Provide convenient APIs for common audio tasks
- Abstract away the low-level constraints while preserving the power
