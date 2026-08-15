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

// TC: Der Absenderkreis. Trainer verlieren das Mitteilungsrecht — Team-Ansagen
// laufen über die Team-Standardgruppen des Chats. Die reine sportliche Leitung
// gewinnt es: sie kam vorher zwar durch den Outer-Guard, scheiterte danach aber
// an der Trainer-Klausel, die kader_trainers-Mitgliedschaft verlangte. Button
// sichtbar, jede Zielgruppe 403 — dieser Test hält den Fix fest.
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
				map[string]any{"body": "Ansage", "targetType": "users"})
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

// TC: Die abgelehnten Zielgruppen-Werte. Die drei Altwerte müssen hart mit 400
// scheitern — 'role' insbesondere, weil es früher stillschweigend an null
// Empfänger zustellte statt einen Fehler zu melden.
func TestSendBroadcast_UngueltigeZielgruppe(t *testing.T) {
	for _, target := range []string{"all", "team", "role", "legacy", "", "spieler_innen"} {
		name := target
		if name == "" {
			name = "(leer)"
		}
		t.Run(name, func(t *testing.T) {
			db := testutil.NewDB(t)
			sender := testutil.CreateUser(t, db, "standard")
			srv := broadcastSrv(t, db)

			token := testutil.Token(t, sender, "standard", []string{"vorstand"})
			body := map[string]any{"body": "Ansage"}
			if target != "" {
				body["targetType"] = target
			}
			res := testutil.Post(t, srv, "/api/chat/broadcasts", token, body)
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("targetType %q: status %d, want 400", target, res.StatusCode)
			}

			var stored int
			db.QueryRow(`SELECT COUNT(*) FROM broadcasts`).Scan(&stored)
			if stored != 0 {
				t.Errorf("targetType %q wurde gespeichert, obwohl abgelehnt", target)
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
		map[string]any{"body": "Ansage", "targetType": "users"})
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
