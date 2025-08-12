# AVAudioSourceNode 1:1 Mapping

This package provides a pure 1:1 mapping to macOS AVAudioSourceNode primitives via CGO.

## Design Philosophy

This is a **low-level building block** in our audio engine architecture:
- Pure 1:1 mapping to AVAudioSourceNode - no abstractions
- Basic silence-generating render block (foundation for audio generation)
- Proper resource management with `Destroy()` method
- Nil safety throughout

## Key Features

- **Exact AVAudioSourceNode mapping**: Direct access to native audio source node
- **Basic render block**: Currently outputs silence - foundation for audio generation  
- **Resource management**: Proper cleanup prevents memory leaks
- **Nil safety**: All methods handle nil receivers gracefully
- **Bridge stability**: Tested with multiple concurrent instances

## Test Philosophy

Our tests focus on **bridge solidity** and **error handling**:

- `TestSourceNode_New` - Verifies successful creation
- `TestSourceNode_GetNodePtr*` - Tests pointer access and nil handling
- `TestSourceNode_Destroy*` - Tests resource cleanup and multiple destroys
- `TestSourceNode_Bridge_Solidity` - Stress tests bridge with multiple instances

This approach ensures the CGO bridge is rock solid before adding complexity.

## Usage

```go
// Create source node
sourceNode, err := New()
if err != nil {
    // Handle creation failure
}

// Get pointer for engine operations
nodePtr := sourceNode.GetNodePtr()

// Always cleanup
defer sourceNode.Destroy()
```

## AVAudioSourceNode Characteristics

1. **Render block**: Currently generates silence - ready for audio generation logic
2. **Node type**: AVAudioSourceNode (inherits from AVAudioNode)
3. **Threading**: Render block executes on audio thread (high performance requirement)

## Next Steps

This primitive enables:
- Integration with AVAudioEngine (attach/connect operations)
- Audio generation (sine waves, metronome clicks, etc.)
- Higher-level source abstractions in `macaudio/engine`
