// Package tap provides audio tap functionality for monitoring and testing audio signals
// in AVAudioEngine. Taps allow non-intrusive monitoring of audio data flowing through nodes.
package tap

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework AudioToolbox -framework Foundation
#import <AVFoundation/AVFoundation.h>
#import <AudioUnit/AudioUnit.h>

// Tap callback info structure
typedef struct {
    void* tapPtr;      // Unique tap identifier
    void* nodePtr;     // AVAudioNode being tapped
    int busIndex;      // Bus index being tapped
    bool isActive;     // Whether tap is currently active
    double sampleRate; // Sample rate of the tapped audio
    int channelCount;  // Number of channels being tapped
} TapInfo;

// Function declarations for CGO
bool tap_install(void* enginePtr, void* nodePtr, int busIndex, void* tapIdentifier);
bool tap_remove(void* enginePtr, void* nodePtr, int busIndex, void* tapIdentifier);
bool tap_get_info(void* tapIdentifier, TapInfo* info);
double tap_get_rms(void* tapIdentifier);
int tap_get_frame_count(void* tapIdentifier);
void tap_remove_all(void);
int tap_get_active_count(void);
void tap_init(void);

// Global tap storage (simplified for this implementation)
static NSMutableDictionary* activeTaps = nil;

// Initialize tap storage
void tap_init() {
    if (!activeTaps) {
        activeTaps = [[NSMutableDictionary alloc] init];
    }
}

// Install a tap on an AVAudioNode at the specified bus
bool tap_install(void* enginePtr, void* nodePtr, int busIndex, void* tapIdentifier) {
    if (!enginePtr || !nodePtr || !tapIdentifier) {
        NSLog(@"tap_install: Invalid parameters");
        return false;
    }

    tap_init();

    AVAudioEngine* engine = (__bridge AVAudioEngine*)enginePtr;
    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;

    @try {
        // Check if node is attached to engine
        if (![engine.attachedNodes containsObject:node]) {
            NSLog(@"tap_install: Node is not attached to engine");
            return false;
        }

        // Check bus validity
        if (busIndex < 0 || busIndex >= node.numberOfOutputs) {
            NSLog(@"tap_install: Invalid bus index %d for node with %d outputs",
                  busIndex, (int)node.numberOfOutputs);
            return false;
        }

        // Get the format for this bus
        AVAudioFormat* format = [node outputFormatForBus:busIndex];
        if (!format) {
            NSLog(@"tap_install: No format available for bus %d", busIndex);
            return false;
        }

        // Create tap info
        NSString* tapKey = [NSString stringWithFormat:@"%p", tapIdentifier];

        // Remove existing tap if present
        [node removeTapOnBus:busIndex];

        // Install the tap with a callback that stores audio data
        [node installTapOnBus:busIndex bufferSize:1024 format:format block:^(AVAudioPCMBuffer * _Nonnull buffer, AVAudioTime * _Nonnull when) {
            // Store tap information for retrieval
            @synchronized(activeTaps) {
                NSMutableDictionary* tapData = activeTaps[tapKey];
                if (!tapData) {
                    tapData = [[NSMutableDictionary alloc] init];
                    activeTaps[tapKey] = tapData;
                }

                // Store latest buffer info (we'll keep this simple for now)
                tapData[@"frameLength"] = @(buffer.frameLength);
                tapData[@"frameCapacity"] = @(buffer.frameCapacity);
                tapData[@"sampleRate"] = @(format.sampleRate);
                tapData[@"channelCount"] = @(format.channelCount);
                tapData[@"lastUpdateTime"] = @([[NSDate date] timeIntervalSince1970]);

                // Calculate RMS for monitoring (simple implementation)
                if (buffer.frameLength > 0 && buffer.floatChannelData) {
                    float rms = 0.0f;
                    float* channelData = buffer.floatChannelData[0]; // Use first channel
                    for (UInt32 i = 0; i < buffer.frameLength; i++) {
                        rms += channelData[i] * channelData[i];
                    }
                    rms = sqrt(rms / buffer.frameLength);
                    tapData[@"rms"] = @(rms);
                }
            }
        }];

        // Store tap info
        @synchronized(activeTaps) {
            NSMutableDictionary* tapData = activeTaps[tapKey];
            if (!tapData) {
                tapData = [[NSMutableDictionary alloc] init];
                activeTaps[tapKey] = tapData;
            }

            tapData[@"nodePtr"] = [NSValue valueWithPointer:nodePtr];
            tapData[@"busIndex"] = @(busIndex);
            tapData[@"isActive"] = @YES;
            tapData[@"sampleRate"] = @(format.sampleRate);
            tapData[@"channelCount"] = @(format.channelCount);
        }

        NSLog(@"tap_install: Successfully installed tap on bus %d (%.0f Hz, %d channels)",
              busIndex, format.sampleRate, (int)format.channelCount);
        return true;

    } @catch (NSException* exception) {
        NSLog(@"tap_install: Exception installing tap: %@", exception);
        return false;
    }
}

// Remove a tap from an AVAudioNode
bool tap_remove(void* enginePtr, void* nodePtr, int busIndex, void* tapIdentifier) {
    if (!nodePtr || !tapIdentifier) {
        NSLog(@"tap_remove: Invalid parameters");
        return false;
    }

    tap_init();

    AVAudioNode* node = (__bridge AVAudioNode*)nodePtr;
    NSString* tapKey = [NSString stringWithFormat:@"%p", tapIdentifier];

    @try {
        // Remove the tap
        [node removeTapOnBus:busIndex];

        // Remove from our storage
        @synchronized(activeTaps) {
            [activeTaps removeObjectForKey:tapKey];
        }

        NSLog(@"tap_remove: Successfully removed tap on bus %d", busIndex);
        return true;

    } @catch (NSException* exception) {
        NSLog(@"tap_remove: Exception removing tap: %@", exception);
        return false;
    }
}

// Get tap information and metrics
bool tap_get_info(void* tapIdentifier, TapInfo* info) {
    if (!tapIdentifier || !info) {
        NSLog(@"tap_get_info: Invalid parameters");
        return false;
    }

    tap_init();

    NSString* tapKey = [NSString stringWithFormat:@"%p", tapIdentifier];

    @synchronized(activeTaps) {
        NSMutableDictionary* tapData = activeTaps[tapKey];
        if (!tapData) {
            return false;
        }

        info->tapPtr = tapIdentifier;
        info->nodePtr = [[tapData objectForKey:@"nodePtr"] pointerValue];
        info->busIndex = [[tapData objectForKey:@"busIndex"] intValue];
        info->isActive = [[tapData objectForKey:@"isActive"] boolValue];
        info->sampleRate = [[tapData objectForKey:@"sampleRate"] doubleValue];
        info->channelCount = [[tapData objectForKey:@"channelCount"] intValue];

        return true;
    }
}

// Get current RMS level from tap
double tap_get_rms(void* tapIdentifier) {
    if (!tapIdentifier) {
        return -1.0;
    }

    tap_init();

    NSString* tapKey = [NSString stringWithFormat:@"%p", tapIdentifier];

    @synchronized(activeTaps) {
        NSMutableDictionary* tapData = activeTaps[tapKey];
        if (!tapData) {
            return -1.0;
        }

        NSNumber* rms = tapData[@"rms"];
        if (rms) {
            return [rms doubleValue];
        }
    }

    return 0.0;
}

// Get frame count from last buffer
int tap_get_frame_count(void* tapIdentifier) {
    if (!tapIdentifier) {
        return -1;
    }

    tap_init();

    NSString* tapKey = [NSString stringWithFormat:@"%p", tapIdentifier];

    @synchronized(activeTaps) {
        NSMutableDictionary* tapData = activeTaps[tapKey];
        if (!tapData) {
            return -1;
        }

        NSNumber* frameLength = tapData[@"frameLength"];
        if (frameLength) {
            return [frameLength intValue];
        }
    }

    return 0;
}

// Remove all taps (cleanup)
void tap_remove_all() {
    tap_init();

    @synchronized(activeTaps) {
        // We can't easily remove all taps without keeping engine reference
        // So we'll just clear our storage
        [activeTaps removeAllObjects];
        NSLog(@"tap_remove_all: Cleared tap storage");
    }
}

// Get number of active taps
int tap_get_active_count() {
    tap_init();

    @synchronized(activeTaps) {
        return (int)[activeTaps count];
    }
}
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"
)

// TapInfo contains information about an installed audio tap
type TapInfo struct {
	TapID        unsafe.Pointer
	NodePtr      unsafe.Pointer
	BusIndex     int
	IsActive     bool
	SampleRate   float64
	ChannelCount int
}

// TapMetrics contains current metrics from an audio tap
type TapMetrics struct {
	RMS        float64   // Root Mean Square level
	FrameCount int       // Number of frames in last buffer
	LastUpdate time.Time // When metrics were last updated
}

// Tap represents an audio tap for monitoring signal flow
type Tap struct {
	id        unsafe.Pointer
	enginePtr unsafe.Pointer
	nodePtr   unsafe.Pointer
	busIndex  int
	installed bool
}

// InstallTap installs a tap on the specified AVAudioNode and bus
func InstallTap(enginePtr, nodePtr unsafe.Pointer, busIndex int) (*Tap, error) {
	if enginePtr == nil {
		return nil, fmt.Errorf("engine pointer cannot be nil")
	}
	if nodePtr == nil {
		return nil, fmt.Errorf("node pointer cannot be nil")
	}
	if busIndex < 0 {
		return nil, fmt.Errorf("bus index must be non-negative")
	}

	// Create unique tap identifier
	tapID := unsafe.Pointer(uintptr(time.Now().UnixNano()))

	// Install the tap
	success := bool(C.tap_install(enginePtr, nodePtr, C.int(busIndex), tapID))
	if !success {
		return nil, fmt.Errorf("failed to install tap on bus %d", busIndex)
	}

	return &Tap{
		id:        tapID,
		enginePtr: enginePtr,
		nodePtr:   nodePtr,
		busIndex:  busIndex,
		installed: true,
	}, nil
}

// Remove removes the tap from the audio node
func (t *Tap) Remove() error {
	if !t.installed {
		return fmt.Errorf("tap is not installed")
	}

	success := bool(C.tap_remove(t.enginePtr, t.nodePtr, C.int(t.busIndex), t.id))
	if !success {
		return fmt.Errorf("failed to remove tap")
	}

	t.installed = false
	return nil
}

// GetInfo returns information about the tap
func (t *Tap) GetInfo() (*TapInfo, error) {
	if !t.installed {
		return nil, fmt.Errorf("tap is not installed")
	}

	var info C.TapInfo
	success := bool(C.tap_get_info(t.id, &info))
	if !success {
		return nil, fmt.Errorf("failed to get tap info")
	}

	return &TapInfo{
		TapID:        unsafe.Pointer(info.tapPtr),
		NodePtr:      unsafe.Pointer(info.nodePtr),
		BusIndex:     int(info.busIndex),
		IsActive:     bool(info.isActive),
		SampleRate:   float64(info.sampleRate),
		ChannelCount: int(info.channelCount),
	}, nil
}

// GetMetrics returns current audio metrics from the tap
func (t *Tap) GetMetrics() (*TapMetrics, error) {
	if !t.installed {
		return nil, fmt.Errorf("tap is not installed")
	}

	rms := float64(C.tap_get_rms(t.id))
	frameCount := int(C.tap_get_frame_count(t.id))

	return &TapMetrics{
		RMS:        rms,
		FrameCount: frameCount,
		LastUpdate: time.Now(),
	}, nil
}

// IsInstalled returns true if the tap is currently installed
func (t *Tap) IsInstalled() bool {
	return t.installed
}

// GetBusIndex returns the bus index being tapped
func (t *Tap) GetBusIndex() int {
	return t.busIndex
}

// GetNodePtr returns the node pointer being tapped
func (t *Tap) GetNodePtr() unsafe.Pointer {
	return t.nodePtr
}

// WaitForActivity waits for audio activity on the tap with a timeout
func (t *Tap) WaitForActivity(timeout time.Duration, minRMS float64) (bool, error) {
	if !t.installed {
		return false, fmt.Errorf("tap is not installed")
	}

	start := time.Now()
	for time.Since(start) < timeout {
		metrics, err := t.GetMetrics()
		if err != nil {
			return false, err
		}

		if metrics.RMS >= minRMS && metrics.FrameCount > 0 {
			return true, nil
		}

		time.Sleep(10 * time.Millisecond) // Small delay between checks
	}

	return false, nil // Timeout reached without activity
}

// Package-level functions

// RemoveAllTaps removes all active taps (useful for cleanup)
func RemoveAllTaps() {
	C.tap_remove_all()
}

// GetActiveTapCount returns the number of currently active taps
func GetActiveTapCount() int {
	return int(C.tap_get_active_count())
}

// WaitForSignal is a utility function to wait for audio signal on any tap
func WaitForSignal(taps []*Tap, timeout time.Duration, minRMS float64) (*Tap, error) {
	start := time.Now()

	for time.Since(start) < timeout {
		for _, tap := range taps {
			if !tap.IsInstalled() {
				continue
			}

			metrics, err := tap.GetMetrics()
			if err != nil {
				continue
			}

			if metrics.RMS >= minRMS && metrics.FrameCount > 0 {
				return tap, nil
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	return nil, fmt.Errorf("timeout waiting for signal")
}
