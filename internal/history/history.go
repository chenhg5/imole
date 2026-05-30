// Package history records and retrieves imole operation logs (backup, clean).
// Entries are written as newline-delimited JSON to ~/.local/share/imole/operations.jsonl.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const defaultLogName = "operations.jsonl"

// Kind is the type of operation recorded.
type Kind string

const (
	KindBackup Kind = "backup"
	KindClean  Kind = "clean"
)

// Entry is a single operation log record.
type Entry struct {
	Time         time.Time `json:"time"`
	Kind         Kind      `json:"kind"`
	Files        int       `json:"files"`
	Size         int64     `json:"size"`
	Destination  string    `json:"destination,omitempty"`  // backup: dest dir
	ManifestPath string    `json:"manifest_path,omitempty"` // clean: manifest used
	Failed       int       `json:"failed,omitempty"`
}

// LogPath returns the path of the operation log file, creating parent directories
// if necessary.
func LogPath() (string, error) {
	base, err := logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, defaultLogName), nil
}

func logDir() (string, error) {
	// Prefer XDG_DATA_HOME; fall back to ~/.local/share.
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "share")
	}
	target := filepath.Join(dir, "imole")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	return target, nil
}

// Append writes a new entry to the operation log.
// Errors are silently ignored so a log failure never interrupts the main flow.
func Append(e Entry) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	path, err := LogPath()
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(e)
}

// Read returns all entries from the log, most-recent first.
func Read(limit int) ([]Entry, error) {
	path, err := LogPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, line := range splitLines(data) {
		var e Entry
		if json.Unmarshal(line, &e) == nil {
			entries = append(entries, e)
		}
	}

	// Reverse: most-recent first.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
