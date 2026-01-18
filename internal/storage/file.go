package storage

import (
	"fmt"
	"os"
	"path/filepath"
)



func PrepareFile(dirPath, fileName string, size int64) (*os.File, error) {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("creating a directory: %w", err)
	} 

	fullPath := filepath.Join(dirPath, fileName)

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("creating a file: %w", err)
	}

	// Allocating memory for a future file
	err = file.Truncate(size)
	if err != nil {
		return nil, fmt.Errorf("memory allocation: %w", err)
	}

	return file, nil
}
