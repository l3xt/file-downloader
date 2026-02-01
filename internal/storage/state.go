package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

const fileExtension string = ".progress"

// Структура с информацией о прогрессе загружаемого файла
type DownloadState struct {
	Dir              string `json:"dir"`
	Name             string `json:"name"`
	ChunkSize        int    `json:"chunkSize"`
	DownloadedChunks []bool `json:"downloadedChunks"`
	DownloadedCount  int64  `json:"downloadedCount"`
}

// Создание экземпляра DownloadState с валидацией
func NewDownloadState(dir, name string, chunkSize, totalChunks int) (*DownloadState, error) {
	if name == "" {
		return nil, fmt.Errorf("invalid file name: %s", name)
	}
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be greater than zero, got: %d", chunkSize)
	}
	if totalChunks <= 0 {
		return nil, fmt.Errorf("number of all chunks must be greater than zero, got: %d", totalChunks)
	}

	return &DownloadState{
		Dir:              dir,
		Name:             name + fileExtension,
		ChunkSize:        chunkSize,
		DownloadedChunks: make([]bool, totalChunks),
		DownloadedCount:  0,
	}, nil
}

func (s *DownloadState) SaveJSON() error {
	fullPath := filepath.Join(s.Dir, s.Name)

	data, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(fullPath, data, 0666)
	if err != nil {
		return err
	}
	return nil
}

// Загружает экземляр структуры из json
func LoadJSON(dir, name string) (*DownloadState, error) {
	if name == "" {
		return nil, fmt.Errorf("invalid file name: %s", name)
	}

	fullPath := filepath.Join(dir, name+fileExtension)

	if _, err := os.Stat(fullPath); err != nil {
		return nil, err
	}

	// reading file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	var state DownloadState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Создает или загружает существующий
func LoadDownloadState(dir, name string, chunkSize, totalChunks int) (*DownloadState, error) {
	state, err := LoadJSON(dir, name)
	if err == nil {
		return state, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return NewDownloadState(dir, name, chunkSize, totalChunks)
}

func (s *DownloadState) SetChunkUploaded(chunkIdx int) {
	// Если уже есть запись о загруженном чанке, то выходим
	if s.DownloadedChunks[chunkIdx] {
		return
	}

	s.DownloadedChunks[chunkIdx] = true
	atomic.AddInt64(&s.DownloadedCount, 1)
}

func (s *DownloadState) IsChunkDownloaded(chunkIdx int) bool {
	return s.DownloadedChunks[chunkIdx]
}
