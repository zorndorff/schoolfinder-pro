package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var detailsCmd = &cobra.Command{
	Use:   "details [school-id]",
	Short: "Get detailed information about a school",
	Long: `Get detailed information about a specific school by NCESSCH ID.
Returns school data as JSON.

Example:
  schoolfinder details 060207001814`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		schoolID := args[0]

		// Initialize database
		db, cleanup, err := InitDB(dataDir)
		if err != nil {
			HandleError(err, "Failed to initialize database")
		}
		defer cleanup()

		// Get school details
		school, err := db.GetSchoolByID(schoolID)
		if err != nil {
			HandleError(err, "Failed to get school details")
		}

		if school == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "No school found with ID: %s\n", schoolID)
			return
		}

		// Convert to JSON output format
		output, err := json.MarshalIndent(school, "", "  ")
		if err != nil {
			HandleError(err, "Failed to encode JSON")
		}

		fmt.Println(string(output))
	},
}

func init() {
	rootCmd.AddCommand(detailsCmd)
}
