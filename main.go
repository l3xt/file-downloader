package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"file-downloader/internal/downloader"
	"file-downloader/internal/sli"
)

func main() {
	// Парсим аргументы
	config, err := sli.ParseArgs(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing args: %v\n", err)
		return
	}

	// Создаём загрузчик
	dl := downloader.NewDownloader(30*time.Second, downloader.DefaultChunkSize)

	// Запускаем загрузки параллельно
	var wg sync.WaitGroup
	errorsCh := make(chan error, len(config.URLs))

	for _, url := range config.URLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			if err := dl.Download(u, config.SavePath); err != nil {
				errorsCh <- fmt.Errorf("processing url %s: %w", u, err)
			}
		}(url)
	}

	// Ожидаем завершения
	go func() {
		wg.Wait()
		close(errorsCh)
	}()

	// Выводим ошибки
	for err := range errorsCh {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}
