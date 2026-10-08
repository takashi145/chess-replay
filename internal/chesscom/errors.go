package chesscom

import "fmt"

type PlayerNotFoundError struct{ Username string }

func (e *PlayerNotFoundError) Error() string {
	return fmt.Sprintf("Player %q was not found on Chess.com.", e.Username)
}

type NoGamesError struct {
	Username string
	Archive  *Archive
}

func (e *NoGamesError) Error() string {
	if e.Archive != nil {
		return fmt.Sprintf("No public games found for %q in %s.", e.Username, e.Archive)
	}
	return fmt.Sprintf("No public games found for %q.", e.Username)
}

type ConnectionError struct {
	Message string
	Err     error
}

func (e *ConnectionError) Error() string { return e.Message }
func (e *ConnectionError) Unwrap() error { return e.Err }
