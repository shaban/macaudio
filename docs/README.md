# MacAudio Design Documents

This directory contains design documents and technical references for the macaudio library.

## Current Documents

### [`MVP.md`](MVP.md)
**Primary Design Document** - Current architecture overview focusing on sampler-based sound generation. Documents the working implementation with:
- Sampler channel architecture using `AVAudioUnitSampler`
- Standard Go CGO practices and build system
- Dynamic bus allocation and channel management
- API usage examples and implementation status

### [`BEST_PRACTICES.md`](BEST_PRACTICES.md)
**Technical Reference** - Hard-learned lessons about AVAudioEngine quirks and requirements. Contains:
- Engine restart requirements for various operations
- Buffer management and memory handling
- Threading considerations and timing issues
- Debugging tips and common pitfalls

### [`SAMPLER_REFERENCE.md`](SAMPLER_REFERENCE.md)  
**Implementation Guide** - Documents the Swift reference implementation that inspired our Go/C sampler approach. Shows:
- Working Swift code that produces sound
- Architectural comparison: direct sampler vs MIDI routing
- Key insights about `AVAudioUnitSampler` usage

### [`TIMEPITCH_FINDINGS_AND_ARCHITECTURE.md`](TIMEPITCH_FINDINGS_AND_ARCHITECTURE.md)
**Technical Analysis** - Detailed findings about AVAudioUnitTimePitch behavior and performance characteristics. Contains:
- Buffer-level timing measurements 
- Rate vs pitch processing differences
- Performance optimization guidelines

## Note on Outdated Documents

This directory has been cleaned up to remove outdated design documents that referred to complex MIDI routing and UUID-based architectures that were replaced with the current simpler sampler-based approach.

The remaining documents focus on the working implementation and useful technical knowledge about AVAudioEngine behavior.
