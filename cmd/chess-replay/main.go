package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"

	"github.com/takashi145/chess-replay/internal/app"
	"github.com/takashi145/chess-replay/internal/chesscom"
	"github.com/takashi145/chess-replay/internal/lichess"
	"github.com/takashi145/chess-replay/internal/ui"
)

// Set at build time by goreleaser.
var version = "dev"

const (
	exitOK        = 0
	exitError     = 1
	exitCancelled = 130
)

const (
	siteChessCom = "chesscom"
	siteLichess  = "lichess"
)

const usage = `Replay Chess.com or Lichess public games in the terminal.

Usage: chess-replay <username> [options]

Options:
  --site <chesscom|lichess>  Site to load games from (default: chesscom)
  --random                   Replay a random past game
  --month <YYYY-MM>          List games from a specific month
  --verbose                  Print diagnostic information
  --version                  Print the version
  -h, --help                 Show this help`

type options struct {
	username    string
	site        string
	random      bool
	month       string
	verbose     bool
	showVersion bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func parseArgs(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("chess-replay", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.site, "site", "", "")
	fs.BoolVar(&opts.random, "random", false, "")
	fs.StringVar(&opts.month, "month", "", "")
	fs.BoolVar(&opts.verbose, "verbose", false, "")
	fs.BoolVar(&opts.showVersion, "version", false, "")

	// The flag package stops at the first positional argument; keep going so options may follow the username.
	var positional []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return opts, err
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		rest = fs.Args()[1:]
	}

	if opts.showVersion {
		return opts, nil
	}

	if opts.site != "" && opts.site != siteChessCom && opts.site != siteLichess {
		return opts, fmt.Errorf("--site expects %s or %s.", siteChessCom, siteLichess)
	}

	if len(positional) != 1 {
		return opts, errors.New("expected exactly one <username> argument")
	}
	opts.username = positional[0]
	return opts, nil
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(stdout, usage)
		return exitOK
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, usage)
		return exitError
	}
	if opts.showVersion {
		fmt.Fprintln(stdout, version)
		return exitOK
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		printError(stderr, "chess-replay needs an interactive terminal.")
		return exitError
	}

	var logf func(string, ...any)
	if opts.verbose {
		logf = func(format string, args ...any) { fmt.Fprintf(stderr, "[verbose] "+format+"\n", args...) }
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	src, err := load(ctx, opts, logf)
	if err != nil {
		return reportError(stderr, err)
	}

	if _, err := tea.NewProgram(ui.New(src, opts.username)).Run(); err != nil {
		fmt.Fprintln(stderr, err)
		return exitError
	}
	return exitOK
}

// loader is what both sites' loaders offer, one method per mode.
type loader interface {
	Latest(ctx context.Context, username string) (app.Source, error)
	Random(ctx context.Context, username string) (app.Source, error)
	Month(ctx context.Context, username string, archive chesscom.Archive) (app.Source, error)
	RandomInMonth(ctx context.Context, username string, archive chesscom.Archive) (app.Source, error)
}

func newLoader(site string, logf func(string, ...any)) loader {
	if site == siteLichess {
		return app.LichessLoader{API: lichess.New(version, logf), Logf: logf}
	}
	return app.Loader{API: chesscom.New(version, logf), Logf: logf}
}

func load(ctx context.Context, opts options, logf func(string, ...any)) (app.Source, error) {
	loader := newLoader(opts.site, logf)

	switch {
	case opts.month != "":
		archive, ok := chesscom.ParseArchive(opts.month)
		if !ok {
			return app.Source{}, errors.New("--month expects the format YYYY-MM.")
		}
		if opts.random {
			return loader.RandomInMonth(ctx, opts.username, archive)
		}
		return loader.Month(ctx, opts.username, archive)
	case opts.random:
		return loader.Random(ctx, opts.username)
	default:
		return loader.Latest(ctx, opts.username)
	}
}

func reportError(stderr io.Writer, err error) int {
	if errors.Is(err, context.Canceled) {
		return exitCancelled
	}

	printError(stderr, err.Error())

	var conn *chesscom.ConnectionError
	if errors.As(err, &conn) {
		printError(stderr, "Check your internet connection and try again.")
	}
	return exitError
}

func printError(w io.Writer, message string) {
	fmt.Fprintf(w, "\x1b[31m%s\x1b[0m\n", message)
}
