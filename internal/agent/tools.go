package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/spf13/cobra"
	"schoolfinder/cmd"
)

// CreateToolsFromCommands creates Fantasy tools from all registered Cobra commands
// except for the specified exclusions (e.g., "serve", "ask")
func CreateToolsFromCommands(rootCmd *cobra.Command, dataDir string, exclusions []string) []fantasy.Tool {
	var tools []fantasy.Tool

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
		tool := createToolForCommand(cobraCmd, dataDir)
		if tool != nil {
			tools = append(tools, tool)
		}
	}

	return tools
}

// createToolForCommand creates a Fantasy tool from a Cobra command
func createToolForCommand(cobraCmd *cobra.Command, dataDir string) fantasy.Tool {
	// Extract the command name (first word in Use)
	cmdName := strings.Split(cobraCmd.Use, " ")[0]

	// Create tool description from command's Short description
	description := cobraCmd.Short
	if description == "" {
		description = fmt.Sprintf("Execute the %s command", cmdName)
	}

	// Create the tool function that calls the underlying functionality directly
	toolFunc := func(ctx context.Context, params map[string]interface{}) (string, error) {
		var result interface{}
		var err error

		switch cmdName {
		case "search":
			// Extract search parameters
			query, ok := params["query"].(string)
			if !ok || query == "" {
				return "", fmt.Errorf("query parameter is required")
			}

			state := ""
			if s, ok := params["state"].(string); ok {
				state = s
			}

			limit := 100
			if l, ok := params["limit"].(float64); ok {
				limit = int(l)
			}

			// Initialize database
			db, cleanup, err := cmd.InitDB(dataDir)
			if err != nil {
				return "", fmt.Errorf("failed to initialize database: %v", err)
			}
			defer cleanup()

			// Execute search
			schools, err := db.SearchSchools(query, state, limit)
			if err != nil {
				return "", fmt.Errorf("failed to search schools: %v", err)
			}

			result = schools

		case "details":
			// Extract school ID parameter
			schoolID, ok := params["school_id"].(string)
			if !ok || schoolID == "" {
				return "", fmt.Errorf("school_id parameter is required")
			}

			// Initialize database
			db, cleanup, err := cmd.InitDB(dataDir)
			if err != nil {
				return "", fmt.Errorf("failed to initialize database: %v", err)
			}
			defer cleanup()

			// Get school details
			school, err := db.GetSchoolByID(schoolID)
			if err != nil {
				return "", fmt.Errorf("failed to get school details: %v", err)
			}

			if school == nil {
				return "", fmt.Errorf("no school found with ID: %s", schoolID)
			}

			result = school

		case "scrape":
			// Extract school ID parameter
			schoolID, ok := params["school_id"].(string)
			if !ok || schoolID == "" {
				return "", fmt.Errorf("school_id parameter is required")
			}

			// Initialize database
			db, cleanup, err := cmd.InitDB(dataDir)
			if err != nil {
				return "", fmt.Errorf("failed to initialize database: %v", err)
			}
			defer cleanup()

			// Get school details first
			school, err := db.GetSchoolByID(schoolID)
			if err != nil {
				return "", fmt.Errorf("failed to get school details: %v", err)
			}

			if school == nil {
				return "", fmt.Errorf("no school found with ID: %s", schoolID)
			}

			// Initialize AI scraper
			aiScraper, err := cmd.InitAIScraper(db)
			if err != nil {
				return "", fmt.Errorf("failed to initialize AI scraper: %v", err)
			}

			// Extract enhanced data
			enhancedData, err := aiScraper.ExtractSchoolDataWithWebSearch(school)
			if err != nil {
				return "", fmt.Errorf("failed to scrape school data: %v", err)
			}

			result = enhancedData

		default:
			return "", fmt.Errorf("unsupported command: %s", cmdName)
		}

		// Convert result to JSON
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to encode result as JSON: %v", err)
		}

		return string(jsonBytes), nil
	}

	// Create parameter schema based on command
	var paramSchema map[string]interface{}

	switch cmdName {
	case "search":
		paramSchema = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for school name, city, district, address, or zip code",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Optional state filter (e.g., CA, NY)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 100)",
				},
			},
			"required": []string{"query"},
		}
	case "details":
		paramSchema = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"school_id": map[string]interface{}{
					"type":        "string",
					"description": "The NCESSCH ID of the school",
				},
			},
			"required": []string{"school_id"},
		}
	case "scrape":
		paramSchema = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"school_id": map[string]interface{}{
					"type":        "string",
					"description": "The NCESSCH ID of the school to scrape enhanced data for",
				},
			},
			"required": []string{"school_id"},
		}
	default:
		// Generic schema for other commands
		paramSchema = map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"args": map[string]interface{}{
					"type":        "string",
					"description": "Arguments for the command",
				},
			},
		}
	}

	return fantasy.NewAgentTool(
		cmdName,
		description,
		toolFunc,
		fantasy.WithParameters(paramSchema),
	)
}
