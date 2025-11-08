package main

import (
	"testing"
)

// TestEnrichScore tests the enrichScore function
func TestEnrichScore(t *testing.T) {
	handler := &WebHandler{}

	testCases := []struct {
		name         string
		score        NAEPScore
		data         *NAEPData
		useDistrict  bool
		expectedProf float64
		expectedAdv  float64
	}{
		{
			name:         "Grade 4 Mathematics with 40% proficient",
			score:        MockNAEPScore("mathematics", 4, 2022, 240.0, 40.0),
			data:         MockNAEPDataMinimal("123456789012", "CA"),
			useDistrict:  false,
			expectedProf: 36.0, // 40 * 0.9 = 36 (90% of proficient+ is proficient)
			expectedAdv:  4.0,  // 40 * 0.1 = 4 (10% of proficient+ is advanced)
		},
		{
			name:         "Grade 8 Reading with 32% proficient",
			score:        MockNAEPScore("reading", 8, 2022, 265.0, 32.0),
			data:         MockNAEPData("123456789012", "CA", "", false, false), // Use full mock data
			useDistrict:  false,
			expectedProf: 28.8, // 32 * 0.9 = 28.8
			expectedAdv:  3.2,  // 32 * 0.1 = 3.2
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handler.enrichScore(tc.score, tc.data, tc.useDistrict)

			if result.Subject != tc.score.Subject {
				t.Errorf("Expected subject %s, got %s", tc.score.Subject, result.Subject)
			}

			if result.Grade != tc.score.Grade {
				t.Errorf("Expected grade %d, got %d", tc.score.Grade, result.Grade)
			}

			if result.ProficientPct != tc.expectedProf {
				t.Errorf("Expected proficient %.2f, got %.2f", tc.expectedProf, result.ProficientPct)
			}

			if result.AdvancedPct != tc.expectedAdv {
				t.Errorf("Expected advanced %.2f, got %.2f", tc.expectedAdv, result.AdvancedPct)
			}

			// Basic should be ~35%
			if result.BasicPct != 35.0 {
				t.Errorf("Expected basic 35.0, got %.2f", result.BasicPct)
			}

			// Below Basic should be remainder
			expectedBelowBasic := 100.0 - tc.score.AtProficient - 35.0
			if result.BelowBasicPct != expectedBelowBasic {
				t.Errorf("Expected below basic %.2f, got %.2f", expectedBelowBasic, result.BelowBasicPct)
			}
		})
	}
}

// TestAddNationalComparison tests the addNationalComparison function
func TestAddNationalComparison(t *testing.T) {
	handler := &WebHandler{}

	nationalScores := map[string]*NAEPScoreView{
		"mathematics-4": {
			NAEPScore: NAEPScore{
				Subject:      "mathematics",
				Grade:        4,
				AtProficient: 35.0,
			},
		},
		"reading-8": {
			NAEPScore: NAEPScore{
				Subject:      "reading",
				Grade:        8,
				AtProficient: 30.0,
			},
		},
	}

	testCases := []struct {
		name            string
		score           NAEPScoreView
		expectedCompare string
		shouldHaveNat   bool
	}{
		{
			name: "State score above national",
			score: NAEPScoreView{
				NAEPScore: NAEPScore{
					Subject:      "mathematics",
					Grade:        4,
					AtProficient: 40.0,
				},
			},
			expectedCompare: "Above",
			shouldHaveNat:   true,
		},
		{
			name: "State score below national",
			score: NAEPScoreView{
				NAEPScore: NAEPScore{
					Subject:      "reading",
					Grade:        8,
					AtProficient: 25.0,
				},
			},
			expectedCompare: "Below",
			shouldHaveNat:   true,
		},
		{
			name: "No matching national score",
			score: NAEPScoreView{
				NAEPScore: NAEPScore{
					Subject:      "science",
					Grade:        4,
					AtProficient: 25.0,
				},
			},
			expectedCompare: "",
			shouldHaveNat:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := tc.score
			handler.addNationalComparison(&score, nationalScores)

			if tc.shouldHaveNat {
				if score.NationalScore == nil {
					t.Error("Expected national score to be set, got nil")
				}
				if score.NationalCompare != tc.expectedCompare {
					t.Errorf("Expected comparison %s, got %s", tc.expectedCompare, score.NationalCompare)
				}
			} else {
				if score.NationalScore != nil {
					t.Error("Expected no national score, but one was set")
				}
			}
		})
	}
}

// TestEnrichNAEPData tests the full enrichNAEPData function
func TestEnrichNAEPData(t *testing.T) {
	handler := &WebHandler{}

	// Use mock data with national scores for comparison
	testData := MockNAEPData("123456789012", "CA", "", false, true)

	t.Run("State scores enrichment", func(t *testing.T) {
		result := handler.enrichNAEPData(testData)

		// Check basic fields
		if result.NCESSCH != testData.NCESSCH {
			t.Errorf("Expected NCESSCH %s, got %s", testData.NCESSCH, result.NCESSCH)
		}

		if result.State != testData.State {
			t.Errorf("Expected State %s, got %s", testData.State, result.State)
		}

		if result.UseDistrict {
			t.Error("Expected UseDistrict to be false")
		}

		// Check that all state scores are enriched
		expectedCount := len(testData.StateScores)
		if len(result.StateScores) != expectedCount {
			t.Errorf("Expected %d state scores, got %d", expectedCount, len(result.StateScores))
		}

		// Verify enrichment of first score
		if len(result.StateScores) > 0 {
			firstScore := result.StateScores[0]
			if firstScore.ProficientPct == 0 {
				t.Error("Expected ProficientPct to be calculated")
			}
			if firstScore.AdvancedPct == 0 {
				t.Error("Expected AdvancedPct to be calculated")
			}

			// Check national comparison is set
			if firstScore.NationalScore == nil {
				t.Error("Expected national comparison to be set")
			}
			if firstScore.NationalCompare == "" {
				t.Error("Expected national comparison string to be set")
			}
		}
	})

	t.Run("Grade grouping", func(t *testing.T) {
		result := handler.enrichNAEPData(testData)

		// Check Grade 4 scores exist
		if len(result.Grade4Scores) == 0 {
			t.Error("Expected some Grade 4 scores")
		}

		for _, score := range result.Grade4Scores {
			if score.Grade != 4 {
				t.Errorf("Expected Grade 4, got %d", score.Grade)
			}
		}

		// Check Grade 8 scores exist
		if len(result.Grade8Scores) == 0 {
			t.Error("Expected some Grade 8 scores")
		}

		for _, score := range result.Grade8Scores {
			if score.Grade != 8 {
				t.Errorf("Expected Grade 8, got %d", score.Grade)
			}
		}
	})

	t.Run("National scores lookup map", func(t *testing.T) {
		result := handler.enrichNAEPData(testData)

		// Check that national lookup map is populated
		if len(result.NationalByKey) == 0 {
			t.Error("Expected national scores in map")
		}

		// Check specific keys exist
		if _, ok := result.NationalByKey["mathematics-4"]; !ok {
			t.Error("Expected 'mathematics-4' key in national map")
		}

		if _, ok := result.NationalByKey["reading-8"]; !ok {
			t.Error("Expected 'reading-8' key in national map")
		}

		// Verify data in map
		if mathScore, ok := result.NationalByKey["mathematics-4"]; ok {
			if mathScore.AtProficient == 0 {
				t.Error("Expected non-zero proficient percentage for mathematics-4")
			}
		}
	})
}

// TestEnrichNAEPDataWithDistrict tests enrichment with district scores
func TestEnrichNAEPDataWithDistrict(t *testing.T) {
	handler := &WebHandler{}

	// Use mock data with district and national scores
	testData := MockNAEPData("123456789012", "CA", "Los Angeles", true, true)

	result := handler.enrichNAEPData(testData)

	if !result.UseDistrict {
		t.Error("Expected UseDistrict to be true when district scores are present")
	}

	if result.District != "Los Angeles" {
		t.Errorf("Expected District 'Los Angeles', got %s", result.District)
	}

	// Should use district scores for grade grouping
	if len(result.Grade4Scores) == 0 {
		t.Error("Expected some Grade 4 scores from district")
	}

	// Verify district scores have national comparisons
	for _, score := range result.DistrictScores {
		if score.NationalScore == nil {
			t.Error("Expected district score to have national comparison")
		}
	}
}

// TestEnrichNAEPDataEmpty tests handling of empty data
func TestEnrichNAEPDataEmpty(t *testing.T) {
	handler := &WebHandler{}

	testData := MockNAEPDataMinimal("123456789012", "CA")
	testData.StateScores = []NAEPScore{} // Clear scores to test empty case

	result := handler.enrichNAEPData(testData)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.StateScores) != 0 {
		t.Error("Expected empty state scores")
	}

	if len(result.Grade4Scores) != 0 {
		t.Error("Expected empty Grade 4 scores")
	}

	if len(result.Grade8Scores) != 0 {
		t.Error("Expected empty Grade 8 scores")
	}
}

// TestNationalComparisonLogic tests the comparison logic edge cases
func TestNationalComparisonLogic(t *testing.T) {
	handler := &WebHandler{}

	testCases := []struct {
		name         string
		localProf    float64
		nationalProf float64
		expected     string
	}{
		{
			name:         "Local exactly equal to national",
			localProf:    35.0,
			nationalProf: 35.0,
			expected:     "Above", // >= comparison, so equal is "Above"
		},
		{
			name:         "Local slightly above national",
			localProf:    35.1,
			nationalProf: 35.0,
			expected:     "Above",
		},
		{
			name:         "Local slightly below national",
			localProf:    34.9,
			nationalProf: 35.0,
			expected:     "Below",
		},
		{
			name:         "Local much below national",
			localProf:    20.0,
			nationalProf: 35.0,
			expected:     "Below",
		},
		{
			name:         "Local much above national",
			localProf:    50.0,
			nationalProf: 35.0,
			expected:     "Above",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock national score with the test's national proficiency
			nationalScore := MockNAEPScore("mathematics", 4, 2022, 236.0, tc.nationalProf)
			nationalView := NAEPScoreView{NAEPScore: nationalScore}

			nationalMap := map[string]*NAEPScoreView{
				"mathematics-4": &nationalView,
			}

			// Create test score with local proficiency
			localScore := MockNAEPScore("mathematics", 4, 2022, 240.0, tc.localProf)
			score := NAEPScoreView{NAEPScore: localScore}

			handler.addNationalComparison(&score, nationalMap)

			if score.NationalCompare != tc.expected {
				t.Errorf("For local=%.1f vs national=%.1f, expected %s, got %s",
					tc.localProf, tc.nationalProf, tc.expected, score.NationalCompare)
			}
		})
	}
}
