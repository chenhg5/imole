//go:build !darwin

package provider

import (
	"context"
	"fmt"
)

// DeleteRequest specifies a single file path to delete from the device.
type DeleteRequest struct {
	Path string `json:"path"`
}

// DeleteResult reports the outcome for a single deletion attempt.
type DeleteResult struct {
	Path    string `json:"path"`
	Deleted bool   `json:"deleted"`
	Error   string `json:"error,omitempty"`
}

func DeleteImageCapture(ctx context.Context, requests []DeleteRequest) ([]DeleteResult, error) {
	_, _ = ctx, requests
	return nil, fmt.Errorf("ImageCaptureCore delete is macOS-only")
}
