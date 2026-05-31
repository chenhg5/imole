// Package progress renders download progress for ImageCaptureCore operations.
package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"golang.org/x/term"

	"github.com/chenhg5/imole/internal/human"
)

type Event struct {
	Event      string `json:"event"`
	Index      int    `json:"index"`
	TotalFiles int    `json:"total_files"`
	Path       string `json:"path"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
}

type Writer struct {
	out           io.Writer
	isTTY         bool
	mu            sync.Mutex
	startTime     time.Time
	fileSizes     map[string]int64 // path → expected total bytes
	totalExpected int64            // sum of all file sizes (grows as "start" events arrive)
	doneBytes     int64            // bytes from fully completed files
	maxLineLen    int              // last overwritten line length, for clearing
}

func NewWriter(out io.Writer) *Writer {
	isTTY := false
	if f, ok := out.(interface{ Fd() uintptr }); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}
	return &Writer{
		out:       out,
		isTTY:     isTTY,
		startTime: time.Now(),
		fileSizes: make(map[string]int64),
	}
}

func (w *Writer) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			w.printLine(line)
			continue
		}
		w.print(event)
	}
	return len(p), nil
}

func (w *Writer) print(event Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	name := baseName(event.Path)

	switch event.Event {
	case "start":
		w.fileSizes[event.Path] = event.Total
		w.totalExpected += event.Total
		if w.isTTY {
			w.overwrite(fmt.Sprintf("[%d/%d] %s  starting · %s",
				event.Index, event.TotalFiles, name, human.Bytes(event.Total)))
		} else {
			fmt.Fprintf(w.out, "[%d/%d] %s  starting · %s\n",
				event.Index, event.TotalFiles, name, human.Bytes(event.Total))
		}

	case "progress":
		if w.isTTY {
			pct := 0.0
			if event.Total > 0 {
				pct = float64(event.Downloaded) * 100 / float64(event.Total)
			}
			bar := miniBar(pct, 16)
			speed, eta := w.speedETA(event.Downloaded)
			line := fmt.Sprintf("[%d/%d] %s  %s %3.0f%%  %s  ETA %s",
				event.Index, event.TotalFiles, truncate(name, 22),
				bar, pct, speed, eta)
			w.overwrite(line)
		} else {
			pct := 0.0
			if event.Total > 0 {
				pct = float64(event.Downloaded) * 100 / float64(event.Total)
			}
			fmt.Fprintf(w.out, "[%d/%d] %s  %.0f%% · %s / %s\n",
				event.Index, event.TotalFiles, name,
				pct, human.Bytes(event.Downloaded), human.Bytes(event.Total))
		}

	case "skip":
		if w.isTTY {
			w.clearLine()
			w.doneBytes += event.Total
		}
		fmt.Fprintf(w.out, "[%d/%d] ↷  %s  already verified\n",
			event.Index, event.TotalFiles, name)

	case "done":
		if w.isTTY {
			w.clearLine()
			w.doneBytes += event.Total
		}
		fmt.Fprintf(w.out, "[%d/%d] ✓  %s  %s\n",
			event.Index, event.TotalFiles, name, human.Bytes(event.Total))

	case "error":
		if w.isTTY {
			w.clearLine()
		}
		fmt.Fprintf(w.out, "[%d/%d] ✗  %s  failed\n",
			event.Index, event.TotalFiles, name)

	default:
		w.printLine(fmt.Sprintf("[%d/%d] %s  %s", event.Index, event.TotalFiles, name, event.Event))
	}
}

// overwrite writes s on the current line using \r (TTY only).
func (w *Writer) overwrite(s string) {
	pad := w.maxLineLen - len(s)
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(w.out, "\r%s%s", s, strings.Repeat(" ", pad))
	if len(s) > w.maxLineLen {
		w.maxLineLen = len(s)
	}
}

// clearLine erases the current overwrite line and moves to the start.
func (w *Writer) clearLine() {
	if w.maxLineLen > 0 {
		fmt.Fprintf(w.out, "\r%s\r", strings.Repeat(" ", w.maxLineLen))
		w.maxLineLen = 0
	}
}

// printLine writes a permanent line (non-TTY or terminal events).
func (w *Writer) printLine(s string) {
	fmt.Fprintln(w.out, s)
}

// speedETA returns formatted speed (e.g. "12.3 MB/s") and ETA (e.g. "00:02:15").
// currentFileBytes is the bytes downloaded for the current file so far.
func (w *Writer) speedETA(currentFileBytes int64) (speed, eta string) {
	elapsed := time.Since(w.startTime).Seconds()
	if elapsed < 0.5 {
		return "-- MB/s", "--:--"
	}
	totalDone := w.doneBytes + currentFileBytes
	bytesPerSec := float64(totalDone) / elapsed
	if bytesPerSec < 1 {
		return "0 B/s", "--:--"
	}

	speed = formatSpeed(bytesPerSec)

	remaining := float64(w.totalExpected-totalDone) / bytesPerSec
	if remaining < 0 {
		remaining = 0
	}
	eta = formatDuration(remaining)
	return speed, eta
}

func formatSpeed(bytesPerSec float64) string {
	switch {
	case bytesPerSec >= 1024*1024:
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/1024/1024)
	case bytesPerSec >= 1024:
		return fmt.Sprintf("%.0f KB/s", bytesPerSec/1024)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}

func formatDuration(secs float64) string {
	if secs > 3600*99 {
		return "--:--"
	}
	total := int(secs)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func miniBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100 * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return fmt.Sprintf("%-*s", max, s)
	}
	return s[:max-1] + "…"
}

func baseName(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
