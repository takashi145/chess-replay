package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/takashi145/chess-replay/internal/app"
	"github.com/takashi145/chess-replay/internal/chesscom"
	"github.com/takashi145/chess-replay/internal/replay"
)

const user = "alice"

func makeGame(opponent, pgn string, minute int) chesscom.Game {
	rating := 1500
	return chesscom.Game{
		PGN:       pgn,
		EndTime:   time.Date(2026, 8, 1, 12, minute, 0, 0, time.UTC),
		TimeClass: "blitz",
		Rated:     true,
		White:     chesscom.Player{Username: user, Rating: &rating, RawResult: "win"},
		Black:     chesscom.Player{Username: opponent, RawResult: "checkmated"},
		URL:       "https://www.chess.com/game/live/" + opponent,
		Rules:     "chess",
	}
}

func press(m Model, keys ...string) (Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		var next tea.Model
		next, cmd = m.Update(keyMsg(k))
		m = next.(Model)
	}
	return m, cmd
}

func keyMsg(k string) tea.KeyMsg {
	switch k {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func singleModel(t *testing.T) Model {
	t.Helper()
	return New(app.Source{Games: []chesscom.Game{makeGame("bob", "1. e4 e5 2. Nf3", 0)}, Single: true}, user)
}

func listModel(t *testing.T) Model {
	t.Helper()
	src := app.Source{
		Title: "alice — Recent games (UTC)",
		Games: []chesscom.Game{
			makeGame("bob", "1. e4 e5", 3),
			makeGame("carol", "1. d4 d5 2. c4", 2),
			makeGame("broken", "1. e4 e4", 1),
		},
		Hidden: 2,
	}
	return New(src, user)
}

func TestSingleGameOpensTheReplayDirectly(t *testing.T) {
	m := singleModel(t)

	if !m.replaying || len(m.snaps) != 4 || m.index != 0 {
		t.Fatalf("replaying=%v snaps=%d index=%d", m.replaying, len(m.snaps), m.index)
	}
	if !strings.Contains(m.View(), "Starting position") {
		t.Error("view should show the starting position")
	}
}

func TestSingleGameWithUnparsablePGNShowsTheReasonInsteadOfTheBoard(t *testing.T) {
	m := New(app.Source{Games: []chesscom.Game{makeGame("bob", "1. e4 e4", 0)}, Single: true}, user)
	view := m.View()

	if !m.replaying {
		t.Fatal("the game should still be opened")
	}
	for _, want := range []string{replay.ErrParse.Error(), "bob", "https://www.chess.com/game/live/bob"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "┌") || strings.Contains(view, "B Back to list") {
		t.Errorf("view should have neither a board nor a way back:\n%s", view)
	}

	// There is no board to move through, but the keys must still be harmless.
	m, _ = press(m, "right", "left", "end", "home", "f", "b")
	if !m.replaying || !strings.Contains(m.View(), replay.ErrParse.Error()) {
		t.Errorf("view changed after pressing keys:\n%s", m.View())
	}
	if _, cmd := press(m, "q"); !isQuit(cmd) {
		t.Error("q should quit")
	}
}

func TestReplayNavigationIsClampedAtBothEnds(t *testing.T) {
	m := singleModel(t)

	m, _ = press(m, "left")
	if m.index != 0 {
		t.Errorf("index = %d, want 0", m.index)
	}

	m, _ = press(m, "right", "right", "right", "right", "right")
	if m.index != 3 {
		t.Errorf("index = %d, want 3", m.index)
	}

	m, _ = press(m, "home")
	if m.index != 0 {
		t.Errorf("home: index = %d", m.index)
	}
	m, _ = press(m, "end")
	if m.index != 3 {
		t.Errorf("end: index = %d", m.index)
	}
}

func TestReplayFlipToggles(t *testing.T) {
	m := singleModel(t)

	m, _ = press(m, "f")
	if !m.flipped || !strings.Contains(m.View(), "h g f e d c b a") {
		t.Error("expected a flipped board")
	}
	m, _ = press(m, "F")
	if m.flipped {
		t.Error("expected the board to flip back")
	}
}

func TestReplayShowsMoveLabels(t *testing.T) {
	m, _ := press(singleModel(t), "right", "right", "right")

	if !strings.Contains(m.View(), "2. Nf3") {
		t.Errorf("view missing white move label:\n%s", m.View())
	}

	m, _ = press(singleModel(t), "right", "right")
	if !strings.Contains(m.View(), "1... e5") {
		t.Errorf("view missing black move label:\n%s", m.View())
	}
}

func TestSingleReplayHasNoBackOption(t *testing.T) {
	m := singleModel(t)

	if strings.Contains(m.View(), "Back to list") {
		t.Error("single replay should not offer going back")
	}
	m, _ = press(m, "b")
	if !m.replaying {
		t.Error("b should do nothing without a list")
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []string{"q", "Q", "esc", "ctrl+c"} {
		if _, cmd := press(singleModel(t), k); !isQuit(cmd) {
			t.Errorf("replay: %q should quit", k)
		}
		if _, cmd := press(listModel(t), k); !isQuit(cmd) {
			t.Errorf("list: %q should quit", k)
		}
	}
}

func TestListShowsGamesAndCounts(t *testing.T) {
	next, _ := listModel(t).Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	view := next.(Model).View()

	for _, want := range []string{"Recent games", "3 games", "2 hidden (Chess960 unsupported)", "vs bob", "vs carol", "2 moves", "3 moves", "? moves"} {
		if !strings.Contains(view, want) {
			t.Errorf("list view missing %q:\n%s", want, view)
		}
	}
}

func TestListHidesColumnsOnNarrowTerminals(t *testing.T) {
	m := listModel(t)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	view := next.(Model).View()

	for _, unwanted := range []string{"blitz", "moves", "2026-08-01"} {
		if strings.Contains(view, unwanted) {
			t.Errorf("narrow view should not contain %q:\n%s", unwanted, view)
		}
	}

	next, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	view = next.(Model).View()
	for _, want := range []string{"blitz", "moves", "2026-08-01 12:03"} {
		if !strings.Contains(view, want) {
			t.Errorf("wide view should contain %q:\n%s", want, view)
		}
	}
}

func TestListCursorWrapsAround(t *testing.T) {
	m := listModel(t)

	m, _ = press(m, "up")
	if m.cursor != 2 {
		t.Errorf("up from top: cursor = %d, want 2", m.cursor)
	}
	m, _ = press(m, "down")
	if m.cursor != 0 {
		t.Errorf("down from bottom: cursor = %d, want 0", m.cursor)
	}
}

func TestListScrollsToKeepTheCursorVisible(t *testing.T) {
	var games []chesscom.Game
	for i := range 30 {
		games = append(games, makeGame("opp"+string(rune('a'+i%26))+string(rune('a'+i/26)), "1. e4 e5", i))
	}
	next, _ := New(app.Source{Title: "t", Games: games}, user).Update(tea.WindowSizeMsg{Width: 100, Height: 15})
	m := next.(Model)

	m, _ = press(m, "end")

	size := m.pageSize()
	if size != 11 || m.top != 19 || m.cursor != 29 {
		t.Errorf("pageSize=%d top=%d cursor=%d", size, m.top, m.cursor)
	}
	if rows := strings.Count(m.View(), "vs opp"); rows != size {
		t.Errorf("view shows %d rows, want %d", rows, size)
	}
}

func TestEnterOpensTheSelectedGameAndBackReturnsToTheList(t *testing.T) {
	m := listModel(t)

	m, _ = press(m, "down", "enter")
	if !m.replaying || !strings.Contains(m.View(), "carol") {
		t.Fatalf("expected carol's game, got:\n%s", m.View())
	}
	if !strings.Contains(m.View(), "B Back to list") {
		t.Error("replay from a list should offer going back")
	}

	m, _ = press(m, "right", "b")
	if m.replaying || m.cursor != 1 {
		t.Errorf("replaying=%v cursor=%d, want list at cursor 1", m.replaying, m.cursor)
	}

	m, _ = press(m, "enter")
	if m.index != 0 {
		t.Errorf("reopening should start from the beginning, index = %d", m.index)
	}
}

func TestOpeningAnUnparsableGameShowsTheReasonAndCanGoBack(t *testing.T) {
	m := listModel(t)

	m, _ = press(m, "end", "enter")

	if !m.replaying || !strings.Contains(m.View(), replay.ErrParse.Error()) {
		t.Fatalf("expected the reason in place of the board:\n%s", m.View())
	}
	if !strings.Contains(m.View(), "B Back to list") {
		t.Error("the way back to the list should still be offered")
	}

	m, _ = press(m, "b", "up", "enter")
	if !m.replaying || m.failure != "" {
		t.Errorf("replaying=%v failure=%q, want a clean replay of the next game", m.replaying, m.failure)
	}
}

func TestFocusedPlayerIsHighlighted(t *testing.T) {
	m := singleModel(t)
	view := m.View()

	if !strings.Contains(view, yellow+"alice (1500)"+reset) {
		t.Errorf("focus player not highlighted:\n%s", view)
	}
	if !strings.Contains(view, bold+"bob"+reset) {
		t.Errorf("opponent should be bold:\n%s", view)
	}
}

func TestProgressBar(t *testing.T) {
	cases := []struct {
		index, total int
		want         string
	}{
		{0, 10, strings.Repeat("─", 24) + " 0 / 10"},
		{10, 10, strings.Repeat("━", 24) + " 10 / 10"},
		{5, 10, strings.Repeat("━", 12) + strings.Repeat("─", 12) + " 5 / 10"},
		{0, 0, strings.Repeat("━", 24) + " 0 / 0"},
	}
	for _, c := range cases {
		if got := renderProgressBar(c.index, c.total); got != c.want {
			t.Errorf("renderProgressBar(%d, %d) = %q, want %q", c.index, c.total, got, c.want)
		}
	}
}

func TestGameInfoFormatting(t *testing.T) {
	g := makeGame("bob", "1. e4", 5)

	if got := formatGameInfo(g); got != "Blitz · Rated · 2026-08-01 12:05 UTC" {
		t.Errorf("formatGameInfo = %q", got)
	}

	g.TimeClass, g.Rated = "", false
	if got := formatGameInfo(g); got != "Unknown · Casual · 2026-08-01 12:05 UTC" {
		t.Errorf("formatGameInfo = %q", got)
	}
}
