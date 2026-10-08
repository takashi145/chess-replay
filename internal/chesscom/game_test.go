package chesscom

import "testing"

const (
	whiteUser = "test-white-user"
	blackUser = "test-black-user"
)

func makeGame(rules string) Game {
	return Game{
		White: Player{Username: whiteUser},
		Black: Player{Username: blackUser},
		Rules: rules,
	}
}

func TestIsChess960(t *testing.T) {
	cases := map[string]bool{"chess960": true, "CHESS960": true, "chess": false, "bughouse": false}
	for rules, want := range cases {
		if got := makeGame(rules).IsChess960(); got != want {
			t.Errorf("IsChess960(%q) = %v, want %v", rules, got, want)
		}
	}
}

func TestPlayerMatchesUsernameCaseInsensitively(t *testing.T) {
	g := makeGame("chess")

	for _, name := range []string{"test-white-user", "TEST-WHITE-USER", "Test-White-User"} {
		if got := g.Player(name).Username; got != whiteUser {
			t.Errorf("Player(%q) = %q, want white", name, got)
		}
	}
	if got := g.Player(blackUser).Username; got != blackUser {
		t.Errorf("Player(black) = %q", got)
	}
}

func TestOpponentIsTheOtherPlayer(t *testing.T) {
	g := makeGame("chess")

	if got := g.Opponent(whiteUser).Username; got != blackUser {
		t.Errorf("Opponent(white) = %q", got)
	}
	if got := g.Opponent(blackUser).Username; got != whiteUser {
		t.Errorf("Opponent(black) = %q", got)
	}
}

func TestParseArchive(t *testing.T) {
	got, ok := ParseArchive("2026-08")
	if !ok || got != (Archive{2026, 8}) {
		t.Errorf("ParseArchive(2026-08) = %v, %v", got, ok)
	}

	invalid := []string{
		"26-08",    // year not 4 digits
		"2026-13",  // month out of range
		"2026-00",  // month out of range
		"",         //
		"2026",     // no separator
		"yyyy-mm",  // non-numeric
		"2026-8-1", // too many parts
		"+026-08",  // sign
		"2026--8",  // sign
		"1899-12",  // year out of range
	}
	for _, value := range invalid {
		if got, ok := ParseArchive(value); ok {
			t.Errorf("ParseArchive(%q) = %v, want failure", value, got)
		}
	}
}

func TestParseArchiveURL(t *testing.T) {
	urls := []string{
		"https://api.chess.com/pub/player/test-user/games/2026/08",
		"https://api.chess.com/pub/player/test-user/games/2026/08/",
	}
	for _, u := range urls {
		got, ok := ParseArchiveURL(u)
		if !ok || got != (Archive{2026, 8}) {
			t.Errorf("ParseArchiveURL(%q) = %v, %v", u, got, ok)
		}
	}

	if got, ok := ParseArchiveURL("2026"); ok {
		t.Errorf("ParseArchiveURL(2026) = %v, want failure", got)
	}
}

func TestArchiveStringPadsYearAndMonth(t *testing.T) {
	if got := (Archive{7, 1}).String(); got != "0007-01" {
		t.Errorf("String() = %q", got)
	}
}
