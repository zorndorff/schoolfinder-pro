package cmd

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"github.com/spf13/cobra"
	"schoolfinder/internal/agent"
)

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a question using Claude AI via Fantasy",
	Long: `Ask a natural language question and get an AI-powered answer using Claude Haiku 4.5.
This command uses the Fantasy library to interact with Claude.

Requires ANTHROPIC_API_KEY environment variable to be set.

Example:
  schoolfinder ask "What are the most important factors when choosing a school?"
  schoolfinder ask "Explain student-teacher ratio"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the question from arguments
		question := args[0]

		// Wrap the initialization functions to match the agent package's interface
		initDBWrapper := func(dataDir string) (agent.DBInterface, func(), error) {
			db, cleanup, err := InitDB(dataDir)
			if err != nil {
				return nil, nil, err
			}
			// Wrap the DBInterface to match agent.DBInterface
			return &dbInterfaceAdapter{db: db}, cleanup, nil
		}

		initAIScraperWrapper := func(db agent.DBInterface) (agent.AIScraperInterface, error) {
			// Unwrap the db to get the original cmd.DBInterface
			adapter := db.(*dbInterfaceAdapter)
			scraper, err := InitAIScraper(adapter.db)
			if err != nil {
				return nil, err
			}
			return &aiScraperInterfaceAdapter{scraper: scraper}, nil
		}

		// Create the agent using the factory with options
		fantasyAgent, err := agent.NewAskAgent(
			rootCmd,
			agent.WithAPIKeyFromEnv(),
			agent.WithDataDir(dataDir),
			agent.WithDBInitializer(initDBWrapper),
			agent.WithAIScraperInitializer(initAIScraperWrapper),
		)
		if err != nil {
			HandleError(err, "Failed to create agent")
		}

		ctx := context.Background()

		// Generate the response
		result, err := fantasyAgent.Generate(ctx, fantasy.AgentCall{Prompt: question})
		if err != nil {
			HandleError(err, "Failed to generate response")
		}

		// Print the response
		fmt.Println(result.Response.Content.Text())
	},
}

// dbInterfaceAdapter adapts cmd.DBInterface to agent.DBInterface
type dbInterfaceAdapter struct {
	db DBInterface
}

func (a *dbInterfaceAdapter) SearchSchools(query string, state string, limit int) ([]interface{}, error) {
	schools, err := a.db.SearchSchools(query, state, limit)
	if err != nil {
		return nil, err
	}
	// Convert []SchoolData to []interface{}
	result := make([]interface{}, len(schools))
	for i, school := range schools {
		result[i] = school
	}
	return result, nil
}

func (a *dbInterfaceAdapter) GetSchoolByID(ncessch string) (interface{}, error) {
	return a.db.GetSchoolByID(ncessch)
}

func (a *dbInterfaceAdapter) Close() error {
	return a.db.Close()
}

func (a *dbInterfaceAdapter) ExecuteQuery(query string) ([]map[string]interface{}, error) {
	// Cast to DBInterfaceExtended to access ExecuteQuery
	dbExt, ok := a.db.(DBInterfaceExtended)
	if !ok {
		return nil, fmt.Errorf("database does not support ExecuteQuery")
	}
	return dbExt.ExecuteQuery(query)
}

// aiScraperInterfaceAdapter adapts cmd.AIScraperInterface to agent.AIScraperInterface
type aiScraperInterfaceAdapter struct {
	scraper AIScraperInterface
}

func (a *aiScraperInterfaceAdapter) ExtractSchoolDataWithWebSearch(school interface{}) (interface{}, error) {
	// Convert interface{} back to *SchoolData
	schoolData, ok := school.(*SchoolData)
	if !ok {
		return nil, fmt.Errorf("invalid school data type")
	}
	return a.scraper.ExtractSchoolDataWithWebSearch(schoolData)
}

func init() {
	rootCmd.AddCommand(askCmd)
}
