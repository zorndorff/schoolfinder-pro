package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	dataDir string
	rootCmd = &cobra.Command{
		Use:   "schoolfinder",
		Short: "School Finder - Search and explore school data",
		Long: `School Finder is a CLI/TUI application for searching and exploring
school data from the Common Core of Data (CCD).

When run without commands, it launches an interactive TUI.
Use subcommands for CLI mode with JSON output.`,
		Run: func(cmd *cobra.Command, args []string) {
			// No subcommand specified - launch TUI
			LaunchTUI(dataDir)
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&dataDir, "data-dir", "d", "tmpdata/", "Directory containing CSV data files")
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetupLogger creates and configures the application logger
func SetupLogger(dataDir string) error {
	logPath := filepath.Join(dataDir, "err.log")

	// Create log file
	_, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// We'll use the global logger from main package
	// The main package will need to export setupLogger or we pass it here
	return nil
}
