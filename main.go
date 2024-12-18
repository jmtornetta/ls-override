package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/term"
)

var colors = map[string]string{
	"gray":         "\033[90m",
	"blue":         "\033[34m",
	"green":        "\033[32m",
	"cyan":         "\033[36m",
	"magenta":      "\033[35m",
	"fadedblue":    "\033[38;2;70;70;150m",   // More faded blue
	"fadedgreen":   "\033[38;2;70;150;70m",   // More faded green
	"fadedcyan":    "\033[38;2;70;150;150m",  // More faded cyan
	"fadedmagenta": "\033[38;2;150;70;150m",  // More faded magenta
	"fadedyellow":  "\033[38;2;150;150;70m",  // More faded yellow
	"fadedred":     "\033[38;2;150;70;70m",   // More faded red
	"fadedgray":    "\033[38;2;100;100;100m", // More faded gray
}

var nameColors = map[string]string{
	"dotdir":  colors["fadedcyan"],
	"dotfile": colors["gray"],
}

// stripANSI removes ANSI escape sequences for length calculation
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func main() {
	args := os.Args[1:]
	cmd := exec.Command("ls", append(args, "--color=always", "-1")...)
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating pipe:", err)
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Error starting ls:", err)
		os.Exit(1)
	}

	var files []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		files = append(files, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading ls output:", err)
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		os.Exit(1)
	}

	// Recolor dotfiles
	for i, f := range files {
		nameStripped := stripANSI(f)
		if strings.HasPrefix(nameStripped, ".") {
			// Check if it's a directory
			info, err := os.Stat(nameStripped)
			if err == nil && info.IsDir() {
				// Dot directory
				files[i] = nameColors["dotdir"] + nameStripped + "\033[0m"
			} else {
				if !strings.Contains(f, "\x1b[") {
					// Dot file with no ls color
					files[i] = nameColors["dotfile"] + nameStripped + "\033[0m"
				}
			}
		}
	}

	// Get terminal width
	termWidth, err := getTerminalWidth()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting terminal width:", err)
		termWidth = 80 // Fallback to 80 if terminal width cannot be determined
	}

	// Define padding between columns
	padding := 2

	// Calculate number of columns
	numCols := 1
	colWidth := 0
	for _, f := range files {
		displayLen := len(stripANSI(f))
		if displayLen > colWidth {
			colWidth = displayLen
		}
	}
	if colWidth > 0 {
		numCols = termWidth / (colWidth + padding)
		if numCols < 1 {
			numCols = 1
		}
	}

	// Calculate max width for each column
	colWidths := make([]int, numCols)
	for i := 0; i < len(files); i++ {
		col := i % numCols
		displayLen := len(stripANSI(files[i]))
		if displayLen > colWidths[col] {
			colWidths[col] = displayLen
		}
	}

	// Print files in a grid
	for i := 0; i < len(files); i += numCols {
		end := i + numCols
		if end > len(files) {
			end = len(files)
		}
		rowFiles := files[i:end]
		var buffer bytes.Buffer
		for j, f := range rowFiles {
			displayLen := len(stripANSI(f))
			buffer.WriteString(f)
			if j < len(rowFiles)-1 {
				spaces := colWidths[j] - displayLen + padding
				if spaces < 1 {
					spaces = 1
				}
				buffer.WriteString(strings.Repeat(" ", spaces))
			}
		}
		fmt.Println(buffer.String())
	}
}

func getTerminalWidth() (int, error) {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 1 {
		return 0, fmt.Errorf("could not determine terminal width")
	}
	return w, nil
}
