package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/provider"
)

func (a *App) runVideos(ctx context.Context, args []string) int {
	var providerName, source, olderThan, largeThan string
	var top int
	var jsonMode bool
	fs := flagSet("videos")
	addProviderFlags(fs, &providerName, &source)
	fs.StringVar(&olderThan, "older-than", "", "show videos older than an age, e.g. 90d, 1y")
	fs.StringVar(&largeThan, "large-than", "", "show videos larger than a size, e.g. 300MB")
	fs.IntVar(&top, "top", 20, "number of videos to show")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	f, err := parseFilter("videos", olderThan, largeThan)
	if err != nil {
		a.printError(&Error{
			Code:       "usage_error",
			Message:    err.Error(),
			Suggestion: "Use --only photos|videos",
			Retryable:  false,
		})
		return ExitUsage
	}
	result, err := scanFromFlags(ctx, providerName, source, f.LargeThan, f.OlderThan)
	if err != nil {
		a.printError(runtimeError("scan_failed", err.Error(), "", true))
		return ExitError
	}
	filtered := provider.FilteredItems(result, f)
	videos := media.TopVideos(filtered, top)
	if a.shouldJSON() || jsonMode {
		return a.writeJSON(videos)
	}
	fmt.Fprintf(a.out, "Top %d Videos\n\n", len(videos))
	for i, item := range videos {
		fmt.Fprintf(a.out, "%2d. %-28s %8s  %s\n", i+1, item.Name, human.Bytes(item.Size), item.ModTime.Format("2006-01-02"))
	}
	return ExitSuccess
}