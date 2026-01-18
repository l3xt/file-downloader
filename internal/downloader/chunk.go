package downloader

import (
	"file-downloader/internal/storage"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxRetries = 3
const retryDelay = 5 * time.Second

// Скачивание файла с помощью чанков
func (d *Downloader) downloadChunks(url string, state *storage.DownloadState, dest io.WriteSeeker) error {
	for chunk := 0; chunk < len(state.DownloadedChunks); chunk++ {
		// Если чанк уже был загружен, что пропускаем
		if state.DownloadedChunks[chunk] {
			continue
		}

		// Загружаем один чанк
		var err error = nil
		for range maxRetries {
			err = d.downloadChunk(url, chunk, dest)
			if err == nil {
				fmt.Printf("Chunk %d downloaded successfully\n", chunk)
				break	// Останавливаем цикл  если чанк скачен
			}

			// Если произошла ошибка
			fmt.Printf("Error downloading chunk %d, try again after %v seconds...\n", chunk, retryDelay.Seconds())
			time.Sleep(retryDelay)
		}

		// Если после всех попыток остается ошибка, то обрываем загрузку остальных чанков
		if err != nil {
			state.SaveJSON()
			return fmt.Errorf("load chunk %d: %w", chunk, err)
		}

		// Делаем пометку что чанк скачан
		state.DownloadedChunks[chunk] = true
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
