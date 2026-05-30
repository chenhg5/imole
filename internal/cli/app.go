package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const Version = "0.1.0"

type App struct {
	out     io.Writer
	err     io.Writer
	isTTY   bool
}

func New(out, err io.Writer) *App {
	return &App{out: out, err: err, isTTY: isTerminal(out)}
}

func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isTerminalFD(int(f.Fd()))
	}
	return false
}

// isTerminalFD checks if the given file descriptor is a terminal.
func isTerminalFD(fd int) bool {
	return false // stub: always false in non-interactive context
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
		fmt.Fprintf(a.err, "imole: %s\n", err.Message)
		if err.Suggestion != "" {
			fmt.Fprintf(a.err, "Suggestion: %s\n", err.Suggestion)
		}
	} else {
		json.NewEncoder(a.err).Encode(err)
	}
}

func (a *App) shouldJSON() bool {
	return !a.isTTY
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