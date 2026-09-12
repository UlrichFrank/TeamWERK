package kader_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// captureLogs lenkt den Default-Logger für die Dauer des Tests in einen Buffer
// um und setzt ihn danach zurück.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// TestListKader_DBFehlerLoggtUndVerraetNichts: schlägt die Abfrage fehl (hier
// durch eine geschlossene Verbindung erzwungen), bekommt der Client nur
// {"error":"internal"} — kein SQL-Text — und der Betrieb eine Log-Zeile mit
// dem Pfad. Vorher stand err.Error() im Body und nichts im Log.
func TestListKader_DBFehlerLoggtUndVerraetNichts(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader", h.ListKader)
	})

	buf := captureLogs(t)
	// Verbindung schließen, damit jede Query im Handler scheitert. Der
	// Cleanup von testutil.NewDB schließt danach erneut — das ist erlaubt.
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	resp := testutil.Get(t, srv, "/api/kader", token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	body := strings.TrimSpace(string(raw))
	if body != `{"error":"internal"}` {
		t.Errorf(`expected {"error":"internal"}, got %q`, body)
	}
	for _, leak := range []string{"sql", "no such", "database"} {
		if strings.Contains(strings.ToLower(body), leak) {
			t.Errorf("body leaks %q: %s", leak, body)
		}
	}
	if !strings.Contains(buf.String(), "/api/kader") {
		t.Errorf("expected a log line carrying the path, got: %s", buf.String())
	}
}
