package katplugins

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// IsRuneLiteRunning checks if RuneLite or RuneLiteLauncher process is currently active.
func IsRuneLiteRunning() bool {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return false
	}

	for {
		exeName := strings.ToLower(windows.UTF16ToString(entry.ExeFile[:]))
		if exeName == "runelite.exe" || exeName == "runelitelauncher.exe" {
			return true
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			break
		}
	}
	return false
}

// isFileLockOrPermissionError checks if an error is due to file locks or permission issues.
func isFileLockOrPermissionError(err error) bool {
	if err == nil {
		return false
	}
	if os.IsPermission(err) || errors.Is(err, os.ErrPermission) {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "access is denied") ||
		strings.Contains(errStr, "used by another process") ||
		strings.Contains(errStr, "sharing violation") ||
		strings.Contains(errStr, "permission denied")
}

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

	// R2 serves upload metadata as x-amz-meta-version; accept the rewriten
	// header too for CDNs configured to emit x-version.
	archiveVersion := resp.Header.Get("x-version")
	if archiveVersion == "" {
		archiveVersion = resp.Header.Get("x-amz-meta-version")
	}

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
