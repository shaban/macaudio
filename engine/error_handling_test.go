package engine

import (
	"testing"
)

func TestErrorHandlingConsistency(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func() error
		wantErr  bool
	}{
		{
			name: "Start with nil engine",
			testFunc: func() error {
				var e Engine
				e.nativeEngine = nil
				return e.Start()
			},
			wantErr: true,
		},
		{
			name: "SetMasterVolume with nil engine",
			testFunc: func() error {
				var e Engine
				e.nativeEngine = nil
				return e.SetMasterVolume(0.5)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got error=%v", tt.wantErr, err)
			}
			if tt.wantErr && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if tt.wantErr && err != nil {
				t.Logf("✅ Correctly caught error: %v", err)
			}
		})
	}
}

func TestSafetyFirstVolumeHandling(t *testing.T) {
	t.Run("Volume safety on failure", func(t *testing.T) {
		// Create engine with nil native engine to simulate failure
		var e Engine
		e.nativeEngine = nil
		e.MasterVolume = 0.8 // Set initial volume

		// Try to set volume - this should fail but preserve safety
		err := e.SetMasterVolume(1.0)
		if err == nil {
			t.Fatal("Expected error when setting volume with nil engine")
		}

		// The cached volume should be set to 0.0 for safety
		if e.MasterVolume != 0.0 {
			t.Errorf("Expected cached volume to be 0.0 for safety, got %f", e.MasterVolume)
		} else {
			t.Logf("✅ Safety-first: Volume cached as 0.0 on failure")
		}
	})
}

func TestVoidMethodSafety(t *testing.T) {
	// Test methods that don't return errors but should handle nil gracefully
	var e Engine
	e.nativeEngine = nil

	t.Run("Stop with nil engine", func(t *testing.T) {
		// Should not panic
		e.Stop()
		t.Log("✅ Stop() handled nil engine gracefully")
	})

	t.Run("Pause with nil engine", func(t *testing.T) {
		// Should not panic
		e.Pause()
		t.Log("✅ Pause() handled nil engine gracefully")
	})

	t.Run("Prepare with nil engine", func(t *testing.T) {
		// Should not panic
		e.Prepare()
		t.Log("✅ Prepare() handled nil engine gracefully")
	})

	t.Run("Reset with nil engine", func(t *testing.T) {
		// Should not panic
		e.Reset()
		t.Log("✅ Reset() handled nil engine gracefully")
	})

	t.Run("Destroy with nil engine", func(t *testing.T) {
		// Should not panic
		e.Destroy()
		t.Log("✅ Destroy() handled nil engine gracefully")
	})
}

func TestNativeErrorMessages(t *testing.T) {
	t.Run("Start error propagation", func(t *testing.T) {
		var e Engine
		e.nativeEngine = nil

		err := e.Start()
		if err == nil {
			t.Fatal("Expected error from Start() with nil engine")
		}

		// The Go layer catches nil engine before calling C, so we expect Go-level error
		expectedErrorSubstrings := []string{"engine", "not", "initialized"}
		errorMsg := err.Error()

		foundExpected := false
		for _, expected := range expectedErrorSubstrings {
			if contains(errorMsg, expected) {
				foundExpected = true
				break
			}
		}

		if !foundExpected {
			t.Errorf("Error message '%s' doesn't contain expected substrings %v", errorMsg, expectedErrorSubstrings)
		} else {
			t.Logf("✅ Proper error message from Go layer: %s", errorMsg)
		}
	})

	t.Run("SetMasterVolume C error propagation", func(t *testing.T) {
		var e Engine
		e.nativeEngine = nil

		err := e.SetMasterVolume(0.5)
		if err == nil {
			t.Fatal("Expected error from SetMasterVolume() with nil engine")
		}

		// This goes to C layer, so we expect C-level error messages
		expectedErrorSubstrings := []string{"wrapper", "null"}
		errorMsg := err.Error()

		foundExpected := false
		for _, expected := range expectedErrorSubstrings {
			if contains(errorMsg, expected) {
				foundExpected = true
				break
			}
		}

		if !foundExpected {
			t.Errorf("Error message '%s' doesn't contain expected substrings %v", errorMsg, expectedErrorSubstrings)
		} else {
			t.Logf("✅ Proper error message from C layer: %s", errorMsg)
		}
	})
}

// Helper function since strings.Contains might not be available
func contains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
