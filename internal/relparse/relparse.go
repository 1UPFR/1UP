package relparse

import (
	"regexp"
	"strings"
)

type ReleaseInfo struct {
	Title  string
	Year   string
	Season string
	IsTV   bool
}

var (
	yearRe   = regexp.MustCompile(`[\.\s\-\(]((19|20)\d{2})[\.\s\-\)]`)
	seasonRe = regexp.MustCompile(`(?i)[\.\s\-]S(\d{1,2})(?:E\d+)?[\.\s\-]`)
	// Tokens qui indiquent la fin du titre
	stopTokens = regexp.MustCompile(`(?i)^(MULTI|MULTi|FRENCH|ENGLISH|VOSTFR|TRUEFRENCH|SUBFRENCH|VFF|VFQ|VF2|` +
		`2160p|1080p|720p|480p|UHD|4K|` +
		`BluRay|Blu-Ray|BDRip|BDRemux|REMUX|BDMV|WEB-DL|WEBRip|WEBDL|WEB|HDTV|DVDRip|HDRip|` +
		`x264|x265|H\.?264|H\.?265|HEVC|AVC|AV1|XviD|DivX|` +
		`DTS|DTS-HD|TrueHD|Atmos|AC3|EAC3|AAC|FLAC|DD5|DDP|` +
		`HDR|HDR10|DV|DoVi|SDR|HLG|` +
		`COMPLETE|PROPER|REPACK|iNTERNAL|EXTENDED|UNRATED|DC|DIRECTORS|` +
		`NF|AMZN|DSNP|ATVP|HMAX|PMTP|PCOK|CRAV|SHO|` +
		`S\d{1,2}|S\d{1,2}E\d+)$`)
)

func Parse(name string) ReleaseInfo {
	info := ReleaseInfo{}

	// Detecter la saison -> c'est une serie
	if m := seasonRe.FindStringSubmatch(name); len(m) > 1 {
		info.Season = m[1]
		info.IsTV = true
	}

	// Detecter l'annee
	if m := yearRe.FindStringSubmatch(name); len(m) > 1 {
		info.Year = m[1]
	}

	// Extraire le titre : tout avant l'annee ou le premier token stop
	// Remplacer les . et _ par des espaces
	clean := strings.ReplaceAll(name, "_", " ")
	tokens := strings.FieldsFunc(clean, func(r rune) bool {
		return r == '.' || r == '-' || r == ' '
	})

	var titleParts []string
	for _, t := range tokens {
		// Arreter au premier token stop ou a l'annee
		if stopTokens.MatchString(t) {
			break
		}
		if t == info.Year {
			break
		}
		titleParts = append(titleParts, t)
	}

	info.Title = strings.Join(titleParts, " ")
	return info
}
