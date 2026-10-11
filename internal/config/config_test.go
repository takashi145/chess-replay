package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadReadsTheSite(t *testing.T) {
	cfg, err := Load(writeConfig(t, `{"site": "lichess"}`))

	if err != nil || cfg.Site != "lichess" {
		t.Errorf("cfg = %+v, err = %v", cfg, err)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "config.json")

	if err := Save(path, Config{Site: "lichess"}); err != nil {
		t.Fatal(err)
	}

	if cfg, err := Load(path); err != nil || cfg.Site != "lichess" {
		t.Errorf("cfg = %+v, err = %v", cfg, err)
	}
}

func TestLoadWithoutAFileGivesEmptySettings(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))

	if err != nil || cfg != (Config{}) {
		t.Errorf("cfg = %+v, err = %v", cfg, err)
	}
}

func TestLoadRejectsAFileItCannotUse(t *testing.T) {
	for name, content := range map[string]string{
		"not json":      "site = lichess",
		"empty file":    "",
		"unknown field": `{"sit": "lichess"}`,
		"wrong type":    `{"site": 3}`,
	} {
		path := writeConfig(t, content)

		_, err := Load(path)

		if err == nil || !strings.Contains(err.Error(), path) {
			t.Errorf("%s: err = %v, want an error naming %s", name, err, path)
		}
	}
}
