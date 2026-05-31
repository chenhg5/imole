package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestGuideAnalysisExposesAgentPlaybook(t *testing.T) {
	var out, err bytes.Buffer
	app := New(&out, &err)

	code := app.Run(context.Background(), []string{"guide", "analysis"})
	if code != ExitSuccess {
		t.Fatalf("guide analysis exit = %d, stderr = %s", code, err.String())
	}
	text := out.String()
	for _, want := range []string{
		"storage analysis playbook",
		"device.storage.free_percent",
		"imole scan --summary --json",
		"imole scan apps --top 20 --json",
		"Do not add --dry-run to scan",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("guide analysis missing %q:\n%s", want, text)
		}
	}
}
