package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// FileManager handles file operations for the application.
type FileManager struct {
	OutputDir string
}

// NewFileManager creates a new FileManager with the specified output directory.
func NewFileManager(outputDir string) *FileManager {
    // Expand home directory if path starts with ~
    expandedPath := expandHomePath(outputDir)
    return &FileManager{OutputDir: expandedPath}
}

// SaveMarkdownFile saves the given content to a markdown file in the output directory.
func (fm *FileManager) SaveMarkdownFile(filename string, content string) error {
	// Create the output directory if it doesn't exist
	if err := os.MkdirAll(fm.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Define the full path for the markdown file
	filePath := filepath.Join(fm.OutputDir, filename)

	// Write the content to the markdown file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	return nil
}

// SaveMarkdown saves markdown content to a file in the specified directory
func (fm *FileManager) SaveMarkdown(content, title, folderName string) (string, error) {
    // Create folder if it doesn't exist
    folderPath := filepath.Join(fm.OutputDir, folderName)
    err := os.MkdirAll(folderPath, 0755)
    if err != nil {
        return "", fmt.Errorf("failed to create directory: %w", err)
    }

    // Create a safe filename from the title
    filename := generateSafeFilename(title)
    
    // Add date to filename for uniqueness
    date := time.Now().Format("2006-01-02")
    filename = fmt.Sprintf("%s-%s.md", date, filename)
    
    // Full path
    fullPath := filepath.Join(folderPath, filename)
    
    // Check if file exists and add suffix if needed
    counter := 1
    for {
        if _, err := os.Stat(fullPath); os.IsNotExist(err) {
            break
        }
        
        // File exists, add counter suffix
        filename = fmt.Sprintf("%s-%s-%d.md", date, generateSafeFilename(title), counter)
        fullPath = filepath.Join(folderPath, filename)
        counter++
    }

    // Write content to file
    err = os.WriteFile(fullPath, []byte(content), 0644)
    if err != nil {
        return "", fmt.Errorf("failed to write content to file: %w", err)
    }

    return fullPath, nil
}

// generateSafeFilename creates a filesystem-safe filename from the title
func generateSafeFilename(title string) string {
    // Convert to lowercase
    title = strings.ToLower(title)
    
    // Remove special characters
    reg := regexp.MustCompile(`[^a-z0-9\s-]`)
    title = reg.ReplaceAllString(title, "")
    
    // Convert spaces to hyphens
    title = strings.ReplaceAll(title, " ", "-")
    
    // Remove consecutive hyphens
    reg = regexp.MustCompile(`-{2,}`)
    title = reg.ReplaceAllString(title, "-")
    
    // Trim hyphens from start/end
    title = strings.Trim(title, "-")
    
    // Limit length
    maxLen := 50
    if len(title) > maxLen {
        title = title[:maxLen]
    }
    
    // Ensure we have at least some characters
    if title == "" {
        title = "untitled"
    }
    
    return title
}

// expandHomePath expands the tilde character to user's home directory
func expandHomePath(path string) string {
    if strings.HasPrefix(path, "~/") {
        home, err := os.UserHomeDir()
        if err == nil {
            return filepath.Join(home, path[2:])
        }
    }
    return path
}