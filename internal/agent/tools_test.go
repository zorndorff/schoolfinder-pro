package agent

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

// Mock implementations for testing
type mockDB struct{}

func (m *mockDB) SearchSchools(query string, state string, limit int) ([]interface{}, error) {
	return []interface{}{"school1", "school2"}, nil
}

func (m *mockDB) GetSchoolByID(ncessch string) (interface{}, error) {
	return "school_details", nil
}

func (m *mockDB) Close() error {
	return nil
}

type mockAIScraper struct{}

func (m *mockAIScraper) ExtractSchoolDataWithWebSearch(school interface{}) (interface{}, error) {
	return "enhanced_data", nil
}

// Mock initialization functions
func mockInitDB(dataDir string) (DBInterface, func(), error) {
	return &mockDB{}, func() {}, nil
}

func mockInitAIScraper(db DBInterface) (AIScraperInterface, error) {
	return &mockAIScraper{}, nil
}

// TestCreateToolsFromCommands tests that Cobra commands are correctly converted to Fantasy tools
func TestCreateToolsFromCommands(t *testing.T) {
	// Create a test root command
	rootCmd := &cobra.Command{
		Use:   "testapp",
		Short: "Test application",
	}

	// Add test commands
	helloCmd := &cobra.Command{
		Use:   "hello_world",
		Short: "Say hello to the world",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	goodbyeCmd := &cobra.Command{
		Use:   "goodbye_world",
		Short: "Say goodbye to the world",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	excludedCmd := &cobra.Command{
		Use:   "excluded",
		Short: "This command should be excluded",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	rootCmd.AddCommand(helloCmd)
	rootCmd.AddCommand(goodbyeCmd)
	rootCmd.AddCommand(excludedCmd)

	// Test 1: Create tools without exclusions
	t.Run("CreateAllTools", func(t *testing.T) {
		tools := CreateToolsFromCommands(rootCmd, "/tmp/test", []string{}, mockInitDB, mockInitAIScraper)

		// Should have 3 tools (hello_world, goodbye_world, excluded)
		if len(tools) != 3 {
			t.Errorf("Expected 3 tools, got %d", len(tools))
		}
	})

	// Test 2: Create tools with exclusions
	t.Run("CreateToolsWithExclusions", func(t *testing.T) {
		tools := CreateToolsFromCommands(rootCmd, "/tmp/test", []string{"excluded"}, mockInitDB, mockInitAIScraper)

		// Should have 2 tools (hello_world, goodbye_world)
		if len(tools) != 2 {
			t.Errorf("Expected 2 tools after exclusions, got %d", len(tools))
		}
	})

	// Test 3: Verify tools are created (check they're not nil)
	t.Run("VerifyToolsNotNil", func(t *testing.T) {
		tools := CreateToolsFromCommands(rootCmd, "/tmp/test", []string{"excluded"}, mockInitDB, mockInitAIScraper)

		for i, tool := range tools {
			if tool == nil {
				t.Errorf("Tool at index %d is nil", i)
			}
		}
	})

	// Test 4: Verify exclusion with prefix matching
	t.Run("ExcludeWithPrefixMatch", func(t *testing.T) {
		// Add command with arguments in Use field
		cmdWithArgs := &cobra.Command{
			Use:   "hello_world [args]",
			Short: "Test command with args",
			Run:   func(cmd *cobra.Command, args []string) {},
		}
		testRoot := &cobra.Command{Use: "test"}
		testRoot.AddCommand(cmdWithArgs)

		tools := CreateToolsFromCommands(testRoot, "/tmp/test", []string{"hello_world"}, mockInitDB, mockInitAIScraper)

		// Should have 0 tools (hello_world excluded by prefix match)
		if len(tools) != 0 {
			t.Errorf("Expected 0 tools with prefix exclusion, got %d", len(tools))
		}
	})
}

// TestCreateToolForCommand tests individual tool creation
func TestCreateToolForCommand(t *testing.T) {
	// Create a test command
	testCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for schools",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	// Create tool from command
	tool := createToolForCommand(testCmd, "/tmp/test", mockInitDB, mockInitAIScraper)

	// Verify tool is created
	if tool == nil {
		t.Fatal("Expected tool to be created, got nil")
	}

	// Test tool execution (this tests the search case)
	t.Run("ExecuteSearchTool", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"query": "test school",
			"state": "CA",
			"limit": float64(10),
		}

		result, err := tool.Function()(ctx, params)
		if err != nil {
			t.Errorf("Tool execution failed: %v", err)
		}

		if result == "" {
			t.Error("Expected non-empty result from tool execution")
		}
	})

	// Test tool execution with missing required parameter
	t.Run("ExecuteSearchToolMissingQuery", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"state": "CA",
		}

		_, err := tool.Function()(ctx, params)
		if err == nil {
			t.Error("Expected error for missing query parameter, got nil")
		}
	})
}

// TestDetailsToolExecution tests the details command tool
func TestDetailsToolExecution(t *testing.T) {
	detailsCmd := &cobra.Command{
		Use:   "details [school-id]",
		Short: "Get detailed information about a school",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	tool := createToolForCommand(detailsCmd, "/tmp/test", mockInitDB, mockInitAIScraper)

	if tool == nil {
		t.Fatal("Expected tool to be created, got nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{
		"school_id": "12345",
	}

	result, err := tool.Function()(ctx, params)
	if err != nil {
		t.Errorf("Details tool execution failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result from details tool execution")
	}
}

// TestScrapeToolExecution tests the scrape command tool
func TestScrapeToolExecution(t *testing.T) {
	scrapeCmd := &cobra.Command{
		Use:   "scrape [school-id]",
		Short: "Scrape enhanced data from school website using AI",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	tool := createToolForCommand(scrapeCmd, "/tmp/test", mockInitDB, mockInitAIScraper)

	if tool == nil {
		t.Fatal("Expected tool to be created, got nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{
		"school_id": "12345",
	}

	result, err := tool.Function()(ctx, params)
	if err != nil {
		t.Errorf("Scrape tool execution failed: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result from scrape tool execution")
	}
}

// TestUnsupportedCommand tests that unsupported commands return an error
func TestUnsupportedCommand(t *testing.T) {
	unsupportedCmd := &cobra.Command{
		Use:   "unsupported",
		Short: "This is an unsupported command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	tool := createToolForCommand(unsupportedCmd, "/tmp/test", mockInitDB, mockInitAIScraper)

	if tool == nil {
		t.Fatal("Expected tool to be created, got nil")
	}

	ctx := context.Background()
	params := map[string]interface{}{}

	_, err := tool.Function()(ctx, params)
	if err == nil {
		t.Error("Expected error for unsupported command, got nil")
	}

	expectedMsg := "unsupported command: unsupported"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestHelloWorldGoodbyeWorldCommands tests the specific scenario from the user's request
func TestHelloWorldGoodbyeWorldCommands(t *testing.T) {
	// Create a test root command
	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "My test application",
	}

	// Add hello_world command
	helloWorldCmd := &cobra.Command{
		Use:   "hello_world",
		Short: "Greet the world",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	// Add goodbye_world command
	goodbyeWorldCmd := &cobra.Command{
		Use:   "goodbye_world",
		Short: "Bid farewell to the world",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	rootCmd.AddCommand(helloWorldCmd)
	rootCmd.AddCommand(goodbyeWorldCmd)

	// Generate tools
	tools := CreateToolsFromCommands(rootCmd, "/tmp/test", []string{}, mockInitDB, mockInitAIScraper)

	// Verify we have exactly 2 tools
	if len(tools) != 2 {
		t.Fatalf("Expected 2 tools (hello_world and goodbye_world), got %d", len(tools))
	}

	// Since we can't easily check the tool names without accessing private fields,
	// we'll verify the tools are not nil and can be executed (though they'll error as unsupported)
	for i, tool := range tools {
		if tool == nil {
			t.Errorf("Tool %d is nil", i)
		}
	}

	t.Log("Successfully created tools for hello_world and goodbye_world commands")
}
