package katplugins

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/xattr"
)

func DescargarArchivo(filePath string, url string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error en la descarga: estado HTTP %s", resp.Status)
	}

	archiveVersion := resp.Header.Get("x-version")

	_, err = io.Copy(f, resp.Body)
	vErr := xattr.Set(filePath, "kversion", []byte(archiveVersion))
	if vErr != nil {
		fmt.Printf("No se pudo settear la version para %s\n", filePath)
	}
	fmt.Printf("Version (%v) setteada para: %v", archiveVersion, filePath)
	return err

}