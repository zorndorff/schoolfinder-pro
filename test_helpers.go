package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// SetupTestDB creates a test database with mock data
func SetupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "schoolfinder-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Copy mock CSV files to temp directory
	testdataDir := "testdata"
	files := []string{
		"ccd_sch_029_2324_w_1a_073124.csv",
		"ccd_sch_059_2324_l_1a_073124.csv",
		"ccd_sch_052_2324_l_1a_073124.csv",
	}

	for _, file := range files {
		src := filepath.Join(testdataDir, file)
		dst := filepath.Join(tmpDir, file)

		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("failed to read %s: %v", src, err)
		}

		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", dst, err)
		}
	}

	// Initialize database
	db, err := NewDB(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

// MockSchool creates a mock School struct for testing
func MockSchool(ncessch, name, district, state, gradeLow, gradeHigh string) *School {
	return &School{
		NCESSCH:    ncessch,
		Name:       name,
		District:   district,
		State:      state,
		StateName:  getStateName(state),
		City:       "Test City",
		SchoolYear: "2023-2024",
		Teachers:   sql.NullFloat64{Float64: 25.0, Valid: true},
		Level:      sql.NullString{String: "Elementary", Valid: true},
		Phone:      sql.NullString{String: "555-1234", Valid: true},
		Website:    sql.NullString{String: "https://test.school.edu", Valid: true},
		Zip:        sql.NullString{String: "12345", Valid: true},
		Street1:    sql.NullString{String: "123 Main St", Valid: true},
		SchoolType: sql.NullString{String: "Regular school", Valid: true},
		GradeLow:   sql.NullString{String: gradeLow, Valid: true},
		GradeHigh:  sql.NullString{String: gradeHigh, Valid: true},
		Enrollment: sql.NullInt64{Int64: 500, Valid: true},
	}
}

// getStateName returns the full state name from abbreviation
func getStateName(abbr string) string {
	stateMap := map[string]string{
		"CA": "California",
		"TX": "Texas",
		"NY": "New York",
		"FL": "Florida",
		"IL": "Illinois",
	}
	if name, ok := stateMap[abbr]; ok {
		return name
	}
	return abbr
}

// MockNAEPData creates mock NAEP data for testing
func MockNAEPData(ncessch, state, district string, hasDistrict, multiYear bool) *NAEPData {
	data := &NAEPData{
		NCESSCH:  ncessch,
		State:    state,
		District: district,
	}

	// Add state scores
	data.StateScores = []NAEPScore{
		MockNAEPScore("mathematics", 4, 2022, 238.0, 40.0),
		MockNAEPScore("reading", 4, 2022, 220.0, 35.0),
		MockNAEPScore("mathematics", 8, 2022, 280.0, 32.0),
	}

	// Add district scores if requested
	if hasDistrict {
		data.DistrictScores = []NAEPScore{
			MockNAEPScore("mathematics", 4, 2022, 236.0, 38.0),
			MockNAEPScore("reading", 4, 2022, 218.0, 33.0),
		}
	}

	// Add national scores
	data.NationalScores = []NAEPScore{
		MockNAEPScore("mathematics", 4, 2022, 235.0, 36.0),
		MockNAEPScore("reading", 4, 2022, 217.0, 33.0),
		MockNAEPScore("mathematics", 8, 2022, 274.0, 26.0),
	}

	return data
}

// MockNAEPDataMultiYear creates NAEP data with multiple years for trend testing
func MockNAEPDataMultiYear(ncessch, state string) *NAEPData {
	data := &NAEPData{
		NCESSCH: ncessch,
		State:   state,
	}

	// Add multi-year state scores
	data.StateScores = []NAEPScore{
		MockNAEPScore("mathematics", 4, 2022, 238.0, 40.0),
		MockNAEPScore("mathematics", 4, 2019, 235.0, 37.0),
		MockNAEPScore("mathematics", 4, 2017, 233.0, 35.0),
		MockNAEPScore("reading", 4, 2022, 220.0, 35.0),
		MockNAEPScore("reading", 4, 2019, 218.0, 33.0),
		MockNAEPScore("mathematics", 8, 2022, 280.0, 32.0),
	}

	return data
}

// MockNAEPDataMinimal creates minimal NAEP data for basic testing
func MockNAEPDataMinimal(ncessch, state string) *NAEPData {
	return &NAEPData{
		NCESSCH: ncessch,
		State:   state,
		StateScores: []NAEPScore{
			MockNAEPScore("mathematics", 4, 2022, 238.0, 40.0),
		},
	}
}

// MockNAEPScore creates a mock NAEP score
func MockNAEPScore(subject string, grade, year int, meanScore, atProficient float64) NAEPScore {
	return NAEPScore{
		Subject:      subject,
		Grade:        grade,
		Year:         year,
		MeanScore:    meanScore,
		AtProficient: atProficient,
	}
}
