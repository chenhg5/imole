package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSchemaAcceptsPositionalCommand(t *testing.T) {
	var out, err bytes.Buffer
	app := New(&out, &err)

	code := app.Run(context.Background(), []string{"schema", "scan"})
	if code != ExitSuccess {
		t.Fatalf("schema scan exit = %d, stderr = %s", code, err.String())
	}

	var schema SchemaCommand
	if decodeErr := json.Unmarshal(out.Bytes(), &schema); decodeErr != nil {
		t.Fatalf("decode schema: %v\n%s", decodeErr, out.String())
	}
	if schema.Name != "scan" {
		t.Fatalf("schema name = %q, want scan", schema.Name)
	}
	for _, flag := range schema.Flags {
		if flag.Name == "dry-run" {
			t.Fatalf("scan schema unexpectedly includes dry-run")
		}
	}
}

func TestScanDryRunErrorExplainsReadOnlyCommand(t *testing.T) {
	var out, err bytes.Buffer
	app := New(&out, &err)

	code := app.Run(context.Background(), []string{"scan", "--top", "10", "--only", "videos", "--dry-run"})
	if code != ExitUsage {
		t.Fatalf("scan --dry-run exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(err.String(), "scan is read-only and does not accept --dry-run") {
		t.Fatalf("stderr did not explain scan dry-run:\n%s", err.String())
	}
}

func TestHelpDoesNotAdvertiseDryRunAsAllCommands(t *testing.T) {
	var out, err bytes.Buffer
	app := New(&out, &err)

	code := app.Run(context.Background(), []string{"help"})
	if code != ExitSuccess {
		t.Fatalf("help exit = %d, stderr = %s", code, err.String())
	}
	help := out.String()
	if strings.Contains(help, "all commands") && strings.Contains(help, "--dry-run") {
		t.Fatalf("help still suggests dry-run is global:\n%s", help)
	}
	if !strings.Contains(help, "backup and clean only") {
		t.Fatalf("help does not scope dry-run to backup/clean:\n%s", help)
	}
}
