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

// GetFileSize gets the size of a file from URL using HEAD request
func GetFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status: %s", resp.Status)
	}

	return resp.ContentLength, nil
}

// GetTotalDownloadSize calculates the total size of all missing files
func GetTotalDownloadSize(missing []DataFile) (int64, error) {
	var totalSize int64
	
	for _, file := range missing {
		size, err := GetFileSize(file.URL)
		if err != nil {
			// If we can't get the size for one file, continue with others
			fmt.Printf("   Warning: Could not get size for %s: %v\n", file.Name, err)
			continue
		}
		totalSize += size
	}
	
	return totalSize, nil
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
	
	// Get total download size
	totalSize, err := GetTotalDownloadSize(missing)
	if err != nil || totalSize == 0 {
		fmt.Println("\nThese files are required for the School Finder to work.")
		fmt.Printf("Total download size: ~%d files (size unknown)\n", len(missing))
	} else {
		totalMB := totalSize / 1024 / 1024
		fmt.Println("\nThese files are required for the School Finder to work.")
		fmt.Printf("Total download size: ~%d files (%d MB)\n", len(missing), totalMB)
	}
	
	fmt.Print("\nWould you like to download them now? (y/N): ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}

// DownloadFileWithProgress downloads a file with enhanced progress tracking
func DownloadFileWithProgress(filepath string, url string, fileIndex, totalFiles int, fileSize, totalSize int64) error {
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

	// Get the file size (use the provided size if available, otherwise use response)
	size := fileSize
	if size == 0 {
		size = resp.ContentLength
	}

	// Create a progress reader with enhanced tracking
	counter := &ProgressCounter{
		Total:     size,
		Name:      filepath,
		FileIndex: fileIndex,
		TotalFiles: totalFiles,
	}

	// Copy with progress
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	fmt.Println() // New line after progress

	return err
}

// ProgressCounter counts bytes as they're written and displays progress
type ProgressCounter struct {
	Total     int64
	Current   int64
	Name      string
	FileIndex int
	TotalFiles int
}

func (pc *ProgressCounter) Write(p []byte) (int, error) {
	n := len(p)
	pc.Current += int64(n)

	// Calculate percentage and display appropriate format
	var percentage float64
	var currentMB, totalMB int64
	
	currentMB = pc.Current / 1024 / 1024
	
	if pc.Total > 0 {
		percentage = float64(pc.Current) / float64(pc.Total) * 100
		totalMB = pc.Total / 1024 / 1024
		// Print progress with known size and file count
		fmt.Printf("\r   Downloading %s... %.1f%% (%d/%d MB) [%d/%d]",
			filepath.Base(pc.Name),
			percentage,
			currentMB,
			totalMB,
			pc.FileIndex,
			pc.TotalFiles)
	} else {
		// Print progress with unknown size and file count
		fmt.Printf("\r   Downloading %s... %d MB downloaded [%d/%d]",
			filepath.Base(pc.Name),
			currentMB,
			pc.FileIndex,
			pc.TotalFiles)
	}

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

		// Get individual file size for progress tracking
		fileSize, _ := GetFileSize(file.URL)

		// Download the ZIP file with enhanced progress
		zipPath := filepath.Join(tempDir, filepath.Base(file.URL))
		if err := DownloadFileWithProgress(zipPath, file.URL, i+1, len(missing), fileSize, 0); err != nil {
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
