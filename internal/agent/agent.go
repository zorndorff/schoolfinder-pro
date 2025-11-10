package agent

import (
	"context"
	"fmt"
	"os"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
)

const (
	defaultModel        = "claude-haiku-4-5"
	defaultSystemPrompt = "You are a helpful assistant specializing in education and school-related topics. You have access to tools that can search schools, get school details, and scrape enhanced data from school websites. Use these tools when appropriate to provide accurate, data-backed answers."
)

// AgentConfig holds the configuration for creating an ask agent
type AgentConfig struct {
	apiKey       string
	model        string
	systemPrompt string
	dataDir      string
	exclusions   []string
	initDB       InitDBFunc
	initScraper  InitAIScraperFunc
}

// AgentOption is a functional option for configuring the agent
type AgentOption func(*AgentConfig) error

// WithAPIKey sets the Anthropic API key
func WithAPIKey(apiKey string) AgentOption {
	return func(c *AgentConfig) error {
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}
		c.apiKey = apiKey
		return nil
	}
}

// WithAPIKeyFromEnv sets the API key from the ANTHROPIC_API_KEY environment variable
func WithAPIKeyFromEnv() AgentOption {
	return func(c *AgentConfig) error {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		c.apiKey = apiKey
		return nil
	}
}

// WithModel sets the Claude model to use (default: claude-haiku-4-5)
func WithModel(model string) AgentOption {
	return func(c *AgentConfig) error {
		if model == "" {
			return fmt.Errorf("model cannot be empty")
		}
		c.model = model
		return nil
	}
}

// WithSystemPrompt sets a custom system prompt
func WithSystemPrompt(prompt string) AgentOption {
	return func(c *AgentConfig) error {
		c.systemPrompt = prompt
		return nil
	}
}

// WithDataDir sets the data directory for database operations
func WithDataDir(dataDir string) AgentOption {
	return func(c *AgentConfig) error {
		c.dataDir = dataDir
		return nil
	}
}

// WithToolExclusions sets command names to exclude from tool generation
func WithToolExclusions(exclusions []string) AgentOption {
	return func(c *AgentConfig) error {
		c.exclusions = exclusions
		return nil
	}
}

// WithDBInitializer sets the database initialization function
func WithDBInitializer(initDB InitDBFunc) AgentOption {
	return func(c *AgentConfig) error {
		c.initDB = initDB
		return nil
	}
}

// WithAIScraperInitializer sets the AI scraper initialization function
func WithAIScraperInitializer(initScraper InitAIScraperFunc) AgentOption {
	return func(c *AgentConfig) error {
		c.initScraper = initScraper
		return nil
	}
}

// NewAskAgent creates a new Fantasy agent configured for answering school-related questions
// It uses the Options pattern for flexible configuration
//
// The rootCmd parameter should be a *cobra.Command from the cmd package.
// It's defined as interface{} to avoid circular imports.
func NewAskAgent(rootCmd interface{}, opts ...AgentOption) (*fantasy.Agent, error) {
	// Initialize config with defaults
	config := &AgentConfig{
		model:        defaultModel,
		systemPrompt: defaultSystemPrompt,
		exclusions:   []string{"serve", "ask"},
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Validate required fields
	if config.apiKey == "" {
		return nil, fmt.Errorf("API key is required (use WithAPIKey or WithAPIKeyFromEnv)")
	}
	if config.initDB == nil {
		return nil, fmt.Errorf("database initializer is required (use WithDBInitializer)")
	}
	if config.initScraper == nil {
		return nil, fmt.Errorf("AI scraper initializer is required (use WithAIScraperInitializer)")
	}

	// Create Fantasy provider for Anthropic
	provider, err := anthropic.New(anthropic.WithAPIKey(config.apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic provider: %w", err)
	}

	ctx := context.Background()

	// Create language model
	model, err := provider.LanguageModel(ctx, config.model)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Claude model: %w", err)
	}

	// Create tools from registered commands
	agentTools := CreateToolsFromCommands(
		rootCmd,
		config.dataDir,
		config.exclusions,
		config.initDB,
		config.initScraper,
	)

	// Create and return the agent
	agent := fantasy.NewAgent(
		model,
		fantasy.WithSystemPrompt(config.systemPrompt),
		fantasy.WithTools(agentTools...),
	)

	return agent, nil
}

// GenerateResponse is a convenience function that creates an agent and generates a response in one call
func GenerateResponse(ctx context.Context, question string, rootCmd interface{}, opts ...AgentOption) (string, error) {
	agent, err := NewAskAgent(rootCmd, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to create agent: %w", err)
	}

	result, err := agent.Generate(ctx, fantasy.AgentCall{Prompt: question})
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return result.Response.Content.Text(), nil
}
