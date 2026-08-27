package katplugins

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/net/context"
)

const (
	gitHubRepoOwner = "PaJauKat"
	gitHubRepoName  = "PaJau-plugins"
	loaderFileName  = "kat.jar"
)

type KatPluginsState string

const (
	StateUnknown          KatPluginsState = "Unknown"
	StateCrackear         KatPluginsState = "Crackear"
	StateInstall          KatPluginsState = "Install"
	StateUpdatePlugins    KatPluginsState = "Update"
	StateUpdateLoader     KatPluginsState = "UpdateLoader"
	StateUpToDate         KatPluginsState = "UpToDate"
	StateMultipleVersions KatPluginsState = "MultipleVersions"
)

type KatPluginsInfo struct {
	State             KatPluginsState `json:"state"`
	StatusText        string          `json:"statusText"`
	StatusColor       string          `json:"statusColor"`
	ButtonText        string          `json:"buttonText"`
	ButtonColor       string          `json:"buttonColor"`
	MainButtonVisible bool            `json:"mainButtonVisible"`
	UninstallVisible  bool            `json:"uninstallVisible"`
	CurrentVersion    string          `json:"currentVersion"`
	LatestVersion     string          `json:"latestVersion"`
}

type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName string               `json:"tag_name"`
	Assets  []GitHubReleaseAsset `json:"assets"`
}

type RuneLiteConfig struct {
	MainClass string   `json:"mainClass"`
	ClassPath []string `json:"classPath"`
	// Preserve other dynamic keys if needed
	Extra map[string]interface{} `json:"-"`
}

func getRuneLiteDirs() (runeliteDir string, pluginsDir string, configFile string) {
	localAppData := os.Getenv("LOCALAPPDATA")
	userProfile := os.Getenv("USERPROFILE")

	runeliteDir = filepath.Join(localAppData, "RuneLite")
	configFile = filepath.Join(runeliteDir, "config.json")
	pluginsDir = filepath.Join(userProfile, ".runelite", "sideloaded-plugins")
	return
}

func IsRuneliteCracked() (bool, error) {
	runeliteDir, _, configFile := getRuneLiteDirs()

	if _, err := os.Stat(runeliteDir); os.IsNotExist(err) {
		return false, nil
	}

	crackerPath := filepath.Join(runeliteDir, loaderFileName)
	if _, err := os.Stat(crackerPath); os.IsNotExist(err) {
		return false, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return false, nil
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return false, nil
	}

	mainClass, _ := rawMap["mainClass"].(string)
	if mainClass != "cl.pajau.runelite.LauncherHijack" {
		return false, nil
	}

	classPathRaw, ok := rawMap["classPath"].([]interface{})
	if !ok {
		return false, nil
	}

	hasCracker := false
	hasRuneliteJar := false
	for _, item := range classPathRaw {
		str, ok := item.(string)
		if ok {
			if str == loaderFileName {
				hasCracker = true
			}
			if str == "RuneLite.jar" {
				hasRuneliteJar = true
			}
		}
	}

	return hasCracker && hasRuneliteJar, nil
}

func InstalarTodo(ctx context.Context) error{
	runeliteDir, pluginsDir, configFile := getRuneLiteDirs()

	if _, err := os.Stat(runeliteDir); os.IsNotExist(err) {
		return errors.New("Runelite AppData directory not found. Please install RuneLite first.")
	}

	//poner Loader
	//todo: check si es necesario descargar el loader
	destPath := filepath.Join(runeliteDir, loaderFileName)
	err := DescargarArchivo(destPath, "http://bucket.pajau.cl/katloader.jar")
	if err != nil {
		return err
	}


	//crack config
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return errors.New("Runelite config.json not found")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		rawMap = make(map[string]interface{})
	}

	rawMap["mainClass"] = "cl.pajau.runelite.LauncherHijack"

	var classPathList []string

	classPathList = append(classPathList, loaderFileName)
	classPathList = append(classPathList, "RuneLite.jar")

	rawMap["classPath"] = classPathList

	outData, err := json.MarshalIndent(rawMap, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configFile, outData, 0644); err != nil {
		return err
	}

	//poner plugins
	//todo: ver si es necesario descargar los plugins
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return err
	}

	version, downloadURL, err := ObtainLastVersionAndAsset()
	if err != nil || downloadURL == "" {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	_ = version
	jarFileName := filepath.Base(downloadURL)
	if !strings.HasSuffix(jarFileName, ".jar") {
		jarFileName = fmt.Sprintf("KatPlugins-%s.jar", version)
	}
	jarPath := filepath.Join(pluginsDir, jarFileName)

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "KatHub")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status: %d", resp.StatusCode)
	}

	out, err := os.Create(jarPath)
	if err != nil {
		return err
	}
	defer out.Close()

	pw := &progressWriter{
		ctx:        ctx,
		totalBytes: resp.ContentLength,
	}

	if _, err := io.Copy(out, io.TeeReader(resp.Body, pw)); err != nil {
		return err
	}

	return nil
}

func ExecuteCrackearRunelite(crack bool) (bool, error) {
	runeliteDir, _, configFile := getRuneLiteDirs()

	if _, err := os.Stat(runeliteDir); os.IsNotExist(err) {
		return false, errors.New("Runelite directory not found. Please install RuneLite first.")
	}

	if crack {
		destPath := filepath.Join(runeliteDir, loaderFileName)

		err := DescargarArchivo(destPath, "http://bucket.pajau.cl/katloader.jar")
		if err != nil {
			return false, err
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return false, errors.New("config.json not found")
		}

		data, err := os.ReadFile(configFile)
		if err != nil {
			return false, err
		}

		var rawMap map[string]interface{}
		if err := json.Unmarshal(data, &rawMap); err != nil {
			rawMap = make(map[string]interface{})
		}

		rawMap["mainClass"] = "cl.pajau.runelite.LauncherHijack"

		var classPathList []string

		classPathList = append(classPathList, loaderFileName)
		classPathList = append(classPathList, "RuneLite.jar")

		rawMap["classPath"] = classPathList

		outData, err := json.MarshalIndent(rawMap, "", "  ")
		if err != nil {
			return false, err
		}

		if err := os.WriteFile(configFile, outData, 0644); err != nil {
			return false, err
		}

		return true, nil
	} else {
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return false, errors.New("config.json not found")
		}

		data, err := os.ReadFile(configFile)
		if err != nil {
			return false, err
		}

		var rawMap map[string]interface{}
		if err := json.Unmarshal(data, &rawMap); err != nil {
			rawMap = make(map[string]interface{})
		}

		rawMap["mainClass"] = "net.runelite.launcher.Launcher"

		var classPathList []string
		if cp, ok := rawMap["classPath"].([]interface{}); ok {
			for _, item := range cp {
				if s, ok := item.(string); ok && s != loaderFileName {
					classPathList = append(classPathList, s)
				}
			}
		}

		hasRuneliteJar := false
		for _, s := range classPathList {
			if s == "RuneLite.jar" {
				hasRuneliteJar = true
			}
		}
		if !hasRuneliteJar {
			classPathList = append(classPathList, "RuneLite.jar")
		}

		rawMap["classPath"] = classPathList

		outData, err := json.MarshalIndent(rawMap, "", "  ")
		if err != nil {
			return false, err
		}

		if err := os.WriteFile(configFile, outData, 0644); err != nil {
			return false, err
		}

		return true, nil
	}
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

func ObtainInstalledVersion() string {
	_, pluginsDir, _ := getRuneLiteDirs()
	matches, err := filepath.Glob(filepath.Join(pluginsDir, "Kat*.jar"))
	if err != nil || len(matches) == 0 {
		return "-1"
	}

	if len(matches) > 1 {
		return "-69" // Multiple versions indicator
	}

	base := filepath.Base(matches[0])
	noExt := strings.TrimSuffix(base, filepath.Ext(base))
	parts := strings.Split(noExt, "-")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "-1"
}

func ExecuteUninstallJar() (bool, error) {
	_, pluginsDir, _ := getRuneLiteDirs()
	matches, err := filepath.Glob(filepath.Join(pluginsDir, "Kat*.jar"))
	if err != nil {
		return false, err
	}

	for _, jar := range matches {
		_ = os.Remove(jar)
	}
	return true, nil
}

type progressWriter struct {
	ctx        context.Context
	totalBytes int64
	downloaded int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	if pw.totalBytes > 0 && pw.ctx != nil {
		percentage := float64(pw.downloaded) / float64(pw.totalBytes) * 100
		runtime.EventsEmit(pw.ctx, "plugin-install-progress", percentage)
	}
	return n, nil
}

func ExecuteInstallOrUpdateJar(ctx context.Context) (bool, error) {
	_, pluginsDir, _ := getRuneLiteDirs()
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return false, err
	}

	version, downloadURL, err := ObtainLastVersionAndAsset()
	if err != nil || downloadURL == "" {
		return false, fmt.Errorf("failed to get latest release: %w", err)
	}

	_ = version
	jarFileName := filepath.Base(downloadURL)
	if !strings.HasSuffix(jarFileName, ".jar") {
		jarFileName = fmt.Sprintf("KatPlugins-%s.jar", version)
	}
	jarPath := filepath.Join(pluginsDir, jarFileName)

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "KatHub")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("download status: %d", resp.StatusCode)
	}

	out, err := os.Create(jarPath)
	if err != nil {
		return false, err
	}
	defer out.Close()

	pw := &progressWriter{
		ctx:        ctx,
		totalBytes: resp.ContentLength,
	}

	if _, err := io.Copy(out, io.TeeReader(resp.Body, pw)); err != nil {
		return false, err
	}

	return true, nil
}

func CalculateKatPluginsState() KatPluginsInfo {
	cracked, err := IsRuneliteCracked()
	if err != nil || !cracked {
		return KatPluginsInfo{
			State:             StateCrackear,
			StatusText:        "",
			StatusColor:       "#f08080",
			ButtonText:        "Install",
			ButtonColor:       "#e63946",
			MainButtonVisible: true,
			UninstallVisible:  false,
			CurrentVersion:    "",
			LatestVersion:     "",
		}
	}

	loaderState, _ := CheckLoader()
	if loaderState != LoaderUpToDate {
		return KatPluginsInfo{
			State:             StateUpdateLoader,
			StatusText:        "New Loader version available",
			StatusColor:       "#f08080",
			ButtonText:        "Update",
			ButtonColor:       "#e63946",
			MainButtonVisible: true,
			UninstallVisible:  true,
			CurrentVersion:    "",
			LatestVersion:     "",
		}
	}

	latestVer, _, _ := ObtainLastVersionAndAsset()
	currentVer := ObtainInstalledVersion()

	if currentVer == "-69" {
		return KatPluginsInfo{
			State:             StateMultipleVersions,
			StatusText:        "Multiple versions detected!",
			StatusColor:       "#ffd700", // Gold/Yellow
			ButtonText:        "Fix",
			ButtonColor:       "#e63946",
			MainButtonVisible: true,
			UninstallVisible:  true,
			CurrentVersion:    currentVer,
			LatestVersion:     latestVer,
		}
	}

	if currentVer == "-1" {
		return KatPluginsInfo{
			State:             StateInstall,
			StatusText:        "Plugin not detected",
			StatusColor:       "#f08080",
			ButtonText:        "Install",
			ButtonColor:       "#e63946",
			MainButtonVisible: true,
			UninstallVisible:  true,
			CurrentVersion:    "",
			LatestVersion:     latestVer,
		}
	}

	if latestVer != "-1" && currentVer < latestVer {
		return KatPluginsInfo{
			State:             StateUpdatePlugins,
			StatusText:        fmt.Sprintf("New version available: v%s", latestVer),
			StatusColor:       "#f08080",
			ButtonText:        "Update",
			ButtonColor:       "#e63946",
			MainButtonVisible: true,
			UninstallVisible:  true,
			CurrentVersion:    currentVer,
			LatestVersion:     latestVer,
		}
	}

	return KatPluginsInfo{
		State:             StateUpToDate,
		StatusText:        fmt.Sprintf("Running latest version v%s ✔", currentVer),
		StatusColor:       "#90ee90",
		ButtonText:        "Uninstall",
		ButtonColor:       "#444444",
		MainButtonVisible: true,
		UninstallVisible:  true,
		CurrentVersion:    currentVer,
		LatestVersion:     latestVer,
	}
}
