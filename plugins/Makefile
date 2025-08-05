# Test AU plugin introspection - both native and Go wrapper
# Plugin introspection with complete Go CGO integration

.PHONY: test-native test-go test-all clean help

# Test the native plugin introspection only
test-native:
	@echo "🔧 Building native AU plugin test..."
	gcc -o test_native native/plugins.m \
		-framework Foundation \
		-framework AudioToolbox \
		-framework AVFoundation \
		-framework AudioUnit \
		-ObjC
	@echo "🚀 Running native AU plugin introspection..."
	./test_native

# Test the Go CGO wrapper
test-go:
	@echo "🧪 Testing Go CGO wrapper for AU plugins..."
	go test -v
	@echo "✅ Go wrapper test complete"

# Test both native and Go wrapper
test-all: test-go
	@echo "✅ All plugin tests complete"

# Clean up
clean:
	@echo "🧹 Cleaning test files..."
	rm -f test_native test_go_wrapper
	@echo "✅ Clean complete"

# Help
help:
	@echo "Plugin Testing - Available Commands:"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test-native   - Test native AU introspection only"
	@echo "  make test-go       - Test Go CGO wrapper"
	@echo "  make test-all      - Test both native and Go (recommended)"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  make clean         - Clean test files"
