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
		return nil, fmt.Errorf("auto delete requires a concrete provider on %s", runtime.GOOS)
	default:
		return nil, fmt.Errorf("provider %q does not support delete; use ImageCapture (macOS only)", providerName)
	}
}
