package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var scrapeCmd = &cobra.Command{
	Use:   "scrape [school-id]",
	Short: "Scrape enhanced data from school website using AI",
	Long: `Scrape enhanced data from a school's website using Claude AI with web search.
Extracts staff contacts, programs, facilities, and other information.
Returns enhanced data as JSON.

Requires ANTHROPIC_API_KEY environment variable to be set.

Example:
  schoolfinder scrape 060207001814`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		schoolID := args[0]

		db, cleanup, err := InitDB(dataDir)
		if err != nil {
			HandleError(err, "Failed to initialize database")
		}
		defer cleanup()

		school, err := db.GetSchoolByID(schoolID)
		if err != nil {
			HandleError(err, "Failed to get school details")
		}

		if school == nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "No school found with ID: %s\n", schoolID)
			return
		}

		aiScraper, err := InitAIScraper(db)
		if err != nil {
			HandleError(err, "Failed to initialize AI scraper")
		}

		enhancedData, err := aiScraper.ExtractSchoolDataWithWebSearch(school)
		if err != nil {
			HandleError(err, "Failed to scrape school data")
		}

		output, err := json.MarshalIndent(enhancedData, "", "  ")
		if err != nil {
			HandleError(err, "Failed to encode JSON")
		}

		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(scrapeCmd)
}
