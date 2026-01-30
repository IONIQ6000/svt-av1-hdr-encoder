package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"svt-av1-encoder/encoder"
)

// Color palette - modern, readable
var (
	colorPrimary    = lipgloss.Color("#7C3AED") // Violet
	colorSecondary  = lipgloss.Color("#06B6D4") // Cyan
	colorSuccess    = lipgloss.Color("#10B981") // Emerald
	colorError      = lipgloss.Color("#EF4444") // Red
	colorWarning    = lipgloss.Color("#F59E0B") // Amber
	colorMuted      = lipgloss.Color("#6B7280") // Gray
	colorText       = lipgloss.Color("#F9FAFB") // White
	colorTextDim    = lipgloss.Color("#9CA3AF") // Light gray
	colorBorder     = lipgloss.Color("#374151") // Dark gray
	colorBackground = lipgloss.Color("#1F2937") // Dark blue-gray
)

var (
	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	// Section headers
	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				MarginTop(1)

	// Main stats box
	statsBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			MarginTop(1)

	// Individual stat styles
	statLabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(10)

	statValueStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	statUnitStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	// File path styles
	fileBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 2).
			MarginTop(1)

	fileLabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(8)

	filePathStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	// Status styles
	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	// Help text
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	// Log viewport
	logBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			MarginTop(1)

	// Percentage styles based on progress
	percentLowStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	percentMidStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	percentHighStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)
)

// formatSpeed handles N/A and missing speed values
func formatSpeed(raw string, speed string) string {
	if raw == "N/A" {
		return "N/A"
	}
	if speed == "" || speed == "0x" {
		return "—"
	}
	return speed
}

// formatBitrateDisplay handles N/A and missing bitrate values
func formatBitrateDisplay(raw string, bitrate string) string {
	if raw == "N/A" || bitrate == "N/A" {
		return "N/A"
	}
	if bitrate == "" {
		return "—"
	}
	return bitrate
}

// formatETADisplay handles unavailable ETA gracefully
func formatETADisplay(eta time.Duration, available bool) string {
	if !available || eta < 0 {
		return "—"
	}
	return formatDuration(eta)
}

// formatPercentage handles cases where percentage cannot be calculated
func formatPercentage(pct float64, totalFrames int64, totalDuration time.Duration) string {
	if totalFrames == 0 && totalDuration == 0 {
		return "..."
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	// Cap display at 99.9% to avoid showing 100% until truly complete
	// (the done state will show completion)
	if pct > 99.9 {
		pct = 99.9
	}
	return fmt.Sprintf("%.1f%%", pct)
}

// getPercentageStyle returns appropriate style based on progress
func getPercentageStyle(pct float64) lipgloss.Style {
	if pct < 33 {
		return percentLowStyle
	} else if pct < 66 {
		return percentMidStyle
	}
	return percentHighStyle
}

// formatSizeDisplay handles early encoding when size is unavailable
func formatSizeDisplay(size int64) string {
	if size <= 0 {
		return "—"
	}
	return formatBytes(size)
}

// View renders the TUI
func (m Model) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render(" ⚡ SVT-AV1-HDR Encoder ")
	b.WriteString(title + "\n")

	switch m.State {
	case StateIdle:
		b.WriteString(m.renderIdleView())

	case StateEncoding:
		b.WriteString(m.renderEncodingView())

	case StateDone:
		b.WriteString(m.renderDoneView())

	case StateError:
		b.WriteString(m.renderErrorView())

	case StateSkipped:
		b.WriteString(m.renderSkippedView())
	}

	// Help footer
	help := helpStyle.Render("  [L] Toggle logs  •  [Q] Quit")
	b.WriteString("\n" + help + "\n")

	return b.String()
}

func (m Model) renderIdleView() string {
	return "\n" + statValueStyle.Render("  Initializing encoder...") + "\n"
}

func (m Model) renderEncodingView() string {
	var b strings.Builder

	if m.Encoder == nil {
		return "\n" + statValueStyle.Render("  Starting encoder...") + "\n"
	}

	prog := m.CurrentProgress

	// Check if we have any progress data yet
	hasProgressData := prog.Frame > 0 || prog.OutTimeUs > 0

	// Progress section
	b.WriteString("\n")

	// Progress bar - clamp to valid range
	percentage := prog.Percentage / 100
	if percentage > 1 {
		percentage = 1
	}
	if percentage < 0 {
		percentage = 0
	}

	// Show a minimal progress if encoding has started but percentage is still 0
	if !hasProgressData && percentage == 0 {
		// Show indeterminate state
		percentage = 0.01 // Just a tiny bit to show something is happening
	}

	progressBar := m.Progress.ViewAs(percentage)

	// Percentage with color based on progress
	var pctStr string
	if !hasProgressData {
		pctStr = "..."
	} else {
		pctStr = formatPercentage(prog.Percentage, prog.TotalFrames, prog.TotalDuration)
	}
	pctStyled := getPercentageStyle(prog.Percentage).Render(pctStr)

	b.WriteString("  " + progressBar + "  " + pctStyled + "\n")

	// Stats section
	elapsed := time.Since(m.StartTime).Round(time.Second)

	// Build stats in a clean grid
	statsContent := m.buildStatsGrid(prog, elapsed)
	b.WriteString(statsBoxStyle.Render(statsContent))
	b.WriteString("\n")

	// Files section
	filesContent := m.buildFilesSection()
	b.WriteString(fileBoxStyle.Render(filesContent))

	// Log viewport if enabled
	if m.ShowLogs {
		b.WriteString("\n")
		logHeader := sectionHeaderStyle.Render("  Encoder Output")
		b.WriteString(logHeader + "\n")
		b.WriteString(logBoxStyle.Render(m.LogViewport.View()))
	}

	return b.String()
}

func (m Model) buildStatsGrid(prog encoder.Progress, elapsed time.Duration) string {
	var lines []string

	// Row 1: Frame progress and FPS
	var frameVal, frameTotal, fpsVal string

	// Handle frame display
	if prog.Frame > 0 {
		frameVal = fmt.Sprintf("%d", prog.Frame)
	} else {
		frameVal = "—"
	}

	// Handle total frames display
	if prog.TotalFrames > 0 {
		frameTotal = fmt.Sprintf("/ %d", prog.TotalFrames)
		if prog.FrameEstimated {
			frameTotal += " ~" // Indicate estimate with tilde
		}
	} else {
		frameTotal = "/ —"
	}

	// Handle FPS display - show placeholder until we have a valid reading
	if prog.FPS > 0 {
		fpsVal = fmt.Sprintf("%.1f", prog.FPS)
	} else if prog.LastValidFPS > 0 {
		// Use last valid FPS if current is 0 (temporary dip)
		fpsVal = fmt.Sprintf("%.1f", prog.LastValidFPS)
	} else {
		fpsVal = "—"
	}

	line1 := lipgloss.JoinHorizontal(lipgloss.Top,
		statLabelStyle.Render("Frame"),
		statValueStyle.Render(frameVal),
		statUnitStyle.Render(" "+frameTotal),
		lipgloss.NewStyle().Width(6).Render(""),
		statLabelStyle.Render("FPS"),
		statValueStyle.Render(fpsVal),
	)
	lines = append(lines, line1)

	// Row 2: Speed and Bitrate
	speedVal := formatSpeed(prog.SpeedRaw, prog.Speed)
	bitrateVal := formatBitrateDisplay(prog.BitrateRaw, prog.Bitrate)

	line2 := lipgloss.JoinHorizontal(lipgloss.Top,
		statLabelStyle.Render("Speed"),
		statValueStyle.Render(speedVal),
		lipgloss.NewStyle().Width(12).Render(""),
		statLabelStyle.Render("Bitrate"),
		statValueStyle.Render(bitrateVal),
	)
	lines = append(lines, line2)

	// Row 3: Size and ETA
	sizeVal := formatSizeDisplay(prog.TotalSize)
	etaVal := formatETADisplay(prog.ETA, prog.ETAAvailable)

	line3 := lipgloss.JoinHorizontal(lipgloss.Top,
		statLabelStyle.Render("Size"),
		statValueStyle.Render(sizeVal),
		lipgloss.NewStyle().Width(12).Render(""),
		statLabelStyle.Render("ETA"),
		statValueStyle.Render(etaVal),
	)
	lines = append(lines, line3)

	// Row 4: Elapsed time
	line4 := lipgloss.JoinHorizontal(lipgloss.Top,
		statLabelStyle.Render("Elapsed"),
		statValueStyle.Render(formatDuration(elapsed)),
	)
	lines = append(lines, line4)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) buildFilesSection() string {
	outputPath := ""
	if m.Encoder != nil {
		outputPath = m.Encoder.OutputPath
	}

	// Truncate paths if too long
	maxPathLen := m.Width - 16
	if maxPathLen < 20 {
		maxPathLen = 60
	}

	inputDisplay := truncatePath(m.InputFile, maxPathLen)
	outputDisplay := truncatePath(outputPath, maxPathLen)

	line1 := fileLabelStyle.Render("Input") + filePathStyle.Render(inputDisplay)
	line2 := fileLabelStyle.Render("Output") + filePathStyle.Render(outputDisplay)

	return line1 + "\n" + line2
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	// Show beginning and end
	if maxLen < 20 {
		return path[:maxLen-3] + "..."
	}
	half := (maxLen - 5) / 2
	return path[:half] + " ... " + path[len(path)-half:]
}

func (m Model) renderDoneView() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(successStyle.Render("  ✓ Encoding Complete!") + "\n")

	if m.Encoder != nil {
		elapsed := time.Since(m.StartTime).Round(time.Second)

		// Get actual file size from disk
		finalSize := m.CurrentProgress.TotalSize
		if actualSize, err := m.Encoder.GetActualOutputSize(); err == nil {
			finalSize = actualSize
		}

		// Check output size
		passed, ratio, err := m.Encoder.CheckOutputSize()

		var lines []string

		// Output path
		lines = append(lines,
			statLabelStyle.Render("Output")+filePathStyle.Render(m.Encoder.OutputPath))

		// Time
		lines = append(lines,
			statLabelStyle.Render("Time")+statValueStyle.Render(formatDuration(elapsed)))

		// Final size
		lines = append(lines,
			statLabelStyle.Render("Size")+statValueStyle.Render(formatBytes(finalSize)))

		// Size comparison
		if err == nil {
			if passed {
				sizeStr := fmt.Sprintf("%.1f%% of original", ratio)
				lines = append(lines, successStyle.Render("  ✓ "+sizeStr))
			} else {
				sizeStr := fmt.Sprintf("%.1f%% of original (exceeds %d%% limit)", ratio, m.Config.MaxSizePercent)
				lines = append(lines, errorStyle.Render("  ✗ "+sizeStr))
			}
		}

		content := lipgloss.JoinVertical(lipgloss.Left, lines...)
		b.WriteString(statsBoxStyle.Render(content))
	}

	return b.String()
}

func (m Model) renderErrorView() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(errorStyle.Render("  ✗ Encoding Failed") + "\n\n")

	// Error message in a box
	errBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorError).
		Padding(0, 2).
		Foreground(colorError).
		Render(m.ErrorMessage)

	b.WriteString(errBox + "\n")

	// Show logs if available
	if m.ShowLogs && m.LogViewport.TotalLineCount() > 0 {
		b.WriteString("\n")
		logHeader := sectionHeaderStyle.Render("  Encoder Output")
		b.WriteString(logHeader + "\n")
		b.WriteString(logBoxStyle.Render(m.LogViewport.View()))
	}

	return b.String()
}

func (m Model) renderSkippedView() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(warningStyle.Render("  ⊘ Encoding Skipped") + "\n\n")

	// Reason
	reasonBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorWarning).
		Padding(0, 2).
		Render(
			statLabelStyle.Render("Reason") + "\n" +
				statValueStyle.Render(m.SkippedReason),
		)

	b.WriteString(reasonBox + "\n\n")

	// Input file
	b.WriteString(fileLabelStyle.Render("Input") + filePathStyle.Render(m.InputFile) + "\n")

	return b.String()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "—"
	}
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
