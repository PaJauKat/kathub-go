package katplugins

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func DescargarArchivo(filePath string, url string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", filePath, err)
	}
	defer f.Close()

	client := &http.Client{Timeout: 2 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "KatHub")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error during download request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download error: HTTP status %s", resp.Status)
	}

	archiveVersion := resp.Header.Get("x-version")

	if _, err = io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to save downloaded content: %w", err)
	}

	if archiveVersion != "" {
		vErr := os.WriteFile(filePath+":kversion", []byte(archiveVersion), 0644)
		if vErr != nil {
			return fmt.Errorf("failed to write version stream for %s: %w", filePath, vErr)
		}
	}

	return nil
}
