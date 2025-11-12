package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var queryOrTable string

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Summarize the contents of a table or query",
	Long: `The SUMMARIZE command can be used to easily compute a number of aggregates over a table or a query.
The SUMMARIZE command launches a query that computes a number of aggregates over all columns
(min, max, approx_unique, avg, std, q25, q50, q75, count), and returns these along with the column name,
column type, and the percentage of NULL values in the column.
Note that the quantiles and percentiles are approximate values.

To summarize the contents of a table, pass a table name:
  schoolfinder summarize --table directory

To summarize a query, pass a query:
  schoolfinder summarize --query "SELECT * FROM directory WHERE ST = 'CA'"

Examples:
  schoolfinder summarize --table directory
  schoolfinder summarize --table enrollment
  schoolfinder summarize --query "SELECT * FROM directory WHERE ST = 'CA' LIMIT 1000"`,
	Run: func(cmd *cobra.Command, args []string) {
		if queryOrTable == "" {
			HandleError(fmt.Errorf("table or query is required"), "Missing parameter")
		}

		// Initialize database
		db, cleanup, err := InitDB(dataDir)
		if err != nil {
			HandleError(err, "Failed to initialize database")
		}
		defer cleanup()

		// Cast to the extended interface to access ExecuteQuery
		dbExt, ok := db.(DBInterfaceExtended)
		if !ok {
			HandleError(fmt.Errorf("database does not support ExecuteQuery"), "Unsupported operation")
		}

		// Build the SUMMARIZE query
		summarizeQuery := fmt.Sprintf("SUMMARIZE %s", queryOrTable)

		// Execute the query
		rows, err := dbExt.ExecuteQuery(summarizeQuery)
		if err != nil {
			HandleError(err, "Failed to execute summarize query")
		}

		// Convert to JSON output
		output, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			HandleError(err, "Failed to encode JSON")
		}

		fmt.Println(string(output))
	},
}

func init() {
	summarizeCmd.Flags().StringVarP(&queryOrTable, "table", "t", "", "Table name or query to summarize (required)")
	summarizeCmd.Flags().StringVarP(&queryOrTable, "query", "q", "", "Query to summarize (alias for --table)")
	summarizeCmd.MarkFlagRequired("table")
	rootCmd.AddCommand(summarizeCmd)
}
