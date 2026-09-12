package auth_test

// Betriebshärtung Welle 2 (design.md Entscheidung 3): der Reset-Token-INSERT
// beim Aktivieren eines Proxy-Kontos lief bisher auf r.Context() — der wird
// storniert, sobald ServeHTTP für PUT /api/users/{id} zurückkehrt, was
// praktisch immer VOR der asynchronen Versand-Goroutine passiert (der Handler
// startet die Goroutine und schreibt danach nur noch w.WriteHeader). Ohne den
// Fix (context.Background() + eigener Timeout statt r.Context()) verliert der
// INSERT dieses Rennen und der Reset-Token landet nie in der DB — der 204
// täuscht Erfolg vor, obwohl das aktivierte Konto sein Passwort nie setzen
// kann.

import (
	"database/sql"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// insertProxyAccount legt ein nicht-aktivierbares Proxy-Konto an (can_login=0,
// keine E-Mail) wie es der Beitritts-/Kinder-Freigabe-Flow erzeugt.
func insertProxyAccount(t *testing.T, db *sql.DB, firstName, lastName string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO users (email, first_name, last_name, password, role, can_login) VALUES ('', ?, ?, '', 'standard', 0)`,
		firstName, lastName)
	if err != nil {
		t.Fatalf("insertProxyAccount: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// TestUpdateUser_AktivierungSchreibtResetTokenTrotzBeendetemRequest (2.3): nach
// einer 204-Antwort auf PUT /api/users/{id} (can_login: 1) existiert der
// Reset-Token in der DB — kurz gepollt, weil der Versand asynchron läuft.
func TestUpdateUser_AktivierungSchreibtResetTokenTrotzBeendetemRequest(t *testing.T) {
	db := testutil.NewDB(t)
	_, token := newVorstand(t, db)
	targetID := insertProxyAccount(t, db, "Lena", "Schmidt")
	srv := newAuthServer(t, db)

	res := testutil.Put(t, srv, "/api/users/"+strconv.Itoa(targetID), token, map[string]any{
		"can_login": 1,
		"email":     "lena.schmidt@test.local",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("UpdateUser: erwartet 204, got %d", res.StatusCode)
	}

	var count int
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM password_reset_tokens WHERE user_id=?`, targetID,
		).Scan(&count); err != nil {
			t.Fatalf("count password_reset_tokens: %v", err)
		}
		if count > 0 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if count != 1 {
		t.Fatalf("erwartet 1 password_reset_tokens-Zeile nach Aktivierung, bekam %d — "+
			"Reset-Token-INSERT darf nicht auf r.Context() laufen (design.md Entscheidung 3)", count)
	}
}
