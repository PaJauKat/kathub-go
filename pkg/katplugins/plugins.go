package katplugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName string               `json:"tag_name"`
	Assets  []GitHubReleaseAsset `json:"assets"`
}

func ObtainLastVersionAndAsset() (string, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", gitHubRepoOwner, gitHubRepoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "-1", "", err
	}
	req.Header.Set("User-Agent", "KatHub")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "-1", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "-1", "", fmt.Errorf("github api status: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "-1", "", err
	}

	version := strings.TrimPrefix(release.TagName, "v")
	downloadURL := ""
	for _, a := range release.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), ".jar") {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return "-1", "", errors.New("no jar asset found")
	}

	return version, downloadURL, nil
}

func ObtainInstalledPluginsVersion() string {
	_, pluginsDir, _ := getRuneLiteDirs()
	matches, err := filepath.Glob(filepath.Join(pluginsDir, "KatPlugins-*.jar"))
	if err != nil || len(matches) == 0 {
		return "-1"
	}

	if len(matches) > 1 {
		return "-69" // Multiple versions indicator
	}

	// The release jar is named KatPlugins-<major>.<minor>.jar (e.g. KatPlugins-2.24.jar)
	re := regexp.MustCompile(`KatPlugins-(\d+)\.(\d+)\.jar$`)
	m := re.FindStringSubmatch(filepath.Base(matches[0]))
	if m != nil {
		return m[1] + "." + m[2]
	}

	return "-1"
}

// CompareVersions compares two semver strings (v1 and v2).
// Returns 1 if v1 > v2, -1 if v1 < v2, and 0 if v1 == v2.
func CompareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(strings.TrimSpace(v1), "v")
	v2 = strings.TrimPrefix(strings.TrimSpace(v2), "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		num1 := 0
		if i < len(parts1) {
			num1, _ = strconv.Atoi(parts1[i])
		}

		num2 := 0
		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}

		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}

	return 0
}
