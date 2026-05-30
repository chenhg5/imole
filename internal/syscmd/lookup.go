// Package syscmd provides command lookup utilities.
package syscmd

import (
	"os/exec"
	"path/filepath"
)

var extraSearchDirs = []string{
	"/opt/homebrew/bin",
	"/usr/local/bin",
}

func LookPath(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	for _, dir := range extraSearchDirs {
		path := filepath.Join(dir, name)
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}
	return "", exec.ErrNotFound
}
