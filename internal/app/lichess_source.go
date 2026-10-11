package app

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/takashi145/chess-replay/internal/chesscom"
	"github.com/takashi145/chess-replay/internal/lichess"
)

const (
	// latestWindow is how many recent games the default mode looks through for a playable one.
	latestWindow = 20
	// randomWindow is how many recent games --random picks from.
	randomWindow = 100
	// maxListedGames bounds a list, so a busy month does not take minutes to download.
	maxListedGames = 500
)

type LichessAPI interface {
	Games(ctx context.Context, username string, q lichess.Query) ([]chesscom.Game, error)
}

// LichessLoader chooses games from Lichess for each command-line mode, like Loader does for Chess.com.
type LichessLoader struct {
	API  LichessAPI
	Logf func(format string, args ...any)
}

func (l LichessLoader) games(ctx context.Context, username string, q lichess.Query) ([]chesscom.Game, error) {
	games, err := l.API.Games(ctx, username, q)
	if err != nil {
		return nil, err
	}
	if l.Logf != nil {
		l.Logf("Fetched %d games", len(games))
	}
	return games, nil
}

func noPlayableGames(username string, games []chesscom.Game, archive *chesscom.Archive) error {
	if len(games) == 0 {
		return &chesscom.NoGamesError{Username: username, Archive: archive}
	}
	return &AllChess960Error{Count: len(games)}
}

func (l LichessLoader) Latest(ctx context.Context, username string) (Source, error) {
	games, err := l.games(ctx, username, lichess.Query{Max: latestWindow})
	if err != nil {
		return Source{}, err
	}

	playable := playableGames(games)
	if len(playable) == 0 {
		return Source{}, noPlayableGames(username, games, nil)
	}
	return Source{Games: playable[:1], Single: true}, nil
}

func (l LichessLoader) Random(ctx context.Context, username string) (Source, error) {
	games, err := l.games(ctx, username, lichess.Query{Max: randomWindow})
	if err != nil {
		return Source{}, err
	}

	playable := playableGames(games)
	if len(playable) == 0 {
		return Source{}, noPlayableGames(username, games, nil)
	}
	return Source{Games: []chesscom.Game{playable[rand.IntN(len(playable))]}, Single: true}, nil
}

// Month lists the games started in the given month, at most the newest maxListedGames of them.
func (l LichessLoader) Month(ctx context.Context, username string, archive chesscom.Archive) (Source, error) {
	since := time.Date(archive.Year, time.Month(archive.Month), 1, 0, 0, 0, 0, time.UTC)
	// One more than the limit tells whether the month has more games than are listed.
	games, err := l.games(ctx, username, lichess.Query{Max: maxListedGames + 1, Since: since, Until: since.AddDate(0, 1, 0)})
	if err != nil {
		return Source{}, err
	}

	truncated := len(games) > maxListedGames
	if truncated {
		games = games[:maxListedGames]
	}

	playable := playableGames(games)
	if len(playable) == 0 {
		return Source{}, noPlayableGames(username, games, &archive)
	}

	title := fmt.Sprintf("%s — Games in %s (UTC)", username, archive)
	if truncated {
		title += fmt.Sprintf(" — newest %d", maxListedGames)
	}
	return Source{Title: title, Games: playable, Hidden: len(games) - len(playable)}, nil
}

// RandomInMonth replays one game chosen from the given month.
func (l LichessLoader) RandomInMonth(ctx context.Context, username string, archive chesscom.Archive) (Source, error) {
	src, err := l.Month(ctx, username, archive)
	if err != nil {
		return Source{}, err
	}
	return Source{Games: []chesscom.Game{src.Games[rand.IntN(len(src.Games))]}, Single: true}, nil
}
