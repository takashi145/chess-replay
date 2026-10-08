# ChessReplay

English | [日本語](README.ja.md)

A tool to replay Chess.com games in the terminal.

<img width="653" height="393" alt="ChessReplay replay screen" src="https://github.com/user-attachments/assets/78f51984-0309-4661-a09d-cfb5e7eedb98" />

## Installation

**Windows** (PowerShell)

```powershell
irm https://raw.githubusercontent.com/takashi145/chess-replay/main/install.ps1 | iex
```

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/takashi145/chess-replay/main/install.sh | sh
```

Run the same command again to update. You can also download the archive for your platform from the
[Releases](https://github.com/takashi145/chess-replay/releases/latest) page, extract it, and put `chess-replay` on your `PATH` yourself.

### What the installer does

- **Windows**: puts `chess-replay.exe` in `%LOCALAPPDATA%\Programs\chess-replay` and adds that folder to your user `PATH`.
- **macOS / Linux**: puts `chess-replay` in `~/.local/bin` (set `CHESS_REPLAY_INSTALL_DIR` to change it).

### Uninstall

**Windows**

1. Delete the install folder:

   ```powershell
   Remove-Item -Recurse "$env:LOCALAPPDATA\Programs\chess-replay"
   ```

2. Open **Edit environment variables for your account** from the Start menu, select `Path`, and remove the
   `...\Programs\chess-replay` entry.

**macOS / Linux**

```sh
rm ~/.local/bin/chess-replay
```

## Usage

```
chess-replay <username> [options]
```

| Argument / Option | Description |
|---|---|
| `<username>` | Chess.com username (required) |
| `--last <N>` | List the most recent N games to choose from |
| `--random` | Replay a random past game |
| `--month <YYYY-MM>` | List games from a specific month |
| `--verbose` | Print diagnostic information to stderr |

With no options, it replays your single most recent game.

### Examples

```
# Replay the single most recent game
chess-replay <username>

# List the 20 most recent games to pick from
chess-replay <username> --last 20

# List games played in August 2026
chess-replay <username> --month 2026-08

# Replay a random past game
chess-replay <username> --random
```

## Controls

**Game list** (`--last` / `--month`)

| Key | Action |
|---|---|
| `↑` / `↓` | Move selection |
| `Enter` | Replay selected game |
| `Q` / `Esc` | Quit |

**Replay screen**

| Key | Action |
|---|---|
| `←` / `→` | Previous / next move |
| `Home` / `End` | Jump to start / end of game |
| `F` | Flip board |
| `B` | Back to game list (when opened from a list) |
| `Q` / `Esc` | Quit |

## Limitations

- Chess960 games are not supported yet. They are hidden from game lists and skipped by `--random`.

## License

MIT
