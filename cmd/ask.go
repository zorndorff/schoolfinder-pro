package cmd

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
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

		// Get API key from environment
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			HandleError(fmt.Errorf("ANTHROPIC_API_KEY not set"), "Failed to initialize Fantasy")
		}

		// Create Fantasy provider for Anthropic
		provider, err := anthropic.New(anthropic.WithAPIKey(apiKey))
		if err != nil {
			HandleError(err, "Failed to create Anthropic provider")
		}

		ctx := context.Background()

		// Create language model with Claude Haiku 4.5
		model, err := provider.LanguageModel(ctx, "claude-haiku-4-5")
		if err != nil {
			HandleError(err, "Failed to initialize Claude model")
		}

		// Create tools from registered Cobra commands
		// Exclude 'serve' and 'ask' commands from tool generation
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

		tools := agent.CreateToolsFromCommands(rootCmd, dataDir, []string{"serve", "ask"}, initDBWrapper, initAIScraperWrapper)

		// Create agent with command tools
		fantasyAgent := fantasy.NewAgent(
			model,
			fantasy.WithSystemPrompt("You are a helpful assistant specializing in education and school-related topics. You have access to tools that can search schools, get school details, and scrape enhanced data from school websites. Use these tools when appropriate to provide accurate, data-backed answers."),
			fantasy.WithTools(tools...),
		)

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
