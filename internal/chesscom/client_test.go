package chesscom

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := New("test", nil)
	c.baseURL = srv.URL
	return c
}

func TestArchivesAreSortedOldestFirst(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"archives":[
			"https://api.chess.com/pub/player/u/games/2026/02",
			"https://api.chess.com/pub/player/u/games/2025/12",
			"not-an-archive",
			"https://api.chess.com/pub/player/u/games/2026/01"]}`))
	})

	got, err := c.Archives(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}

	want := []Archive{{2025, 12}, {2026, 1}, {2026, 2}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("archives[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestArchivesUnknownPlayer(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	_, err := c.Archives(context.Background(), "nobody")

	var notFound *PlayerNotFoundError
	if !errors.As(err, &notFound) || notFound.Username != "nobody" {
		t.Errorf("err = %v, want PlayerNotFoundError", err)
	}
}

func TestGamesSkipsGamesWithoutPGNAndConvertsFields(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"games":[
			{"url":"u1","pgn":"1. e4 e5","time_control":"600","end_time":1700000000,"rated":true,
			 "time_class":"rapid","rules":"chess",
			 "white":{"username":"w","rating":1500,"result":"win"},
			 "black":{"username":"b","result":"checkmated"}},
			{"url":"u2","pgn":"  "}]}`))
	})

	games, err := c.Games(context.Background(), "w", Archive{2023, 11})
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 {
		t.Fatalf("got %d games, want 1", len(games))
	}

	g := games[0]
	if g.White.Rating == nil || *g.White.Rating != 1500 || g.Black.Rating != nil {
		t.Errorf("ratings = %v / %v", g.White.Rating, g.Black.Rating)
	}
	if g.EndTime.Unix() != 1700000000 || g.EndTime.Location().String() != "UTC" {
		t.Errorf("EndTime = %v", g.EndTime)
	}
	if g.TimeClass != "rapid" || !g.Rated || g.URL != "u1" {
		t.Errorf("game = %+v", g)
	}
}

func TestGamesMissingMonthForExistingPlayer(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/player/u/games/archives" {
			w.Write([]byte(`{"archives":[]}`))
			return
		}
		http.NotFound(w, r)
	})

	_, err := c.Games(context.Background(), "u", Archive{2026, 8})

	var noGames *NoGamesError
	if !errors.As(err, &noGames) || noGames.Archive == nil || *noGames.Archive != (Archive{2026, 8}) {
		t.Errorf("err = %v, want NoGamesError for 2026-08", err)
	}
}

func TestGamesMissingPlayerIsReportedAsPlayerNotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	_, err := c.Games(context.Background(), "nobody", Archive{2026, 8})

	var notFound *PlayerNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("err = %v, want PlayerNotFoundError", err)
	}
}

func TestServerErrorMentionsTheStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	_, err := c.Archives(context.Background(), "u")

	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("err = %v, want an error mentioning 500", err)
	}
}

func TestUnreachableServerIsAConnectionError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	c.baseURL = closedServerURL(t)

	_, err := c.Archives(context.Background(), "u")

	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Errorf("err = %v, want ConnectionError", err)
	}
}

func closedServerURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	return srv.URL
}

func TestCancelledContextIsNotAConnectionError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Archives(ctx, "u")

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
