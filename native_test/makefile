clean:
	rm -f *.o test_neural
current:
	clang -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit -o test_neural main_current.m
async:
	clang -x objective-c -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit -fobjc-arc -o test_neural main_async.m
original:
	clang -x objective-c -framework Foundation -framework AudioToolbox -framework AVFoundation -framework AudioUnit -fobjc-arc -o test_neural main_original.m
test-current:
	make clean
	make current
	@echo "Testing current version..." >&2
	./test_neural > current.json
	@echo "Current version test complete. Results saved to current.json" >&2
test-async:
	make clean
	make async
	@echo "Testing async version..." >&2
	./test_neural > async.json
	@echo "Async version test complete. Results saved to async.json" >&2
test-original:
	make clean
	make original
	@echo "Testing original version..." >&2
	./test_neural > original.json
	@echo "Original version test complete. Results saved to original.json" >&2
