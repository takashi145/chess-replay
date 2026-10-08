package chesscom

import "strings"

type Result int

const (
	Win Result = iota
	Loss
	Draw
)

func (r Result) String() string {
	switch r {
	case Win:
		return "Win"
	case Draw:
		return "Draw"
	default:
		return "Loss"
	}
}

var drawRawResults = map[string]bool{
	"agreed": true, "repetition": true, "stalemate": true,
	"insufficient": true, "50move": true, "timevsinsufficient": true,
}

// Describe reports how the game ended for self. Chess.com marks the winner's own result as "win";
// the terminal reason (checkmate, timeout, ...) is only present on the losing/drawing side.
func Describe(self, opponent Player) (Result, string) {
	raw := strings.ToLower(self.RawResult)
	isWin := raw == "win"

	result := Loss
	switch {
	case isWin:
		result = Win
	case drawRawResults[raw]:
		result = Draw
	}

	reasonSource := self.RawResult
	if isWin {
		reasonSource = opponent.RawResult
	}
	return result, describeReason(reasonSource)
}

func describeReason(raw string) string {
	switch strings.ToLower(raw) {
	case "checkmated":
		return "Checkmate"
	case "timeout":
		return "Timeout"
	case "resigned":
		return "Resignation"
	case "repetition":
		return "Repetition"
	case "stalemate":
		return "Stalemate"
	case "insufficient":
		return "Insufficient Material"
	case "50move":
		return "50-move Rule"
	case "agreed":
		return "Agreement"
	case "abandoned":
		return "Abandoned"
	case "timevsinsufficient":
		return "Timeout vs Insufficient Material"
	case "win":
		return "Win"
	default:
		return raw
	}
}
