package cmd

import (
	"fmt"
	"os"
)

// This file contains shared initialization logic that will be implemented
// by importing main package functions. The main package will need to export
// certain initialization functions for CLI commands to use.

// LaunchTUI will be implemented to call the main package's TUI launcher
var LaunchTUI func(dataDir string)

// InitializeDB will be implemented to call the main package's DB initialization
type DBInterface interface {
	SearchSchools(query string, state string, limit int) ([]SchoolData, error)
	GetSchoolByID(ncessch string) (*SchoolData, error)
	Close() error
}

// AIScraperInterface defines the interface for AI scraping
type AIScraperInterface interface {
	ExtractSchoolDataWithWebSearch(school *SchoolData) (*EnhancedSchoolDataJSON, error)
}

// SchoolData represents a school record (matches main.School)
type SchoolData struct {
	NCESSCH        string   `json:"ncessch"`
	Name           string   `json:"name"`
	State          string   `json:"state"`
	StateName      string   `json:"state_name"`
	City           string   `json:"city"`
	District       string   `json:"district"`
	DistrictID     *string  `json:"district_id,omitempty"`
	SchoolYear     string   `json:"school_year"`
	Teachers       *float64 `json:"teachers,omitempty"`
	Level          *string  `json:"level,omitempty"`
	Phone          *string  `json:"phone,omitempty"`
	Website        *string  `json:"website,omitempty"`
	Zip            *string  `json:"zip,omitempty"`
	Street1        *string  `json:"street1,omitempty"`
	Street2        *string  `json:"street2,omitempty"`
	Street3        *string  `json:"street3,omitempty"`
	SchoolType     *string  `json:"school_type,omitempty"`
	GradeLow       *string  `json:"grade_low,omitempty"`
	GradeHigh      *string  `json:"grade_high,omitempty"`
	CharterText    *string  `json:"charter_text,omitempty"`
	Enrollment     *int64   `json:"enrollment,omitempty"`
}

// EnhancedSchoolDataJSON represents enhanced data from AI scraping
type EnhancedSchoolDataJSON struct {
	NCESSCH          string          `json:"ncessch"`
	SchoolName       string          `json:"school_name"`
	ExtractedAt      string          `json:"extracted_at"`
	SourceURL        string          `json:"source_url"`
	MarkdownContent  string          `json:"markdown_content"`
	Principal        string          `json:"principal,omitempty"`
	VicePrincipals   []string        `json:"vice_principals,omitempty"`
	Mascot           string          `json:"mascot,omitempty"`
	SchoolColors     []string        `json:"school_colors,omitempty"`
	Founded          string          `json:"founded,omitempty"`
	StaffContacts    []StaffContact  `json:"staff_contacts,omitempty"`
	MainOfficeEmail  string          `json:"main_office_email,omitempty"`
	MainOfficePhone  string          `json:"main_office_phone,omitempty"`
	APCourses        []string        `json:"ap_courses,omitempty"`
	Honors           []string        `json:"honors,omitempty"`
	SpecialPrograms  []string        `json:"special_programs,omitempty"`
	Languages        []string        `json:"languages,omitempty"`
	Sports           []string        `json:"sports,omitempty"`
	Clubs            []string        `json:"clubs,omitempty"`
	Arts             []string        `json:"arts,omitempty"`
	Facilities       []string        `json:"facilities,omitempty"`
	BellSchedule     string          `json:"bell_schedule,omitempty"`
	SchoolHours      string          `json:"school_hours,omitempty"`
	Achievements     []string        `json:"achievements,omitempty"`
	Accreditations   []string        `json:"accreditations,omitempty"`
	Mission          string          `json:"mission,omitempty"`
	Notes            string          `json:"notes,omitempty"`
}

// StaffContact represents staff contact information
type StaffContact struct {
	Name       string `json:"name"`
	Title      string `json:"title,omitempty"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Department string `json:"department,omitempty"`
}

// HandleError prints error and exits
func HandleError(err error, message string) {
	fmt.Fprintf(os.Stderr, "Error: %s: %v\n", message, err)
	os.Exit(1)
}
