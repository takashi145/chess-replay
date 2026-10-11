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

// fakeLichess serves games newest first, honouring Max like the real API does.
type fakeLichess struct {
	games   []chesscom.Game
	queries []lichess.Query
}

func (f *fakeLichess) Games(_ context.Context, _ string, q lichess.Query) ([]chesscom.Game, error) {
	f.queries = append(f.queries, q)
	games := f.games
	if q.Max > 0 && len(games) > q.Max {
		games = games[:q.Max]
	}
	return slices.Clone(games), nil
}

// manyGames returns n standard games, newest first.
func manyGames(n int) []chesscom.Game {
	var games []chesscom.Game
	for i := range n {
		g := game(fmt.Sprintf("g%d", i), 0, "chess")
		g.EndTime = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC).Add(-time.Duration(i) * time.Minute)
		games = append(games, g)
	}
	return games
}

var sep = chesscom.Archive{Year: 2026, Month: 9}

func TestLichessLatestPicksTheNewestPlayableGame(t *testing.T) {
	api := &fakeLichess{games: []chesscom.Game{game("960", 30, "chess960"), game("new", 20, "chess"), game("old", 1, "chess")}}

	src, err := LichessLoader{API: api}.Latest(context.Background(), "u")

	if err != nil || !src.Single || !slices.Equal(urls(src.Games), []string{"new"}) {
		t.Fatalf("src = %+v, err = %v", src, err)
	}
}

func TestLichessNoGames(t *testing.T) {
	loader := LichessLoader{API: &fakeLichess{}}
	ctx := context.Background()

	for name, err := range map[string]error{
		"latest": second(loader.Latest(ctx, "u")),
		"random": second(loader.Random(ctx, "u")),
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

func TestLichessAllChess960(t *testing.T) {
	loader := LichessLoader{API: &fakeLichess{games: []chesscom.Game{game("a", 1, "chess960"), game("b", 2, "chess960")}}}
	ctx := context.Background()

	for name, err := range map[string]error{
		"latest": second(loader.Latest(ctx, "u")),
		"random": second(loader.Random(ctx, "u")),
		"month":  second(loader.Month(ctx, "u", sep)),
	} {
		var all *AllChess960Error
		if !errors.As(err, &all) || all.Count != 2 {
			t.Errorf("%s: err = %v, want AllChess960Error for 2 games", name, err)
		}
	}
}

func TestLichessRandomPicksFromTheRecentGames(t *testing.T) {
	api := &fakeLichess{games: []chesscom.Game{game("960", 30, "chess960"), game("a", 20, "chess"), game("b", 1, "chess")}}

	for range 20 {
		src, err := LichessLoader{API: api}.Random(context.Background(), "u")
		if err != nil || !src.Single || len(src.Games) != 1 || src.Games[0].IsChess960() {
			t.Fatalf("src = %+v, err = %v", src, err)
		}
	}
}

func TestLichessMonthAsksForThatUTCMonth(t *testing.T) {
	api := &fakeLichess{games: []chesscom.Game{game("b", 20, "chess"), game("a", 10, "chess"), game("960", 1, "chess960")}}

	src, err := LichessLoader{API: api}.Month(context.Background(), "u", sep)

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(urls(src.Games), []string{"b", "a"}) || src.Hidden != 1 || src.Title != "u — Games in 2026-09 (UTC)" {
		t.Errorf("games = %v, hidden = %d, title = %q", urls(src.Games), src.Hidden, src.Title)
	}
	q := api.queries[0]
	if !q.Since.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) || !q.Until.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("since = %v, until = %v", q.Since, q.Until)
	}
}

func TestLichessMonthAtTheLimitIsNotMarkedAsCut(t *testing.T) {
	api := &fakeLichess{games: manyGames(maxListedGames)}

	src, err := LichessLoader{API: api}.Month(context.Background(), "u", sep)

	if err != nil || len(src.Games) != maxListedGames || src.Title != "u — Games in 2026-09 (UTC)" {
		t.Errorf("games = %d, title = %q, err = %v", len(src.Games), src.Title, err)
	}
}

func TestLichessMonthOverTheLimitShowsTheNewestAndSaysSo(t *testing.T) {
	api := &fakeLichess{games: manyGames(maxListedGames + 200)}

	src, err := LichessLoader{API: api}.Month(context.Background(), "u", sep)

	if err != nil || len(src.Games) != maxListedGames {
		t.Fatalf("games = %d, err = %v, want %d games", len(src.Games), err, maxListedGames)
	}
	if want := fmt.Sprintf("u — Games in 2026-09 (UTC) — newest %d", maxListedGames); src.Title != want {
		t.Errorf("title = %q, want %q", src.Title, want)
	}
	if src.Games[0].URL != "g0" {
		t.Errorf("first game = %q, want the newest", src.Games[0].URL)
	}
}

func TestLichessRandomInMonth(t *testing.T) {
	api := &fakeLichess{games: []chesscom.Game{game("b", 20, "chess"), game("960", 10, "chess960"), game("a", 1, "chess")}}

	for range 20 {
		src, err := LichessLoader{API: api}.RandomInMonth(context.Background(), "u", sep)
		if err != nil || !src.Single || len(src.Games) != 1 || src.Games[0].IsChess960() {
			t.Fatalf("src = %+v, err = %v", src, err)
		}
	}
}
