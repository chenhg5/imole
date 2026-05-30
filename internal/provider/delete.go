package provider

import (
	"context"
	"fmt"
	"runtime"
)

// Delete deletes the given file paths from the connected iPhone using the
// specified provider. Only verified paths (from a backup manifest) should be
// passed here. sourcePath is only used when providerName is Filesystem (or Auto
// on non-macOS); it is the root mount point of the iPhone's DCIM directory.
func Delete(ctx context.Context, providerName Name, sourcePath string, requests []DeleteRequest) ([]DeleteResult, error) {
	switch providerName {
	case Filesystem:
		return DeleteFilesystem(sourcePath, requests)
	case ImageCapture:
		return DeleteImageCapture(ctx, requests)
	case Auto:
		if runtime.GOOS == "darwin" {
			return DeleteImageCapture(ctx, requests)
		}
		// On Linux/Windows: use filesystem deletion when --source is provided.
		if sourcePath != "" {
			return DeleteFilesystem(sourcePath, requests)
		}
		hint := "mount the iPhone with ifuse (sudo apt install ifuse && ifuse ~/iphone) then pass --source ~/iphone/DCIM"
		if runtime.GOOS == "windows" {
			hint = "open iTunes, connect the iPhone, then pass --source with the DCIM path shown in Windows Explorer (e.g. --source \"\\\\Apple\\iPhone\\Internal Storage\\DCIM\")"
		}
		return nil, fmt.Errorf("on %s, USB deletion is not supported; %s", runtime.GOOS, hint)
	default:
		return nil, fmt.Errorf("provider %q does not support delete; use imagecapture (macOS) or --source PATH (Linux/Windows)", providerName)
	}
}
