package cli

import (
	"fmt"
	"strings"
)

// ANSI escape codes. Only applied when a.isTTY is true, so agent/pipe output
// is always plain text.
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiCyan    = "\033[36m"
	ansiRed     = "\033[31m"
	ansiMagenta = "\033[35m"
	ansiBlue    = "\033[34m"
)

func (a *App) bold(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiBold + s + ansiReset
}

func (a *App) dim(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiDim + s + ansiReset
}

func (a *App) green(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiGreen + s + ansiReset
}

func (a *App) yellow(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiYellow + s + ansiReset
}

func (a *App) cyan(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiCyan + s + ansiReset
}

func (a *App) red(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiRed + s + ansiReset
}

func (a *App) magenta(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiMagenta + s + ansiReset
}

func (a *App) blue(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiBlue + s + ansiReset
}

func (a *App) boldCyan(s string) string {
	if !a.isTTY {
		return s
	}
	return ansiBold + ansiCyan + s + ansiReset
}

// check returns a colored ✓ or ✗ symbol.
func (a *App) check(ok bool) string {
	if ok {
		return a.green("✓")
	}
	return a.red("✗")
}

// progressBar returns a visual progress bar string (e.g. "████░░░░░░░░░░░░ 25%").
// Width is 16 blocks. Bar color is green (<50%), yellow (50-80%), red (>80%).
func (a *App) progressBar(percent float64) string {
	total := 16
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(percent / 100 * float64(total))
	empty := total - filled

	var bar string
	if a.isTTY {
		color := a.green
		if percent >= 80 {
			color = a.red
		} else if percent >= 50 {
			color = a.yellow
		}
		bar = color(strings.Repeat("█", filled) + strings.Repeat("░", empty))
	} else {
		bar = strings.Repeat("█", filled) + strings.Repeat("░", empty)
	}
	return fmt.Sprintf("%s %5.1f%%", bar, percent)
}

// status prints a status line to stderr. Suppressed only when NO_COLOR is set
// (which signals fully non-interactive / agent mode). ANSI dim is applied only
// when stdout is also a TTY (same color gate as other output helpers).
func (a *App) status(msg string) {
	if a.showStatus {
		fmt.Fprintln(a.err, a.dim(msg))
	}
}
