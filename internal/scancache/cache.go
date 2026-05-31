// Package scancache provides a simple disk cache for scan results.
// A successful scan is always written to cache. The --cache flag on the
// scan command causes the CLI to read from cache instead of rescanning,
// so the heavy USB/ImageCaptureCore enumeration can be skipped when the
// data is still fresh.
package scancache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/imole/internal/media"
)

const DefaultTTL = time.Hour

type Entry struct {
	ScannedAt time.Time    `json:"scanned_at"`
	Provider  string       `json:"provider"`
	Source    string       `json:"source"`
	Result    media.Result `json:"result"`
}

// CacheDir returns (and creates) the imole cache directory.
func CacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "imole")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func cacheFile(provider, source string) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("%s\x00%s", provider, source)
	h := sha256.Sum256([]byte(key))
	return filepath.Join(dir, fmt.Sprintf("scan-%x.json", h[:8])), nil
}

// Write saves result to disk. Errors are silently ignored by callers
// (best-effort — a cache write failure must never break the main flow).
func Write(provider, source string, result media.Result) error {
	path, err := cacheFile(provider, source)
	if err != nil {
		return err
	}
	entry := Entry{
		ScannedAt: time.Now(),
		Provider:  provider,
		Source:    source,
		Result:    result,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Read loads a cached scan result. Returns the entry and true when the
// cache exists and is younger than ttl. Returns false otherwise.
func Read(provider, source string, ttl time.Duration) (Entry, bool) {
	path, err := cacheFile(provider, source)
	if err != nil {
		return Entry{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, false
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return Entry{}, false
	}
	if len(entry.Result.Items) == 0 {
		return Entry{}, false
	}
	if time.Since(entry.ScannedAt) > ttl {
		return Entry{}, false
	}
	return entry, true
}

// Age returns how old the most recent cache file for provider+source is.
// Returns a zero duration and false if no cache exists.
func Age(provider, source string) (time.Duration, bool) {
	path, err := cacheFile(provider, source)
	if err != nil {
		return 0, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return time.Since(info.ModTime()), true
}
