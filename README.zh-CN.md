<div align="center">
  <h1>iMole</h1>
  <p><em>🐹 从终端备份、清理 iPhone 存储空间</em></p>
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

> **不买更多 iCloud 也能释放 iPhone 空间。** iMole 扫描 iPhone 存储占用情况，将照片和视频备份到电脑，验证每个文件，然后安全删除原文件 — 一条命令搞定。

## 快速上手

**把这些扔给 LLM → 它自动完成所有操作：**

```
帮我把手机里超过6个月的照片和视频备份到 ~/backup，然后删除原始文件释放空间
```

```
扫描一下我的iPhone存储空间，看看哪些app最占空间，给我清理建议
```

```
我刚从日本回来，帮我备份所有照片和视频，然后删除原始文件
```

```
帮我从iPhone腾出50GB空间，备份旧视频和照片，然后删除已验证的备份
```

**安装**

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

**或者：手动操作**

```bash
imole doctor                                           # 检查设备连接

imole scan --summary                                   # 查看媒体和应用存储
# Total:   38,421 files · 286.4 GB
# Videos:   1,204 files · 172.8 GB
# Photos:  37,217 files · 113.6 GB

imole scan media --summary                             # 仅媒体摘要
imole scan --top 10 --only videos                      # 找出最大的视频
imole scan apps --top 20                               # 应用存储排行

imole backup --to ~/iphone-backup --file DCIM/202507__/IMG_7523.MOV --dry-run # 预览单个文件
imole backup --to ~/iphone-backup --only videos --older-than 90d --dry-run   # 预览
imole backup --to ~/iphone-backup --only videos --older-than 90d              # 备份

imole report --manifest ~/iphone-backup/manifest.json  # 确认已验证

imole clean  --manifest ~/iphone-backup/manifest.json  # 从 iPhone 删除
# → 在 iPhone 上：照片 → 相簿 → 最近删除 → 全部删除 → 空间释放 🎉
```

## 功能特点

- **空间诊断** — 通过 USB 扫描 DCIM，按大小排序，按时间或类型筛选
- **应用存储排行** — 用 `imole scan apps` 查看 iOS 报告的 App/数据使用量
- **智能备份** — 复制到任意本地路径，按年月整理，验证文件大小
- **增量备份** — 重复备份到同一目录时自动跳过已验证文件，无需重复下载
- **云存储备份** — `--to rclone:<远端>:<路径>` 可同步到 Google Drive、S3、OneDrive、Dropbox 等 70+ 个云盘
- **清单文件** — 每次备份都会生成 `manifest.json`，记录源路径、大小和验证状态
- **安全删除** — `imole clean` 只删除 manifest 中 `verified: true` 的文件
- **跨平台** — macOS (ImageCaptureCore 原生 USB)、Linux (gphoto2 / ifuse)、Windows (`--source PATH`)
- **AI 友好** — `--json` 输出、`--fields` 字段选择、`imole schema` 机器可读 API
- **操作日志** — `imole history` 显示历史备份和删除记录

## 平台支持

| 功能 | macOS | Linux | Windows |
|---------|:-----:|:-----:|:-------:|
| USB 自动扫描 | ✅ ImageCaptureCore | ✅ gphoto2 | ➖ |
| 通过 `--source PATH` 扫描 | ✅ | ✅ | ✅ |
| 备份（复制+验证） | ✅ | ✅ | ✅ |
| 通过 USB 删除（原生） | ✅ ImageCaptureCore | ❌ | ❌ |
| 通过 `--source PATH` 删除 | ✅ | ✅ ifuse | ✅ iTunes 挂载 |
| 设备检测 | ✅ | ✅ | ✅ |
| 应用存储排行 | ✅ ideviceinstaller | ✅ ideviceinstaller | ➖ |

## 安装

### npm（推荐 — 支持 macOS、Linux 和 Windows）

```bash
npm install -g @getimole/imole
```

在所有平台都可以用，Node.js 安装后自动下载预编译二进制。

### 脚本安装（macOS / Linux）

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

### Homebrew（macOS）

```bash
brew install imole
```

### 从源码编译

```bash
go install github.com/chenhg5/imole/cmd/imole@latest
```

## 命令概览

<p align="center">
  <img src="docs/images/imole_screenshot.png" alt="imole --help output" width="800"/>
</p>

## 依赖说明

**macOS** — 媒体扫描/备份无需额外安装。ImageCaptureCore 是系统内置的。设备详情和应用存储信息需要：

```shell
brew install libimobiledevice   # 可选，用于 imole doctor 获取设备详情
brew install ideviceinstaller    # 可选，用于 imole scan apps
```

如果 `ideviceinstaller` 不存在，`imole scan --summary` 仍会显示媒体摘要，只是应用存储显示不可用。只有 `imole scan apps` 需要 `ideviceinstaller`。

**Linux**

```shell
sudo apt install libimobiledevice-utils gphoto2   # USB 扫描
sudo apt install ifuse                             # 挂载 DCIM 为文件系统
```

> **通过 ifuse 完整备份+删除流程：**
> ```shell
> idevicepair pair                                  # 一次性配对信任
> mkdir -p ~/iphone && ifuse ~/iphone               # 挂载
> imole backup --source ~/iphone/DCIM --to ~/iphone-backup
> imole clean  --manifest ~/iphone-backup/manifest.json --source ~/iphone/DCIM
> fusermount -u ~/iphone                            # 用完后卸载
> ```

**Windows** — 安装 iTunes（提供 USB 驱动并将 iPhone 挂载为可浏览设备）：

> **1.** 安装 iTunes，连接 iPhone，解锁并点击"信任此电脑"
> **2.** 打开文件资源管理器 → 此电脑 → [iPhone] → 内部存储 → DCIM
> **3.** 记下地址栏中的路径，例如 `\\Apple\iPhone\Internal Storage\DCIM`

```powershell
# 扫描
imole.exe scan --source "\\Apple\iPhone\Internal Storage\DCIM"

# 备份
imole.exe backup --source "\\Apple\iPhone\Internal Storage\DCIM" --to C:\iphone-backup

# 删除已验证文件（立即释放空间）
imole.exe clean --manifest C:\iphone-backup\manifest.json --source "\\Apple\iPhone\Internal Storage\DCIM"
```

## 命令

```bash
imole doctor                        # 检查设备连接和依赖
imole scan    [flags]               # 扫描报告（摘要、前 N 名或完整）
imole backup  --to PATH [filters]   # 备份匹配媒体，写入 manifest.json
imole report  --manifest PATH       # 汇总备份清单
imole clean   --manifest PATH       # 从 iPhone 删除已验证文件
imole guide   [topic]               # 分步清理指南（微信、Telegram...）
imole history [--limit N]           # 显示最近备份和删除操作
imole update  [--check|--nightly]   # 更新 imole 到最新版本
imole schema  [command]             # 机器可读命令 schema（AI 友好）
```

**常用筛选**

```bash
--only all|photos|videos
--older-than 90d|6m|1y
--large-than 500MB|1GB
--ext EXT          # 按文件扩展名筛选，如 png（截图）、heic、mov
--limit N          # 筛选后限制数量（按大小排序）
--file REL_PATH    # backup: 选择文件；clean: 限制为清单中的已验证文件；可重复
--json             # 强制 JSON 输出
--fields a,b       # 选择 JSON 字段（点号路径）
```

**元数据筛选** — 需要 `--with-meta`（获取 EXIF；首次运行约 30-60 秒，结果缓存 7 天）

```bash
--with-meta                      # 启用元数据获取
--country NAME                   # 按 GPS 解析的国家/地区筛选，如 Japan、CN、Asia
--no-gps                         # 保留没有 GPS 坐标的文件
--taken-after / --taken-before   # 日期范围，如 --taken-after 2024-01-01
--duration-gt N                  # 时长大于 N 秒的视频
--min-width / --max-width N      # 像素宽度范围
--min-height / --max-height N    # 像素高度范围
```

**预览标志**

```bash
--dry-run        # backup 和 clean 支持；scan 是只读的，不接受此参数
```

## 使用示例

### 诊断存储占用

```bash
$ imole scan --summary

iMole Stats

Total:   38,421 files · 286.4 GB
Photos:  37,217 files · 113.6 GB
Videos:   1,204 files · 172.8 GB

$ imole scan --top 5 --only videos

Top 5 Videos

   1. IMG_8821.MOV              8.2 GiB  2025-10-02
   2. IMG_7731.MOV              4.6 GiB  2025-08-11
   3. IMG_6602.MOV              3.9 GiB  2024-12-31
   4. IMG_5501.MOV              2.1 GiB  2024-09-15
   5. IMG_4412.MOV              1.8 GiB  2024-06-20
```

### 备份旧视频并从设备删除

```bash
# 1. 预览将要备份的内容
$ imole backup --to ~/iphone-backup --only videos --older-than 90d --dry-run
Dry-run: 48 files (62.4 GB) would be copied (exit 10)

# 2. 执行备份
$ imole backup --to ~/iphone-backup --only videos --older-than 90d
Backup complete
Destination: /Users/you/iphone-backup
Selected:    48 files · 62.4 GB
Copied:      48 files · 62.4 GB
Verified:    48 files · 62.4 GB
Manifest:    /Users/you/iphone-backup/manifest.json

# 3. 从 iPhone 删除已验证文件
$ imole clean --manifest ~/iphone-backup/manifest.json
Clean plan

Manifest:       /Users/you/iphone-backup/manifest.json
Verified files: 48 (62.4 GB)

Files to delete (showing 15 of 48):
    1. IMG_8821.MOV                          8.2 GB
    2. IMG_7731.MOV                          4.6 GB
    ...

Warning: This will delete the files listed above from your iPhone.
         iMole only deletes files verified in the manifest.
         Files will remain in Recently Deleted for 30 days.

Proceed? [y/N] y
Deleting 48 files via auto provider...

Delete complete
  Deleted: 48 files · 62.4 GB

Final step to reclaim space:
  On iPhone → Photos → Albums → Recently Deleted → Delete All
  Estimated space freed after that step: ~62.4 GB
```

### 备份到云存储（rclone）

```bash
# 安装并配置 rclone：https://rclone.org/install/
rclone config   # 添加远端，例如 "gdrive"（Google Drive）或 "s3"（AWS S3）

# 使用 --to rclone:<远端名>:<远端路径>
imole backup --to rclone:gdrive:iPhone/backup --only videos --older-than 90d
imole backup --to rclone:s3:my-bucket/iphone --only photos
imole backup --to rclone:onedrive:iPhone --dry-run
```

imole 先将文件备份到本地 staging 目录（`~/.imole/rclone-cache/`），验证后调用 `rclone copy` 推送到云端。
支持 Google Drive、S3、OneDrive、Dropbox、Backblaze B2、SFTP 等 70+ 种云存储。

### 查看 iMole 操作历史

```bash
$ imole history

iMole Operation History

  2026-05-31 02:41  backup   48 files · 62.4 GB → ~/iphone-backup
  2026-05-31 02:45  clean    48 files · 62.4 GB  [manifest: ~/iphone-backup/manifest.json]

$ imole history --json | jq '.[0]'
```

### 非交互式使用（脚本 / AI）

```bash
# 机器可读的统计
imole scan --summary --json --fields total_size_human,video_files,old_size_human

# 前 N 个视频的 JSON
imole scan --top 20 --only videos --json

# 使用缓存跳过慢速 USB 扫描
imole scan --cache --summary --json

# 完整备份+清理流程，无提示
imole backup --to ~/backup --only videos --older-than 90d
imole clean  --manifest ~/backup/manifest.json --yes

# 发现可用参数
imole schema scan
imole schema backup

# 推荐的分析流程
imole guide analysis
```

AI 应该先用 `imole schema <command>` 了解可用参数，再组合命令。不要对只读命令（`scan`、`scan apps`、`doctor`、`report`、`history`、`schema`、`guide`）使用 `--dry-run`。

### 安全地让 AI agent 驱动 iMole

在启动 agent 会话前设置 `IMOLE_NO_DELETE`。agent 可以自由扫描、备份、查看报告和历史记录，但 `imole clean` 会拒绝运行并返回结构化错误。只有人类可以取消这个限制。

```bash
# 在 shell 配置文件或启动 agent 前：
export IMOLE_NO_DELETE=1

# agent 现在可以安全运行：
imole doctor
imole scan
imole scan --summary --json
imole backup --to ~/backup --only videos --older-than 90d
imole report --manifest ~/backup/manifest.json

# 这会被阻止 — clean 返回错误码 1：
imole clean --manifest ~/backup/manifest.json
# error: IMOLE_NO_DELETE is set — deletion is disabled in this environment
# hint:  Unset IMOLE_NO_DELETE if you want to allow deletion: unset IMOLE_NO_DELETE

# 准备删除时，取消设置后手动运行：
unset IMOLE_NO_DELETE
imole clean --manifest ~/backup/manifest.json
```

Agent 分析流程：

1. 运行 `imole doctor --json` 查看 `device.storage.free_percent`
2. 判断压力等级：`<5%` 紧急，`5-10%` 高，`10-20%` 中等，`>20%` 低
3. 运行 `imole scan --summary --json` 比较媒体、视频和应用估算
4. 如果视频能满足目标，先从风险最低的筛选开始：旧视频 → 大视频 → 所有视频
5. 如果应用存储占主导，运行 `imole scan apps --top 20 --json` 并推荐应用特定的清理路径。不要声称 iMole 能直接清理微信等私有的 app 缓存
6. 只对有副作用的命令使用 dry-run：`backup --dry-run`，然后 `clean --dry-run`

同样的流程可以直接从 CLI 运行：

```bash
imole guide analysis
```

## 安全设计

iMole 将 iPhone 媒体视为不可替代的数据，而非缓存。

- **先预览** — 有副作用的命令（`backup`、`clean`）支持 `--dry-run`
- **只读扫描** — `scan` 和 `scan apps` 绝不修改设备，不接受 `--dry-run`
- **删除保护** — 设置 `IMOLE_NO_DELETE=1` 可在环境级别阻止所有删除。AI agent 运行时特别有用：agent 可以自由扫描和备份，但无法删除，除非人类明确取消设置
- **先备份后删除** — `clean` 需要读取 `manifest.json`，没有清单文件会拒绝运行
- **验证后再删除** — 只有 manifest 中标记 `verified: true` 的文件才能删除
- **操作审计** — `imole history` 和 `~/.local/share/imole/operations.jsonl` 记录每次备份和删除
- **最近删除** — 通过 USB 删除（macOS）时，文件会在 iOS"最近删除"中保留 30 天；iMole 会提醒你清理。通过 `--source PATH`（Linux/Windows 文件系统挂载）删除时，空间立即释放
- **iCloud 警告** — 如果开启了 iCloud 照片，通过 iMole 删除也会从 iCloud 删除。iMole 会警告你

iMole 无法自动清理：

- 微信、Telegram 或其他应用沙盒存储（用 `imole guide` 获取分步说明）
- iOS 系统数据
- 仅存在 iCloud 的内容（未下载到设备）

## 使用技巧

- **先从视频开始** — 一个 4K 视频可能比上千张照片还大。先运行 `imole scan --top 20 --only videos`
- **用 `--dry-run` 预览 backup/clean** — 提交之前总是先预览。退出码 `10` 表示预览通过
- **缩小筛选范围** — `--only videos --older-than 1y` 用最低风险回收最多空间
- **iCloud 用户** — 如果 iCloud 照片同步开着，通过 iMole 删除也会从 iCloud 删除。先备份
- **Linux/Windows** — 先挂载 iPhone DCIM 文件夹（Linux 用 ifuse，Windows 用 iTunes），然后传 `--source PATH`
- **找截图** — iPhone 截图总是保存为 PNG，相机照片是 HEIC 或 JPEG。用 `--ext png` 定位。结合 `--min-width` 和 `--min-height`（需要 `--with-meta`）可以精确匹配屏幕尺寸来识别。参见下面的[识别和备份截图](#识别和备份截图)

### 识别和备份截图

iPhone 截图一定是 `.png`；相机照片是 `.heic` 或 `.jpeg`。所以 `--ext png` 是可靠的第一层筛选。但偶尔也有通过 AirDrop 或消息收到的 PNG，所以是高度可信而非绝对。

要近乎精确识别，结合屏幕尺寸筛选（需要 `--with-meta`）：

| 设备 | 屏幕分辨率 |
|---|---|
| iPhone 16 Pro | 1206 × 2622 |
| iPhone 15 Pro | 1179 × 2556 |
| iPhone 14 / 15 | 1170 × 2532 |
| iPhone SE (3rd gen) | 750 × 1334 |

```bash
# 步骤 1 — 快速统计（无需元数据）
imole scan --ext png --json

# 步骤 2 — 用屏幕尺寸精确匹配（获取 EXIF，首次运行后缓存）
imole scan --ext png --min-width 1100 --min-height 2400 --json

# 步骤 3 — 清理前先备份截图
imole backup --to ~/iphone-backup/screenshots --ext png --dry-run
imole backup --to ~/iphone-backup/screenshots --ext png

# 步骤 4 — 用尺寸精确备份
imole backup --to ~/iphone-backup/screenshots --ext png --min-width 1100 --min-height 2400 --dry-run
imole backup --to ~/iphone-backup/screenshots --ext png --min-width 1100 --min-height 2400
```

## 鸣谢

iMole 灵感来自 [Mole](https://github.com/tw93/mole) — @tw93 开发的出色 macOS 系统清理工具。Mole 证明了一个 CLI 二进制文件可以替代笨重的 GUI 应用进行系统维护，其 agent 友好的设计理念深刻影响了 iMole 的构建方式。如果你想清理 Mac，值得一试。

## 贡献

欢迎 Issues 和 PR。提交前请运行 `go test ./...`。

## License

MIT