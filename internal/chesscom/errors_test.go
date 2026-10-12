package chesscom

import "testing"

func TestPlayerNotFoundErrorNamesTheSite(t *testing.T) {
	err := &PlayerNotFoundError{Site: "Chess.com", Username: "bob"}
	want := `Player "bob" was not found on Chess.com.`
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
