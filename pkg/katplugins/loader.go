package katplugins

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/xattr"
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
	_, err := os.Stat(loaderPath)
	if os.IsNotExist(err) {
		return LoaderFileNotFound, nil
	}
	verByte, err := xattr.Get(loaderPath, "kversion")
	if err != nil {
		return CantReadVersion, err
	}

	localLoaderVersion, err := strconv.Atoi(string(verByte))
	if err != nil {
		return CantReadOnlineVersion, err
	}

	resp, err := http.Head("https://bucket.pajau.cl/katloader.jar")
	if err != nil {
		return CantReadOnlineVersion, err
	}

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
	} else {
		return LoaderUpToDate, nil
	}

}

