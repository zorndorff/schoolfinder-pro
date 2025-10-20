package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var (
	port int
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the web server",
		Long: `Start the HTTP web server with HTMX interface.

The web server provides a browser-based interface for searching and exploring
school data, with the same functionality as the TUI plus API endpoints.`,
		Run: func(cmd *cobra.Command, args []string) {
			runServe()
		},
	}
)

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to run the server on")
}

func runServe() {
	// Initialize database
	db, cleanup, err := InitDB(dataDir)
	if err != nil {
		HandleError(err, "Failed to initialize database")
	}
	defer cleanup()

	fmt.Printf("Starting School Finder web server...\n")
	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("Port: %d\n\n", port)

	// Start the server (this will be implemented in main.go)
	if err := StartServer(db, port, dataDir); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}

// StartServer is set by main package
var StartServer func(db DBInterface, port int, dataDir string) error
