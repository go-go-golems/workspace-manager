package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)
)

func main() {
	if err := Execute(); err != nil {
		errorMsg := errorStyle.Render("✗ Error: " + err.Error())
		fmt.Fprintln(os.Stderr, errorMsg)
		os.Exit(1)
	}
}
