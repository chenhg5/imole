<div align="center">
  <h1>iMole</h1>
  <p><em>🐹 Haz backup y libera espacio en tu iPhone desde la terminal.</em></p>
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

> **Libera espacio en tu iPhone sin comprar más iCloud.** iMole escanea el almacenamiento de tu iPhone, hace backup de fotos y videos a tu computadora, verifica cada archivo y luego elimina los originales de forma segura — todo con un solo comando.

## Inicio Rápido

**Dale esto a un LLM → hace todo automáticamente:**

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

**Instalación**

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

**O: manualmente**

```bash
imole doctor                                           # verificar conexión

imole scan --summary                                   # ver medios y almacenamiento
# Total:   38,421 files · 286.4 GB
# Videos:   1,204 files · 172.8 GB
# Photos:  37,217 files · 113.6 GB

imole scan media --summary                             # solo medios
imole scan --top 10 --only videos                      # mayores videos
imole scan apps --top 20                               # ranking de apps

imole backup --to ~/iphone-backup --file DCIM/202507__/IMG_7523.MOV --dry-run # previsualizar
imole backup --to ~/iphone-backup --only videos --older-than 90d --dry-run   # previsualizar
imole backup --to ~/iphone-backup --only videos --older-than 90d              # backup

imole report --manifest ~/iphone-backup/manifest.json  # confirmar verificación

imole clean  --manifest ~/iphone-backup/manifest.json  # eliminar del iPhone
# → en iPhone: Fotos → Álbumes → Eliminados recientemente → Eliminar todo → espacio liberado 🎉
```

## Características

- **Diagnóstico de espacio** — escanea DCIM por USB, ordena por tamaño, filtra por edad o tipo
- **Ranking de almacenamiento de apps** — muestra uso de App/Datos reportado por iOS con `imole scan apps`
- **Backup inteligente** — copia a cualquier ruta local, organizado por año/mes, verificado por tamaño
- **Manifiesto** — cada backup escribe `manifest.json` con ruta, tamaño y estado de verificación
- **Eliminación segura** — `imole clean` solo elimina archivos con `verified: true` en el manifiesto
- **Multiplataforma** — macOS (ImageCaptureCore), Linux (gphoto2 / ifuse), Windows (`--source PATH`)
- **Amigable para agentes IA** — salida `--json`, selección `--fields`, `imole schema` API legible por máquinas
- **Registro de operaciones** — `imole history` muestra historial de backups y eliminaciones

## Soporte de Plataformas

| Función | macOS | Linux | Windows |
|---------|:-----:|:-----:|:-------:|
| Escaneo USB automático | ✅ ImageCaptureCore | ✅ gphoto2 | ➖ |
| Escaneo vía `--source PATH` | ✅ | ✅ | ✅ |
| Backup (copia + verificación) | ✅ | ✅ | ✅ |
| Eliminación via USB (nativa) | ✅ ImageCaptureCore | ❌ | ❌ |
| Eliminación vía `--source PATH` | ✅ | ✅ ifuse | ✅ iTunes mount |
| Detección de dispositivo | ✅ | ✅ | ✅ |
| Ranking de almacenamiento de apps | ✅ ideviceinstaller | ✅ ideviceinstaller | ➖ |

## Instalación

### npm (recomendado — funciona en macOS, Linux y Windows)

```bash
npm install -g @getimole/imole
```

### Script (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

### Homebrew (macOS)

```bash
brew install imole
```

### Desde código fuente

```bash
go install github.com/chenhg5/imole/cmd/imole@latest
```

## Comandos

```bash
imole doctor                        # Verificar conexión y dependencias
imole scan    [flags]               # Reporte de escaneo (resumen, top N, completo)
imole backup  --to PATH [filters]   # Backup de medios, escribe manifest.json
imole report  --manifest PATH       # Resumir manifiesto de backup
imole clean   --manifest PATH       # Eliminar archivos verificados del iPhone
imole guide   [topic]               # Guía paso a paso de limpieza
imole history [--limit N]           # Mostrar operaciones recientes
imole update  [--check|--nightly]   # Actualizar imole a la última versión
imole schema  [command]             # Schema legible por máquinas (para agentes IA)
```

**Filtros comunes**

```bash
--only all|photos|videos
--older-than 90d|6m|1y
--large-than 500MB|1GB
--ext EXT          # filtrar por extensión: png (capturas), heic, mov
--limit N          # limitar a N resultados (ordenado por tamaño)
--file REL_PATH    # backup: seleccionar archivo; clean:仅限于清单中已验证文件
--json             # forzar salida JSON
--fields a,b       # seleccionar campos JSON (ruta con puntos)
```

## Diseño de Seguridad

iMole trata los medios del iPhone como datos irremplazables, no como caché.

- **Primero previsualizar** — comandos con efectos (`backup`, `clean`) soportan `--dry-run`
- **Escaneos de solo lectura** — `scan` y `scan apps` nunca modifican el dispositivo
- **Protección contra eliminación** — establece `IMOLE_NO_DELETE=1` para bloquear toda eliminación a nivel de entorno. Útil cuando un agente IA ejecuta iMole
- **Backup antes de eliminar** — `clean` requiere `manifest.json`, sin él se niega a ejecutar
- **Verificar antes de eliminar** — solo archivos con `verified: true` en el manifiesto son elegibles para eliminación
- **Registro de auditoría** — `imole history` y `~/.local/share/imole/operations.jsonl` registran cada operación
- **Eliminados recientemente** — cuando se elimina por USB (macOS), los archivos permanecen 30 días en iOS "Eliminados recientemente". Cuando se elimina vía `--source PATH` (Linux/Windows), el espacio se libera inmediatamente
- **Advertencia de iCloud** — si Photos de iCloud está activado, eliminar por iMole también elimina de iCloud. iMole te advierte

## Contribuir

Issues y PRs bienvenidos. Ejecuta `go test ./...` antes de enviar.

## License

MIT