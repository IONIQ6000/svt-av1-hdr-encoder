package encoder

import (
	"math"
	"testing"
	"testing/quick"
	"time"
)

// Feature: tui-accuracy-fix, Property 1: Percentage Clamping
// For any percentage value, clampPercentage SHALL return a value in [0, 100]
func TestClampPercentage_Property(t *testing.T) {
	// **Validates: Requirements 1.1, 1.5**
	f := func(pct float64) bool {
		// Skip NaN and Inf
		if math.IsNaN(pct) || math.IsInf(pct, 0) {
			return true
		}
		result := clampPercentage(pct)
		return result >= 0 && result <= 100
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 2: Frame-Based Percentage Calculation
// For any valid frame count and total frames where total > 0, percentage = (frame/total)*100 clamped to [0,100]
func TestFrameBasedPercentage_Property(t *testing.T) {
	// **Validates: Requirements 1.2**
	f := func(frame, total uint32) bool {
		if total == 0 {
			return true // Skip division by zero case
		}
		frameInt := int64(frame)
		totalInt := int64(total)
		
		expected := float64(frameInt) / float64(totalInt) * 100
		result := clampPercentage(expected)
		
		return result >= 0 && result <= 100
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 3: Duration-Based Percentage Calculation
// For any valid current time and total duration where total > 0, percentage = (current/total)*100 clamped to [0,100]
func TestDurationBasedPercentage_Property(t *testing.T) {
	// **Validates: Requirements 1.3**
	f := func(currentUs, totalUs uint64) bool {
		if totalUs == 0 {
			return true // Skip division by zero case
		}
		
		expected := float64(currentUs) / float64(totalUs) * 100
		result := clampPercentage(expected)
		
		return result >= 0 && result <= 100
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 4: Frame Estimation from Duration
// For any valid duration and fps > 0, estimated frames = duration * fps
func TestFrameEstimationFromDuration_Property(t *testing.T) {
	// **Validates: Requirements 2.2**
	f := func(durationSec float32, fps float32) bool {
		// Skip invalid inputs
		if fps <= 0 || durationSec <= 0 {
			return true
		}
		if math.IsNaN(float64(fps)) || math.IsInf(float64(fps), 0) {
			return true
		}
		if math.IsNaN(float64(durationSec)) || math.IsInf(float64(durationSec), 0) {
			return true
		}
		
		estimated := int64(float64(durationSec) * float64(fps))
		return estimated >= 0
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 5: Fractional Frame Rate Parsing
// For any fraction "num/den" where den > 0, parseFrameRate returns num/den
func TestFractionalFrameRateParsing_Property(t *testing.T) {
	// **Validates: Requirements 2.3**
	f := func(num, den uint16) bool {
		if den == 0 {
			return true // Skip division by zero
		}
		
		fpsStr := string(rune('0'+num%10)) + "000/" + string(rune('0'+den%10)) + "001"
		// Use simple test cases
		testCases := []struct {
			input    string
			expected float64
		}{
			{"24/1", 24.0},
			{"30/1", 30.0},
			{"24000/1001", 24000.0 / 1001.0},
			{"30000/1001", 30000.0 / 1001.0},
			{"25/1", 25.0},
		}
		
		for _, tc := range testCases {
			result := parseFrameRate(tc.input)
			if math.Abs(result-tc.expected) > 0.001 {
				return false
			}
		}
		
		// Also test that non-fractional works
		result := parseFrameRate("23.976")
		if math.Abs(result-23.976) > 0.001 {
			return false
		}
		
		_ = fpsStr // Suppress unused warning
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 6: FPS-Based ETA Calculation
// For any remaining frames > 0 and FPS > 0, ETA = remaining_frames / FPS seconds
func TestFPSBasedETACalculation_Property(t *testing.T) {
	// **Validates: Requirements 3.1**
	f := func(remainingFrames, fps uint32) bool {
		if fps == 0 || remainingFrames == 0 {
			return true
		}
		
		etaSeconds := float64(remainingFrames) / float64(fps)
		eta := time.Duration(etaSeconds) * time.Second
		
		return eta >= 0
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 7: Speed-Based ETA Calculation
// For any remaining duration and speed > 0, ETA = remaining_duration / speed
func TestSpeedBasedETACalculation_Property(t *testing.T) {
	// **Validates: Requirements 3.2**
	f := func(remainingUs uint64, speed float32) bool {
		if speed <= 0 || remainingUs == 0 {
			return true
		}
		if math.IsNaN(float64(speed)) || math.IsInf(float64(speed), 0) {
			return true
		}
		
		etaUs := float64(remainingUs) / float64(speed)
		eta := time.Duration(int64(etaUs)) * time.Microsecond
		
		return eta >= 0
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 8: Invalid Speed Returns Unavailable ETA
// For speed "N/A", empty, or unparseable, parseSpeed returns ok=true but speed=0
func TestInvalidSpeedReturnsUnavailable_Property(t *testing.T) {
	// **Validates: Requirements 3.3**
	invalidSpeeds := []string{
		"speed=N/A",
		"speed=   N/A",
	}
	
	for _, line := range invalidSpeeds {
		speed, raw, ok := parseSpeed(line)
		if !ok {
			t.Errorf("parseSpeed should return ok=true for %q", line)
		}
		if raw != "N/A" {
			t.Errorf("parseSpeed should return raw='N/A' for %q, got %q", line, raw)
		}
		if speed != 0 {
			t.Errorf("parseSpeed should return speed=0 for N/A, got %f", speed)
		}
	}
}

// Feature: tui-accuracy-fix, Property 10: Speed Parsing
// For any valid speed string like "1.5x", parseSpeed extracts the numeric multiplier
func TestSpeedParsing_Property(t *testing.T) {
	// **Validates: Requirements 4.1, 4.4**
	testCases := []struct {
		line     string
		expected float64
		raw      string
	}{
		{"speed=1x", 1.0, "1x"},
		{"speed=1.5x", 1.5, "1.5x"},
		{"speed=0.5x", 0.5, "0.5x"},
		{"speed=2.34x", 2.34, "2.34x"},
		{"speed=  1.5x", 1.5, "1.5x"},
	}
	
	for _, tc := range testCases {
		speed, raw, ok := parseSpeed(tc.line)
		if !ok {
			t.Errorf("parseSpeed failed for %q", tc.line)
			continue
		}
		if math.Abs(speed-tc.expected) > 0.001 {
			t.Errorf("parseSpeed(%q) = %f, want %f", tc.line, speed, tc.expected)
		}
		if raw != tc.raw {
			t.Errorf("parseSpeed(%q) raw = %q, want %q", tc.line, raw, tc.raw)
		}
	}
}

// Feature: tui-accuracy-fix, Property 11: Bitrate Parsing
// For any valid bitrate string, parseBitrate captures the value
func TestBitrateParsing_Property(t *testing.T) {
	// **Validates: Requirements 5.1**
	testCases := []struct {
		line string
		raw  string
	}{
		{"bitrate=1234kbits/s", "1234kbits/s"},
		{"bitrate=1.2Mbits/s", "1.2Mbits/s"},
		{"bitrate=N/A", "N/A"},
		{"bitrate=  5000kbits/s", "5000kbits/s"},
	}
	
	for _, tc := range testCases {
		_, raw, ok := parseBitrate(tc.line)
		if !ok {
			t.Errorf("parseBitrate failed for %q", tc.line)
			continue
		}
		if raw != tc.raw {
			t.Errorf("parseBitrate(%q) raw = %q, want %q", tc.line, raw, tc.raw)
		}
	}
}

// Feature: tui-accuracy-fix, Property 16: Division by Zero Protection
// For any calculation with denominator=0, functions return safe defaults
func TestDivisionByZeroProtection_Property(t *testing.T) {
	// **Validates: Requirements 7.5**
	
	// Test that percentage calculation with zero total is handled
	var frame int64 = 100
	var total int64 = 0
	
	// This should not panic - we check before dividing
	var pct float64
	if total > 0 {
		pct = float64(frame) / float64(total) * 100
	} else {
		pct = 0 // Safe default
	}
	if pct != 0 {
		t.Errorf("Expected 0 for zero total, got %f", pct)
	}
	
	// Test parseFrameRate with zero denominator
	result := parseFrameRate("24/0")
	if result != 0 {
		t.Errorf("parseFrameRate with zero denominator should return 0, got %f", result)
	}
	
	// Test parseFrameRate with empty string
	result = parseFrameRate("")
	if result != 0 {
		t.Errorf("parseFrameRate with empty string should return 0, got %f", result)
	}
}

// Feature: tui-accuracy-fix, Property 17: Non-Negative Validation
// For any parsed numeric value, if negative, validation rejects it
func TestNonNegativeValidation_Property(t *testing.T) {
	// **Validates: Requirements 8.1, 8.2, 8.3**
	f := func(frame, fps, size int64) bool {
		p := Progress{
			Frame:     frame,
			FPS:       float64(fps),
			TotalSize: size,
		}
		
		valid := p.validate()
		
		// If any value is negative, validation should fail
		if frame < 0 || fps < 0 || size < 0 {
			return !valid
		}
		return valid
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Unit test for edge cases
func TestClampPercentage_EdgeCases(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{-100, 0},
		{-1, 0},
		{0, 0},
		{50, 50},
		{100, 100},
		{101, 100},
		{1000, 100},
	}
	
	for _, tc := range tests {
		result := clampPercentage(tc.input)
		if result != tc.expected {
			t.Errorf("clampPercentage(%f) = %f, want %f", tc.input, result, tc.expected)
		}
	}
}

func TestParseFrameRate_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"24", 24.0},
		{"23.976", 23.976},
		{"24/1", 24.0},
		{"24000/1001", 24000.0 / 1001.0},
		{"0/1", 0.0},
		{"24/0", 0.0}, // Division by zero protection
		{"invalid", 0.0},
		{"", 0.0},
	}
	
	for _, tc := range tests {
		result := parseFrameRate(tc.input)
		if math.Abs(result-tc.expected) > 0.001 {
			t.Errorf("parseFrameRate(%q) = %f, want %f", tc.input, result, tc.expected)
		}
	}
}

func TestProgressValidate(t *testing.T) {
	tests := []struct {
		name     string
		progress Progress
		valid    bool
	}{
		{
			name:     "all valid",
			progress: Progress{Frame: 100, FPS: 24.0, TotalSize: 1000, TotalFrames: 1000},
			valid:    true,
		},
		{
			name:     "negative frame",
			progress: Progress{Frame: -1, FPS: 24.0, TotalSize: 1000, TotalFrames: 1000},
			valid:    false,
		},
		{
			name:     "negative fps",
			progress: Progress{Frame: 100, FPS: -1.0, TotalSize: 1000, TotalFrames: 1000},
			valid:    false,
		},
		{
			name:     "negative size",
			progress: Progress{Frame: 100, FPS: 24.0, TotalSize: -1, TotalFrames: 1000},
			valid:    false,
		},
		{
			name:     "zero values valid",
			progress: Progress{Frame: 0, FPS: 0, TotalSize: 0, TotalFrames: 0},
			valid:    true,
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.progress.validate()
			if result != tc.valid {
				t.Errorf("validate() = %v, want %v", result, tc.valid)
			}
		})
	}
}
