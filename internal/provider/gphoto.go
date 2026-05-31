package provider

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/syscmd"
)

var gphotoListLine = regexp.MustCompile(`^#([0-9]+)\s+(.+?)\s+([0-9.]+)\s+([KMG]?B)\s+(.+)$`)

func ScanGPhoto(ctx context.Context, opts media.Options) (media.Result, error) {
	gphoto, err := syscmd.LookPath("gphoto2")
	if err != nil {
		return media.Result{}, fmt.Errorf("gphoto2 not found")
	}
	cmd := exec.CommandContext(ctx, gphoto, "--list-files")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return media.Result{}, fmt.Errorf("gphoto2: %s", trimGPhotoError(out))
	}
	items := parseGPhotoList(out)
	if len(items) == 0 {
		return media.Result{}, fmt.Errorf("gphoto2 did not expose any media files")
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})
	summary := summarize("gphoto2", items, opts)
	return media.Result{Summary: summary, Items: items}, nil
}

// trimGPhotoError extracts the most relevant error line from gphoto2 output.
// gphoto2 dumps verbose debug text; we want just the one-liner error code.
func trimGPhotoError(out []byte) string {
	lines := strings.Split(string(out), "\n")
	// Prefer lines with "Error" and a code, e.g. "*** Error (-53: ...)"
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "*** Error (") {
			// Strip leading/trailing stars and spaces
			l = strings.Trim(l, "* ")
			return l
		}
	}
	// Fall back to the first "An error occurred" line
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "An error occurred") {
			// Truncate at the first colon+detail if too long
			if idx := strings.Index(l, "):"); idx > 0 {
				return l[:idx+1]
			}
			if len(l) > 120 {
				return l[:120] + "…"
			}
			return l
		}
	}
	// Last resort: first non-empty line, truncated
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "***") {
			if len(l) > 120 {
				return l[:120] + "…"
			}
			return l
		}
	}
	return strings.TrimSpace(string(out[:min(len(out), 120)]))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseGPhotoList(out []byte) []media.Item {
	var items []media.Item
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		match := gphotoListLine.FindStringSubmatch(line)
		if len(match) != 6 {
			continue
		}
		size := parseGPhotoSize(match[3], match[4])
		name := strings.TrimSpace(match[2])
		item := media.NewItem("gphoto2", "gphoto2/"+name, size, time.Time{})
		item.SourcePath = "gphoto2://" + match[1]
		items = append(items, item)
	}
	return items
}

func parseGPhotoSize(value, unit string) int64 {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(unit) {
	case "GB":
		n *= 1024 * 1024 * 1024
	case "MB":
		n *= 1024 * 1024
	case "KB":
		n *= 1024
	}
	return int64(n)
}

func summarize(root string, items []media.Item, opts media.Options) media.Summary {
	s := media.Summary{Root: root}
	for _, item := range items {
		s.TotalFiles++
		s.TotalSize += item.Size
		switch item.Kind {
		case "photo":
			s.PhotoFiles++
			s.PhotoSize += item.Size
		case "video":
			s.VideoFiles++
			s.VideoSize += item.Size
		default:
			s.OtherFiles++
			s.OtherSize += item.Size
		}
		if opts.LargeThreshold > 0 && item.Size >= opts.LargeThreshold {
			s.LargeFiles++
			s.LargeSize += item.Size
		}
		if opts.OldBeforeUnix > 0 && !item.ModTime.IsZero() && item.ModTime.Unix() < opts.OldBeforeUnix {
			s.OldFiles++
			s.OldSize += item.Size
		}
	}
	return s
}
