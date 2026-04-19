package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var tmdbProxyBase = ""
const tmdbImageBase = "https://image.tmdb.org/t/p/w200"

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

func (a *App) SearchTMDB(query string, mediaType string) ([]TMDBResult, error) {
	params := url.Values{}
	params.Set("t", "search")
	params.Set("q", query)

	resp, err := http.Get(tmdbProxyBase + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("erreur connexion TMDB: %w", err)
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
			ID:         id,
			Title:      title,
			Year:       r.Years,
			PosterPath: r.Poster,
			MediaType:  mt,
			Overview:   r.Overview,
			Popularity: r.NoteTmdb,
		})
	}
	return results, nil
}

func (a *App) GetTMDBDetails(id int, mediaType string) (TMDBDetails, error) {
	t := "movie"
	if mediaType == "tv" {
		t = "tv"
	}

	params := url.Values{}
	params.Set("t", t)
	params.Set("q", strconv.Itoa(id))

	resp, err := http.Get(tmdbProxyBase + "?" + params.Encode())
	if err != nil {
		return TMDBDetails{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TMDBDetails{}, err
	}

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
		ID:         raw.ID,
		Title:      title,
		Year:       year,
		Overview:   raw.Overview,
		PosterPath: poster,
		MediaType:  mediaType,
		Genres:     genres,
		Rating:     raw.VoteAverage,
		Runtime:    raw.Runtime,
	}, nil
}
