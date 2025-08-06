// main.m - Test just Neural DSP with each version

#import "./plugins_async.m"

int main() {
    @autoreleasepool {
        printf("Testing Neural DSP with version 1...\n");
        char *result1 = IntrospectAudioUnitsWithTimeout("aumf", "NMAS", "NDSP");
        // Check for indexed values in result1
        printf("Result 1: %s\n", result1);
    }
}