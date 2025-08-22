package engine

import (
	"math"
	"testing"
)

// TestValidationUniformity verifies that all validation functions consistently
// reject invalid values and return errors (no clamping behavior)
func TestValidationUniformity(t *testing.T) {
	t.Run("VolumeValidation", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float32
			wantError bool
		}{
			{"ValidMin", 0.0, false},
			{"ValidMid", 0.5, false},
			{"ValidMax", 1.0, false},
			{"InvalidNegative", -0.1, true},
			{"InvalidAboveMax", 1.1, true},
			{"InvalidNaN", float32(math.NaN()), true},
			{"InvalidInfPositive", float32(math.Inf(1)), true},
			{"InvalidInfNegative", float32(math.Inf(-1)), true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateVolume(tt.value)
				if (err != nil) != tt.wantError {
					t.Errorf("ValidateVolume(%v) error = %v, wantError = %v", tt.value, err, tt.wantError)
				}
			})
		}
	})

	t.Run("PanValidation", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float32
			wantError bool
		}{
			{"ValidMin", -1.0, false},
			{"ValidCenter", 0.0, false},
			{"ValidMax", 1.0, false},
			{"ValidLeft", -0.5, false},
			{"ValidRight", 0.5, false},
			{"InvalidBelowMin", -1.1, true},
			{"InvalidAboveMax", 1.1, true},
			{"InvalidNaN", float32(math.NaN()), true},
			{"InvalidInfPositive", float32(math.Inf(1)), true},
			{"InvalidInfNegative", float32(math.Inf(-1)), true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidatePan(tt.value)
				if (err != nil) != tt.wantError {
					t.Errorf("ValidatePan(%v) error = %v, wantError = %v", tt.value, err, tt.wantError)
				}
			})
		}
	})

	t.Run("RateValidation", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float32
			wantError bool
		}{
			{"ValidMin", 0.25, false},
			{"ValidNormal", 1.0, false},
			{"ValidMax", 1.25, false},
			{"ValidMid", 0.75, false},
			{"InvalidZero", 0.0, true},
			{"InvalidNegative", -0.5, true},
			{"InvalidTooSlow", 0.1, true},
			{"InvalidTooFast", 2.0, true},
			{"InvalidNaN", float32(math.NaN()), true},
			{"InvalidInfPositive", float32(math.Inf(1)), true},
			{"InvalidInfNegative", float32(math.Inf(-1)), true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateRate(tt.value)
				if (err != nil) != tt.wantError {
					t.Errorf("ValidateRate(%v) error = %v, wantError = %v", tt.value, err, tt.wantError)
				}
			})
		}
	})

	t.Run("PitchValidation", func(t *testing.T) {
		tests := []struct {
			name      string
			value     float32
			wantError bool
		}{
			{"ValidMin", -12.0, false},
			{"ValidCenter", 0.0, false},
			{"ValidMax", 12.0, false},
			{"ValidDown", -6.0, false},
			{"ValidUp", 6.0, false},
			{"InvalidBelowMin", -12.1, true},
			{"InvalidAboveMax", 12.1, true},
			{"InvalidWayDown", -24.0, true},
			{"InvalidWayUp", 24.0, true},
			{"InvalidNaN", float32(math.NaN()), true},
			{"InvalidInfPositive", float32(math.Inf(1)), true},
			{"InvalidInfNegative", float32(math.Inf(-1)), true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidatePitch(tt.value)
				if (err != nil) != tt.wantError {
					t.Errorf("ValidatePitch(%v) error = %v, wantError = %v", tt.value, err, tt.wantError)
				}
			})
		}
	})

	t.Run("FilePathValidation", func(t *testing.T) {
		tests := []struct {
			name      string
			value     string
			wantError bool
		}{
			{"ValidPath", "/path/to/file.wav", false},
			{"ValidSystemPath", "/System/Library/Sounds/Ping.aiff", false},
			{"InvalidEmpty", "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateFilePath(tt.value)
				if (err != nil) != tt.wantError {
					t.Errorf("ValidateFilePath(%q) error = %v, wantError = %v", tt.value, err, tt.wantError)
				}
			})
		}
	})
}

// TestValidationConsistency verifies that validation is applied consistently
// across all parameter setting methods
func TestValidationConsistency(t *testing.T) {
	config := DefaultTestEngineConfig()
	engine, cleanup := CreateTestEngine(t, config)
	defer cleanup()

	// Create a playback channel for testing
	channel, err := engine.CreatePlaybackChannel("/System/Library/Sounds/Ping.aiff")
	if err != nil {
		t.Fatalf("Failed to create playback channel: %v", err)
	}

	t.Run("VolumeConsistency", func(t *testing.T) {
		// All volume setting methods should reject the same invalid values
		invalidVolume := float32(1.5)

		// Test engine master volume
		if err := engine.SetMasterVolume(invalidVolume); err == nil {
			t.Error("SetMasterVolume should reject invalid volume")
		}

		// Test channel volume
		if err := channel.SetVolume(invalidVolume); err == nil {
			t.Error("SetVolume should reject invalid volume")
		}
	})

	t.Run("PanConsistency", func(t *testing.T) {
		// All pan setting methods should reject the same invalid values
		invalidPan := float32(1.5)

		// Test channel pan
		if err := channel.SetPan(invalidPan); err == nil {
			t.Error("SetPan should reject invalid pan")
		}
	})

	t.Run("PlaybackParameterConsistency", func(t *testing.T) {
		// All playback parameter methods should reject the same invalid values
		invalidRate := float32(2.0)
		invalidPitch := float32(24.0)

		// Test playback rate
		if err := channel.SetPlaybackRate(invalidRate); err == nil {
			t.Error("SetPlaybackRate should reject invalid rate")
		}

		// Test pitch
		if err := channel.SetPitch(invalidPitch); err == nil {
			t.Error("SetPitch should reject invalid pitch")
		}
	})
}
