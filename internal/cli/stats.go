package cli

// runStats and StatsResult have been merged into runScan (--summary flag).
// This file is kept as a placeholder for the StatsResult type used by schema.go.

// StatsResult is the agent-friendly stats output with pre-computed human-readable sizes.
type StatsResult struct {
	TotalFiles int64       `json:"total_files"`
	TotalSize  int64       `json:"total_size"`
	TotalHuman string      `json:"total_size_human"`
	PhotoFiles int64       `json:"photo_files"`
	PhotoSize  int64       `json:"photo_size"`
	PhotoHuman string      `json:"photo_size_human"`
	VideoFiles int64       `json:"video_files"`
	VideoSize  int64       `json:"video_size"`
	VideoHuman string      `json:"video_size_human"`
	OldFiles   int64       `json:"old_files"`
	OldSize    int64       `json:"old_size"`
	OldHuman   string      `json:"old_size_human"`
	LargeFiles int64       `json:"large_files"`
	LargeSize  int64       `json:"large_size"`
	LargeHuman string      `json:"large_size_human"`
	Filter     StatsFilter `json:"filter"`
}

type StatsFilter struct {
	Only      string `json:"only"`
	OlderThan string `json:"older_than,omitempty"`
	LargeThan string `json:"large_than,omitempty"`
	Source    string `json:"source,omitempty"`
}
