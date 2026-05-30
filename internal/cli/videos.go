package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/provider"
)

func (a *App) runVideos(ctx context.Context, args []string) int {
	var providerName, source, only, olderThan, largeThan, fields string
	var top int
	var jsonMode bool
	fs := flagSet("videos")
	addProviderFlags(fs, &providerName, &source)
	addFilterFlags(fs, &only, &olderThan, &largeThan)
	fs.IntVar(&top, "top", 20, "number of videos to show")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	f, err := parseFilter(only, olderThan, largeThan)
	if err != nil {
		a.printError(&Error{
			Code:       "usage_error",
			Message:    err.Error(),
			Suggestion: "Use --only photos|videos, --older-than 90d|6m|1y, --large-than 500MB|1GB",
			Retryable:  false,
		})
		return ExitUsage
	}
	result, err := scanFromFlags(ctx, providerName, source, f.LargeThan, f.OlderThan)
	if err != nil {
		a.printError(runtimeError("scan_failed", err.Error(), "Try: imole videos --source /path/to/DCIM", true))
		return ExitError
	}
	filtered := provider.FilteredItems(result, f)
	videos := media.TopVideos(filtered, top)
	if a.shouldJSON() || jsonMode {
		return a.outputJSON(videos, fields)
	}
	fmt.Fprintf(a.out, "%s\n\n", a.bold(fmt.Sprintf("Top %d Videos", len(videos))))
	for i, item := range videos {
		fmt.Fprintf(a.out, "%2d. %-28s %s  %s\n",
			i+1,
			item.Name,
			a.cyan(fmt.Sprintf("%8s", human.Bytes(item.Size))),
			a.dim(item.ModTime.Format("2006-01-02")),
		)
	}
	return ExitSuccess
}