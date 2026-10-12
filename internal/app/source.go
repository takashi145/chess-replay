// Package app decides which games to show for each command-line mode.
package app

import (
	"context"
	"fmt"
	"slices"

	"github.com/takashi145/chess-replay/internal/chesscom"
)

// API is the part of the Chess.com client the app needs.
type API interface {
	Archives(ctx context.Context, username string) ([]chesscom.Archive, error)
	Games(ctx context.Context, username string, archive chesscom.Archive) ([]chesscom.Game, error)
}

// Source is what the UI shows: a single game to replay right away, or a list to choose from.
type Source struct {
	Title  string
	Games  []chesscom.Game // newest first
	Single bool
	More   func(ctx context.Context) (Source, error) // fetches the games after these; nil when there are none
}

type Loader struct {
	API  API
	Logf func(format string, args ...any)
}

func (l Loader) logf(format string, args ...any) {
	if l.Logf != nil {
		l.Logf(format, args...)
	}
}

func (l Loader) archives(ctx context.Context, username string) ([]chesscom.Archive, error) {
	archives, err := l.API.Archives(ctx, username)
	if err != nil {
		return nil, err
	}
	if len(archives) == 0 {
		return nil, &chesscom.NoGamesError{Username: username}
	}
	return archives, nil
}

func (l Loader) games(ctx context.Context, username string, archive chesscom.Archive) ([]chesscom.Game, error) {
	games, err := l.API.Games(ctx, username, archive)
	if err != nil {
		return nil, err
	}
	l.logf("Archive %s: %d games", archive, len(games))
	return games, nil
}

func (l Loader) Latest(ctx context.Context, username string) (Source, error) {
	archives, err := l.archives(ctx, username)
	if err != nil {
		return Source{}, err
	}

	for i := len(archives) - 1; i >= 0; i-- {
		games, err := l.games(ctx, username, archives[i])
		if err != nil {
			return Source{}, err
		}

		if len(games) > 0 {
			sortNewestFirst(games)
			return Source{Games: games[:1], Single: true}, nil
		}
	}
	return Source{}, &chesscom.NoGamesError{Username: username}
}

func (l Loader) Month(ctx context.Context, username string, archive chesscom.Archive) (Source, error) {
	games, err := l.games(ctx, username, archive)
	if err != nil {
		return Source{}, err
	}
	if len(games) == 0 {
		return Source{}, &chesscom.NoGamesError{Username: username, Archive: &archive}
	}

	sortNewestFirst(games)
	return Source{
		Title: fmt.Sprintf("%s — Games in %s (UTC)", username, archive),
		Games: games,
	}, nil
}

func sortNewestFirst(games []chesscom.Game) {
	slices.SortStableFunc(games, func(a, b chesscom.Game) int { return b.PlayedAt.Compare(a.PlayedAt) })
}
