package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/takashi145/chess-replay/internal/config"
)

func validSite(site string) bool { return site == siteChessCom || site == siteLichess }

// defaultSite returns the site set in the settings file, or "" when there is none.
func defaultSite() (string, error) {
	path, err := config.Path()
	if err != nil {
		return "", nil // Without a config directory there is no settings file to read.
	}
	return siteFromFile(path)
}

func siteFromFile(path string) (string, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if cfg.Site != "" && !validSite(cfg.Site) {
		return "", fmt.Errorf("Invalid site %q in %s: expected %s or %s.", cfg.Site, path, siteChessCom, siteLichess)
	}
	return cfg.Site, nil
}

// showConfig prints where the settings file is and which site is the default.
func showConfig(w io.Writer, path string) error {
	site, err := siteFromFile(path)
	if err != nil {
		return err
	}

	found := ""
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		found = " (not found)"
	}
	source := ""
	if site == "" {
		site, source = siteChessCom, " (built-in)"
	}
	fmt.Fprintf(w, "Config file: %s%s\nDefault site: %s%s\n", path, found, site, source)
	return nil
}

// setSite saves site as the default, replacing whatever the settings file held.
func setSite(w io.Writer, path, site string) error {
	if err := config.Save(path, config.Config{Site: site}); err != nil {
		return err
	}
	fmt.Fprintf(w, "Default site set to %s.\nConfig file: %s\n", site, path)
	return nil
}

// runConfigCommand handles --show-config and --set-site, which need no username or terminal.
func runConfigCommand(opts options, stdout, stderr io.Writer) int {
	path, err := config.Path()
	if err != nil {
		printError(stderr, err.Error())
		return exitError
	}

	if opts.showConfig {
		err = showConfig(stdout, path)
	} else {
		err = setSite(stdout, path, opts.setSite)
	}
	if err != nil {
		printError(stderr, err.Error())
		return exitError
	}
	return exitOK
}
