package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// teaProgram is a package-level reference used by the auth callback goroutine
// to send messages back to the running TUI program.
var teaProgram *tea.Program

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	teaProgram = p
	if _, err := p.Run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
