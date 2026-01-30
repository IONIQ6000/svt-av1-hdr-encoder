package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"svt-av1-encoder/config"
	"svt-av1-encoder/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: svt-av1-encoder <input-file>")
		fmt.Println()
		fmt.Println("Encodes video using FFmpeg with SVT-AV1-HDR encoder.")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  svt-av1-encoder movie.mkv")
		os.Exit(1)
	}

	inputFile := os.Args[1]

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file not found: %s\n", inputFile)
		os.Exit(1)
	}

	// Use default configuration
	cfg := config.DefaultConfig()

	// Create and run the TUI
	model := tui.NewModel(inputFile, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
