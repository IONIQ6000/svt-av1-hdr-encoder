package encoder

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"svt-av1-encoder/config"
	"sync"
	"time"
)

// Progress represents the current encoding progress
type Progress struct {
	Frame         int64
	FPS           float64
	Bitrate       string
	TotalSize     int64
	OutTimeUs     int64
	Speed         string
	Percentage    float64
	TotalFrames   int64
	TotalDuration time.Duration
	ETA           time.Duration

	// Fields for accuracy and display
	SpeedRaw       string    // Raw speed string from FFmpeg (may be "N/A")
	BitrateRaw     string    // Raw bitrate string from FFmpeg
	ETAAvailable   bool      // Whether ETA can be calculated
	StartTime      time.Time // When encoding started (for warmup detection)
	LastValidFPS   float64   // Last known good FPS value
	LastValidSpeed float64   // Last known good speed multiplier
	FrameEstimated bool      // Whether TotalFrames is estimated vs actual
	SourceFPS      float64   // Source video frame rate (for accurate frame estimation)
}

// clampPercentage ensures percentage is within 0-100 range
func clampPercentage(pct float64) float64 {
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// validate checks all progress values for sanity
func (p *Progress) validate() bool {
	return p.Frame >= 0 && p.FPS >= 0 && p.TotalSize >= 0 && p.TotalFrames >= 0
}

// parseSpeed extracts speed multiplier from FFmpeg output
// Returns (speed float64, raw string, ok bool)
func parseSpeed(line string) (float64, string, bool) {
	// Match speed=N/A or speed=1.23x (with optional whitespace)
	speedRe := regexp.MustCompile(`speed=\s*([\d.]+x|N/A)\s*$`)
	m := speedRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return 0, "", false
	}
	raw := m[1]
	if raw == "N/A" {
		return 0, raw, true
	}
	speedStr := strings.TrimSuffix(raw, "x")
	speed, err := strconv.ParseFloat(speedStr, 64)
	if err != nil {
		return 0, raw, false
	}
	return speed, raw, true
}

// parseBitrate extracts bitrate from FFmpeg output
// Returns (normalized string, raw string, ok bool)
func parseBitrate(line string) (string, string, bool) {
	// Match various bitrate formats: 1234kbits/s, 1.2Mbits/s, N/A, or plain bits/s
	// Also handle edge cases like whitespace variations
	bitrateRe := regexp.MustCompile(`bitrate=\s*([\d.]+\s*[kKmMgG]?bits?/s|N/A)\s*$`)
	m := bitrateRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return "", "", false
	}
	raw := strings.TrimSpace(m[1])
	return raw, raw, true
}

// parseFrameRate handles fractional formats like "24000/1001" or "23.976"
func parseFrameRate(fpsStr string) float64 {
	fpsStr = strings.TrimSpace(fpsStr)
	if fpsStr == "" {
		return 0
	}
	if strings.Contains(fpsStr, "/") {
		parts := strings.Split(fpsStr, "/")
		if len(parts) == 2 {
			num, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			den, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil && den > 0 {
				return num / den
			}
		}
		return 0
	}
	fps, _ := strconv.ParseFloat(fpsStr, 64)
	return fps
}

// parseOutTime parses FFmpeg's out_time format "HH:MM:SS.microseconds"
func parseOutTime(timeStr string) int64 {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" || timeStr == "N/A" {
		return -1
	}

	// Format: HH:MM:SS.microseconds (e.g., "00:01:23.456789")
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return -1
	}

	hours, err1 := strconv.ParseInt(parts[0], 10, 64)
	mins, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return -1
	}

	// Seconds may have decimal (microseconds)
	secParts := strings.Split(parts[2], ".")
	secs, err3 := strconv.ParseInt(secParts[0], 10, 64)
	if err3 != nil {
		return -1
	}

	var microsecs int64 = 0
	if len(secParts) > 1 {
		// Pad or truncate to 6 digits for microseconds
		usStr := secParts[1]
		for len(usStr) < 6 {
			usStr += "0"
		}
		if len(usStr) > 6 {
			usStr = usStr[:6]
		}
		microsecs, _ = strconv.ParseInt(usStr, 10, 64)
	}

	totalUs := hours*3600*1000000 + mins*60*1000000 + secs*1000000 + microsecs
	return totalUs
}

// Encoder handles FFmpeg encoding with svt-av1-hdr
type Encoder struct {
	Config     config.Config
	InputPath  string
	OutputPath string
	Progress   Progress
	cmd        *exec.Cmd
	Done       bool
	Error      error
	LogLines   []string
	mu         sync.Mutex // Protects Progress and LogLines
}

// New creates a new Encoder instance
func New(inputPath string, cfg config.Config) *Encoder {
	// Generate output path (same directory, .av1.mkv extension)
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	outputPath := base + ".av1.mkv"

	return &Encoder{
		Config:     cfg,
		InputPath:  inputPath,
		OutputPath: outputPath,
		LogLines:   make([]string, 0),
	}
}

// GetTotalFrames probes the input file to get total frame count and source FPS
func (e *Encoder) GetTotalFrames() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get frame rate first - we need this for any estimation
	fpsCmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=r_frame_rate,avg_frame_rate",
		"-of", "csv=p=0",
		e.InputPath,
	)

	fpsOutput, _ := fpsCmd.Output()
	fpsParts := strings.Split(strings.TrimSpace(string(fpsOutput)), ",")

	var sourceFPS float64
	if len(fpsParts) >= 1 {
		// Prefer r_frame_rate (real frame rate) over avg_frame_rate
		sourceFPS = parseFrameRate(fpsParts[0])
		if sourceFPS <= 0 && len(fpsParts) >= 2 {
			sourceFPS = parseFrameRate(fpsParts[1])
		}
	}

	e.mu.Lock()
	if sourceFPS > 0 {
		e.Progress.SourceFPS = sourceFPS
	} else {
		e.Progress.SourceFPS = 24.0 // Fallback assumption
	}
	e.mu.Unlock()

	// Try nb_frames from container (most accurate)
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=nb_frames",
		"-of", "csv=p=0",
		e.InputPath,
	)

	output, err := cmd.Output()
	if err == nil {
		frames, _ := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
		if frames > 0 {
			e.mu.Lock()
			e.Progress.TotalFrames = frames
			e.Progress.FrameEstimated = false
			e.mu.Unlock()
			return nil
		}
	}

	// Try counting frames with packet counting (slow but accurate for problem files)
	// Skip this for now - too slow. Fall back to duration-based estimation.

	// Fallback: Estimate from duration
	return e.estimateFramesFromDuration()
}

func (e *Encoder) estimateFramesFromDuration() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get duration from format (more reliable than stream duration)
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		e.InputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil || duration <= 0 {
		return nil
	}

	// Sanity check: duration should be reasonable (less than 24 hours)
	if duration > 86400 {
		// Very long video - proceed with caution, don't estimate frames
		e.mu.Lock()
		e.Progress.TotalDuration = time.Duration(duration * float64(time.Second))
		e.mu.Unlock()
		return nil
	}

	e.mu.Lock()
	e.Progress.TotalDuration = time.Duration(duration * float64(time.Second))

	// Only estimate frames if we have source FPS and it's reasonable
	if e.Progress.SourceFPS > 0 && e.Progress.SourceFPS < 1000 {
		estimatedFrames := int64(duration * e.Progress.SourceFPS)
		// Sanity check: estimated frames should be positive and reasonable
		if estimatedFrames > 0 && estimatedFrames < 100000000 { // Max ~100M frames
			e.Progress.TotalFrames = estimatedFrames
			e.Progress.FrameEstimated = true
		}
	}
	e.mu.Unlock()

	return nil
}

// buildFFmpegArgs constructs the FFmpeg command arguments
func (e *Encoder) buildFFmpegArgs() []string {
	args := []string{
		"-hide_banner",
		"-progress", "pipe:1", // Progress output to stdout
		"-i", e.InputPath,
		"-map", "0",
		"-map", "-0:d", // Remove data streams
	}

	// Remove unwanted languages
	for _, lang := range e.Config.RemoveLanguages {
		args = append(args, "-map", fmt.Sprintf("-0:a:m:language:%s", lang))
		args = append(args, "-map", fmt.Sprintf("-0:s:m:language:%s", lang))
	}

	// Remove image codecs
	for _, codec := range e.Config.RemoveImageCodecs {
		args = append(args, "-map", fmt.Sprintf("-0:v:m:codec_name:%s", codec))
	}

	// Video encoding settings
	svtParams := fmt.Sprintf(
		"tune=%d:enable-variance-boost=%d:variance-boost-strength=%d:sharpness=%d:enable-tf=%d:film-grain=%d",
		e.Config.Tune,
		boolToInt(e.Config.VarianceBoost),
		e.Config.VarianceBoostStrength,
		e.Config.Sharpness,
		e.Config.TFStrength,
		e.Config.FilmGrain,
	)

	args = append(args,
		"-c:v", "libsvtav1",
		"-crf", strconv.Itoa(e.Config.CRF),
		"-preset", strconv.Itoa(e.Config.Preset),
		"-g", "240",         // Keyframe every 240 frames (~10 sec at 24fps, ~8 sec at 30fps)
		"-keyint_min", "48", // Minimum keyframe interval (scene changes still insert keyframes)
		"-pix_fmt", "yuv420p10le",
		"-svtav1-params", svtParams,
		"-c:a", "copy",
		"-c:s", "copy",
		"-y",
		e.OutputPath,
	)

	return args
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Start begins the encoding process
func (e *Encoder) Start() error {
	args := e.buildFFmpegArgs()
	e.cmd = exec.Command("ffmpeg", args...)

	e.addLog(fmt.Sprintf("Starting encode: %s", e.InputPath))
	e.addLog(fmt.Sprintf("Output: %s", e.OutputPath))
	e.addLog(fmt.Sprintf("Command: ffmpeg %s", strings.Join(args, " ")))

	// Set encoding start time
	e.mu.Lock()
	e.Progress.StartTime = time.Now()
	e.mu.Unlock()

	stdout, err := e.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := e.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	go e.parseProgress(stdout)
	go e.captureStderr(stderr)

	go func() {
		err := e.cmd.Wait()
		e.mu.Lock()
		if err != nil {
			e.Error = err
			e.LogLines = append(e.LogLines, fmt.Sprintf("Encoding error: %v", err))
		} else {
			// Encoding completed successfully - finalize progress
			e.finalizeProgressLocked()
			e.LogLines = append(e.LogLines, "Encoding completed successfully!")
		}
		e.Done = true
		e.mu.Unlock()
	}()

	return nil
}

// finalizeProgressLocked corrects progress values when encoding completes (must hold mutex)
func (e *Encoder) finalizeProgressLocked() {
	// The actual frame count is whatever we encoded
	if e.Progress.Frame > 0 {
		e.Progress.TotalFrames = e.Progress.Frame
		e.Progress.FrameEstimated = false // It's now the real count
	}
	// Set to 100% complete
	e.Progress.Percentage = 100
	e.Progress.ETA = 0
	e.Progress.ETAAvailable = false
}

// progressUpdate holds a batch of progress values
type progressUpdate struct {
	frame      int64
	frameSet   bool
	fps        float64
	fpsSet     bool
	bitrate    string
	bitrateRaw string
	bitrateSet bool
	size       int64
	sizeSet    bool
	outTimeUs  int64
	outTimeSet bool
	speed      float64
	speedRaw   string
	speedSet   bool
}

// parseProgress reads FFmpeg progress output from stdout
// FFmpeg -progress outputs key=value pairs, with "progress=continue" or "progress=end" as batch markers
func (e *Encoder) parseProgress(r io.Reader) {
	scanner := bufio.NewScanner(r)

	// Increase buffer size to handle potentially long lines (1MB max)
	// Default is 64KB which can be exceeded by some FFmpeg metadata
	const maxScannerBuffer = 1024 * 1024
	scanner.Buffer(make([]byte, 0, 64*1024), maxScannerBuffer)

	// Current batch of updates
	var batch progressUpdate

	for scanner.Scan() {
		line := scanner.Text()

		// Check for batch completion marker
		if strings.HasPrefix(line, "progress=") {
			// Apply the batch
			e.applyProgressBatch(batch)

			// Check if this is the final marker
			if line == "progress=end" {
				e.mu.Lock()
				e.finalizeProgressLocked()
				e.mu.Unlock()
			}

			// Reset for next batch
			batch = progressUpdate{}
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "frame":
			if frame, err := strconv.ParseInt(value, 10, 64); err == nil && frame >= 0 {
				batch.frame = frame
				batch.frameSet = true
			}

		case "fps":
			if fps, err := strconv.ParseFloat(value, 64); err == nil && fps >= 0 {
				batch.fps = fps
				batch.fpsSet = true
			}

		case "bitrate":
			batch.bitrateRaw = value
			if value != "N/A" && value != "" {
				batch.bitrate = value
			} else {
				batch.bitrate = "N/A"
			}
			batch.bitrateSet = true

		case "total_size":
			if size, err := strconv.ParseInt(value, 10, 64); err == nil && size >= 0 {
				batch.size = size
				batch.sizeSet = true
			}

		case "out_time_us":
			if us, err := strconv.ParseInt(value, 10, 64); err == nil && us >= 0 {
				batch.outTimeUs = us
				batch.outTimeSet = true
			}

		case "out_time_ms":
			if ms, err := strconv.ParseInt(value, 10, 64); err == nil && ms >= 0 {
				batch.outTimeUs = ms * 1000
				batch.outTimeSet = true
			}

		case "out_time":
			// Parse HH:MM:SS.microseconds format
			if us := parseOutTime(value); us >= 0 {
				batch.outTimeUs = us
				batch.outTimeSet = true
			}

		case "speed":
			batch.speedRaw = value
			if value == "N/A" {
				batch.speedSet = true
			} else {
				// Parse "1.23x" format
				speedStr := strings.TrimSuffix(value, "x")
				if speed, err := strconv.ParseFloat(speedStr, 64); err == nil && speed >= 0 {
					batch.speed = speed
					batch.speedSet = true
				}
			}
		}
	}

	// Check for scanner errors (e.g., token too long)
	if err := scanner.Err(); err != nil {
		e.addLog(fmt.Sprintf("Progress reader error: %v", err))
	}

	// Apply any remaining batch
	if batch.frameSet || batch.fpsSet || batch.sizeSet {
		e.applyProgressBatch(batch)
	}
}

// applyProgressBatch updates the encoder progress with a batch of values
func (e *Encoder) applyProgressBatch(batch progressUpdate) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Apply frame count
	if batch.frameSet {
		e.Progress.Frame = batch.frame
		// Always adjust total if current exceeds it - container metadata can be wrong too
		if batch.frame > e.Progress.TotalFrames && e.Progress.TotalFrames > 0 {
			e.Progress.TotalFrames = batch.frame
			// Mark as estimated since we're correcting it
			e.Progress.FrameEstimated = true
		}
	}

	// Apply FPS
	if batch.fpsSet {
		e.Progress.FPS = batch.fps
		if batch.fps > 0 {
			e.Progress.LastValidFPS = batch.fps
		}
	}

	// Apply bitrate
	if batch.bitrateSet {
		e.Progress.BitrateRaw = batch.bitrateRaw
		e.Progress.Bitrate = batch.bitrate
	}

	// Apply size
	if batch.sizeSet {
		e.Progress.TotalSize = batch.size
	}

	// Apply time
	if batch.outTimeSet {
		e.Progress.OutTimeUs = batch.outTimeUs
	}

	// Apply speed
	if batch.speedSet {
		e.Progress.SpeedRaw = batch.speedRaw
		if batch.speedRaw == "N/A" {
			e.Progress.Speed = "N/A"
			// Don't clear LastValidSpeed - keep using previous valid value for ETA
		} else if batch.speed > 0 {
			e.Progress.Speed = batch.speedRaw
			e.Progress.LastValidSpeed = batch.speed
		} else {
			// Speed is 0 or unparseable but not N/A - show raw value
			e.Progress.Speed = batch.speedRaw
		}
	}

	// Calculate percentage
	e.calculatePercentageLocked()

	// Calculate ETA
	e.calculateETALocked()
}

// calculatePercentageLocked computes progress percentage (must hold mutex)
func (e *Encoder) calculatePercentageLocked() {
	var framePct, timePct float64
	hasFramePct := false
	hasTimePct := false

	// Calculate frame-based percentage
	if e.Progress.TotalFrames > 0 && e.Progress.Frame > 0 {
		framePct = float64(e.Progress.Frame) / float64(e.Progress.TotalFrames) * 100
		hasFramePct = true
	}

	// Calculate time-based percentage
	if e.Progress.TotalDuration > 0 && e.Progress.OutTimeUs > 0 {
		totalUs := e.Progress.TotalDuration.Microseconds()
		if totalUs > 0 {
			timePct = float64(e.Progress.OutTimeUs) / float64(totalUs) * 100
			hasTimePct = true
		}
	}

	// Determine which percentage to use
	if hasFramePct && hasTimePct {
		// Cross-validate: if they differ significantly, time-based is usually more reliable
		// because duration from container is more accurate than frame count estimates
		diff := framePct - timePct
		if diff < 0 {
			diff = -diff
		}

		// If frame-based is more than 10% off from time-based, prefer time-based
		// Also prefer time-based if frame count was estimated (less reliable)
		if diff > 10 || e.Progress.FrameEstimated {
			e.Progress.Percentage = clampPercentage(timePct)
		} else {
			e.Progress.Percentage = clampPercentage(framePct)
		}
		return
	}

	// Use whichever is available
	if hasTimePct {
		e.Progress.Percentage = clampPercentage(timePct)
		return
	}
	if hasFramePct {
		e.Progress.Percentage = clampPercentage(framePct)
		return
	}

	// Cannot calculate
	e.Progress.Percentage = 0
}

// calculateETALocked computes ETA (must hold mutex)
func (e *Encoder) calculateETALocked() {
	// Check warmup period (first 5 seconds) - SVT-AV1 needs time to stabilize
	// Values during warmup are unreliable and cause erratic ETA jumps
	if !e.Progress.StartTime.IsZero() && time.Since(e.Progress.StartTime) < 5*time.Second {
		e.Progress.ETAAvailable = false
		e.Progress.ETA = -1
		return
	}

	var newETA time.Duration
	etaCalculated := false

	// Method 1: Time-based with speed multiplier (most accurate and reliable)
	// Speed multiplier from FFmpeg directly tells us real-time vs media-time ratio
	if e.Progress.LastValidSpeed > 0 && e.Progress.TotalDuration > 0 && e.Progress.OutTimeUs > 0 {
		totalUs := e.Progress.TotalDuration.Microseconds()
		remainingUs := totalUs - e.Progress.OutTimeUs
		if remainingUs > 0 {
			// ETA = remaining_media_time / speed_multiplier
			etaUs := float64(remainingUs) / e.Progress.LastValidSpeed
			newETA = time.Duration(int64(etaUs)) * time.Microsecond
			etaCalculated = true
		}
	}

	// Method 2: Frame-based with encoding FPS (only if time-based not available)
	// Skip if frame count is estimated - it's unreliable
	if !etaCalculated && !e.Progress.FrameEstimated {
		fps := e.Progress.LastValidFPS
		if fps > 0 && e.Progress.TotalFrames > 0 && e.Progress.Frame > 0 {
			remainingFrames := e.Progress.TotalFrames - e.Progress.Frame
			if remainingFrames > 0 {
				etaSeconds := float64(remainingFrames) / fps
				newETA = time.Duration(etaSeconds * float64(time.Second))
				etaCalculated = true
			}
		}
	}

	// Method 3: Elapsed time extrapolation (fallback)
	// Only use after some progress has been made for stability
	if !etaCalculated && e.Progress.Percentage > 2 && !e.Progress.StartTime.IsZero() {
		elapsed := time.Since(e.Progress.StartTime)
		if elapsed > 10*time.Second {
			// ETA = elapsed * (100 - pct) / pct
			remainingPct := 100 - e.Progress.Percentage
			if remainingPct > 0 {
				newETA = time.Duration(float64(elapsed) * remainingPct / e.Progress.Percentage)
				etaCalculated = true
			}
		}
	}

	if !etaCalculated {
		e.Progress.ETAAvailable = false
		e.Progress.ETA = -1
		return
	}

	// Apply smoothing to prevent erratic jumps
	// Use heavier smoothing (70% old, 30% new) for more stable display
	if e.Progress.ETAAvailable && e.Progress.ETA > 0 {
		oldETA := e.Progress.ETA
		// Clamp extreme changes - if new ETA differs by more than 50%, apply extra dampening
		diff := float64(newETA - oldETA)
		if diff < 0 {
			diff = -diff
		}
		if diff > float64(oldETA)*0.5 {
			// Large jump - apply heavier smoothing
			e.Progress.ETA = time.Duration(float64(newETA)*0.2 + float64(oldETA)*0.8)
		} else {
			e.Progress.ETA = time.Duration(float64(newETA)*0.3 + float64(oldETA)*0.7)
		}
	} else {
		e.Progress.ETA = newETA
	}
	e.Progress.ETAAvailable = true
}

// captureStderr captures FFmpeg stderr output for logs and duration parsing
func (e *Encoder) captureStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	durationRe := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
	fpsRe := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*fps`)
	tbnRe := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*tbr`)

	for scanner.Scan() {
		line := scanner.Text()

		e.mu.Lock()

		// Parse duration if we don't have it yet
		if e.Progress.TotalDuration == 0 {
			if m := durationRe.FindStringSubmatch(line); len(m) > 4 {
				h, _ := strconv.Atoi(m[1])
				min, _ := strconv.Atoi(m[2])
				s, _ := strconv.Atoi(m[3])
				cs, _ := strconv.Atoi(m[4]) // centiseconds

				totalDur := time.Duration(h)*time.Hour +
					time.Duration(min)*time.Minute +
					time.Duration(s)*time.Second +
					time.Duration(cs*10)*time.Millisecond

				e.Progress.TotalDuration = totalDur

				// Update frame estimate if we now have duration
				if e.Progress.SourceFPS > 0 && (e.Progress.TotalFrames == 0 || e.Progress.FrameEstimated) {
					e.Progress.TotalFrames = int64(totalDur.Seconds() * e.Progress.SourceFPS)
					e.Progress.FrameEstimated = true
				}
			}
		}

		// Try to extract source FPS from stream info if we don't have it
		if e.Progress.SourceFPS == 0 {
			// Try "XX fps" format
			if m := fpsRe.FindStringSubmatch(line); len(m) > 1 {
				if fps, err := strconv.ParseFloat(m[1], 64); err == nil && fps > 0 && fps < 1000 {
					e.Progress.SourceFPS = fps
				}
			}
			// Also try "XX tbr" (time base rate)
			if e.Progress.SourceFPS == 0 {
				if m := tbnRe.FindStringSubmatch(line); len(m) > 1 {
					if fps, err := strconv.ParseFloat(m[1], 64); err == nil && fps > 0 && fps < 1000 {
						e.Progress.SourceFPS = fps
					}
				}
			}
		}

		// Keep stderr logs (but filter out progress-like lines)
		if !strings.HasPrefix(line, "frame=") &&
			!strings.HasPrefix(line, "size=") &&
			!strings.HasPrefix(line, "fps=") &&
			line != "" {
			const maxLogs = 100
			e.LogLines = append(e.LogLines, line)
			if len(e.LogLines) > maxLogs {
				e.LogLines = e.LogLines[len(e.LogLines)-maxLogs:]
			}
		}

		e.mu.Unlock()
	}
}

func (e *Encoder) addLog(line string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	const maxLogs = 100
	e.LogLines = append(e.LogLines, line)
	if len(e.LogLines) > maxLogs {
		e.LogLines = e.LogLines[len(e.LogLines)-maxLogs:]
	}
}

// Stop terminates the encoding process
func (e *Encoder) Stop() {
	if e.cmd != nil && e.cmd.Process != nil {
		e.cmd.Process.Kill()
	}
}

// GetState returns a thread-safe snapshot of the encoder state
func (e *Encoder) GetState() (Progress, []string, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Copy log lines to avoid race conditions
	logs := make([]string, len(e.LogLines))
	copy(logs, e.LogLines)

	return e.Progress, logs, e.Done, e.Error
}

// CheckOutputSize verifies the output file meets size requirements
func (e *Encoder) CheckOutputSize() (bool, float64, error) {
	if e.Config.MaxSizePercent == 0 {
		return true, 0, nil
	}

	inputInfo, err := os.Stat(e.InputPath)
	if err != nil {
		return false, 0, err
	}

	outputInfo, err := os.Stat(e.OutputPath)
	if err != nil {
		return false, 0, err
	}

	ratio := float64(outputInfo.Size()) / float64(inputInfo.Size()) * 100
	return ratio <= float64(e.Config.MaxSizePercent), ratio, nil
}

// GetActualOutputSize returns the actual file size from disk
func (e *Encoder) GetActualOutputSize() (int64, error) {
	info, err := os.Stat(e.OutputPath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetBitrate returns the bitrate of the video stream in kbps
func (e *Encoder) GetBitrate() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try format bitrate first
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "format=bit_rate",
		"-of", "csv=p=0",
		e.InputPath,
	)

	output, err := cmd.Output()
	if err == nil {
		bitrateStr := strings.TrimSpace(string(output))
		if bitrateStr != "N/A" && bitrateStr != "" {
			if bps, err := strconv.ParseInt(bitrateStr, 10, 64); err == nil && bps > 0 {
				return int(bps / 1000), nil
			}
		}
	}

	// Fallback: Try stream bitrate
	cmd = exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=bit_rate",
		"-of", "csv=p=0",
		e.InputPath,
	)

	output, err = cmd.Output()
	if err == nil {
		bitrateStr := strings.TrimSpace(string(output))
		if bitrateStr != "N/A" && bitrateStr != "" {
			if bps, err := strconv.ParseInt(bitrateStr, 10, 64); err == nil && bps > 0 {
				return int(bps / 1000), nil
			}
		}
	}

	return 0, fmt.Errorf("could not determine bitrate")
}
