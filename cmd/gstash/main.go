package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JonathanInTheClouds/gstash/internal/git"
	"github.com/JonathanInTheClouds/gstash/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	version := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *version {
		fmt.Println("gstash version", Version)
		os.Exit(0)
	}

	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, "✗ Not inside a git repository.")
		os.Exit(1)
	}

	m := ui.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}