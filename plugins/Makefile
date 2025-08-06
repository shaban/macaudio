# Test AU plugin introspection - both native and Go wrapper
# Plugin introspection with complete Go CGO integration

.PHONY: test-native test-go test-all clean help

clean:
	@echo "🧹 Cleaning all build artifacts..."
	rm -f test_native test_static test_obj test_go_wrapper
	rm -f native/*.o native/*.a
	rm -f *.o *.a
	go clean -cache
	go clean -testcache
	@echo "✅ Clean complete"

# Native-only targets (don't mix with CGO)
lib-native: clean
	@echo "🔧 Building static library (native only)..."
	gcc -c native/plugins.m -o native/plugins.o \
		-framework Foundation \
		-framework AudioToolbox \
		-framework AVFoundation \
		-framework AudioUnit \
		-ObjC
	ar rcs native/libplugins.a native/plugins.o

# Test the Go CGO wrapper
test-go: clean
	@echo "🧪 Testing Go CGO wrapper for AU plugins..."
	go test -v
	@echo "✅ Go wrapper test complete"

# Test both native and Go wrapper
test-all: clean test-go
	@echo "✅ All plugin tests complete"

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
