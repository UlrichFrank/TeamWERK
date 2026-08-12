package settings_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/settings"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// defaultAusrichterID liefert die von Migration 048 geseedete Default-Zeile.
func defaultAusrichterID(t *testing.T, db *sql.DB) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`SELECT id FROM ausrichter WHERE is_default = 1`).Scan(&id); err != nil {
		t.Fatalf("Default-Ausrichter lesen: %v", err)
	}
	return id
}

// countDefaults zählt die Default-Zeilen. Die Invariante "genau eine" wird nach
// JEDEM Schreibpfad geprüft — sie ist die Voraussetzung dafür, dass die
// Auflösung total ist.
func countDefaults(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ausrichter WHERE is_default = 1`).Scan(&n); err != nil {
		t.Fatalf("COUNT defaults: %v", err)
	}
	return n
}

func countAusrichter(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ausrichter`).Scan(&n); err != nil {
		t.Fatalf("COUNT ausrichter: %v", err)
	}
	return n
}

// setSpieltagAusrichter schreibt einen Tageseintrag direkt. ausrichterID=0
// schreibt bewusst NULL — der Zustand, den das Löschen eines Ausrichters
// hinterlässt.
func setSpieltagAusrichter(t *testing.T, db *sql.DB, date string, seasonID, ausrichterID int) {
	t.Helper()
	var arg any
	if ausrichterID > 0 {
		arg = ausrichterID
	}
	if _, err := db.Exec(
		`INSERT INTO spieltag_ausrichter (date, season_id, ausrichter_id) VALUES (?, ?, ?)`,
		date, seasonID, arg); err != nil {
		t.Fatalf("spieltag_ausrichter schreiben: %v", err)
	}
}

// createTemplateItem legt eine Vorlagen-Zeile an, optional an einen Ausrichter
// gebunden (ausrichterID=0 → NULL, "gilt immer").
func createTemplateItem(t *testing.T, db *sql.DB, templateID, dutyTypeID, ausrichterID int) int {
	t.Helper()
	var arg any
	if ausrichterID > 0 {
		arg = ausrichterID
	}
	res, err := db.Exec(
		`INSERT INTO game_template_items (template_id, duty_type_id, slots_count, ausrichter_id)
		 VALUES (?, ?, 1, ?)`, templateID, dutyTypeID, arg)
	if err != nil {
		t.Fatalf("game_template_items schreiben: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func createTemplate(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO game_templates (name, template_type) VALUES (?, 'heim')`, name)
	if err != nil {
		t.Fatalf("game_templates schreiben: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// --- Auflösung: total in allen drei Ausgangslagen -------------------------

// TestResolveAusrichter_FehlendeZeile_LiefertDefault: der Regelfall — kein
// expliziter Eintrag, trotzdem ein Ergebnis (design.md Decision 2).
func TestResolveAusrichter_FehlendeZeile_LiefertDefault(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	want := defaultAusrichterID(t, db)

	id, explicit, err := settings.ResolveAusrichterForDayDetailed(context.Background(), db, "2026-09-14", seasonID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id != want {
		t.Errorf("erwartet Default-ID %d, bekam %d", want, id)
	}
	if explicit {
		t.Error("erwartet is_explicit=false bei fehlender Zeile")
	}
}

// TestResolveAusrichter_NullEintrag_LiefertDefault: ein Eintrag mit
// ausrichter_id IS NULL MUSS sich exakt wie eine fehlende Zeile verhalten —
// genau das hält den Zustand nach dem Löschen eines Ausrichters trivial korrekt.
func TestResolveAusrichter_NullEintrag_LiefertDefault(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	want := defaultAusrichterID(t, db)
	setSpieltagAusrichter(t, db, "2026-09-14", seasonID, 0)

	id, explicit, err := settings.ResolveAusrichterForDayDetailed(context.Background(), db, "2026-09-14", seasonID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id != want {
		t.Errorf("NULL-Eintrag muss auf den Default %d fallen, bekam %d", want, id)
	}
	if explicit {
		t.Error("erwartet is_explicit=false bei NULL-Eintrag")
	}
}

// TestResolveAusrichter_ExpliziterWert_Gewinnt.
func TestResolveAusrichter_ExpliziterWert_Gewinnt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	other, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	setSpieltagAusrichter(t, db, "2026-09-14", seasonID, other.ID)

	id, explicit, err := settings.ResolveAusrichterForDayDetailed(context.Background(), db, "2026-09-14", seasonID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id != other.ID {
		t.Errorf("erwartet expliziten Wert %d, bekam %d", other.ID, id)
	}
	if !explicit {
		t.Error("erwartet is_explicit=true bei explizitem Eintrag")
	}

	// Die schmale Variante liefert denselben Wert.
	plain, err := settings.ResolveAusrichterForDay(context.Background(), db, "2026-09-14", seasonID)
	if err != nil {
		t.Fatalf("ResolveAusrichterForDay: %v", err)
	}
	if plain != id {
		t.Errorf("ResolveAusrichterForDay soll %d liefern, bekam %d", id, plain)
	}
}

// TestResolveAusrichter_SaisonTrennung: derselbe Kalendertag in zwei Saisons
// kollidiert nicht — season_id gehört deshalb in den Primärschlüssel.
func TestResolveAusrichter_SaisonTrennung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonA := testutil.CreateSeason(t, db, "2025/26")
	seasonB := testutil.CreateSeason(t, db, "2026/27")
	defaultID := defaultAusrichterID(t, db)

	a, err := settings.CreateAusrichter(context.Background(), db, settings.AusrichterInput{Name: "Verein A"})
	if err != nil {
		t.Fatalf("CreateAusrichter A: %v", err)
	}
	b, err := settings.CreateAusrichter(context.Background(), db, settings.AusrichterInput{Name: "Verein B"})
	if err != nil {
		t.Fatalf("CreateAusrichter B: %v", err)
	}
	setSpieltagAusrichter(t, db, "2026-09-14", seasonA, a.ID)
	setSpieltagAusrichter(t, db, "2026-09-14", seasonB, b.ID)

	gotA, err := settings.ResolveAusrichterForDay(context.Background(), db, "2026-09-14", seasonA)
	if err != nil {
		t.Fatalf("Resolve Saison A: %v", err)
	}
	gotB, err := settings.ResolveAusrichterForDay(context.Background(), db, "2026-09-14", seasonB)
	if err != nil {
		t.Fatalf("Resolve Saison B: %v", err)
	}
	if gotA != a.ID || gotB != b.ID {
		t.Errorf("erwartet A=%d/B=%d, bekam A=%d/B=%d", a.ID, b.ID, gotA, gotB)
	}
	if gotA == defaultID || gotB == defaultID {
		t.Error("beide Saisons sollten ihren eigenen expliziten Wert liefern, nicht den Default")
	}
}

// TestResolveAusrichter_ISOTimestamp_MatchtTrotzdem: der SQLite-DATE-Gotcha —
// ein durchgereichtes "2026-09-14T00:00:00Z" darf nicht still am gespeicherten
// "2026-09-14" vorbeilaufen und den Default liefern.
func TestResolveAusrichter_ISOTimestamp_MatchtTrotzdem(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	other, err := settings.CreateAusrichter(context.Background(), db, settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	setSpieltagAusrichter(t, db, "2026-09-14", seasonID, other.ID)

	id, explicit, err := settings.ResolveAusrichterForDayDetailed(
		context.Background(), db, "2026-09-14T00:00:00Z", seasonID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if id != other.ID || !explicit {
		t.Errorf("erwartet %d/explicit, bekam %d/explicit=%v", other.ID, id, explicit)
	}
}

// TestResolveAusrichter_OhneDefault_LiefertFehler: fehlt die Default-Zeile
// (DB ohne Migration 048), ist das ein Datenfehler — die Funktion MUSS laut
// scheitern statt 0 zurückzugeben. Eine ID 0 würde gegen kein Item matchen und
// das Gate im Regen still alles ausfiltern lassen. Bewusst anders als
// GetBewirtungVerhaeltnis, wo ein fachlicher Ersatzwert existiert.
func TestResolveAusrichter_OhneDefault_LiefertFehler(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`DELETE FROM ausrichter`); err != nil {
		t.Fatalf("Default löschen: %v", err)
	}

	id, _, err := settings.ResolveAusrichterForDayDetailed(context.Background(), db, "2026-09-14", seasonID)
	if !errors.Is(err, settings.ErrNoDefaultAusrichter) {
		t.Errorf("erwartet ErrNoDefaultAusrichter, bekam %v", err)
	}
	if id != 0 {
		t.Errorf("erwartet id=0 im Fehlerfall, bekam %d", id)
	}
}

// --- Default-Invariante über alle Schreibpfade ----------------------------

// TestCreateAusrichter_AlsDefault_HaeltInvariante: nach dem Anlegen mit
// IsDefault trägt genau eine Zeile die Markierung, und es ist die neue.
func TestCreateAusrichter_AlsDefault_HaeltInvariante(t *testing.T) {
	db := testutil.NewDB(t)
	alt := defaultAusrichterID(t, db)

	neu, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen", IsDefault: true, SortOrder: 5})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	if !neu.IsDefault || neu.SortOrder != 5 {
		t.Errorf("Rückgabe unerwartet: %+v", neu)
	}
	if n := countDefaults(t, db); n != 1 {
		t.Fatalf("erwartet genau 1 Default, bekam %d", n)
	}
	if got := defaultAusrichterID(t, db); got != neu.ID {
		t.Errorf("erwartet neuen Default %d, bekam %d", neu.ID, got)
	}

	altRow, err := settings.GetAusrichter(context.Background(), db, alt)
	if err != nil {
		t.Fatalf("GetAusrichter alt: %v", err)
	}
	if altRow.IsDefault {
		t.Error("der bisherige Default muss seine Markierung verloren haben")
	}
}

// TestCreateAusrichter_OhneDefault_LaesstDefaultUnberuehrt.
func TestCreateAusrichter_OhneDefault_LaesstDefaultUnberuehrt(t *testing.T) {
	db := testutil.NewDB(t)
	alt := defaultAusrichterID(t, db)

	neu, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	if neu.IsDefault {
		t.Error("ohne IsDefault darf der neue Eintrag kein Default sein")
	}
	if n := countDefaults(t, db); n != 1 {
		t.Fatalf("erwartet genau 1 Default, bekam %d", n)
	}
	if got := defaultAusrichterID(t, db); got != alt {
		t.Errorf("Default sollte unverändert %d sein, bekam %d", alt, got)
	}
	if !neu.Aktiv {
		t.Error("ein frisch angelegter Ausrichter ist aktiv")
	}
}

// TestCreateAusrichter_Namensdublette_SchreibtNichts.
func TestCreateAusrichter_Namensdublette_SchreibtNichts(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"}); err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	before := countAusrichter(t, db)

	_, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen", IsDefault: true})
	if !errors.Is(err, settings.ErrDuplicateName) {
		t.Fatalf("erwartet ErrDuplicateName, bekam %v", err)
	}
	if got := countAusrichter(t, db); got != before {
		t.Errorf("es darf nichts geschrieben worden sein: vorher %d, nachher %d", before, got)
	}
	// Auch der Default-Wechsel der abgelehnten Anlage muss zurückgerollt sein.
	if n := countDefaults(t, db); n != 1 {
		t.Errorf("erwartet genau 1 Default, bekam %d", n)
	}
}

// TestCreateAusrichter_LeererName_SchreibtNichts.
func TestCreateAusrichter_LeererName_SchreibtNichts(t *testing.T) {
	db := testutil.NewDB(t)
	before := countAusrichter(t, db)

	if _, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "   "}); !errors.Is(err, settings.ErrEmptyName) {
		t.Fatalf("erwartet ErrEmptyName, bekam %v", err)
	}
	if got := countAusrichter(t, db); got != before {
		t.Errorf("es darf nichts geschrieben worden sein: vorher %d, nachher %d", before, got)
	}
}

// TestUpdateAusrichter_DefaultWechsel_EntziehtAltemDieMarkierung.
func TestUpdateAusrichter_DefaultWechsel_EntziehtAltemDieMarkierung(t *testing.T) {
	db := testutil.NewDB(t)
	alt := defaultAusrichterID(t, db)
	neu, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}

	yes := true
	updated, err := settings.UpdateAusrichter(context.Background(), db, neu.ID,
		settings.AusrichterUpdate{IsDefault: &yes})
	if err != nil {
		t.Fatalf("UpdateAusrichter: %v", err)
	}
	if !updated.IsDefault {
		t.Error("Rückgabe soll den neuen Default-Zustand spiegeln")
	}
	if n := countDefaults(t, db); n != 1 {
		t.Fatalf("erwartet genau 1 Default, bekam %d", n)
	}
	if got := defaultAusrichterID(t, db); got != neu.ID {
		t.Errorf("erwartet Default %d, bekam %d", neu.ID, got)
	}

	// Und die Auflösung folgt dem Wechsel sofort.
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	got, err := settings.ResolveAusrichterForDay(context.Background(), db, "2026-09-14", seasonID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != neu.ID {
		t.Errorf("Auflösung soll dem neuen Default %d folgen, bekam %d (alt: %d)", neu.ID, got, alt)
	}
}

// TestUpdateAusrichter_DefaultMarkierungEntziehen_Abgelehnt: den Default auf
// is_default=0 zu setzen ließe null Defaults zurück — die Auflösung wäre nicht
// mehr total. Verlagern (anderen Eintrag markieren) ist der einzige Weg.
func TestUpdateAusrichter_DefaultMarkierungEntziehen_Abgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	defaultID := defaultAusrichterID(t, db)

	no := false
	name := "Umbenannt"
	_, err := settings.UpdateAusrichter(context.Background(), db, defaultID,
		settings.AusrichterUpdate{IsDefault: &no, Name: &name})
	if !errors.Is(err, settings.ErrDefaultRequired) {
		t.Fatalf("erwartet ErrDefaultRequired, bekam %v", err)
	}
	if n := countDefaults(t, db); n != 1 {
		t.Errorf("erwartet genau 1 Default, bekam %d", n)
	}
	// Das im selben Request mitgesendete, für sich gültige Feld darf nicht
	// halb durchgeschrieben worden sein.
	row, err := settings.GetAusrichter(context.Background(), db, defaultID)
	if err != nil {
		t.Fatalf("GetAusrichter: %v", err)
	}
	if row.Name == "Umbenannt" {
		t.Error("bei abgelehntem Update darf der Name nicht geschrieben worden sein")
	}
}

// TestUpdateAusrichter_DefaultDeaktivieren_Abgelehnt: ein inaktiver Default
// wäre der Wert, auf den jeder ungepflegte Spieltag auflöst — und zugleich
// einer, den man explizit gar nicht setzen dürfte.
func TestUpdateAusrichter_DefaultDeaktivieren_Abgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	defaultID := defaultAusrichterID(t, db)

	no := false
	_, err := settings.UpdateAusrichter(context.Background(), db, defaultID,
		settings.AusrichterUpdate{Aktiv: &no})
	if !errors.Is(err, settings.ErrDefaultRequired) {
		t.Fatalf("erwartet ErrDefaultRequired, bekam %v", err)
	}
	row, err := settings.GetAusrichter(context.Background(), db, defaultID)
	if err != nil {
		t.Fatalf("GetAusrichter: %v", err)
	}
	if !row.Aktiv {
		t.Error("der Default muss aktiv geblieben sein")
	}
}

// TestUpdateAusrichter_Felder_PointerSemantik: ein fehlendes Feld bleibt
// unverändert.
func TestUpdateAusrichter_Felder_PointerSemantik(t *testing.T) {
	db := testutil.NewDB(t)
	a, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen", SortOrder: 3})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}

	no := false
	updated, err := settings.UpdateAusrichter(context.Background(), db, a.ID,
		settings.AusrichterUpdate{Aktiv: &no})
	if err != nil {
		t.Fatalf("UpdateAusrichter: %v", err)
	}
	if updated.Aktiv {
		t.Error("aktiv sollte 0 sein")
	}
	if updated.Name != "TV Ötlingen" || updated.SortOrder != 3 {
		t.Errorf("nicht gesendete Felder müssen unverändert bleiben, bekam %+v", updated)
	}
}

// TestUpdateAusrichter_Namensdublette_SchreibtNichts: der gleichzeitig
// gesendete, für sich gültige sort_order darf nicht hängenbleiben.
func TestUpdateAusrichter_Namensdublette_SchreibtNichts(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "Verein A"}); err != nil {
		t.Fatalf("CreateAusrichter A: %v", err)
	}
	b, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "Verein B", SortOrder: 2})
	if err != nil {
		t.Fatalf("CreateAusrichter B: %v", err)
	}

	name := "Verein A"
	sort := 99
	_, err = settings.UpdateAusrichter(context.Background(), db, b.ID,
		settings.AusrichterUpdate{Name: &name, SortOrder: &sort})
	if !errors.Is(err, settings.ErrDuplicateName) {
		t.Fatalf("erwartet ErrDuplicateName, bekam %v", err)
	}
	row, err := settings.GetAusrichter(context.Background(), db, b.ID)
	if err != nil {
		t.Fatalf("GetAusrichter: %v", err)
	}
	if row.Name != "Verein B" || row.SortOrder != 2 {
		t.Errorf("nichts darf geschrieben worden sein, bekam %+v", row)
	}
}

// TestUpdateAusrichter_Unbekannt_LiefertNotFound.
func TestUpdateAusrichter_Unbekannt_LiefertNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	name := "Egal"
	if _, err := settings.UpdateAusrichter(context.Background(), db, 99999,
		settings.AusrichterUpdate{Name: &name}); !errors.Is(err, settings.ErrAusrichterNotFound) {
		t.Errorf("erwartet ErrAusrichterNotFound, bekam %v", err)
	}
}

// --- Löschen: die asymmetrische Kaskade ----------------------------------

// TestDeleteAusrichter_Kaskade: Spieltage werden ENTKOPPELT (auf NULL, danach
// löst der Tag auf den Default auf), gebundene Vorlagen-Zeilen werden
// MITGELÖSCHT. Ein SET NULL auf den Items würde sie auf "gilt immer" heben und
// nach dem Löschen MEHR Dienste erzeugen als vorher (design.md Decision 6).
func TestDeleteAusrichter_Kaskade(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	defaultID := defaultAusrichterID(t, db)

	victim, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}

	dates := []string{"2026-09-14", "2026-09-21", "2026-09-28"}
	for _, d := range dates {
		setSpieltagAusrichter(t, db, d, seasonID, victim.ID)
	}

	templateID := createTemplate(t, db, "Heimspiel Standard")
	dutyTypeID := testutil.CreateDutyType(t, db, "Kuchen", 1)
	otherDutyTypeID := testutil.CreateDutyType(t, db, "Kasse", 1)
	boundA := createTemplateItem(t, db, templateID, dutyTypeID, victim.ID)
	boundB := createTemplateItem(t, db, templateID, otherDutyTypeID, victim.ID)
	unbound := createTemplateItem(t, db, templateID, dutyTypeID, 0)

	if err := settings.DeleteAusrichter(context.Background(), db, victim.ID); err != nil {
		t.Fatalf("DeleteAusrichter: %v", err)
	}

	// 1. Spieltage überleben, entkoppelt.
	var dayCount, nullCount int
	if err := db.QueryRow(
		`SELECT COUNT(*), SUM(CASE WHEN ausrichter_id IS NULL THEN 1 ELSE 0 END)
		 FROM spieltag_ausrichter WHERE season_id = ?`, seasonID).Scan(&dayCount, &nullCount); err != nil {
		t.Fatalf("spieltag_ausrichter zählen: %v", err)
	}
	if dayCount != len(dates) || nullCount != len(dates) {
		t.Errorf("erwartet %d Spieltage, alle mit ausrichter_id IS NULL; bekam %d Zeilen / %d NULL",
			len(dates), dayCount, nullCount)
	}
	// ... und lösen danach auf den Default auf.
	for _, d := range dates {
		id, explicit, err := settings.ResolveAusrichterForDayDetailed(context.Background(), db, d, seasonID)
		if err != nil {
			t.Fatalf("Resolve %s: %v", d, err)
		}
		if id != defaultID || explicit {
			t.Errorf("%s: erwartet Default %d/is_explicit=false, bekam %d/%v", d, defaultID, id, explicit)
		}
	}

	// 2. Gebundene Vorlagen-Zeilen sind WEG — nicht auf NULL gesetzt.
	for _, itemID := range []int{boundA, boundB} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE id = ?`, itemID).Scan(&n); err != nil {
			t.Fatalf("item zählen: %v", err)
		}
		if n != 0 {
			t.Errorf("gebundenes Item %d existiert noch — ein SET NULL würde es auf 'gilt immer' heben", itemID)
		}
	}
	// 3. Die ungebundene Zeile bleibt unangetastet.
	var unboundLeft int
	if err := db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE id = ? AND ausrichter_id IS NULL`,
		unbound).Scan(&unboundLeft); err != nil {
		t.Fatalf("unbound zählen: %v", err)
	}
	if unboundLeft != 1 {
		t.Error("die ungebundene Vorlagen-Zeile darf nicht mitgelöscht werden")
	}

	// 4. Kein verwaistes ausrichter_id bleibt zurück.
	var orphans int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM game_template_items WHERE ausrichter_id = ?`, victim.ID).Scan(&orphans); err != nil {
		t.Fatalf("orphans zählen: %v", err)
	}
	if orphans != 0 {
		t.Errorf("erwartet 0 verwaiste Bindungen, bekam %d", orphans)
	}
}

// TestDeleteAusrichter_Default_Abgelehnt: ohne Default wäre die Auflösung nicht
// mehr total. Es darf dabei auch nichts anderes geschrieben worden sein.
func TestDeleteAusrichter_Default_Abgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	defaultID := defaultAusrichterID(t, db)
	setSpieltagAusrichter(t, db, "2026-09-14", seasonID, defaultID)

	templateID := createTemplate(t, db, "Heimspiel Standard")
	dutyTypeID := testutil.CreateDutyType(t, db, "Kuchen", 1)
	itemID := createTemplateItem(t, db, templateID, dutyTypeID, defaultID)

	err := settings.DeleteAusrichter(context.Background(), db, defaultID)
	if !errors.Is(err, settings.ErrDefaultUndeletable) {
		t.Fatalf("erwartet ErrDefaultUndeletable, bekam %v", err)
	}

	if _, err := settings.GetAusrichter(context.Background(), db, defaultID); err != nil {
		t.Errorf("der Default muss unverändert existieren: %v", err)
	}
	var bound int
	if err := db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE id = ? AND ausrichter_id = ?`,
		itemID, defaultID).Scan(&bound); err != nil {
		t.Fatalf("item zählen: %v", err)
	}
	if bound != 1 {
		t.Error("die Vorlagen-Zeile darf bei abgelehntem Löschen nicht angefasst werden")
	}
	var stillSet int
	if err := db.QueryRow(`SELECT COUNT(*) FROM spieltag_ausrichter WHERE ausrichter_id = ?`,
		defaultID).Scan(&stillSet); err != nil {
		t.Fatalf("spieltag zählen: %v", err)
	}
	if stillSet != 1 {
		t.Error("der Spieltag darf bei abgelehntem Löschen nicht entkoppelt werden")
	}
}

// TestDeleteAusrichter_Unbekannt_LiefertNotFound.
func TestDeleteAusrichter_Unbekannt_LiefertNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	if err := settings.DeleteAusrichter(context.Background(), db, 99999); !errors.Is(err, settings.ErrAusrichterNotFound) {
		t.Errorf("erwartet ErrAusrichterNotFound, bekam %v", err)
	}
}

// --- Liste und Verwendungsübersicht --------------------------------------

// TestListAusrichter_IncludeInactive.
func TestListAusrichter_IncludeInactive(t *testing.T) {
	db := testutil.NewDB(t)
	a, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen", SortOrder: 1})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	no := false
	if _, err := settings.UpdateAusrichter(context.Background(), db, a.ID,
		settings.AusrichterUpdate{Aktiv: &no}); err != nil {
		t.Fatalf("UpdateAusrichter: %v", err)
	}

	active, err := settings.ListAusrichter(context.Background(), db, false)
	if err != nil {
		t.Fatalf("ListAusrichter: %v", err)
	}
	for _, it := range active {
		if it.ID == a.ID {
			t.Error("ein deaktivierter Eintrag darf ohne includeInactive nicht erscheinen")
		}
	}
	all, err := settings.ListAusrichter(context.Background(), db, true)
	if err != nil {
		t.Fatalf("ListAusrichter(all): %v", err)
	}
	if len(all) != len(active)+1 {
		t.Errorf("erwartet einen Eintrag mehr mit includeInactive: %d vs %d", len(all), len(active))
	}
}

// TestAusrichterUsage_BenenntBeideReferenzen: die Vorab-Anzeige muss beide
// Seiten der Asymmetrie zeigen — Spieltage (überleben) und Vorlagen-Zeilen
// (verschwinden mit).
func TestAusrichterUsage_BenenntBeideReferenzen(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	a, err := settings.CreateAusrichter(context.Background(), db,
		settings.AusrichterInput{Name: "TV Ötlingen"})
	if err != nil {
		t.Fatalf("CreateAusrichter: %v", err)
	}
	setSpieltagAusrichter(t, db, "2026-09-14", seasonID, a.ID)

	templateID := createTemplate(t, db, "Heimspiel Standard")
	dutyTypeID := testutil.CreateDutyType(t, db, "Kuchen", 1)
	createTemplateItem(t, db, templateID, dutyTypeID, a.ID)
	createTemplateItem(t, db, templateID, dutyTypeID, 0) // ungebunden, gehört nicht dazu

	usage, err := settings.AusrichterUsage(context.Background(), db, a.ID)
	if err != nil {
		t.Fatalf("AusrichterUsage: %v", err)
	}
	if len(usage.GameDays) != 1 {
		t.Fatalf("erwartet 1 Spieltag, bekam %d", len(usage.GameDays))
	}
	if usage.GameDays[0].Date != "2026-09-14" || usage.GameDays[0].SeasonID != seasonID {
		t.Errorf("Spieltag unerwartet: %+v", usage.GameDays[0])
	}
	if usage.GameDays[0].SeasonName != "2025/26" {
		t.Errorf("erwartet Saisonname '2025/26', bekam %q", usage.GameDays[0].SeasonName)
	}
	if len(usage.TemplateItems) != 1 {
		t.Fatalf("erwartet 1 gebundene Vorlagen-Zeile, bekam %d", len(usage.TemplateItems))
	}
	item := usage.TemplateItems[0]
	if item.TemplateName != "Heimspiel Standard" || item.DutyTypeName != "Kuchen" {
		t.Errorf("Vorlagen-Zeile soll Vorlagen- und Dienstnamen tragen, bekam %+v", item)
	}
}

// TestAusrichterUsage_Unbekannt_LiefertNotFound.
func TestAusrichterUsage_Unbekannt_LiefertNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := settings.AusrichterUsage(context.Background(), db, 99999); !errors.Is(err, settings.ErrAusrichterNotFound) {
		t.Errorf("erwartet ErrAusrichterNotFound, bekam %v", err)
	}
}

// TestGetAusrichter_Unbekannt_LiefertNotFound.
func TestGetAusrichter_Unbekannt_LiefertNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := settings.GetAusrichter(context.Background(), db, 99999); !errors.Is(err, settings.ErrAusrichterNotFound) {
		t.Errorf("erwartet ErrAusrichterNotFound, bekam %v", err)
	}
}
