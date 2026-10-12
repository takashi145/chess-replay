// Package config reads and writes the optional settings file.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Config struct {
	Site string `json:"site"`
}

// Path returns where the settings file lives, inside the user's config directory.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "chess-replay", "config.json"), nil
}

// Save writes cfg to path, creating the directory if needed and replacing any existing file.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("Could not write %s: %w", path, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("Could not write %s: %w", path, err)
	}
	return nil
}

// Load reads the settings file at path. A missing file gives empty settings.
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("Could not read %s: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("Could not read %s: %w", path, err)
	}
	return cfg, nil
}
