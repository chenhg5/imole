package cli

import (
	"context"
	"fmt"
	"strings"
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
			{Name: "fields", Type: "string", Default: "", Description: "comma-separated dot-paths to include in JSON output, e.g. device.name,device.storage.free_percent"},
		},
	},
	"scan": {
		Name:        "scan",
		Description: "Scan iPhone storage. --summary shows media + app storage; media subcommand keeps media-only behavior.",
		Flags: []SchemaFlag{
			{Name: "provider", Type: "string", Default: "auto", Description: "media provider: auto, filesystem, imagecapture, gphoto"},
			{Name: "source", Type: "string", Default: "", Description: "scan a local mounted path instead of USB device"},
			{Name: "only", Type: "string", Default: "all", Enum: []string{"all", "photos", "videos"}, Description: "filter by media type"},
			{Name: "ext", Type: "string", Default: "", Description: "filter by file extension, e.g. png (≈screenshots on iPhone), heic, mov, jpg"},
			{Name: "top", Type: "int", Default: "0", Description: "show top N largest files sorted by size; 0 = summary mode"},
			{Name: "limit", Type: "int", Default: "0", Description: "cap result to N items after filtering (largest first); 0 = no limit"},
			{Name: "summary", Type: "bool", Default: "false", Description: "combined media + app summary"},
			{Name: "cache", Type: "bool", Default: "false", Description: "use cached scan result if available and less than 1 hour old"},
			{Name: "older-than", Type: "string", Default: "", Description: "filter: older than age, e.g. 90d, 6m, 1y"},
			{Name: "large-than", Type: "string", Default: "", Description: "filter: larger than size, e.g. 500MB, 1GB"},
			{Name: "with-meta", Type: "bool", Default: "false", Description: "fetch EXIF metadata (GPS, taken date, dimensions); first run ~30-60s, cached 7 days"},
			{Name: "country", Type: "string", Default: "", Description: "keep items whose GPS resolves to this country or region; auto-enables --with-meta"},
			{Name: "no-gps", Type: "bool", Default: "false", Description: "keep only items with no GPS coordinates; auto-enables --with-meta"},
			{Name: "taken-after", Type: "string", Default: "", Description: "keep items taken on or after date YYYY-MM-DD; auto-enables --with-meta"},
			{Name: "taken-before", Type: "string", Default: "", Description: "keep items taken before date YYYY-MM-DD; auto-enables --with-meta"},
			{Name: "duration-gt", Type: "float", Default: "0", Description: "keep videos longer than N seconds; auto-enables --with-meta"},
			{Name: "min-width", Type: "int", Default: "0", Description: "keep items with width >= N pixels; auto-enables --with-meta"},
			{Name: "min-height", Type: "int", Default: "0", Description: "keep items with height >= N pixels; auto-enables --with-meta"},
			{Name: "max-width", Type: "int", Default: "0", Description: "keep items with width <= N pixels; auto-enables --with-meta"},
			{Name: "max-height", Type: "int", Default: "0", Description: "keep items with height <= N pixels; auto-enables --with-meta"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON format"},
			{Name: "fields", Type: "string", Default: "", Description: "comma-separated dot-paths to include in JSON output"},
		},
	},
	"backup": {
		Name:        "backup",
		Description: "Back up media and write manifest",
		Flags: []SchemaFlag{
			{Name: "provider", Type: "string", Default: "auto", Description: "media provider: auto, filesystem, imagecapture, gphoto"},
			{Name: "source", Type: "string", Default: "", Description: "scan an existing mounted media path"},
			{Name: "to", Type: "string", Required: true, Description: "backup destination directory"},
			{Name: "file", Type: "string[]", Default: "", Description: "specific rel_path from scan output; repeat --file for multiple files"},
			{Name: "limit", Type: "int", Default: "0", Description: "back up at most N files (largest first); 0 = no limit"},
			{Name: "dry-run", Type: "bool", Default: "false", Description: "preview backup without copying"},
			{Name: "yes", Type: "bool", Default: "false", Description: "skip confirmation prompt"},
			{Name: "only", Type: "string", Default: "all", Enum: []string{"all", "photos", "videos"}, Description: "media filter: all, photos, videos"},
			{Name: "ext", Type: "string", Default: "", Description: "filter by file extension, e.g. png, heic, mov"},
			{Name: "older-than", Type: "string", Default: "", Description: "include media older than an age, e.g. 90d"},
			{Name: "large-than", Type: "string", Default: "", Description: "include media larger than a size, e.g. 500MB"},
			{Name: "with-meta", Type: "bool", Default: "false", Description: "fetch EXIF metadata for GPS/date/country filtering; auto-enabled when metadata filters are set"},
			{Name: "country", Type: "string", Default: "", Description: "back up items from a specific country/region; auto-enables --with-meta"},
			{Name: "no-gps", Type: "bool", Default: "false", Description: "back up items with no GPS data only; auto-enables --with-meta"},
			{Name: "taken-after", Type: "string", Default: "", Description: "back up items taken on or after YYYY-MM-DD; auto-enables --with-meta"},
			{Name: "taken-before", Type: "string", Default: "", Description: "back up items taken before YYYY-MM-DD; auto-enables --with-meta"},
			{Name: "duration-gt", Type: "float", Default: "0", Description: "back up videos longer than N seconds; auto-enables --with-meta"},
			{Name: "min-width", Type: "int", Default: "0", Description: "back up items with width >= N pixels; auto-enables --with-meta"},
			{Name: "min-height", Type: "int", Default: "0", Description: "back up items with height >= N pixels; auto-enables --with-meta"},
			{Name: "max-width", Type: "int", Default: "0", Description: "back up items with width <= N pixels; auto-enables --with-meta"},
			{Name: "max-height", Type: "int", Default: "0", Description: "back up items with height <= N pixels; auto-enables --with-meta"},
			{Name: "json", Type: "bool", Default: "false", Description: "output JSON format"},
			{Name: "fields", Type: "string", Default: "", Description: "comma-separated dot-paths to include in JSON output"},
		},
	},
	"scan apps": {
		Name:        "scan apps",
		Description: "Rank apps by iPhone storage usage using iOS installation_proxy disk usage fields",
		Flags: []SchemaFlag{
			{Name: "scope", Type: "string", Default: "user", Enum: []string{"user", "system", "all"}, Description: "apps to list"},
			{Name: "top", Type: "int", Default: "30", Description: "number of apps to show"},
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
			{Name: "topic", Type: "string", Default: "", Enum: []string{"analysis", "photos", "wechat", "system-data", "trust"}, Description: "specific topic; analysis is the agent storage diagnosis playbook"},
		},
	},
	"clean": {
		Name:        "clean",
		Description: "Delete verified files from iPhone using a backup manifest",
		Flags: []SchemaFlag{
			{Name: "manifest", Type: "string", Required: false, Description: "path to manifest.json from a previous backup; if omitted, prints recommended flow"},
			{Name: "provider", Type: "string", Default: "auto", Description: "deletion provider: auto, imagecapture (macOS USB), filesystem (mount path)"},
			{Name: "source", Type: "string", Required: false, Description: "mount point of iPhone DCIM directory (Linux: ifuse mount, Windows: iTunes drive); enables filesystem deletion with immediate space reclaim"},
			{Name: "file", Type: "string[]", Default: "", Description: "specific verified source_rel from manifest to delete; repeat --file for multiple files"},
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
	"uninstall": {
		Name:        "uninstall",
		Description: "Remove a user-installed app from iPhone (guarded by IMOLE_NO_DELETE env and system-app protection)",
		Flags: []SchemaFlag{
			{Name: "bundle-id", Type: "string", Required: true, Description: "app bundle ID to uninstall, e.g. com.example.myapp; use 'imole scan apps' to find bundle IDs"},
			{Name: "dry-run", Type: "bool", Default: "false", Description: "preview: show what would be uninstalled without removing the app"},
			{Name: "yes", Type: "bool", Default: "false", Description: "skip confirmation prompt"},
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
	if cmdName == "" && fs.NArg() > 0 {
		cmdName = strings.Join(fs.Args(), " ")
	}
	if cmdName != "" && fs.NArg() > 0 && strings.Join(fs.Args(), " ") != cmdName {
		a.printError(usageError("schema accepts one command name"))
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
