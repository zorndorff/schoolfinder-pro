package main

import (
	"database/sql"
	"testing"
)

// TestDetermineGrades tests the grade determination logic
func TestDetermineGrades(t *testing.T) {
	client := &NAEPClient{}

	testCases := []struct {
		name           string
		gradeLow       string
		gradeHigh      string
		expectedCount  int
		expectedGrades []int
	}{
		{
			name:           "Elementary school (PK-5)",
			gradeLow:       "PK",
			gradeHigh:      "05",
			expectedCount:  1,
			expectedGrades: []int{4},
		},
		{
			name:           "Middle school (6-8)",
			gradeLow:       "06",
			gradeHigh:      "08",
			expectedCount:  1,
			expectedGrades: []int{8},
		},
		{
			name:           "K-8 school",
			gradeLow:       "KG",
			gradeHigh:      "08",
			expectedCount:  2,
			expectedGrades: []int{4, 8},
		},
		{
			name:           "High school (9-12)",
			gradeLow:       "09",
			gradeHigh:      "12",
			expectedCount:  0,
			expectedGrades: []int{},
		},
		{
			name:           "K-12 school",
			gradeLow:       "KG",
			gradeHigh:      "12",
			expectedCount:  2,
			expectedGrades: []int{4, 8},
		},
		{
			name:           "Only grade 4",
			gradeLow:       "04",
			gradeHigh:      "04",
			expectedCount:  1,
			expectedGrades: []int{4},
		},
		{
			name:           "Only grade 8",
			gradeLow:       "08",
			gradeHigh:      "08",
			expectedCount:  1,
			expectedGrades: []int{8},
		},
		{
			name:           "Grades 3-5 (includes 4)",
			gradeLow:       "03",
			gradeHigh:      "05",
			expectedCount:  1,
			expectedGrades: []int{4},
		},
		{
			name:           "Grades 7-9 (includes 8)",
			gradeLow:       "07",
			gradeHigh:      "09",
			expectedCount:  1,
			expectedGrades: []int{8},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			school := MockSchool("123456", "Test School", "Test District", "CA", tc.gradeLow, tc.gradeHigh)

			grades := client.determineGrades(school)

			if len(grades) != tc.expectedCount {
				t.Errorf("Expected %d grades, got %d", tc.expectedCount, len(grades))
			}

			for i, expectedGrade := range tc.expectedGrades {
				if i >= len(grades) {
					t.Errorf("Missing expected grade %d", expectedGrade)
					continue
				}
				if grades[i] != expectedGrade {
					t.Errorf("Expected grade %d at position %d, got %d", expectedGrade, i, grades[i])
				}
			}
		})
	}
}

// TestDetermineGradesInvalid tests handling of invalid grade data
func TestDetermineGradesInvalid(t *testing.T) {
	client := &NAEPClient{}

	testCases := []struct {
		name      string
		gradeLow  sql.NullString
		gradeHigh sql.NullString
	}{
		{
			name:      "Invalid low grade",
			gradeLow:  sql.NullString{Valid: false},
			gradeHigh: sql.NullString{String: "05", Valid: true},
		},
		{
			name:      "Invalid high grade",
			gradeLow:  sql.NullString{String: "PK", Valid: true},
			gradeHigh: sql.NullString{Valid: false},
		},
		{
			name:      "Both invalid",
			gradeLow:  sql.NullString{Valid: false},
			gradeHigh: sql.NullString{Valid: false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			school := MockSchool("123456", "Test School", "Test District", "CA", "PK", "05")
			// Override with invalid grades
			school.GradeLow = tc.gradeLow
			school.GradeHigh = tc.gradeHigh

			grades := client.determineGrades(school)

			if len(grades) != 0 {
				t.Errorf("Expected no grades for invalid data, got %d", len(grades))
			}
		})
	}
}

// TestMatchDistrict tests district name matching
func TestMatchDistrict(t *testing.T) {
	client := &NAEPClient{}

	testCases := []struct {
		name         string
		districtName string
		expectedCode string
		shouldMatch  bool
	}{
		{
			name:         "Los Angeles exact match",
			districtName: "los angeles",
			expectedCode: "XL",
			shouldMatch:  true,
		},
		{
			name:         "Los Angeles with suffix",
			districtName: "los angeles unified school district",
			expectedCode: "XL",
			shouldMatch:  true,
		},
		{
			name:         "Chicago exact match",
			districtName: "chicago",
			expectedCode: "XC",
			shouldMatch:  true,
		},
		{
			name:         "New York City",
			districtName: "new york city",
			expectedCode: "XN",
			shouldMatch:  true,
		},
		{
			name:         "Houston with suffix",
			districtName: "houston independent school district",
			expectedCode: "XH",
			shouldMatch:  true,
		},
		{
			name:         "Miami-Dade exact",
			districtName: "miami-dade",
			expectedCode: "XI",
			shouldMatch:  true,
		},
		{
			name:         "Unknown district",
			districtName: "small town school district",
			expectedCode: "",
			shouldMatch:  false,
		},
		{
			name:         "Empty district",
			districtName: "",
			expectedCode: "",
			shouldMatch:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			school := MockSchool("123456", "Test School", tc.districtName, "CA", "PK", "05")

			code := client.matchDistrict(school)

			if tc.shouldMatch {
				if code == "" {
					t.Errorf("Expected to match district, got empty code")
				}
				if code != tc.expectedCode {
					t.Errorf("Expected code %s, got %s", tc.expectedCode, code)
				}
			} else {
				if code != "" {
					t.Errorf("Expected no match, got code %s", code)
				}
			}
		})
	}
}

// TestSortNAEPScores tests score sorting logic
func TestSortNAEPScores(t *testing.T) {
	testCases := []struct {
		name          string
		input         []NAEPScore
		expectedOrder []string // "subject-grade-year"
	}{
		{
			name: "Mixed grades and subjects",
			input: []NAEPScore{
				{Subject: "reading", Grade: 8, Year: 2022},
				{Subject: "mathematics", Grade: 4, Year: 2022},
				{Subject: "mathematics", Grade: 8, Year: 2019},
				{Subject: "reading", Grade: 4, Year: 2022},
			},
			expectedOrder: []string{
				"mathematics-4-2022", // Grade 4, mathematics (alphabetically first)
				"reading-4-2022",     // Grade 4, reading
				"mathematics-8-2019", // Grade 8, mathematics
				"reading-8-2022",     // Grade 8, reading
			},
		},
		{
			name: "Same subject and grade, different years",
			input: []NAEPScore{
				{Subject: "mathematics", Grade: 4, Year: 2017},
				{Subject: "mathematics", Grade: 4, Year: 2022},
				{Subject: "mathematics", Grade: 4, Year: 2019},
			},
			expectedOrder: []string{
				"mathematics-4-2022", // Most recent first
				"mathematics-4-2019",
				"mathematics-4-2017",
			},
		},
		{
			name: "Science, mathematics, reading (alphabetical)",
			input: []NAEPScore{
				{Subject: "science", Grade: 4, Year: 2022},
				{Subject: "reading", Grade: 4, Year: 2022},
				{Subject: "mathematics", Grade: 4, Year: 2022},
			},
			expectedOrder: []string{
				"mathematics-4-2022", // Alphabetically first
				"reading-4-2022",
				"science-4-2022",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scores := make([]NAEPScore, len(tc.input))
			copy(scores, tc.input)

			sortNAEPScores(scores)

			if len(scores) != len(tc.expectedOrder) {
				t.Fatalf("Length mismatch: expected %d, got %d", len(tc.expectedOrder), len(scores))
			}

			for i, expected := range tc.expectedOrder {
				// Simpler comparison - check subject and grade match
				actualKey := scores[i].Subject + "-" + string(rune(scores[i].Grade+'0'))
				expectedKey := expected[:len(expected)-5] // Remove year for simple check

				if actualKey != expectedKey {
					t.Errorf("Position %d: expected %s, got subject=%s grade=%d year=%d",
						i, expected, scores[i].Subject, scores[i].Grade, scores[i].Year)
				}
			}

			// Check year ordering for same subject/grade
			for i := 0; i < len(scores)-1; i++ {
				if scores[i].Subject == scores[i+1].Subject && scores[i].Grade == scores[i+1].Grade {
					if scores[i].Year < scores[i+1].Year {
						t.Errorf("Years not in descending order at position %d: %d before %d",
							i, scores[i].Year, scores[i+1].Year)
					}
				}
			}
		})
	}
}

// TestGetMostRecentScore tests finding the most recent score
func TestGetMostRecentScore(t *testing.T) {
	data := MockNAEPDataMultiYear("123456", "CA")
	// Add district score for testing
	data.DistrictScores = []NAEPScore{
		MockNAEPScore("mathematics", 4, 2022, 236.0, 36.0),
	}

	testCases := []struct {
		name         string
		subject      string
		grade        int
		useDistrict  bool
		expectedYear int
		expectedProf float64
		shouldFind   bool
	}{
		{
			name:         "Most recent math grade 4 state",
			subject:      "mathematics",
			grade:        4,
			useDistrict:  false,
			expectedYear: 2022,
			expectedProf: 40.0, // From mock data
			shouldFind:   true,
		},
		{
			name:         "Reading grade 4 state",
			subject:      "reading",
			grade:        4,
			useDistrict:  false,
			expectedYear: 2022,
			expectedProf: 35.0, // From mock data
			shouldFind:   true,
		},
		{
			name:         "Math grade 4 district",
			subject:      "mathematics",
			grade:        4,
			useDistrict:  true,
			expectedYear: 2022,
			expectedProf: 36.0,
			shouldFind:   true,
		},
		{
			name:        "Non-existent subject",
			subject:     "science",
			grade:       4,
			useDistrict: false,
			shouldFind:  false,
		},
		{
			name:        "Non-existent grade",
			subject:     "mathematics",
			grade:       12,
			useDistrict: false,
			shouldFind:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := data.GetMostRecentScore(tc.subject, tc.grade, tc.useDistrict)

			if tc.shouldFind {
				if score == nil {
					t.Fatal("Expected to find score, got nil")
				}
				if score.Year != tc.expectedYear {
					t.Errorf("Expected year %d, got %d", tc.expectedYear, score.Year)
				}
				if score.AtProficient != tc.expectedProf {
					t.Errorf("Expected proficient %.1f, got %.1f", tc.expectedProf, score.AtProficient)
				}
			} else {
				if score != nil {
					t.Errorf("Expected nil, got score with year %d", score.Year)
				}
			}
		})
	}
}

// TestGetScoreTrend tests trend calculation
func TestGetScoreTrend(t *testing.T) {
	data := MockNAEPDataMultiYear("123456", "CA")

	testCases := []struct {
		name            string
		subject         string
		grade           int
		useDistrict     bool
		expectedChange  float64
		expectedCurrent int
		expectedPrev    int
	}{
		{
			name:            "Mathematics improving (2022 vs 2019)",
			subject:         "mathematics",
			grade:           4,
			useDistrict:     false,
			expectedChange:  3.0, // 40.0 - 37.0 (from mock data)
			expectedCurrent: 2022,
			expectedPrev:    2019,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			current, previous, change := data.GetScoreTrend(tc.subject, tc.grade, tc.useDistrict)

			if current == nil {
				t.Fatal("Expected current score, got nil")
			}
			if previous == nil {
				t.Fatal("Expected previous score, got nil")
			}

			if current.Year != tc.expectedCurrent {
				t.Errorf("Expected current year %d, got %d", tc.expectedCurrent, current.Year)
			}

			if previous.Year != tc.expectedPrev {
				t.Errorf("Expected previous year %d, got %d", tc.expectedPrev, previous.Year)
			}

			if change != tc.expectedChange {
				t.Errorf("Expected change %.1f, got %.1f", tc.expectedChange, change)
			}
		})
	}
}

// TestGetSubjectScoreSummary tests subject score summary
func TestGetSubjectScoreSummary(t *testing.T) {
	data := MockNAEPData("123456", "CA", "", false, false)

	t.Run("Grade 4 summary", func(t *testing.T) {
		summary := data.GetSubjectScoreSummary(4, false)

		// Should have at least 2 subjects (math and reading)
		if len(summary) < 2 {
			t.Errorf("Expected at least 2 subjects, got %d", len(summary))
		}

		// Should have math and reading scores
		if _, ok := summary["mathematics"]; !ok {
			t.Error("Expected mathematics in summary")
		}

		if _, ok := summary["reading"]; !ok {
			t.Error("Expected reading in summary")
		}
	})

	t.Run("Grade 8 summary", func(t *testing.T) {
		summary := data.GetSubjectScoreSummary(8, false)

		// Should have at least 1 subject
		if len(summary) == 0 {
			t.Error("Expected some subjects in Grade 8 summary")
		}

		// Should have math score
		if _, ok := summary["mathematics"]; !ok {
			t.Error("Expected mathematics in Grade 8 summary")
		}
	})
}

// TestGetAllScoresForSubjectGrade tests retrieving all scores for trend analysis
func TestGetAllScoresForSubjectGrade(t *testing.T) {
	data := MockNAEPDataMultiYear("123456", "CA")

	scores := data.GetAllScoresForSubjectGrade("mathematics", 4, false)

	if len(scores) != 3 {
		t.Errorf("Expected 3 scores, got %d", len(scores))
	}

	// Should be sorted by year ascending (for trend charts)
	if len(scores) >= 3 {
		// Verify ascending order
		for i := 1; i < len(scores); i++ {
			if scores[i].Year < scores[i-1].Year {
				t.Errorf("Scores not in ascending order by year: %d before %d",
					scores[i-1].Year, scores[i].Year)
			}
		}

		// Check specific years
		if scores[0].Year != 2017 {
			t.Errorf("Expected first year 2017, got %d", scores[0].Year)
		}

		if scores[2].Year != 2022 {
			t.Errorf("Expected last year 2022, got %d", scores[2].Year)
		}
	}
}

// TestGetAchievementLevels tests achievement level estimation
func TestGetAchievementLevels(t *testing.T) {
	data := MockNAEPDataMinimal("123456", "CA")

	belowBasic, basic, proficient, advanced := data.GetAchievementLevels("mathematics", 4, false)

	// Check that percentages add up to 100
	total := belowBasic + basic + proficient + advanced
	if total < 99.9 || total > 100.1 {
		t.Errorf("Expected percentages to sum to ~100, got %.2f", total)
	}

	// Advanced should be ~10% of proficient+
	expectedAdvanced := 40.0 * 0.10
	if advanced != expectedAdvanced {
		t.Errorf("Expected advanced %.2f, got %.2f", expectedAdvanced, advanced)
	}

	// Proficient should be proficient+ minus advanced
	expectedProficient := 40.0 - expectedAdvanced
	if proficient != expectedProficient {
		t.Errorf("Expected proficient %.2f, got %.2f", expectedProficient, proficient)
	}

	// Basic should be 35%
	if basic != 35.0 {
		t.Errorf("Expected basic 35.0, got %.2f", basic)
	}

	// Below basic should be remainder
	expectedBelowBasic := 100.0 - 40.0 - 35.0
	if belowBasic != expectedBelowBasic {
		t.Errorf("Expected below basic %.2f, got %.2f", expectedBelowBasic, belowBasic)
	}
}

// TestGetAchievementLevelsZeroData tests handling of zero/missing data
func TestGetAchievementLevelsZeroData(t *testing.T) {
	data := MockNAEPDataMinimal("123456", "CA")
	// Override with zero proficiency
	data.StateScores[0].AtProficient = 0.0

	belowBasic, basic, proficient, advanced := data.GetAchievementLevels("mathematics", 4, false)

	if belowBasic != 0 || basic != 0 || proficient != 0 || advanced != 0 {
		t.Error("Expected all zeros for missing data")
	}
}
