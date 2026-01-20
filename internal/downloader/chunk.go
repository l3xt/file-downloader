package downloader

import (
	"file-downloader/internal/storage"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const maxRetries = 3
const retryDelay = 5 * time.Second
const numWorkers = 8

type ChunkJob struct {
	url      string
	chunkNum int
	ws     io.WriteSeeker
}

type ChunkResult struct {
	chunkNum int
	err      error
}

func (d *Downloader) worker(jobs <-chan ChunkJob, results chan<- ChunkResult, wg *sync.WaitGroup) {
	for j := range jobs {
		var err error
		result := ChunkResult{
			chunkNum: j.chunkNum,
			err:      nil,
		}

		for range maxRetries {
			err = d.downloadChunk(j.url, j.chunkNum, j.ws)
			if err == nil {
				fmt.Printf("Chunk %d downloaded successfully\n", j.chunkNum)
				results <- result
				wg.Done()
				break // Останавливаем цикл  если чанк скачен
			}

			// Если произошла ошибка
			fmt.Printf("Error downloading chunk %d, try again after %v seconds...\n", j.chunkNum, retryDelay.Seconds())
			time.Sleep(retryDelay)
		}

		// Если после всех попыток ошибка осталась
		if err != nil {
			result.err = err
			results <- result
		}
		wg.Done()
	}
}

// Скачивание файла с помощью чанков
func (d *Downloader) downloadChunks(url string, state *storage.DownloadState, dest io.WriteSeeker) error {
	chunksCount := len(state.DownloadedChunks)
	jobsCh := make(chan ChunkJob, chunksCount)
	resultsCh := make(chan ChunkResult, chunksCount)

	var wg sync.WaitGroup

	// Запускаем воркеры
	for range numWorkers {
		go d.worker(jobsCh, resultsCh, &wg)
	}

	for chunk := range chunksCount {
		// Если чанк уже был загружен, что пропускаем
		if state.DownloadedChunks[chunk] {
			continue
		}

		// Загружаем один чанк
		wg.Add(1)
		jobsCh <- ChunkJob{
			url:      url,
			chunkNum: chunk,
			ws:       dest,
		}
	}
	close(jobsCh)

	for range resultsCh {
		result := <-resultsCh
		if result.err != nil {
			return fmt.Errorf("chunk %d download: \n", result.chunkNum)
		}
	}

	// Сохраняем прогресс в файл прогресса
	if err := state.SaveJSON(); err != nil {
		return err
	}
	return nil
}

// Скачивание одного чанка
func (d *Downloader) downloadChunk(url string, chunkNum int, dest io.WriteSeeker) error {
	start := int64(d.chunkSize * chunkNum)
	end := start + int64(d.chunkSize) - 1

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server does not support Range (status %d)", resp.StatusCode)
	}

	// Позиционируемся в файле
	if _, err := dest.Seek(start, io.SeekStart); err != nil {
		return err
	}

	// Копируем данные
	if _, err := io.Copy(dest, resp.Body); err != nil {
		return err
	}

	return nil
}
