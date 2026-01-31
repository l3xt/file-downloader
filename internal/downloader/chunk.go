package downloader

import (
	"context"
	"file-downloader/internal/storage"
	"file-downloader/internal/ui"
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

func (d *Downloader) worker(ctx context.Context, jobs <-chan ChunkJob, results chan<- ChunkResult, wg *sync.WaitGroup, tracker ui.Tracker) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			tracker.Logf("Worker: context closed")
			return
		case job, opened := <- jobs:
			if !opened {
				// Канал закрыт, работы больше нет
				tracker.Logf("Worker: channel closed")
				return
			}
			
			if ctx.Err() != nil {
				tracker.Logf("Worker: chunk %d claimed, but context done", job.chunkNum)
				return
			}

			result := ChunkResult{
				chunkNum: job.chunkNum,
				err:      nil,
			}

			var err error
			for range maxRetries {
				err = d.downloadChunk(job.url, job.chunkNum, job.sf, tracker)
				if err == nil {
					tracker.Logf("Чанк %d загружен.", job.chunkNum)
					break // Останавливаем цикл  если чанк скачен
				}

				// Если произошла ошибка
				tracker.Logf("Error downloading chunk %d, try again after %v seconds...", job.chunkNum, retryDelay.Seconds())
				
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryDelay):
					// Ничего не делаем, чтобы код продолжился и цикл пошел в ретрай
				}
			}

			// Устанавливаем ошибку если она есть
			result.err = err
			
			select {
			case <- ctx.Done():
				return
			case results <- result:
			}
		}
	}
}

// Скачивание файла с помощью чанков
func (d *Downloader) downloadChunks(ctx context.Context, url string, state *storage.DownloadState, sf *storage.SafeFile, tracker ui.Tracker) error {
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

	consumerWG.Go(func() {
		for result := range resultsCh {
			if result.err == nil {
				state.SetChunkUploaded(result.chunkNum)
			} else {
				tracker.Logf("Error downloading chunk %d: %s", result.chunkNum, result.err.Error())
			}
		}
	})

	chunksCount := len(state.DownloadedChunks)
	for chunk := range chunksCount {
		// Если чанк уже был загружен, то пропускаем
		if state.IsChunkDownloaded(chunk) {
			continue
		}

		// Загружаем один чанк
		jobsCh <- ChunkJob{
			url:      url,
			chunkNum: chunk,
			sf:       sf,
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
func (d *Downloader) downloadChunk(url string, chunkNum int, sf *storage.SafeFile, tracker ui.Tracker) error {
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

	// Блокируем мьютекс
	sf.Lock()
	defer sf.Unlock()
	
	// Позиционируемся в файле
	if _, err := sf.File.Seek(start, io.SeekStart); err != nil {
		return err
	}

	// Оборачиваем источник в reader для прогресс бара
	reader := tracker.ProxyReader(resp.Body)
	defer reader.Close()

	// Копируем данные
	if _, err := io.Copy(sf.File, reader); err != nil {
		return err
	}

	return nil
}
