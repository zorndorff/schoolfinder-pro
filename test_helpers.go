package main

import (
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
