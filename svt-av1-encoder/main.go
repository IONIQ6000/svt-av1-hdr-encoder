package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"svt-av1-encoder/config"
	"svt-av1-encoder/tui"
)

func main() {
	// Define flags
	profileFlag := flag.String("profile", "default", "Encoding profile: default, quality, podcast, compress, extreme, film")
	listProfiles := flag.Bool("list-profiles", false, "List all available profiles and exit")

	// Custom usage
	flag.Usage = func() {
		fmt.Println("Usage: svt-av1-encoder [options] <input-file>")
		fmt.Println()
		fmt.Println("Encodes video using FFmpeg with SVT-AV1-HDR encoder.")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Profiles:")
		for _, p := range config.AvailableProfiles() {
			fmt.Printf("  %-10s %s\n", p, config.ProfileDescription(p))
		}
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  svt-av1-encoder movie.mkv                    # Use default profile")
		fmt.Println("  svt-av1-encoder -profile=podcast video.mp4   # Use podcast profile")
		fmt.Println("  svt-av1-encoder -profile=quality movie.mkv   # Use quality profile")
	}

	flag.Parse()

	// Handle --list-profiles
	if *listProfiles {
		fmt.Println("Available encoding profiles:")
		fmt.Println()
		for _, p := range config.AvailableProfiles() {
			cfg := config.GetProfile(p)
			fmt.Printf("  %s\n", p)
			fmt.Printf("    %s\n", config.ProfileDescription(p))
			fmt.Printf("    CRF: %d, Preset: %d\n", cfg.CRF, cfg.Preset)
			fmt.Println()
		}
		os.Exit(0)
	}

	// Check for input file
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := args[0]

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Input file not found: %s\n", inputFile)
		os.Exit(1)
	}

	// Parse profile
	profile := config.Profile(strings.ToLower(*profileFlag))
	validProfile := false
	for _, p := range config.AvailableProfiles() {
		if p == profile {
			validProfile = true
			break
		}
	}
	if !validProfile {
		fmt.Fprintf(os.Stderr, "Error: Unknown profile '%s'\n", *profileFlag)
		fmt.Fprintf(os.Stderr, "Available profiles: default, quality, podcast, compress, extreme, film\n")
		os.Exit(1)
	}

	// Get configuration for selected profile
	cfg := config.GetProfile(profile)

	// Create and run the TUI
	model := tui.NewModel(inputFile, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
