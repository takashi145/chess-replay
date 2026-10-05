// Package ui is the terminal interface: a game list and the replay screen.
package ui

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/takashi145/chess-replay/internal/app"
	"github.com/takashi145/chess-replay/internal/chesscom"
	"github.com/takashi145/chess-replay/internal/replay"
)

const (
	progressBarWidth = 24

	// Lines around the list rows: title, blank, blank, message, hint.
	listReservedLines = 5
	minPageSize       = 3

	defaultWidth  = 80
	defaultHeight = 24

	bold   = "\x1b[1m"
	yellow = "\x1b[1;33m"
	grey   = "\x1b[90m"
	red    = "\x1b[31m"
	cyan   = "\x1b[1;36m"
)

type entry struct {
	game  chesscom.Game
	moves int // -1 when the PGN could not be parsed
}

// Model shows a list of games to choose from, and replays the chosen one.
// A source with a single game skips the list.
type Model struct {
	username string
	title    string
	hidden   int
	entries  []entry

	width, height int
	cursor, top   int
	message       string

	replaying bool
	game      chesscom.Game
	snaps     []replay.Snapshot
	index     int
	flipped   bool
}

// New returns an error when a single game cannot be replayed, so the caller can report it and exit.
func New(src app.Source, username string) (Model, error) {
	m := Model{
		username: username,
		title:    src.Title,
		hidden:   src.Hidden,
		width:    defaultWidth,
		height:   defaultHeight,
	}

	if src.Single {
		if err := m.open(src.Games[0]); err != nil {
			return m, err
		}
		return m, nil
	}

	for _, g := range src.Games {
		moves, err := replay.CountMoves(g.PGN)
		if err != nil {
			moves = -1
		}
		m.entries = append(m.entries, entry{game: g, moves: moves})
	}
	return m, nil
}

func (m *Model) open(g chesscom.Game) error {
	snaps, err := replay.Build(g.PGN)
	if err != nil {
		return err
	}

	m.game = g
	m.snaps = snaps
	m.index = 0
	m.flipped = false
	m.replaying = true
	return nil
}

func (m Model) Init() tea.Cmd { return tea.ClearScreen }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scrollIntoView()
		return m, nil
	case tea.KeyMsg:
		key := strings.ToLower(msg.String())
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.replaying {
			return m.updateReplay(key)
		}
		return m.updateList(key)
	}
	return m, nil
}

func (m Model) updateList(key string) (tea.Model, tea.Cmd) {
	n := len(m.entries)

	switch key {
	case "up", "k":
		m.cursor = (m.cursor - 1 + n) % n
	case "down", "j":
		m.cursor = (m.cursor + 1) % n
	case "pgup":
		m.cursor = max(0, m.cursor-m.pageSize())
	case "pgdown":
		m.cursor = min(n-1, m.cursor+m.pageSize())
	case "home":
		m.cursor = 0
	case "end":
		m.cursor = n - 1
	case "enter":
		if err := m.open(m.entries[m.cursor].game); err != nil {
			m.message = err.Error()
		} else {
			m.message = ""
		}
	case "q", "esc":
		return m, tea.Quit
	}

	m.scrollIntoView()
	return m, nil
}

func (m Model) updateReplay(key string) (tea.Model, tea.Cmd) {
	last := len(m.snaps) - 1

	switch key {
	case "right", "l":
		m.index = min(m.index+1, last)
	case "left", "h":
		m.index = max(m.index-1, 0)
	case "home":
		m.index = 0
	case "end":
		m.index = last
	case "f":
		m.flipped = !m.flipped
	case "b":
		if m.openedFromList() {
			m.replaying = false
			return m, tea.ClearScreen
		}
	case "q", "esc":
		return m, tea.Quit
	}
	return m, nil
}

// A single game is replayed without a list, so there is nothing to go back to.
func (m Model) openedFromList() bool { return len(m.entries) > 0 }

func (m Model) pageSize() int {
	return max(minPageSize, min(m.height-listReservedLines, len(m.entries)))
}

func (m *Model) scrollIntoView() {
	size := m.pageSize()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+size {
		m.top = m.cursor - size + 1
	}
	m.top = max(0, min(m.top, len(m.entries)-size))
}

func (m Model) View() string {
	if m.replaying {
		return m.viewReplay()
	}
	return m.viewList()
}

func (m Model) viewList() string {
	var b strings.Builder

	summary := fmt.Sprintf("%s(%d games", grey, len(m.entries))
	if m.hidden > 0 {
		summary += fmt.Sprintf(",%s \x1b[33m%d hidden (Chess960 unsupported)%s", reset, m.hidden, grey)
	}
	summary += ")" + reset
	fmt.Fprintf(&b, "%s%s%s %s\n\n", bold, m.title, reset, summary)

	showTimeClass := m.width >= 70
	showMoves := m.width >= 85
	showDate := m.width >= 100

	end := min(m.top+m.pageSize(), len(m.entries))
	for i := m.top; i < end; i++ {
		e := m.entries[i]
		opponent := e.game.Opponent(m.username)
		result, _ := chesscom.Describe(e.game.Player(m.username), opponent)

		line := fmt.Sprintf("vs %-16s %-6s", opponent.Username, result)
		if showTimeClass {
			line += fmt.Sprintf(" %-8s", e.game.TimeClass)
		}
		if showMoves {
			moves := "? moves"
			if e.moves >= 0 {
				moves = fmt.Sprintf("%d moves", e.moves)
			}
			line += fmt.Sprintf(" %-10s", moves)
		}
		if showDate {
			line += " " + e.game.EndTime.UTC().Format("2006-01-02 15:04")
		}

		row := fmt.Sprintf("%3d. %s", i+1, line)
		if i == m.cursor {
			fmt.Fprintf(&b, "%s> %s%s\n", cyan, row, reset)
		} else {
			fmt.Fprintf(&b, "  %s\n", row)
		}
	}

	b.WriteString("\n")
	if m.message != "" {
		fmt.Fprintf(&b, "%s%s%s", red, m.message, reset)
	}
	fmt.Fprintf(&b, "\n%s↑/↓ Move   Enter Replay   Q Quit%s", grey, reset)
	return b.String()
}

func (m Model) viewReplay() string {
	snap := m.snaps[m.index]
	top, bottom := m.game.Black, m.game.White
	if m.flipped {
		top, bottom = bottom, top
	}

	var b strings.Builder
	fmt.Fprintf(&b, " %s\n", m.formatPlayer(top))
	b.WriteString("          vs\n")
	fmt.Fprintf(&b, "   %s\n", m.formatPlayer(bottom))
	fmt.Fprintf(&b, "   %s%s%s\n", grey, formatGameInfo(m.game), reset)
	fmt.Fprintf(&b, "   %s%s%s\n", grey, m.game.URL, reset)
	b.WriteString("\n")
	b.WriteString(renderBoard(snap, m.flipped))
	b.WriteString("\n\n")
	b.WriteString(formatMove(snap))
	b.WriteString("\n\n")
	b.WriteString(renderProgressBar(m.index, len(m.snaps)-1))
	b.WriteString("\n\n")

	backHint := ""
	if m.openedFromList() {
		backHint = "B Back to list   "
	}
	fmt.Fprintf(&b, "%s← Previous   → Next   F Flip   Home/End   %sQ Quit%s", grey, backHint, reset)
	return b.String()
}

func (m Model) formatPlayer(p chesscom.Player) string {
	label := p.Username
	if p.Rating != nil {
		label = fmt.Sprintf("%s (%d)", p.Username, *p.Rating)
	}

	if strings.EqualFold(p.Username, m.username) {
		return yellow + label + reset
	}
	return bold + label + reset
}

func formatGameInfo(g chesscom.Game) string {
	timeClass := "Unknown"
	if g.TimeClass != "" {
		timeClass = strings.ToUpper(g.TimeClass[:1]) + g.TimeClass[1:]
	}
	rated := "Casual"
	if g.Rated {
		rated = "Rated"
	}
	return fmt.Sprintf("%s · %s · %s UTC", timeClass, rated, g.EndTime.UTC().Format("2006-01-02 15:04"))
}

func formatMove(s replay.Snapshot) string {
	if s.SAN == "" {
		return "Starting position"
	}

	fullMove := (s.MoveNumber + 1) / 2
	if s.MoveNumber%2 == 1 {
		return fmt.Sprintf("%d. %s", fullMove, s.SAN)
	}
	return fmt.Sprintf("%d... %s", fullMove, s.SAN)
}

func renderProgressBar(index, total int) string {
	filled := progressBarWidth
	if total != 0 {
		filled = int(math.Round(float64(progressBarWidth) * float64(index) / float64(total)))
	}
	return strings.Repeat("━", filled) + strings.Repeat("─", progressBarWidth-filled) + fmt.Sprintf(" %d / %d", index, total)
}
