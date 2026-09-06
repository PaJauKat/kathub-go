package katplugins

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ObtainLastVersionAndAsset returns the latest plugin version and its download
// URL served by the R2 bucket. The version comes from the object metadata
// (x-amz-meta-version), same as katloader and katmanager.
func ObtainLastVersionAndAsset() (string, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(pluginsDownloadURL)
	if err != nil {
		return "-1", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "-1", "", fmt.Errorf("r2 status: %d", resp.StatusCode)
	}

	version := resp.Header.Get("x-version")
	if version == "" {
		version = resp.Header.Get("x-amz-meta-version")
	}
	if version == "" {
		return "-1", "", errors.New("no version metadata on bucket object")
	}

	return strings.TrimPrefix(version, "v"), pluginsDownloadURL, nil
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
