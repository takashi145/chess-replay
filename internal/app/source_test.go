package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/takashi145/chess-replay/internal/chesscom"
)

type fakeAPI struct {
	archives []chesscom.Archive
	games    map[chesscom.Archive][]chesscom.Game
	fetched  []chesscom.Archive
}

func (f *fakeAPI) Archives(context.Context, string) ([]chesscom.Archive, error) {
	return f.archives, nil
}

func (f *fakeAPI) Games(_ context.Context, _ string, a chesscom.Archive) ([]chesscom.Game, error) {
	f.fetched = append(f.fetched, a)
	return f.games[a], nil
}

func game(id string, minute int, rules string) chesscom.Game {
	return chesscom.Game{
		URL:     id,
		Rules:   rules,
		EndTime: time.Date(2026, 8, 1, 12, minute, 0, 0, time.UTC),
	}
}

func urls(games []chesscom.Game) []string {
	var out []string
	for _, g := range games {
		out = append(out, g.URL)
	}
	return out
}

var (
	jul = chesscom.Archive{Year: 2026, Month: 7}
	aug = chesscom.Archive{Year: 2026, Month: 8}
)

func TestLatestPicksTheNewestPlayableGame(t *testing.T) {
	api := &fakeAPI{
		archives: []chesscom.Archive{jul, aug},
		games: map[chesscom.Archive][]chesscom.Game{
			aug: {game("old", 1, "chess"), game("960", 30, "chess960"), game("new", 20, "chess")},
			jul: {game("july", 59, "chess")},
		},
	}

	src, err := Loader{API: api}.Latest(context.Background(), "u")

	if err != nil || !src.Single || !slices.Equal(urls(src.Games), []string{"new"}) {
		t.Fatalf("src = %+v, err = %v", src, err)
	}
	if len(api.fetched) != 1 {
		t.Errorf("fetched %v, want only the newest month", api.fetched)
	}
}

func TestLatestFallsBackToAnOlderMonth(t *testing.T) {
	api := &fakeAPI{
		archives: []chesscom.Archive{jul, aug},
		games: map[chesscom.Archive][]chesscom.Game{
			aug: {game("960", 30, "chess960")},
			jul: {game("july", 5, "chess")},
		},
	}

	src, err := Loader{API: api}.Latest(context.Background(), "u")

	if err != nil || !slices.Equal(urls(src.Games), []string{"july"}) {
		t.Fatalf("src = %+v, err = %v", src, err)
	}
}

func TestLatestWithoutArchivesIsNoGames(t *testing.T) {
	_, err := Loader{API: &fakeAPI{}}.Latest(context.Background(), "u")

	var noGames *chesscom.NoGamesError
	if !errors.As(err, &noGames) {
		t.Errorf("err = %v, want NoGamesError", err)
	}
}

func TestRandomReturnsAPlayableGame(t *testing.T) {
	api := &fakeAPI{
		archives: []chesscom.Archive{jul, aug},
		games: map[chesscom.Archive][]chesscom.Game{
			aug: {game("960", 1, "chess960")},
			jul: {game("ok", 2, "chess")},
		},
	}

	for range 20 {
		src, err := Loader{API: api}.Random(context.Background(), "u")
		if err != nil || !src.Single || !slices.Equal(urls(src.Games), []string{"ok"}) {
			t.Fatalf("src = %+v, err = %v", src, err)
		}
	}
}

func TestRandomLooksAtAtMostSixMonths(t *testing.T) {
	api := &fakeAPI{games: map[chesscom.Archive][]chesscom.Game{}}
	for m := 1; m <= 12; m++ {
		api.archives = append(api.archives, chesscom.Archive{Year: 2025, Month: m})
	}

	_, err := Loader{API: api}.Random(context.Background(), "u")

	var noGames *chesscom.NoGamesError
	if !errors.As(err, &noGames) {
		t.Errorf("err = %v, want NoGamesError", err)
	}
	if len(api.fetched) != maxRandomArchives {
		t.Errorf("fetched %d months, want %d", len(api.fetched), maxRandomArchives)
	}
}

func TestLastListsNewestFirstAndCountsHiddenChess960(t *testing.T) {
	api := &fakeAPI{
		archives: []chesscom.Archive{jul, aug},
		games: map[chesscom.Archive][]chesscom.Game{
			aug: {game("a", 10, "chess"), game("b", 20, "chess"), game("960", 30, "chess960")},
			jul: {game("j", 1, "chess")},
		},
	}

	src, err := Loader{API: api}.Last(context.Background(), "u", 2)

	if err != nil || src.Single {
		t.Fatalf("src = %+v, err = %v", src, err)
	}
	if !slices.Equal(urls(src.Games), []string{"b", "a"}) || src.Hidden != 1 {
		t.Errorf("games = %v, hidden = %d", urls(src.Games), src.Hidden)
	}
	if len(api.fetched) != 1 {
		t.Errorf("fetched %v, want it to stop once enough games were found", api.fetched)
	}
}

func TestLastCrossesMonthBoundaries(t *testing.T) {
	api := &fakeAPI{
		archives: []chesscom.Archive{jul, aug},
		games: map[chesscom.Archive][]chesscom.Game{
			aug: {game("a", 10, "chess")},
			jul: {game("j", 1, "chess")},
		},
	}

	src, err := Loader{API: api}.Last(context.Background(), "u", 5)

	if err != nil || !slices.Equal(urls(src.Games), []string{"a", "j"}) {
		t.Fatalf("src = %+v, err = %v", src, err)
	}
}

func TestLastWithOnlyChess960Games(t *testing.T) {
	api := &fakeAPI{
		archives: []chesscom.Archive{aug},
		games:    map[chesscom.Archive][]chesscom.Game{aug: {game("960", 1, "chess960")}},
	}

	_, err := Loader{API: api}.Last(context.Background(), "u", 5)

	var all *AllChess960Error
	if !errors.As(err, &all) || all.Count != 1 {
		t.Errorf("err = %v, want AllChess960Error", err)
	}
}

func TestMonthListsPlayableGames(t *testing.T) {
	api := &fakeAPI{games: map[chesscom.Archive][]chesscom.Game{
		aug: {game("a", 10, "chess"), game("960", 20, "chess960"), game("b", 30, "chess")},
	}}

	src, err := Loader{API: api}.Month(context.Background(), "u", aug)

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(urls(src.Games), []string{"b", "a"}) || src.Hidden != 1 || src.Title != "u — Games in 2026-08 (UTC)" {
		t.Errorf("src = %+v", src)
	}
}

func TestMonthWithoutGames(t *testing.T) {
	_, err := Loader{API: &fakeAPI{}}.Month(context.Background(), "u", aug)

	var noGames *chesscom.NoGamesError
	if !errors.As(err, &noGames) || noGames.Archive == nil || *noGames.Archive != aug {
		t.Errorf("err = %v, want NoGamesError for the month", err)
	}
}

func TestMonthWithOnlyChess960Games(t *testing.T) {
	api := &fakeAPI{games: map[chesscom.Archive][]chesscom.Game{aug: {game("960", 1, "chess960"), game("960b", 2, "chess960")}}}

	_, err := Loader{API: api}.Month(context.Background(), "u", aug)

	var all *AllChess960Error
	if !errors.As(err, &all) || all.Count != 2 {
		t.Errorf("err = %v, want AllChess960Error", err)
	}
}
