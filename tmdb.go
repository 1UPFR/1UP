package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/1UPFR/1UP/internal/relparse"
)

var tmdbProxyBase = ""
var tmdbAPIKey = ""

const tmdbImageBase = "https://image.tmdb.org/t/p/w200"
const tmdbOfficialBase = "https://api.themoviedb.org/3"

type TMDBResult struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	Year       string  `json:"year"`
	PosterPath string  `json:"posterPath"`
	MediaType  string  `json:"mediaType"`
	Overview   string  `json:"overview"`
	Popularity float64 `json:"popularity"`
}

type TMDBDetails struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Year       string   `json:"year"`
	Overview   string   `json:"overview"`
	PosterPath string   `json:"posterPath"`
	MediaType  string   `json:"mediaType"`
	Genres     []string `json:"genres"`
	Rating     float64  `json:"rating"`
	Runtime    int      `json:"runtime"`
}

type tmdbSearchResult struct {
	Title         string          `json:"title"`
	Years         string          `json:"years"`
	EnglishTitle  string          `json:"english_title"`
	OriginalTitle string          `json:"original_title"`
	Poster        string          `json:"poster"`
	TmdbID        string          `json:"tmdb_id"`
	TmdbURL       string          `json:"tmdb_url"`
	ApiURL        string          `json:"api_url"`
	NoteTmdb      float64         `json:"note_tmdb"`
	Overview      string          `json:"overview"`
	Genres        json.RawMessage `json:"genres"`
}

type tmdbDetailResult struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	Name          string  `json:"name"`
	OriginalTitle string  `json:"original_title"`
	OriginalName  string  `json:"original_name"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	ReleaseDate   string  `json:"release_date"`
	FirstAirDate  string  `json:"first_air_date"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
	Genres        []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

// SearchTMDB cherche via le proxy, fallback sur l'API officielle
func (a *App) SearchTMDB(query string, mediaType string) ([]TMDBResult, error) {
	// Essayer le proxy d'abord
	if tmdbProxyBase != "" {
		results, err := searchProxy(query, mediaType)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}

	// Fallback API officielle
	if tmdbAPIKey != "" {
		return searchOfficial(query, mediaType)
	}

	return nil, fmt.Errorf("aucune source TMDB disponible")
}

// GetTMDBDetails via le proxy, fallback sur l'API officielle
func (a *App) GetTMDBDetails(id int, mediaType string) (TMDBDetails, error) {
	if tmdbProxyBase != "" {
		details, err := detailsProxy(id, mediaType)
		if err == nil && details.ID > 0 {
			return details, nil
		}
	}

	if tmdbAPIKey != "" {
		return detailsOfficial(id, mediaType)
	}

	return TMDBDetails{}, fmt.Errorf("aucune source TMDB disponible")
}

// ── Proxy ────────────────────────────────────────────────

func searchProxy(query string, mediaType string) ([]TMDBResult, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	params := url.Values{}
	params.Set("t", "search")
	params.Set("q", query)

	resp, err := client.Get(tmdbProxyBase + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Results []tmdbSearchResult `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	var results []TMDBResult
	for _, r := range raw.Results {
		id, _ := strconv.Atoi(r.TmdbID)
		if id == 0 {
			continue
		}
		title := r.Title
		if title == "" {
			title = r.EnglishTitle
		}
		if title == "" {
			title = r.OriginalTitle
		}
		mt := mediaType
		if mt == "" {
			mt = "movie"
		}
		if strings.Contains(r.ApiURL, "t=tv") || strings.Contains(r.TmdbURL, "/tv/") {
			mt = "tv"
		}
		results = append(results, TMDBResult{
			ID: id, Title: title, Year: r.Years, PosterPath: r.Poster,
			MediaType: mt, Overview: r.Overview, Popularity: r.NoteTmdb,
		})
	}
	return results, nil
}

func detailsProxy(id int, mediaType string) (TMDBDetails, error) {
	t := "movie"
	if mediaType == "tv" {
		t = "tv"
	}
	client := &http.Client{Timeout: 10 * time.Second}
	params := url.Values{}
	params.Set("t", t)
	params.Set("q", strconv.Itoa(id))

	resp, err := client.Get(tmdbProxyBase + "?" + params.Encode())
	if err != nil {
		return TMDBDetails{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return parseDetailResult(body, mediaType)
}

// ── API Officielle ───────────────────────────────────────

func searchOfficial(query string, mediaType string) ([]TMDBResult, error) {
	// Parser le nom de release pour extraire titre et annee
	info := relparse.Parse(query)
	if info.Title == "" {
		info.Title = query
	}

	// Determiner movie ou tv
	searchType := "movie"
	if info.IsTV || mediaType == "tv" {
		searchType = "tv"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	params := url.Values{}
	params.Set("api_key", tmdbAPIKey)
	params.Set("language", "fr-FR")
	params.Set("query", info.Title)
	params.Set("page", "1")
	params.Set("include_adult", "false")
	if info.Year != "" {
		params.Set("year", info.Year)
	}

	endpoint := fmt.Sprintf("%s/search/%s?%s", tmdbOfficialBase, searchType, params.Encode())
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var raw struct {
		Results []struct {
			ID           int     `json:"id"`
			Title        string  `json:"title"`
			Name         string  `json:"name"`
			Overview     string  `json:"overview"`
			PosterPath   string  `json:"poster_path"`
			ReleaseDate  string  `json:"release_date"`
			FirstAirDate string  `json:"first_air_date"`
			VoteAverage  float64 `json:"vote_average"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	// Si aucun resultat et on a pas essaye l'autre type, essayer
	if len(raw.Results) == 0 && mediaType == "" {
		otherType := "tv"
		if searchType == "tv" {
			otherType = "movie"
		}
		params.Set("query", info.Title)
		endpoint = fmt.Sprintf("%s/search/%s?%s", tmdbOfficialBase, otherType, params.Encode())
		resp2, err := client.Get(endpoint)
		if err == nil {
			defer resp2.Body.Close()
			body2, _ := io.ReadAll(resp2.Body)
			json.Unmarshal(body2, &raw)
			searchType = otherType
		}
	}

	var results []TMDBResult
	for _, r := range raw.Results {
		title := r.Title
		if title == "" {
			title = r.Name
		}
		year := r.ReleaseDate
		if year == "" {
			year = r.FirstAirDate
		}
		if len(year) > 4 {
			year = year[:4]
		}
		poster := ""
		if r.PosterPath != "" {
			poster = tmdbImageBase + r.PosterPath
		}
		results = append(results, TMDBResult{
			ID: r.ID, Title: title, Year: year, PosterPath: poster,
			MediaType: searchType, Overview: r.Overview, Popularity: r.VoteAverage,
		})
	}
	return results, nil
}

func detailsOfficial(id int, mediaType string) (TMDBDetails, error) {
	t := "movie"
	if mediaType == "tv" {
		t = "tv"
	}
	client := &http.Client{Timeout: 10 * time.Second}
	endpoint := fmt.Sprintf("%s/%s/%d?api_key=%s&language=fr-FR", tmdbOfficialBase, t, id, tmdbAPIKey)

	resp, err := client.Get(endpoint)
	if err != nil {
		return TMDBDetails{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return parseDetailResult(body, mediaType)
}

// ── Helpers ──────────────────────────────────────────────

func parseDetailResult(body []byte, mediaType string) (TMDBDetails, error) {
	var raw tmdbDetailResult
	if err := json.Unmarshal(body, &raw); err != nil {
		return TMDBDetails{}, err
	}

	title := raw.Title
	if title == "" {
		title = raw.Name
	}
	if title == "" {
		title = raw.OriginalTitle
	}
	if title == "" {
		title = raw.OriginalName
	}

	year := raw.ReleaseDate
	if year == "" {
		year = raw.FirstAirDate
	}
	if len(year) > 4 {
		year = year[:4]
	}

	var genres []string
	for _, g := range raw.Genres {
		if g.Name != "" {
			genres = append(genres, g.Name)
		}
	}

	poster := ""
	if raw.PosterPath != "" {
		poster = tmdbImageBase + raw.PosterPath
	}

	return TMDBDetails{
		ID: raw.ID, Title: title, Year: year, Overview: raw.Overview,
		PosterPath: poster, MediaType: mediaType, Genres: genres,
		Rating: raw.VoteAverage, Runtime: raw.Runtime,
	}, nil
}
