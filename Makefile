# macaudio - macOS Audio/MIDI Device Library Makefile
# Silent library with configurable JSON logging

.PHONY: test test-audio test-midi test-all clean help info test-clean

# Default target - run all tests
all: test-all

# Test audio devices with JSON logging
test-audio:
	@echo "🔊 Testing Audio Device Library..."
	go test -v ./devices -run TestGetAudioDevices
	@echo "✅ Audio test complete"

# Test MIDI devices with JSON logging
test-midi:
	@echo "🎹 Testing MIDI Device Library..."
	go test -v ./devices -run TestGetAllMIDIDevices
	@echo "✅ MIDI test complete"

# Test all devices (comprehensive)
test-all:
	@echo "📱 Testing Complete Audio/MIDI Device Library..."
	go test -v ./devices
	@echo "✅ All tests complete"

# Clean build cache
clean:
	@echo "🧹 Cleaning build cache..."
	go clean -cache -testcache
	@echo "✅ Clean complete"

# Test with clean build
test-clean: clean test-all

# Show library info
info:
	@echo "📋 Library Information:"
	@echo "  Go version: $(shell go version)"
	@echo "  GOOS: $(shell go env GOOS)"
	@echo "  GOARCH: $(shell go env GOARCH)"
	@echo "  CGO_ENABLED: $(shell go env CGO_ENABLED)"
	@echo "  Library: Silent Audio/MIDI Device Enumeration"
	@echo "  Logging: Configurable via SetJSONLogging(true/false)"

# Help
help:
	@echo "macaudio - macOS Audio/MIDI Device Library - Available Commands:"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test-audio   - Test audio device enumeration"
	@echo "  make test-midi    - Test MIDI device enumeration"
	@echo "  make test-all     - Test both audio and MIDI (default)"
	@echo "  make test-clean   - Clean build and test all"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  make clean        - Clean build cache"
	@echo "  make info         - Show library information"
	@echo ""
	@echo "📦 Usage in code:"
	@echo "  import \"macaudio/devices\""
	@echo "  devices.SetJSONLogging(true)     // Enable debug output"
	@echo "  audioDevs, err := devices.GetAllAudioDevices()"
	@echo "  midiDevs, err := devices.GetAllMIDIDevices()"
