package binutil

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	binaries embed.FS
	once     sync.Once
	cacheDir string
)

// Init enregistre le filesystem embarqué contenant les binaires
func Init(embedded embed.FS) {
	binaries = embedded
}

func ensureCacheDir() (string, error) {
	var err error
	once.Do(func() {
		home, e := os.UserHomeDir()
		if e != nil {
			err = e
			return
		}
		cacheDir = filepath.Join(home, ".cache", "1up", "bin")
		err = os.MkdirAll(cacheDir, 0755)
	})
	return cacheDir, err
}

// ExtractBinary extrait un binaire embarqué vers le cache et retourne son chemin
func ExtractBinary(name string) (string, error) {
	dir, err := ensureCacheDir()
	if err != nil {
		return "", fmt.Errorf("erreur cache dir: %w", err)
	}

	if runtime.GOOS == "windows" {
		name = name + ".exe"
	}

	destPath := filepath.Join(dir, name)

	// Si deja extrait, on retourne le chemin
	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}

	// Chercher dans binaries/<os>-<arch>/<name> (desktop) puis binaries/<name> (cli/web)
	// Utiliser / et non filepath.Join car embed.FS utilise toujours des /
	platform := runtime.GOOS + "-" + runtime.GOARCH
	paths := []string{
		"binaries/" + platform + "/" + name,
		"binaries/" + name,
	}

	var data []byte
	var readErr error
	for _, p := range paths {
		data, readErr = fs.ReadFile(binaries, p)
		if readErr == nil {
			break
		}
	}
	if readErr != nil {
		return "", fmt.Errorf("binaire %s non embarque: %w", name, readErr)
	}

	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return "", fmt.Errorf("erreur extraction %s: %w", name, err)
	}

	return destPath, nil
}
