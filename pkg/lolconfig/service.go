package lolconfig

import (
	"os"
	"path/filepath"
)

type LolConfigState string

const (
	StateUnknown  LolConfigState = "Unknown"
	StateLocked   LolConfigState = "Locked"
	StateUnlocked LolConfigState = "Unlocked"
)

type LolConfigInfo struct {
	State       LolConfigState `json:"state"`
	StatusText  string         `json:"statusText"`
	StatusColor string         `json:"statusColor"`
	ButtonText  string         `json:"buttonText"`
	CanToggle   bool           `json:"canToggle"`
}

func getLolConfigFilePath() string {
	return filepath.Join("C:\\", "Riot Games", "League of Legends", "Config", "PersistedSettings.json")
}

func CheckLolConfigState() LolConfigInfo {
	filePath := getLolConfigFilePath()

	info, err := os.Stat(filePath)
	if err != nil {
		return LolConfigInfo{
			State:       StateUnknown,
			StatusText:  "Config not found",
			StatusColor: "#888888",
			ButtonText:  "Error",
			CanToggle:   false,
		}
	}

	// In Windows, FileMode 0200 is write permission (0400 is read only in Go terms)
	// If (mode & 0222) == 0, it's read-only
	isReadOnly := (info.Mode().Perm() & 0222) == 0

	if isReadOnly {
		return LolConfigInfo{
			State:       StateLocked,
			StatusText:  "Locked ✔",
			StatusColor: "#90ee90", // LightGreen
			ButtonText:  "Unlock",
			CanToggle:   true,
		}
	}

	return LolConfigInfo{
		State:       StateUnlocked,
		StatusText:  "Unlocked ❌",
		StatusColor: "#f08080", // LightCoral
		ButtonText:  "Lock",
		CanToggle:   true,
	}
}

func ModifyLolConfig(lock bool) (bool, error) {
	filePath := getLolConfigFilePath()
	if _, err := os.Stat(filePath); err != nil {
		return false, err
	}

	var mode os.FileMode
	if lock {
		mode = 0444 // Read-only
	} else {
		mode = 0666 // Read-write
	}

	err := os.Chmod(filePath, mode)
	if err != nil {
		return false, err
	}

	return true, nil
}
