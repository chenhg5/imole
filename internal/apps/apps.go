// Package apps provides iOS app storage listing via ideviceinstaller.
package apps

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/syscmd"
)

type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
	ScopeAll    Scope = "all"
)

type App struct {
	Name        string `json:"name"`
	BundleID    string `json:"bundle_id"`
	StaticSize  int64  `json:"static_size"`
	DynamicSize int64  `json:"dynamic_size"`
	TotalSize   int64  `json:"total_size"`
}

type Result struct {
	Scope string `json:"scope"`
	Apps  []App  `json:"apps"`
}

func List(ctx context.Context, scope Scope) (Result, error) {
	installer, err := syscmd.LookPath("ideviceinstaller")
	if err != nil {
		return Result{}, fmt.Errorf("ideviceinstaller not found; install with: brew install ideviceinstaller")
	}
	if scope == "" {
		scope = ScopeUser
	}
	var scopeArg string
	switch scope {
	case ScopeUser:
		scopeArg = "--user"
	case ScopeSystem:
		scopeArg = "--system"
	case ScopeAll:
		scopeArg = "--all"
	default:
		return Result{}, fmt.Errorf("invalid scope %q", scope)
	}

	runCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	args := []string{
		"list", scopeArg, "--xml",
		"-a", "CFBundleIdentifier",
		"-a", "CFBundleDisplayName",
		"-a", "CFBundleName",
		"-a", "StaticDiskUsage",
		"-a", "DynamicDiskUsage",
	}
	out, err := exec.CommandContext(runCtx, installer, args...).CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("ideviceinstaller list failed: %s", strings.TrimSpace(string(out)))
	}
	apps, err := parsePlistApps(out)
	if err != nil {
		return Result{}, err
	}
	sort.SliceStable(apps, func(i, j int) bool {
		return apps[i].TotalSize > apps[j].TotalSize
	})
	return Result{Scope: string(scope), Apps: apps}, nil
}

func parsePlistApps(data []byte) ([]App, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var apps []App
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "dict" {
			continue
		}
		values, err := parseDict(decoder)
		if err != nil {
			return nil, err
		}
		bundleID := values["CFBundleIdentifier"]
		if bundleID == "" {
			continue
		}
		name := first(values["CFBundleDisplayName"], values["CFBundleName"], bundleID)
		staticSize := parseInt(values["StaticDiskUsage"])
		dynamicSize := parseInt(values["DynamicDiskUsage"])
		apps = append(apps, App{
			Name:        name,
			BundleID:    bundleID,
			StaticSize:  staticSize,
			DynamicSize: dynamicSize,
			TotalSize:   staticSize + dynamicSize,
		})
	}
	return apps, nil
}

func parseDict(decoder *xml.Decoder) (map[string]string, error) {
	values := make(map[string]string)
	var key string
	for {
		tok, err := decoder.Token()
		if err != nil {
			return values, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "key":
				text, err := readElementText(decoder)
				if err != nil {
					return values, err
				}
				key = text
			case "string", "integer", "true", "false":
				text := ""
				if t.Name.Local == "true" {
					text = "true"
				} else if t.Name.Local == "false" {
					text = "false"
				} else {
					var err error
					text, err = readElementText(decoder)
					if err != nil {
						return values, err
					}
				}
				if key != "" {
					values[key] = text
					key = ""
				}
			}
		case xml.EndElement:
			if t.Name.Local == "dict" {
				return values, nil
			}
		}
	}
}

func readElementText(decoder *xml.Decoder) (string, error) {
	var b strings.Builder
	for {
		tok, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			b.Write([]byte(t))
		case xml.EndElement:
			return strings.TrimSpace(b.String()), nil
		}
	}
}

func parseInt(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
