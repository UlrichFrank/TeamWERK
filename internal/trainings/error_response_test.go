package trainings_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
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

// TestListSessions_DBFehlerLoggtUndVerraetNichts: schlägt die Abfrage fehl
// (hier über eine geschlossene Verbindung erzwungen), bekommt der Client nur
// {"error":"internal"} und der Betrieb eine Log-Zeile mit dem Pfad. Vorher
// stand "internal error" als Plain-Text im Body und die Ursache nur auf
// os.Stderr — ohne Pfad, Methode und Nutzer.
func TestListSessions_DBFehlerLoggtUndVerraetNichts(t *testing.T) {
	db := testutil.NewDB(t)
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", nil)

	buf := captureLogs(t)
	// Verbindung schließen, damit jede Query im Handler scheitert. Der
	// Cleanup von testutil.NewDB schließt danach erneut — das ist erlaubt.
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	resp := testutil.Get(t, srv, "/api/training-sessions", token)
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
	if !strings.Contains(buf.String(), "/api/training-sessions") {
		t.Errorf("expected a log line carrying the path, got: %s", buf.String())
	}
}
