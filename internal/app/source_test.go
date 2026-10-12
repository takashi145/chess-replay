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
		URL:      id,
		Rules:    rules,
		PlayedAt: time.Date(2026, 8, 1, 12, minute, 0, 0, time.UTC),
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

func TestLatestPicksTheNewestGame(t *testing.T) {
	api := &fakeAPI{
		archives: []chesscom.Archive{jul, aug},
		games: map[chesscom.Archive][]chesscom.Game{
			aug: {game("old", 1, "chess"), game("960", 30, "chess960"), game("new", 20, "chess")},
			jul: {game("july", 59, "chess")},
		},
	}

	src, err := Loader{API: api}.Latest(context.Background(), "u")

	if err != nil || !src.Single || !slices.Equal(urls(src.Games), []string{"960"}) {
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
			aug: {},
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

func TestMonthListsAllGamesNewestFirst(t *testing.T) {
	api := &fakeAPI{games: map[chesscom.Archive][]chesscom.Game{
		aug: {game("a", 10, "chess"), game("960", 20, "chess960"), game("b", 30, "chess")},
	}}

	src, err := Loader{API: api}.Month(context.Background(), "u", aug)

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(urls(src.Games), []string{"b", "960", "a"}) || src.Title != "u — Games in 2026-08 (UTC)" {
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
