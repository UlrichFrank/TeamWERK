package auth_test

import (
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Ein Nicht-Admin (Vorstand) darf einen bestehenden Admin NICHT herabstufen.
func TestUpdateUserRole_NonAdminCannotDemoteAdmin(t *testing.T) {
	db := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, db, "standard")
	adminTargetID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(adminTargetID)+"/role",
		testutil.Token(t, vorstandID, "standard", []string{"vorstand"}),
		map[string]string{"role": "standard"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = ?`, adminTargetID).Scan(&role)
	if role != "admin" {
		t.Errorf("target admin must stay admin, got %q", role)
	}
}

// Ein Nicht-Admin darf die EIGENE Rolle nicht ändern.
func TestUpdateUserRole_NonAdminCannotChangeOwnRole(t *testing.T) {
	db := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	// Seit dem Wegfall von presseteam gibt es nur noch admin|standard; die
	// Selbstzuweisung von admin blockiert schon die Admin-Regel. Geprüft wird
	// deshalb mit standard: der 403 kommt allein daher, dass Ziel = Aufrufer ist.
	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(vorstandID)+"/role",
		testutil.Token(t, vorstandID, "standard", []string{"vorstand"}),
		map[string]string{"role": "standard"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = ?`, vorstandID).Scan(&role)
	if role != "standard" {
		t.Errorf("own role must stay standard, got %q", role)
	}
}

// Ein Admin darf einen Admin weiterhin herabstufen (kein Regress durch den Guard).
func TestUpdateUserRole_AdminCanDemoteAdmin(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	adminTargetID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(adminTargetID)+"/role",
		testutil.Token(t, adminID, "admin", nil),
		map[string]string{"role": "standard"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = ?`, adminTargetID).Scan(&role)
	if role != "standard" {
		t.Errorf("admin must be able to demote admin, role=%q", role)
	}
}

// Legitime Nicht-Admin-Rollenpflege bleibt möglich: Vorstand setzt einen
// Nicht-Admin-Account auf standard → 204. Seit dem Wegfall von presseteam ist
// standard die einzige Zielrolle, die ein Vorstand vergeben darf (admin bleibt
// Admins vorbehalten) — geprüft wird also der Durchlass des Guards, nicht ein
// Rollenwechsel mit sichtbarem Wertunterschied.
func TestUpdateUserRole_VorstandManagesNonAdmin_OK(t *testing.T) {
	db := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, db, "standard")
	targetID := testutil.CreateUser(t, db, "admin")
	srv := newAuthServer(t, db)
	// Ausgangslage: Ziel ist KEIN Admin (sonst greift die Degradierungs-Sperre).
	if _, err := db.Exec(`UPDATE users SET role='standard' WHERE id=?`, targetID); err != nil {
		t.Fatalf("prepare target: %v", err)
	}

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(targetID)+"/role",
		testutil.Token(t, vorstandID, "standard", []string{"vorstand"}),
		map[string]string{"role": "standard"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = ?`, targetID).Scan(&role)
	if role != "standard" {
		t.Errorf("expected standard, got %q", role)
	}
}

// Die abgeschaffte Rolle wird nicht mehr angenommen — weder vom CHECK noch vom
// Handler. Der 400 kommt aus der Rollen-Validierung, vor jedem DB-Zugriff.
func TestUpdateUserRole_PresseteamAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	targetID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(targetID)+"/role",
		testutil.Token(t, adminID, "admin", nil),
		map[string]string{"role": "presseteam"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 für die abgeschaffte Rolle, got %d", res.StatusCode)
	}
	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = ?`, targetID).Scan(&role)
	if role != "standard" {
		t.Errorf("Rolle darf unverändert bleiben, got %q", role)
	}
}
