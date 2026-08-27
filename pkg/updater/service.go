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
	"strings"
	"time"
)

const (
	CurrentAppVersion = "1.1.2"
	RepoOwner         = "PaJauKat"
	RepoName          = "KatHub"
)

type UpdateInfo struct {
	CurrentVersion   string `json:"currentVersion"`
	LatestVersion    string `json:"latestVersion"`
	UpdateAvailable  bool   `json:"updateAvailable"`
	DownloadURL      string `json:"downloadUrl"`
	ReleaseURL       string `json:"releaseUrl"`
}

type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ReleaseResponse struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
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

	for _, a := range release.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), ".zip") {
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
	// Simple semver compare
	latParts := strings.Split(latest, ".")
	curParts := strings.Split(current, ".")

	for i := 0; i < len(latParts) && i < len(curParts); i++ {
		if latParts[i] > curParts[i] {
			return true
		} else if latParts[i] < curParts[i] {
			return false
		}
	}
	return len(latParts) > len(curParts)
}

func DownloadAndApplyUpdate(downloadURL string) error {
	if downloadURL == "" {
		return errors.New("empty download URL")
	}

	tempZip := filepath.Join(os.TempDir(), "KatHub_Update.zip")
	tempExtract := filepath.Join(os.TempDir(), "KatHub_Update")

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
		return fmt.Errorf("download error: %d", resp.StatusCode)
	}

	out, err := os.Create(tempZip)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return err
	}
	out.Close()

	// Extract zip
	_ = os.RemoveAll(tempExtract)
	r, err := zip.OpenReader(tempZip)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		destPath := filepath.Join(tempExtract, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	// In Windows updater fashion, launch installer or open extracted folder
	_ = exec.Command("cmd", "/c", "start", tempExtract).Start()
	os.Exit(0)
	return nil
}
