package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var appVersion = "1.3.6"

type UpdateInfo struct {
	Available bool   `json:"available"`
	Latest    string `json:"latest"`
	URL       string `json:"url"`
}

func (a *App) GetAppVersion() string {
	return appVersion
}

func (a *App) CheckUpdate() (UpdateInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/1UPFR/1UP/releases/latest")
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("erreur check update: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UpdateInfo{}, err
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return UpdateInfo{}, err
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(appVersion, "v")

	return UpdateInfo{
		Available: latest != current && latest != "",
		Latest:    latest,
		URL:       release.HTMLURL,
	}, nil
}
