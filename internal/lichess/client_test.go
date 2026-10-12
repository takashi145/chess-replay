package lichess

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/takashi145/chess-replay/internal/chesscom"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := New("test", nil)
	c.baseURL = srv.URL
	return c
}

const finishedGame = `{"id":"abc12345","rated":true,"variant":"standard","speed":"blitz",
	"createdAt":1699999000000,"lastMoveAt":1700000000000,"status":"mate","winner":"white",
	"players":{"white":{"user":{"name":"w"},"rating":1500},"black":{"user":{"name":"b"},"rating":1400}},
	"moves":"e4 e5","pgn":"1. e4 { [%clk 0:03:00] } 1... e5 { [%clk 0:03:00] }"}`

func TestGamesSendsTheQuery(t *testing.T) {
	var got *http.Request
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
	})
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	if _, err := c.Games(context.Background(), "some user", Query{Max: 5, Since: since, Until: until}); err != nil {
		t.Fatal(err)
	}

	if got.URL.EscapedPath() != "/api/games/user/some%20user" {
		t.Errorf("path = %q", got.URL.EscapedPath())
	}
	q := got.URL.Query()
	want := map[string]string{
		"clocks": "true", "pgnInJson": "true", "max": "5",
		"since": "1788220800000", "until": "1790812800000",
	}
	for k, v := range want {
		if q.Get(k) != v {
			t.Errorf("query %s = %q, want %q", k, q.Get(k), v)
		}
	}
	if got.Header.Get("Accept") != "application/x-ndjson" {
		t.Errorf("Accept = %q", got.Header.Get("Accept"))
	}
	if got.Header.Get("User-Agent") == "" {
		t.Error("User-Agent is empty")
	}
}

func TestGamesOmitsUnsetQueryValues(t *testing.T) {
	var got *http.Request
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { got = r })

	if _, err := c.Games(context.Background(), "u", Query{}); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"max", "since", "until"} {
		if got.URL.Query().Has(k) {
			t.Errorf("query has %s, want it omitted", k)
		}
	}
}

func TestGamesConvertsFields(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Lichess separates games with newlines and may send empty keep-alive lines.
		w.Write([]byte(finishedGame + "\n\n"))
	})

	page, err := c.Games(context.Background(), "w", Query{})
	games := page.Games
	if err != nil {
		t.Fatal(err)
	}

	if len(games) != 1 {
		t.Fatalf("got %d games, want 1", len(games))
	}
	g := games[0]
	if g.White.Username != "w" || g.Black.Username != "b" {
		t.Errorf("players = %q, %q", g.White.Username, g.Black.Username)
	}
	if g.White.Rating == nil || *g.White.Rating != 1500 {
		t.Errorf("white rating = %v", g.White.Rating)
	}
	if g.TimeClass != "blitz" || !g.Rated || g.Rules != "chess" {
		t.Errorf("TimeClass = %q, Rated = %v, Rules = %q", g.TimeClass, g.Rated, g.Rules)
	}
	if want := time.UnixMilli(1699999000000).UTC(); !g.PlayedAt.Equal(want) {
		t.Errorf("PlayedAt = %v, want %v", g.PlayedAt, want)
	}
	if want := c.baseURL + "/abc12345"; g.URL != want {
		t.Errorf("URL = %q, want %q", g.URL, want)
	}
	if g.PGN == "" {
		t.Error("PGN is empty")
	}
	if result, reason := chesscom.Describe(g.Player("w"), g.Opponent("w")); result != chesscom.Win || reason != "Checkmate" {
		t.Errorf("result = %v, %q", result, reason)
	}
}

func TestGamesKeepsNewestFirstOrder(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"new","status":"mate","winner":"white","variant":"standard","moves":"e4","pgn":"1. e4"}` + "\n" +
			`{"id":"old","status":"mate","winner":"white","variant":"standard","moves":"e4","pgn":"1. e4"}` + "\n"))
	})

	page, err := c.Games(context.Background(), "u", Query{})
	games := page.Games
	if err != nil {
		t.Fatal(err)
	}

	if len(games) != 2 || games[0].URL != c.baseURL+"/new" || games[1].URL != c.baseURL+"/old" {
		t.Errorf("games = %+v", games)
	}
}

func TestGamesKeepsGamesWithoutAResultOrMoves(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(
			`{"id":"aborted","status":"aborted","variant":"standard","moves":"e4","pgn":"1. e4"}` + "\n" +
				`{"id":"nomoves","status":"mate","winner":"white","variant":"standard","moves":"","pgn":"[Event \"x\"]"}` + "\n" +
				`{"id":"nowinner","status":"resign","variant":"standard","moves":"e4","pgn":"1. e4"}` + "\n" +
				finishedGame + "\n"))
	})

	page, err := c.Games(context.Background(), "w", Query{})
	games := page.Games
	if err != nil {
		t.Fatal(err)
	}

	if len(games) != 4 {
		t.Fatalf("games = %+v, want all four", games)
	}
	result, reason := chesscom.Describe(games[0].Player("a"), games[0].Opponent("a"))
	if result != chesscom.NoResult || reason != "Aborted" {
		t.Errorf("aborted game: got %v, %q", result, reason)
	}
}

func TestGamesMarksOtherVariantsAsUnsupported(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"z","status":"mate","winner":"white","variant":"crazyhouse","moves":"e4","pgn":"1. e4"}` + "\n"))
	})

	page, err := c.Games(context.Background(), "u", Query{})
	games := page.Games
	if err != nil {
		t.Fatal(err)
	}

	if len(games) != 1 || games[0].Rules != "crazyhouse" {
		t.Errorf("games = %+v, want one crazyhouse game", games)
	}
}

func TestGamesPlayersWithoutAnAccount(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"z","status":"mate","winner":"white","variant":"standard","moves":"e4","pgn":"1. e4",
			"players":{"white":{"user":{"name":"w"},"rating":1500},"black":{"aiLevel":3}}}` + "\n"))
	})

	page, err := c.Games(context.Background(), "w", Query{})
	games := page.Games
	if err != nil {
		t.Fatal(err)
	}

	if len(games) != 1 || games[0].Black.Username != "(anonymous)" || games[0].Black.Rating != nil {
		t.Errorf("games = %+v", games)
	}
}

func TestGamesWithNoGamesIsNotAnError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})

	page, err := c.Games(context.Background(), "u", Query{})
	games := page.Games

	if err != nil || len(games) != 0 {
		t.Errorf("games = %v, err = %v", games, err)
	}
}

func TestGamesNextPointsPastTheLastGameSent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":"a","status":"mate","winner":"white","variant":"standard","createdAt":300,"moves":"e4","pgn":"1. e4"}` + "\n" +
			`{"id":"b","status":"mate","winner":"black","variant":"standard","createdAt":200,"moves":"e4","pgn":"1. e4"}` + "\n"))
	})

	page, err := c.Games(context.Background(), "u", Query{Max: 2})

	if err != nil || len(page.Games) != 2 || !page.Next.Equal(time.UnixMilli(200)) {
		t.Errorf("page = %+v, err = %v, want two games and Next at 200ms", page, err)
	}
}

func TestGamesNextIsZeroOnTheLastPage(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(finishedGame + "\n"))
	})

	for name, q := range map[string]Query{"fewer than asked": {Max: 5}, "no limit": {}} {
		page, err := c.Games(context.Background(), "u", q)

		if err != nil || len(page.Games) != 1 || !page.Next.IsZero() {
			t.Errorf("%s: page = %+v, err = %v, want a zero Next", name, page, err)
		}
	}
}

func TestGamesUnknownPlayer(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })

	_, err := c.Games(context.Background(), "nobody", Query{})

	var notFound *chesscom.PlayerNotFoundError
	if !errors.As(err, &notFound) || notFound.Username != "nobody" || notFound.Site != "Lichess" {
		t.Errorf("err = %v, want PlayerNotFoundError for Lichess", err)
	}
}

func TestGamesHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   string
	}{
		{http.StatusTooManyRequests, "Lichess is limiting requests. Wait a minute and try again."},
		{http.StatusInternalServerError, "Lichess API returned 500 Internal Server Error."},
	} {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		})

		_, err := c.Games(context.Background(), "u", Query{})

		if err == nil || err.Error() != tc.want {
			t.Errorf("status %d: err = %v, want %q", tc.status, err, tc.want)
		}
	}
}

func TestGamesUnreadableResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	})

	_, err := c.Games(context.Background(), "u", Query{})

	if err == nil || err.Error() != "Lichess API returned an unreadable response." {
		t.Errorf("err = %v", err)
	}
}

func TestGamesConnectionError(t *testing.T) {
	c := New("test", nil)
	c.baseURL = "http://127.0.0.1:1" // Nothing listens here.

	_, err := c.Games(context.Background(), "u", Query{})

	var conn *chesscom.ConnectionError
	if !errors.As(err, &conn) {
		t.Errorf("err = %v, want ConnectionError", err)
	}
}

func TestGamesStalledWhileReadingTimesOut(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(finishedGame + "\n"))
		w.(http.Flusher).Flush()
		<-r.Context().Done() // Send nothing more.
	})
	c.httpClient.Timeout = 200 * time.Millisecond

	_, err := c.Games(context.Background(), "u", Query{})

	var conn *chesscom.ConnectionError
	if !errors.As(err, &conn) || conn.Message != "The request to Lichess timed out." {
		t.Errorf("err = %v, want a timed out ConnectionError", err)
	}
}

func TestGamesCancelledContextPropagates(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Games(ctx, "u", Query{})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestRawResultsAreUnderstoodByDescribe(t *testing.T) {
	cases := []struct {
		status, winner string
		self           string // the white player
		result         chesscom.Result
		reason         string
	}{
		{"mate", "white", "w", chesscom.Win, "Checkmate"},
		{"mate", "black", "w", chesscom.Loss, "Checkmate"},
		{"resign", "white", "w", chesscom.Win, "Resignation"},
		{"resign", "black", "w", chesscom.Loss, "Resignation"},
		{"outoftime", "white", "w", chesscom.Win, "Timeout"},
		{"timeout", "black", "w", chesscom.Loss, "Abandoned"},
		{"timeout", "white", "w", chesscom.Win, "Abandoned"},
		{"outoftime", "", "w", chesscom.Draw, "Timeout vs Insufficient Material"},
		{"aborted", "", "w", chesscom.NoResult, "Aborted"},
		{"noStart", "", "w", chesscom.NoResult, "Not started"},
		{"timeout", "", "w", chesscom.NoResult, "Unknown"},
		{"unknownFinish", "", "w", chesscom.NoResult, "Unknown"},
		{"draw", "", "w", chesscom.Draw, ""},
		{"stalemate", "", "w", chesscom.Draw, "Stalemate"},
		{"insufficientMaterialClaim", "", "w", chesscom.Draw, "Insufficient Material"},
		{"variantEnd", "white", "w", chesscom.Win, "Variant rule"},
	}
	for _, tc := range cases {
		white, black := rawResults(tc.status, tc.winner)
		result, reason := chesscom.Describe(
			chesscom.Player{Username: "w", RawResult: white},
			chesscom.Player{Username: "b", RawResult: black},
		)
		if result != tc.result || reason != tc.reason {
			t.Errorf("%s/%s: got %v, %q; want %v, %q", tc.status, tc.winner, result, reason, tc.result, tc.reason)
		}
	}
}
