package notify

import (
	"database/sql"
	"fmt"
	"sort"
	"testing"
)

// --- Fixtures -------------------------------------------------------------
//
// Bewusst rohes SQL statt internal/testutil: testutil baut den vollen Server
// und importiert damit auch notify — als interner Test wäre das ein Zyklus.
// Dieselbe Entscheidung wie in notify_test.go, dessen newTestDB/insertUser hier
// mitbenutzt werden.

func insertSeason(t *testing.T, db *sql.DB, name string, active bool) int {
	t.Helper()
	flag := 0
	if active {
		flag = 1
	}
	res, err := db.Exec(
		`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES (?, '2025-09-01', '2026-06-30', ?)`,
		name, flag)
	if err != nil {
		t.Fatalf("insert season: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func insertTeam(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO teams (name, age_class, gender) VALUES (?, 'Erwachsene', 'mixed')`, name)
	if err != nil {
		t.Fatalf("insert team: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func insertKader(t *testing.T, db *sql.DB, teamID, seasonID int) int {
	t.Helper()
	var maxNum int
	db.QueryRow(`SELECT COALESCE(MAX(team_number), 0) FROM kader WHERE season_id=?`, seasonID).Scan(&maxNum)
	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, 'Erwachsene', 'mixed', ?, ?)`,
		seasonID, teamID, maxNum+1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertMember legt ein Mitglied an; userID=0 erzeugt eines ohne Nutzerkonto.
func insertMember(t *testing.T, db *sql.DB, name string, userID int) int {
	t.Helper()
	var userArg any
	if userID > 0 {
		userArg = userID
	}
	res, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, user_id) VALUES ('Test', ?, 'aktiv', ?)`,
		name, userArg)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func addKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_members (kader_id, member_id) VALUES (?,?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_member: %v", err)
	}
}

func addExtendedMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?,?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_extended_member: %v", err)
	}
}

func addKaderTrainer(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?,?)`, kaderID, memberID); err != nil {
		t.Fatalf("insert kader_trainer: %v", err)
	}
}

func addFamilyLink(t *testing.T, db *sql.DB, parentUserID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, parentUserID, memberID); err != nil {
		t.Fatalf("insert family_link: %v", err)
	}
}

func sorted(ids []int) []int {
	out := append([]int(nil), ids...)
	sort.Ints(out)
	return out
}

func equalIDs(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	g, w := sorted(got), sorted(want)
	for i := range g {
		if g[i] != w[i] {
			return false
		}
	}
	return true
}

// --- Tests ----------------------------------------------------------------

// Der Kern des Changes: Stammkader UND erweiterter Kader jeweils samt Eltern,
// dazu die Trainer. Vor diesem Change fehlte je nach Meldungsweg eine dieser
// Gruppen — die Trainer an jeder Stelle.
func TestTeamAudience_FuenfGruppen(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)
	team := insertTeam(t, db, "mB1")
	kader := insertKader(t, db, team, season)

	uPlayer := insertUser(t, db, "player@test.local")
	uPlayerParent := insertUser(t, db, "playerparent@test.local")
	uExt := insertUser(t, db, "ext@test.local")
	uExtParent := insertUser(t, db, "extparent@test.local")
	uTrainer := insertUser(t, db, "trainer@test.local")
	uStranger := insertUser(t, db, "stranger@test.local")

	mPlayer := insertMember(t, db, "Player", uPlayer)
	mExt := insertMember(t, db, "Ext", uExt)
	mTrainer := insertMember(t, db, "Trainer", uTrainer)
	mStranger := insertMember(t, db, "Stranger", uStranger)

	addKaderMember(t, db, kader, mPlayer)
	addExtendedMember(t, db, kader, mExt)
	addKaderTrainer(t, db, kader, mTrainer)
	addFamilyLink(t, db, uPlayerParent, mPlayer)
	addFamilyLink(t, db, uExtParent, mExt)
	_ = mStranger // im Verein, aber in keinem Kader dieser Mannschaft

	got := TeamAudience(db, team)
	want := []int{uPlayer, uPlayerParent, uExt, uExtParent, uTrainer}
	if !equalIDs(got, want) {
		t.Errorf("TeamAudience = %v, want %v (Stammkader + erw. Kader + Eltern beider + Trainer)", sorted(got), sorted(want))
	}
	for _, id := range got {
		if id == uStranger {
			t.Errorf("TeamAudience enthält ein Mitglied ohne Kader-Zugehörigkeit (user %d)", uStranger)
		}
	}
}

func TestTeamAudience_KeineDoppeltenEmpfaenger(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)
	teamA := insertTeam(t, db, "mB1")
	teamB := insertTeam(t, db, "mB2")
	kaderA := insertKader(t, db, teamA, season)
	kaderB := insertKader(t, db, teamB, season)

	uBoth := insertUser(t, db, "both@test.local")
	uParent := insertUser(t, db, "parent@test.local")
	mBoth := insertMember(t, db, "Both", uBoth)

	// In beiden Listen desselben Kaders UND zusätzlich im Kader der zweiten
	// betroffenen Mannschaft — drei Gründe für dieselbe Meldung.
	addKaderMember(t, db, kaderA, mBoth)
	addExtendedMember(t, db, kaderA, mBoth)
	addExtendedMember(t, db, kaderB, mBoth)
	addFamilyLink(t, db, uParent, mBoth)

	got := TeamAudience(db, teamA, teamB)
	if !equalIDs(got, []int{uBoth, uParent}) {
		t.Fatalf("TeamAudience = %v, want genau [%d %d] ohne Dubletten", sorted(got), uBoth, uParent)
	}
}

func TestTeamAudience_MitgliedOhneNutzerkonto(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)
	team := insertTeam(t, db, "mB1")
	kader := insertKader(t, db, team, season)

	mNoUser := insertMember(t, db, "NoUser", 0)
	addExtendedMember(t, db, kader, mNoUser)

	uParent := insertUser(t, db, "parent@test.local")
	addFamilyLink(t, db, uParent, mNoUser)

	got := TeamAudience(db, team)
	// Das Kind hat kein Konto und kann keine Meldung bekommen; das Elternteil
	// hat eines und muss sie bekommen.
	if !equalIDs(got, []int{uParent}) {
		t.Errorf("TeamAudience = %v, want nur das Elternteil %d", sorted(got), uParent)
	}
}

func TestTeamAudience_NichtAktiveSaisonZaehltNicht(t *testing.T) {
	db := newTestDB(t)
	oldSeason := insertSeason(t, db, "2024/25", false)
	newSeason := insertSeason(t, db, "2025/26", true)
	team := insertTeam(t, db, "mB1")
	oldKader := insertKader(t, db, team, oldSeason)
	newKader := insertKader(t, db, team, newSeason)

	uGone := insertUser(t, db, "gone@test.local")
	uCurrent := insertUser(t, db, "current@test.local")
	mGone := insertMember(t, db, "Gone", uGone)
	mCurrent := insertMember(t, db, "Current", uCurrent)

	addKaderMember(t, db, oldKader, mGone)
	addExtendedMember(t, db, newKader, mCurrent)

	got := TeamAudience(db, team)
	if !equalIDs(got, []int{uCurrent}) {
		t.Errorf("TeamAudience = %v, want nur %d — der Kader der Vorsaison darf nicht zählen", sorted(got), uCurrent)
	}
}

func TestTeamAudience_LeereTeamListe(t *testing.T) {
	db := newTestDB(t)
	if got := TeamAudience(db); len(got) != 0 {
		t.Errorf("TeamAudience() ohne Teams = %v, want leer", got)
	}
}

func TestTeamAudience_UnbekanntesTeamLiefertLeer(t *testing.T) {
	db := newTestDB(t)
	insertSeason(t, db, "2025/26", true)
	if got := TeamAudience(db, 999); len(got) != 0 {
		t.Errorf("TeamAudience(999) = %v, want leer", got)
	}
}

// Ein Query-Fehler darf keine halbe Empfängerliste liefern — lieber keine
// Meldung als eine an einen willkürlichen Teil der Mannschaft.
func TestTeamAudience_QueryFehlerLiefertLeer(t *testing.T) {
	db := newTestDB(t)
	if _, err := db.Exec(`DROP TABLE kader_extended_members`); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if got := TeamAudience(db, 1); got != nil {
		t.Errorf("TeamAudience nach Query-Fehler = %v, want nil", got)
	}
}

// Sanity: die Platzhalter-Liste wird für alle vier Zweige korrekt vervielfacht.
// Ein falsch gezähltes Argument-Set fällt sonst erst bei mehreren Teams auf.
func TestTeamAudience_MehrereTeams(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)

	var teams []int
	var want []int
	for i := range 3 {
		team := insertTeam(t, db, fmt.Sprintf("team%d", i))
		kader := insertKader(t, db, team, season)
		u := insertUser(t, db, fmt.Sprintf("p%d@test.local", i))
		m := insertMember(t, db, fmt.Sprintf("P%d", i), u)
		addKaderMember(t, db, kader, m)
		teams = append(teams, team)
		want = append(want, u)
	}

	if got := TeamAudience(db, teams...); !equalIDs(got, want) {
		t.Errorf("TeamAudience = %v, want %v", sorted(got), sorted(want))
	}
}

// Trainer stehen in ihrer Funktion in der Menge, nicht als Kind: ein an einem
// Trainer hängender family_link darf keinen Empfänger erzeugen — sonst bekäme
// die Mutter eines volljährigen Trainers dessen Mannschaftsmeldungen.
func TestTeamAudience_TrainerOhneElternZweig(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)
	team := insertTeam(t, db, "mB1")
	kader := insertKader(t, db, team, season)

	uTrainer := insertUser(t, db, "trainer@test.local")
	mTrainer := insertMember(t, db, "Trainer", uTrainer)
	addKaderTrainer(t, db, kader, mTrainer)

	uTrainerParent := insertUser(t, db, "trainerparent@test.local")
	addFamilyLink(t, db, uTrainerParent, mTrainer)

	got := TeamAudience(db, team)
	if !equalIDs(got, []int{uTrainer}) {
		t.Errorf("TeamAudience = %v, want nur den Trainer %d (kein Eltern-Zweig für Trainer)", sorted(got), uTrainer)
	}
}

// Ein Trainer, der zugleich im Kader spielt, ist ein Empfänger, nicht zwei.
func TestTeamAudience_TrainerUndSpielerInEinerPerson(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)
	team := insertTeam(t, db, "mB1")
	kader := insertKader(t, db, team, season)

	u := insertUser(t, db, "spielertrainer@test.local")
	m := insertMember(t, db, "SpielerTrainer", u)
	addKaderMember(t, db, kader, m)
	addKaderTrainer(t, db, kader, m)

	if got := TeamAudience(db, team); !equalIDs(got, []int{u}) {
		t.Errorf("TeamAudience = %v, want genau [%d]", sorted(got), u)
	}
}
