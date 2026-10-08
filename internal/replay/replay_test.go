package replay

import (
	"errors"
	"testing"
)

func TestBuildReturnsOneSnapshotPerPlyPlusInitial(t *testing.T) {
	snaps, err := Build("1. e4 e5 2. Nf3 Nc6")
	if err != nil {
		t.Fatal(err)
	}

	if len(snaps) != 5 { // initial position + 4 plies
		t.Fatalf("got %d snapshots, want 5", len(snaps))
	}
	if snaps[0].From != nil || snaps[0].To != nil {
		t.Errorf("initial snapshot should have no move squares: %+v", snaps[0])
	}

	wantSAN := []string{"", "e4", "e5", "Nf3", "Nc6"}
	for i, want := range wantSAN {
		if snaps[i].SAN != want || snaps[i].MoveNumber != i {
			t.Errorf("snaps[%d] = %q (move %d), want %q (move %d)", i, snaps[i].SAN, snaps[i].MoveNumber, want, i)
		}
	}
}

func TestBuildTracksPiecesAndMoveSquares(t *testing.T) {
	snaps, err := Build("1. e4 e5")
	if err != nil {
		t.Fatal(err)
	}

	start, afterE4 := snaps[0], snaps[1]
	if p := start.Board[4][1]; p == nil || p.Color != White || p.Kind != Pawn {
		t.Errorf("e2 at start = %+v", p)
	}
	if afterE4.Board[4][1] != nil {
		t.Error("e2 should be empty after e4")
	}
	if p := afterE4.Board[4][3]; p == nil || p.Color != White || p.Kind != Pawn {
		t.Errorf("e4 after e4 = %+v", p)
	}
	if *afterE4.From != (Square{4, 1}) || *afterE4.To != (Square{4, 3}) {
		t.Errorf("move squares = %v -> %v", *afterE4.From, *afterE4.To)
	}
}

func TestCountMoves(t *testing.T) {
	n, err := CountMoves("1. e4 e5 2. Nf3")
	if err != nil || n != 3 {
		t.Errorf("CountMoves = %d, %v", n, err)
	}
}

func TestUnreplayablePGNIsAParseError(t *testing.T) {
	cases := map[string]string{
		"empty":         "",
		"illegal moves": "1. e4 e4 2. e4 e4",
		"parser panics": "01", // the library panics on this input
		"not a pgn":     "not a pgn",
	}

	for name, pgn := range cases {
		if _, err := Build(pgn); !errors.Is(err, ErrParse) {
			t.Errorf("Build(%s) err = %v, want ErrParse", name, err)
		}
		if _, err := CountMoves(pgn); !errors.Is(err, ErrParse) {
			t.Errorf("CountMoves(%s) err = %v, want ErrParse", name, err)
		}
	}
}
