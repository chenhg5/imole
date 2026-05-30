package cli

import (
	"context"
	"fmt"
)

func (a *App) runClean(_ context.Context, args []string) int {
	var experimental bool
	fs := flagSet("clean")
	fs.BoolVar(&experimental, "experimental", false, "show experimental cleanup status")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	if experimental {
		fmt.Fprintln(a.out, "Experimental delete is not enabled in v0.1.")
		fmt.Fprintln(a.out, "iMole will only support deletion after manifest verification and strict DCIM-only guards.")
		return ExitSuccess
	}
	fmt.Fprintln(a.out, "Safe cleanup mode")
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "iMole v0.1 does not delete iPhone media automatically.")
	fmt.Fprintln(a.out, "Recommended flow:")
	fmt.Fprintln(a.out, "  1. imole scan")
	fmt.Fprintln(a.out, "  2. imole backup --source /path/to/DCIM --to /path/to/backup --only videos --older-than 90d")
	fmt.Fprintln(a.out, "  3. imole report --manifest /path/to/backup/manifest.json")
	fmt.Fprintln(a.out, "  4. Delete verified imports with Image Capture or Photos")
	fmt.Fprintln(a.out, "  5. Empty Recently Deleted on iPhone")
	return ExitSuccess
}