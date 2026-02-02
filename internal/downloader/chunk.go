package downloader

import (
	"context"
	"errors"
	"file-downloader/internal/storage"
	"file-downloader/internal/ui"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const maxRetries = 3
const retryDelay = 5 * time.Second
const numWorkers = 8

type ChunkJob struct {
	url      string
	chunkNum int
	file     *os.File
}

type ChunkResult struct {
	chunkNum int
	err      error
}

func (d *Downloader) worker(ctx context.Context, jobs <-chan ChunkJob, results chan<- ChunkResult, wg *sync.WaitGroup, tracker ui.Tracker) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok || ctx.Err() != nil {
				return
			}
			
			result := ChunkResult{
				chunkNum: job.chunkNum,
				err:      nil,
			}

			for range maxRetries {
				result.err = d.downloadChunk(ctx, job.url, job.chunkNum, job.file, tracker)
				if result.err == nil {
					break // Останавливаем цикл  если чанк скачен
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(retryDelay):
					tracker.Logf("Error downloading chunk %d, try again after %v seconds...", job.chunkNum, retryDelay.Seconds())
				}
			}

			select {
			case <-ctx.Done():
				return
			case results <- result:
			}
		}
	}
}

// Скачивание файла с помощью чанков
func (d *Downloader) downloadChunks(ctx context.Context, url string, state *storage.DownloadState, file *os.File, tracker ui.Tracker) error {
	jobsCh := make(chan ChunkJob, numWorkers)
	resultsCh := make(chan ChunkResult, numWorkers)

	// для воркеров
	var workerWG sync.WaitGroup
	// для обработки результатов
	var consumerWG sync.WaitGroup

	// Запускаем воркеры
	for range numWorkers {
		workerWG.Add(1)
		go d.worker(ctx, jobsCh, resultsCh, &workerWG, tracker)
	}

	consumerWG.Add(1)
	// Горутина для обработки результатов
	go func(state *storage.DownloadState) {
		defer consumerWG.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-resultsCh:
				if !ok {
					return	// Канал закрыт, завершаем обработку
				}
				if result.err == nil {
					state.SetChunkUploaded(result.chunkNum)
				} else {
					tracker.Logf("Error downloading chunk %d: %s", result.chunkNum, result.err.Error())
				}
			}
		}
	}(state)

	chunksCount := len(state.DownloadedChunks)

	Loop:
	for chunk := range chunksCount {
		// Если чанк уже был загружен, то пропускаем
		if state.IsChunkDownloaded(chunk) {
			continue
		}

		// Загружаем один чанк
		job := ChunkJob{
			url:      url,
			chunkNum: chunk,
			file:     file,
		}
		select {
		case <-ctx.Done():
			break Loop	// Выходим из цикла если контекст закрыт
		case jobsCh <- job:
			// Ничего не делаем, работа отправлена
		}
	}

	close(jobsCh)
	workerWG.Wait()
	close(resultsCh)
	consumerWG.Wait()

	// Сохраняем прогресс в файл прогресса
	if err := state.SaveJSON(); err != nil {
		return err
	}
	return nil
}

// Скачивание одного чанка
func (d *Downloader) downloadChunk(ctx context.Context, url string, chunkNum int, file *os.File, tracker ui.Tracker) error {
	start := int64(d.chunkSize * chunkNum)
	end := start + int64(d.chunkSize) - 1

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := d.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			tracker.Logf("Chunk %d download canceled", chunkNum)
			return nil
		} else if errors.Is(err, context.DeadlineExceeded) {
			tracker.Logf("Chunk %d download timeout", chunkNum)
			return fmt.Errorf("timeout: %w", err)
		}
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Оборачиваем в ProxyReader чтобы видеть прогресс загрузки
	reader := tracker.ProxyReader(resp.Body)
	defer reader.Close()

	// Читаем кусок файла в память
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("reading body: %w", err)
	}

	// Записываем кусок в файл на нужное место
	_, err = file.WriteAt(data, start)
	if err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	return nil
}
