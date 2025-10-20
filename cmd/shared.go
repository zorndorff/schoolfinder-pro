package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

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

// DBInterface wraps database operations for CLI commands
type DBInterface interface {
	SearchSchools(query string, state string, limit int) ([]SchoolData, error)
	GetSchoolByID(ncessch string) (*SchoolData, error)
	Close() error
}

// AIScraperInterface defines the interface for AI scraping
type AIScraperInterface interface {
	ExtractSchoolDataWithWebSearch(school *SchoolData) (*EnhancedSchoolDataJSON, error)
}

// These variables will be set by main package
var (
	LaunchTUI    func(dataDir string)
	InitDB       func(dataDir string) (DBInterface, func(), error)
	InitAIScraper func(db DBInterface) (AIScraperInterface, error)
)

// HandleError prints error and exits
func HandleError(err error, message string) {
	fmt.Fprintf(os.Stderr, "Error: %s: %v\n", message, err)
	os.Exit(1)
}

// dbWrapper wraps the main.DB to implement DBInterface
type dbWrapper struct {
	db interface {
		SearchSchools(query string, state string, limit int) ([]interface{}, error)
		GetSchoolByID(ncessch string) (interface{}, error)
		Close() error
	}
}

func (w *dbWrapper) SearchSchools(query string, state string, limit int) ([]SchoolData, error) {
	results, err := w.db.SearchSchools(query, state, limit)
	if err != nil {
		return nil, err
	}
	
	schools := make([]SchoolData, len(results))
	for i, r := range results {
		schools[i] = convertSchool(r)
	}
	return schools, nil
}

func (w *dbWrapper) GetSchoolByID(ncessch string) (*SchoolData, error) {
	result, err := w.db.GetSchoolByID(ncessch)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	school := convertSchool(result)
	return &school, nil
}

func (w *dbWrapper) Close() error {
	return w.db.Close()
}

// convertSchool converts main.School to cmd.SchoolData
func convertSchool(s interface{}) SchoolData {
	type mainSchool struct {
		NCESSCH        string
		Name           string
		State          string
		StateName      string
		City           string
		District       string
		DistrictID     sql.NullString
		SchoolYear     string
		Teachers       sql.NullFloat64
		Level          sql.NullString
		Phone          sql.NullString
		Website        sql.NullString
		Zip            sql.NullString
		Street1        sql.NullString
		Street2        sql.NullString
		Street3        sql.NullString
		SchoolType     sql.NullString
		GradeLow       sql.NullString
		GradeHigh      sql.NullString
		CharterText    sql.NullString
		Enrollment     sql.NullInt64
	}
	
	school := s.(mainSchool)
	
	data := SchoolData{
		NCESSCH:    school.NCESSCH,
		Name:       school.Name,
		State:      school.State,
		StateName:  school.StateName,
		City:       school.City,
		District:   school.District,
		SchoolYear: school.SchoolYear,
	}
	
	if school.DistrictID.Valid {
		data.DistrictID = &school.DistrictID.String
	}
	if school.Teachers.Valid {
		data.Teachers = &school.Teachers.Float64
	}
	if school.Level.Valid {
		data.Level = &school.Level.String
	}
	if school.Phone.Valid {
		data.Phone = &school.Phone.String
	}
	if school.Website.Valid {
		data.Website = &school.Website.String
	}
	if school.Zip.Valid {
		data.Zip = &school.Zip.String
	}
	if school.Street1.Valid {
		data.Street1 = &school.Street1.String
	}
	if school.Street2.Valid {
		data.Street2 = &school.Street2.String
	}
	if school.Street3.Valid {
		data.Street3 = &school.Street3.String
	}
	if school.SchoolType.Valid {
		data.SchoolType = &school.SchoolType.String
	}
	if school.GradeLow.Valid {
		data.GradeLow = &school.GradeLow.String
	}
	if school.GradeHigh.Valid {
		data.GradeHigh = &school.GradeHigh.String
	}
	if school.CharterText.Valid {
		data.CharterText = &school.CharterText.String
	}
	if school.Enrollment.Valid {
		data.Enrollment = &school.Enrollment.Int64
	}
	
	return data
}

// aiScraperWrapper wraps the main.AIScraperService to implement AIScraperInterface
type aiScraperWrapper struct {
	scraper interface {
		ExtractSchoolDataWithWebSearch(school interface{}) (interface{}, error)
	}
}

func (w *aiScraperWrapper) ExtractSchoolDataWithWebSearch(school *SchoolData) (*EnhancedSchoolDataJSON, error) {
	mainSchool := convertToMainSchool(school)
	result, err := w.scraper.ExtractSchoolDataWithWebSearch(mainSchool)
	if err != nil {
		return nil, err
	}
	return convertEnhancedData(result), nil
}

func convertToMainSchool(s *SchoolData) interface{} {
	type mainSchool struct {
		NCESSCH        string
		Name           string
		State          string
		StateName      string
		City           string
		District       string
		DistrictID     sql.NullString
		SchoolYear     string
		Teachers       sql.NullFloat64
		Level          sql.NullString
		Phone          sql.NullString
		Website        sql.NullString
		Zip            sql.NullString
		Street1        sql.NullString
		Street2        sql.NullString
		Street3        sql.NullString
		SchoolType     sql.NullString
		GradeLow       sql.NullString
		GradeHigh      sql.NullString
		CharterText    sql.NullString
		Enrollment     sql.NullInt64
	}
	
	school := mainSchool{
		NCESSCH:    s.NCESSCH,
		Name:       s.Name,
		State:      s.State,
		StateName:  s.StateName,
		City:       s.City,
		District:   s.District,
		SchoolYear: s.SchoolYear,
	}
	
	if s.DistrictID != nil {
		school.DistrictID = sql.NullString{String: *s.DistrictID, Valid: true}
	}
	if s.Teachers != nil {
		school.Teachers = sql.NullFloat64{Float64: *s.Teachers, Valid: true}
	}
	if s.Level != nil {
		school.Level = sql.NullString{String: *s.Level, Valid: true}
	}
	if s.Phone != nil {
		school.Phone = sql.NullString{String: *s.Phone, Valid: true}
	}
	if s.Website != nil {
		school.Website = sql.NullString{String: *s.Website, Valid: true}
	}
	if s.Zip != nil {
		school.Zip = sql.NullString{String: *s.Zip, Valid: true}
	}
	if s.Street1 != nil {
		school.Street1 = sql.NullString{String: *s.Street1, Valid: true}
	}
	if s.Street2 != nil {
		school.Street2 = sql.NullString{String: *s.Street2, Valid: true}
	}
	if s.Street3 != nil {
		school.Street3 = sql.NullString{String: *s.Street3, Valid: true}
	}
	if s.SchoolType != nil {
		school.SchoolType = sql.NullString{String: *s.SchoolType, Valid: true}
	}
	if s.GradeLow != nil {
		school.GradeLow = sql.NullString{String: *s.GradeLow, Valid: true}
	}
	if s.GradeHigh != nil {
		school.GradeHigh = sql.NullString{String: *s.GradeHigh, Valid: true}
	}
	if s.CharterText != nil {
		school.CharterText = sql.NullString{String: *s.CharterText, Valid: true}
	}
	if s.Enrollment != nil {
		school.Enrollment = sql.NullInt64{Int64: *s.Enrollment, Valid: true}
	}
	
	return school
}

func convertEnhancedData(e interface{}) *EnhancedSchoolDataJSON {
	type mainEnhancedData struct {
		NCESSCH          string
		SchoolName       string
		ExtractedAt      time.Time
		SourceURL        string
		MarkdownContent  string
		Principal        string
		VicePrincipals   []string
		Mascot           string
		SchoolColors     []string
		Founded          string
		StaffContacts    []struct {
			Name       string
			Title      string
			Email      string
			Phone      string
			Department string
		}
		MainOfficeEmail  string
		MainOfficePhone  string
		APCourses        []string
		Honors           []string
		SpecialPrograms  []string
		Languages        []string
		Sports           []string
		Clubs            []string
		Arts             []string
		Facilities       []string
		BellSchedule     string
		SchoolHours      string
		Achievements     []string
		Accreditations   []string
		Mission          string
		Notes            string
	}
	
	enhanced := e.(mainEnhancedData)
	
	data := &EnhancedSchoolDataJSON{
		NCESSCH:          enhanced.NCESSCH,
		SchoolName:       enhanced.SchoolName,
		ExtractedAt:      enhanced.ExtractedAt.Format(time.RFC3339),
		SourceURL:        enhanced.SourceURL,
		MarkdownContent:  enhanced.MarkdownContent,
		Principal:        enhanced.Principal,
		VicePrincipals:   enhanced.VicePrincipals,
		Mascot:           enhanced.Mascot,
		SchoolColors:     enhanced.SchoolColors,
		Founded:          enhanced.Founded,
		MainOfficeEmail:  enhanced.MainOfficeEmail,
		MainOfficePhone:  enhanced.MainOfficePhone,
		APCourses:        enhanced.APCourses,
		Honors:           enhanced.Honors,
		SpecialPrograms:  enhanced.SpecialPrograms,
		Languages:        enhanced.Languages,
		Sports:           enhanced.Sports,
		Clubs:            enhanced.Clubs,
		Arts:             enhanced.Arts,
		Facilities:       enhanced.Facilities,
		BellSchedule:     enhanced.BellSchedule,
		SchoolHours:      enhanced.SchoolHours,
		Achievements:     enhanced.Achievements,
		Accreditations:   enhanced.Accreditations,
		Mission:          enhanced.Mission,
		Notes:            enhanced.Notes,
	}
	
	for _, contact := range enhanced.StaffContacts {
		data.StaffContacts = append(data.StaffContacts, StaffContact{
			Name:       contact.Name,
			Title:      contact.Title,
			Email:      contact.Email,
			Phone:      contact.Phone,
			Department: contact.Department,
		})
	}
	
	return data
}
