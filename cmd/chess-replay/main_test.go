package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takashi145/chess-replay/internal/app"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want options
	}{
		{"username only", []string{"alice"}, options{username: "alice"}},
		{"options before username", []string{"--month", "2026-08", "alice"}, options{username: "alice", month: "2026-08"}},
		{"options after username", []string{"alice", "--month", "2026-08", "--verbose"}, options{username: "alice", month: "2026-08", verbose: true}},
		{"single dash", []string{"alice", "-random"}, options{username: "alice", random: true}},
		{"month", []string{"alice", "--month", "2026-08"}, options{username: "alice", month: "2026-08"}},
		{"equals form", []string{"alice", "--month=2026-08"}, options{username: "alice", month: "2026-08"}},
		{"month and random", []string{"alice", "--month", "2026-08", "--random"}, options{username: "alice", month: "2026-08", random: true}},
		{"site", []string{"alice", "--site", "lichess"}, options{username: "alice", site: "lichess"}},
		{"site before username", []string{"--site=chesscom", "alice"}, options{username: "alice", site: "chesscom"}},
		{"set site needs no username", []string{"--set-site", "lichess"}, options{setSite: "lichess"}},
		{"show config needs no username", []string{"--show-config"}, options{showConfig: true}},
		{"version needs no username", []string{"--version"}, options{showVersion: true}},
	}

	for _, c := range cases {
		got, err := parseArgs(c.args)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestParseArgsRejectsBadInput(t *testing.T) {
	bad := map[string][]string{
		"no username":    {},
		"two usernames":  {"alice", "bob"},
		"unknown option": {"alice", "--nope"},
		"removed --last": {"alice", "--last", "5"},
		"missing value":  {"alice", "--month"},
		"unknown site":   {"alice", "--site", "nope"},
		"site no value":  {"alice", "--site"},

		"set unknown site":       {"--set-site", "nope"},
		"set site no value":      {"--set-site"},
		"set site empty":         {"--set-site="},
		"set site and show":      {"--set-site", "lichess", "--show-config"},
		"set site with name":     {"alice", "--set-site", "lichess"},
		"set site with site":     {"--set-site", "lichess", "--site", "lichess"},
		"show config with rand":  {"--show-config", "--random"},
		"show config with month": {"--show-config", "--month", "2026-08"},
	}

	for name, args := range bad {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestNewLoaderFollowsTheSite(t *testing.T) {
	if _, ok := newLoader("lichess", nil).(app.LichessLoader); !ok {
		t.Error("lichess: not a LichessLoader")
	}
	for _, site := range []string{"", "chesscom"} {
		if _, ok := newLoader(site, nil).(app.Loader); !ok {
			t.Errorf("%q: not a Loader", site)
		}
	}
}

func TestSiteFromFile(t *testing.T) {
	write := func(content string) string {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	for content, want := range map[string]string{
		`{"site": "lichess"}`:  "lichess",
		`{"site": "chesscom"}`: "chesscom",
		`{}`:                   "",
	} {
		if got, err := siteFromFile(write(content)); err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", content, got, err, want)
		}
	}

	path := write(`{"site": "nope"}`)
	if _, err := siteFromFile(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("invalid site: err = %v, want an error naming %s", err, path)
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		if _, err := parseArgs([]string{arg}); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("%s: err = %v, want flag.ErrHelp", arg, err)
		}
	}
}
