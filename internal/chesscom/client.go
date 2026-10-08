// Package chesscom is a client for the Chess.com public API, and the types it returns.
package chesscom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.chess.com/pub"

type Client struct {
	baseURL    string
	userAgent  string
	httpClient *http.Client
	logf       func(format string, args ...any)
}

// New returns a client for the Chess.com public API. logf receives diagnostic messages and may be nil.
func New(version string, logf func(format string, args ...any)) *Client {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Client{
		baseURL:    defaultBaseURL,
		userAgent:  fmt.Sprintf("chess-replay/%s (+https://github.com/takashi145/chess-replay)", version),
		httpClient: &http.Client{Timeout: 20 * time.Second},
		logf:       logf,
	}
}

type archivesResponse struct {
	Archives []string `json:"archives"`
}

type gamesResponse struct {
	Games []gameDTO `json:"games"`
}

type gameDTO struct {
	URL       string    `json:"url"`
	PGN       string    `json:"pgn"`
	EndTime   int64     `json:"end_time"`
	Rated     bool      `json:"rated"`
	TimeClass string    `json:"time_class"`
	Rules     string    `json:"rules"`
	White     playerDTO `json:"white"`
	Black     playerDTO `json:"black"`
}

type playerDTO struct {
	Username string `json:"username"`
	Rating   *int   `json:"rating"`
	Result   string `json:"result"`
}

// Archives returns the months in which the player has games, oldest first.
func (c *Client) Archives(ctx context.Context, username string) ([]Archive, error) {
	var resp archivesResponse
	notFound := &PlayerNotFoundError{Username: username}
	if err := c.get(ctx, fmt.Sprintf("%s/player/%s/games/archives", c.baseURL, url.PathEscape(username)), notFound, &resp); err != nil {
		return nil, err
	}

	var archives []Archive
	for _, u := range resp.Archives {
		if a, ok := ParseArchiveURL(u); ok {
			archives = append(archives, a)
		} else {
			c.logf("Could not parse archive URL: %s", u)
		}
	}

	slices.SortFunc(archives, func(a, b Archive) int {
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		return a.Month - b.Month
	})
	return archives, nil
}

func (c *Client) Games(ctx context.Context, username string, archive Archive) ([]Game, error) {
	var resp gamesResponse
	u := fmt.Sprintf("%s/player/%s/games/%04d/%02d", c.baseURL, url.PathEscape(username), archive.Year, archive.Month)
	notFound := &NoGamesError{Username: username, Archive: &archive}
	if err := c.get(ctx, u, notFound, &resp); err != nil {
		var noGames *NoGamesError
		if errors.As(err, &noGames) {
			// Tell a missing player apart from a month without games.
			if _, archivesErr := c.Archives(ctx, username); archivesErr != nil {
				return nil, archivesErr
			}
		}
		return nil, err
	}

	var games []Game
	for _, dto := range resp.Games {
		if strings.TrimSpace(dto.PGN) == "" {
			continue
		}
		games = append(games, dto.toGame())
	}
	return games, nil
}

func (c *Client) get(ctx context.Context, rawURL string, onNotFound error, out any) error {
	c.logf("GET %s", rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err() // The caller cancelled (e.g. Ctrl+C); let it propagate as-is.
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return &ConnectionError{Message: "The request to Chess.com timed out.", Err: err}
		}
		return &ConnectionError{Message: "Could not connect to Chess.com.", Err: err}
	}
	defer resp.Body.Close()

	c.logf("HTTP %s", resp.Status)

	if resp.StatusCode == http.StatusNotFound {
		return onNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Chess.com API returned %s.", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("Chess.com API returned an unreadable response.")
	}
	return nil
}

func (d gameDTO) toGame() Game {
	rules := d.Rules
	if rules == "" {
		rules = "chess"
	}
	return Game{
		PGN:       d.PGN,
		EndTime:   time.Unix(d.EndTime, 0).UTC(),
		TimeClass: d.TimeClass,
		Rated:     d.Rated,
		White:     Player{Username: d.White.Username, Rating: d.White.Rating, RawResult: d.White.Result},
		Black:     Player{Username: d.Black.Username, Rating: d.Black.Rating, RawResult: d.Black.Result},
		URL:       d.URL,
		Rules:     rules,
	}
}
