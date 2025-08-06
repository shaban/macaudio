// main.m - Test just Neural DSP with each version

#import "./plugins_current.m"

int main() {
    @autoreleasepool {
        char *result1 = IntrospectAudioUnits("aumf", "NMAS", "NDSP");
        if (result1) {
            printf("%s", result1);  // Only output the JSON
            free(result1);
        }
        return 0;
    }
}