package chat_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// broadcastSrv baut Handler + Server für die Sende-Route und verschluckt Pushes.
func broadcastSrv(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	h.SetPushFn(func(*sql.DB, *appconfig.Config, int, string, string, string, int) {})
	return testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/broadcasts", h.SendBroadcast)
	})
}

// sendResult ist die Antwort von POST /api/chat/broadcasts.
type sendResult struct {
	ID         int `json:"id"`
	Recipients int `json:"recipients"`
}

// clubWide baut die targets-Liste für ein vereinsweites Ziel (ohne teamId).
func clubWide(kind string) []any {
	return []any{map[string]any{"kind": kind}}
}

// teamTarget baut die targets-Liste für eine Team-Standardgruppe.
func teamTarget(kind string, teamID int) []any {
	return []any{map[string]any{"kind": kind, "teamId": teamID}}
}

// TC: Der Absenderkreis für VEREINSWEITE Ziele. Trainer haben zwar seit
// mitteilung-team-gruppen wieder ein Senderecht, aber nur für die
// Standardgruppen ihrer eigenen Kader — an "Alle Nutzer" kommen weiterhin nur
// admin, vorstand und sportliche Leitung.
func TestSendBroadcast_Absenderkreis(t *testing.T) {
	tests := []struct {
		name          string
		role          string
		clubFunctions []string
		wantStatus    int
	}{
		{"admin", "admin", nil, http.StatusCreated},
		{"vorstand", "standard", []string{"vorstand"}, http.StatusCreated},
		{"sportliche_leitung", "standard", []string{"sportliche_leitung"}, http.StatusCreated},
		{"reiner trainer", "standard", []string{"trainer"}, http.StatusForbidden},
		{"spieler", "standard", []string{"spieler"}, http.StatusForbidden},
		{"kassierer", "standard", []string{"kassierer"}, http.StatusForbidden},
		{"ohne Funktion", "standard", nil, http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			sender := testutil.CreateUser(t, db, tc.role)
			srv := broadcastSrv(t, db)

			token := testutil.Token(t, sender, tc.role, tc.clubFunctions)
			res := testutil.Post(t, srv, "/api/chat/broadcasts", token,
				map[string]any{"body": "Ansage", "targets": clubWide("users")})
			if res.StatusCode != tc.wantStatus {
				t.Fatalf("status %d, want %d", res.StatusCode, tc.wantStatus)
			}

			var stored int
			if err := db.QueryRow(`SELECT COUNT(*) FROM broadcasts`).Scan(&stored); err != nil {
				t.Fatalf("count broadcasts: %v", err)
			}
			wantStored := 0
			if tc.wantStatus == http.StatusCreated {
				wantStored = 1
			}
			if stored != wantStored {
				t.Errorf("%d gespeicherte Mitteilungen, want %d", stored, wantStored)
			}
		})
	}
}

// TC: Die abgelehnten Ziele. Die Altwerte müssen hart mit 400 scheitern —
// 'role' insbesondere, weil es früher stillschweigend an null Empfänger
// zustellte statt einen Fehler zu melden. Dazu die beiden Formfehler des
// targets-Arrays: leer, und eine teamId, die nicht zum Kind passt (in beide
// Richtungen — dieselbe Bindung erzwingt der CHECK auf broadcast_targets).
func TestSendBroadcast_UngueltigeZielgruppe(t *testing.T) {
	tests := []struct {
		name    string
		targets any
	}{
		{"all", clubWide("all")},
		{"team", clubWide("team")},
		{"role", clubWide("role")},
		{"legacy", clubWide("legacy")},
		{"spieler_innen", clubWide("spieler_innen")},
		{"(fehlt)", nil},
		{"(leer)", []any{}},
		{"team-Ziel ohne teamId", clubWide("team_spieler")},
		{"vereinsweites Ziel mit teamId", teamTarget("users", 1)},
		{"ein gültiges, ein unbekanntes Ziel", []any{
			map[string]any{"kind": "users"},
			map[string]any{"kind": "nonsense"},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			sender := testutil.CreateUser(t, db, "standard")
			srv := broadcastSrv(t, db)

			token := testutil.Token(t, sender, "standard", []string{"vorstand"})
			body := map[string]any{"body": "Ansage"}
			if tc.targets != nil {
				body["targets"] = tc.targets
			}
			res := testutil.Post(t, srv, "/api/chat/broadcasts", token, body)
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("targets %v: status %d, want 400", tc.targets, res.StatusCode)
			}

			var stored int
			db.QueryRow(`SELECT COUNT(*) FROM broadcasts`).Scan(&stored)
			if stored != 0 {
				t.Errorf("targets %v wurden gespeichert, obwohl abgelehnt", tc.targets)
			}
		})
	}
}

// TC: Der Absender bekommt seine Zeile mit gesetztem read_at (die eigene
// Mitteilung darf nicht als ungelesen erscheinen), zählt aber nicht in
// recipients — sonst meldete der Composer eine Person zu viel.
func TestSendBroadcast_AbsenderZeileAberNichtGezaehlt(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	testutil.CreateUser(t, db, "standard")
	testutil.CreateUser(t, db, "standard")
	srv := broadcastSrv(t, db)

	token := testutil.Token(t, sender, "standard", []string{"vorstand"})
	res := testutil.Post(t, srv, "/api/chat/broadcasts", token,
		map[string]any{"body": "Ansage", "targets": clubWide("users")})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}
	body := decodeJSON[sendResult](t, res)

	if body.Recipients != 2 {
		t.Errorf("recipients = %d, want 2 (drei User, Absender zählt nicht)", body.Recipients)
	}

	var readAt sql.NullString
	if err := db.QueryRow(
		`SELECT read_at FROM broadcast_reads WHERE broadcast_id = ? AND user_id = ?`,
		body.ID, sender).Scan(&readAt); err != nil {
		t.Fatalf("Absender hat keine broadcast_reads-Zeile: %v", err)
	}
	if !readAt.Valid {
		t.Error("Absenderzeile hat read_at NULL — die eigene Mitteilung erschiene als ungelesen")
	}
}
