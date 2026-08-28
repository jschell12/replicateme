package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Provider        string `json:"provider"`
	Model           string `json:"model,omitempty"`
	QuirkLevel      int    `json:"quirkLevel"`
	DefaultPlatform string `json:"defaultPlatform"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".replicateme")
}

func ConfigDir() string {
	return configDir()
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func Load() Config {
	cfg := Config{
		Provider:        "anthropic",
		QuirkLevel:      50,
		DefaultPlatform: "imessage",
	}

	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}

	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath(), append(data, '\n'), 0644)
}
