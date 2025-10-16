package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DataFile represents a required data file with its download URL
type DataFile struct {
	Name string
	URL  string
}

// RequiredDataFiles lists all the data files needed for the application
var RequiredDataFiles = []DataFile{
	{
		Name: "ccd_sch_029_2324_w_1a_073124.csv",
		URL:  "https://nces.ed.gov/ccd/Data/zip/ccd_sch_029_2324_w_1a_073124.zip",
	},
	{
		Name: "ccd_sch_059_2324_l_1a_073124.csv",
		URL:  "https://nces.ed.gov/ccd/Data/zip/ccd_sch_059_2324_l_1a_073124.zip",
	},
	{
		Name: "ccd_sch_052_2324_l_1a_073124.csv",
		URL:  "https://nces.ed.gov/ccd/Data/zip/ccd_sch_052_2324_l_1a_073124.zip",
	},
}

// CheckDataFiles checks if all required data files exist in the data directory
func CheckDataFiles(dataDir string) ([]DataFile, error) {
	var missing []DataFile

	for _, file := range RequiredDataFiles {
		filePath := filepath.Join(dataDir, file.Name)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missing = append(missing, file)
		}
	}

	return missing, nil
}

// PromptUserForDownload asks the user if they want to download missing files
func PromptUserForDownload(missing []DataFile) bool {
	if len(missing) == 0 {
		return false
	}

	fmt.Println("\n⚠️  Missing required data files:")
	for _, file := range missing {
		fmt.Printf("   - %s\n", file.Name)
	}
	fmt.Println("\nThese files are required for the School Finder to work.")
	fmt.Printf("Total download size: ~%d files (several hundred MB)\n", len(missing))
	fmt.Print("\nWould you like to download them now? (y/N): ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}

// DownloadFile downloads a file from a URL and shows progress
func DownloadFile(filepath string, url string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Get the file size
	size := resp.ContentLength

	// Create a progress reader
	counter := &ProgressCounter{
		Total: size,
		Name:  filepath,
	}

	// Copy with progress
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	fmt.Println() // New line after progress

	return err
}

// ProgressCounter counts bytes as they're written and displays progress
type ProgressCounter struct {
	Total   int64
	Current int64
	Name    string
}

func (pc *ProgressCounter) Write(p []byte) (int, error) {
	n := len(p)
	pc.Current += int64(n)

	// Calculate percentage
	percentage := float64(pc.Current) / float64(pc.Total) * 100

	// Print progress
	fmt.Printf("\r   Downloading %s... %.1f%% (%d/%d MB)",
		filepath.Base(pc.Name),
		percentage,
		pc.Current/1024/1024,
		pc.Total/1024/1024)

	return n, nil
}

// UnzipFile extracts a zip file to a destination directory
func UnzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Skip directories and only extract CSV files
		if f.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}

		// Construct the target path
		fpath := filepath.Join(dest, filepath.Base(f.Name))

		// Open the file in the zip
		rc, err := f.Open()
		if err != nil {
			return err
		}

		// Create the destination file
		outFile, err := os.Create(fpath)
		if err != nil {
			rc.Close()
			return err
		}

		// Copy the content
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}

		fmt.Printf("   ✓ Extracted: %s\n", filepath.Base(fpath))
	}

	return nil
}

// DownloadAndExtractFiles downloads and extracts all missing data files
func DownloadAndExtractFiles(dataDir string, missing []DataFile) error {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create a temp directory for downloads
	tempDir := filepath.Join(dataDir, ".temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory

	fmt.Println("\n📥 Downloading data files...")
	fmt.Println("This may take several minutes depending on your connection.\n")

	for i, file := range missing {
		fmt.Printf("[%d/%d] Processing %s...\n", i+1, len(missing), file.Name)

		// Download the ZIP file
		zipPath := filepath.Join(tempDir, filepath.Base(file.URL))
		if err := DownloadFile(zipPath, file.URL); err != nil {
			return fmt.Errorf("failed to download %s: %w", file.URL, err)
		}

		// Extract the ZIP file
		fmt.Printf("   Extracting...\n")
		if err := UnzipFile(zipPath, dataDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", zipPath, err)
		}

		// Remove the ZIP file
		os.Remove(zipPath)

		fmt.Println()
	}

	fmt.Println("✅ All data files downloaded and extracted successfully!\n")
	return nil
}
