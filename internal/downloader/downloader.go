package downloader

import (
	"context"
	"file-downloader/internal/storage"
	"file-downloader/internal/ui"
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
	sf, err := storage.NewSafeFile(dirPath, fileName, meta.Size)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	// Загрузка данных
	if meta.Resumable && meta.ChunksCount > 1 {
		// Загрузка по чанкам
		state, loadErr := storage.LoadDownloadState(dirPath, fileName, d.chunkSize, meta.ChunksCount)
		if loadErr != nil {
			return loadErr
		}

		tracker.SetCurrent(int64(state.DownloadedCount * state.ChunkSize))
		err = d.downloadChunks(ctx, url, state, sf, tracker)
	} else {
		// Загрузка обычная
		err = d.downloadSimple(url, sf.File, tracker)
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
