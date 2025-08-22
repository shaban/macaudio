package engine

import (
	"testing"
)

// TestErrorHandlingWithHelpers demonstrates using helper for consistent error testing
func TestErrorHandlingWithHelpers(t *testing.T) {
	tests := []ErrorTestCase{
		{
			Name: "Start with nil engine",
			TestFunc: func() error {
				var e Engine
				e.nativeEngine = nil
				return e.Start()
			},
			WantErr: true,
		},
		{
			Name: "SetMasterVolume with nil engine",
			TestFunc: func() error {
				var e Engine
				e.nativeEngine = nil
				return e.SetMasterVolume(0.5)
			},
			WantErr: true,
		},
		{
			Name: "GetMasterVolume with nil engine",
			TestFunc: func() error {
				var e Engine
				e.nativeEngine = nil
				// This doesn't return an error, but we're testing the pattern
				_ = e.GetMasterVolume() // Should return 0.0 safely
				return nil
			},
			WantErr: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			ValidateErrorTestCase(t, testCase)
		})
	}
}

// TestEngineVolumeHandling tests volume operations with expectations
func TestEngineVolumeHandling(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	tests := []struct {
		name         string
		setVolume    float32
		expectVolume float32
		expectError  bool
	}{
		{
			name:         "NormalVolume",
			setVolume:    0.5,
			expectVolume: 0.5,
			expectError:  false,
		},
		{
			name:         "MinVolume",
			setVolume:    0.0,
			expectVolume: 0.0,
			expectError:  false,
		},
		{
			name:         "MaxVolume",
			setVolume:    1.0,
			expectVolume: 1.0,
			expectError:  false,
		},
		{
			name:         "NegativeVolume", // Should reject negative values
			setVolume:    -0.1,
			expectVolume: 0.0,  // Not used for error cases
			expectError:  true, // Should return an error
		},
		{
			name:         "VolumeAboveMax", // Should reject values above 1.0
			setVolume:    1.5,
			expectVolume: 0.0,  // Not used for error cases
			expectError:  true, // Should return an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the volume before the test to check it remains unchanged on error
			volumeBefore := engine.GetMasterVolume()

			// Test setting volume
			err := engine.SetMasterVolume(tt.setVolume)
			if (err != nil) != tt.expectError {
				t.Errorf("SetMasterVolume(%v) error = %v, wantErr %v", tt.setVolume, err, tt.expectError)
			}

			if !tt.expectError {
				// Test getting volume - should be the new value
				actualVolume := engine.GetMasterVolume()
				if actualVolume != tt.expectVolume {
					t.Errorf("GetMasterVolume() = %v, want %v", actualVolume, tt.expectVolume)
				}
			} else {
				// Test getting volume - should be unchanged from before the failed operation
				actualVolume := engine.GetMasterVolume()
				if actualVolume != volumeBefore {
					t.Errorf("GetMasterVolume() after error = %v, want %v (unchanged)", actualVolume, volumeBefore)
				}
			}
		})
	}
}
