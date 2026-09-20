package gamestats

import (
	"context"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// at baut einen Zeitpunkt in Berliner Zeit.
func at(t *testing.T, date, clock string) time.Time {
	t.Helper()
	return timez.ParseDT(date, clock, timez.Berlin())
}

// seedGame legt eine Begegnung mit gegebenem Datum, Anwurf und sGID an.
func seedGame(t *testing.T, s *Store, staffelID int, no, date, clock, sgid string) {
	t.Helper()
	_, err := s.db.Exec(`
		INSERT INTO bwhv_games (staffel_id, game_no, sgid, date, time, home_team, guest_team)
		VALUES (?,?,?,?,?,'A','B')`, staffelID, no, sgid, date, clock)
	if err != nil {
		t.Fatalf("seedGame: %v", err)
	}
}

func TestPollFenster_VorAnwurfPlusZweiStundenKeinAbruf(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-19", "16:00", "")

	due, err := s.DueStaffeln(context.Background(), seasonID, at(t, "2026-09-19", "17:30"))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Errorf("fällig = %d, erwartet 0 — der Anwurf liegt erst 1,5 h zurück", len(due))
	}
}

func TestPollFenster_NachAnwurfPlusZweiStundenFaellig(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-19", "16:00", "")

	due, err := s.DueStaffeln(context.Background(), seasonID, at(t, "2026-09-19", "18:15"))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 {
		t.Fatalf("fällig = %d, erwartet 1", len(due))
	}
	if due[0].Reason != "spieltag" || due[0].Open != 1 {
		t.Errorf("Grund = %q, offen = %d; erwartet spieltag/1", due[0].Reason, due[0].Open)
	}
}

// Das Fenster hängt am FRÜHESTEN Anwurf des Tages, nicht am letzten — sonst
// wartete ein Spieltag mit einem späten Spiel unnötig lange.
func TestPollFenster_FruehesterAnwurfEntscheidet(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-19", "10:00", "")
	seedGame(t, s, staffelID, "900002", "2026-09-19", "20:00", "")

	due, err := s.DueStaffeln(context.Background(), seasonID, at(t, "2026-09-19", "12:30"))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 {
		t.Errorf("fällig = %d, erwartet 1 (10:00 + 2 h ist erreicht)", len(due))
	}
}

func TestPollFenster_AlleSGIDVorhandenBeendetDenTag(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-19", "16:00", "3504061")
	seedGame(t, s, staffelID, "900002", "2026-09-19", "18:00", "3504062")

	due, err := s.DueStaffeln(context.Background(), seasonID, at(t, "2026-09-19", "21:00"))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Errorf("fällig = %d, erwartet 0 — jede Begegnung ist versorgt", len(due))
	}
}

func TestPollFenster_TagOhneBegegnungIstNichtFaellig(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-10-05", "16:00", "")

	due, err := s.DueStaffeln(context.Background(), seasonID, at(t, "2026-09-19", "18:00"))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Errorf("fällig = %d, erwartet 0 — heute ist kein Spiel", len(due))
	}
}

// Ein Bericht, der am Spieltag nicht freigegeben wurde, darf nicht endgültig
// verloren sein.
func TestPollFenster_NachzuegerFruehererTage(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-19", "16:00", "")

	due, err := s.DueStaffeln(context.Background(), seasonID, at(t, "2026-09-20", "08:00"))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].Reason != "nachzug" {
		t.Fatalf("erwartet ein Nachzug-Fenster, bekam %+v", due)
	}
}

func TestPollFenster_NachzugEndetNachDreiTagen(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-10", "16:00", "")

	due, err := s.DueStaffeln(context.Background(), seasonID, at(t, "2026-09-20", "08:00"))
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Errorf("fällig = %d, erwartet 0 — der Termin liegt außerhalb des Nachholfensters", len(due))
	}
}

// Die Kadenz begrenzt die Abrufe: ein gerade gelaufener Poll wiederholt sich
// nicht in derselben Minute.
func TestPollFenster_KadenzVerhindertDauerabruf(t *testing.T) {
	_, s, seasonID, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-19", "16:00", "")
	now := at(t, "2026-09-19", "18:15")
	if _, err := s.db.Exec(`UPDATE bwhv_staffeln SET polled_at = ? WHERE id = ?`,
		now.Add(-5*time.Minute).UTC().Format(time.RFC3339), staffelID); err != nil {
		t.Fatal(err)
	}

	due, err := s.DueStaffeln(context.Background(), seasonID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Errorf("fällig = %d, erwartet 0 — vor 5 min gelaufen, Kadenz ist 15 min", len(due))
	}
}

func TestPollResult_ChangedNurBeiEchterAenderung(t *testing.T) {
	if (PollResult{Staffeln: 3}).Changed() {
		t.Error("ein Lauf ohne Änderung darf nicht broadcasten")
	}
	if !(PollResult{ReportsParsed: 1}).Changed() {
		t.Error("ein neuer Bericht ist eine Änderung")
	}
}
