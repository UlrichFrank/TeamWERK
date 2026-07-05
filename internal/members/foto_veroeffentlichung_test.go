package members_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Happy-Path: Vorstand setzt foto_veroeffentlichung=true ohne Datum → 204,
// Feld persistiert, Datum wird serverseitig gesetzt.
func TestUpdateMember_FotoVeroeffentlichung_SetztFeldUndDatum(t *testing.T) {
	database := testutil.NewDB(t)
	memberID := testutil.CreateMember(t, database, 0)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{
			"first_name":             "Foto",
			"last_name":              "Frei",
			"status":                 "aktiv",
			"join_date":              "2026-01-01",
			"foto_veroeffentlichung": true,
		})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT /members/{id}: expected 204, got %d", res.StatusCode)
	}

	var foto int
	var fotoDate *string
	if err := database.QueryRow(
		`SELECT foto_veroeffentlichung, foto_veroeffentlichung_date FROM members WHERE id=?`, memberID).
		Scan(&foto, &fotoDate); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if foto != 1 {
		t.Errorf("foto_veroeffentlichung: got %d, want 1", foto)
	}
	if fotoDate == nil || *fotoDate == "" {
		t.Errorf("foto_veroeffentlichung_date: erwartet gesetzt, war NULL/leer")
	}
}

// Fehlerfall: Ein nicht-privilegierter Nutzer (standard ohne Vereinsfunktion)
// darf Mitgliedsdaten nicht direkt schreiben → 403.
func TestUpdateMember_FotoVeroeffentlichung_NichtPrivilegiert403(t *testing.T) {
	database := testutil.NewDB(t)
	memberID := testutil.CreateMember(t, database, 0)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", nil)
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{
			"first_name":             "Foto",
			"last_name":              "Frei",
			"status":                 "aktiv",
			"join_date":              "2026-01-01",
			"foto_veroeffentlichung": true,
		})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("PUT /members/{id} ohne Recht: expected 403, got %d", res.StatusCode)
	}
}

// Ein dsgvo-Draft trägt foto_veroeffentlichung; Annehmen übernimmt das Feld
// (inkl. gesetztem Datum) auf das Mitglied.
func TestDsgvoDraft_UebernimmtFotoVeroeffentlichung(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, userID)
	h := members.NewHandler(database, hub.NewHub())

	newValue, _ := json.Marshal(map[string]bool{
		"verarbeitung":           true,
		"weitergabe":             false,
		"foto_veroeffentlichung": true,
	})
	draft, err := h.CreateOrUpdateDraft(memberID, userID, members.ChangeRequest{
		FieldName: "dsgvo",
		NewValue:  json.RawMessage(newValue),
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateDraft: %v", err)
	}
	if err := h.AcceptDraft(draft.ID); err != nil {
		t.Fatalf("AcceptDraft: %v", err)
	}

	var foto int
	var fotoDate *string
	if err := database.QueryRow(
		`SELECT foto_veroeffentlichung, foto_veroeffentlichung_date FROM members WHERE id=?`, memberID).
		Scan(&foto, &fotoDate); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if foto != 1 {
		t.Errorf("foto_veroeffentlichung nach Accept: got %d, want 1", foto)
	}
	if fotoDate == nil || *fotoDate == "" {
		t.Errorf("foto_veroeffentlichung_date nach Accept: erwartet gesetzt, war NULL/leer")
	}
}
