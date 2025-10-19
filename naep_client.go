package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	cacheDir   string
	cacheTTL   time.Duration
}

// NAEP API response structures
type naepAPIResponse struct {
	Status int              `json:"status"`
	Result []naepDataPoint  `json:"result"`
}

type naepDataPoint struct {
	Value        string `json:"value"`
	ErrorFlag    string `json:"errorFlag"`
	Year         string `json:"year"`
	Jurisdiction string `json:"jurisLabel"`
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
func NewNAEPClient(cacheDir string) *NAEPClient {
	if cacheDir == "" {
		cacheDir = ".naep_cache"
	}

	// Create cache directory
	os.MkdirAll(cacheDir, 0755)

	return &NAEPClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cacheDir:   cacheDir,
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

	// Check if school serves grade 12
	if lowNum <= 12 && highNum >= 12 {
		grades = append(grades, 12)
	}

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
		year, _ := strconv.Atoi(dp.Year)
		meanScore, _ := strconv.ParseFloat(dp.Value, 64)
		errorFlag, _ := strconv.Atoi(dp.ErrorFlag)

		score := NAEPScore{
			Subject:      subjectName,
			Grade:        grade,
			Year:         year,
			Jurisdiction: dp.Jurisdiction,
			JurisCode:    jurisCode,
			MeanScore:    meanScore,
			ErrorCode:    errorFlag,
		}

		// Find matching achievement level data
		for _, alc := range alcScores {
			if alc.Year == dp.Year {
				proficientPlus, _ := strconv.ParseFloat(alc.Value, 64)
				score.AtProficient = proficientPlus
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
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d for URL: %s", resp.StatusCode, apiURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp naepAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON (body: %s): %w", string(body[:min(len(body), 200)]), err)
	}

	if apiResp.Status != 200 {
		return nil, fmt.Errorf("API status not OK: %d (body: %s)", apiResp.Status, string(body[:min(len(body), 200)]))
	}

	if len(apiResp.Result) == 0 {
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

// getCachedData retrieves cached NAEP data
func (c *NAEPClient) getCachedData(ncessch string) (*NAEPData, error) {
	cachePath := filepath.Join(c.cacheDir, ncessch+".json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cached NAEPData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	// Check if cache is expired
	if time.Since(cached.ExtractedAt) > c.cacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	return &cached, nil
}

// cacheData caches NAEP data to disk
func (c *NAEPClient) cacheData(ncessch string, data *NAEPData) error {
	cachePath := filepath.Join(c.cacheDir, ncessch+".json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, jsonData, 0644)
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
