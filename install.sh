#!/bin/sh
# Usage: curl -fsSL https://raw.githubusercontent.com/takashi145/chess-replay/main/install.sh | sh
# Set CHESS_REPLAY_INSTALL_DIR to install somewhere other than ~/.local/bin.

# Everything runs from main, called on the last line, so a partially downloaded script does nothing.

set -eu

# Asset names must match the ones produced by .github/workflows/release.yml.
base_url="https://github.com/takashi145/chess-replay/releases/latest/download"

tmp=""
sums=""
extract=""

cleanup() {
    [ -n "$tmp" ] && rm -f "$tmp"
    [ -n "$sums" ] && rm -f "$sums"
    [ -n "$extract" ] && rm -rf "$extract"
    return 0
}

download() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1" -o "$2"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        echo "curl or wget is required." >&2
        return 1
    fi
}

sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d ' ' -f 1
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | cut -d ' ' -f 1
    else
        echo "sha256sum or shasum is required." >&2
        return 1
    fi
}

main() {
    install_dir="${CHESS_REPLAY_INSTALL_DIR:-$HOME/.local/bin}"

    case "$(uname -s)" in
        Linux) os=linux ;;
        Darwin) os=osx ;;
        *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64 | amd64) arch=x64 ;;
        arm64 | aarch64) arch=arm64 ;;
        *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
    esac

    # No linux-arm64 build in .github/workflows/release.yml yet; remove this when one is added.
    if [ "$os-$arch" = "linux-arm64" ]; then
        echo "Linux arm64 is not supported yet." >&2
        exit 1
    fi

    asset="chess-replay-$os-$arch.tar.gz"

    mkdir -p "$install_dir"

    # Download to temp files first so a failed or tampered download never replaces the installed binary.
    # The binary's temp file lives in install_dir so the final mv is an atomic rename.
    trap cleanup EXIT
    trap 'exit 1' HUP INT TERM
    tmp=$(mktemp "$install_dir/.chess-replay.XXXXXX")
    sums=$(mktemp)
    extract=$(mktemp -d)

    echo "Downloading chess-replay..."
    download "$base_url/$asset" "$tmp"
    download "$base_url/SHA256SUMS" "$sums"

    expected=$(awk -v f="$asset" '$2 == f || $2 == "*" f { print $1; exit }' "$sums")
    if [ -z "$expected" ]; then
        echo "Checksum for $asset not found in SHA256SUMS." >&2
        exit 1
    fi
    actual=$(sha256 "$tmp")
    if [ "$actual" != "$expected" ]; then
        echo "Checksum mismatch for $asset. Aborting installation." >&2
        exit 1
    fi

    tar -xzf "$tmp" -C "$extract" chess-replay
    mv -f "$extract/chess-replay" "$tmp"
    chmod 755 "$tmp"
    mv -f "$tmp" "$install_dir/chess-replay"

    echo "Installed chess-replay to $install_dir/chess-replay"

    case ":$PATH:" in
        *":$install_dir:"*) echo "Try: chess-replay <username>" ;;
        *)
            echo
            echo "$install_dir is not on your PATH. Add this line to your shell profile:"
            echo "  export PATH=\"$install_dir:\$PATH\""
            ;;
    esac
}

main "$@"
