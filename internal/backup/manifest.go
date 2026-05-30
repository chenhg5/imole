package backup

import (
	"encoding/json"
	"os"
	"time"
)

const ManifestName = "manifest.json"

type Manifest struct {
	Version   int            `json:"version"`
	Device    string         `json:"device,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	Root      string         `json:"root"`
	Files     []ManifestFile `json:"files"`
	Summary   Summary        `json:"summary"`
}

type ManifestFile struct {
	SourceRel string    `json:"source_rel"`
	DestRel   string    `json:"dest_rel"`
	Kind      string    `json:"kind"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	Verified  bool      `json:"verified"`
	Skipped   bool      `json:"skipped,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type Summary struct {
	SelectedFiles int   `json:"selected_files"`
	CopiedFiles   int   `json:"copied_files"`
	SkippedFiles  int   `json:"skipped_files"`
	FailedFiles   int   `json:"failed_files"`
	VerifiedFiles int   `json:"verified_files"`
	SelectedSize  int64 `json:"selected_size"`
	CopiedSize    int64 `json:"copied_size"`
	VerifiedSize  int64 `json:"verified_size"`
}

func WriteManifest(path string, manifest Manifest) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(manifest)
}

func ReadManifest(path string) (Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, err
	}
	defer f.Close()
	var manifest Manifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}
