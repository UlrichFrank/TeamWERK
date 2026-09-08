package kader_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestDeleteKader_MitTrainingsAbgelehnt: ein LEERER Altkader ist ohne die
// Trainings-Guard löschbar — genau das Ergebnis eines Saisonaufräumens — und
// nähme seine Trainings samt Anwesenheits- und RSVP-Historie mit
// (proposal.md — Invariante 4).
func TestDeleteKader_MitTrainingsAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2025-10-01")

	// Der Kader ist leer — nur die Trainingshistorie hängt an ihm.
	var memberCount int
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=?`, kaderID).Scan(&memberCount)
	if memberCount != 0 {
		t.Fatalf("Vorbedingung: Kader muss leer sein, hat aber %d Mitglieder", memberCount)
	}

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Delete("/api/kader/{id}", h.DeleteKader)
	})

	resp := testutil.Delete(t, srv, "/api/kader/"+strconv.Itoa(kaderID), token)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409, bekommen %d", resp.StatusCode)
	}

	var kaderRows, sessionRows int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE id=?`, kaderID).Scan(&kaderRows)
	db.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE id=?`, sessionID).Scan(&sessionRows)
	if kaderRows != 1 || sessionRows != 1 {
		t.Fatalf("Kader=%d Termin=%d — die Historie muss den Löschversuch überleben", kaderRows, sessionRows)
	}
}

// TestUpdateKader_ErweiterterKaderBeiUebungsgruppeAbgelehnt: der erweiterte
// Kader ist die einzige unerwünschte Fläche, die an kader_id statt an teams.id
// hängt — und damit das einzige anwendungsseitige Gate dieses Changes
// (design.md — Entscheidung 3). 409, nicht 404: der Kader existiert.
func TestUpdateKader_ErweiterterKaderBeiUebungsgruppeAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	memberID := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/kader/{id}", h.UpdateKader)
	})

	resp := testutil.Put(t, srv, "/api/kader/"+strconv.Itoa(groupID), token,
		map[string]any{"extended_members_add": []int{memberID}})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("erwartet 409, bekommen %d", resp.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM kader_extended_members WHERE kader_id=?`, groupID).Scan(&n)
	if n != 0 {
		t.Fatalf("kader_extended_members hat %d Zeilen, erwartet 0", n)
	}
}

// TestUpdateKader_ErweiterterKaderBeiMannschaftErlaubt: das Gate greift genau
// bei kind='practice' — die Mannschaftsvariante bleibt unberührt. Ohne diesen
// Gegentest wäre ein zu breites Gate nicht von einem richtigen zu unterscheiden.
func TestUpdateKader_ErweiterterKaderBeiMannschaftErlaubt(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	memberID := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/kader/{id}", h.UpdateKader)
	})

	resp := testutil.Put(t, srv, "/api/kader/"+strconv.Itoa(kaderID), token,
		map[string]any{"extended_members_add": []int{memberID}})
	defer resp.Body.Close()
	// UpdateKader antwortet im Erfolgsfall 204 — geprüft wird der Kontrast zum
	// 409 der Übungsgruppe, nicht der genaue Erfolgs-Code.
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekommen %d", resp.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM kader_extended_members WHERE kader_id=? AND member_id=?`,
		kaderID, memberID).Scan(&n)
	if n != 1 {
		t.Fatalf("erweiterter Kader nicht gepflegt (%d Zeilen)", n)
	}
}

// TestCopyFromSeason_UeberspringtUebungsgruppen: der Kopierer keyt auf
// age_class|gender und ruft ensureTeam — ohne den kind='team'-Filter
// kollabierten alle Übungsgruppen auf den Schlüssel "|" und ensureTeam legte
// ein Team ohne Altersklasse und Geschlecht an (design.md — Entscheidung 7).
func TestCopyFromSeason_UeberspringtUebungsgruppen(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	fromSeason := testutil.CreateSeason(t, db, "2025/26")
	testutil.CreatePracticeGroup(t, db, fromSeason, "Torwarttraining")
	testutil.CreatePracticeGroup(t, db, fromSeason, "Athletik")
	toSeason := testutil.CreateSeason(t, db, "2026/27")

	var teamsBefore int
	db.QueryRow(`SELECT COUNT(*) FROM teams`).Scan(&teamsBefore)

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/kader/copy-from-season", h.CopyFromSeason)
	})

	resp := testutil.Post(t, srv, "/api/kader/copy-from-season", token, map[string]any{
		"from_season_id": fromSeason,
		"to_season_id":   toSeason,
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekommen %d", resp.StatusCode)
	}

	var practiceInTarget int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE season_id=? AND kind='practice'`, toSeason).Scan(&practiceInTarget)
	if practiceInTarget != 0 {
		t.Errorf("Zielsaison enthält %d Übungsgruppen, erwartet 0", practiceInTarget)
	}
	var anyInTarget int
	db.QueryRow(`SELECT COUNT(*) FROM kader WHERE season_id=?`, toSeason).Scan(&anyInTarget)
	if anyInTarget != 0 {
		t.Errorf("Zielsaison enthält %d Kader, erwartet 0 (die Quelle hatte nur Übungsgruppen)", anyInTarget)
	}
	var teamsAfter int
	db.QueryRow(`SELECT COUNT(*) FROM teams`).Scan(&teamsAfter)
	if teamsAfter != teamsBefore {
		t.Errorf("teams-Zeilen: vorher %d, nachher %d — ensureTeam darf für Übungsgruppen nie laufen", teamsBefore, teamsAfter)
	}
}
