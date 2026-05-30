package provider

import (
	"context"

	"github.com/chenhg5/imole/internal/media"
)

func ScanFilesystem(ctx context.Context, source string, opts media.Options) (media.Result, error) {
	return media.Scan(ctx, source, opts)
}
