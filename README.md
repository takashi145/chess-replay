# ChessReplay

English | [日本語](README.ja.md)

A tool to step through past Chess.com and Lichess games move by move in the terminal.

This project is not affiliated with or endorsed by Chess.com or Lichess.

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
| `<username>` | Chess.com or Lichess username (required, except with `--set-site` and `--show-config`) |
| `--site <chesscom\|lichess>` | Site to load games from (default: `chesscom`) |
| `--month <YYYY-MM>` | List games from a specific month |
| `--set-site <chesscom\|lichess>` | Save the site to use when `--site` is not given, then exit |
| `--show-config` | Show the config file's location and the current default site, then exit |
| `--verbose` | Print diagnostic information to stderr |

With no options, it replays your single most recent game. The site is Chess.com unless you pass `--site` or have saved another one with `--set-site`.

### Examples

```
# Replay the single most recent game
chess-replay <username>

# Replay the single most recent game of a Lichess player
chess-replay --site lichess <username>

# List games played in August 2026
chess-replay <username> --month 2026-08

# Make Lichess the default site
chess-replay --set-site lichess

# Show the config file and the current default site
chess-replay --show-config
```

## Controls

**Game list** (`--month`)

| Key | Action |
|---|---|
| `↑` / `↓` | Move selection |
| `Enter` | Replay selected game |
| `Q` / `Esc` | Quit |

Lichess lists load 30 games at a time. Select **Load more games** at the bottom and press `Enter` to load the next ones.

**Replay screen**

| Key | Action |
|---|---|
| `←` / `→` | Previous / next move |
| `Home` / `End` | Jump to start / end of game |
| `F` | Flip board |
| `B` | Back to game list (when opened from a list) |
| `Q` / `Esc` | Quit |

## Limitations

- Some game types, such as Chess960, are not supported yet.

## License

MIT
