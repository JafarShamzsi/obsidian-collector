package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileManager handles file operations for the application.
type FileManager struct {
	OutputDir string
}

// NewFileManager creates a new FileManager with the specified output directory.
func NewFileManager(outputDir string) *FileManager {
	return &FileManager{OutputDir: outputDir}
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