package provider

import (
	"context"
	"fmt"
	"runtime"

	"github.com/chenhg5/imole/internal/media"
)

type DownloadRequest struct {
	Item    media.Item `json:"item"`
	DestRel string     `json:"dest_rel"`
}

type DownloadResult struct {
	SourceRel string `json:"source_rel"`
	DestRel   string `json:"dest_rel"`
	Verified  bool   `json:"verified"`
	Skipped   bool   `json:"skipped"`
	Error     string `json:"error,omitempty"`
}

func Download(ctx context.Context, providerName Name, requests []DownloadRequest, destRoot string) ([]DownloadResult, error) {
	switch providerName {
	case ImageCapture:
		return DownloadImageCapture(ctx, requests, destRoot)
	case Auto:
		if runtime.GOOS == "darwin" {
			return DownloadImageCapture(ctx, requests, destRoot)
		}
		return nil, fmt.Errorf("auto download requires a concrete provider on %s", runtime.GOOS)
	default:
		return nil, fmt.Errorf("provider %q does not support direct download", providerName)
	}
}
