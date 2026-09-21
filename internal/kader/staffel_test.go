package kader

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// staffelFixture legt Saison, Team und Kader an und liefert einen Server für
// PUT /api/kader/{id}.
func staffelFixture(t *testing.T, gender, ageClass string) (*sql.DB, string, int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	teamID := testutil.CreateTeam(t, db, "B-Jugend männlich")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`UPDATE kader SET gender = ?, age_class = ? WHERE id = ?`,
		gender, ageClass, kaderID); err != nil {
		t.Fatal(err)
	}
	adminID := testutil.CreateUser(t, db, "admin")
	return db, testutil.Token(t, adminID, "admin", nil), kaderID
}

func putStaffel(t *testing.T, db *sql.DB, token string, kaderID int, staffel any) *http.Response {
	t.Helper()
	h := NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/kader/{id}", h.UpdateKader)
	})
	return testutil.Put(t, srv, "/api/kader/"+itoa(kaderID), token,
		map[string]any{"staffel": staffel})
}

func staffelOf(t *testing.T, db *sql.DB, kaderID int) (string, bool) {
	t.Helper()
	var s sql.NullString
	if err := db.QueryRow(`SELECT staffel FROM kader WHERE id = ?`, kaderID).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s.String, s.Valid
}

// PUT /api/kader/{id} antwortet seit jeher mit 204 No Content. Dieser Change
// ändert den Vertrag nicht — die Staffel reiht sich in die bestehenden Felder
// ein, statt eine eigene Antwortform einzuführen.
func TestKaderStaffel_HappyPath(t *testing.T) {
	db, token, kaderID := staffelFixture(t, "m", "B-Jugend")
	resp := putStaffel(t, db, token, kaderID, "mB-RL-BW")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Status = %d, erwartet 204", resp.StatusCode)
	}
	if got, ok := staffelOf(t, db, kaderID); !ok || got != "mB-RL-BW" {
		t.Errorf("gespeicherte Staffel = %q (gesetzt: %v), erwartet mB-RL-BW", got, ok)
	}
}

func TestKaderStaffel_UnpassendesGeschlecht(t *testing.T) {
	db, token, kaderID := staffelFixture(t, "f", "B-Jugend")
	resp := putStaffel(t, db, token, kaderID, "mB-RL-BW")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Status = %d, erwartet 400", resp.StatusCode)
	}
	if _, ok := staffelOf(t, db, kaderID); ok {
		t.Error("bei 400 darf nichts gespeichert worden sein")
	}
}

func TestKaderStaffel_UnpassendeAltersklasse(t *testing.T) {
	db, token, kaderID := staffelFixture(t, "m", "C-Jugend")
	resp := putStaffel(t, db, token, kaderID, "mB-RL-BW")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Status = %d, erwartet 400", resp.StatusCode)
	}
}

func TestKaderStaffel_NichtInterpretierbarerCode(t *testing.T) {
	db, token, kaderID := staffelFixture(t, "m", "B-Jugend")
	resp := putStaffel(t, db, token, kaderID, "Hallenrunde Nord")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Status = %d, erwartet 400", resp.StatusCode)
	}
}

// Übungsgruppen treten in keiner Staffel an. 409, nicht 404: der Kader
// existiert, nur passt die Operation nicht zur Variante.
func TestKaderStaffel_UebungsgruppeAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	practiceID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	resp := putStaffel(t, db, token, practiceID, "mB-RL-BW")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Status = %d, erwartet 409", resp.StatusCode)
	}
	if _, ok := staffelOf(t, db, practiceID); ok {
		t.Error("an einer Übungsgruppe darf keine Staffel gespeichert werden")
	}
}

func TestKaderStaffel_LeerenEntferntZuordnung(t *testing.T) {
	db, token, kaderID := staffelFixture(t, "m", "B-Jugend")
	r1 := putStaffel(t, db, token, kaderID, "mB-RL-BW")
	r1.Body.Close()
	r2 := putStaffel(t, db, token, kaderID, "")
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusNoContent {
		t.Fatalf("Status = %d, erwartet 204", r2.StatusCode)
	}
	if _, ok := staffelOf(t, db, kaderID); ok {
		t.Error("leerer Code muss die Zuordnung auf NULL setzen")
	}
}

// Fehlt das Feld ganz, bleibt die Zuordnung unverändert (Tri-State).
func TestKaderStaffel_FehlendesFeldAendertNichts(t *testing.T) {
	db, token, kaderID := staffelFixture(t, "m", "B-Jugend")
	r1 := putStaffel(t, db, token, kaderID, "mB-RL-BW")
	r1.Body.Close()

	h := NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/kader/{id}", h.UpdateKader)
	})
	resp := testutil.Put(t, srv, "/api/kader/"+itoa(kaderID), token, map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Status = %d, erwartet 204", resp.StatusCode)
	}
	if got, ok := staffelOf(t, db, kaderID); !ok || got != "mB-RL-BW" {
		t.Errorf("Staffel = %q (gesetzt: %v), erwartet unverändert mB-RL-BW", got, ok)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
