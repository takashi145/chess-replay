package main

import (
	"errors"
	"flag"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want options
	}{
		{"username only", []string{"alice"}, options{username: "alice"}},
		{"options before username", []string{"--last", "5", "alice"}, options{username: "alice", last: 5, lastSet: true}},
		{"options after username", []string{"alice", "--last", "5", "--verbose"}, options{username: "alice", last: 5, lastSet: true, verbose: true}},
		{"single dash", []string{"alice", "-random"}, options{username: "alice", random: true}},
		{"month", []string{"alice", "--month", "2026-08"}, options{username: "alice", month: "2026-08"}},
		{"explicit zero is still set", []string{"alice", "--last", "0"}, options{username: "alice", lastSet: true}},
		{"equals form", []string{"alice", "--last=3"}, options{username: "alice", last: 3, lastSet: true}},
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
		"last not int":   {"alice", "--last", "many"},
		"missing value":  {"alice", "--month"},
	}

	for name, args := range bad {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		if _, err := parseArgs([]string{arg}); !errors.Is(err, flag.ErrHelp) {
			t.Errorf("%s: err = %v, want flag.ErrHelp", arg, err)
		}
	}
}
