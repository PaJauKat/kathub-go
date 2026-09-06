package katplugins

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	loaderDownloadURL  = "https://bucket.pajau.cl/katloader.jar"
	managerDownloadURL = "https://bucket.pajau.cl/katmanager.jar"
	pluginsDownloadURL = "https://bucket.pajau.cl/katplugins.jar"
)

type LoaderState int

const (
	LoaderFileNotFound LoaderState = iota
	CantReadVersion
	CantReadOnlineVersion
	LoaderNeedsUpdate
	LoaderUpToDate
	OnlineVersionNotSpecified
	CantParseOnlineVersion
)

type LoaderResult struct {
	State   LoaderState
	Version int
}

func CheckLoader() (LoaderState, error) {
	runeliteDir, _, _ := getRuneLiteDirs()
	loaderPath := filepath.Join(runeliteDir, loaderFileName)
	if _, err := os.Stat(loaderPath); os.IsNotExist(err) {
		return LoaderFileNotFound, nil
	}

	verByte, err := os.ReadFile(loaderPath + ":kversion")
	if err != nil {
		return CantReadVersion, err
	}

	localLoaderVersion, err := strconv.Atoi(string(verByte))
	if err != nil {
		return CantReadVersion, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(loaderDownloadURL)
	if err != nil {
		return CantReadOnlineVersion, err
	}
	defer resp.Body.Close()

	// R2 serves upload metadata as x-amz-meta-version; if the CDN is configured to
	// rewrite it, it may arrive as x-version. Accept both.
	onlineLoaderVersionString := resp.Header.Get("x-version")
	if onlineLoaderVersionString == "" {
		onlineLoaderVersionString = resp.Header.Get("x-amz-meta-version")
	}
	if onlineLoaderVersionString == "" {
		return OnlineVersionNotSpecified, nil
	}

	onlineLoaderVersion, err := strconv.Atoi(onlineLoaderVersionString)
	if err != nil {
		return CantParseOnlineVersion, err
	}

	if localLoaderVersion < onlineLoaderVersion {
		return LoaderNeedsUpdate, nil
	}

	return LoaderUpToDate, nil
}

type ManagerState int

const (
	ManagerFileNotFound ManagerState = iota
	ManagerNoVersionInfo
	ManagerNeedsUpdate
	ManagerUpToDate
)

// CheckManager compares the local Kat Manager version stream (KatManager.jar:kversion)
// against the online version served by the manager download URL. If no local version
// stream exists (e.g. the jar was installed manually), it is treated as up to date to
// avoid forcing updates on installations the tool did not create.
func CheckManager() (ManagerState, error) {
	_, managerDir, _ := getKatPluginDirs()
	managerPath := filepath.Join(managerDir, managerFileName)
	if _, err := os.Stat(managerPath); os.IsNotExist(err) {
		return ManagerFileNotFound, nil
	}

	verByte, err := os.ReadFile(managerPath + ":kversion")
	if err != nil {
		return ManagerUpToDate, nil
	}

	localVersion, err := strconv.Atoi(string(verByte))
	if err != nil {
		return ManagerUpToDate, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(managerDownloadURL)
	if err != nil {
		return ManagerUpToDate, nil
	}
	defer resp.Body.Close()

	onlineVersionString := resp.Header.Get("x-version")
	if onlineVersionString == "" {
		onlineVersionString = resp.Header.Get("x-amz-meta-version")
	}
	if onlineVersionString == "" {
		return ManagerUpToDate, nil
	}

	onlineVersion, err := strconv.Atoi(onlineVersionString)
	if err != nil {
		return ManagerUpToDate, nil
	}

	if localVersion < onlineVersion {
		return ManagerNeedsUpdate, nil
	}

	return ManagerUpToDate, nil
}
