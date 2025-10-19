package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	stateFilter string
	searchLimit int
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for schools",
	Long: `Search for schools by name, city, district, address, or zip code.
Results are returned as JSON.

Examples:
  schoolfinder search "Lincoln High"
  schoolfinder search --state CA "Lincoln"
  schoolfinder search --limit 10 "Elementary"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		// Initialize database
		db, cleanup, err := InitDB(dataDir)
		if err != nil {
			HandleError(err, "Failed to initialize database")
		}
		defer cleanup()

		// Search schools
		schools, err := db.SearchSchools(query, stateFilter, searchLimit)
		if err != nil {
			HandleError(err, "Failed to search schools")
		}

		// Convert to JSON output format
		output, err := json.MarshalIndent(schools, "", "  ")
		if err != nil {
			HandleError(err, "Failed to encode JSON")
		}

		fmt.Println(string(output))
	},
}

func init() {
	searchCmd.Flags().StringVarP(&stateFilter, "state", "s", "", "Filter by state (e.g., CA, NY)")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 100, "Maximum number of results")
	rootCmd.AddCommand(searchCmd)
}
