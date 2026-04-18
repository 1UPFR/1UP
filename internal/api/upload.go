package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/1UPFR/1UP/internal/config"
)

type CheckResult struct {
	Code    int    `json:"code"`
	Explain string `json:"Explain"`
	Exists  bool   `json:"exists"`
}

func CheckRelease(cfg *config.APIConfig, releaseName string) (*CheckResult, error) {
	url := cfg.URL + cfg.APIKey + "&check=" + releaseName
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erreur check API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Code    json.RawMessage `json:"code"`
		Explain string          `json:"Explain"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	// code peut etre int ou string
	code := 0
	var codeInt int
	if json.Unmarshal(raw.Code, &codeInt) == nil {
		code = codeInt
	} else {
		var codeStr string
		if json.Unmarshal(raw.Code, &codeStr) == nil {
			fmt.Sscanf(codeStr, "%d", &code)
		}
	}

	return &CheckResult{
		Code:    code,
		Explain: raw.Explain,
		Exists:  code != 1,
	}, nil
}

type UploadResult struct {
	StatusCode int    `json:"status_code"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

func Upload(cfg *config.APIConfig, releaseName string, nzbPath string, mediainfoJSONPath string) (*UploadResult, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("clé API non configurée")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("rlsname", releaseName); err != nil {
		return nil, fmt.Errorf("erreur champ rlsname: %w", err)
	}

	if err := writer.WriteField("upload", "upload"); err != nil {
		return nil, fmt.Errorf("erreur champ upload: %w", err)
	}

	if err := addFile(writer, "nzb", nzbPath); err != nil {
		return nil, fmt.Errorf("erreur ajout nzb: %w", err)
	}

	if err := addFile(writer, "generated_nfo_json", mediainfoJSONPath); err != nil {
		return nil, fmt.Errorf("erreur ajout mediainfo: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("erreur fermeture multipart: %w", err)
	}

	url := cfg.URL + cfg.APIKey
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("erreur création requête: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur envoi API: %w", err)
	}
	defer resp.Body.Close()

	result := &UploadResult{
		StatusCode: resp.StatusCode,
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	if !result.Success {
		respBody, _ := io.ReadAll(resp.Body)
		result.Error = string(respBody)
	}

	return result, nil
}

type ManualUploadParams struct {
	ReleaseName      string
	NZBPath          string
	MediaInfoJSONPath string
	BDInfoFullPath   string
	BDInfoMiniPath   string
}

func UploadManual(cfg *config.APIConfig, params ManualUploadParams) (*UploadResult, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("cle API non configuree")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("upload", "upload")
	writer.WriteField("rlsname", params.ReleaseName)

	if err := addFile(writer, "nzb", params.NZBPath); err != nil {
		return nil, fmt.Errorf("erreur ajout nzb: %w", err)
	}

	if params.MediaInfoJSONPath != "" {
		if err := addFile(writer, "generated_nfo_json", params.MediaInfoJSONPath); err != nil {
			return nil, fmt.Errorf("erreur ajout mediainfo: %w", err)
		}
	}

	if params.BDInfoFullPath != "" {
		if err := addFile(writer, "bdinfo_full", params.BDInfoFullPath); err != nil {
			return nil, fmt.Errorf("erreur ajout bdinfo_full: %w", err)
		}
	}

	if params.BDInfoMiniPath != "" {
		if err := addFile(writer, "bdinfo_mini", params.BDInfoMiniPath); err != nil {
			return nil, fmt.Errorf("erreur ajout bdinfo_mini: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := cfg.URL + cfg.APIKey
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur envoi API: %w", err)
	}
	defer resp.Body.Close()

	result := &UploadResult{
		StatusCode: resp.StatusCode,
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
	}
	if !result.Success {
		respBody, _ := io.ReadAll(resp.Body)
		result.Error = string(respBody)
	}
	return result, nil
}

func addFile(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	return err
}
