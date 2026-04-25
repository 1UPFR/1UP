package bdinfo

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type ParsedBDInfo struct {
	Resolution        string `json:"resolution"`
	VideoCodec        string `json:"videoCodec"`
	AudioCodec        string `json:"audioCodec"`
	AudioLanguages    string `json:"audioLanguages"`
	SubtitleLanguages string `json:"subtitleLanguages"`
	HDRFormat         string `json:"hdrFormat"`
	Duration          string `json:"duration"`
	FileSize          int64  `json:"fileSize"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	Bitrate           int    `json:"bitrate"`
	FrameRate         float64 `json:"frameRate"`
}

func ParseFile(path string) (*ParsedBDInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}

func Parse(content string) (*ParsedBDInfo, error) {
	info := &ParsedBDInfo{}

	// Disc Size
	if m := regexp.MustCompile(`Disc Size:\s+([\d,]+)\s+bytes`).FindStringSubmatch(content); len(m) > 1 {
		info.FileSize, _ = strconv.ParseInt(strings.ReplaceAll(m[1], ",", ""), 10, 64)
	}

	// Chercher la section PLAYLIST REPORT
	playlistIdx := strings.Index(content, "PLAYLIST REPORT:")
	if playlistIdx < 0 {
		return nil, fmt.Errorf("section PLAYLIST REPORT non trouvee")
	}
	playlist := content[playlistIdx:]

	// Duration
	if m := regexp.MustCompile(`Length:\s+(\d+):(\d+):(\d+)`).FindStringSubmatch(playlist); len(m) > 3 {
		h, _ := strconv.Atoi(m[1])
		min, _ := strconv.Atoi(m[2])
		info.Duration = fmt.Sprintf("%dh %02dmin", h, min)
	}

	// Total Bitrate
	if m := regexp.MustCompile(`Total Bitrate:\s+([\d.]+)\s+Mbps`).FindStringSubmatch(playlist); len(m) > 1 {
		mbps, _ := strconv.ParseFloat(m[1], 64)
		info.Bitrate = int(mbps * 1000000)
	}

	// VIDEO section
	parseVideo(playlist, info)

	// AUDIO section
	parseAudio(playlist, info)

	// SUBTITLES section
	parseSubtitles(playlist, info)

	return info, nil
}

func parseVideo(content string, info *ParsedBDInfo) {
	vidIdx := strings.Index(content, "VIDEO:")
	if vidIdx < 0 {
		return
	}
	vidSection := content[vidIdx:]

	// Trouver la fin de la section VIDEO (prochaine section)
	for _, end := range []string{"AUDIO:", "SUBTITLES:", "FILES:", "CHAPTERS:"} {
		if idx := strings.Index(vidSection[6:], end); idx >= 0 {
			vidSection = vidSection[:idx+6]
			break
		}
	}

	// Codec
	if strings.Contains(vidSection, "HEVC") || strings.Contains(vidSection, "MPEG-H HEVC") {
		info.VideoCodec = "HEVC"
	} else if strings.Contains(vidSection, "AVC") || strings.Contains(vidSection, "MPEG-4 AVC") {
		info.VideoCodec = "AVC"
	} else if strings.Contains(vidSection, "VC-1") {
		info.VideoCodec = "VC-1"
	} else if strings.Contains(vidSection, "MPEG-2") {
		info.VideoCodec = "MPEG-2"
	}

	// Resolution
	if m := regexp.MustCompile(`(\d{3,4})p`).FindStringSubmatch(vidSection); len(m) > 1 {
		h, _ := strconv.Atoi(m[1])
		info.Height = h
		switch {
		case h >= 2160:
			info.Width = 3840
			info.Resolution = "2160p"
		case h >= 1080:
			info.Width = 1920
			info.Resolution = "1080p"
		case h >= 720:
			info.Width = 1280
			info.Resolution = "720p"
		case h >= 576:
			info.Width = 720
			info.Resolution = "576p"
		case h >= 480:
			info.Width = 720
			info.Resolution = "480p"
		default:
			info.Resolution = fmt.Sprintf("%dp", h)
		}
	} else if m := regexp.MustCompile(`(\d{3,4})i`).FindStringSubmatch(vidSection); len(m) > 1 {
		h, _ := strconv.Atoi(m[1])
		info.Height = h
		info.Width = 1920
		info.Resolution = fmt.Sprintf("%di", h)
	}

	// Frame rate
	if m := regexp.MustCompile(`([\d.]+)\s*fps`).FindStringSubmatch(vidSection); len(m) > 1 {
		fr, _ := strconv.ParseFloat(m[1], 64)
		info.FrameRate = math.Round(fr*100) / 100
	}

	// HDR
	desc := strings.ToLower(vidSection)
	switch {
	case strings.Contains(desc, "dolby vision") && strings.Contains(desc, "hdr10"):
		info.HDRFormat = "HDR DV"
	case strings.Contains(desc, "dolby vision"):
		info.HDRFormat = "DV"
	case strings.Contains(desc, "hdr10+"):
		info.HDRFormat = "HDR10+"
	case strings.Contains(desc, "hdr10"):
		info.HDRFormat = "HDR10"
	case strings.Contains(desc, "pq") || strings.Contains(desc, "st 2084"):
		info.HDRFormat = "HDR10"
	case strings.Contains(desc, "hlg"):
		info.HDRFormat = "HLG"
	}
}

var langMap = map[string]string{
	"english":    "Anglais",
	"french":     "Francais",
	"german":     "Allemand",
	"spanish":    "Espagnol",
	"italian":    "Italien",
	"portuguese": "Portugais",
	"japanese":   "Japonais",
	"chinese":    "Chinois",
	"korean":     "Coreen",
	"russian":    "Russe",
	"dutch":      "Neerlandais",
	"arabic":     "Arabe",
	"hindi":      "Hindi",
	"polish":     "Polonais",
	"turkish":    "Turc",
	"swedish":    "Suedois",
	"norwegian":  "Norvegien",
	"danish":     "Danois",
	"finnish":    "Finnois",
	"czech":      "Tcheque",
	"hungarian":  "Hongrois",
	"thai":       "Thai",
	"greek":      "Grec",
	"romanian":   "Roumain",
	"hebrew":     "Hebreu",
}

func normalizeLang(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	if mapped, ok := langMap[l]; ok {
		return mapped
	}
	if l != "" {
		// Capitalize first letter
		return strings.ToUpper(l[:1]) + l[1:]
	}
	return ""
}

func parseAudio(content string, info *ParsedBDInfo) {
	audioIdx := strings.Index(content, "AUDIO:")
	if audioIdx < 0 {
		return
	}
	audioSection := content[audioIdx:]

	// Trouver la fin
	for _, end := range []string{"SUBTITLES:", "FILES:", "CHAPTERS:"} {
		if idx := strings.Index(audioSection[6:], end); idx >= 0 {
			audioSection = audioSection[:idx+6]
			break
		}
	}

	// Chaque ligne audio : Codec Language Bitrate Description
	// Ex: DTS-HD Master Audio             French           2037 kbps      2.0 / 48 kHz / ...
	re := regexp.MustCompile(`(?m)^(DTS-HD Master Audio|DTS-HD High Resolution Audio|DTS|Dolby TrueHD|Dolby Digital|Dolby Digital Plus|Dolby Atmos|TrueHD|PCM|LPCM|FLAC|AAC)\s+(\w+)\s+`)

	matches := re.FindAllStringSubmatch(audioSection, -1)

	langs := []string{}
	seen := map[string]bool{}
	for i, m := range matches {
		if i == 0 {
			info.AudioCodec = normalizeAudioCodec(m[1])
		}
		lang := normalizeLang(m[2])
		if lang != "" && !seen[lang] {
			langs = append(langs, lang)
			seen[lang] = true
		}
	}
	info.AudioLanguages = strings.Join(langs, ", ")
}

func normalizeAudioCodec(codec string) string {
	c := strings.ToLower(codec)
	switch {
	case strings.Contains(c, "truehd") && strings.Contains(c, "atmos"):
		return "TrueHD Atmos"
	case strings.Contains(c, "truehd"):
		return "TrueHD"
	case strings.Contains(c, "dts-hd master"):
		return "DTS-HD MA"
	case strings.Contains(c, "dts-hd high"):
		return "DTS-HD HR"
	case strings.Contains(c, "dts"):
		return "DTS"
	case strings.Contains(c, "dolby digital plus"):
		return "EAC3"
	case strings.Contains(c, "dolby digital"):
		return "AC3"
	case strings.Contains(c, "dolby atmos"):
		return "Atmos"
	case strings.Contains(c, "pcm") || strings.Contains(c, "lpcm"):
		return "LPCM"
	case strings.Contains(c, "flac"):
		return "FLAC"
	case strings.Contains(c, "aac"):
		return "AAC"
	}
	return codec
}

func parseSubtitles(content string, info *ParsedBDInfo) {
	subIdx := strings.Index(content, "SUBTITLES:")
	if subIdx < 0 {
		return
	}
	subSection := content[subIdx:]

	// Trouver la fin
	for _, end := range []string{"FILES:", "CHAPTERS:", "STREAM DIAGNOSTICS:"} {
		if idx := strings.Index(subSection[10:], end); idx >= 0 {
			subSection = subSection[:idx+10]
			break
		}
	}

	// Ex: Presentation Graphics           French          24.04 kbps      1920x1080 / 964 Captions
	re := regexp.MustCompile(`(?m)^Presentation Graphics\s+(\w+)\s+`)
	matches := re.FindAllStringSubmatch(subSection, -1)

	subs := []string{}
	seen := map[string]bool{}
	for _, m := range matches {
		lang := normalizeLang(m[1])
		if lang != "" && !seen[lang] {
			subs = append(subs, lang)
			seen[lang] = true
		}
	}
	info.SubtitleLanguages = strings.Join(subs, ", ")
}
