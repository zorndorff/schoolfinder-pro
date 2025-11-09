package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/spf13/cobra"
)

// Input types for each tool
type SearchInput struct {
	Query string `json:"query" jsonschema:"required,description=Search query for school name, city, district, address, or zip code"`
	State string `json:"state,omitempty" jsonschema:"description=Optional state filter (e.g., CA, NY)"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=Maximum number of results (default: 100)"`
}

type DetailsInput struct {
	SchoolID string `json:"school_id" jsonschema:"required,description=The NCESSCH ID of the school"`
}

type ScrapeInput struct {
	SchoolID string `json:"school_id" jsonschema:"required,description=The NCESSCH ID of the school to scrape enhanced data for"`
}

type GenericInput struct {
	Args string `json:"args,omitempty" jsonschema:"description=Arguments for the command"`
}

// DBInterface defines the database operations needed for tools
type DBInterface interface {
	SearchSchools(query string, state string, limit int) ([]interface{}, error)
	GetSchoolByID(ncessch string) (interface{}, error)
	Close() error
}

// AIScraperInterface defines the AI scraper operations needed for tools
type AIScraperInterface interface {
	ExtractSchoolDataWithWebSearch(school interface{}) (interface{}, error)
}

// InitDBFunc is the function signature for database initialization
type InitDBFunc func(dataDir string) (DBInterface, func(), error)

// InitAIScraperFunc is the function signature for AI scraper initialization
type InitAIScraperFunc func(db DBInterface) (AIScraperInterface, error)

// CreateToolsFromCommands creates Fantasy tools from all registered Cobra commands
// except for the specified exclusions (e.g., "serve", "ask")
func CreateToolsFromCommands(
	rootCmd *cobra.Command,
	dataDir string,
	exclusions []string,
	initDB InitDBFunc,
	initAIScraper InitAIScraperFunc,
) []fantasy.AgentTool {
	var tools []fantasy.AgentTool

	// Iterate through all registered commands
	for _, cobraCmd := range rootCmd.Commands() {
		// Check if command should be excluded
		skip := false
		for _, excl := range exclusions {
			if cobraCmd.Use == excl || strings.HasPrefix(cobraCmd.Use, excl) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Create a tool for this command
		tool := createToolForCommand(cobraCmd, dataDir, initDB, initAIScraper)
		if tool != nil {
			tools = append(tools, tool)
		}
	}

	return tools
}

// createToolForCommand creates a Fantasy tool from a Cobra command
func createToolForCommand(
	cobraCmd *cobra.Command,
	dataDir string,
	initDB InitDBFunc,
	initAIScraper InitAIScraperFunc,
) fantasy.AgentTool {
	// Extract the command name (first word in Use)
	cmdName := strings.Split(cobraCmd.Use, " ")[0]

	// Create tool description from command's Short description
	description := cobraCmd.Short
	if description == "" {
		description = fmt.Sprintf("Execute the %s command", cmdName)
	}

	switch cmdName {
	case "search":
		return fantasy.NewAgentTool(
			cmdName,
			description,
			func(ctx context.Context, input SearchInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				// Validate input
				if input.Query == "" {
					return fantasy.NewTextErrorResponse("query parameter is required"), nil
				}

				// Set default limit if not provided
				if input.Limit == 0 {
					input.Limit = 100
				}

				// Initialize database
				db, cleanup, err := initDB(dataDir)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to initialize database: %v", err)), nil
				}
				defer cleanup()

				// Execute search
				schools, err := db.SearchSchools(input.Query, input.State, input.Limit)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to search schools: %v", err)), nil
				}

				// Convert result to JSON
				jsonBytes, err := json.MarshalIndent(schools, "", "  ")
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to encode result as JSON: %v", err)), nil
				}

				return fantasy.NewTextResponse(string(jsonBytes)), nil
			},
		)

	case "details":
		return fantasy.NewAgentTool(
			cmdName,
			description,
			func(ctx context.Context, input DetailsInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				// Validate input
				if input.SchoolID == "" {
					return fantasy.NewTextErrorResponse("school_id parameter is required"), nil
				}

				// Initialize database
				db, cleanup, err := initDB(dataDir)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to initialize database: %v", err)), nil
				}
				defer cleanup()

				// Get school details
				school, err := db.GetSchoolByID(input.SchoolID)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to get school details: %v", err)), nil
				}

				if school == nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("no school found with ID: %s", input.SchoolID)), nil
				}

				// Convert result to JSON
				jsonBytes, err := json.MarshalIndent(school, "", "  ")
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to encode result as JSON: %v", err)), nil
				}

				return fantasy.NewTextResponse(string(jsonBytes)), nil
			},
		)

	case "scrape":
		return fantasy.NewAgentTool(
			cmdName,
			description,
			func(ctx context.Context, input ScrapeInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				// Validate input
				if input.SchoolID == "" {
					return fantasy.NewTextErrorResponse("school_id parameter is required"), nil
				}

				// Initialize database
				db, cleanup, err := initDB(dataDir)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to initialize database: %v", err)), nil
				}
				defer cleanup()

				// Get school details first
				school, err := db.GetSchoolByID(input.SchoolID)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to get school details: %v", err)), nil
				}

				if school == nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("no school found with ID: %s", input.SchoolID)), nil
				}

				// Initialize AI scraper
				aiScraper, err := initAIScraper(db)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to initialize AI scraper: %v", err)), nil
				}

				// Extract enhanced data
				enhancedData, err := aiScraper.ExtractSchoolDataWithWebSearch(school)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to scrape school data: %v", err)), nil
				}

				// Convert result to JSON
				jsonBytes, err := json.MarshalIndent(enhancedData, "", "  ")
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to encode result as JSON: %v", err)), nil
				}

				return fantasy.NewTextResponse(string(jsonBytes)), nil
			},
		)

	default:
		// For unsupported commands, create a tool that returns an error
		return fantasy.NewAgentTool(
			cmdName,
			description,
			func(ctx context.Context, input GenericInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("unsupported command: %s", cmdName)), nil
			},
		)
	}
}
