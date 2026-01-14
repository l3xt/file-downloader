package downloader

import (
	"fmt"
	"io"
	"net/http"
)

func (d *Downloader) downloadChunks(url string, meta *FileMetadata, dest io.WriteSeeker) error {
	for chunk := 0; chunk < meta.ChunksCount; chunk++ {
		if err := d.downloadChunk(url, chunk, dest); err != nil {
			return err
		}
	}
	return nil
}

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


