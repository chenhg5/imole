package progress

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/chenhg5/imole/internal/human"
)

type Event struct {
	Event      string `json:"event"`
	Index      int    `json:"index"`
	TotalFiles int    `json:"total_files"`
	Path       string `json:"path"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
}

type Writer struct {
	out io.Writer
}

func NewWriter(out io.Writer) *Writer {
	return &Writer{out: out}
}

func (w *Writer) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			fmt.Fprintln(w.out, line)
			continue
		}
		w.print(event)
	}
	return len(p), nil
}

func (w *Writer) print(event Event) {
	name := event.Path
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	prefix := fmt.Sprintf("[%d/%d] %s", event.Index, event.TotalFiles, name)
	switch event.Event {
	case "start":
		fmt.Fprintf(w.out, "%s starting · %s\n", prefix, human.Bytes(event.Total))
	case "progress":
		percent := 0.0
		if event.Total > 0 {
			percent = float64(event.Downloaded) * 100 / float64(event.Total)
		}
		fmt.Fprintf(w.out, "%s %.0f%% · %s / %s\n", prefix, percent, human.Bytes(event.Downloaded), human.Bytes(event.Total))
	case "skip":
		fmt.Fprintf(w.out, "%s skipped · already verified\n", prefix)
	case "done":
		fmt.Fprintf(w.out, "%s done · %s\n", prefix, human.Bytes(event.Total))
	case "error":
		fmt.Fprintf(w.out, "%s failed\n", prefix)
	default:
		fmt.Fprintf(w.out, "%s %s\n", prefix, event.Event)
	}
}
