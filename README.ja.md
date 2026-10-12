# ChessReplay

Chess.com と Lichess の過去の対局を、ターミナルで 1 手ずつ見返すツールです。

このプロジェクトは非公式のツールです。Chess.com・Lichess とは関係がなく、どちらからも公認されたものでもありません。

<img width="653" height="393" alt="ChessReplay の再生画面" src="https://github.com/user-attachments/assets/78f51984-0309-4661-a09d-cfb5e7eedb98" />

## インストール

**Windows**（PowerShell）

```powershell
irm https://raw.githubusercontent.com/takashi145/chess-replay/main/install.ps1 | iex
```

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/takashi145/chess-replay/main/install.sh | sh
```

アップデートするときは、同じコマンドをもう一度実行してください。
[Releases](https://github.com/takashi145/chess-replay/releases/latest) ページからお使いの環境用のアーカイブをダウンロードして展開し、
`chess-replay` を自分で `PATH` の通った場所に置くこともできます。

### インストーラーが行うこと

- **Windows**：`chess-replay.exe` を `%LOCALAPPDATA%\Programs\chess-replay` に置き、そのフォルダをユーザーの `PATH` に追加します。
- **macOS / Linux**：`chess-replay` を `~/.local/bin` に置きます（`CHESS_REPLAY_INSTALL_DIR` で変更できます）。

### アンインストール

**Windows**

1. インストール先のフォルダを削除します。

   ```powershell
   Remove-Item -Recurse "$env:LOCALAPPDATA\Programs\chess-replay"
   ```

2. スタートメニューから「**環境変数を編集**」を開き、`Path` を選択して
   `...\Programs\chess-replay` のエントリを削除します。

**macOS / Linux**

```sh
rm ~/.local/bin/chess-replay
```

## 使い方

```
chess-replay <username> [options]
```

| 引数 / オプション | 説明 |
|---|---|
| `<username>` | Chess.com または Lichess のユーザー名（`--set-site` と `--show-config` 以外では必須） |
| `--site <chesscom\|lichess>` | 対局を取得するサイト（既定: `chesscom`） |
| `--month <YYYY-MM>` | 指定した月の対局を一覧表示する |
| `--set-site <chesscom\|lichess>` | `--site` を指定しないときに使うサイトを保存して終了する |
| `--show-config` | 設定ファイルの場所と、現在の既定のサイトを表示して終了する |
| `--verbose` | 診断情報を標準エラー出力に表示する |

オプションを指定しない場合は、直近の 1 局を再生します。`--site` を指定するか、`--set-site` で別のサイトを保存していない限り、サイトは Chess.com です。

### 例

```
# 直近の 1 局を再生
chess-replay <username>

# Lichess のユーザーの直近の 1 局を再生
chess-replay --site lichess <username>

# 2026 年 8 月の対局を一覧表示
chess-replay <username> --month 2026-08

# Lichess を既定のサイトにする
chess-replay --set-site lichess

# 設定ファイルの場所と、現在の既定のサイトを表示
chess-replay --show-config
```

## 操作方法

**対局一覧**（`--month`）

| キー | 操作 |
|---|---|
| `↑` / `↓` | 選択を移動 |
| `Enter` | 選択した対局を再生 |
| `Q` / `Esc` | 終了 |

Lichess の一覧は 30 局ずつ読み込みます。一番下の **Load more games** を選んで `Enter` を押すと、続きを読み込みます。

**再生画面**

| キー | 操作 |
|---|---|
| `←` / `→` | 前の手 / 次の手 |
| `Home` / `End` | 対局の最初 / 最後へ移動 |
| `F` | 盤面を反転 |
| `B` | 対局一覧に戻る（一覧から開いた場合） |
| `Q` / `Esc` | 終了 |

## 制限事項

- Chess960 など、一部の種類の対局にはまだ対応していません。

## ライセンス

MIT
