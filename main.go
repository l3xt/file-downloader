package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"sync"
	"time"

	"file-downloader/internal/downloader"
	"file-downloader/internal/sli"
	"file-downloader/internal/ui"
)

func main() {
	// Парсим аргументы
	config, err := sli.ParseArgs(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing args: %v\n", err)
		return
	}

	// Создаем UI контейнер
	var wg sync.WaitGroup
	c := ui.New(&wg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh, os.Interrupt)

	go func() {
		<- signalsCh
		c.Logf("Программа завершает свою работу...")
		cancel()
	}()

	// Создаём загрузчик
	dl := downloader.NewDownloader(30*time.Second, downloader.DefaultChunkSize)

	// Запускаем загрузки параллельно
	errorsCh := make(chan error, len(config.URLs))
	for _, url := range config.URLs {
		wg.Add(1)

		go func(u string) {
			defer wg.Done()
			if err := dl.Download(ctx, u, config.SavePath, c); err != nil {
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
