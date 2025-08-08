/*
Copyright © 2025 CODA Project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/common-creation/coda/cmd"
)

func main() {
	// Create context for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start shutdown handler in background
	go func() {
		<-sigChan
		cmd.ShowInfo("Shutting down gracefully...")

		// Cancel context to signal shutdown
		cancel()

		// Attempt graceful MCP shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		done := make(chan struct{})
		go func() {
			defer close(done)
			if err := cmd.ShutdownMCP(); err != nil {
				cmd.ShowWarning("Error shutting down MCP: %v", err)
			}
		}()

		select {
		case <-done:
			cmd.ShowInfo("MCP shutdown completed")
		case <-shutdownCtx.Done():
			cmd.ShowWarning("MCP shutdown timed out")
		}

		// Force exit if needed
		os.Exit(0)
	}()

	// Execute the main command
	cmd.Execute()
}
