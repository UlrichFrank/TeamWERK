package games_test

// Tests für die Ausrichter-Überschreibungen im Massenlauf
// (heimspieltag-ausrichter, Tasks 7.1–7.3). Nutzt die Fixtures aus
// ausrichter_handler_test.go (newHostFixture, gebundenerTagMitZusage,
// defaultAusrichterID) und die Werkzeuge aus bulkregen_handler_test.go
// (decodeBulkRegen, dbFingerprint) — die Vorschau-Garantie ist dieselbe wie bei
// den Vorlagen-Änderungen und wird deshalb auch gleich bewiesen.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// bulkHostBody baut einen Massenlauf-Request über den Tag der Fixture, ohne
// defaults/overrides: jeder Termin behält sein gespeichertes Template, die
// einzige Änderung ist der Ausrichter.
func bulkHostBody(f hostFixture, hostOverrides []map[string]any) map[string]any {
	body := map[string]any{"from": f.date, "to": f.date}
	if hostOverrides != nil {
		body["host_overrides"] = hostOverrides
	}
	return body
}

func findDay(t *testing.T, got bulkRegenResponseDTO, date string) bulkRegenDayDTO {
	t.Helper()
	for _, d := range got.Days {
		if d.Date == date {
			return d
		}
	}
	t.Fatalf("kein days-Eintrag für %s, got %+v", date, got.Days)
	return bulkRegenDayDTO{}
}

// --- Vorschau weist die Wirkung aus und schreibt nichts ---------------------------------

// Spec-Szenario "Änderung im Dialog erscheint in der Bilanz": ein host_override
// gatet die gebundene Vorlagen-Zeile aus — die Vorschau meldet den entfallenden
// Dienst samt Zusage, persistiert aber nichts.
func TestBulkRegenPreview_HostOverride_WirkungOhneSchreiben(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	ausrichterA, _, _ := gebundenerTagMitZusage(t, f, srv)
	defaultID := defaultAusrichterID(t, f.db)

	before := dbFingerprint(t, f.db)
	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", f.token,
		bulkHostBody(f, []map[string]any{{"date": f.date, "ausrichter_id": defaultID}}))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	got := decodeBulkRegen(t, res)

	if got.Applied {
		t.Errorf("Vorschau darf applied=true nicht melden")
	}
	if got.Totals.Deleted != 1 || got.Totals.AssignmentsLost != 1 {
		t.Errorf("erwartet 1 entfallenden Dienst mit 1 verlorenen Zusage, got totals=%+v", got.Totals)
	}
	day := findDay(t, got, f.date)
	if day.EffectiveAusrichterID != defaultID || !day.IsExplicit {
		t.Errorf("erwartet wirksamen Ausrichter %d (explizit), got %+v", defaultID, day)
	}
	if day.StoredAusrichterID == nil || *day.StoredAusrichterID != ausrichterA {
		t.Errorf("erwartet gespeicherten Ausrichter %d, got %+v", ausrichterA, day.StoredAusrichterID)
	}
	if after := dbFingerprint(t, f.db); after != before {
		t.Errorf("Vorschau hat geschrieben:\n--- vorher ---\n%s\n--- nachher ---\n%s", before, after)
	}
}

// --- Apply persistiert den Tageswert und entspricht der Vorschau ------------------------

// Spec-Szenario "Ausrichter-Überschreibung wirkt auf die erzeugten Dienste":
// derselbe Request einmal als Vorschau, einmal als Apply — gleiche Bilanz, und
// danach steht der Tageswert in spieltag_ausrichter.
func TestBulkRegenApply_HostOverride_PersistiertUndEntsprichtVorschau(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	gebundenerTagMitZusage(t, f, srv)
	defaultID := defaultAusrichterID(t, f.db)
	body := bulkHostBody(f, []map[string]any{{"date": f.date, "ausrichter_id": defaultID}})

	previewRes := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", f.token, body)
	defer previewRes.Body.Close()
	preview := decodeBulkRegen(t, previewRes)

	applyRes := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", f.token, body)
	defer applyRes.Body.Close()
	if applyRes.StatusCode != http.StatusOK {
		t.Fatalf("apply erwartet 200, got %d", applyRes.StatusCode)
	}
	apply := decodeBulkRegen(t, applyRes)

	if !apply.Applied {
		t.Errorf("apply muss applied=true melden")
	}
	if apply.Totals != preview.Totals {
		t.Errorf("Bilanz weicht ab:\nVorschau %+v\nApply    %+v", preview.Totals, apply.Totals)
	}
	var stored int
	if err := f.db.QueryRow(
		`SELECT ausrichter_id FROM spieltag_ausrichter WHERE date=? AND season_id=?`,
		f.date, f.seasonID).Scan(&stored); err != nil {
		t.Fatalf("spieltag_ausrichter nach apply: %v", err)
	}
	if stored != defaultID {
		t.Errorf("erwartet gespeicherten Ausrichter %d, got %d", defaultID, stored)
	}
	// Die gebundene Zeile gilt nur für A — nach dem Wechsel gibt es den Dienst nicht mehr.
	if n := countRows(t, f.db, "duty_slots", "game_id=?", f.gameID); n != 0 {
		t.Errorf("erwartet 0 Slots nach dem ausgegateten Lauf, got %d", n)
	}
}

// --- Ohne host_overrides bleibt der gespeicherte Wert unangetastet ----------------------

// Spec-Szenario "Ohne host_override bleibt der gespeicherte Ausrichter wirksam":
// ein Lauf ohne Ausrichter-Angabe darf spieltag_ausrichter nicht anfassen — auch
// nicht, um den geerbten Default festzuschreiben.
func TestBulkRegenApply_OhneHostOverrides_SpieltagAusrichterUnveraendert(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	defaultID := defaultAusrichterID(t, f.db)

	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", f.token, bulkHostBody(f, nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	got := decodeBulkRegen(t, res)

	day := findDay(t, got, f.date)
	if day.EffectiveAusrichterID != defaultID {
		t.Errorf("erwartet geerbten Default %d, got %d", defaultID, day.EffectiveAusrichterID)
	}
	if day.IsExplicit {
		t.Errorf("ein Tag ohne expliziten Eintrag muss is_explicit=false melden, got %+v", day)
	}
	if day.StoredAusrichterID != nil {
		t.Errorf("erwartet keinen gespeicherten Wert, got %d", *day.StoredAusrichterID)
	}
	if n := countRows(t, f.db, "spieltag_ausrichter", "season_id=?", f.seasonID); n != 0 {
		t.Errorf("erwartet 0 Zeilen in spieltag_ausrichter, got %d", n)
	}
	// Der Dienst des Tages bleibt: ohne Bindung greift das Gate nicht.
	if n := countRows(t, f.db, "duty_slots", "game_id=?", f.gameID); n != 1 {
		t.Errorf("erwartet 1 unveränderten Slot, got %d", n)
	}
}

// --- Validierung ------------------------------------------------------------------------

// Spec-Szenario "Unbekannter Ausrichter wird abgelehnt" — inklusive der
// Reihenfolge-Garantie: die Ablehnung passiert vor dem ersten Schreibvorgang,
// die template_id-Updates der Vorlagen-Ebene dürfen nicht teilweise stehen.
func TestBulkRegenPreview_UnbekannterAusrichter_400(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	before := dbFingerprint(t, f.db)

	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/apply", f.token,
		bulkHostBody(f, []map[string]any{{"date": f.date, "ausrichter_id": 999999}}))
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("erwartet 400, got %d", res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.Error != "unknown_ausrichter" {
		t.Errorf("erwartet 'unknown_ausrichter', got %q", body.Error)
	}
	if after := dbFingerprint(t, f.db); after != before {
		t.Errorf("abgelehnter Request hat geschrieben:\n--- vorher ---\n%s\n--- nachher ---\n%s", before, after)
	}
}

// Ein Override außerhalb des Zeitraums wird abgelehnt statt still ignoriert: er
// würde einen Tageswert setzen, den dieser Lauf nicht regeneriert — die Bilanz
// zeigte die Wirkung dann nicht.
func TestBulkRegenPreview_HostOverrideAusserhalbZeitraum_400(t *testing.T) {
	f := newHostFixture(t)
	srv := prodserver.New(t, f.db)
	defaultID := defaultAusrichterID(t, f.db)

	res := testutil.Post(t, srv, "/api/duty-slots/bulk-regen/preview", f.token,
		bulkHostBody(f, []map[string]any{{"date": bulkFutureDate(t, 40), "ausrichter_id": defaultID}}))
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("erwartet 400, got %d", res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.Error != "host_override_out_of_range" {
		t.Errorf("erwartet 'host_override_out_of_range', got %q", body.Error)
	}
}
