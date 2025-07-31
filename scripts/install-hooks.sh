#!/bin/bash

set -e

echo "Setting up pre-commit hooks for CODA project..."

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required for pre-commit. Please install Python 3 first."
    exit 1
fi

# Check if pip is installed
if ! command -v pip3 &> /dev/null; then
    echo "Error: pip3 is required. Please install pip3 first."
    exit 1
fi

# Install pre-commit
echo "Installing pre-commit..."
pip3 install --user pre-commit

# Add user's Python bin directory to PATH if not already there
PYTHON_USER_BIN=$(python3 -m site --user-base)/bin
if [[ ":$PATH:" != *":$PYTHON_USER_BIN:"* ]]; then
    echo "Adding $PYTHON_USER_BIN to PATH..."
    echo "Please add the following line to your shell configuration file (.bashrc, .zshrc, etc.):"
    echo "export PATH=\"\$PATH:$PYTHON_USER_BIN\""
fi

# Check if pre-commit is accessible
if ! command -v pre-commit &> /dev/null; then
    echo "Warning: pre-commit command not found. Please ensure $PYTHON_USER_BIN is in your PATH."
    echo "You may need to restart your shell or run: export PATH=\"\$PATH:$PYTHON_USER_BIN\""
    exit 1
fi

# Install the git hook scripts
echo "Installing git hooks..."
pre-commit install

# Run against all files to check current state
echo "Running pre-commit on all files (this may take a while on first run)..."
pre-commit run --all-files || true

echo ""
echo "✅ Pre-commit hooks installed successfully!"
echo ""
echo "The following checks will run automatically before each commit:"
echo "  - Go formatting (go fmt)"
echo "  - Go vetting (go vet)"
echo "  - Go imports organization"
echo "  - Go cyclomatic complexity check"
echo "  - Go module tidiness (go mod tidy)"
echo "  - Go unit tests"
echo "  - GolangCI-Lint"
echo "  - Trailing whitespace removal"
echo "  - End of file fixer"
echo "  - YAML validation"
echo "  - JSON validation"
echo "  - Large file check"
echo "  - Merge conflict check"
echo "  - Secret detection"
echo "  - Markdown linting"
echo ""
echo "To run hooks manually: pre-commit run --all-files"
echo "To skip hooks temporarily: git commit --no-verify"