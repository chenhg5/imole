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
		// On Linux/Windows the backup command copies from the filesystem directly
		// when --source is provided; this path is only reached for provider-based
		// downloads which are macOS-only for now.
		return nil, fmt.Errorf(
			"direct download via provider is macOS-only; on %s use --source PATH to scan a mounted DCIM path, then backup copies files from that path directly",
			runtime.GOOS,
		)
	default:
		return nil, fmt.Errorf("provider %q does not support direct download", providerName)
	}
}
