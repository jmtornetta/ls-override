#!/usr/bin/env bash
set -e

# Ensure ~/.local/bin exists
mkdir -p "$HOME/.local/bin"

# Build the Go project (assuming main.go is the entry point)
go build -o "$HOME/.local/bin/ls-override" main.go

# Check if ~/.local/bin is in PATH using a simple if/then
if echo ":$PATH:" | grep -q ":$HOME/.local/bin:"; then
    echo "Binary installed at $HOME/.local/bin/ls-override and is already in PATH."
else
    echo "Binary installed at $HOME/.local/bin/ls-override."
    echo "It appears $HOME/.local/bin is not in your PATH."
    echo "Add the following line to your shell startup file (e.g. ~/.bashrc or ~/.zshrc):"
    echo 'export PATH="$HOME/.local/bin:$PATH"'
fi
