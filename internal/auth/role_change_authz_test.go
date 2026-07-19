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

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(vorstandID)+"/role",
		testutil.Token(t, vorstandID, "standard", []string{"vorstand"}),
		map[string]string{"role": "presseteam"})
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

// Legitime Nicht-Admin-Rollenpflege bleibt möglich: Vorstand ändert die Rolle
// eines Nicht-Admin-Accounts (hier standard→presseteam) → 204.
func TestUpdateUserRole_VorstandManagesNonAdmin_OK(t *testing.T) {
	db := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, db, "standard")
	targetID := testutil.CreateUser(t, db, "standard")
	srv := newAuthServer(t, db)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/users/"+itoa(targetID)+"/role",
		testutil.Token(t, vorstandID, "standard", []string{"vorstand"}),
		map[string]string{"role": "presseteam"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = ?`, targetID).Scan(&role)
	if role != "presseteam" {
		t.Errorf("expected presseteam, got %q", role)
	}
}
