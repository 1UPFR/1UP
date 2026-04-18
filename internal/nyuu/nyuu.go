package nyuu

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/1UPFR/1UP/internal/binutil"
	"github.com/1UPFR/1UP/internal/config"
)

type Progress struct {
	Percent  float64 `json:"percent"`
	Articles string  `json:"articles"`
	Speed    string  `json:"speed"`
	ETA      string  `json:"eta"`
	Done     bool    `json:"done"`
	Error    string  `json:"error,omitempty"`
}

type Result struct {
	NZBPath string `json:"nzb_path"`
}

var progressRegex = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
var articlesRegex = regexp.MustCompile(`(\d+)\s*/\s*(\d+)`)
var speedRegex = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*([KMGT]i?B/s|B/s)`)
var etaRegex = regexp.MustCompile(`ETA\s+(\d{2}:\d{2}(?::\d{2})?)`)


func binaryPath() string {
	// Essayer le binaire embarqué
	if path, err := binutil.ExtractBinary("nyuu"); err == nil {
		return path
	}
	// Fallback sur le PATH système
	if path, err := exec.LookPath("nyuu"); err == nil {
		return path
	}
	return "nyuu"
}

func Run(cfg *config.NyuuConfig, inputFiles []string, nzbOutputPath string, releaseName string, onProgress func(Progress)) (*Result, error) {
	args := buildArgs(cfg, inputFiles, nzbOutputPath, releaseName)
	cmd := exec.Command(binaryPath(), args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("erreur pipe stderr: %w", err)
	}

	cmd.Stdout = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("erreur démarrage nyuu: %w", err)
	}

	stderrLines := parseProgress(stderr, onProgress)

	if err := cmd.Wait(); err != nil {
		errMsg := strings.Join(stderrLines, "\n")
		onProgress(Progress{Done: true, Error: errMsg})
		return nil, fmt.Errorf("erreur nyuu: %w\n%s", err, errMsg)
	}

	onProgress(Progress{Percent: 100, Done: true})
	return &Result{NZBPath: nzbOutputPath}, nil
}

func buildArgs(cfg *config.NyuuConfig, inputFiles []string, nzbOutputPath string, releaseName string) []string {
	args := []string{
		"-h", cfg.Host,
		"-P", strconv.Itoa(cfg.Port),
		"-u", cfg.User,
		"-p", cfg.Password,
		"-n", strconv.Itoa(cfg.Connections),
		"-g", cfg.Group,
		"-o", nzbOutputPath,
		"--nzb-title", releaseName,
		"-f", "{rand(14)} {rand(14)}@{rand(5)}.{rand(3)}",
		"--message-id", "{rand(32)}@{rand(8)}.{rand(3)}",
		"--subject", "{rand(32)}",
		"--nzb-subject", `[{0filenum}/{files}] - "{filename}" yEnc ({part}/{parts})`,
		"--obfuscate-articles",
		"--overwrite",
		"--progress=stderr",
	}

	if cfg.SSL {
		args = append(args, "-S")
	}

	if cfg.ExtraArgs != "" {
		extra := strings.Fields(cfg.ExtraArgs)
		args = append(args, extra...)
	}

	args = append(args, inputFiles...)
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

		p := Progress{}
		updated := false

		if matches := progressRegex.FindStringSubmatch(line); len(matches) >= 2 {
			if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
				p.Percent = pct
				updated = true
			}
		}

		if matches := articlesRegex.FindStringSubmatch(line); len(matches) >= 3 {
			p.Articles = matches[1] + "/" + matches[2]
			updated = true
		}

		if matches := speedRegex.FindStringSubmatch(line); len(matches) >= 3 {
			p.Speed = matches[1] + " " + matches[2]
			updated = true
		}

		if matches := etaRegex.FindStringSubmatch(line); len(matches) >= 2 {
			p.ETA = matches[1]
			updated = true
		}

		if updated {
			onProgress(p)
		}
	}

	return lines
}

func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		// Standard line breaks
		if data[i] == '\n' || data[i] == '\r' {
			return i + 1, data[:i], nil
		}
		// ANSI escape: \x1b[0G (cursor to column 0) used by Nyuu
		if data[i] == 0x1b && i+3 < len(data) && data[i+1] == '[' {
			// Find end of escape sequence (letter)
			for j := i + 2; j < len(data); j++ {
				if (data[j] >= 'A' && data[j] <= 'Z') || (data[j] >= 'a' && data[j] <= 'z') {
					if i > 0 {
						return j + 1, data[:i], nil
					}
					// Skip the escape sequence itself
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
