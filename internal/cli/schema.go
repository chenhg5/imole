package cli

import (
	"context"
	"fmt"
)

type SchemaCommand struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Flags       []SchemaFlag `json:"flags,omitempty"`
}

type SchemaFlag struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description"`
	Required    bool     `json:"required,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

var commandSchemas = map[string]SchemaCommand{
	"doctor": {
		Name:        "doctor",
		Description: "Check device and local dependencies",
		Flags: []SchemaFlag{
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON format"},
			{Name: "fields", Type: "string", Default: "", Description: "comma-separated dot-paths to include in JSON output, e.g. device.name,device.udid"},
		},
	},
	"scan": {
		Name:        "scan",
		Description: "Scan media — summary report, top N by size, or compact stats. Replaces old stats and videos commands.",
		Flags: []SchemaFlag{
			{Name: "provider", Type: "string", Default: "auto", Description: "media provider: auto, filesystem, imagecapture, gphoto"},
			{Name: "source", Type: "string", Default: "", Description: "scan a local mounted path instead of USB device"},
			{Name: "only", Type: "string", Default: "all", Enum: []string{"all", "photos", "videos"}, Description: "filter by media type"},
			{Name: "top", Type: "int", Default: "0", Description: "show top N largest files sorted by size; 0 = summary mode"},
			{Name: "summary", Type: "bool", Default: "false", Description: "compact stats table only (equivalent to old stats command)"},
			{Name: "older-than", Type: "string", Default: "", Description: "filter: older than age, e.g. 90d, 6m, 1y"},
			{Name: "large-than", Type: "string", Default: "", Description: "filter: larger than size, e.g. 500MB, 1GB"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON format"},
			{Name: "fields", Type: "string", Default: "", Description: "comma-separated dot-paths to include in JSON output"},
		},
	},
	// stats and videos are kept as undocumented aliases for backward compatibility.
	"stats": {
		Name:        "stats",
		Description: "Alias for: scan --summary. Kept for backward compatibility.",
		Flags: []SchemaFlag{
			{Name: "provider", Type: "string", Default: "auto", Description: "media provider"},
			{Name: "source", Type: "string", Default: "", Description: "local path"},
			{Name: "only", Type: "string", Default: "all", Description: "media filter"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON"},
			{Name: "fields", Type: "string", Default: "", Description: "JSON field filter"},
		},
	},
	"videos": {
		Name:        "videos",
		Description: "Alias for: scan --only videos --top N. Kept for backward compatibility.",
		Flags: []SchemaFlag{
			{Name: "provider", Type: "string", Default: "auto", Description: "media provider"},
			{Name: "source", Type: "string", Default: "", Description: "local path"},
			{Name: "top", Type: "int", Default: "20", Description: "number of videos to show"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON"},
		},
	},
	"backup": {
		Name:        "backup",
		Description: "Back up media and write manifest",
		Flags: []SchemaFlag{
			{Name: "provider", Type: "string", Default: "auto", Description: "media provider: auto, filesystem, imagecapture, gphoto"},
			{Name: "source", Type: "string", Default: "", Description: "scan an existing mounted media path"},
			{Name: "to", Type: "string", Required: true, Description: "backup destination directory"},
			{Name: "dry-run", Type: "bool", Default: "false", Description: "preview backup without copying"},
			{Name: "only", Type: "string", Default: "all", Enum: []string{"all", "photos", "videos"}, Description: "media filter: all, photos, videos"},
			{Name: "older-than", Type: "string", Default: "", Description: "include media older than an age, e.g. 90d"},
			{Name: "large-than", Type: "string", Default: "", Description: "include media larger than a size, e.g. 500MB"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON format"},
			{Name: "fields", Type: "string", Default: "", Description: "comma-separated dot-paths to include in JSON output"},
		},
	},
	"report": {
		Name:        "report",
		Description: "Summarize a backup manifest",
		Flags: []SchemaFlag{
			{Name: "manifest", Type: "string", Required: true, Description: "path to manifest.json"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON format"},
			{Name: "fields", Type: "string", Default: "", Description: "comma-separated dot-paths to include in JSON output"},
		},
	},
	"guide": {
		Name:        "guide",
		Description: "Show cleanup guidance",
		Flags: []SchemaFlag{
			{Name: "topic", Type: "string", Default: "", Description: "specific topic: ios-updates, screenshots, whatsapp, etc."},
		},
	},
	"clean": {
		Name:        "clean",
		Description: "Delete verified files from iPhone using a backup manifest",
		Flags: []SchemaFlag{
			{Name: "manifest", Type: "string", Required: false, Description: "path to manifest.json from a previous backup; if omitted, prints recommended flow"},
			{Name: "provider", Type: "string", Default: "auto", Description: "deletion provider: auto, imagecapture (macOS USB), filesystem (mount path)"},
			{Name: "source", Type: "string", Required: false, Description: "mount point of iPhone DCIM directory (Linux: ifuse mount, Windows: iTunes drive); enables filesystem deletion with immediate space reclaim"},
			{Name: "dry-run", Type: "bool", Default: "false", Description: "preview deletion without removing files (exit 10 = safe to proceed)"},
			{Name: "yes", Type: "bool", Default: "false", Description: "skip confirmation prompt; useful for scripting"},
		},
	},
	"history": {
		Name:        "history",
		Description: "Show recent backup and delete operations",
		Flags: []SchemaFlag{
			{Name: "limit", Type: "int", Default: "20", Description: "maximum number of entries to show"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON format"},
		},
	},
	"update": {
		Name:        "update",
		Description: "Update imole to the latest release",
		Flags: []SchemaFlag{
			{Name: "check", Type: "bool", Default: "false", Description: "check for updates without installing"},
			{Name: "nightly", Type: "bool", Default: "false", Description: "install latest unreleased build from main branch (requires go)"},
		},
	},
	"schema": {
		Name:        "schema",
		Description: "Show command structure and parameters",
		Flags: []SchemaFlag{
			{Name: "command", Type: "string", Default: "", Description: "specific command to show schema for; omit for all"},
		},
	},
}

func (a *App) runSchema(ctx context.Context, args []string) int {
	var cmdName string
	fs := flagSet("schema")
	fs.StringVar(&cmdName, "command", "", "command to show schema for")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError("invalid schema flags"))
		return ExitUsage
	}

	if cmdName == "" {
		// Output all schemas
		schemas := make([]SchemaCommand, 0, len(commandSchemas))
		for _, s := range commandSchemas {
			schemas = append(schemas, s)
		}
		return a.writeJSON(schemas)
	}

	schema, ok := commandSchemas[cmdName]
	if !ok {
		a.printError(&Error{
			Code:       "not_found",
			Message:    fmt.Sprintf("unknown command %q", cmdName),
			Suggestion: "Run: imole schema",
			Retryable:  false,
		})
		return ExitNotFound
	}
	return a.writeJSON(schema)
}
