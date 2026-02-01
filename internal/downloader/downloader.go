package downloader

import (
	"context"
	"file-downloader/internal/storage"
	"file-downloader/internal/ui"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

const DefaultChunkSize = 5 * 1024 * 1024 // 5 МБ

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
func (d *Downloader) Download(ctx context.Context, url, dirPath string, c *ui.Container) error {
	// Получение метаданных
	meta, err := FetchMetadata(d.client, url, d.chunkSize)
	if err != nil {
		return fmt.Errorf("metadata fetch: %w", err)
	}

	fileName := path.Base(url)

	// Создание трекера для отслеживания прогресса загрузки
	tracker := c.AddBar(meta.Size, fileName)

	// Создание файла
	fullPath := filepath.Join(dirPath, fileName)
	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	// Выделяем память
	if err := os.Truncate(fullPath, meta.Size); err != nil {
		return err
	}

	// Загрузка данных
	if meta.Resumable && meta.ChunksCount > 1 {
		// Загрузка по чанкам
		state, loadErr := storage.LoadDownloadState(dirPath, fileName, d.chunkSize, meta.ChunksCount)
		if loadErr != nil {
			return loadErr
		}

		tracker.SetCurrent(state.DownloadedCount * int64(state.ChunkSize))
		err = d.downloadChunks(ctx, url, state, f, tracker)
	} else {
		// Загрузка обычная
		err = d.downloadSimple(url, f, tracker)
	}
	if err != nil {
		return err
	}

	return nil
}

// Загрузка напрямую без чанков
func (d *Downloader) downloadSimple(url string, dest io.Writer, tracker ui.Tracker) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Оборачиваем источник в reader для прогресс бара
	reader := tracker.ProxyReader(resp.Body)
	defer reader.Close()

	if _, err = io.Copy(dest, reader); err != nil {
		return fmt.Errorf("copying data: %w", err)
	}

	return nil
}
