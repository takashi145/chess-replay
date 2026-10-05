package ui

import (
	"fmt"
	"strings"

	"github.com/takashi145/chess-replay/internal/replay"
)

// Both sides use the same glyphs; the side is distinguished by color only.
// The filled glyphs (e.g. U+265F) are avoided because some terminals draw them as an emoji, ignoring the color.
const (
	whiteColor          = "37"
	blackColor          = "38;5;208"
	highlightBackground = "48;5;59"
	reset               = "\x1b[0m"
)

var glyphs = map[replay.PieceKind]rune{
	replay.King:   '♔',
	replay.Queen:  '♕',
	replay.Rook:   '♖',
	replay.Bishop: '♗',
	replay.Knight: '♘',
	replay.Pawn:   '♙',
}

// Returns ANSI-colored text for the 8x8 board only (caller adds header/footer).
func renderBoard(s replay.Snapshot, flipped bool) string {
	files := fileLabels(flipped)
	var b strings.Builder

	fmt.Fprintf(&b, "    %s\n", files)
	b.WriteString("  ┌─────────────────┐\n")

	for _, rank := range ranks(flipped) {
		fmt.Fprintf(&b, "%d │ ", rank+1)

		for _, file := range fileOrder(flipped) {
			sq := replay.Square{File: file, Rank: rank}
			highlighted := (s.From != nil && *s.From == sq) || (s.To != nil && *s.To == sq)

			b.WriteString(renderSquare(s.Board[file][rank], highlighted))
			b.WriteByte(' ')
		}

		b.WriteString("│\n")
	}

	b.WriteString("  └─────────────────┘\n")
	fmt.Fprintf(&b, "    %s", files)

	return b.String()
}

func renderSquare(piece *replay.Piece, highlighted bool) string {
	if piece == nil {
		if highlighted {
			return "\x1b[" + highlightBackground + "m " + reset
		}
		return " "
	}

	color := whiteColor
	if piece.Color == replay.Black {
		color = blackColor
	}
	if highlighted {
		color += ";" + highlightBackground
	}
	return fmt.Sprintf("\x1b[%sm%c%s", color, glyphs[piece.Kind], reset)
}

func ranks(flipped bool) []int {
	if flipped {
		return []int{0, 1, 2, 3, 4, 5, 6, 7}
	}
	return []int{7, 6, 5, 4, 3, 2, 1, 0}
}

func fileOrder(flipped bool) []int {
	if flipped {
		return []int{7, 6, 5, 4, 3, 2, 1, 0}
	}
	return []int{0, 1, 2, 3, 4, 5, 6, 7}
}

func fileLabels(flipped bool) string {
	labels := make([]string, 0, 8)
	for _, f := range fileOrder(flipped) {
		labels = append(labels, string(rune('a'+f)))
	}
	return strings.Join(labels, " ")
}
