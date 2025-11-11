package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

type School struct {
	NCESSCH     string
	Name        string
	State       string
	StateName   string
	City        string
	District    string
	DistrictID  sql.NullString // LEAID for NAEP district matching
	SchoolYear  string
	Teachers    sql.NullFloat64
	Level       sql.NullString
	Phone       sql.NullString
	Website     sql.NullString
	Zip         sql.NullString
	Street1     sql.NullString
	Street2     sql.NullString
	Street3     sql.NullString
	SchoolType  sql.NullString
	GradeLow    sql.NullString
	GradeHigh   sql.NullString
	CharterText sql.NullString
	Enrollment  sql.NullInt64
}

type DB struct {
	conn    *sql.DB
	dataDir string
	hasFTS  bool // Whether FTS extension is available
}

func NewDB(dataDir string) (*DB, error) {
	dbPath := filepath.Join(dataDir, "data.duckdb")

	// Check if database needs to be initialized
	needsInit := false
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		needsInit = true
	}

	// Open the database file
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to open DuckDB database", "error", err, "db_path", dbPath)
		}
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}

	d := &DB{
		conn:    db,
		dataDir: dataDir,
	}

	// Initialize database if needed
	if needsInit {
		fmt.Println("📊 Initializing database from CSV files...")
		if err := d.initializeDatabase(); err != nil {
			db.Close()
			if logger != nil {
				logger.Error("Database initialization failed", "error", err, "data_dir", dataDir)
			}
			return nil, fmt.Errorf("failed to initialize database: %w", err)
		}
		fmt.Println("✅ Database initialized successfully!")
		if logger != nil {
			logger.Info("Database initialized successfully", "db_path", dbPath)
		}
	} else {
		// For existing databases, ensure FTS extension is loaded
		_, err := d.conn.Exec("LOAD fts;")
		if err != nil {
			// Try installing first if not available
			_, err = d.conn.Exec("INSTALL fts; LOAD fts;")
			if err != nil {
				// FTS extension is optional - warn but don't fail
				if logger != nil {
					logger.Warn("FTS extension not available for existing database", "error", err, "db_path", dbPath)
				}
				d.hasFTS = false
			} else {
				d.hasFTS = true
			}
		} else {
			d.hasFTS = true
		}

		// Ensure cache tables exist (for databases created before cache tables were added)
		if err := d.createCacheTables(); err != nil {
			if logger != nil {
				logger.Warn("Failed to create cache tables on existing database", "error", err)
			}
			// Don't fail - cache tables are optional
		}
	}

	return d, nil
}

// initializeDatabase creates tables and loads data from CSV files
func (d *DB) initializeDatabase() error {
	directoryFile := filepath.Join(d.dataDir, "ccd_sch_029_2324_w_1a_073124.csv")
	teacherFile := filepath.Join(d.dataDir, "ccd_sch_059_2324_l_1a_073124.csv")
	enrollmentFile := filepath.Join(d.dataDir, "ccd_sch_052_2324_l_1a_073124.csv")

	// Install and load FTS extension
	fmt.Println("   Installing FTS extension...")
	start := time.Now()
	_, err := d.conn.Exec("INSTALL fts; LOAD fts;")
	if err != nil {
		// FTS extension is optional - warn but don't fail
		fmt.Printf("   ⚠ FTS extension not available (search may be slower): %v\n", err)
		if logger != nil {
			logger.Warn("FTS extension not available", "error", err)
		}
		d.hasFTS = false
	} else {
		fmt.Printf("   ✓ FTS extension loaded (%v)\n", time.Since(start))
		d.hasFTS = true
	}

	// Start transaction for faster bulk insert
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error - will fail if transaction was committed
	}()

	// Create directory table
	fmt.Println("   Loading school directory...")
	start = time.Now()
	_, err = tx.Exec(fmt.Sprintf(`
		CREATE TABLE directory AS
		SELECT * FROM read_csv('%s', all_varchar=true)
	`, directoryFile))
	if err != nil {
		return fmt.Errorf("failed to create directory table: %w", err)
	}
	fmt.Printf("   ✓ Directory loaded (%v)\n", time.Since(start))

	// Create indexes on directory table
	fmt.Println("   Creating indexes on directory...")
	start = time.Now()
	_, err = tx.Exec(`CREATE INDEX idx_directory_ncessch ON directory(NCESSCH)`)
	if err != nil {
		return fmt.Errorf("failed to create index on NCESSCH: %w", err)
	}
	_, err = tx.Exec(`CREATE INDEX idx_directory_state ON directory(ST)`)
	if err != nil {
		return fmt.Errorf("failed to create index on ST: %w", err)
	}
	_, err = tx.Exec(`CREATE INDEX idx_directory_name ON directory(SCH_NAME)`)
	if err != nil {
		return fmt.Errorf("failed to create index on SCH_NAME: %w", err)
	}
	fmt.Printf("   ✓ Indexes created (%v)\n", time.Since(start))

	// Create teacher table
	fmt.Println("   Loading teacher data...")
	start = time.Now()
	_, err = tx.Exec(fmt.Sprintf(`
		CREATE TABLE teachers AS
		SELECT * FROM read_csv('%s', all_varchar=true)
	`, teacherFile))
	if err != nil {
		return fmt.Errorf("failed to create teachers table: %w", err)
	}
	fmt.Printf("   ✓ Teachers loaded (%v)\n", time.Since(start))

	// Create index on teachers table
	_, err = tx.Exec(`CREATE INDEX idx_teachers_ncessch ON teachers(NCESSCH)`)
	if err != nil {
		return fmt.Errorf("failed to create index on teachers NCESSCH: %w", err)
	}

	// Create enrollment table
	fmt.Println("   Loading enrollment data...")
	start = time.Now()
	_, err = tx.Exec(fmt.Sprintf(`
		CREATE TABLE enrollment AS
		SELECT * FROM read_csv('%s', all_varchar=true)
	`, enrollmentFile))
	if err != nil {
		return fmt.Errorf("failed to create enrollment table: %w", err)
	}
	fmt.Printf("   ✓ Enrollment loaded (%v)\n", time.Since(start))

	// Create index on enrollment table
	_, err = tx.Exec(`CREATE INDEX idx_enrollment_ncessch ON enrollment(NCESSCH)`)
	if err != nil {
		return fmt.Errorf("failed to create index on enrollment NCESSCH: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create FTS index (must be done outside transaction)
	// Only try if FTS extension is available
	if d.hasFTS {
		fmt.Println("   Creating full-text search index...")
		start = time.Now()
		_, err = d.conn.Exec(`
			PRAGMA create_fts_index(
				'directory',
				'NCESSCH',
				'SCH_NAME',
				'LEA_NAME',
				'MCITY',
				'MSTREET1',
				'MZIP',
				overwrite=1
			)
		`)
		if err != nil {
			// FTS index creation failed - warn but don't fail initialization
			fmt.Printf("   ⚠ FTS index creation failed (search will use fallback): %v\n", err)
			d.hasFTS = false
		} else {
			fmt.Printf("   ✓ FTS index created (%v)\n", time.Since(start))
		}
	} else {
		fmt.Println("   ⚠ Skipping FTS index creation (extension not available)")
	}

	// Create cache tables
	fmt.Println("   Creating cache tables...")
	start = time.Now()
	if err := d.createCacheTables(); err != nil {
		return fmt.Errorf("failed to create cache tables: %w", err)
	}
	fmt.Printf("   ✓ Cache tables created (%v)\n", time.Since(start))

	return nil
}

// createCacheTables creates tables for caching AI scraper and NAEP data
func (d *DB) createCacheTables() error {
	// Create AI scraper cache table
	_, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS ai_scraper_cache (
			ncessch VARCHAR PRIMARY KEY,
			school_name VARCHAR,
			extracted_at TIMESTAMP,
			source_url VARCHAR,
			markdown_content TEXT,
			legacy_data JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to create ai_scraper_cache table", "error", err)
		}
		return fmt.Errorf("failed to create ai_scraper_cache table: %w", err)
	}

	// Create NAEP cache table
	_, err = d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS naep_cache (
			ncessch VARCHAR PRIMARY KEY,
			state VARCHAR,
			district VARCHAR,
			extracted_at TIMESTAMP,
			state_scores JSON,
			district_scores JSON,
			national_scores JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to create naep_cache table", "error", err)
		}
		return fmt.Errorf("failed to create naep_cache table: %w", err)
	}

	if logger != nil {
		logger.Info("Cache tables created successfully")
	}

	return nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) SearchSchools(query string, state string, limit int) ([]School, error) {
	var schools []School

	// Build the SQL query using FTS when query is provided
	var sqlQuery string
	var args []interface{}

	if query != "" {
		if d.hasFTS {
			// Use full-text search with relevance ranking
			args = append(args, query)

			stateFilter := ""
			if state != "" {
				argIdx := 2
				stateFilter = fmt.Sprintf("AND d.ST = $%d", argIdx)
				args = append(args, state)
			}

			sqlQuery = fmt.Sprintf(`
				SELECT
					d.NCESSCH,
					d.SCH_NAME,
					d.ST,
					d.STATENAME,
					COALESCE(d.MCITY, ''),
					COALESCE(d.LEA_NAME, ''),
					d.LEAID,
					d.SCHOOL_YEAR,
					t.TEACHERS,
					d.LEVEL,
					d.PHONE,
					d.WEBSITE,
					d.MZIP,
					d.MSTREET1,
					d.MSTREET2,
					d.MSTREET3,
					d.SCH_TYPE_TEXT,
					d.GSLO,
					d.GSHI,
					d.CHARTER_TEXT,
					e.STUDENT_COUNT
				FROM directory d
				LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
				LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total'
				WHERE fts_main_directory.match_bm25(d.NCESSCH, $1) IS NOT NULL
				%s
				ORDER BY fts_main_directory.match_bm25(d.NCESSCH, $1) DESC
				LIMIT %d
			`, stateFilter, limit)
		} else {
			// Fallback to LIKE-based search when FTS is not available
			searchPattern := "%" + query + "%"
			args = append(args, searchPattern)
			argIdx := 2

			stateFilter := ""
			if state != "" {
				stateFilter = fmt.Sprintf("AND d.ST = $%d", argIdx)
				args = append(args, state)
				argIdx++
			}

			sqlQuery = fmt.Sprintf(`
				SELECT
					d.NCESSCH,
					d.SCH_NAME,
					d.ST,
					d.STATENAME,
					COALESCE(d.MCITY, ''),
					COALESCE(d.LEA_NAME, ''),
					d.LEAID,
					d.SCHOOL_YEAR,
					t.TEACHERS,
					d.LEVEL,
					d.PHONE,
					d.WEBSITE,
					d.MZIP,
					d.MSTREET1,
					d.MSTREET2,
					d.MSTREET3,
					d.SCH_TYPE_TEXT,
					d.GSLO,
					d.GSHI,
					d.CHARTER_TEXT,
					e.STUDENT_COUNT
				FROM directory d
				LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
				LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total'
				WHERE (
					LOWER(d.SCH_NAME) LIKE LOWER($1)
					OR LOWER(d.MCITY) LIKE LOWER($1)
					OR LOWER(d.LEA_NAME) LIKE LOWER($1)
					OR LOWER(d.MSTREET1) LIKE LOWER($1)
					OR d.MZIP LIKE $1
				)
				%s
				ORDER BY d.SCH_NAME
				LIMIT %d
			`, stateFilter, limit)
		}
	} else {
		// No search query, just filter by state if provided
		whereClause := "WHERE 1=1"
		argIdx := 1

		if state != "" {
			whereClause += fmt.Sprintf(" AND d.ST = $%d", argIdx)
			args = append(args, state)
		}

		sqlQuery = fmt.Sprintf(`
			SELECT
				d.NCESSCH,
				d.SCH_NAME,
				d.ST,
				d.STATENAME,
				COALESCE(d.MCITY, ''),
				COALESCE(d.LEA_NAME, ''),
				d.LEAID,
				d.SCHOOL_YEAR,
				t.TEACHERS,
				d.LEVEL,
				d.PHONE,
				d.WEBSITE,
				d.MZIP,
				d.MSTREET1,
				d.MSTREET2,
				d.MSTREET3,
				d.SCH_TYPE_TEXT,
				d.GSLO,
				d.GSHI,
				d.CHARTER_TEXT,
				e.STUDENT_COUNT
			FROM directory d
			LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
			LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total'
			%s
			ORDER BY d.SCH_NAME
			LIMIT %d
		`, whereClause, limit)
	}

	rows, err := d.conn.Query(sqlQuery, args...)
	if err != nil {
		if logger != nil {
			logger.Error("School search query failed", "error", err, "query", query, "state", state, "limit", limit)
		}
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var s School
		err := rows.Scan(
			&s.NCESSCH,
			&s.Name,
			&s.State,
			&s.StateName,
			&s.City,
			&s.District,
			&s.DistrictID,
			&s.SchoolYear,
			&s.Teachers,
			&s.Level,
			&s.Phone,
			&s.Website,
			&s.Zip,
			&s.Street1,
			&s.Street2,
			&s.Street3,
			&s.SchoolType,
			&s.GradeLow,
			&s.GradeHigh,
			&s.CharterText,
			&s.Enrollment,
		)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to scan school row", "error", err, "query", query, "state", state)
			}
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		schools = append(schools, s)
	}

	if err := rows.Err(); err != nil {
		if logger != nil {
			logger.Error("Row iteration error in search", "error", err, "query", query, "state", state, "results_count", len(schools))
		}
		return nil, err
	}

	return schools, nil
}

func (d *DB) GetSchoolByID(ncessch string) (*School, error) {
	sqlQuery := `
		SELECT
			d.NCESSCH,
			d.SCH_NAME,
			d.ST,
			d.STATENAME,
			COALESCE(d.MCITY, ''),
			COALESCE(d.LEA_NAME, ''),
			d.LEAID,
			d.SCHOOL_YEAR,
			t.TEACHERS,
			d.LEVEL,
			d.PHONE,
			d.WEBSITE,
			d.MZIP,
			d.MSTREET1,
			d.MSTREET2,
			d.MSTREET3,
			d.SCH_TYPE_TEXT,
			d.GSLO,
			d.GSHI,
			d.CHARTER_TEXT,
			e.STUDENT_COUNT
		FROM directory d
		LEFT JOIN teachers t ON d.NCESSCH = t.NCESSCH
		LEFT JOIN enrollment e ON d.NCESSCH = e.NCESSCH AND e.TOTAL_INDICATOR = 'Education Unit Total'
		WHERE d.NCESSCH = $1
		LIMIT 1
	`

	var s School
	err := d.conn.QueryRow(sqlQuery, ncessch).Scan(
		&s.NCESSCH,
		&s.Name,
		&s.State,
		&s.StateName,
		&s.City,
		&s.District,
		&s.DistrictID,
		&s.SchoolYear,
		&s.Teachers,
		&s.Level,
		&s.Phone,
		&s.Website,
		&s.Zip,
		&s.Street1,
		&s.Street2,
		&s.Street3,
		&s.SchoolType,
		&s.GradeLow,
		&s.GradeHigh,
		&s.CharterText,
		&s.Enrollment,
	)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to get school by ID", "error", err, "ncessch", ncessch)
		}
		return nil, fmt.Errorf("school not found: %w", err)
	}

	return &s, nil
}

func (d *DB) GetStates() ([]string, error) {
	sqlQuery := `
		SELECT DISTINCT ST
		FROM directory
		ORDER BY ST
	`

	rows, err := d.conn.Query(sqlQuery)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to query states", "error", err)
		}
		return nil, err
	}
	defer rows.Close()

	var states []string
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			if logger != nil {
				logger.Error("Failed to scan state row", "error", err)
			}
			return nil, err
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		if logger != nil {
			logger.Error("Row iteration error in GetStates", "error", err, "states_count", len(states))
		}
		return nil, err
	}

	return states, nil
}

func (s *School) DisplayName() string {
	return fmt.Sprintf("%s (%s, %s)", s.Name, s.City, s.State)
}

func (s *School) TeachersString() string {
	if s.Teachers.Valid {
		return fmt.Sprintf("%.1f", s.Teachers.Float64)
	}
	return "N/A"
}

func (s *School) LevelString() string {
	if s.Level.Valid {
		return s.Level.String
	}
	return "N/A"
}

func (s *School) PhoneString() string {
	if s.Phone.Valid && s.Phone.String != "" {
		return s.Phone.String
	}
	return "N/A"
}

func (s *School) WebsiteString() string {
	if s.Website.Valid && s.Website.String != "" {
		return s.Website.String
	}
	return "N/A"
}

func (s *School) FullAddress() string {
	var parts []string

	if s.Street1.Valid && s.Street1.String != "" {
		parts = append(parts, s.Street1.String)
	}
	if s.Street2.Valid && s.Street2.String != "" {
		parts = append(parts, s.Street2.String)
	}
	if s.Street3.Valid && s.Street3.String != "" {
		parts = append(parts, s.Street3.String)
	}

	if len(parts) == 0 {
		return "N/A"
	}

	var address string
	for i, part := range parts {
		if i > 0 {
			address += "\n                      "
		}
		address += part
	}

	return address
}

func (s *School) ZipString() string {
	if s.Zip.Valid && s.Zip.String != "" {
		return s.Zip.String
	}
	return "N/A"
}

func (s *School) SchoolTypeString() string {
	if s.SchoolType.Valid && s.SchoolType.String != "" {
		return s.SchoolType.String
	}
	return "N/A"
}

func (s *School) GradeRangeString() string {
	if s.GradeLow.Valid && s.GradeHigh.Valid {
		low := s.GradeLow.String
		high := s.GradeHigh.String

		// Convert grade codes to readable format
		gradeMap := map[string]string{
			"PK": "Pre-K", "KG": "K", "01": "1", "02": "2", "03": "3",
			"04": "4", "05": "5", "06": "6", "07": "7", "08": "8",
			"09": "9", "10": "10", "11": "11", "12": "12", "13": "13",
			"UG": "Ungraded", "AE": "Adult Ed",
		}

		lowDisplay := gradeMap[low]
		highDisplay := gradeMap[high]

		if lowDisplay == "" {
			lowDisplay = low
		}
		if highDisplay == "" {
			highDisplay = high
		}

		return fmt.Sprintf("%s - %s", lowDisplay, highDisplay)
	}
	return "N/A"
}

func (s *School) CharterString() string {
	if s.CharterText.Valid && s.CharterText.String != "" {
		charter := s.CharterText.String
		if charter == "Not applicable" || charter == "No" {
			return "No"
		}
		return "Yes"
	}
	return "N/A"
}

func (s *School) EnrollmentString() string {
	if s.Enrollment.Valid {
		return fmt.Sprintf("%d", s.Enrollment.Int64)
	}
	return "N/A"
}

func (s *School) StudentTeacherRatio() string {
	if s.Enrollment.Valid && s.Teachers.Valid && s.Teachers.Float64 > 0 {
		ratio := float64(s.Enrollment.Int64) / s.Teachers.Float64
		return fmt.Sprintf("%.1f:1", ratio)
	}
	return "N/A"
}

// SaveAIScraperCache saves AI scraper data to the database cache
func (d *DB) SaveAIScraperCache(ncessch, schoolName, sourceURL, markdownContent string, legacyData []byte, extractedAt time.Time) error {
	query := `
		INSERT INTO ai_scraper_cache (ncessch, school_name, source_url, markdown_content, legacy_data, extracted_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (ncessch) DO UPDATE SET
			school_name = EXCLUDED.school_name,
			source_url = EXCLUDED.source_url,
			markdown_content = EXCLUDED.markdown_content,
			legacy_data = EXCLUDED.legacy_data,
			extracted_at = EXCLUDED.extracted_at,
			created_at = CURRENT_TIMESTAMP
	`

	_, err := d.conn.Exec(query, ncessch, schoolName, sourceURL, markdownContent, string(legacyData), extractedAt)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to save AI scraper cache", "error", err, "ncessch", ncessch)
		}
		return fmt.Errorf("failed to save AI scraper cache: %w", err)
	}

	if logger != nil {
		logger.Info("Saved AI scraper data to database cache", "ncessch", ncessch, "school_name", schoolName)
	}

	return nil
}

// LoadAIScraperCache loads AI scraper data from the database cache
func (d *DB) LoadAIScraperCache(ncessch string, maxAge time.Duration) (schoolName, sourceURL, markdownContent string, legacyData []byte, extractedAt time.Time, err error) {
	query := `
		SELECT school_name, source_url, markdown_content, legacy_data, extracted_at
		FROM ai_scraper_cache
		WHERE ncessch = $1
	`

	var legacyDataStr sql.NullString
	err = d.conn.QueryRow(query, ncessch).Scan(&schoolName, &sourceURL, &markdownContent, &legacyDataStr, &extractedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", nil, time.Time{}, fmt.Errorf("no cache entry found")
		}
		if logger != nil {
			logger.Error("Failed to load AI scraper cache", "error", err, "ncessch", ncessch)
		}
		return "", "", "", nil, time.Time{}, fmt.Errorf("failed to load AI scraper cache: %w", err)
	}

	// Check if cache is expired
	if time.Since(extractedAt) > maxAge {
		return "", "", "", nil, time.Time{}, fmt.Errorf("cache expired")
	}

	if legacyDataStr.Valid && legacyDataStr.String != "" {
		legacyData = []byte(legacyDataStr.String)
	}

	if logger != nil {
		logger.Info("Loaded AI scraper data from database cache", "ncessch", ncessch, "age_hours", int(time.Since(extractedAt).Hours()))
	}

	return schoolName, sourceURL, markdownContent, legacyData, extractedAt, nil
}

// SaveNAEPCache saves NAEP data to the database cache
func (d *DB) SaveNAEPCache(ncessch, state, district string, stateScores, districtScores, nationalScores []byte, extractedAt time.Time) error {
	query := `
		INSERT INTO naep_cache (ncessch, state, district, state_scores, district_scores, national_scores, extracted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (ncessch) DO UPDATE SET
			state = EXCLUDED.state,
			district = EXCLUDED.district,
			state_scores = EXCLUDED.state_scores,
			district_scores = EXCLUDED.district_scores,
			national_scores = EXCLUDED.national_scores,
			extracted_at = EXCLUDED.extracted_at,
			created_at = CURRENT_TIMESTAMP
	`

	_, err := d.conn.Exec(query, ncessch, state, district, string(stateScores), string(districtScores), string(nationalScores), extractedAt)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to save NAEP cache", "error", err, "ncessch", ncessch)
		}
		return fmt.Errorf("failed to save NAEP cache: %w", err)
	}

	if logger != nil {
		logger.Info("Saved NAEP data to database cache", "ncessch", ncessch, "state", state)
	}

	return nil
}

// LoadNAEPCache loads NAEP data from the database cache
func (d *DB) LoadNAEPCache(ncessch string, maxAge time.Duration) (state, district string, stateScores, districtScores, nationalScores []byte, extractedAt time.Time, err error) {
	query := `
		SELECT state, district, state_scores, district_scores, national_scores, extracted_at
		FROM naep_cache
		WHERE ncessch = $1
	`

	var stateScoresStr, districtScoresStr, nationalScoresStr sql.NullString
	var districtNull sql.NullString

	err = d.conn.QueryRow(query, ncessch).Scan(&state, &districtNull, &stateScoresStr, &districtScoresStr, &nationalScoresStr, &extractedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil, nil, nil, time.Time{}, fmt.Errorf("no cache entry found")
		}
		if logger != nil {
			logger.Error("Failed to load NAEP cache", "error", err, "ncessch", ncessch)
		}
		return "", "", nil, nil, nil, time.Time{}, fmt.Errorf("failed to load NAEP cache: %w", err)
	}

	// Check if cache is expired
	if time.Since(extractedAt) > maxAge {
		return "", "", nil, nil, nil, time.Time{}, fmt.Errorf("cache expired")
	}

	if districtNull.Valid {
		district = districtNull.String
	}

	if stateScoresStr.Valid && stateScoresStr.String != "" {
		stateScores = []byte(stateScoresStr.String)
	}

	if districtScoresStr.Valid && districtScoresStr.String != "" {
		districtScores = []byte(districtScoresStr.String)
	}

	if nationalScoresStr.Valid && nationalScoresStr.String != "" {
		nationalScores = []byte(nationalScoresStr.String)
	}

	if logger != nil {
		logger.Info("Loaded NAEP data from database cache", "ncessch", ncessch, "age_days", int(time.Since(extractedAt).Hours()/24))
	}

	return state, district, stateScores, districtScores, nationalScores, extractedAt, nil
}
