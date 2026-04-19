package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type ParParConfig struct {
	SliceSize   string `json:"slice_size"`
	Memory      string `json:"memory"`
	Threads     int    `json:"threads"`
	Redundancy  string `json:"redundancy"`
	ExtraArgs   string `json:"extra_args"`
}

type NyuuConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	SSL         bool   `json:"ssl"`
	User        string `json:"user"`
	Password    string `json:"password"`
	Connections int    `json:"connections"`
	Group       string `json:"group"`
	ExtraArgs   string `json:"extra_args"`
}

type APIConfig struct {
	URL     string `json:"url"`
	APIKey  string `json:"apikey"`
	Enabled bool   `json:"enabled"`
}

type Config struct {
	ParPar    ParParConfig `json:"parpar"`
	Nyuu      NyuuConfig   `json:"nyuu"`
	API       APIConfig    `json:"api"`
	OutputDir string       `json:"output_dir"`
}

func DefaultConfig() *Config {
	return &Config{
		ParPar: ParParConfig{
			SliceSize:  "10M",
			Memory:     "4096M",
			Threads:    16,
			Redundancy: "20%",
		},
		Nyuu: NyuuConfig{
			Port:        563,
			SSL:         true,
			Connections: 20,
			Group:       "alt.binaries.boneless",
		},
		API: APIConfig{},
		OutputDir: "",
	}
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "1up")
	return dir, os.MkdirAll(dir, 0755)
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if saveErr := cfg.Save(); saveErr != nil {
				return nil, saveErr
			}
			return cfg, nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func BinaryDir() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	return filepath.Join("binaries", goos+"-"+goarch)
}
