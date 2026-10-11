package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestShowConfig(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	saved := filepath.Join(dir, "saved.json")
	if err := os.WriteFile(saved, []byte(`{"site": "lichess"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, tc := range map[string]struct{ path, want string }{
		"no file": {missing, "Config file: " + missing + " (not found)\nDefault site: chesscom (built-in)\n"},
		"saved":   {saved, "Config file: " + saved + "\nDefault site: lichess\n"},
	} {
		var out bytes.Buffer

		if err := showConfig(&out, tc.path); err != nil || out.String() != tc.want {
			t.Errorf("%s: got %q, %v; want %q", name, out.String(), err, tc.want)
		}
	}
}

func TestSetSiteSavesTheSiteAndCreatesTheDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "dir", "config.json")
	var out bytes.Buffer

	if err := setSite(&out, path, "lichess"); err != nil {
		t.Fatal(err)
	}

	if site, err := siteFromFile(path); err != nil || site != "lichess" {
		t.Errorf("saved site = %q, %v", site, err)
	}
	if want := "Default site set to lichess.\nConfig file: " + path + "\n"; out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
}

func TestSetSiteReplacesAFileThatCannotBeRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := setSite(&bytes.Buffer{}, path, "chesscom"); err != nil {
		t.Fatal(err)
	}

	if site, err := siteFromFile(path); err != nil || site != "chesscom" {
		t.Errorf("saved site = %q, %v", site, err)
	}
}
