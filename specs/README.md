# MacAudio Specifications

This directory contains technical specifications for the macaudio library.

## Current Specifications

### [`TESTING.md`](TESTING.md)
**Testing Guidelines** - Comprehensive testing strategy for mixed Go/AVFoundation code. Covers:
- Test tiers: unit, integration, and manual audible tests
- CI-friendly testing practices avoiding real-time dependencies
- Testing conventions and utility functions
- Hardware assumption management

## Removed Specifications

The specifications directory has been significantly streamlined. Removed documents included:
- `ARCHITECTURE.md` - Complex UUID-based architecture (replaced by simpler approach)
- `IMPLEMENTATION.md` - Detailed implementation specs for obsolete features  
- `TECHNICAL_ANALYSIS.md` - Analysis of approaches that were ultimately not used
- `SPECIFICATION_PROCESS_ANALYSIS.md` - Process documentation for removed features
- `ADAPTIVE_POLLING.md` - Device polling optimization (not implemented)
- `DOCUMENTATION.md` - Documentation requirements for removed features

The current implementation focuses on a much simpler sampler-based architecture documented in [`docs/MVP.md`](../docs/MVP.md).
