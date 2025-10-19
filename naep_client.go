package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NAEPScore represents a single NAEP assessment score
type NAEPScore struct {
	Subject      string  `json:"subject"`
	Grade        int     `json:"grade"`
	Year         int     `json:"year"`
	Jurisdiction string  `json:"jurisdiction"`      // Full name like "California" or "Los Angeles"
	JurisCode    string  `json:"jurisdiction_code"` // NAEP code like "CA" or "XL"

	// Statistical data
	MeanScore   float64 `json:"mean_score"`
	MeanScoreSE float64 `json:"mean_score_se"` // Standard error

	// Achievement levels (percentages)
	BelowBasic   float64 `json:"below_basic"`
	AtBasic      float64 `json:"at_basic"`
	AtProficient float64 `json:"at_proficient"`
	AtAdvanced   float64 `json:"at_advanced"`

	// Metadata
	ErrorCode int `json:"error_code,omitempty"` // NAEP error code (0 = no error)
}

// NAEPData represents all NAEP data for a school
type NAEPData struct {
	NCESSCH        string      `json:"ncessch"`
	State          string      `json:"state"`
	District       string      `json:"district,omitempty"`
	ExtractedAt    time.Time   `json:"extracted_at"`
	StateScores    []NAEPScore `json:"state_scores"`
	DistrictScores []NAEPScore `json:"district_scores,omitempty"`
}

// NAEPClient handles NAEP API requests and caching
type NAEPClient struct {
	httpClient *http.Client
	db         *DB
	cacheTTL   time.Duration
}

// NAEP API response structures
type naepAPIResponse struct {
	Status int              `json:"status"`
	Result []naepDataPoint  `json:"result"`
}

type naepDataPoint struct {
	Value        float64 `json:"value"`
	ErrorFlag    int     `json:"errorFlag"`
	Year         int     `json:"year"`
	Jurisdiction string  `json:"jurisLabel"`
}

// Map of NAEP large city districts to jurisdiction codes
var naepDistrictMap = map[string]string{
	"albuquerque":                      "XQ",
	"atlanta":                          "XA",
	"austin":                           "XU",
	"baltimore city":                   "XM",
	"boston":                           "XB",
	"charlotte":                        "XT",
	"chicago":                          "XC",
	"clark county":                     "XX",
	"cleveland":                        "XV",
	"dallas":                           "XS",
	"denver":                           "XY",
	"detroit":                          "XR",
	"district of columbia":             "XW",
	"duval county":                     "XE",
	"fort worth":                       "XZ",
	"fresno":                           "XF",
	"guilford county":                  "XG",
	"hillsborough county":              "XO",
	"houston":                          "XH",
	"jefferson county":                 "XJ",
	"los angeles":                      "XL",
	"miami-dade":                       "XI",
	"milwaukee":                        "XK",
	"new york city":                    "XN",
	"philadelphia":                     "XP",
	"san diego":                        "XD",
	"shelby county":                    "YA",
}

// NAEP subject codes
var naepSubjects = map[string]struct {
	code     string
	subscale string
}{
	"mathematics": {"mathematics", "MRPCM"},
	"reading":     {"reading", "RRPCM"},
	"science":     {"science", "SRPUV"},
}

// NewNAEPClient creates a new NAEP API client
func NewNAEPClient(db *DB) *NAEPClient {
	if logger != nil {
		logger.Info("NAEP client initialized with database caching", "cache_ttl_days", 90)
	}

	return &NAEPClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		db:         db,
		cacheTTL:   90 * 24 * time.Hour, // 90 days
	}
}

// FetchNAEPData fetches NAEP data for a school
func (c *NAEPClient) FetchNAEPData(school *School) (*NAEPData, error) {
	// Check cache first
	if cached, err := c.getCachedData(school.NCESSCH); err == nil {
		return cached, nil
	}

	data := &NAEPData{
		NCESSCH:     school.NCESSCH,
		State:       school.State,
		ExtractedAt: time.Now(),
	}

	// Determine which grades to fetch based on school's grade range
	grades := c.determineGrades(school)
	if len(grades) == 0 {
		return nil, fmt.Errorf("no NAEP grades applicable for this school (grade range: %s-%s)",
			school.GradeLow.String, school.GradeHigh.String)
	}

	// Determine years to fetch (most recent assessments)
	years := []string{"2022", "2019", "2017"}

	// Fetch state-level data
	stateScores, err := c.fetchScoresForJurisdiction(school.State, grades, years)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch state scores: %w", err)
	}

	if len(stateScores) == 0 {
		return nil, fmt.Errorf("no NAEP data available for state %s, grades %v, years %v",
			school.State, grades, years)
	}

	data.StateScores = stateScores

	// Attempt to fetch district-level data for large cities
	if districtCode := c.matchDistrict(school); districtCode != "" {
		districtScores, err := c.fetchScoresForJurisdiction(districtCode, grades, years)
		if err == nil && len(districtScores) > 0 {
			data.District = school.District
			data.DistrictScores = districtScores
		}
	}

	// Cache the data
	c.cacheData(school.NCESSCH, data)

	return data, nil
}

// determineGrades determines which NAEP grades (4, 8, 12) apply to this school
// Note: Grade 12 is excluded because NAEP only assesses grade 12 at the national level,
// not at state or district levels. State-level assessments are only available for grades 4 and 8.
func (c *NAEPClient) determineGrades(school *School) []int {
	var grades []int

	if !school.GradeLow.Valid || !school.GradeHigh.Valid {
		return grades
	}

	// Convert grade codes to numbers
	gradeMap := map[string]int{
		"PK": -1, "KG": 0, "01": 1, "02": 2, "03": 3, "04": 4,
		"05": 5, "06": 6, "07": 7, "08": 8, "09": 9, "10": 10,
		"11": 11, "12": 12, "13": 13,
	}

	lowNum := gradeMap[school.GradeLow.String]
	highNum := gradeMap[school.GradeHigh.String]

	// Check if school serves grade 4
	if lowNum <= 4 && highNum >= 4 {
		grades = append(grades, 4)
	}

	// Check if school serves grade 8
	if lowNum <= 8 && highNum >= 8 {
		grades = append(grades, 8)
	}

	// NOTE: Grade 12 is excluded because NAEP grade 12 assessments are only
	// available at the national level, not for individual states or districts.
	// Including grade 12 would cause API 400 errors for state/district queries.
	// If needed in the future, grade 12 data would require separate national-level queries.

	return grades
}

// matchDistrict attempts to match school district to NAEP large city districts
func (c *NAEPClient) matchDistrict(school *School) string {
	districtName := strings.ToLower(strings.TrimSpace(school.District))

	// Direct match
	if code, ok := naepDistrictMap[districtName]; ok {
		return code
	}

	// Partial match (contains key district name)
	for key, code := range naepDistrictMap {
		if strings.Contains(districtName, key) {
			return code
		}
	}

	return ""
}

// fetchScoresForJurisdiction fetches NAEP scores for a jurisdiction
func (c *NAEPClient) fetchScoresForJurisdiction(jurisCode string, grades []int, years []string) ([]NAEPScore, error) {
	var allScores []NAEPScore
	var errors []string

	// Fetch for each subject
	for subjectName, subjectInfo := range naepSubjects {
		for _, grade := range grades {
			scores, err := c.fetchSubjectScores(jurisCode, subjectName, subjectInfo.code, subjectInfo.subscale, grade, years)
			if err != nil {
				// Collect errors but don't fail entire request if one subject/grade combo fails
				errors = append(errors, fmt.Sprintf("%s grade %d: %v", subjectName, grade, err))
				continue
			}
			allScores = append(allScores, scores...)
		}
	}

	// If we got some scores, return them even if some failed
	if len(allScores) > 0 {
		return allScores, nil
	}

	// If we got no scores and have errors, return error
	if len(errors) > 0 {
		return nil, fmt.Errorf("all requests failed: %s", strings.Join(errors, "; "))
	}

	return allScores, nil
}

// fetchSubjectScores fetches scores for a specific subject/grade/jurisdiction
func (c *NAEPClient) fetchSubjectScores(jurisCode, subjectName, subjectCode, subscale string, grade int, years []string) ([]NAEPScore, error) {
	// Build URL for mean scores
	meanURL := c.buildNAEPURL(map[string]string{
		"type":         "data",
		"subject":      subjectCode,
		"grade":        strconv.Itoa(grade),
		"subscale":     subscale,
		"variable":     "TOTAL",
		"jurisdiction": jurisCode,
		"stattype":     "MN:MN",
		"Year":         strings.Join(years, ","),
	})

	meanScores, err := c.fetchAndParse(meanURL)
	if err != nil {
		return nil, err
	}

	// Build URL for achievement levels (cumulative proficient+)
	alcURL := c.buildNAEPURL(map[string]string{
		"type":         "data",
		"subject":      subjectCode,
		"grade":        strconv.Itoa(grade),
		"subscale":     subscale,
		"variable":     "TOTAL",
		"jurisdiction": jurisCode,
		"stattype":     "ALC:AP", // At or above proficient
		"Year":         strings.Join(years, ","),
	})

	alcScores, _ := c.fetchAndParse(alcURL)

	// Combine mean and achievement level data
	var scores []NAEPScore
	for _, dp := range meanScores {
		score := NAEPScore{
			Subject:      subjectName,
			Grade:        grade,
			Year:         dp.Year,
			Jurisdiction: dp.Jurisdiction,
			JurisCode:    jurisCode,
			MeanScore:    dp.Value,
			ErrorCode:    dp.ErrorFlag,
		}

		// Find matching achievement level data
		for _, alc := range alcScores {
			if alc.Year == dp.Year {
				score.AtProficient = alc.Value
				break
			}
		}

		scores = append(scores, score)
	}

	return scores, nil
}

// buildNAEPURL builds a NAEP API URL with parameters
func (c *NAEPClient) buildNAEPURL(params map[string]string) string {
	baseURL := "https://www.nationsreportcard.gov/DataService/GetAdhocData.aspx"

	u, _ := url.Parse(baseURL)
	q := u.Query()

	for key, value := range params {
		q.Set(key, value)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// fetchAndParse fetches and parses NAEP API response
func (c *NAEPClient) fetchAndParse(apiURL string) ([]naepDataPoint, error) {
	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		if logger != nil {
			logger.Error("NAEP API HTTP request failed", "error", err, "url", apiURL)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if logger != nil {
			logger.Error("NAEP API returned non-OK status", "status_code", resp.StatusCode, "url", apiURL)
		}
		return nil, fmt.Errorf("API returned status %d for URL: %s", resp.StatusCode, apiURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to read NAEP API response body", "error", err, "url", apiURL)
		}
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp naepAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		bodyPreview := string(body[:min(len(body), 200)])
		if logger != nil {
			logger.Error("Failed to parse NAEP API JSON response",
				"error", err,
				"url", apiURL,
				"body_preview", bodyPreview,
				slog.Int("body_length", len(body)))
		}
		return nil, fmt.Errorf("failed to parse JSON (body: %s): %w", bodyPreview, err)
	}

	if apiResp.Status != 200 {
		bodyPreview := string(body[:min(len(body), 200)])
		if logger != nil {
			logger.Error("NAEP API returned error status",
				"api_status", apiResp.Status,
				"url", apiURL,
				"body_preview", bodyPreview)
		}
		return nil, fmt.Errorf("API status not OK: %d (body: %s)", apiResp.Status, bodyPreview)
	}

	if len(apiResp.Result) == 0 {
		if logger != nil {
			logger.Warn("NAEP API returned empty results", "url", apiURL)
		}
		return nil, fmt.Errorf("no results returned from API")
	}

	return apiResp.Result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getCachedData retrieves cached NAEP data from the database
func (c *NAEPClient) getCachedData(ncessch string) (*NAEPData, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	state, district, stateScoresJSON, districtScoresJSON, extractedAt, err := c.db.LoadNAEPCache(ncessch, c.cacheTTL)
	if err != nil {
		return nil, err
	}

	data := &NAEPData{
		NCESSCH:     ncessch,
		State:       state,
		District:    district,
		ExtractedAt: extractedAt,
	}

	// Unmarshal state scores
	if len(stateScoresJSON) > 0 {
		if err := json.Unmarshal(stateScoresJSON, &data.StateScores); err != nil {
			if logger != nil {
				logger.Error("Failed to unmarshal state scores from cache", "error", err, "ncessch", ncessch)
			}
			return nil, fmt.Errorf("failed to unmarshal state scores: %w", err)
		}
	}

	// Unmarshal district scores
	if len(districtScoresJSON) > 0 {
		if err := json.Unmarshal(districtScoresJSON, &data.DistrictScores); err != nil {
			if logger != nil {
				logger.Error("Failed to unmarshal district scores from cache", "error", err, "ncessch", ncessch)
			}
			return nil, fmt.Errorf("failed to unmarshal district scores: %w", err)
		}
	}

	return data, nil
}

// cacheData caches NAEP data to the database
func (c *NAEPClient) cacheData(ncessch string, data *NAEPData) error {
	if c.db == nil {
		return fmt.Errorf("database not available")
	}

	// Marshal scores to JSON
	stateScoresJSON, err := json.Marshal(data.StateScores)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to marshal state scores", "error", err, "ncessch", ncessch)
		}
		return fmt.Errorf("failed to marshal state scores: %w", err)
	}

	var districtScoresJSON []byte
	if len(data.DistrictScores) > 0 {
		districtScoresJSON, err = json.Marshal(data.DistrictScores)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to marshal district scores", "error", err, "ncessch", ncessch)
			}
			return fmt.Errorf("failed to marshal district scores: %w", err)
		}
	}

	return c.db.SaveNAEPCache(
		ncessch,
		data.State,
		data.District,
		stateScoresJSON,
		districtScoresJSON,
		data.ExtractedAt,
	)
}

// GetMostRecentScore returns the most recent score for a subject/grade
func (data *NAEPData) GetMostRecentScore(subject string, grade int, useDistrict bool) *NAEPScore {
	scores := data.StateScores
	if useDistrict && len(data.DistrictScores) > 0 {
		scores = data.DistrictScores
	}

	var mostRecent *NAEPScore
	for i := range scores {
		score := &scores[i]
		if score.Subject == subject && score.Grade == grade {
			if mostRecent == nil || score.Year > mostRecent.Year {
				mostRecent = score
			}
		}
	}

	return mostRecent
}

// GetScoreTrend returns score change between two most recent years
func (data *NAEPData) GetScoreTrend(subject string, grade int, useDistrict bool) (current, previous *NAEPScore, change float64) {
	scores := data.StateScores
	if useDistrict && len(data.DistrictScores) > 0 {
		scores = data.DistrictScores
	}

	var relevantScores []NAEPScore
	for _, score := range scores {
		if score.Subject == subject && score.Grade == grade && score.MeanScore > 0 {
			relevantScores = append(relevantScores, score)
		}
	}

	if len(relevantScores) < 2 {
		if len(relevantScores) == 1 {
			return &relevantScores[0], nil, 0
		}
		return nil, nil, 0
	}

	// Sort by year descending
	for i := 0; i < len(relevantScores)-1; i++ {
		for j := i + 1; j < len(relevantScores); j++ {
			if relevantScores[j].Year > relevantScores[i].Year {
				relevantScores[i], relevantScores[j] = relevantScores[j], relevantScores[i]
			}
		}
	}

	current = &relevantScores[0]
	previous = &relevantScores[1]
	change = current.MeanScore - previous.MeanScore

	return current, previous, change
}

// GetAllScoresForSubjectGrade returns all scores for a subject/grade (for trend visualization)
func (data *NAEPData) GetAllScoresForSubjectGrade(subject string, grade int, useDistrict bool) []NAEPScore {
	scores := data.StateScores
	if useDistrict && len(data.DistrictScores) > 0 {
		scores = data.DistrictScores
	}

	var relevantScores []NAEPScore
	for _, score := range scores {
		if score.Subject == subject && score.Grade == grade && score.MeanScore > 0 {
			relevantScores = append(relevantScores, score)
		}
	}

	// Sort by year ascending for trend charts
	for i := 0; i < len(relevantScores)-1; i++ {
		for j := i + 1; j < len(relevantScores); j++ {
			if relevantScores[j].Year < relevantScores[i].Year {
				relevantScores[i], relevantScores[j] = relevantScores[j], relevantScores[i]
			}
		}
	}

	return relevantScores
}

// GetSubjectScoreSummary returns the most recent score for each subject at a given grade
func (data *NAEPData) GetSubjectScoreSummary(grade int, useDistrict bool) map[string]float64 {
	summary := make(map[string]float64)

	subjects := []string{"mathematics", "reading", "science"}
	for _, subject := range subjects {
		mostRecent := data.GetMostRecentScore(subject, grade, useDistrict)
		if mostRecent != nil && mostRecent.MeanScore > 0 {
			summary[subject] = mostRecent.MeanScore
		}
	}

	return summary
}

// GetAchievementLevels returns achievement level percentages (estimated from available data)
// Note: NAEP API provides AtProficient which is cumulative (Proficient + Advanced)
// This method estimates the breakdown based on typical NAEP distributions
func (data *NAEPData) GetAchievementLevels(subject string, grade int, useDistrict bool) (belowBasic, basic, proficient, advanced float64) {
	mostRecent := data.GetMostRecentScore(subject, grade, useDistrict)
	if mostRecent == nil || mostRecent.AtProficient == 0 {
		return 0, 0, 0, 0
	}

	// We have AtProficient which is Proficient + Advanced
	proficientPlus := mostRecent.AtProficient

	// Estimate breakdown based on typical NAEP patterns:
	// Advanced is typically 8-12% of Proficient+
	// Proficient is the rest
	// Basic is typically 30-40% of total
	// Below Basic is the remainder

	advanced = proficientPlus * 0.10 // Estimate 10% of prof+ is advanced
	proficient = proficientPlus - advanced

	// Estimate Basic as ~35% and Below Basic as remainder
	basic = 35.0
	belowBasic = 100.0 - proficientPlus - basic

	// Ensure non-negative
	if belowBasic < 0 {
		basic += belowBasic
		belowBasic = 0
	}

	return belowBasic, basic, proficient, advanced
}
