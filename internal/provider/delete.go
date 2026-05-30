package provider

import (
	"context"
	"fmt"
	"runtime"
)

// Delete deletes the given file paths from the connected iPhone using the specified provider.
// Only verified paths (from a backup manifest) should be passed here.
func Delete(ctx context.Context, providerName Name, requests []DeleteRequest) ([]DeleteResult, error) {
	switch providerName {
	case ImageCapture:
		return DeleteImageCapture(ctx, requests)
	case Auto:
		if runtime.GOOS == "darwin" {
			return DeleteImageCapture(ctx, requests)
		}
		// On Linux/Windows: deletion from a mounted DCIM path is handled by the
		// filesystem provider (imole clean --source PATH). USB PTP-based deletion
		// via gphoto2 is not yet implemented.
		return nil, fmt.Errorf(
			"USB delete is macOS-only (ImageCaptureCore); on %s mount the iPhone with ifuse and use --source PATH",
			runtime.GOOS,
		)
	default:
		return nil, fmt.Errorf("provider %q does not support delete; use ImageCapture (macOS only)", providerName)
	}
}
