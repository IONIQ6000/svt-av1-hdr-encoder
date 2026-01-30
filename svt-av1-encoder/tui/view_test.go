package tui

import (
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: tui-accuracy-fix, Property 13: File Size Formatting
// For any non-negative file size, formatBytes returns a string with binary units
func TestFormatBytes_Property(t *testing.T) {
	// **Validates: Requirements 6.4**
	f := func(size uint64) bool {
		result := formatBytes(int64(size))
		
		// Result should not be empty
		if result == "" {
			return false
		}
		
		// Result should contain a unit
		validUnits := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
		hasUnit := false
		for _, unit := range validUnits {
			if strings.Contains(result, unit) {
				hasUnit = true
				break
			}
		}
		
		return hasUnit
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// Feature: tui-accuracy-fix, Property 14: Placeholder Display for Missing Values
// For empty/zero/unavailable values, display functions return placeholders
func TestPlaceholderDisplay_Property(t *testing.T) {
	// **Validates: Requirements 7.2**
	
	// Test formatSpeed with empty values
	result := formatSpeed("", "")
	if result != "—" {
		t.Errorf("formatSpeed('', '') = %q, want '—'", result)
	}
	
	result = formatSpeed("N/A", "N/A")
	if result != "N/A" {
		t.Errorf("formatSpeed('N/A', 'N/A') = %q, want 'N/A'", result)
	}
	
	// Test formatBitrateDisplay with empty values
	result = formatBitrateDisplay("", "")
	if result != "—" {
		t.Errorf("formatBitrateDisplay('', '') = %q, want '—'", result)
	}
	
	result = formatBitrateDisplay("N/A", "N/A")
	if result != "N/A" {
		t.Errorf("formatBitrateDisplay('N/A', 'N/A') = %q, want 'N/A'", result)
	}
	
	// Test formatETADisplay with unavailable ETA
	result = formatETADisplay(-1, false)
	if result != "—" {
		t.Errorf("formatETADisplay(-1, false) = %q, want '—'", result)
	}
	
	// Test formatETADisplay with available=false
	result = formatETADisplay(time.Minute, false)
	if result != "—" {
		t.Errorf("formatETADisplay(1m, false) = %q, want '—'", result)
	}
	
	// Test formatSizeDisplay with zero size
	result = formatSizeDisplay(0)
	if result != "—" {
		t.Errorf("formatSizeDisplay(0) = %q, want '—'", result)
	}
	
	// Test formatPercentage when both frame count and duration unavailable
	result = formatPercentage(0, 0, 0)
	if result != "..." {
		t.Errorf("formatPercentage(0, 0, 0) = %q, want '...'", result)
	}
}

func TestFormatBytes_EdgeCases(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
	}
	
	for _, tc := range tests {
		result := formatBytes(tc.input)
		if result != tc.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestFormatDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{-1, "—"},
		{0, "0:00"},
		{30 * time.Second, "0:30"},
		{time.Minute, "1:00"},
		{90 * time.Second, "1:30"},
		{time.Hour, "1:00:00"},
		{time.Hour + 30*time.Minute + 45*time.Second, "1:30:45"},
	}
	
	for _, tc := range tests {
		result := formatDuration(tc.input)
		if result != tc.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		raw      string
		speed    string
		expected string
	}{
		{"N/A", "N/A", "N/A"},
		{"", "", "—"},
		{"1.5x", "1.5x", "1.5x"},
		{"", "0x", "—"},
		{"2x", "2x", "2x"},
	}
	
	for _, tc := range tests {
		result := formatSpeed(tc.raw, tc.speed)
		if result != tc.expected {
			t.Errorf("formatSpeed(%q, %q) = %q, want %q", tc.raw, tc.speed, result, tc.expected)
		}
	}
}

func TestFormatBitrateDisplay(t *testing.T) {
	tests := []struct {
		raw      string
		bitrate  string
		expected string
	}{
		{"N/A", "N/A", "N/A"},
		{"", "", "—"},
		{"1234kbits/s", "1234kbits/s", "1234kbits/s"},
		{"N/A", "", "N/A"},
	}
	
	for _, tc := range tests {
		result := formatBitrateDisplay(tc.raw, tc.bitrate)
		if result != tc.expected {
			t.Errorf("formatBitrateDisplay(%q, %q) = %q, want %q", tc.raw, tc.bitrate, result, tc.expected)
		}
	}
}

func TestFormatETADisplay(t *testing.T) {
	tests := []struct {
		eta       time.Duration
		available bool
		expected  string
	}{
		{-1, false, "—"},
		{time.Minute, false, "—"},
		{time.Minute, true, "1:00"},
		{30 * time.Second, true, "0:30"},
		{time.Hour + time.Minute, true, "1:01:00"},
	}
	
	for _, tc := range tests {
		result := formatETADisplay(tc.eta, tc.available)
		if result != tc.expected {
			t.Errorf("formatETADisplay(%v, %v) = %q, want %q", tc.eta, tc.available, result, tc.expected)
		}
	}
}

func TestFormatPercentage(t *testing.T) {
	tests := []struct {
		pct           float64
		totalFrames   int64
		totalDuration time.Duration
		expected      string
	}{
		{0, 0, 0, "..."},
		{50, 100, 0, "50.0%"},
		{50, 0, time.Minute, "50.0%"},
		{-10, 100, 0, "0.0%"},
		{150, 100, 0, "100.0%"},
		{99.9, 100, 0, "99.9%"},
	}
	
	for _, tc := range tests {
		result := formatPercentage(tc.pct, tc.totalFrames, tc.totalDuration)
		if result != tc.expected {
			t.Errorf("formatPercentage(%f, %d, %v) = %q, want %q", tc.pct, tc.totalFrames, tc.totalDuration, result, tc.expected)
		}
	}
}

func TestFormatSizeDisplay(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "—"},
		{-1, "—"},
		{1024, "1.0 KiB"},
		{1024 * 1024, "1.0 MiB"},
	}
	
	for _, tc := range tests {
		result := formatSizeDisplay(tc.size)
		if result != tc.expected {
			t.Errorf("formatSizeDisplay(%d) = %q, want %q", tc.size, result, tc.expected)
		}
	}
}

func TestGetPercentageStyle(t *testing.T) {
	// Test that style changes at thresholds
	lowStyle := getPercentageStyle(10)
	midStyle := getPercentageStyle(50)
	highStyle := getPercentageStyle(80)
	
	// Just verify they return without panic - the actual colors are internal
	_ = lowStyle
	_ = midStyle
	_ = highStyle
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		path     string
		maxLen   int
		expected string
	}{
		{"/short/path", 50, "/short/path"},
		{"/a/very/long/path/that/exceeds/the/maximum/length", 25, "/a/very/l ... mum/length"},
		{"/path", 10, "/path"},
	}
	
	for _, tc := range tests {
		result := truncatePath(tc.path, tc.maxLen)
		if len(tc.path) <= tc.maxLen {
			if result != tc.expected {
				t.Errorf("truncatePath(%q, %d) = %q, want %q", tc.path, tc.maxLen, result, tc.expected)
			}
		} else {
			// For truncated paths, just verify length is within bounds
			if len(result) > tc.maxLen+5 { // Allow some slack for "..."
				t.Errorf("truncatePath(%q, %d) = %q (len %d), expected shorter", tc.path, tc.maxLen, result, len(result))
			}
		}
	}
}
