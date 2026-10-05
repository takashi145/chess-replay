// Package app decides which games to show for each command-line mode.
package app

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/takashi145/chess-replay/internal/chesscom"
)

// maxRandomArchives bounds how many months --random will fetch while looking for a playable game.
const maxRandomArchives = 6

// API is the part of the Chess.com client the app needs.
type API interface {
	Archives(ctx context.Context, username string) ([]chesscom.Archive, error)
	Games(ctx context.Context, username string, archive chesscom.Archive) ([]chesscom.Game, error)
}

// Source is what the UI shows: a single game to replay right away, or a list to choose from.
type Source struct {
	Title  string
	Games  []chesscom.Game // newest first
	Hidden int             // Chess960 games left out of Games
	Single bool
}

type AllChess960Error struct{ Count int }

func (e *AllChess960Error) Error() string {
	return fmt.Sprintf("All %d games are Chess960, which is not supported yet.", e.Count)
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

		playable := playableGames(games)
		if len(playable) > 0 {
			return Source{Games: playable[:1], Single: true}, nil
		}
	}
	return Source{}, &chesscom.NoGamesError{Username: username}
}

func (l Loader) Random(ctx context.Context, username string) (Source, error) {
	archives, err := l.archives(ctx, username)
	if err != nil {
		return Source{}, err
	}

	rand.Shuffle(len(archives), func(i, j int) { archives[i], archives[j] = archives[j], archives[i] })
	archives = archives[:min(maxRandomArchives, len(archives))]

	for _, archive := range archives {
		games, err := l.games(ctx, username, archive)
		if err != nil {
			return Source{}, err
		}

		if playable := playableGames(games); len(playable) > 0 {
			return Source{Games: []chesscom.Game{playable[rand.IntN(len(playable))]}, Single: true}, nil
		}
	}
	return Source{}, &chesscom.NoGamesError{Username: username}
}

func (l Loader) Last(ctx context.Context, username string, count int) (Source, error) {
	count = max(1, count)

	archives, err := l.archives(ctx, username)
	if err != nil {
		return Source{}, err
	}

	var collected []chesscom.Game
	playableCount := 0
	for i := len(archives) - 1; i >= 0 && playableCount < count; i-- {
		games, err := l.games(ctx, username, archives[i])
		if err != nil {
			return Source{}, err
		}
		collected = append(collected, games...)
		playableCount += len(playableGames(games))
	}

	sortNewestFirst(collected)

	var recent []chesscom.Game
	hidden := 0
	for _, game := range collected {
		if game.IsChess960() {
			hidden++
		} else {
			recent = append(recent, game)
		}

		if len(recent) == count {
			break
		}
	}

	if len(recent) == 0 {
		if hidden == 0 {
			return Source{}, &chesscom.NoGamesError{Username: username}
		}
		return Source{}, &AllChess960Error{Count: hidden}
	}
	return Source{Title: username + " — Recent games (UTC)", Games: recent, Hidden: hidden}, nil
}

func (l Loader) Month(ctx context.Context, username string, archive chesscom.Archive) (Source, error) {
	games, err := l.games(ctx, username, archive)
	if err != nil {
		return Source{}, err
	}
	if len(games) == 0 {
		return Source{}, &chesscom.NoGamesError{Username: username, Archive: &archive}
	}

	playable := playableGames(games)
	if len(playable) == 0 {
		return Source{}, &AllChess960Error{Count: len(games)}
	}

	return Source{
		Title:  fmt.Sprintf("%s — Games in %s (UTC)", username, archive),
		Games:  playable,
		Hidden: len(games) - len(playable),
	}, nil
}

// playableGames returns the non-Chess960 games, newest first.
func playableGames(games []chesscom.Game) []chesscom.Game {
	var playable []chesscom.Game
	for _, g := range games {
		if !g.IsChess960() {
			playable = append(playable, g)
		}
	}
	sortNewestFirst(playable)
	return playable
}

func sortNewestFirst(games []chesscom.Game) {
	slices.SortStableFunc(games, func(a, b chesscom.Game) int { return b.EndTime.Compare(a.EndTime) })
}
