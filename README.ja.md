# ChessReplay

Chess.com の対局をターミナルで再生するツールです。

<img width="548" alt="ChessReplay の再生画面" src="https://github.com/user-attachments/assets/51e2f9a5-b611-4a59-8e05-97a715c6356e" />

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
| `<username>` | Chess.com のユーザー名（必須） |
| `--last <N>` | 直近 N 局を一覧表示し、その中から選ぶ |
| `--random` | 過去の対局からランダムに 1 局再生する |
| `--month <YYYY-MM>` | 指定した月の対局を一覧表示する |
| `--verbose` | 診断情報を標準エラー出力に表示する |

オプションを指定しない場合は、直近の 1 局を再生します。

### 例

```
# 直近の 1 局を再生
chess-replay <username>

# 直近 20 局を一覧表示して選ぶ
chess-replay <username> --last 20

# 2026 年 8 月の対局を一覧表示
chess-replay <username> --month 2026-08

# 過去の対局をランダムに再生
chess-replay <username> --random
```

## 操作方法

**対局一覧**（`--last` / `--month`）

| キー | 操作 |
|---|---|
| `↑` / `↓` | 選択を移動 |
| `Enter` | 選択した対局を再生 |
| `Q` / `Esc` | 終了 |

**再生画面**

| キー | 操作 |
|---|---|
| `←` / `→` | 前の手 / 次の手 |
| `Home` / `End` | 対局の最初 / 最後へ移動 |
| `F` | 盤面を反転 |
| `B` | 対局一覧に戻る（一覧から開いた場合） |
| `Q` / `Esc` | 終了 |

## 制限事項

- Chess960の対局にはまだ対応していません。対局一覧には表示されず、`--random` でも選ばれません。

## ライセンス

MIT
