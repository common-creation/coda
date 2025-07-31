# Simple Go Project

This is a test project used for E2E testing of CODA.

## Features

- Simple command-line argument processing
- Basic arithmetic operations
- Example of a buggy function for testing code review

## Usage

```bash
go build -o program main.go
./program "World"
```

## Files

- `main.go` - Main application code
- `utils.go` - Utility functions
- `config.json` - Configuration file
- `data.txt` - Sample data file

## Known Issues

- The `multiply` function has a bug (returns addition instead of multiplication)
- Error handling could be improved
- No input validation for the name parameter

## TODO

- [ ] Fix the multiply function
- [ ] Add proper error handling
- [ ] Add unit tests
- [ ] Add input validation