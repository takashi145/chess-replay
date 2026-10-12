package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/takashi145/chess-replay/internal/chesscom"
	"github.com/takashi145/chess-replay/internal/lichess"
)

// fakeLichess serves games newest first, honouring Max and Until like the real API does.
// A Lichess game's PlayedAt is its start time, which is what the real API filters on.
type fakeLichess struct {
	games   []chesscom.Game
	queries []lichess.Query
}

func (f *fakeLichess) Games(_ context.Context, _ string, q lichess.Query) (lichess.Page, error) {
	f.queries = append(f.queries, q)

	var games []chesscom.Game
	for _, g := range f.games {
		if q.Until.IsZero() || g.PlayedAt.Before(q.Until) {
			games = append(games, g)
		}
	}

	var page lichess.Page
	if q.Max > 0 && len(games) >= q.Max {
		games = games[:q.Max]
		page.Next = games[len(games)-1].PlayedAt
	}
	page.Games = slices.Clone(games)
	return page, nil
}

// manyGames returns n standard games, newest first.
func manyGames(n int) []chesscom.Game {
	var games []chesscom.Game
	for i := range n {
		g := game(fmt.Sprintf("g%d", i), 0, "chess")
		g.PlayedAt = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC).Add(-time.Duration(i) * time.Minute)
		games = append(games, g)
	}
	return games
}

// nextPages follows More from src to the end and returns the URLs of every page, in order.
func nextPages(t *testing.T, src Source) [][]string {
	t.Helper()
	pages := [][]string{urls(src.Games)}
	for src.More != nil {
		var err error
		if src, err = src.More(context.Background()); err != nil {
			t.Fatal(err)
		}
		pages = append(pages, urls(src.Games))
	}
	return pages
}

var sep = chesscom.Archive{Year: 2026, Month: 9}

func TestLichessLatestPicksTheNewestGame(t *testing.T) {
	api := &fakeLichess{games: []chesscom.Game{game("960", 30, "chess960"), game("new", 20, "chess"), game("old", 1, "chess")}}

	src, err := LichessLoader{API: api}.Latest(context.Background(), "u")

	if err != nil || !src.Single || !slices.Equal(urls(src.Games), []string{"960"}) {
		t.Fatalf("src = %+v, err = %v", src, err)
	}
}

func TestLichessNoGames(t *testing.T) {
	loader := LichessLoader{API: &fakeLichess{}}
	ctx := context.Background()

	for name, err := range map[string]error{
		"latest": second(loader.Latest(ctx, "u")),
		"month":  second(loader.Month(ctx, "u", sep)),
	} {
		var noGames *chesscom.NoGamesError
		if !errors.As(err, &noGames) || noGames.Username != "u" {
			t.Errorf("%s: err = %v, want NoGamesError", name, err)
		}
	}
}

func second(_ Source, err error) error { return err }

func TestLichessMonthErrorNamesTheMonth(t *testing.T) {
	_, err := LichessLoader{API: &fakeLichess{}}.Month(context.Background(), "u", sep)

	var noGames *chesscom.NoGamesError
	if !errors.As(err, &noGames) || noGames.Archive == nil || *noGames.Archive != sep {
		t.Errorf("err = %v, want NoGamesError for 2026-09", err)
	}
}

func TestLichessMonthAsksForThatUTCMonth(t *testing.T) {
	api := &fakeLichess{games: []chesscom.Game{game("b", 20, "chess"), game("a", 10, "chess"), game("960", 1, "chess960")}}

	src, err := LichessLoader{API: api}.Month(context.Background(), "u", sep)

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(urls(src.Games), []string{"b", "a", "960"}) || src.Title != "u — Games in 2026-09 (UTC)" {
		t.Errorf("games = %v, title = %q", urls(src.Games), src.Title)
	}
	q := api.queries[0]
	if !q.Since.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) || !q.Until.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("since = %v, until = %v", q.Since, q.Until)
	}
}

func TestLichessMonthPagesThroughTheGames(t *testing.T) {
	api := &fakeLichess{games: manyGames(2*pageSize + 10)}

	src, err := LichessLoader{API: api}.Month(context.Background(), "u", sep)

	if err != nil {
		t.Fatal(err)
	}
	pages := nextPages(t, src)
	if len(pages) != 3 || len(pages[0]) != pageSize || len(pages[1]) != pageSize || len(pages[2]) != 10 {
		t.Fatalf("got %d pages, want %d, %d and 10 games", len(pages), pageSize, pageSize)
	}
	if pages[0][0] != "g0" || pages[1][0] != fmt.Sprintf("g%d", pageSize) || pages[2][9] != fmt.Sprintf("g%d", 2*pageSize+9) {
		t.Errorf("pages do not follow on from each other: %v %v %v", pages[0][0], pages[1][0], pages[2][9])
	}
	firstQuery, nextQuery := api.queries[0], api.queries[1]
	if !nextQuery.Since.Equal(firstQuery.Since) || !nextQuery.Until.Equal(api.games[pageSize-1].PlayedAt) {
		t.Errorf("next query = %+v, want the same month, ending at the last game of the first page", nextQuery)
	}
}
