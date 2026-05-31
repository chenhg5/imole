// Package cli provides the iMole command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const Version = "0.1.0"

type App struct {
	out   io.Writer
	err   io.Writer
	isTTY bool
}

func New(out, err io.Writer) *App {
	return &App{out: out, err: err, isTTY: isTerminal(out)}
}

func isTerminal(w io.Writer) bool {
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

	switch args[0] {
	case "schema":
		return a.runSchema(ctx, args[1:])
	case "doctor":
		return a.runDoctor(ctx, args[1:])
	case "scan":
		return a.runScan(ctx, args[1:])
	case "stats":
		return a.runStats(ctx, args[1:])
	case "videos":
		return a.runVideos(ctx, args[1:])
	case "backup":
		return a.runBackup(ctx, args[1:])
	case "report":
		return a.runReport(ctx, args[1:])
	case "guide":
		return a.runGuide(ctx, args[1:])
	case "clean":
		return a.runClean(ctx, args[1:])
	case "history":
		return a.runHistory(ctx, args[1:])
	case "update":
		return a.runUpdate(ctx, args[1:])
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