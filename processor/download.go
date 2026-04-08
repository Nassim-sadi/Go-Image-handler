package processor

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type DownloadResult struct {
	FilePath string
	FileName string
	Error    error
}

type Downloader struct {
	client     *http.Client
	workerPool int
	semaphore  chan struct{}
}

func NewDownloader(workers int) *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		workerPool: workers,
		semaphore:  make(chan struct{}, workers),
	}
}

func (d *Downloader) Download(urlStr string) (*DownloadResult, error) {
	result := &DownloadResult{}

	resp, err := d.client.Get(urlStr)
	if err != nil {
		return result, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return result, fmt.Errorf("not an image: %s", contentType)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return result, fmt.Errorf("invalid URL: %w", err)
	}

	fileName := filepath.Base(parsedURL.Path)
	if fileName == "" || strings.Contains(fileName, ".") == false {
		ext := d.getExtension(contentType)
		fileName = "image_" + time.Now().Format("20060102150405") + ext
	}

	result.FileName = fileName

	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, "imagehandler_"+fileName)

	out, err := os.Create(tmpPath)
	if err != nil {
		return result, fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		return result, fmt.Errorf("failed to write file: %w", err)
	}

	result.FilePath = tmpPath
	return result, nil
}

func (d *Downloader) DownloadBatch(urls []string, progressChan chan<- DownloadProgress) []DownloadResult {
	results := make([]DownloadResult, len(urls))
	var wg sync.WaitGroup

	for i, urlStr := range urls {
		d.semaphore <- struct{}{}
		wg.Add(1)

		go func(idx int, u string) {
			defer func() {
				<-d.semaphore
				wg.Done()
			}()

			progressChan <- DownloadProgress{Index: idx, Status: "downloading", Progress: 0}

			result, err := d.Download(u)
			results[idx] = *result
			results[idx].Error = err

			if err != nil {
				progressChan <- DownloadProgress{Index: idx, Status: "error", Error: err.Error()}
			} else {
				progressChan <- DownloadProgress{Index: idx, Status: "downloaded", Progress: 100}
			}
		}(i, urlStr)
	}

	wg.Wait()
	return results
}

func (d *Downloader) getExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "image/tiff":
		return ".tiff"
	default:
		return ".jpg"
	}
}

type DownloadProgress struct {
	Index    int
	Status   string
	Progress int
	Error    string
}
