package games_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

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

// TestListGames_DBFehlerLoggtUndVerraetNichts: schlägt die Abfrage fehl (hier
// über eine geschlossene Verbindung erzwungen), bekommt der Client nur
// {"error":"internal"} und der Betrieb eine Log-Zeile mit dem Pfad. Vorher
// stand "internal error" als Plain-Text im Body und die Ursache nur auf
// os.Stderr — ohne Pfad, Methode und Nutzer.
func TestListGames_DBFehlerLoggtUndVerraetNichts(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	userID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", nil)

	buf := captureLogs(t)
	// Verbindung schließen, damit jede Query im Handler scheitert. Der
	// Cleanup von testutil.NewDB schließt danach erneut — das ist erlaubt.
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	res := testutil.Get(t, srv, "/api/games", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.StatusCode)
	}
	raw, _ := io.ReadAll(res.Body)
	body := strings.TrimSpace(string(raw))
	if body != `{"error":"internal"}` {
		t.Errorf(`expected {"error":"internal"}, got %q`, body)
	}
	for _, leak := range []string{"sql", "no such", "database"} {
		if strings.Contains(strings.ToLower(body), leak) {
			t.Errorf("body leaks %q: %s", leak, body)
		}
	}
	if !strings.Contains(buf.String(), "/api/games") {
		t.Errorf("expected a log line carrying the path, got: %s", buf.String())
	}
}

// TestListGames_LimitGedeckelt: ?limit=100000 liefert höchstens 200 Einträge —
// ein Client kann die Tabelle nicht in einem Zug abziehen. total bleibt die
// echte Gesamtzahl, damit die Seite weiterblättern kann.
func TestListGames_LimitGedeckelt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}
	for i := 0; i < 205; i++ {
		if _, err := db.Exec(
			`INSERT INTO games (season_id, opponent, date, time, event_type, is_home) VALUES (?,?,?,?,?,?)`,
			seasonID, "Gegner", "2026-01-15", "18:00", "heim", 1); err != nil {
			t.Fatalf("insert game %d: %v", i, err)
		}
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, "/api/games?limit=100000", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) > 200 {
		t.Errorf("limit not capped: got %d items", len(resp.Items))
	}
	if resp.Total != 205 {
		t.Errorf("expected total=205, got %d", resp.Total)
	}
}
