//go:build !darwin

package provider

import (
	"context"
	"fmt"
)

func DownloadImageCapture(ctx context.Context, requests []DownloadRequest, destRoot string) ([]DownloadResult, error) {
	_, _, _ = ctx, requests, destRoot
	return nil, fmt.Errorf("ImageCaptureCore download is macOS-only")
}
