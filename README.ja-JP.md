<div align="center">
  <h1>iMole</h1>
  <p><em>🐹 ターミナルからiPhoneストレージをバックアップ・整理</em></p>
  <p style="font-size:1.1em; color:#aaaaaa;">Inspired by <a href="https://github.com/tw93/mole">Mole</a></p>
</div>

<p align="center">
  <img src="docs/images/mole_with_iphone.png" alt="iMole with iPhone" width="400"/>
</p>

<p align="center">
  <a href="https://github.com/chenhg5/imole/stargazers"><img src="https://img.shields.io/github/stars/chenhg5/imole?style=flat-square" alt="Stars"></a>
  <a href="https://github.com/chenhg5/imole/releases"><img src="https://img.shields.io/github/v/tag/chenhg5/imole?label=version&style=flat-square" alt="Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square" alt="License"></a>
  <a href="https://github.com/chenhg5/imole/commits"><img src="https://img.shields.io/github/commit-activity/m/chenhg5/imole?style=flat-square" alt="Commits"></a>
  <a href="https://t.me/+ZpgBu1dlmCszODBl"><img src="https://img.shields.io/badge/chat-Telegram-blue?style=flat-square&logo=Telegram" alt="Telegram"></a>
</p>

<p align="center">
  <a href="./README.md">English</a> | <a href="./README.zh-CN.md">中文</a> | <a href="./README.ja-JP.md">日本語</a> | <a href="./README.es-ES.md">Español</a> | <a href="./README.zh-TW.md">繁體中文</a>
</p>

> **iCloudを追加購入せずにiPhoneストレージを解放。** iMoleはiPhoneストレージ消費状況をスキャンし、写真と動画をPCにバックアップ、ファイルを検証してから安全に削除 — 1コマンドで完了。

## クイックスタート

**これをLLMに渡す → 全部自動処理：**

```
iPhoneで6ヶ月より古い写真と動画を~/backupにバックアップして、元のファイルを削除して容量を解放して
```

```
iPhoneのストレージをスキャンして、どこが最も容量を使っているか確認し、削除して大丈夫なものを教えて
```

```
日本から帰ってきたばかり — すべての写真と動画をバックアップして、iPhoneから元のファイルを削除して
```

```
iPhoneから50GB解放して古い動画と写真をバックアップして、検証済みバックアップを削除して
```

**インストール**

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

**または：手動操作**

```bash
imole doctor                                           # デバイス接続確認

imole scan --summary                                   # メディアとアプリストレージ確認
# Total:   38,421 files · 286.4 GB
# Videos:   1,204 files · 172.8 GB
# Photos:  37,217 files · 113.6 GB

imole scan media --summary                             # メディアのみ要約
imole scan --top 10 --only videos                      # 最大動画を探す
imole scan apps --top 20                               # アプリストレージランキング

imole backup --to ~/iphone-backup --file DCIM/202507__/IMG_7523.MOV --dry-run # 1ファイルプレビュー
imole backup --to ~/iphone-backup --only videos --older-than 90d --dry-run   # プレビュー
imole backup --to ~/iphone-backup --only videos --older-than 90d              # バックアップ

imole report --manifest ~/iphone-backup/manifest.json  # 検証済み確認

imole clean  --manifest ~/iphone-backup/manifest.json  # iPhoneから削除
# → iPhoneで：写真 → アルバム → 最近削除 → すべて削除 → 容量解放 🎉
```

## 機能

- **容量診断** — USBでDCIMをスキャン、サイズ順・日時/種類でソート
- **アプリストレージランキング** — `imole scan apps` でiOS報告のApp/データ使用量を表示
- **スマートバックアップ** — 任意パスにコピー、年/月整理、サイズ検証
- **マニフェスト** — バックアップごとに `manifest.json` を生成（パス、サイズ、検証状態記録）
- **安全削除** — `imole clean` はmanifestで `verified: true` のファイルのみ削除
- **クロスプラットフォーム** — macOS (ImageCaptureCore)、Linux (gphoto2 / ifuse)、Windows (`--source PATH`)
- **AI対応** — `--json`出力、`--fields`フィールド選択、`imole schema`機械可読API
- **操作ログ** — `imole history` でバックアップ/削除履歴を表示

## プラットフォーム対応

| 機能 | macOS | Linux | Windows |
|---------|:-----:|:-----:|:-------:|
| USB自動スキャン | ✅ ImageCaptureCore | ✅ gphoto2 | ➖ |
| `--source PATH` スキャン | ✅ | ✅ | ✅ |
| バックアップ（複製+検証） | ✅ | ✅ | ✅ |
| USB削除（ネイティブ） | ✅ ImageCaptureCore | ❌ | ❌ |
| `--source PATH` 削除 | ✅ | ✅ ifuse | ✅ iTunes マウント |
| デバイス検出 | ✅ | ✅ | ✅ |
| アプリストレージランキング | ✅ ideviceinstaller | ✅ ideviceinstaller | ➖ |

## インストール

### npm（推奨 — macOS、Linux、Windows対応）

```bash
npm install -g @getimole/imole
```

Node.js導入済みの全プラットフォームで動作、自动でバイナリをダウンロード。

### スクリプト（macOS / Linux）

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

### Homebrew（macOS）

```bash
brew install imole
```

### ソースからビルド

```bash
go install github.com/chenhg5/imole/cmd/imole@latest
```

## コマンド概要

<p align="center">
  <img src="docs/images/imole_screenshot.png" alt="imole --help output" width="800"/>
</p>

## 依存関係

**macOS** — メディアスキャン/バックアップに追加インストール不要。ImageCaptureCoreはシステム組み込み。デバイス詳報とアプリストレージ情報：

```shell
brew install libimobiledevice   # 任意、imole doctor のデバイス詳報用
brew install ideviceinstaller    # 任意、imole scan apps 用
```

`ideviceinstaller`がない場合、`imole scan --summary` はメディアサマリーを表示するがアプリストレージは利用不可と表示。`imole scan apps` のみ必要。

**Linux**

```shell
sudo apt install libimobiledevice-utils gphoto2   # USBスキャン
sudo apt install ifuse                             # DCIMをファイルシステムとしてマウント
```

**Windows** — iTunesをインストール（USBドライバー提供、iPhoneをブラウズ可能に）：

```powershell
# スキャン
imole.exe scan --source "\\Apple\iPhone\Internal Storage\DCIM"

# バックアップ
imole.exe backup --source "\\Apple\iPhone\Internal Storage\DCIM" --to C:\iphone-backup

# 検証済みファイル削除（即座に容量解放）
imole.exe clean --manifest C:\iphone-backup\manifest.json --source "\\Apple\iPhone\Internal Storage\DCIM"
```

## コマンド

```bash
imole doctor                        # デバイス接続と依存関係チェック
imole scan    [flags]               # スキャンレポート（サマリー、上位N、完整）
imole backup  --to PATH [filters]   # マッチしたメディアをバックアップ、manifest.json生成
imole report  --manifest PATH       # バックアップマニフェストを要約
imole clean   --manifest PATH       # iPhoneから検証済みファイルを削除
imole guide   [topic]               # ステップバイステップ清理ガイド
imole history [--limit N]           # 最近のバックアップ/削除操作を表示
imole update  [--check|--nightly]   # imoleを最新リリースに更新
imole schema  [command]             # 機械可読コマンドschema（AI対応）
```

**共通フィルター**

```bash
--only all|photos|videos
--older-than 90d|6m|1y
--large-than 500MB|1GB
--ext EXT          # ファイル拡張子でフィルター、png（スクリーンショット）、heic、movなど
--limit N          # フィルター後の制限数（サイズ順）
--file REL_PATH    # backup: ファイル選択; clean: マニフェスト内の検証済みファイルに制限; 繰り返し可
--json             # JSON出力を強制
--fields a,b       # JSONフィールド選択（ドットパス）
```

## 安全設計

iMoleはiPhoneメディアを代替不可能なデータとして扱う。

- **まずプレビュー** — 副作用のあるコマンド（`backup`、`clean`）は `--dry-run` 対応
- **読み取り専用スキャン** — `scan` と `scan apps` はデバイスを変更しない、`--dry-run` 不可
- **削除保護** — `IMOLE_NO_DELETE=1` で環境レベルで全削除をブロック。AI agent動作時に有用
- **削除前にバックアップ** — `clean` は `manifest.json` 必須、なしでは動作しない
- **検証後削除** — manifestで `verified: true` のファイルのみ削除可能
- **監査証跡** — `imole history` と `~/.local/share/imole/operations.jsonl` が全操作を記録
- **最近の削除** — USB削除（macOS）はiOS「最近の削除」に30日間残留。iMoleがリマインダー表示。`--source PATH`（Linux/Windowsファイルシステムmount）削除は即座に容量解放
- **iCloud警告** — iCloud Photos有効の場合、iMole削除はiCloudからも削除。iMoleが警告

## 貢献

IssuesとPR大歓迎。提交前に `go test ./...` を実行。

## License

MIT