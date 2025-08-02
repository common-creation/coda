/*
Copyright © 2025 CODA Project
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version information variables
// These are set at build time using ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	Date      = "unknown"
	GoVersion = runtime.Version()
)

var (
	verbose    bool
	jsonOutput bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display detailed version information about CODA.

Shows the version number, build information, and platform details.`,
	RunE: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Command flags
	versionCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show detailed version information")
	versionCmd.Flags().BoolVar(&jsonOutput, "json", false, "output version information as JSON")
}

func runVersion(cmd *cobra.Command, args []string) error {
	versionInfo := getVersionInfo()

	if jsonOutput {
		return outputJSON(versionInfo)
	}

	if verbose {
		return outputVerbose(versionInfo)
	}

	// Simple version output
	fmt.Printf("CODA version %s\n", versionInfo.Version)
	return nil
}

// VersionInfo contains all version-related information
type VersionInfo struct {
	Version      string            `json:"version"`
	Commit       string            `json:"commit"`
	Date         string            `json:"date"`
	GoVersion    string            `json:"go_version"`
	Platform     string            `json:"platform"`
	Architecture string            `json:"architecture"`
	OS           string            `json:"os"`
	BuildInfo    map[string]string `json:"build_info,omitempty"`
	Features     []string          `json:"features,omitempty"`
}

func getVersionInfo() VersionInfo {
	info := VersionInfo{
		Version:      Version,
		Commit:       Commit,
		Date:         Date,
		GoVersion:    GoVersion,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}

	// Get build info if available
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		info.BuildInfo = make(map[string]string)

		// Main module information
		if buildInfo.Main.Version != "" {
			info.BuildInfo["module_version"] = buildInfo.Main.Version
		}
		if buildInfo.Main.Sum != "" {
			info.BuildInfo["module_sum"] = buildInfo.Main.Sum
		}

		// Build settings
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				if info.Commit == "unknown" {
					info.Commit = setting.Value
				}
			case "vcs.time":
				if info.Date == "unknown" {
					info.Date = setting.Value
				}
			case "vcs.modified":
				info.BuildInfo["vcs_modified"] = setting.Value
			case "GOOS", "GOARCH", "CGO_ENABLED":
				info.BuildInfo[setting.Key] = setting.Value
			}
		}

		// Dependency count
		info.BuildInfo["dependency_count"] = fmt.Sprintf("%d", len(buildInfo.Deps))
	}

	// Add feature flags
	info.Features = getEnabledFeatures()

	return info
}

func getEnabledFeatures() []string {
	features := []string{}

	// Check for enabled features based on build tags or runtime checks
	features = append(features, "file-operations")
	features = append(features, "ai-chat")
	features = append(features, "multi-model-support")

	// Tools are always available
	features = append(features, "tool-execution")

	return features
}

func outputJSON(info VersionInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

func outputVerbose(info VersionInfo) error {
	fmt.Printf("CODA version %s\n", info.Version)
	fmt.Printf("Commit: %s\n", info.Commit)
	fmt.Printf("Built: %s\n", info.Date)
	fmt.Printf("Go version: %s\n", info.GoVersion)
	fmt.Printf("Platform: %s\n", info.Platform)

	if len(info.Features) > 0 {
		fmt.Println("\nEnabled features:")
		for _, feature := range info.Features {
			fmt.Printf("  - %s\n", feature)
		}
	}

	if len(info.BuildInfo) > 0 {
		fmt.Println("\nBuild information:")
		for key, value := range info.BuildInfo {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// Check for updates (optional feature)
	if updateInfo := checkForUpdates(); updateInfo != "" {
		fmt.Printf("\n%s\n", updateInfo)
	}

	return nil
}

func checkForUpdates() string {
	// This would check for updates in a real implementation
	// For now, return empty string
	return ""
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	if Version == "dev" {
		return fmt.Sprintf("CODA %s (commit: %s)", Version, getShortCommit())
	}
	return fmt.Sprintf("CODA %s", Version)
}

func getShortCommit() string {
	if len(Commit) >= 7 {
		return Commit[:7]
	}
	return Commit
}
