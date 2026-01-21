package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type SafeFile struct {
	File *os.File
	Mu   sync.Mutex
}

func NewSafeFile(dirPath, fileName string, size int64) (*SafeFile, error) {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("creating a directory: %w", err)
	}

	fullPath := filepath.Join(dirPath, fileName)

	f, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("creating a file: %w", err)
	}

	// Выделяем память под будущий файл
	err = f.Truncate(size)
	if err != nil {
		return nil, fmt.Errorf("memory allocation: %w", err)
	}

	return &SafeFile{
		File: f,
	}, nil
}
