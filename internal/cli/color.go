package cli

// ANSI escape codes. Only applied when a.isTTY is true, so agent/pipe output
// is always plain text.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiRed    = "\033[31m"
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

// check returns a colored ✓ or ✗ symbol.
func (a *App) check(ok bool) string {
	if ok {
		return a.green("✓")
	}
	return a.red("✗")
}
