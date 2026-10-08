package chesscom

import "testing"

func player(raw string) Player { return Player{Username: "player", RawResult: raw} }

func TestDescribeWinnerReadsReasonFromOpponent(t *testing.T) {
	// Chess.com always reports the winner's own result as "win", so the
	// actual reason (checkmate, timeout, ...) has to be read off the opponent.
	result, reason := Describe(player("win"), player("checkmated"))

	if result != Win || reason != "Checkmate" {
		t.Errorf("got %v, %q", result, reason)
	}
}

func TestDescribeDrawVariants(t *testing.T) {
	for _, raw := range []string{"agreed", "repetition", "stalemate", "insufficient", "50move", "timevsinsufficient"} {
		if result, _ := Describe(player(raw), player(raw)); result != Draw {
			t.Errorf("Describe(%q) = %v, want Draw", raw, result)
		}
	}
}

func TestDescribeLoserReadsOwnReason(t *testing.T) {
	result, reason := Describe(player("timeout"), player("win"))

	if result != Loss || reason != "Timeout" {
		t.Errorf("got %v, %q", result, reason)
	}
}

func TestDescribeKnownReasons(t *testing.T) {
	cases := map[string]string{
		"checkmated":         "Checkmate",
		"timeout":            "Timeout",
		"resigned":           "Resignation",
		"repetition":         "Repetition",
		"stalemate":          "Stalemate",
		"insufficient":       "Insufficient Material",
		"50move":             "50-move Rule",
		"agreed":             "Agreement",
		"abandoned":          "Abandoned",
		"timevsinsufficient": "Timeout vs Insufficient Material",
		"win":                "Win",
	}
	for raw, want := range cases {
		if _, reason := Describe(player(raw), player("win")); reason != want {
			t.Errorf("reason for %q = %q, want %q", raw, reason, want)
		}
	}
}

func TestDescribeUnknownReasonIsReturnedAsIs(t *testing.T) {
	if _, reason := Describe(player("some_future_reason"), player("win")); reason != "some_future_reason" {
		t.Errorf("reason = %q", reason)
	}
}
