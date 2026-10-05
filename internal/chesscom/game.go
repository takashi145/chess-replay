package chesscom

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Player struct {
	Username  string
	Rating    *int
	RawResult string
}

type Game struct {
	PGN       string
	EndTime   time.Time
	TimeClass string
	Rated     bool
	White     Player
	Black     Player
	URL       string
	Rules     string
}

func (g Game) IsChess960() bool { return strings.EqualFold(g.Rules, "chess960") }

func (g Game) Player(username string) Player {
	if g.isWhite(username) {
		return g.White
	}
	return g.Black
}

func (g Game) Opponent(username string) Player {
	if g.isWhite(username) {
		return g.Black
	}
	return g.White
}

func (g Game) isWhite(username string) bool {
	return strings.EqualFold(g.White.Username, username)
}

type Archive struct {
	Year, Month int
}

func (a Archive) String() string { return fmt.Sprintf("%04d-%02d", a.Year, a.Month) }

// ParseArchiveURL reads Chess.com archive URLs, which look like ".../games/2026/08".
func ParseArchiveURL(url string) (Archive, bool) {
	segments := strings.Split(strings.TrimRight(url, "/"), "/")
	if len(segments) < 2 {
		return Archive{}, false
	}
	return ParseArchive(segments[len(segments)-2] + "-" + segments[len(segments)-1])
}

// ParseArchive reads user input in the form "YYYY-MM".
func ParseArchive(value string) (Archive, bool) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 || len(parts[0]) != 4 {
		return Archive{}, false
	}

	year, ok := parseDigits(parts[0])
	if !ok {
		return Archive{}, false
	}
	month, ok := parseDigits(parts[1])
	if !ok {
		return Archive{}, false
	}

	if year < 1900 || month < 1 || month > 12 {
		return Archive{}, false
	}
	return Archive{Year: year, Month: month}, true
}

// Unlike strconv.Atoi alone, this rejects signs.
func parseDigits(s string) (int, bool) {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}
