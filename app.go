package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/1UPFR/1UP/internal/api"
	"github.com/1UPFR/1UP/internal/config"
	"github.com/1UPFR/1UP/internal/history"
	"github.com/1UPFR/1UP/internal/nyuu"
	"github.com/1UPFR/1UP/internal/parpar"
)

type App struct {
	ctx     context.Context
	cfg     *config.Config
	history *history.DB
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur chargement config: %v\n", err)
		cfg = config.DefaultConfig()
	}
	a.cfg = cfg

	h, err := history.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur ouverture historique: %v\n", err)
	}
	a.history = h
}

func (a *App) shutdown(ctx context.Context) {
	if a.history != nil {
		a.history.Close()
	}
}

// GetConfig retourne la configuration actuelle
func (a *App) GetConfig() *config.Config {
	return a.cfg
}

// SaveConfig sauvegarde la configuration
func (a *App) SaveConfig(cfg config.Config) error {
	a.cfg = &cfg
	return a.cfg.Save()
}

// SelectFile ouvre un selecteur de fichier natif
func (a *App) SelectFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Selectionner un fichier",
		Filters: []runtime.FileFilter{
			{DisplayName: "Fichiers video", Pattern: "*.mkv;*.mp4;*.iso"},
		},
	})
}

// SelectDirectory ouvre un selecteur de dossier natif
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Selectionner un dossier",
	})
}

// SelectFileWithFilter ouvre un selecteur avec filtre
func (a *App) SelectFileWithFilter(title string, pattern string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "Fichiers", Pattern: pattern},
		},
	})
}

// ManualUpload envoie un NZB + mediainfo/bdinfo existants sur l'API
func (a *App) ManualUpload(releaseName string, nzbPath string, mediainfoPath string, bdinfoFullPath string, bdinfoMiniPath string) (*api.UploadResult, error) {
	if !a.cfg.API.Enabled || a.cfg.API.APIKey == "" {
		return nil, fmt.Errorf("API desactivee ou cle manquante")
	}

	// Enregistrer dans l'historique
	entry := &history.Entry{
		ReleaseName: releaseName,
		FilePath:    nzbPath,
		Status:      "processing",
	}
	var historyID int64
	if a.history != nil {
		historyID, _ = a.history.Add(entry)
	}

	result, err := api.UploadManual(&a.cfg.API, api.ManualUploadParams{
		ReleaseName:       releaseName,
		NZBPath:           nzbPath,
		MediaInfoJSONPath: mediainfoPath,
		BDInfoFullPath:    bdinfoFullPath,
		BDInfoMiniPath:    bdinfoMiniPath,
	})

	if err != nil {
		if a.history != nil {
			a.history.Update(historyID, "error", nzbPath, "", err.Error())
		}
		return nil, err
	}

	apiResult := ""
	if result != nil {
		j, _ := json.Marshal(result)
		apiResult = string(j)
	}

	if a.history != nil {
		status := "success"
		errMsg := ""
		if result != nil && !result.Success {
			status = "error"
			errMsg = result.Error
		}
		a.history.Update(historyID, status, nzbPath, apiResult, errMsg)
	}

	return result, nil
}

// CheckRelease verifie si une release existe deja sur l'API
func (a *App) CheckRelease(releaseName string) (*api.CheckResult, error) {
	if !a.cfg.API.Enabled || a.cfg.API.APIKey == "" {
		return &api.CheckResult{Exists: false, Explain: "API desactivee"}, nil
	}
	return api.CheckRelease(&a.cfg.API, releaseName)
}

// GeneratePar2 lance ParPar sur un fichier
func (a *App) GeneratePar2(inputPath string) error {
	return parpar.Run(&a.cfg.ParPar, inputPath, func(p parpar.Progress) {
		runtime.EventsEmit(a.ctx, "parpar:progress", p)
	})
}

// PostToUsenet lance Nyuu pour poster les fichiers
func (a *App) PostToUsenet(inputFiles []string, outputDir string, releaseName string) (*nyuu.Result, error) {
	os.MkdirAll(outputDir, 0755)
	nzbPath := filepath.Join(outputDir, releaseName+".nzb")

	return nyuu.Run(&a.cfg.Nyuu, inputFiles, nzbPath, releaseName, func(p nyuu.Progress) {
		runtime.EventsEmit(a.ctx, "nyuu:progress", p)
	})
}

// resolveOutputDir retourne le dossier de sortie absolu
func (a *App) resolveOutputDir(inputPath string) string {
	dir := a.cfg.OutputDir
	if dir == "" {
		return filepath.Dir(inputPath)
	}
	if !filepath.IsAbs(dir) {
		abs, err := filepath.Abs(dir)
		if err == nil {
			return abs
		}
	}
	return dir
}

// UploadToAPI envoie le NZB et le MediaInfo vers l'API
func (a *App) UploadToAPI(releaseName string, nzbPath string, mediainfoJSON string) (*api.UploadResult, error) {
	outputDir := a.resolveOutputDir(nzbPath)
	jsonPath := filepath.Join(outputDir, releaseName+".json")

	if err := os.WriteFile(jsonPath, []byte(mediainfoJSON), 0644); err != nil {
		return nil, fmt.Errorf("erreur ecriture mediainfo json: %w", err)
	}

	return api.Upload(&a.cfg.API, releaseName, nzbPath, jsonPath)
}

// SaveMediaInfoJSON sauvegarde le JSON mediainfo sur disque pour l'upload API
func (a *App) SaveMediaInfoJSON(inputPath string, mediainfoJSON string) (string, error) {
	ext := filepath.Ext(inputPath)
	releaseName := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outputDir := a.resolveOutputDir(inputPath)
	os.MkdirAll(outputDir, 0755)
	jsonPath := filepath.Join(outputDir, releaseName+".json")
	if err := os.WriteFile(jsonPath, []byte(mediainfoJSON), 0644); err != nil {
		return "", fmt.Errorf("erreur ecriture mediainfo json: %w", err)
	}
	return jsonPath, nil
}

// ProcessFile orchestre le workflow complet : par2 -> post -> mediainfo -> upload
// queueID est l'identifiant frontend de l'item dans la queue (passe aux events)
func (a *App) ProcessFile(inputPath string, queueID string) error {
	ext := filepath.Ext(inputPath)
	releaseName := strings.TrimSuffix(filepath.Base(inputPath), ext)
	outputDir := a.resolveOutputDir(inputPath)
	os.MkdirAll(outputDir, 0755)

	emit := func(event string, data interface{}) {
		runtime.EventsEmit(a.ctx, event, map[string]interface{}{
			"queueID": queueID,
			"data":    data,
		})
	}

	// Enregistrer dans l'historique
	entry := &history.Entry{
		ReleaseName: releaseName,
		FilePath:    inputPath,
		Status:      "processing",
	}
	var historyID int64
	if a.history != nil {
		historyID, _ = a.history.Add(entry)
	}

	emit("status", "Generation par2...")

	// 1. ParPar
	err := parpar.Run(&a.cfg.ParPar, inputPath, func(p parpar.Progress) {
		emit("parpar:progress", p)
	})
	if err != nil {
		if a.history != nil {
			a.history.Update(historyID, "error", "", "", err.Error())
		}
		return fmt.Errorf("erreur par2: %w", err)
	}

	// 2. Collecter tous les fichiers (original + par2)
	par2Pattern := filepath.Join(filepath.Dir(inputPath), releaseName+".*.par2")
	par2Files, _ := filepath.Glob(par2Pattern)
	mainPar2 := filepath.Join(filepath.Dir(inputPath), releaseName+".par2")

	allFiles := []string{inputPath}
	if _, err := os.Stat(mainPar2); err == nil {
		allFiles = append(allFiles, mainPar2)
	}
	allFiles = append(allFiles, par2Files...)

	emit("status", "Post Usenet...")

	// 3. Nyuu
	nzbPath := filepath.Join(outputDir, releaseName+".nzb")
	result, err := nyuu.Run(&a.cfg.Nyuu, allFiles, nzbPath, releaseName, func(p nyuu.Progress) {
		emit("nyuu:progress", p)
	})
	if err != nil {
		if a.history != nil {
			a.history.Update(historyID, "error", "", "", err.Error())
		}
		return fmt.Errorf("erreur post usenet: %w", err)
	}

	apiResultStr := ""
	isISO := strings.EqualFold(ext, ".iso")

	// 4. Upload API (pas pour les ISO)
	if !isISO && a.cfg.API.Enabled && a.cfg.API.APIKey != "" {
		emit("status", "Upload API...")

		jsonPath := filepath.Join(outputDir, releaseName+".json")
		if _, err := os.Stat(jsonPath); err == nil {
			uploadResult, err := api.Upload(&a.cfg.API, releaseName, result.NZBPath, jsonPath)
			if err != nil {
				if a.history != nil {
					a.history.Update(historyID, "error", result.NZBPath, "", err.Error())
				}
				return fmt.Errorf("erreur upload API: %w", err)
			}
			resultJSON, _ := json.Marshal(uploadResult)
			apiResultStr = string(resultJSON)
			emit("upload:result", apiResultStr)
		}
	}

	// 5. Nettoyage par2
	emit("status", "Nettoyage par2...")
	if _, err := os.Stat(mainPar2); err == nil {
		os.Remove(mainPar2)
	}
	for _, f := range par2Files {
		os.Remove(f)
	}

	// Succes
	if a.history != nil {
		a.history.Update(historyID, "success", result.NZBPath, apiResultStr, "")
	}

	emit("status", "Termine")
	return nil
}

// SetHistoryMediaInfo met a jour les infos media d'une entree historique
func (a *App) SetHistoryMediaInfo(historyID int64, resolution, videoCodec, audioCodec, hdrFormat string, fileSize int64, duration, audioLangs, subtitleLangs string) {
	if a.history == nil {
		return
	}
	a.history.DB().Exec(`
		UPDATE history SET resolution=?, video_codec=?, audio_codec=?, hdr_format=?,
			file_size=?, duration=?, audio_langs=?, subtitle_langs=?
		WHERE id=?`,
		resolution, videoCodec, audioCodec, hdrFormat,
		fileSize, duration, audioLangs, subtitleLangs, historyID)
}

// SetHistoryTMDB met a jour les infos TMDB d'une entree historique
func (a *App) SetHistoryTMDB(historyID int64, title, year, poster, mediaType string) {
	if a.history == nil {
		return
	}
	a.history.DB().Exec(`
		UPDATE history SET tmdb_title=?, tmdb_year=?, tmdb_poster=?, tmdb_type=? WHERE id=?`,
		title, year, poster, mediaType, historyID)
}

// HistoryList retourne la liste paginee de l'historique
func (a *App) HistoryList(params history.ListParams) (*history.ListResult, error) {
	if a.history == nil {
		return &history.ListResult{Entries: []history.Entry{}}, nil
	}
	return a.history.List(params)
}

// HistoryStats retourne les statistiques
func (a *App) HistoryStats() (map[string]interface{}, error) {
	if a.history == nil {
		return map[string]interface{}{}, nil
	}
	return a.history.Stats()
}

// HistoryDelete supprime une entree
func (a *App) HistoryDelete(id int64) error {
	if a.history == nil {
		return nil
	}
	return a.history.Delete(id)
}

// HistoryClear vide tout l'historique
func (a *App) HistoryClear() error {
	if a.history == nil {
		return nil
	}
	return a.history.Clear()
}
