package notify

import (
	"database/sql"
	"log/slog"
	"strings"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
)

// RSVPChangeWindow ist das Fenster vor Terminbeginn, in dem eine Umentscheidung
// eines Spielers die Trainer benachrichtigt. Außerhalb davon ist genug Zeit, die
// Teilnehmerliste von sich aus anzusehen.
const RSVPChangeWindow = 7 * 24 * time.Hour

// rsvpStatusLabel übersetzt einen RSVP-Status in das Wort, das auch die
// Oberfläche zeigt.
func rsvpStatusLabel(status string) string {
	switch status {
	case "confirmed":
		return "Zusage"
	case "declined":
		return "Absage"
	case "maybe":
		return "Vielleicht"
	}
	return status
}

// RSVPChange beschreibt eine abgeschlossene RSVP-Änderung, so wie sie die beiden
// RSVP-Handler (Spiele, Trainings) nach dem Upsert kennen. Die Domäne löst die
// betroffenen Kader selbst auf — bei Spielen über die Mannschaften in der Saison
// des Spiels, bei Trainings direkt über `training_sessions.kader_id`.
type RSVPChange struct {
	KaderIDs    []int
	MemberID    int       // Ziel-Mitglied, dessen Antwort sich geändert hat
	ActorUserID int       // wer den Request abgesendet hat (Spieler, Elternteil, Staff)
	PrevStatus  string    // "" = es gab vorher keine Antwort
	NewStatus   string    //
	Reason      string    //
	Start       time.Time // Terminbeginn
	Now         time.Time //
	Subject     string    // "Heimspiel vs. TV Beispiel", "Training"
	When        string    // EventWhen(date, clock)
	URL         string    // Sprungziel, z. B. /termine?focus=game-12
}

// RSVPChangeRecipients entscheidet, ob eine RSVP-Änderung die Trainer
// benachrichtigt, und liefert die Empfänger. nil heißt „keine Meldung".
//
// Gemeldet wird nur eine echte Umentscheidung: es gab vorher eine Antwort, der
// Status wechselt, der Termin liegt in der Zukunft und beginnt in höchstens
// RSVPChangeWindow, und das Ziel-Mitglied ist nicht selbst Trainer eines der
// Kader (dessen eigene Antwort ist keine Spieler-Umentscheidung). Der Auslöser
// wird herausgefiltert — anders als bei TeamAudience ist seine Handlung hier der
// Inhalt der Meldung an Dritte, nicht eine Bestätigung an ihn selbst.
func RSVPChangeRecipients(db *sql.DB, c RSVPChange) []int {
	if c.PrevStatus == "" || c.PrevStatus == c.NewStatus {
		return nil
	}
	until := c.Start.Sub(c.Now)
	if until <= 0 || until > RSVPChangeWindow {
		return nil
	}
	if len(c.KaderIDs) == 0 {
		return nil
	}
	if isKaderTrainer(db, c.MemberID, c.KaderIDs) {
		return nil
	}
	var out []int
	for _, uid := range KaderTrainers(db, c.KaderIDs...) {
		if uid != c.ActorUserID {
			out = append(out, uid)
		}
	}
	return out
}

// SendRSVPChange benachrichtigt die Trainer über eine Umentscheidung, sofern
// RSVPChangeRecipients jemanden liefert. Die Zustellung läuft über SendAsync —
// eine fehlschlagende Meldung darf die bereits gespeicherte RSVP nie berühren.
// Kategorie ist `operativ` (Profil-Schalter „Vereinsaufgaben"): die Meldung gilt
// dem Trainer in seiner Funktion, nicht als Mitglied der Mannschaft.
func SendRSVPChange(db *sql.DB, cfg *appconfig.Config, c RSVPChange) {
	recipients := RSVPChangeRecipients(db, c)
	if len(recipients) == 0 {
		return
	}
	name := memberName(db, c.MemberID)
	title := "Umentscheidung"
	if name != "" {
		title += ": " + name
	}
	body := RSVPChangeBody(name, c.Subject, c.When, c.PrevStatus, c.NewStatus, c.Reason)
	SendAsync(db, cfg, recipients, "operativ", title, body, c.URL)
}

// RSVPChangeBody baut den Text einer Umentscheidungs-Meldung:
//
//	RSVPChangeBody("Max Muster", "Heimspiel vs. TV Beispiel", "am 27.09.2026 um 15:00 Uhr", "confirmed", "declined", "krank")
//	  → "Max Muster hat für Heimspiel vs. TV Beispiel am 27.09.2026 um 15:00 Uhr von Zusage auf Absage umgestellt. Grund: krank"
func RSVPChangeBody(member, subject, when, from, to, reason string) string {
	if member = strings.TrimSpace(member); member == "" {
		member = "Ein Mitglied"
	}
	if subject = strings.TrimSpace(subject); subject == "" {
		subject = fallbackSubject
	}
	var b strings.Builder
	b.WriteString(member)
	b.WriteString(" hat für ")
	b.WriteString(subject)
	if when = strings.TrimSpace(when); when != "" {
		b.WriteString(" ")
		b.WriteString(when)
	}
	b.WriteString(" von ")
	b.WriteString(rsvpStatusLabel(from))
	b.WriteString(" auf ")
	b.WriteString(rsvpStatusLabel(to))
	b.WriteString(" umgestellt.")
	if reason = TrimReason(reason); reason != "" {
		b.WriteString(" Grund: ")
		b.WriteString(reason)
	}
	return b.String()
}

// KaderTrainers liefert die Nutzerkonten der Trainer (`kader_trainers`) der
// übergebenen Kader, dedupliziert. Trainer ohne Nutzerkonto fehlen. Bewusst
// ohne Saisonfilter: der Aufrufer übergibt bereits die Kader der richtigen
// Saison. Wie TeamAudience ohne `error` — ein Query-Fehler wird protokolliert.
func KaderTrainers(db *sql.DB, kaderIDs ...int) []int {
	if len(kaderIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(kaderIDs)), ",")
	args := make([]any, len(kaderIDs))
	for i, id := range kaderIDs {
		args[i] = id
	}
	rows, err := db.Query(`
		SELECT DISTINCT m.user_id
		FROM kader_trainers kt
		JOIN members m ON m.id = kt.member_id
		WHERE kt.kader_id IN (`+placeholders+`) AND m.user_id IS NOT NULL`, args...)
	if err != nil {
		slog.Error("notify.KaderTrainers query", "kader_ids", kaderIDs, "error", err)
		return nil
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			slog.Error("notify.KaderTrainers scan", "error", err)
			return nil
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		slog.Error("notify.KaderTrainers rows", "error", err)
		return nil
	}
	return out
}

// isKaderTrainer meldet, ob das Mitglied Trainer eines der Kader ist. Im
// Fehlerfall false: dann geht die Meldung eher einmal zu viel raus als zu wenig.
func isKaderTrainer(db *sql.DB, memberID int, kaderIDs []int) bool {
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(kaderIDs)), ",")
	args := make([]any, 0, len(kaderIDs)+1)
	args = append(args, memberID)
	for _, id := range kaderIDs {
		args = append(args, id)
	}
	var n int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM kader_trainers
		WHERE member_id = ? AND kader_id IN (`+placeholders+`)`, args...).Scan(&n); err != nil {
		slog.Error("notify.isKaderTrainer", "member_id", memberID, "error", err)
		return false
	}
	return n > 0
}

// memberName liefert „Vorname Nachname" eines Mitglieds oder "".
func memberName(db *sql.DB, memberID int) string {
	var first, last string
	if err := db.QueryRow(
		`SELECT COALESCE(first_name, ''), COALESCE(last_name, '') FROM members WHERE id = ?`,
		memberID).Scan(&first, &last); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
}
