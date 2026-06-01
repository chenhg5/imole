// Package cli provides the iMole command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// Version is the current imole version.
// It can be overridden at build time with:
//
//	go build -ldflags="-X github.com/chenhg5/imole/internal/cli.Version=x.y.z"
var Version = "0.1.0"

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type App struct {
	out        io.Writer
	err        io.Writer
	in         io.Reader // stdin for interactive prompts
	isTTY      bool      // stdout looks interactive → human-readable output + color
	showStatus bool      // stderr should show progress messages
	debugMode  bool      // --debug: verbose output to stderr
}

func New(out, err io.Writer) *App {
	tty := isTerminal(out)
	return &App{
		out:        out,
		err:        err,
		in:         os.Stdin,
		isTTY:      tty,
		showStatus: shouldShowStatus(),
	}
}

// startSpinner starts an animated spinner on stderr while a long operation runs.
// It returns a stop function: call stop("final message") to clear the spinner
// and optionally print a final status line. Noop when not a TTY.
func (a *App) startSpinner(msg string) func(finalMsg string) {
	if !a.isTTY || !a.showStatus {
		if a.showStatus {
			fmt.Fprintln(a.err, a.dim(msg))
		}
		return func(string) {}
	}

	// Truncate msg so the full spinner line fits on one terminal row.
	// Prefix "⠋ " = 2 visible chars + ANSI codes; reserve 4 chars of margin.
	const prefixWidth = 2
	const margin = 4
	maxMsg := spinnerMsgWidth(msg, prefixWidth+margin)

	stopCh := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case finalMsg := <-stopCh:
				// \r\033[K — go to col 0, erase to end of line (ANSI, no width tracking needed).
				fmt.Fprintf(a.err, "\r\033[K")
				if finalMsg != "" {
					fmt.Fprintln(a.err, a.dim(finalMsg))
				}
				return
			default:
				fmt.Fprintf(a.err, "\r%s %s\033[K", a.cyan(spinnerFrames[i%len(spinnerFrames)]), a.dim(maxMsg))
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	return func(finalMsg string) {
		stopCh <- finalMsg
		wg.Wait()
	}
}

// spinnerMsgWidth truncates msg so that (prefixWidth + len(msg)) fits within
// the current terminal column width. Falls back to 72 columns if unavailable.
func spinnerMsgWidth(msg string, prefixWidth int) string {
	cols := 72
	if w, _, err := term.GetSize(int(os.Stderr.Fd())); err == nil && w > 0 {
		cols = w
	}
	maxLen := cols - prefixWidth
	if maxLen < 10 {
		maxLen = 10
	}
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-1] + "…"
}

// debug writes a verbose debug line to stderr when --debug is active.
func (a *App) debug(format string, args ...any) {
	if a.debugMode {
		msg := a.dim("[debug] " + fmt.Sprintf(format, args...))
		fmt.Fprintln(a.err, msg)
	}
}

// shouldShowStatus reports whether progress/status messages should be written
// to stderr. Unlike isTTY (which also respects NO_COLOR for output format),
// progress messages are shown whenever a human might be watching stderr —
// unless NO_COLOR is set (which signals fully non-interactive / agent usage).
func shouldShowStatus() bool {
	// FORCE_COLOR wins: human explicitly wants interactive output.
	if v := os.Getenv("FORCE_COLOR"); v != "" && v != "0" {
		return true
	}
	// NO_COLOR signals fully non-interactive / agent mode.
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	// A real PTY on stderr means a human is watching.
	return term.IsTerminal(int(os.Stderr.Fd()))
}

// isTerminal reports whether w looks like an interactive terminal.
// Priority order:
//  1. FORCE_COLOR env var set → true  (explicit human override, wins over everything)
//  2. NO_COLOR env var set → false    (https://no-color.org/, signals agent/non-interactive)
//  3. TERM=dumb → false
//  4. golang.org/x/term PTY check
func isTerminal(w io.Writer) bool {
	if v := os.Getenv("FORCE_COLOR"); v != "" && v != "0" {
		return true
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.runHelp()
		return ExitSuccess
	}

	// Strip --debug from args before dispatch so it works with any subcommand.
	filtered := args[:0:len(args)]
	for _, arg := range args {
		if arg == "--debug" {
			a.debugMode = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	args = filtered

	switch args[0] {
	case "schema":
		return a.runSchema(ctx, args[1:])
	case "doctor":
		return a.runDoctor(ctx, args[1:])
	case "scan":
		return a.runScan(ctx, args[1:])
	case "backup":
		return a.runBackup(ctx, args[1:])
	case "report":
		return a.runReport(ctx, args[1:])
	case "guide":
		return a.runGuide(ctx, args[1:])
	case "clean":
		return a.runClean(ctx, args[1:])
	case "uninstall":
		return a.runUninstall(ctx, args[1:])
	case "history":
		return a.runHistory(ctx, args[1:])
	case "update":
		return a.runUpdate(ctx, args[1:])
	case "completion":
		return a.runCompletion(ctx, args[1:])
	case "plan":
		return a.runPlan(ctx, args[1:])
	case "help", "--help", "-h":
		a.runHelp()
		return ExitSuccess
	case "version", "--version", "-v":
		fmt.Fprintf(a.out, "imole %s\n", Version)
		return ExitSuccess
	default:
		a.printError(&Error{
			Code:       "unknown_command",
			Message:    fmt.Sprintf("unknown command %q", args[0]),
			Suggestion: "Run: imole help",
			Retryable:  false,
		})
		return ExitUsage
	}
}

func (a *App) printError(err *Error) {
	if a.isTTY {
		fmt.Fprintf(a.err, "%s %s\n", a.red("error:"), err.Message)
		if err.Suggestion != "" {
			fmt.Fprintf(a.err, "%s %s\n", a.yellow("hint: "), err.Suggestion)
		}
	} else {
		json.NewEncoder(a.err).Encode(err)
	}
}

func (a *App) shouldJSON() bool {
	return !a.isTTY
}

func (a *App) outputJSON(v any, fields string) int {
	if fields == "" {
		return a.writeJSON(v)
	}
	return a.writeJSONFields(v, fields)
}

// writeJSONFields extracts only the specified dot-path fields from v and outputs them.
// Example: fields="summary.total_files,summary.photo_files" → {"summary":{"total_files":42,"photo_files":30}}
func (a *App) writeJSONFields(v any, fields string) int {
	raw, err := json.Marshal(v)
	if err != nil {
		return ExitError
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		// If v is not an object (e.g. array), fall back to full output
		return a.writeJSON(v)
	}

	result := make(map[string]any)
	for _, field := range strings.Split(fields, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		val := getNestedValue(m, field)
		setNestedValue(result, field, val)
	}

	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return ExitError
	}
	return ExitSuccess
}

// getNestedValue traverses a map using dot-separated path.
func getNestedValue(m map[string]any, path string) any {
	parts := strings.SplitN(path, ".", 2)
	val, ok := m[parts[0]]
	if !ok {
		return nil
	}
	if len(parts) == 1 {
		return val
	}
	if sub, ok := val.(map[string]any); ok {
		return getNestedValue(sub, parts[1])
	}
	return nil
}

// setNestedValue sets a value in a nested map structure using dot-separated path.
func setNestedValue(m map[string]any, path string, val any) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 1 {
		m[parts[0]] = val
		return
	}
	sub, ok := m[parts[0]].(map[string]any)
	if !ok {
		sub = make(map[string]any)
		m[parts[0]] = sub
	}
	setNestedValue(sub, parts[1], val)
}

func (a *App) writeJSON(v any) int {
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return ExitError
	}
	return ExitSuccess
}

func usageError(msg string) *Error {
	return &Error{
		Code:       "usage_error",
		Message:    msg,
		Suggestion: "Run: imole help or imole schema <command>",
		Retryable:  false,
	}
}

func runtimeError(code, msg, suggestion string, retryable bool) *Error {
	return &Error{
		Code:       code,
		Message:    msg,
		Suggestion: suggestion,
		Retryable:  retryable,
	}
}
