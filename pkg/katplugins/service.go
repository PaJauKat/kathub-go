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
	gitHubRepoOwner   = "PaJauKat"
	gitHubRepoName    = "PaJau-plugins"
	loaderFileName    = "kat.jar"
	managerFileName   = "KatManager.jar"
	hijackMainClass   = "cl.pajau.runelite.LauncherHijack"
	standardMainClass = "net.runelite.launcher.Launcher"
	runeLiteJarName   = "RuneLite.jar"
)

type KatPluginsState string

const (
	StateUnknown          KatPluginsState = "Unknown"
	StateCrackear         KatPluginsState = "Crackear"
	StateInstall          KatPluginsState = "Install"
	StateUpdatePlugins    KatPluginsState = "Update"
	StateUpdateLoader     KatPluginsState = "UpdateLoader"
	StateUpdateManager    KatPluginsState = "UpdateManager"
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

type RuneLiteConfig struct {
	MainClass string   `json:"mainClass"`
	ClassPath []string `json:"classPath"`
	// Preserve other dynamic keys if needed
	Extra map[string]interface{} `json:"-"`
}

func getKatPluginDirs() (katDir string, managerDir string, pluginsDir string) {
	userProfile := os.Getenv("USERPROFILE")
	katDir = filepath.Join(userProfile, ".runelite", "kat")
	managerDir = filepath.Join(katDir, "manager")
	pluginsDir = filepath.Join(katDir, "plugins")
	return
}

func getRuneLiteDirs() (runeliteDir string, pluginsDir string, configFile string) {
	localAppData := os.Getenv("LOCALAPPDATA")

	runeliteDir = filepath.Join(localAppData, "RuneLite")
	configFile = filepath.Join(runeliteDir, "config.json")
	_, _, pluginsDir = getKatPluginDirs()
	return
}

// managerInstalled checks whether the Kat Manager jar is present in the manager
// directory. The loader (HijackedClientBackup) loads the manager from
// ~/.runelite/kat/manager/ to show the authentication panel and unlock the plugins.
func managerInstalled() bool {
	_, managerDir, _ := getKatPluginDirs()
	// The Gradle task (KatManagerJar) names the artifact KatManager-<version>.jar,
	// so accept any KatManager*.jar in the manager directory.
	matches, err := filepath.Glob(filepath.Join(managerDir, "KatManager*.jar"))
	return err == nil && len(matches) > 0
}

// EnsureManagerInstalled downloads the Kat Manager jar into ~/.runelite/kat/manager/
// when it is not present yet. The manager is what lets users authenticate with
// Discord from inside RuneLite.
func EnsureManagerInstalled() error {
	_, managerDir, _ := getKatPluginDirs()
	if managerInstalled() {
		return nil
	}
	
	if err := os.MkdirAll(managerDir, 0755); err != nil {
		return fmt.Errorf("failed to create manager directory: %w", err)
	}
	return DescargarArchivo(filepath.Join(managerDir, managerFileName), managerDownloadURL)
}

// EnsureManagerUpdated redownloads the Kat Manager jar when the installed version
// is older than the one served by the download URL.
func EnsureManagerUpdated() error {
	state, _ := CheckManager()
	if state != ManagerNeedsUpdate {
		return nil
	}
	
	_, managerDir, _ := getKatPluginDirs()
	return DescargarArchivo(filepath.Join(managerDir, managerFileName), managerDownloadURL)
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
	if mainClass != hijackMainClass {
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
			if str == runeLiteJarName {
				hasRuneliteJar = true
			}
		}
	}

	return hasCracker && hasRuneliteJar, nil
}

// DownloadLatestPlugins handles the download and progress tracking of the latest plugin jar.
func DownloadLatestPlugins(ctx context.Context, pluginsDir string) error {
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	version, downloadURL, err := ObtainLastVersionAndAsset()
	if err != nil || downloadURL == "" {
		return fmt.Errorf("failed to get latest release asset: %w", err)
	}

	// Remove old plugins before writing new one to prevent multiple versions
	if _, err := DeletePluginJars(); err != nil {
		return fmt.Errorf("failed to clean existing plugin jars: %w", err)
	}

	jarFileName := filepath.Base(downloadURL)
	if !strings.HasSuffix(strings.ToLower(jarFileName), ".jar") {
		jarFileName = fmt.Sprintf("KatPlugins-%s.jar", version)
	}
	jarPath := filepath.Join(pluginsDir, jarFileName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build download request: %w", err)
	}
	req.Header.Set("User-Agent", "KatHub")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download status error: %d", resp.StatusCode)
	}

	out, err := os.Create(jarPath)
	if err != nil {
		return fmt.Errorf("failed to create plugin file: %w", err)
	}
	defer out.Close()

	pw := &progressWriter{
		ctx:        ctx,
		totalBytes: resp.ContentLength,
	}

	if _, err := io.Copy(out, io.TeeReader(resp.Body, pw)); err != nil {
		return fmt.Errorf("failed while saving plugin jar: %w", err)
	}

	return nil
}

// EnsureLoaderConfigured ensures the loader jar is downloaded/updated and config.json is hijacked.
func EnsureLoaderConfigured(runeliteDir, configFile string) error {
	loaderState, err := CheckLoader()
	if loaderState != LoaderUpToDate {
		if IsRuneLiteRunning() {
			return errors.New("No se pudo reemplazar kat.jar. Debes cerrar RuneLite antes de continuar.")
		}

		destPath := filepath.Join(runeliteDir, loaderFileName)
		if err := DescargarArchivo(destPath, loaderDownloadURL); err != nil {
			if IsRuneLiteRunning() || isFileLockOrPermissionError(err) {
				return errors.New("No se pudo reemplazar kat.jar. Debes cerrar RuneLite antes de continuar.")
			}
			return fmt.Errorf("no se pudo reemplazar el archivo kat.jar. Por favor, asegúrate de cerrar RuneLite e inténtalo de nuevo: %w", err)
		}
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return errors.New("RuneLite config.json not found")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read RuneLite config: %w", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		rawMap = make(map[string]interface{})
	}

	rawMap["mainClass"] = hijackMainClass

	var classPathList []string
	if cp, ok := rawMap["classPath"].([]interface{}); ok {
		for _, item := range cp {
			if s, ok := item.(string); ok && s != loaderFileName && s != runeLiteJarName && s != "EthanVannInstaller.jar" {
				classPathList = append(classPathList, s)
			}
		}
	}

	// Ensure loader is first, RuneLite.jar is present
	classPathList = append([]string{loaderFileName, runeLiteJarName}, classPathList...)

	rawMap["classPath"] = classPathList

	outData, err := json.MarshalIndent(rawMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize RuneLite config: %w", err)
	}

	if err := os.WriteFile(configFile, outData, 0644); err != nil {
		return fmt.Errorf("failed to write RuneLite config: %w", err)
	}

	return nil
}

// InstalarTodo checks and installs or updates the loader and plugins as needed.
func InstalarTodo(ctx context.Context) error {
	runeliteDir, pluginsDir, configFile := getRuneLiteDirs()

	if _, err := os.Stat(runeliteDir); os.IsNotExist(err) {
		return errors.New("RuneLite AppData directory not found. Please install RuneLite first.")
	}

	// 1. Ensure loader and config hijack
	if err := EnsureLoaderConfigured(runeliteDir, configFile); err != nil {
		return err
	}

	// 1.5 Ensure the Kat Manager is installed (required by the loader to authenticate)
	if err := EnsureManagerInstalled(); err != nil {
		return err
	}
	if err := EnsureManagerUpdated(); err != nil {
		return err
	}

	// 2. Check plugin status and download only if needed
	installedVer := ObtainInstalledPluginsVersion()
	latestVer, _, err := ObtainLastVersionAndAsset()
	if err != nil {
		// If obtaining latest fails, attempt download anyway or return error
		return fmt.Errorf("failed to check latest plugin version: %w", err)
	}

	needDownload := false
	switch {
	case installedVer == "-1": // No plugins installed
		needDownload = true
	case installedVer == "-69": // Multiple versions installed
		needDownload = true
	case latestVer != "-1" && CompareVersions(latestVer, installedVer) > 0: // Newer version available
		needDownload = true
	}

	if needDownload {
		if err := DownloadLatestPlugins(ctx, pluginsDir); err != nil {
			return err
		}
	}

	return nil
}

func DesinstalarTodo() error {
	runeliteDir, _, configFile := getRuneLiteDirs()

	// Restore config.json
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return errors.New("config.json not found")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		rawMap = make(map[string]interface{})
	}

	rawMap["mainClass"] = standardMainClass

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
		if s == runeLiteJarName {
			hasRuneliteJar = true
			break
		}
	}
	if !hasRuneliteJar {
		classPathList = append(classPathList, runeLiteJarName)
	}

	rawMap["classPath"] = classPathList

	outData, err := json.MarshalIndent(rawMap, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configFile, outData, 0644); err != nil {
		return err
	}

	// Remove loader and legacy artifacts (ignore failures if files are in use)
	_ = os.Remove(filepath.Join(runeliteDir, loaderFileName))
	_ = os.Remove(filepath.Join(runeliteDir, loaderFileName+":kversion"))
	_ = os.Remove(filepath.Join(runeliteDir, "EthanVannInstaller.jar"))

	// Remove plugins
	if _, err := DeletePluginJars(); err != nil {
		return err
	}

	// Remove Kat Manager and auth token (~/.runelite/kat tree)
	katDir, managerDir, _ := getKatPluginDirs()
	_ = os.RemoveAll(managerDir)
	_ = os.Remove(filepath.Join(katDir, "token.dat"))
	_ = os.RemoveAll(katDir)

	return nil
}

func DeletePluginJars() (bool, error) {
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

	if !managerInstalled() {
		return KatPluginsInfo{
			State:             StateInstall,
			StatusText:        "Kat Manager not detected",
			StatusColor:       "#f08080",
			ButtonText:        "Install",
			ButtonColor:       "#e63946",
			MainButtonVisible: true,
			UninstallVisible:  true,
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

	managerState, _ := CheckManager()
	if managerState == ManagerNeedsUpdate {
		return KatPluginsInfo{
			State:             StateUpdateManager,
			StatusText:        "New Kat Manager version available",
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
	currentVer := ObtainInstalledPluginsVersion()

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

	if latestVer != "-1" && CompareVersions(latestVer, currentVer) > 0 {
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
