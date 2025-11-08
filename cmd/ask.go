package cmd

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"github.com/spf13/cobra"
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
		model, err := provider.LanguageModel(ctx, "claude-haiku-4.5-20251001")
		if err != nil {
			HandleError(err, "Failed to initialize Claude model")
		}

		// Create a simple agent with no tools for now
		agent := fantasy.NewAgent(
			model,
			fantasy.WithSystemPrompt("You are a helpful assistant specializing in education and school-related topics. Provide clear, concise, and informative answers."),
		)

		// Generate the response
		result, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: question})
		if err != nil {
			HandleError(err, "Failed to generate response")
		}

		// Print the response
		fmt.Println(result.Response.Content.Text())
	},
}

func init() {
	rootCmd.AddCommand(askCmd)
}
