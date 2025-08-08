package main

import (
	"github.com/common-creation/coda/cmd"
)

// Version information (populated during build)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version information
	cmd.SetVersion(version, commit, date)

	// Execute the root command
	cmd.Execute()
}
