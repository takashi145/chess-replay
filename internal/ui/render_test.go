package ui

import (
	"strings"
	"testing"

	"github.com/takashi145/chess-replay/internal/replay"
)

const pawn = "♙"

func snapshotAfter(t *testing.T, pgn string, ply int) replay.Snapshot {
	t.Helper()
	snaps, err := replay.Build(pgn)
	if err != nil {
		t.Fatal(err)
	}
	return snaps[ply]
}

func TestRenderWhiteAndBlackPiecesUseDifferentColors(t *testing.T) {
	out := renderBoard(snapshotAfter(t, "1. e4 e5", 0), false)

	if !strings.Contains(out, "\x1b[37m"+pawn+reset) {
		t.Error("white pawn color missing")
	}
	if !strings.Contains(out, "\x1b[38;5;208m"+pawn+reset) {
		t.Error("black pawn color missing")
	}
}

func TestRenderNeverUsesFilledGlyphsOrEmojiSelectors(t *testing.T) {
	out := renderBoard(snapshotAfter(t, "1. e4 e5", 0), false)

	// U+265F (filled pawn) is drawn as an emoji by some terminals, ignoring the color.
	for _, bad := range []string{"♚", "♛", "♜", "♝", "♞", "♟", "︎", "️"} {
		if strings.Contains(out, bad) {
			t.Errorf("output contains %q", bad)
		}
	}
}

func TestRenderFileLabelsFollowOrientation(t *testing.T) {
	snap := snapshotAfter(t, "1. e4 e5", 0)

	if !strings.Contains(renderBoard(snap, false), "a b c d e f g h") {
		t.Error("expected files a-h when not flipped")
	}
	if !strings.Contains(renderBoard(snap, true), "h g f e d c b a") {
		t.Error("expected files h-a when flipped")
	}
}

func TestRenderHighlightsFromAndToSquares(t *testing.T) {
	out := renderBoard(snapshotAfter(t, "1. e4", 1), false)

	if !strings.Contains(out, "\x1b["+highlightBackground+"m "+reset) {
		t.Error("vacated e2 should be highlighted")
	}
	if !strings.Contains(out, "\x1b[37;"+highlightBackground+"m"+pawn+reset) {
		t.Error("pawn on e4 should be highlighted")
	}
}

func TestRenderCapturingPieceKeepsItsOwnColorOnHighlight(t *testing.T) {
	out := renderBoard(snapshotAfter(t, "1. e4 d5 2. exd5", 3), false)

	if !strings.Contains(out, "\x1b[37;"+highlightBackground+"m"+pawn+reset) {
		t.Error("capturing white pawn lost its color")
	}
}

func TestRenderFlippedPutsRankOneAtTheTop(t *testing.T) {
	snap := snapshotAfter(t, "1. e4 e5", 0)

	lines := strings.Split(renderBoard(snap, true), "\n")
	// lines: file labels, top border, 8 ranks, bottom border, file labels
	if !strings.HasPrefix(lines[2], "1 │") || !strings.HasPrefix(lines[9], "8 │") {
		t.Errorf("rank order wrong:\n%s", strings.Join(lines, "\n"))
	}
}
