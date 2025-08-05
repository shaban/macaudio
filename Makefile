# macaudio - macOS Audio/MIDI Library Makefile
# Root makefile for the complete macaudio library

.PHONY: test test-devices clean help info test-clean

# Default target - run comprehensive device tests
all: test-devices

# Test device library (comprehensive test of all device functionality)
test-devices:
	@echo "📱 Testing Complete Device Library Package..."
	go test -v ./devices
	@echo "✅ Device library tests complete"

# Clean build cache
clean:
	@echo "🧹 Cleaning build cache..."
	go clean -cache -testcache
	@echo "✅ Clean complete"

# Test with clean build
test-clean: clean test-devices

# Show library info
info:
	@echo "📋 macaudio Library Information:"
	@echo "  Go version: $(shell go version)"
	@echo "  GOOS: $(shell go env GOOS)"
	@echo "  GOARCH: $(shell go env GOARCH)"
	@echo "  CGO_ENABLED: $(shell go env CGO_ENABLED)"
	@echo "  Library: macOS Audio/MIDI Device Enumeration"
	@echo "  Main Package: macaudio/devices"
	@echo "  API: devices.GetAudio() and devices.GetMIDI()"

# Help
help:
	@echo "macaudio - macOS Audio/MIDI Library - Available Commands:"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test-devices  - Test complete device library (default)"
	@echo "  make test-clean    - Clean build and test devices"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  make clean         - Clean build cache"
	@echo "  make info          - Show library information"
	@echo ""
	@echo "📦 Package-specific testing:"
	@echo "  cd devices && make help    # See device-specific test options"
	@echo "  cd devices && make smoke   # Quick device validation"
	@echo ""
	@echo "📦 Usage in Go code:"
	@echo "  import \"macaudio/devices\""
	@echo "  devices.SetJSONLogging(true)     // Enable debug output"
	@echo "  audioDevs, err := devices.GetAudio()"
	@echo "  midiDevs, err := devices.GetMIDI()"
