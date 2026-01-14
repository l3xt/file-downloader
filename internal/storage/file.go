package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func PrepareFile(dirPath, fileName string, size int64) (*os.File, error) {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	} 

	fullPath := filepath.Join(dirPath, fileName)

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	err = file.Truncate(size)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate space: %w", err)
	}

	return file, nil
}
