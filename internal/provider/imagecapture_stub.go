//go:build !darwin

package provider

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/media"
)

func ScanImageCapture(ctx context.Context, opts media.Options) (media.Result, error) {
	_, _ = ctx, opts
	return media.Result{}, fmt.Errorf("ImageCaptureCore provider is macOS-only")
}
