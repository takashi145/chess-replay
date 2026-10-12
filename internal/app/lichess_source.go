package app

import (
	"context"
	"fmt"
	"time"

	"github.com/takashi145/chess-replay/internal/chesscom"
	"github.com/takashi145/chess-replay/internal/lichess"
)

const (
	// latestWindow is how many recent games the default mode looks through for a playable one.
	latestWindow = 20
	// pageSize is how many games a list fetches at a time.
	pageSize = 30
)

type LichessAPI interface {
	Games(ctx context.Context, username string, q lichess.Query) (lichess.Page, error)
}

// LichessLoader chooses games from Lichess for each command-line mode, like Loader does for Chess.com.
type LichessLoader struct {
	API  LichessAPI
	Logf func(format string, args ...any)
}

func (l LichessLoader) games(ctx context.Context, username string, q lichess.Query) (lichess.Page, error) {
	page, err := l.API.Games(ctx, username, q)
	if err != nil {
		return lichess.Page{}, err
	}
	if l.Logf != nil {
		l.Logf("Fetched %d games", len(page.Games))
	}
	return page, nil
}

func monthQuery(archive chesscom.Archive) lichess.Query {
	since := time.Date(archive.Year, time.Month(archive.Month), 1, 0, 0, 0, 0, time.UTC)
	return lichess.Query{Since: since, Until: since.AddDate(0, 1, 0)}
}

func (l LichessLoader) Latest(ctx context.Context, username string) (Source, error) {
	page, err := l.games(ctx, username, lichess.Query{Max: latestWindow})
	if err != nil {
		return Source{}, err
	}

	if len(page.Games) == 0 {
		return Source{}, &chesscom.NoGamesError{Username: username}
	}
	return Source{Games: page.Games[:1], Single: true}, nil
}

// Month lists the games started in the given month, a page at a time.
func (l LichessLoader) Month(ctx context.Context, username string, archive chesscom.Archive) (Source, error) {
	title := fmt.Sprintf("%s — Games in %s (UTC)", username, archive)
	return l.list(ctx, username, monthQuery(archive), title, &archive)
}

func (l LichessLoader) list(ctx context.Context, username string, q lichess.Query, title string, archive *chesscom.Archive) (Source, error) {
	src, err := l.page(ctx, username, q)
	if err != nil {
		return Source{}, err
	}
	if len(src.Games) == 0 {
		return Source{}, &chesscom.NoGamesError{Username: username, Archive: archive}
	}

	src.Title = title
	return src, nil
}

// page fetches one page of games. Its More fetches the page after it.
func (l LichessLoader) page(ctx context.Context, username string, q lichess.Query) (Source, error) {
	q.Max = pageSize
	page, err := l.games(ctx, username, q)
	if err != nil {
		return Source{}, err
	}

	src := Source{Games: page.Games}
	if !page.Next.IsZero() {
		next := q
		next.Until = page.Next
		src.More = func(ctx context.Context) (Source, error) {
			return l.page(ctx, username, next)
		}
	}
	return src, nil
}
