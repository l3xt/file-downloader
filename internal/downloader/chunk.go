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
	sf       *storage.SafeFile
}

type ChunkResult struct {
	chunkNum int
	err      error
}

func (d *Downloader) worker(jobs <-chan ChunkJob, results chan<- ChunkResult, wg *sync.WaitGroup) {
	for j := range jobs {
		result := ChunkResult{
			chunkNum: j.chunkNum,
			err:      nil,
		}

		var err error
		for range maxRetries {
			err = d.downloadChunk(j.url, j.chunkNum, j.sf)
			if err == nil {
				fmt.Printf("Chunk %d downloaded successfully\n", j.chunkNum)
				break // Останавливаем цикл  если чанк скачен
			}

			// Если произошла ошибка
			fmt.Printf("Error downloading chunk %d, try again after %v seconds...\n", j.chunkNum, retryDelay.Seconds())
			time.Sleep(retryDelay)
		}

		// Устанавливаем ошибку если она есть
		result.err = err
		results <- result
		wg.Done()
	}
}

// Скачивание файла с помощью чанков
func (d *Downloader) downloadChunks(url string, state *storage.DownloadState, sf *storage.SafeFile) error {
	chunksCount := len(state.DownloadedChunks)
	jobsCh := make(chan ChunkJob, chunksCount)
	resultsCh := make(chan ChunkResult, chunksCount)

	var wg sync.WaitGroup

	// Запускаем воркеры
	for range numWorkers {
		go d.worker(jobsCh, resultsCh, &wg)
	}

	go func() {
		for result := range resultsCh {
			if result.err == nil {
				state.DownloadedChunks[result.chunkNum] = true
			} else {
				fmt.Printf("Error downloading chunk %d: %s\n", result.chunkNum, result.err.Error())
			}
		}
	}()

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
			sf:       sf,
		}
	}

	close(jobsCh)
	wg.Wait()
	close(resultsCh)

	// Сохраняем прогресс в файл прогресса
	if err := state.SaveJSON(); err != nil {
		return err
	}
	return nil
}

// Скачивание одного чанка
func (d *Downloader) downloadChunk(url string, chunkNum int, sf *storage.SafeFile) error {
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
	sf.Mu.Lock()
	if _, err := sf.File.Seek(start, io.SeekStart); err != nil {
		return err
	}

	// Копируем данные
	if _, err := io.Copy(sf.File, resp.Body); err != nil {
		return err
	}
	sf.Mu.Unlock()

	return nil
}
