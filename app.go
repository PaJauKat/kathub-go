package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"kathub/pkg/katplugins"
	"kathub/pkg/lolconfig"
	"kathub/pkg/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetKatPluginsState returns current state and UI metadata of KatPlugins
func (a *App) GetKatPluginsState() katplugins.KatPluginsInfo {
	return katplugins.CalculateKatPluginsState()
}

// HandleKatPluginsMainAction handles main action button click depending on current state
func (a *App) HandleKatPluginsMainAction() (bool, error) {
	stateInfo := katplugins.CalculateKatPluginsState()

	switch stateInfo.State {
	case katplugins.StateCrackear:
		ok, err := katplugins.ExecuteCrackearRunelite(true)
		if !ok || err != nil {
			return false, fmt.Errorf("failed to crack runelite: %v", err)
		}
		// Continue directly to install
		return katplugins.ExecuteInstallOrUpdateJar(a.ctx)

	case katplugins.StateUpdateLoader:
		err := katplugins.InstalarTodo(a.ctx)
		if err != nil{
			return false, err
		}
		return true, nil

	case katplugins.StateInstall, katplugins.StateUpdatePlugins:
		return katplugins.ExecuteInstallOrUpdateJar(a.ctx)

	case katplugins.StateUpToDate:
		_, _ = katplugins.ExecuteUninstallJar()
		return katplugins.ExecuteCrackearRunelite(false)

	case katplugins.StateMultipleVersions:
		_, _ = katplugins.ExecuteUninstallJar()
		return katplugins.ExecuteInstallOrUpdateJar(a.ctx)

	default:
		return false, errors.New("unknown action state")
	}
}

// UninstallKatPlugins removes jars and un-hijacks runelite config
func (a *App) UninstallKatPlugins() (bool, error) {
	ok, err := katplugins.ExecuteUninstallJar()
	if !ok || err != nil {
		return false, fmt.Errorf("error uninstalling jar: %v", err)
	}

	ok, err = katplugins.ExecuteCrackearRunelite(false)
	if !ok || err != nil {
		return false, fmt.Errorf("error restoring config: %v", err)
	}

	return true, nil
}

// GetLolConfigState returns League of Legends config locker state
func (a *App) GetLolConfigState() lolconfig.LolConfigInfo {
	return lolconfig.CheckLolConfigState()
}

// ToggleLolConfig locks or unlocks the LoL PersistedSettings.json
func (a *App) ToggleLolConfig(lock bool) (bool, error) {
	return lolconfig.ModifyLolConfig(lock)
}

// InstallAccountManager downloads and extracts KatAccountManager
func (a *App) InstallAccountManager() (bool, error) {
	userProfile := os.Getenv("USERPROFILE")
	runeliteSettings := filepath.Join(userProfile, ".runelite", "settings.json")

	// Update settings.json if exists
	if data, err := os.ReadFile(runeliteSettings); err == nil {
		var rawMap map[string]interface{}
		if err := json.Unmarshal(data, &rawMap); err == nil {
			var argsList []interface{}
			if existing, ok := rawMap["clientArguments"].([]interface{}); ok {
				argsList = existing
			}

			hasInsecure := false
			for _, arg := range argsList {
				if s, ok := arg.(string); ok && s == "--insecure-write-credentials" {
					hasInsecure = true
					break
				}
			}

			if !hasInsecure {
				argsList = append(argsList, "--insecure-write-credentials")
				rawMap["clientArguments"] = argsList
				if outData, err := json.MarshalIndent(rawMap, "", "  "); err == nil {
					_ = os.WriteFile(runeliteSettings, outData, 0644)
				}
			}
		}
	}

	appData := os.Getenv("APPDATA")
	appsDir := filepath.Join(appData, "KatHub", "Apps")
	managerDir := filepath.Join(appsDir, "KatAccountManager")
	tempZip := filepath.Join(os.TempDir(), "KatAccounts.zip")

	if err := os.MkdirAll(appsDir, 0755); err != nil {
		return false, err
	}

	urlDescarga := "https://github.com/PaJauKat/KatAccountManager/releases/download/v1.0/KatAccounts.zip"
	req, err := http.NewRequest("GET", urlDescarga, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "KatHub")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to download Account Manager: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(tempZip)
	if err != nil {
		return false, err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return false, err
	}
	out.Close()

	_ = os.RemoveAll(managerDir)

	// Extract
	r, err := zip.OpenReader(tempZip)
	if err != nil {
		return false, err
	}
	defer r.Close()

	for _, f := range r.File {
		destPath := filepath.Join(managerDir, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return false, err
		}

		rc, err := f.Open()
		if err != nil {
			return false, err
		}

		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return false, err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return false, err
		}
	}

	_ = os.Remove(tempZip)
	return true, nil
}

// OpenAccountManager opens explorer or Account Manager directory
func (a *App) OpenAccountManager() {
	_ = exec.Command("explorer.exe").Start()
}

// CheckForUpdates checks PaJauKat/KatHub GitHub releases
func (a *App) CheckForUpdates() (updater.UpdateInfo, error) {
	return updater.CheckForAppUpdates()
}

// DownloadAndApplyUpdate triggers app update download
func (a *App) DownloadAndApplyUpdate(downloadURL string) error {
	return updater.DownloadAndApplyUpdate(downloadURL)
}

// ShowDialog helper for native dialogs if needed
func (a *App) ShowDialog(dialogType string, title string, message string) (string, error) {
	var res string
	var err error
	if dialogType == "confirm" {
		res, err = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:          runtime.QuestionDialog,
			Title:         title,
			Message:       message,
			Buttons:       []string{"Yes", "No"},
			DefaultButton: "Yes",
			CancelButton:  "No",
		})
	} else if dialogType == "error" {
		res, err = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   title,
			Message: message,
		})
	} else {
		res, err = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   title,
			Message: message,
		})
	}
	return res, err
}
