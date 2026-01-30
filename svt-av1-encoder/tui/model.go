package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"svt-av1-encoder/config"
	"svt-av1-encoder/encoder"
)

// State represents the current application state
type State int

const (
	StateIdle State = iota
	StateEncoding
	StateDone
	StateError
	StateSkipped
)

type SkippedMsg struct {
	Reason string
}

// EncoderStartedMsg is sent when the encoder has started successfully
type EncoderStartedMsg struct {
	Encoder *encoder.Encoder
}

type EncoderErrorMsg struct {
	Err error
}

// Model is the Bubble Tea model for the TUI
type Model struct {
	Encoder         *encoder.Encoder
	Config          config.Config
	State           State
	Progress        progress.Model
	LogViewport     viewport.Model
	ShowLogs        bool
	Width           int
	Height          int
	InputFile       string
	StartTime       time.Time
	ErrorMessage    string
	SkippedReason   string
	CurrentProgress encoder.Progress // Local safe copy
}

// TickMsg is sent periodically to update the UI
type TickMsg time.Time

// NewModel creates a new TUI model
func NewModel(inputFile string, cfg config.Config) Model {
	// Custom gradient: violet -> cyan -> emerald (matches our color scheme)
	prog := progress.New(
		progress.WithGradient("#7C3AED", "#10B981"),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	vp := viewport.New(80, 12)
	vp.SetContent("")

	return Model{
		Config:      cfg,
		State:       StateIdle,
		Progress:    prog,
		LogViewport: vp,
		ShowLogs:    false,
		InputFile:   inputFile,
	}
}

// Init initializes the Bubble Tea program
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.startEncoding(),
	)
}

func (m *Model) startEncoding() tea.Cmd {
	return func() tea.Msg {
		enc := encoder.New(m.InputFile, m.Config)

		// Check bitrate if configured
		if m.Config.MinBitrate > 0 {
			bitrate, err := enc.GetBitrate()
			if err == nil && bitrate > 0 {
				if bitrate < m.Config.MinBitrate {
					return SkippedMsg{
						Reason: fmt.Sprintf("Source bitrate %d kbps is below minimum %d kbps", bitrate, m.Config.MinBitrate),
					}
				}
			}
			// If we can't determine bitrate, we proceed safely
		}

		// Get total frames for progress calculation
		if err := enc.GetTotalFrames(); err != nil {
			return EncoderErrorMsg{Err: err}
		}

		if err := enc.Start(); err != nil {
			return EncoderErrorMsg{Err: err}
		}

		return EncoderStartedMsg{Encoder: enc}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.Encoder != nil {
				m.Encoder.Stop()
			}
			return m, tea.Quit
		case "l":
			m.ShowLogs = !m.ShowLogs
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Progress.Width = msg.Width - 20
		m.LogViewport.Width = msg.Width - 4

		// Ensure viewport height doesn't go negative
		logHeight := msg.Height - 20
		if logHeight < 0 {
			logHeight = 0
		}
		m.LogViewport.Height = logHeight

	case EncoderStartedMsg:
		m.Encoder = msg.Encoder
		m.State = StateEncoding
		m.StartTime = time.Now()
		cmds = append(cmds, tickCmd())

	case EncoderErrorMsg:
		m.State = StateError
		m.ErrorMessage = msg.Err.Error()
		return m, nil

	case SkippedMsg:
		m.State = StateSkipped
		m.SkippedReason = msg.Reason
		return m, nil

	case TickMsg:
		if m.Encoder != nil {
			// Thread-safe state retrieval
			prog, logs, done, err := m.Encoder.GetState()

			// Update local state
			m.CurrentProgress = prog

			// Update log viewport content
			if len(logs) > 0 {
				m.LogViewport.SetContent(strings.Join(logs, "\n"))
				m.LogViewport.GotoBottom()
			}

			// Check if encoding is done
			if done {
				if err != nil {
					m.State = StateError
					m.ErrorMessage = err.Error()
				} else {
					m.State = StateDone
				}
				return m, nil
			}

			cmds = append(cmds, tickCmd())
		}

	case error:
		m.State = StateError
		m.ErrorMessage = msg.Error()
		return m, nil
	}

	// Update viewport if showing logs
	if m.ShowLogs {
		var cmd tea.Cmd
		m.LogViewport, cmd = m.LogViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
