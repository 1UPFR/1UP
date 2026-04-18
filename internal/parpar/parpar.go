package parpar

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/1UPFR/1UP/internal/binutil"
	"github.com/1UPFR/1UP/internal/config"
)

type Progress struct {
	Percent float64 `json:"percent"`
	Speed   string  `json:"speed"`
	ETA     string  `json:"eta"`
	Done    bool    `json:"done"`
	Error   string  `json:"error,omitempty"`
}

var progressRegex = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)

func binaryPath() string {
	// Essayer le binaire embarqué
	if path, err := binutil.ExtractBinary("parpar"); err == nil {
		return path
	}
	// Fallback sur le PATH système
	if path, err := exec.LookPath("parpar"); err == nil {
		return path
	}
	return "parpar"
}

func Run(cfg *config.ParParConfig, inputPath string, onProgress func(Progress)) error {
	ext := filepath.Ext(inputPath)
	baseName := strings.TrimSuffix(inputPath, ext)
	outputPath := baseName + ".par2"

	args := buildArgs(cfg, outputPath, inputPath)
	cmd := exec.Command(binaryPath(), args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("erreur pipe stderr: %w", err)
	}

	cmd.Stdout = io.Discard

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erreur démarrage parpar: %w", err)
	}

	stderrLines := parseProgress(stderr, onProgress)

	if err := cmd.Wait(); err != nil {
		errMsg := strings.Join(stderrLines, "\n")
		onProgress(Progress{Done: true, Error: errMsg})
		return fmt.Errorf("erreur parpar: %w\n%s", err, errMsg)
	}

	onProgress(Progress{Percent: 100, Done: true})
	return nil
}

func buildArgs(cfg *config.ParParConfig, outputPath, inputPath string) []string {
	args := []string{
		"-s", cfg.SliceSize,
		"-S",
		"-m", cfg.Memory,
		"-t", strconv.Itoa(cfg.Threads),
		"-r", cfg.Redundancy,
		"-O",
		"-o", outputPath,
	}

	if cfg.ExtraArgs != "" {
		extra := strings.Fields(cfg.ExtraArgs)
		args = append(args, extra...)
	}

	args = append(args, inputPath)
	return args
}

func parseProgress(r io.Reader, onProgress func(Progress)) []string {
	scanner := bufio.NewScanner(r)
	scanner.Split(scanLines)

	var lines []string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		lines = append(lines, line)

		matches := progressRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			pct, err := strconv.ParseFloat(matches[1], 64)
			if err == nil {
				onProgress(Progress{Percent: pct})
			}
		}
	}

	return lines
}

func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' || data[i] == '\r' {
			return i + 1, data[:i], nil
		}
		// ANSI escape: \x1b[...X used by some tools for progress
		if data[i] == 0x1b && i+3 < len(data) && data[i+1] == '[' {
			for j := i + 2; j < len(data); j++ {
				if (data[j] >= 'A' && data[j] <= 'Z') || (data[j] >= 'a' && data[j] <= 'z') {
					if i > 0 {
						return j + 1, data[:i], nil
					}
					i = j
					break
				}
			}
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
