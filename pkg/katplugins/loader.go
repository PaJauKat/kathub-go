package katplugins

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	loaderDownloadURL = "https://bucket.pajau.cl/katloader.jar"
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

	onlineLoaderVersionString := resp.Header.Get("x-version")
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
