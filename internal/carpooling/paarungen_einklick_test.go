package carpooling_test

import (
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/carpooling"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestRequestPairing_EinseitigSuche_LegtEintragAn: Mitfahren ohne eigenes Gesuch
// legt den Suche-Spiegel an und erstellt eine pending-Paarung (initiiert_von='suche').
func TestRequestPairing_EinseitigSuche_LegtEintragAn(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	bieterID := testutil.CreateUser(t, db, "standard")
	requesterID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()

	token := testutil.Token(t, requesterID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"bieteId": bieteID,
		"plaetze": 1,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var sucheCount int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`, gameID, requesterID).Scan(&sucheCount)
	if sucheCount != 1 {
		t.Errorf("expected 1 suche entry for requester, got %d", sucheCount)
	}

	var pairCount int
	var initiertVon, status string
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrt_paarungen WHERE biete_id = ?`, bieteID).Scan(&pairCount)
	db.QueryRow(`SELECT initiiert_von, status FROM mitfahrt_paarungen WHERE biete_id = ?`, bieteID).Scan(&initiertVon, &status)
	if pairCount != 1 || initiertVon != "suche" || status != "pending" {
		t.Errorf("expected 1 pending pairing initiiert_von=suche, got count=%d von=%q status=%q", pairCount, initiertVon, status)
	}
}

// TestRequestPairing_EinseitigFuerKind: Elternteil fragt ohne Kind-Eintrag mit
// forUserId an → Suche-Eintrag entsteht für das Kind.
func TestRequestPairing_EinseitigFuerKind(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	parentID, childUserID, _ := setupParentChild(t, db)
	bieterID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"bieteId":   bieteID,
		"forUserId": childUserID,
		"plaetze":   1,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`, gameID, childUserID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 suche entry for child, got %d", count)
	}
}

// TestRequestPairing_EinseitigBiete_LegtEintragAn: Platz anbieten ohne eigenes
// Angebot legt den Biete-Spiegel an (initiiert_von='biete').
func TestRequestPairing_EinseitigBiete_LegtEintragAn(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	sucherID := testutil.CreateUser(t, db, "standard")
	driverID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, sucherID)
	sucheID, _ := sucheRes.LastInsertId()

	token := testutil.Token(t, driverID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"sucheId": sucheID,
		"plaetze": 4,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var bieteCount int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'biete'`, gameID, driverID).Scan(&bieteCount)
	if bieteCount != 1 {
		t.Errorf("expected 1 biete entry for driver, got %d", bieteCount)
	}

	var initiertVon string
	db.QueryRow(`SELECT initiiert_von FROM mitfahrt_paarungen WHERE suche_id = ?`, sucheID).Scan(&initiertVon)
	if initiertVon != "biete" {
		t.Errorf("expected initiiert_von=biete, got %q", initiertVon)
	}
}

// TestRequestPairing_WiederverwendetSuche: ein bestehendes Gesuch ohne aktive
// Paarung wird wiederverwendet — kein zweiter Suche-Eintrag.
func TestRequestPairing_WiederverwendetSuche(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	bieterID := testutil.CreateUser(t, db, "standard")
	requesterID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()
	// Requester hat bereits ein Gesuch ohne Paarung.
	db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 2)`, gameID, requesterID)

	token := testutil.Token(t, requesterID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"bieteId": bieteID,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`, gameID, requesterID).Scan(&count)
	if count != 1 {
		t.Errorf("expected suche entry reused (1), got %d", count)
	}
}

// TestRequestPairing_FremderForUserId: forUserId ohne Bezug → 403, kein Eintrag.
func TestRequestPairing_FremderForUserId(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	requesterID := testutil.CreateUser(t, db, "standard")
	bieterID := testutil.CreateUser(t, db, "standard")
	strangerID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()

	token := testutil.Token(t, requesterID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"bieteId":   bieteID,
		"forUserId": strangerID,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`, gameID, strangerID).Scan(&count)
	if count != 0 {
		t.Errorf("expected no suche entry for stranger, got %d", count)
	}
}

// TestRequestPairing_EinseitigKapazitaetVoll: volle Kapazität → 409, kein
// Spiegel-Eintrag persistiert (Atomarität der Transaktion).
func TestRequestPairing_EinseitigKapazitaetVoll(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	bieterID := testutil.CreateUser(t, db, "standard")
	occupantID := testutil.CreateUser(t, db, "standard")
	requesterID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	// Biete mit nur 1 Platz, bereits von occupant belegt.
	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 1)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()
	occRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, occupantID)
	occSucheID, _ := occRes.LastInsertId()
	db.Exec(`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status) VALUES (?, ?, 'suche', 'confirmed')`, bieteID, occSucheID)

	token := testutil.Token(t, requesterID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{
		"bieteId": bieteID,
		"plaetze": 1,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`, gameID, requesterID).Scan(&count)
	if count != 0 {
		t.Errorf("expected no phantom suche entry after 409, got %d", count)
	}
}

// TestRequestPairing_Altpfad_Regression: zweiseitiger {bieteId,sucheId}-Body
// funktioniert unverändert (Happy-Path) und gibt 403 ohne Bezug.
func TestRequestPairing_Altpfad_Regression(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	bieterID := testutil.CreateUser(t, db, "standard")
	sucherID := testutil.CreateUser(t, db, "standard")
	strangerID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, sucherID)
	sucheID, _ := sucheRes.LastInsertId()

	// Happy-Path: Sucher paart selbst.
	okToken := testutil.Token(t, sucherID, "standard", nil)
	okRes := testutil.Post(t, srv, "/api/mitfahrt-paarungen", okToken, map[string]any{
		"bieteId": bieteID,
		"sucheId": sucheID,
	})
	defer okRes.Body.Close()
	if okRes.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for two-sided happy path, got %d", okRes.StatusCode)
	}

	// Fehlerfall: Unbeteiligter ohne Bezug → 403.
	badToken := testutil.Token(t, strangerID, "standard", nil)
	badRes := testutil.Post(t, srv, "/api/mitfahrt-paarungen", badToken, map[string]any{
		"bieteId": bieteID,
		"sucheId": sucheID,
	})
	defer badRes.Body.Close()
	if badRes.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for unrelated user, got %d", badRes.StatusCode)
	}
}

// TestRequestPairing_LeererBody: weder bieteId noch sucheId → 400.
func TestRequestPairing_LeererBody(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrt-paarungen", token, map[string]any{})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", res.StatusCode)
	}
}
