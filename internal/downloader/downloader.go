package downloader

import (
	"file-downloader/internal/storage"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"
)

const DefaultChunkSize = 10 * 1024 * 1024 // 10 МБ

// Структура загручика
type Downloader struct {
	client    *http.Client
	chunkSize int
}

// Создание экземпляра
func NewDownloader(timeout time.Duration, chunkSize int) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: timeout,
		},
		chunkSize: chunkSize,
	}
}

// Загрузка одного url
func (d *Downloader) Download(url, dirPath string) error {
	fmt.Println("Start download file...")

	// Получение метаданных
	meta, err := FetchMetadata(d.client, url, d.chunkSize)
	if err != nil {
		return fmt.Errorf("metadata fetch: %w", err)
	}

	fmt.Printf("size: %d, support chunks: %v, chunks count: %d\n",
		meta.Size, meta.Resumable, meta.ChunksCount)

	// Создание файла
	fileName := path.Base(url)
	file, err := storage.PrepareFile(dirPath, fileName, meta.Size)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	// Загрузка данных
	if meta.Resumable && meta.ChunksCount > 1 {
		// Загрузка по чанкам
		state, loadErr := storage.LoadDownloadState(dirPath, fileName, d.chunkSize, meta.ChunksCount)
		if loadErr != nil {
			return loadErr
		}

		err = d.downloadChunks(url, state, file)
	} else {
		// Загрузка обычная
		err = d.downloadSimple(url, file)
	}
	if err != nil {
		return err
	}

	return nil
}

// Загрузка напрямую без чанков
func (d *Downloader) downloadSimple(url string, dest io.Writer) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	if _, err = io.Copy(dest, resp.Body); err != nil {
		return fmt.Errorf("copying data: %w", err)
	}

	return nil
}
