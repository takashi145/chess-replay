// Package lichess is a client for the Lichess API. It returns the same game types as the chesscom package.
package lichess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/takashi145/chess-replay/internal/chesscom"
)

const (
	defaultBaseURL = "https://lichess.org"
	siteName       = "Lichess"
)

type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
	logf       func(format string, args ...any)
}

// New returns a client for the Lichess API.
func New(version string, logf func(format string, args ...any)) *Client {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Client{
		baseURL:    defaultBaseURL,
		userAgent:  fmt.Sprintf("chess-replay/%s (+https://github.com/takashi145/chess-replay)", version),
		httpClient: &http.Client{Timeout: 60 * time.Second},
		logf:       logf,
	}
}

// Query selects which games Games returns, newest first. Zero values mean no limit.
type Query struct {
	Max          int
	Since, Until time.Time
}

type gameDTO struct {
	ID        string `json:"id"`
	Rated     bool   `json:"rated"`
	Variant   string `json:"variant"`
	Speed     string `json:"speed"`
	CreatedAt int64  `json:"createdAt"`
	Status    string `json:"status"`
	Players   struct {
		White playerDTO `json:"white"`
		Black playerDTO `json:"black"`
	} `json:"players"`
	Winner string `json:"winner"`
	Moves  string `json:"moves"`
	PGN    string `json:"pgn"`
}

type playerDTO struct {
	User *struct {
		Name string `json:"name"`
	} `json:"user"`
	Rating *int `json:"rating"`
}

type Page struct {
	Games []chesscom.Game
	// Next is the Until value that fetches the games after this page. It is the zero time
	// when Lichess sent fewer games than asked for, so this page was the last.
	Next time.Time
}

// Games returns the player's finished games, newest first, with their clocks in the PGN.
// Every game is returned, whether or not it can be replayed.
func (c *Client) Games(ctx context.Context, username string, q Query) (Page, error) {
	params := url.Values{}
	params.Set("clocks", "true")
	params.Set("pgnInJson", "true")
	if q.Max > 0 {
		params.Set("max", strconv.Itoa(q.Max))
	}
	if !q.Since.IsZero() {
		params.Set("since", strconv.FormatInt(q.Since.UnixMilli(), 10))
	}
	if !q.Until.IsZero() {
		params.Set("until", strconv.FormatInt(q.Until.UnixMilli(), 10))
	}
	u := fmt.Sprintf("%s/api/games/user/%s?%s", c.baseURL, url.PathEscape(username), params.Encode())

	body, err := c.get(ctx, u, username)
	if err != nil {
		return Page{}, err
	}
	defer body.Close()

	var page Page
	sent := 0
	dec := json.NewDecoder(body)
	for {
		var dto gameDTO
		if err := dec.Decode(&dto); err != nil {
			if errors.Is(err, io.EOF) {
				return page, nil
			}
			if ctx.Err() != nil {
				return Page{}, ctx.Err()
			}
			if isTimeout(err) {
				return Page{}, &chesscom.ConnectionError{Message: "The request to Lichess timed out.", Err: err}
			}
			return Page{}, errors.New("Lichess API returned an unreadable response.")
		}

		sent++
		if q.Max > 0 && sent == q.Max {
			page.Next = time.UnixMilli(dto.CreatedAt).UTC()
		}
		page.Games = append(page.Games, dto.toGame(c.baseURL))
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (c *Client) get(ctx context.Context, rawURL, username string) (io.ReadCloser, error) {
	c.logf("GET %s", rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/x-ndjson")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if isTimeout(err) {
			return nil, &chesscom.ConnectionError{Message: "The request to Lichess timed out.", Err: err}
		}
		return nil, &chesscom.ConnectionError{Message: "Could not connect to Lichess.", Err: err}
	}

	c.logf("HTTP %s", resp.Status)

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return resp.Body, nil
	}
	resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, &chesscom.PlayerNotFoundError{Site: siteName, Username: username}
	case http.StatusTooManyRequests:
		return nil, errors.New("Lichess is limiting requests. Wait a minute and try again.")
	}
	return nil, fmt.Errorf("Lichess API returned %s.", resp.Status)
}

func (d gameDTO) toGame(baseURL string) chesscom.Game {
	white, black := rawResults(d.Status, d.Winner)

	rules := "chess"
	if d.Variant != "standard" {
		rules = d.Variant
	}
	return chesscom.Game{
		PGN:       d.PGN,
		PlayedAt:  time.UnixMilli(d.CreatedAt).UTC(),
		TimeClass: d.Speed,
		Rated:     d.Rated,
		White:     chesscom.Player{Username: d.Players.White.name(), Rating: d.Players.White.Rating, RawResult: white},
		Black:     chesscom.Player{Username: d.Players.Black.name(), Rating: d.Players.Black.Rating, RawResult: black},
		URL:       baseURL + "/" + d.ID,
		Rules:     rules,
	}
}

// noAccountName stands in for players without an account.
const noAccountName = "(anonymous)"

func (p playerDTO) name() string {
	if p.User == nil || p.User.Name == "" {
		return noAccountName
	}
	return p.User.Name
}

// rawResults expresses how a game ended in the result words chesscom.Describe understands.
// A game with no winner that is not a draw has no result; "aborted" is kept, the rest become "unknownFinish".
func rawResults(status, winner string) (white, black string) {
	switch status {
	case "draw":
		return "draw", "draw"
	case "stalemate":
		return "stalemate", "stalemate"
	case "insufficientMaterialClaim":
		return "insufficient", "insufficient"
	}
	// A player ran out of time but the other side could not have won: Lichess scores it as a draw.
	if status == "outoftime" && winner == "" {
		return "timevsinsufficient", "timevsinsufficient"
	}

	loser := status
	switch status {
	case "mate":
		loser = "checkmated"
	case "resign":
		loser = "resigned"
	case "outoftime":
		loser = "timeout"
	case "timeout":
		loser = "abandoned"
	}

	switch winner {
	case "white":
		return "win", loser
	case "black":
		return loser, "win"
	}
	if status == "aborted" {
		return status, status
	}
	return "unknownFinish", "unknownFinish"
}
