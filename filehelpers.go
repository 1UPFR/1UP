package main

import (
	"encoding/base64"
	"fmt"
	"os"
)

// GetFileSize retourne la taille d'un fichier
func (a *App) GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("erreur stat fichier: %w", err)
	}
	return info.Size(), nil
}

// ReadFileChunk lit un chunk de fichier et retourne en base64
func (a *App) ReadFileChunk(filePath string, offset int64, size int) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("erreur ouverture fichier: %w", err)
	}
	defer f.Close()

	buf := make([]byte, size)
	n, err := f.ReadAt(buf, offset)
	if err != nil && n == 0 {
		return "", fmt.Errorf("erreur lecture: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf[:n]), nil
}
