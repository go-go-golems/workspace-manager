package output

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
)

var (
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true).
			Underline(true)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)

// PrintError prints an error message with styling
func PrintError(format string, args ...interface{}) {
	msg := ErrorStyle.Render("✗ " + fmt.Sprintf(format, args...))
	fmt.Fprintln(os.Stderr, msg)
}

// PrintSuccess prints a success message with styling
func PrintSuccess(format string, args ...interface{}) {
	msg := SuccessStyle.Render("✓ " + fmt.Sprintf(format, args...))
	fmt.Println(msg)
}

// PrintInfo prints an info message with styling - replaces log.Info for user-facing output
func PrintInfo(format string, args ...interface{}) {
	msg := InfoStyle.Render("ℹ " + fmt.Sprintf(format, args...))
	fmt.Println(msg)
}

// PrintWarning prints a warning message with styling
func PrintWarning(format string, args ...interface{}) {
	msg := WarningStyle.Render("⚠ " + fmt.Sprintf(format, args...))
	fmt.Println(msg)
}

// PrintHeader prints a header message with styling
func PrintHeader(format string, args ...interface{}) {
	msg := HeaderStyle.Render(fmt.Sprintf(format, args...))
	fmt.Println(msg)
}

// LogInfo logs at info level while also printing pretty output to user
func LogInfo(userMsg string, logMsg string, fields ...interface{}) {
	PrintInfo("%s", userMsg)
}

// LogError logs at error level while also printing pretty output to user
func LogError(userMsg string, logMsg string, fields ...interface{}) {
	PrintError("%s", userMsg)
}

// LogWarn logs at warn level while also printing pretty output to user
func LogWarn(userMsg string, logMsg string, fields ...interface{}) {
	PrintWarning("%s", userMsg)
}

// Spinner creates a simple text-based spinner for operations
func Spinner(w io.Writer, msg string) func() {
	chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				fmt.Fprintf(w, "\r%s %s", chars[i%len(chars)], msg)
				i++
			}
		}
	}()

	return func() {
		done <- true
		fmt.Fprintf(w, "\r%s\n", SuccessStyle.Render(msg+" completed"))
	}
}
