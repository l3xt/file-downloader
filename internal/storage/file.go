package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type SafeFile struct {
	File *os.File
	mu   sync.Mutex
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

func (sf *SafeFile) CopyAt(src io.Reader, startPos int64) error {
	sf.mu.Lock()
	// Позиционируемся в файле
	if _, err := sf.File.Seek(startPos, io.SeekStart); err != nil {
		return err
	}

	// Копируем данные
	if _, err := io.Copy(sf.File, src); err != nil {
		return err
	}
	sf.mu.Unlock()

	return nil
}

func (sf *SafeFile) Write(b []byte) (n int, err error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	return sf.File.Write(b)
}

func (sf *SafeFile) Seek(offset int64, whence int) (ret int64, err error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	return sf.File.Seek(offset, whence)
}
