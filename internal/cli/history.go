package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/history"
	"github.com/chenhg5/imole/internal/human"
)

func (a *App) runHistory(_ context.Context, args []string) int {
	var limit int
	var jsonMode bool
	fs := flagSet("history")
	fs.IntVar(&limit, "limit", 20, "maximum number of entries to show")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	entries, err := history.Read(limit)
	if err != nil {
		a.printError(runtimeError("history_read_failed", err.Error(), "", false))
		return ExitError
	}

	if a.shouldJSON() || jsonMode {
		return a.writeJSON(entries)
	}

	logPath, _ := history.LogPath()
	fmt.Fprintln(a.out, "iMole Operation History")
	fmt.Fprintf(a.out, "Log: %s\n", absPath(logPath))
	fmt.Fprintln(a.out)

	if len(entries) == 0 {
		fmt.Fprintln(a.out, "No operations recorded yet.")
		fmt.Fprintln(a.out, "Run imole backup or imole clean to start logging.")
		return ExitSuccess
	}

	for _, e := range entries {
		ts := e.Time.Format("2006-01-02 15:04")
		switch e.Kind {
		case history.KindBackup:
			fmt.Fprintf(a.out, "  %s  backup  %d files · %s  →  %s\n",
				ts, e.Files, human.Bytes(e.Size), e.Destination)
			if e.Failed > 0 {
				fmt.Fprintf(a.out, "                         (%d failed)\n", e.Failed)
			}
		case history.KindClean:
			fmt.Fprintf(a.out, "  %s  clean   %d files · %s  [manifest: %s]\n",
				ts, e.Files, human.Bytes(e.Size), e.ManifestPath)
			if e.Failed > 0 {
				fmt.Fprintf(a.out, "                         (%d failed)\n", e.Failed)
			}
		default:
			fmt.Fprintf(a.out, "  %s  %s\n", ts, e.Kind)
		}
	}
	return ExitSuccess
}
