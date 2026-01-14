package downloader

import (
	"file-downloader/internal/storage"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

)

const DefaultChunkSize = 10 * 1024 * 1024	// 10 МБ

type Downloader struct {
	client *http.Client
	chunkSize int
}

func New(timeout time.Duration, chunkSize int) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: timeout,
		},
		chunkSize: chunkSize,
	}
}

func (d *Downloader) Download(url, dirPath string) error {
	fmt.Println("Начинаю скачивать файл...")
	
	// Получение метаданных
	meta, err := FetchMetadata(d.client, url, d.chunkSize)
	if err != nil {
		return fmt.Errorf("metadata fetch failed: %w", err)
	}

    fmt.Printf("Размер: %d, докачка: %v, чанки: %d\n", 
		meta.Size, meta.Resumable, meta.ChunksCount)
	
	// Создание файла
	fileName := path.Base(url)
	file, err := storage.PrepareFile(dirPath, fileName, meta.Size)
	if err != nil {
		return fmt.Errorf("file create failed: %w", err)
	}
	defer file.Close()

	// Загрузка данных
	if meta.Resumable && meta.ChunksCount > 1 {
		// Загрузка по чанкам
		err = d.downloadChunks(url, meta, file)
	} else {
		// Загрузка обычная
		err = d.downloadSimple(url, file)
	}
	if err != nil {
		return err
	}

	return nil
}

func (d *Downloader) downloadSimple(url string, dest io.Writer) error {
	resp, err := d.client.Get(url)
	if err != nil {
		return fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	if _, err = io.Copy(dest, resp.Body); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

