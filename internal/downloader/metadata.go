package downloader

import (
	"fmt"
	"net/http"
	"strconv"
)

// Структура с метаданными
type FileMetadata struct {
	Size int64
	Resumable bool
	ChunksCount int
}

func FetchMetadata(client *http.Client, url string, chunkSize int) (*FileMetadata, error) {
	resp, err := client.Head(url)
	if err != nil {
		return nil, fmt.Errorf("head request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	// Получаем значение заголовка Content-Length
	sizeStr := resp.Header.Get("Content-Length")
	if sizeStr == "" {
		return nil, fmt.Errorf("header Content-Length missing")
	}

	// Конвертация размера в Int 
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Length value: %w", err)
	}

	// Поддерживает ли URL Range
	resumable := resp.Header.Get("Accept-Ranges") == "bytes"

	chunks := int((size + int64(chunkSize) - 1) / int64(chunkSize))

	return &FileMetadata{
		Size: size,
		Resumable: resumable,
		ChunksCount: chunks,
	}, nil
}