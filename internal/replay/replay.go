// Package replay turns a PGN into the board after every move.
package replay

import (
	"errors"
	"strings"

	lib "github.com/corentings/chess/v2"
)

// ErrParse is returned when a game's PGN cannot be turned into a replay.
var ErrParse = errors.New("This game could not be replayed. It may use a chess variant that is not supported.")

type Side int

const (
	White Side = iota
	Black
)

type PieceKind int

const (
	Pawn PieceKind = iota
	Knight
	Bishop
	Rook
	Queen
	King
)

type Piece struct {
	Color Side
	Kind  PieceKind
}

type Square struct {
	File, Rank int
}

// Snapshot is the board after MoveNumber plies (0 is the starting position).
type Snapshot struct {
	Board      [8][8]*Piece // indexed [file][rank]
	MoveNumber int
	SAN        string // empty for the starting position
	From, To   *Square
}

// Build returns one snapshot per ply, preceded by the starting position.
func Build(pgn string) ([]Snapshot, error) {
	game, err := parse(pgn)
	if err != nil {
		return nil, err
	}

	positions := game.Positions()
	moves := game.Moves()

	snapshots := make([]Snapshot, 0, len(moves)+1)
	snapshots = append(snapshots, Snapshot{
		Board: captureBoard(positions[0]),
	})

	for i, move := range moves {
		snapshots = append(snapshots, Snapshot{
			Board:      captureBoard(positions[i+1]),
			MoveNumber: i + 1,
			SAN:        lib.AlgebraicNotation{}.Encode(positions[i], move),
			From:       toSquare(move.S1()),
			To:         toSquare(move.S2()),
		})
	}
	return snapshots, nil
}

func CountMoves(pgn string) (int, error) {
	game, err := parse(pgn)
	if err != nil {
		return 0, err
	}
	return len(game.Moves()), nil
}

func parse(pgn string) (game *lib.Game, err error) {
	// The library's parser panics on some malformed input (e.g. "01"); treat that as a parse failure too.
	defer func() {
		if recover() != nil {
			game, err = nil, ErrParse
		}
	}()

	scanner := lib.NewScanner(strings.NewReader(pgn))
	if !scanner.HasNext() {
		return nil, ErrParse
	}
	game, err = scanner.ParseNext()
	if err != nil || game == nil {
		return nil, ErrParse
	}
	return game, nil
}

func captureBoard(pos *lib.Position) [8][8]*Piece {
	var grid [8][8]*Piece
	board := pos.Board()
	for file := range 8 {
		for rank := range 8 {
			p := board.Piece(lib.NewSquare(lib.File(file), lib.Rank(rank)))
			if p == lib.NoPiece {
				continue
			}
			grid[file][rank] = &Piece{Color: toSide(p.Color()), Kind: toKind(p.Type())}
		}
	}
	return grid
}

func toSquare(sq lib.Square) *Square {
	return &Square{File: int(sq.File()), Rank: int(sq.Rank())}
}

func toSide(c lib.Color) Side {
	if c == lib.White {
		return White
	}
	return Black
}

func toKind(t lib.PieceType) PieceKind {
	switch t {
	case lib.Knight:
		return Knight
	case lib.Bishop:
		return Bishop
	case lib.Rook:
		return Rook
	case lib.Queen:
		return Queen
	case lib.King:
		return King
	default:
		return Pawn
	}
}
