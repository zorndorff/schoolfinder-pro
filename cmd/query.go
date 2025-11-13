package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var queryString string

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query the database (DuckDB SQL)",
	Long: `Execute the requested QUERY against the DuckDB database.
The query can be any valid DuckDB SQL query, including SELECT, DESCRIBE, SHOW TABLES, etc.

Examples:
  schoolfinder query --sql "SELECT * FROM directory LIMIT 5"
  schoolfinder query --sql "SELECT COUNT(*) as total FROM directory"
  schoolfinder query --sql "SHOW TABLES"`,
	Run: func(cmd *cobra.Command, args []string) {
		if queryString == "" {
			HandleError(fmt.Errorf("query is required"), "Missing query parameter")
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

		// Execute the query
		rows, err := dbExt.ExecuteQuery(queryString)
		if err != nil {
			HandleError(err, "Failed to execute query")
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
	queryCmd.Flags().StringVarP(&queryString, "sql", "q", "", "SQL query to execute (required)")
	_ = queryCmd.MarkFlagRequired("sql")
	rootCmd.AddCommand(queryCmd)
}
