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
	"fadedblue":    "\033[38;2;70;70;150m",
	"fadedgreen":   "\033[38;2;70;150;70m",
	"fadedcyan":    "\033[38;2;70;150;150m",
	"fadedmagenta": "\033[38;2;150;70;150m",
	"fadedyellow":  "\033[38;2;150;150;70m",
	"fadedred":     "\033[38;2;150;70;70m",
	"fadedgray":    "\033[38;2;100;100;100m",
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
	cmd := exec.Command("ls", append(args, "--color=always", "-1", "-A", "-F", "--group-directories-first")...)
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
			info, err := os.Stat(nameStripped)
			if err == nil && info.IsDir() {
				files[i] = nameColors["dotdir"] + nameStripped + "\033[0m"
			} else {
				// Only recolor if ls didn't color it
				if !strings.Contains(f, "\x1b[") {
					files[i] = nameColors["dotfile"] + nameStripped + "\033[0m"
				}
			}
		}
	}

	termWidth, err := getTerminalWidth()
	if err != nil {
		// If we can't determine, default to 80
		termWidth = 80
	}

	if len(files) == 0 {
		return
	}

	// We'll try to find the optimal number of columns.
	// Start from the maximum possible columns and go down until we find a fit.
	// Maximum possible columns can't be more than number of files.
	// Also can't be more than termWidth/2 just as a heuristic to avoid silly loops.
	maxPossibleCols := len(files)
	if maxPossibleCols > termWidth {
		maxPossibleCols = termWidth
	}
	if maxPossibleCols < 1 {
		maxPossibleCols = 1
	}

	padding := 2

	bestCols := 1

	// Try from maxPossibleCols down to 1
	for tryCols := maxPossibleCols; tryCols > 0; tryCols-- {
		rows := (len(files) + tryCols - 1) / tryCols

		// Compute column widths for this layout
		colWidths := make([]int, tryCols)
		totalWidth := 0
		for col := 0; col < tryCols; col++ {
			maxW := 0
			for row := 0; row < rows; row++ {
				index := col*rows + row
				if index >= len(files) {
					break
				}
				displayLen := len(stripANSI(files[index]))
				if displayLen > maxW {
					maxW = displayLen
				}
			}
			colWidths[col] = maxW
		}

		for i, w := range colWidths {
			totalWidth += w
			if i < tryCols-1 {
				totalWidth += padding
			}
		}

		if totalWidth <= termWidth {
			// This fits, record it and break (since we are going top-down from largest cols)
			bestCols = tryCols
			break
		}
	}

	// Now print using bestCols in vertical layout
	rows := (len(files) + bestCols - 1) / bestCols
	colWidths := make([]int, bestCols)
	for col := 0; col < bestCols; col++ {
		maxW := 0
		for row := 0; row < rows; row++ {
			index := col*rows + row
			if index >= len(files) {
				break
			}
			displayLen := len(stripANSI(files[index]))
			if displayLen > maxW {
				maxW = displayLen
			}
		}
		colWidths[col] = maxW
	}

	for row := 0; row < rows; row++ {
		var buffer bytes.Buffer
		for col := 0; col < bestCols; col++ {
			index := col*rows + row
			if index >= len(files) {
				break
			}
			f := files[index]
			displayLen := len(stripANSI(f))
			buffer.WriteString(f)
			if col < bestCols-1 {
				spaces := colWidths[col] - displayLen + padding
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
