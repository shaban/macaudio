# macaudio/devices - Device Enumeration Library Makefile
# Silent library with configurable JSON logging

.PHONY: test test-audio test-midi test-compatibility test-unified test-all clean help info test-clean

# Default target - run all tests
all: test-all

# Test audio devices specifically
test-audio:
	@echo "🔊 Testing Audio Device Enumeration..."
	go test -v -run TestGetAudio
	@echo "✅ Audio test complete"

# Test MIDI devices specifically  
test-midi:
	@echo "🎹 Testing MIDI Device Enumeration..."
	go test -v -run TestGetMIDI
	@echo "✅ MIDI test complete"

# Test device compatibility methods
test-compatibility:
	@echo "🔗 Testing Device Compatibility Methods..."
	go test -v -run TestAudioDeviceCompatibility
	@echo "✅ Compatibility test complete"

# Test unified device operations
test-unified:
	@echo "🎛️ Testing Unified Device Operations..."
	go test -v -run TestConvenienceFilters
	@echo "✅ Unified test complete"

# Test all device functionality (comprehensive)
test-all:
	@echo "📱 Testing Complete Device Library..."
	go test -v
	@echo "✅ All device tests complete"

# Test with verbose output and specific filters
test-verbose:
	@echo "🔍 Running Verbose Device Tests..."
	go test -v -count=1
	@echo "✅ Verbose test complete"

# Clean build cache
clean:
	@echo "🧹 Cleaning device build cache..."
	go clean -cache -testcache
	@echo "✅ Device clean complete"

# Test with clean build
test-clean: clean test-all

# Show device library info
info:
	@echo "📋 Device Library Information:"
	@echo "  Package: macaudio/devices"
	@echo "  Go version: $(shell go version)"
	@echo "  GOOS: $(shell go env GOOS)"
	@echo "  GOARCH: $(shell go env GOARCH)"
	@echo "  CGO_ENABLED: $(shell go env CGO_ENABLED)"
	@echo "  Library: Silent Audio/MIDI Device Enumeration"
	@echo "  Logging: Configurable via SetJSONLogging(true/false)"
	@echo "  API: devices.GetAudio() and devices.GetMIDI()"

# Run a quick smoke test
smoke:
	@echo "💨 Running Device Library Smoke Test..."
	go test -v -run "TestGetAudio|TestGetMIDI" -short
	@echo "✅ Smoke test complete"

# Help
help:
	@echo "macaudio/devices - Device Library - Available Commands:"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test-audio        - Test audio device enumeration"
	@echo "  make test-midi         - Test MIDI device enumeration"
	@echo "  make test-compatibility - Test device compatibility methods"
	@echo "  make test-unified      - Test unified device operations"
	@echo "  make test-all          - Test all device functionality (default)"
	@echo "  make test-verbose      - Test with extra verbosity"
	@echo "  make test-clean        - Clean build and test all"
	@echo "  make smoke             - Quick smoke test"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  make clean             - Clean build cache"
	@echo "  make info              - Show device library information"
	@echo ""
	@echo "📦 Usage in Go code:"
	@echo "  import \"macaudio/devices\""
	@echo "  devices.SetJSONLogging(true)     // Enable debug output"
	@echo "  audioDevs, err := devices.GetAudio()"
	@echo "  midiDevs, err := devices.GetMIDI()"
	@echo ""
	@echo "🎯 Common workflows:"
	@echo "  cd devices && make test-audio     # Test just audio"
	@echo "  cd devices && make smoke          # Quick validation"
	@echo "  cd devices && make test-all       # Full device test suite"
