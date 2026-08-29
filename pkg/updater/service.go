package updater

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	CurrentAppVersion = "0.0.0"
)

const (
	RepoOwner = "PaJauKat"
	RepoName  = "kathub-go"
)

type UpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl"`
	ReleaseURL      string `json:"releaseUrl"`
	ReleaseNotes    string `json:"releaseNotes"`
}

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ReleaseResponse struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Body    string         `json:"body"`
	Assets  []ReleaseAsset `json:"assets"`
}

func CheckForAppUpdates() (UpdateInfo, error) {
	info := UpdateInfo{
		CurrentVersion:  CurrentAppVersion,
		LatestVersion:   CurrentAppVersion,
		UpdateAvailable: false,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("User-Agent", "KatHub-Updater")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("github api status: %d", resp.StatusCode)
	}

	var release ReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return info, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	info.LatestVersion = latestVersion
	info.ReleaseURL = release.HTMLURL
	info.ReleaseNotes = release.Body

	// Prioritize .exe, .zip, or standalone setup
	for _, a := range release.Assets {
		name := strings.ToLower(a.Name)
		
		if strings.Contains(name, "kathub.exe") {
			info.DownloadURL = a.BrowserDownloadURL
			break
		}
	}

	if isVersionNewer(latestVersion, CurrentAppVersion) {
		info.UpdateAvailable = true
	}

	return info, nil
}

func isVersionNewer(latest, current string) bool {
	latParts := strings.Split(latest, ".")
	curParts := strings.Split(current, ".")

	maxLen := len(latParts)
	if len(curParts) > maxLen {
		maxLen = len(curParts)
	}

	for i := 0; i < maxLen; i++ {
		var latNum, curNum int
		if i < len(latParts) {
			latNum, _ = strconv.Atoi(latParts[i])
		}
		if i < len(curParts) {
			curNum, _ = strconv.Atoi(curParts[i])
		}
		if latNum > curNum {
			return true
		} else if latNum < curNum {
			return false
		}
	}
	return false
}

// ApplySeamlessUpdate downloads update, schedules self-replacement via batch script, and restarts KatHub
func ApplySeamlessUpdate(downloadURL string) error {
	if downloadURL == "" {
		return errors.New("empty download URL")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return err
	}

	tempFile := filepath.Join(os.TempDir(), "KatHub_Update_Download")
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "KatHub-Updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download error: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(tempFile)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}
	out.Close()

	var newExePath string
	if strings.HasSuffix(strings.ToLower(downloadURL), ".zip") {
		zipExtractDir := filepath.Join(os.TempDir(), "KatHub_Update_Extracted")
		_ = os.RemoveAll(zipExtractDir)
		if err := unzipFile(tempFile, zipExtractDir); err != nil {
			return err
		}

		// Find .exe inside extracted zip
		foundExe := ""
		_ = filepath.Walk(zipExtractDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".exe") {
				foundExe = path
				return filepath.SkipDir
			}
			return nil
		})

		if foundExe == "" {
			return errors.New("no executable found in downloaded update archive")
		}
		newExePath = foundExe
	} else {
		newExePath = tempFile
	}

	// Create seamless update batch script
	updaterBat := filepath.Join(os.TempDir(), "kathub_update.bat")
	pid := os.Getpid()

	batContent := fmt.Sprintf(`@echo off
timeout /t 1 /nobreak > NUL
:WAIT_LOOP
tasklist /fi "PID eq %d" 2>NUL | find "%d" > NUL
if %%ERRORLEVEL%% == 0 (
    timeout /t 1 /nobreak > NUL
    goto WAIT_LOOP
)
copy /y "%s" "%s"
start "" "%s"
del "%%~f0"
`, pid, pid, newExePath, currentExe, currentExe)

	if err := os.WriteFile(updaterBat, []byte(batContent), 0755); err != nil {
		return err
	}

	// Launch updater script detached without showing command window
	cmd := exec.Command("cmd.exe", "/c", updaterBat)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return err
	}

	os.Exit(0)
	return nil
}

func unzipFile(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
