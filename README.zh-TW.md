<div align="center">
  <h1>iMole</h1>
  <p><em>🐹 從終端備份、整理 iPhone 儲存空間</em></p>
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

> **不買更多 iCloud 也能釋放 iPhone 空間。** iMole 掃描 iPhone 儲存占用情況，將照片和影片備份到電腦，驗證每個檔案，然後安全刪除原始檔案 — 一個命令搞定。

## 快速上手

**把這些丟給 LLM → 它自動完成所有操作：**

```
Back up all photos and videos older than 6 months from my iPhone to ~/backup,
then delete the originals to free up space
```

```
Scan my iPhone storage and tell me which apps are taking up the most space,
then suggest what I can safely remove
```

```
I just got back from Japan — back up all my photos and videos and delete
the originals from my iPhone
```

```
Free up 50GB from my iPhone by backing up old videos and photos, then
deleting the verified backups
```

**安裝**

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

**或者：手動操作**

```bash
imole doctor                                           # 檢查裝置連接

imole scan --summary                                   # 檢視媒體和應用儲存
# Total:   38,421 files · 286.4 GB
# Videos:   1,204 files · 172.8 GB
# Photos:  37,217 files · 113.6 GB

imole scan media --summary                             # 僅媒體摘要
imole scan --top 10 --only videos                      # 找出最大的影片
imole scan apps --top 20                               # 應用儲存排行

imole backup --to ~/iphone-backup --file DCIM/202507__/IMG_7523.MOV --dry-run # 預覽單個檔案
imole backup --to ~/iphone-backup --only videos --older-than 90d --dry-run   # 預覽
imole backup --to ~/iphone-backup --only videos --older-than 90d              # 備份

imole report --manifest ~/iphone-backup/manifest.json  # 確認已驗證

imole clean  --manifest ~/iphone-backup/manifest.json  # 從 iPhone 刪除
# → 在 iPhone 上：照片 → 相簿 → 最近刪除 → 全部刪除 → 空間釋放 🎉
```

## 功能特點

- **空間診斷** — 透過 USB 掃描 DCIM，按大小排序，按時間或類型篩選
- **應用儲存排行** — 用 `imole scan apps` 查看 iOS 報告的 App/資料使用量
- **智慧備份** — 複製到任意本機路徑，按年月整理，驗證檔案大小
- **清單檔案** — 每次備份都會生成 `manifest.json`，記錄來源路徑、大小和驗證狀態
- **安全刪除** — `imole clean` 只刪除 manifest 中 `verified: true` 的檔案
- **跨平台** — macOS (ImageCaptureCore 原生 USB)、Linux (gphoto2 / ifuse)、Windows (`--source PATH`)
- **AI 友好** — `--json` 輸出、`--fields` 欄位選擇、`imole schema` 機器可讀 API
- **操作日誌** — `imole history` 顯示歷史備份和刪除記錄

## 平台支援

| 功能 | macOS | Linux | Windows |
|---------|:-----:|:-----:|:-------:|
| USB 自動掃描 | ✅ ImageCaptureCore | ✅ gphoto2 | ➖ |
| 透過 `--source PATH` 掃描 | ✅ | ✅ | ✅ |
| 備份（複製+驗證） | ✅ | ✅ | ✅ |
| 透過 USB 刪除（原生） | ✅ ImageCaptureCore | ❌ | ❌ |
| 透過 `--source PATH` 刪除 | ✅ | ✅ ifuse | ✅ iTunes 掛載 |
| 裝置偵測 | ✅ | ✅ | ✅ |
| 應用儲存排行 | ✅ ideviceinstaller | ✅ ideviceinstaller | ➖ |

## 安裝

### npm（推薦 — 支援 macOS、Linux 和 Windows）

```bash
npm install -g @getimole/imole
```

在所有平台都可以用，Node.js 安裝後自動下載預編譯二元檔。

### 指令稿安裝（macOS / Linux）

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

### Homebrew（macOS）

```bash
brew install imole
```

### 從原始碼編譯

```bash
go install github.com/chenhg5/imole/cmd/imole@latest
```

## 命令概覽

<p align="center">
  <img src="docs/images/imole_screenshot.png" alt="imole --help output" width="800"/>
</p>

## 依賴說明

**macOS** — 媒體掃描/備份無需額外安裝。ImageCaptureCore 是系統內建的。裝置詳情和應用儲存資訊需要：

```shell
brew install libimobiledevice   # 任意，用於 imole doctor 取得裝置詳情
brew install ideviceinstaller    # 任意，用於 imole scan apps
```

如果 `ideviceinstaller` 不存在，`imole scan --summary` 仍會顯示媒體摘要，只是應用儲存顯示不可用。只有 `imole scan apps` 需要 `ideviceinstaller`。

**Linux**

```shell
sudo apt install libimobiledevice-utils gphoto2   # USB 掃描
sudo apt install ifuse                             # 掛載 DCIM 為檔案系統
```

**Windows** — 安裝 iTunes（提供 USB 驅動並將 iPhone 掛載為可瀏覽裝置）：

```powershell
# 掃描
imole.exe scan --source "\\Apple\iPhone\Internal Storage\DCIM"

# 備份
imole.exe backup --source "\\Apple\iPhone\Internal Storage\DCIM" --to C:\iphone-backup

# 刪除已驗證檔案（立即釋放空間）
imole.exe clean --manifest C:\iphone-backup\manifest.json --source "\\Apple\iPhone\Internal Storage\DCIM"
```

## 命令

```bash
imole doctor                        # 檢查裝置連接和依賴
imole scan    [flags]               # 掃描報告（摘要、前 N 名或完整）
imole backup  --to PATH [filters]   # 備份匹配媒體，寫入 manifest.json
imole report  --manifest PATH       # 摘要備份清單
imole clean   --manifest PATH       # 從 iPhone 刪除已驗證檔案
imole guide   [topic]               # 逐步清理指南
imole history [--limit N]           # 顯示最近備份和刪除操作
imole update  [--check|--nightly]   # 更新 imole 到最新版本
imole schema  [command]             # 機器可讀命令 schema（AI 友好）
```

**常用篩選**

```bash
--only all|photos|videos
--older-than 90d|6m|1y
--large-than 500MB|1GB
--ext EXT          # 按檔案副檔名篩選，如 png（截圖）、heic、mov
--limit N          # 篩選後限制數量（按大小排序）
--file REL_PATH    # backup: 選擇檔案；clean: 限制為清單中的已驗證檔案；可重複
--json             # 強制 JSON 輸出
--fields a,b       # 選擇 JSON 欄位（點號路徑）
```

## 安全設計

iMole 將 iPhone 媒體視為不可替代的資料，而非快取。

- **先預覽** — 有副作用的命令（`backup`、`clean`）支援 `--dry-run`
- **唯讀掃描** — `scan` 和 `scan apps` 絕不修改裝置，不接受 `--dry-run`
- **刪除保護** — 設定 `IMOLE_NO_DELETE=1` 可在環境層級阻止所有刪除。AI agent 執行時特別有用
- **先備份後刪除** — `clean` 需要讀取 `manifest.json`，沒有清單檔案會拒絕執行
- **驗證後刪除** — 只有 manifest 中標記 `verified: true` 的檔案才能刪除
- **操作審計** — `imole history` 和 `~/.local/share/imole/operations.jsonl` 記錄每次備份和刪除
- **最近刪除** — 透過 USB 刪除（macOS）時，檔案會在 iOS「最近刪除」中保留 30 天。透過 `--source PATH`（Linux/Windows 檔案系統掛載）刪除時，空間立即釋放
- **iCloud 警告** — 如果開啟了 iCloud 照片，透過 iMole 刪除也會從 iCloud 刪除。iMole 會警告你

## 貢獻

歡迎 Issues 和 PR。提交前請執行 `go test ./...`。

## License

MIT