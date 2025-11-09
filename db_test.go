package main

import (
	"testing"
)

// TestNewDB tests database initialization with mock data
func TestNewDB(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("Expected database to be initialized")
	}

	if db.conn == nil {
		t.Fatal("Expected database connection to be established")
	}
}

// TestSearchSchools tests school search functionality
func TestSearchSchools(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	testCases := []struct {
		name          string
		query         string
		state         string
		expectedCount int
		expectedName  string
	}{
		{
			name:          "Search by school name",
			query:         "Lincoln",
			state:         "",
			expectedCount: 1,
			expectedName:  "Lincoln Elementary School",
		},
		{
			name:          "Search by city",
			query:         "San Francisco",
			state:         "",
			expectedCount: 1,
			expectedName:  "Lincoln Elementary School",
		},
		{
			name:          "Search with state filter",
			query:         "School",
			state:         "CA",
			expectedCount: 2, // Lincoln and Washington are in CA
		},
		{
			name:          "Search in specific state",
			query:         "Jefferson",
			state:         "TX",
			expectedCount: 1,
			expectedName:  "Jefferson Middle School",
		},
		{
			name:          "No results",
			query:         "NonexistentSchool",
			state:         "",
			expectedCount: 0,
		},
		{
			name:          "Empty query with state filter",
			query:         "",
			state:         "NY",
			expectedCount: 1,
			expectedName:  "Roosevelt Charter School",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schools, err := db.SearchSchools(tc.query, tc.state, 100)
			if err != nil {
				t.Fatalf("SearchSchools failed: %v", err)
			}

			if len(schools) != tc.expectedCount {
				t.Errorf("Expected %d schools, got %d", tc.expectedCount, len(schools))
			}

			if tc.expectedName != "" && len(schools) > 0 {
				if schools[0].Name != tc.expectedName {
					t.Errorf("Expected first school to be %s, got %s", tc.expectedName, schools[0].Name)
				}
			}
		})
	}
}

// TestGetSchoolByID tests retrieving a specific school by ID
func TestGetSchoolByID(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	testCases := []struct {
		name          string
		ncessch       string
		shouldFind    bool
		expectedName  string
		expectedState string
	}{
		{
			name:          "Valid school ID",
			ncessch:       "360000100001",
			shouldFind:    true,
			expectedName:  "Lincoln Elementary School",
			expectedState: "CA",
		},
		{
			name:          "Another valid school ID",
			ncessch:       "360000100002",
			shouldFind:    true,
			expectedName:  "Washington High School",
			expectedState: "CA",
		},
		{
			name:       "Non-existent school ID",
			ncessch:    "999999999999",
			shouldFind: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			school, err := db.GetSchoolByID(tc.ncessch)

			if tc.shouldFind {
				if err != nil {
					t.Fatalf("GetSchoolByID failed: %v", err)
				}
				if school == nil {
					t.Fatal("Expected school to be found")
				}
				if school.Name != tc.expectedName {
					t.Errorf("Expected name %s, got %s", tc.expectedName, school.Name)
				}
				if school.State != tc.expectedState {
					t.Errorf("Expected state %s, got %s", tc.expectedState, school.State)
				}
			} else {
				if err == nil {
					t.Error("Expected error for non-existent school")
				}
			}
		})
	}
}

// TestSchoolFields tests that school fields are correctly populated
func TestSchoolFields(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	school, err := db.GetSchoolByID("360000100001")
	if err != nil {
		t.Fatalf("GetSchoolByID failed: %v", err)
	}

	// Test basic fields
	if school.NCESSCH != "360000100001" {
		t.Errorf("Expected NCESSCH 360000100001, got %s", school.NCESSCH)
	}

	if school.City != "San Francisco" {
		t.Errorf("Expected city San Francisco, got %s", school.City)
	}

	// Test nullable fields
	if !school.Teachers.Valid {
		t.Error("Expected Teachers to be valid")
	}
	if school.Teachers.Float64 != 25.5 {
		t.Errorf("Expected 25.5 teachers, got %.1f", school.Teachers.Float64)
	}

	if !school.Enrollment.Valid {
		t.Error("Expected Enrollment to be valid")
	}
	if school.Enrollment.Int64 != 500 {
		t.Errorf("Expected 500 students, got %d", school.Enrollment.Int64)
	}

	if !school.Phone.Valid {
		t.Error("Expected Phone to be valid")
	}

	if !school.Website.Valid {
		t.Error("Expected Website to be valid")
	}
}

// TestSchoolHelperMethods tests the helper methods on School struct
func TestSchoolHelperMethods(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	school, err := db.GetSchoolByID("360000100001")
	if err != nil {
		t.Fatalf("GetSchoolByID failed: %v", err)
	}

	// Test TeachersString
	teachersStr := school.TeachersString()
	if teachersStr != "25.5" {
		t.Errorf("Expected teachers string '25.5', got '%s'", teachersStr)
	}

	// Test EnrollmentString
	enrollmentStr := school.EnrollmentString()
	if enrollmentStr != "500" {
		t.Errorf("Expected enrollment string '500', got '%s'", enrollmentStr)
	}

	// Test StudentTeacherRatio
	ratio := school.StudentTeacherRatio()
	if ratio != "19.6:1" {
		t.Errorf("Expected ratio '19.6:1', got '%s'", ratio)
	}

	// Test GradeRangeString
	gradeRange := school.GradeRangeString()
	if gradeRange != "Pre-K - 5" {
		t.Errorf("Expected grade range 'Pre-K - 5', got '%s'", gradeRange)
	}

	// Test CharterString
	charter := school.CharterString()
	if charter != "No" {
		t.Errorf("Expected charter 'No', got '%s'", charter)
	}

	// Test charter school
	charterSchool, err := db.GetSchoolByID("360000100004")
	if err != nil {
		t.Fatalf("GetSchoolByID failed for charter school: %v", err)
	}
	charterStr := charterSchool.CharterString()
	if charterStr != "Yes" {
		t.Errorf("Expected charter 'Yes' for charter school, got '%s'", charterStr)
	}
}

// TestSearchLimit tests that search respects the limit parameter
func TestSearchLimit(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	// Search with limit of 2
	schools, err := db.SearchSchools("School", "", 2)
	if err != nil {
		t.Fatalf("SearchSchools failed: %v", err)
	}

	if len(schools) > 2 {
		t.Errorf("Expected at most 2 schools, got %d", len(schools))
	}
}

// TestDatabasePersistence tests that data persists across queries
func TestDatabasePersistence(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	defer cleanup()

	// First query
	schools1, err := db.SearchSchools("Lincoln", "", 100)
	if err != nil {
		t.Fatalf("First SearchSchools failed: %v", err)
	}

	// Second query
	schools2, err := db.SearchSchools("Lincoln", "", 100)
	if err != nil {
		t.Fatalf("Second SearchSchools failed: %v", err)
	}

	// Results should be the same
	if len(schools1) != len(schools2) {
		t.Errorf("Expected consistent results, got %d then %d", len(schools1), len(schools2))
	}
}
