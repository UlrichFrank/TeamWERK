//go:build measure

package measure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// repoRoot walks up from the current working directory until it finds go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repoRoot: go.mod not found from working directory")
		}
		dir = parent
	}
}

// writeReport renders the three measurement tables to metrics/PAYLOAD.md at the
// repo root. The output is deterministic (no timestamp) so re-runs diff cleanly
// against the committed baseline.
func writeReport(t *testing.T, root string, payloads []routeResult, revals []revalResult, fanouts []fanoutResult) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("# Payload-Messung\n\n")
	b.WriteString(fmt.Sprintf("> Deterministischer Seed (measureRefTime=%s): %d Members / %d Games / %d Duty-Slots / %d Training-Sessions / %d Duty-Types (10× 3072-Byte instruction_md) / %d Chat-Nachrichten.\n",
		measureRefTime.Format("2006-01-02"), seedMembersTotal, seedGames, seedDutySlots, seedTrainingSessions, seedDutyTypes, seedChatMessages))
	b.WriteString("> Erhoben über den vollen Produktions-Router (testutil/prodserver) mit Admin-Bearer. Erzeugt via `make measure` (nicht Teil des blockierenden Gates).\n\n")

	b.WriteString("## Payload pro Route\n\n")
	b.WriteString("| Route | Pfad | Status | Bytes |\n|---|---|---:|---:|\n")
	for _, r := range payloads {
		b.WriteString(fmt.Sprintf("| %s | `%s` | %d | %d |\n", r.Label, r.Path, r.Status, r.Bytes))
	}

	b.WriteString("\n## Referenzdaten-Revalidierung (If-None-Match)\n\n")
	b.WriteString("| Route | 1. Call | 2. Call (If-None-Match) |\n|---|---|---|\n")
	for _, r := range revals {
		etag := r.ETag
		if etag == "" {
			etag = "kein ETag"
		}
		b.WriteString(fmt.Sprintf("| %s | %d / %d B | %d / %d B _(%s)_ |\n",
			r.Label, r.Status1, r.Bytes1, r.Status2, r.Bytes2, etag))
	}

	b.WriteString("\n## SSE-Fan-out pro Mutation (8 Clients C1..C8)\n\n")
	b.WriteString("| Mutation | Clients erreicht | Verteilung |\n|---|---:|---|\n")
	for _, f := range fanouts {
		labels := make([]string, 0, len(f.PerClient))
		for l := range f.PerClient {
			labels = append(labels, l)
		}
		sort.Strings(labels)
		parts := make([]string, 0, len(labels))
		for _, l := range labels {
			parts = append(parts, fmt.Sprintf("%s:%d", l, f.PerClient[l]))
		}
		b.WriteString(fmt.Sprintf("| %s | %d/%d | %s |\n",
			f.Mutation, f.ClientsReached, len(f.PerClient), strings.Join(parts, " ")))
	}

	path := filepath.Join(root, "metrics", "PAYLOAD.md")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	return path
}

func TestMeasure_WritesReport(t *testing.T) {
	db := testutil.NewDB(t)
	data := measureSeed(t, db)
	baseURL := startServer(t, data)

	payloads := measurePayloads(t, baseURL, data.adminToken, data)
	revals := measureRevalidation(t, baseURL, data.adminToken)
	fanouts := measureFanout(t, baseURL, data)

	path := writeReport(t, repoRoot(t), payloads, revals, fanouts)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	got := string(content)
	for _, section := range []string{"## Payload pro Route", "## Referenzdaten-Revalidierung", "## SSE-Fan-out pro Mutation"} {
		if !strings.Contains(got, section) {
			t.Errorf("report missing section %q", section)
		}
	}
	t.Logf("report written to %s", path)
}
