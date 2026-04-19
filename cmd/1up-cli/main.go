package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/1UPFR/1UP/internal/api"
	"github.com/1UPFR/1UP/internal/binutil"
	"github.com/1UPFR/1UP/internal/config"
	"github.com/1UPFR/1UP/internal/nyuu"
	"github.com/1UPFR/1UP/internal/parpar"
)

//go:embed binaries
var embeddedBinaries embed.FS

var AppVersion = "dev"
var apiBaseURL = ""

func main() {
	binutil.Init(embeddedBinaries)
	versionFlag := flag.Bool("version", false, "Afficher la version")
	configFlag := flag.Bool("config", false, "Afficher le chemin de la config")
	noAPI := flag.Bool("no-api", false, "Desactiver l'upload API")
	flag.Parse()

	if *versionFlag {
		fmt.Println("1UP CLI", AppVersion)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur config: %v\n", err)
		os.Exit(1)
	}
	if apiBaseURL != "" {
		api.BaseURL = apiBaseURL
	}

	if *configFlag {
		home, _ := os.UserHomeDir()
		fmt.Println(filepath.Join(home, ".config", "1up", "config.json"))
		os.Exit(0)
	}

	// Verifier la config et mode interactif si incomplet
	if cfg.Nyuu.Host == "" {
		fmt.Println("1UP CLI", AppVersion)
		fmt.Println()
		fmt.Println("Configuration manquante. Repondez aux questions suivantes :")
		fmt.Println()
		setupInteractive(cfg)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("1UP CLI", AppVersion)
		fmt.Println()
		fmt.Println("Usage: 1up-cli [options] <fichier1> [fichier2] ...")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	for _, inputPath := range args {
		if err := processFile(cfg, inputPath, *noAPI); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur %s: %v\n", inputPath, err)
		}
	}
}

func processFile(cfg *config.Config, inputPath string, noAPI bool) error {
	ext := filepath.Ext(inputPath)
	releaseName := strings.TrimSuffix(filepath.Base(inputPath), ext)

	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(inputPath)
	}
	if !filepath.IsAbs(outputDir) {
		outputDir, _ = filepath.Abs(outputDir)
	}
	os.MkdirAll(outputDir, 0755)

	// 1. ParPar
	fmt.Printf("[1/3] Generation par2 : %s\n", releaseName)
	err := parpar.Run(&cfg.ParPar, inputPath, func(p parpar.Progress) {
		if p.Done {
			if p.Error != "" {
				fmt.Printf("\r  ParPar ERREUR: %s\n", p.Error)
			} else {
				fmt.Printf("\r  ParPar 100%%         \n")
			}
		} else {
			fmt.Printf("\r  ParPar %.1f%%  ", p.Percent)
		}
	})
	if err != nil {
		return fmt.Errorf("par2: %w", err)
	}

	// 2. Collecter les fichiers
	par2Pattern := filepath.Join(filepath.Dir(inputPath), releaseName+".*.par2")
	par2Files, _ := filepath.Glob(par2Pattern)
	mainPar2 := filepath.Join(filepath.Dir(inputPath), releaseName+".par2")

	allFiles := []string{inputPath}
	if _, err := os.Stat(mainPar2); err == nil {
		allFiles = append(allFiles, mainPar2)
	}
	allFiles = append(allFiles, par2Files...)

	// 3. Nyuu
	fmt.Printf("[2/3] Post Usenet : %d fichiers\n", len(allFiles))
	nzbPath := filepath.Join(outputDir, releaseName+".nzb")
	result, err := nyuu.Run(&cfg.Nyuu, allFiles, nzbPath, releaseName, func(p nyuu.Progress) {
		if p.Done {
			if p.Error != "" {
				fmt.Printf("\r  Nyuu ERREUR: %s\n", p.Error)
			} else {
				fmt.Printf("\r  Nyuu 100%%           \n")
			}
		} else {
			info := fmt.Sprintf("%.1f%%", p.Percent)
			if p.Articles != "" {
				info += " " + p.Articles
			}
			if p.Speed != "" {
				info += " " + p.Speed
			}
			if p.ETA != "" {
				info += " ETA " + p.ETA
			}
			fmt.Printf("\r  Nyuu %s  ", info)
		}
	})
	if err != nil {
		return fmt.Errorf("nyuu: %w", err)
	}

	fmt.Printf("  NZB: %s\n", result.NZBPath)

	isISO := strings.EqualFold(ext, ".iso")

	// 4. MediaInfo JSON (si mediainfo est installe et pas un ISO)
	jsonPath := filepath.Join(outputDir, releaseName+".json")
	if !isISO {
		miPath := findMediaInfo()
		if miPath != "" {
			fmt.Println("[3/4] Generation MediaInfo JSON...")
			out, err := exec.Command(miPath, "--Output=JSON", "--Full", inputPath).Output()
			if err == nil {
				os.WriteFile(jsonPath, out, 0644)
				fmt.Println("  MediaInfo OK")
			} else {
				fmt.Fprintf(os.Stderr, "  MediaInfo erreur: %v\n", err)
			}
		} else {
			fmt.Println("[3/4] mediainfo non disponible, skip")
		}
	}

	// 5. Upload API (pas pour les ISO)
	if !isISO && !noAPI && cfg.API.Enabled && cfg.API.APIKey != "" {
		fmt.Println("[4/4] Upload API...")

		if _, err := os.Stat(jsonPath); err == nil {
			uploadResult, err := api.Upload(&cfg.API, releaseName, result.NZBPath, jsonPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  API erreur: %v\n", err)
			} else if uploadResult.Success {
				fmt.Println("  API OK")
			} else {
				fmt.Fprintf(os.Stderr, "  API erreur: %s\n", uploadResult.Error)
			}
		} else {
			fmt.Println("  API: pas de JSON mediainfo, skip")
		}
	} else if isISO {
		fmt.Println("[3/3] ISO: pas d'envoi API")
	} else {
		fmt.Println("[4/4] API desactivee, skip")
	}

	// Nettoyage par2
	if _, err := os.Stat(mainPar2); err == nil {
		os.Remove(mainPar2)
	}
	for _, f := range par2Files {
		os.Remove(f)
	}

	fmt.Printf("Termine: %s\n\n", releaseName)
	return nil
}

func setupInteractive(cfg *config.Config) {
	reader := bufio.NewReader(os.Stdin)

	ask := func(label string, current string) string {
		if current != "" {
			fmt.Printf("  %s [%s]: ", label, current)
		} else {
			fmt.Printf("  %s: ", label)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return current
		}
		return line
	}

	askInt := func(label string, current int) int {
		s := ask(label, strconv.Itoa(current))
		v, err := strconv.Atoi(s)
		if err != nil {
			return current
		}
		return v
	}

	askBool := func(label string, current bool) bool {
		def := "n"
		if current {
			def = "o"
		}
		fmt.Printf("  %s (o/n) [%s]: ", label, def)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" {
			return current
		}
		return line == "o" || line == "oui" || line == "y" || line == "yes"
	}

	fmt.Println("--- Serveur Usenet ---")
	cfg.Nyuu.Host = ask("Hote", cfg.Nyuu.Host)
	cfg.Nyuu.Port = askInt("Port", cfg.Nyuu.Port)
	cfg.Nyuu.SSL = askBool("SSL", cfg.Nyuu.SSL)
	cfg.Nyuu.User = ask("Utilisateur", cfg.Nyuu.User)
	cfg.Nyuu.Password = ask("Mot de passe", cfg.Nyuu.Password)
	cfg.Nyuu.Connections = askInt("Connexions", cfg.Nyuu.Connections)
	cfg.Nyuu.Group = ask("Groupe", cfg.Nyuu.Group)

	fmt.Println()
	fmt.Println("--- ParPar ---")
	cfg.ParPar.SliceSize = ask("Taille slice", cfg.ParPar.SliceSize)
	cfg.ParPar.Memory = ask("Memoire max", cfg.ParPar.Memory)
	cfg.ParPar.Threads = askInt("Threads", cfg.ParPar.Threads)
	cfg.ParPar.Redundancy = ask("Redondance", cfg.ParPar.Redundancy)

	fmt.Println()
	fmt.Println("--- API ---")
	cfg.API.Enabled = askBool("Activer l'API", cfg.API.Enabled)
	if cfg.API.Enabled {
		cfg.API.URL = ask("URL API", cfg.API.URL)
		cfg.API.APIKey = ask("Cle API", cfg.API.APIKey)
	}

	fmt.Println()
	cfg.OutputDir = ask("Dossier de sortie (vide = meme que source)", cfg.OutputDir)

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur sauvegarde config: %v\n", err)
	} else {
		fmt.Println()
		fmt.Println("Configuration sauvegardee.")
	}
	fmt.Println()
}

func findMediaInfo() string {
	// Essayer le binaire embarque
	if path, err := binutil.ExtractBinary("mediainfo"); err == nil {
		return path
	}
	// Fallback PATH systeme
	if path, err := exec.LookPath("mediainfo"); err == nil {
		return path
	}
	return ""
}
