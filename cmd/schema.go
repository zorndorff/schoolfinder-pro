package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// SchemaOutput represents the schema information for a table
type SchemaOutput struct {
	TableName   string       `json:"table_name"`
	ColumnCount int          `json:"column_count"`
	Columns     []ColumnInfo `json:"columns"`
}

// ColumnInfo represents information about a single column
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Retrieve a summary of the DuckDB database schema",
	Long: `Retrieve a summary of the local DuckDB database schema.
This command returns information about all tables and their columns in the database.

Examples:
  schoolfinder schema`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize database
		db, cleanup, err := InitDB(dataDir)
		if err != nil {
			HandleError(err, "Failed to initialize database")
		}
		defer cleanup()

		// Get schema information for all tables
		tables := []string{"directory", "teachers", "enrollment", "ai_scraper_cache", "naep_cache"}
		schemas := make([]SchemaOutput, 0, len(tables))

		for _, tableName := range tables {
			schema, err := getTableSchema(db, tableName)
			if err != nil {
				// Skip tables that don't exist
				continue
			}
			schemas = append(schemas, schema)
		}

		// Convert to JSON output
		output, err := json.MarshalIndent(schemas, "", "  ")
		if err != nil {
			HandleError(err, "Failed to encode JSON")
		}

		fmt.Println(string(output))
	},
}

// getTableSchema retrieves schema information for a specific table
func getTableSchema(db DBInterface, tableName string) (SchemaOutput, error) {
	// Cast to the extended interface to access ExecuteQuery
	dbExt, ok := db.(DBInterfaceExtended)
	if !ok {
		return SchemaOutput{}, fmt.Errorf("database does not support ExecuteQuery")
	}

	query := fmt.Sprintf("PRAGMA table_info('%s')", tableName)
	rows, err := dbExt.ExecuteQuery(query)
	if err != nil {
		return SchemaOutput{}, fmt.Errorf("failed to get schema for table %s: %w", tableName, err)
	}

	schema := SchemaOutput{
		TableName: tableName,
		Columns:   []ColumnInfo{},
	}

	for _, row := range rows {
		// PRAGMA table_info returns: cid, name, type, notnull, dflt_value, pk
		if len(row) >= 3 {
			name, _ := row["name"].(string)
			colType, _ := row["type"].(string)
			notnull, _ := row["notnull"].(string)

			nullable := "YES"
			if notnull == "1" {
				nullable = "NO"
			}

			schema.Columns = append(schema.Columns, ColumnInfo{
				Name:     name,
				Type:     colType,
				Nullable: nullable,
			})
		}
	}

	schema.ColumnCount = len(schema.Columns)

	return schema, nil
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
