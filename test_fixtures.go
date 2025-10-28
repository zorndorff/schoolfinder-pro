package main

import (
	"database/sql"
	"time"
)

// MockSchool creates a test school with realistic data
func MockSchool(ncessch, name, district, state string, gradeLow, gradeHigh string) *School {
	return &School{
		NCESSCH:    ncessch,
		Name:       name,
		District:   district,
		State:      state,
		StateName:  "Test State",
		City:       "Test City",
		SchoolYear: "2023-24",
		GradeLow:   sql.NullString{String: gradeLow, Valid: true},
		GradeHigh:  sql.NullString{String: gradeHigh, Valid: true},
		Zip:        sql.NullString{String: "90001", Valid: true},
		Phone:      sql.NullString{String: "555-0100", Valid: true},
		Website:    sql.NullString{String: "https://test.school", Valid: true},
		Enrollment: sql.NullInt64{Int64: 500, Valid: true},
		Teachers:   sql.NullFloat64{Float64: 25.0, Valid: true},
	}
}

// MockNAEPScore creates a test NAEP score
func MockNAEPScore(subject string, grade, year int, meanScore, atProficient float64) NAEPScore {
	jurisdiction := "Test State"
	if grade == 4 || grade == 8 {
		jurisdiction = "California"
	}

	return NAEPScore{
		Subject:      subject,
		Grade:        grade,
		Year:         year,
		Jurisdiction: jurisdiction,
		JurisCode:    "CA",
		MeanScore:    meanScore,
		MeanScoreSE:  2.5,
		AtProficient: atProficient,
		ErrorCode:    0,
	}
}

// MockNAEPData creates comprehensive test NAEP data
func MockNAEPData(ncessch, state, district string, includeDistrict, includeNational bool) *NAEPData {
	data := &NAEPData{
		NCESSCH:     ncessch,
		State:       state,
		District:    district,
		ExtractedAt: time.Now(),
	}

	// State scores for grades 4 and 8
	data.StateScores = []NAEPScore{
		MockNAEPScore("mathematics", 4, 2022, 240.0, 40.0),
		MockNAEPScore("reading", 4, 2022, 220.0, 35.0),
		MockNAEPScore("science", 4, 2019, 150.0, 30.0),
		MockNAEPScore("mathematics", 4, 2019, 238.0, 38.0),
		MockNAEPScore("reading", 4, 2019, 218.0, 33.0),
		MockNAEPScore("mathematics", 8, 2022, 280.0, 30.0),
		MockNAEPScore("reading", 8, 2022, 265.0, 32.0),
		MockNAEPScore("science", 8, 2019, 152.0, 28.0),
	}

	// District scores (optional)
	if includeDistrict {
		data.DistrictScores = []NAEPScore{
			MockNAEPScore("mathematics", 4, 2022, 238.0, 38.0),
			MockNAEPScore("reading", 4, 2022, 218.0, 33.0),
			MockNAEPScore("mathematics", 8, 2022, 278.0, 28.0),
			MockNAEPScore("reading", 8, 2022, 263.0, 30.0),
		}
	}

	// National scores (optional)
	if includeNational {
		data.NationalScores = []NAEPScore{
			{Subject: "mathematics", Grade: 4, Year: 2022, MeanScore: 236.0, AtProficient: 36.0, Jurisdiction: "Nation", JurisCode: "NP"},
			{Subject: "reading", Grade: 4, Year: 2022, MeanScore: 217.0, AtProficient: 33.0, Jurisdiction: "Nation", JurisCode: "NP"},
			{Subject: "science", Grade: 4, Year: 2019, MeanScore: 147.0, AtProficient: 28.0, Jurisdiction: "Nation", JurisCode: "NP"},
			{Subject: "mathematics", Grade: 8, Year: 2022, MeanScore: 274.0, AtProficient: 26.0, Jurisdiction: "Nation", JurisCode: "NP"},
			{Subject: "reading", Grade: 8, Year: 2022, MeanScore: 260.0, AtProficient: 31.0, Jurisdiction: "Nation", JurisCode: "NP"},
			{Subject: "science", Grade: 8, Year: 2019, MeanScore: 149.0, AtProficient: 26.0, Jurisdiction: "Nation", JurisCode: "NP"},
		}
	}

	return data
}

// MockNAEPDataMinimal creates minimal NAEP data for basic tests
func MockNAEPDataMinimal(ncessch, state string) *NAEPData {
	return &NAEPData{
		NCESSCH:     ncessch,
		State:       state,
		ExtractedAt: time.Now(),
		StateScores: []NAEPScore{
			MockNAEPScore("mathematics", 4, 2022, 240.0, 40.0),
			MockNAEPScore("reading", 4, 2022, 220.0, 35.0),
		},
	}
}

// MockNAEPDataMultiYear creates data with multiple years for trend testing
func MockNAEPDataMultiYear(ncessch, state string) *NAEPData {
	return &NAEPData{
		NCESSCH:     ncessch,
		State:       state,
		ExtractedAt: time.Now(),
		StateScores: []NAEPScore{
			MockNAEPScore("mathematics", 4, 2017, 235.0, 35.0),
			MockNAEPScore("mathematics", 4, 2019, 237.0, 37.0),
			MockNAEPScore("mathematics", 4, 2022, 240.0, 40.0),
			MockNAEPScore("reading", 4, 2017, 215.0, 30.0),
			MockNAEPScore("reading", 4, 2019, 218.0, 32.0),
			MockNAEPScore("reading", 4, 2022, 220.0, 35.0),
		},
	}
}

// MockDB is a minimal DB interface for testing
type MockDB struct {
	Schools    map[string]*School
	NAEPCache  map[string]*NAEPData
	AICache    map[string]*EnhancedSchoolData
	SearchFunc func(query, state string, limit int) ([]School, error)
}

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		Schools:   make(map[string]*School),
		NAEPCache: make(map[string]*NAEPData),
		AICache:   make(map[string]*EnhancedSchoolData),
	}
}

// AddSchool adds a school to the mock database
func (m *MockDB) AddSchool(school *School) {
	m.Schools[school.NCESSCH] = school
}

// GetSchoolByID returns a school from the mock database
func (m *MockDB) GetSchoolByID(ncessch string) (*School, error) {
	school, ok := m.Schools[ncessch]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return school, nil
}

// AddNAEPCache adds NAEP data to the mock cache
func (m *MockDB) AddNAEPCache(ncessch string, data *NAEPData) {
	m.NAEPCache[ncessch] = data
}

// GetNAEPCache returns NAEP data from the mock cache
func (m *MockDB) GetNAEPCache(ncessch string) (*NAEPData, bool) {
	data, ok := m.NAEPCache[ncessch]
	return data, ok
}

// MockNAEPClient is a mock NAEP client for testing
type MockNAEPClient struct {
	FetchFunc func(*School) (*NAEPData, error)
}

// NewMockNAEPClient creates a new mock NAEP client
func NewMockNAEPClient() *MockNAEPClient {
	return &MockNAEPClient{
		FetchFunc: func(school *School) (*NAEPData, error) {
			// Default: return comprehensive mock data
			return MockNAEPData(school.NCESSCH, school.State, school.District, false, true), nil
		},
	}
}

// FetchNAEPData calls the mock fetch function
func (m *MockNAEPClient) FetchNAEPData(school *School) (*NAEPData, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(school)
	}
	return MockNAEPData(school.NCESSCH, school.State, school.District, false, true), nil
}

// MockAIScraperService is a mock AI scraper for testing
type MockAIScraperService struct {
	ScrapeFunc func(*School) (*EnhancedSchoolData, error)
}

// NewMockAIScraperService creates a new mock AI scraper
func NewMockAIScraperService() *MockAIScraperService {
	return &MockAIScraperService{}
}
