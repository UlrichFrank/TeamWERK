package teams

import (
	"context"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Gate-Helper für die Aufgaben-/Strafen-Routen. Bewusst inline im teams-Package
// (wie GetRosters eigener Zugriffscheck), nicht in internal/policy — die Checks
// brauchen DB-Lookups, und policy führt keine *sql.DB. Alle Checks sind auf die
// aktive Saison (seasons.is_active=1) des Teams gescoped. Admins passen immer.

// isTrainerOfTeam meldet, ob der Nutzer Trainer eines Kaders dieses Teams in der
// aktiven Saison ist. Gate für Catalog-Pflege, Aufgaben-Zuweisung und
// Strafenwart-Ernennung.
func (h *Handler) isTrainerOfTeam(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims.Role == "admin" {
		return true, nil
	}
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM kader_trainers kt
		JOIN kader k   ON k.id = kt.kader_id
		JOIN members m ON m.id = kt.member_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id = ?`,
		teamID, claims.UserID).Scan(&n)
	return n > 0, err
}

// isStrafenwartOfTeam meldet, ob der Nutzer als Strafenwart eines Kaders dieses
// Teams in der aktiven Saison eingetragen ist. Gate fürs Vergeben/Stornieren/
// Zurücksetzen von Strafen. Verhindert strukturell die Bestrafung eines fremden
// Teams: kein Appointment-Row → false.
func (h *Handler) isStrafenwartOfTeam(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims.Role == "admin" {
		return true, nil
	}
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM kader_strafenwarte ks
		JOIN kader k   ON k.id = ks.kader_id
		JOIN members m ON m.id = ks.member_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id = ?`,
		teamID, claims.UserID).Scan(&n)
	return n > 0, err
}

// isKassenwartOfTeam meldet, ob der Nutzer als Kassenwart eines Kaders dieses
// Teams in der aktiven Saison eingetragen ist. Sibling von isStrafenwartOfTeam.
// Teil des Write-Gates fürs Kassenbuch (Trainer ODER Kassenwart). Kein
// Appointment-Row → false, was Fremd-Team-Buchungen strukturell ausschließt.
func (h *Handler) isKassenwartOfTeam(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims.Role == "admin" {
		return true, nil
	}
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM kader_kassenwarte kk
		JOIN kader k   ON k.id = kk.kader_id
		JOIN members m ON m.id = kk.member_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id = ?`,
		teamID, claims.UserID).Scan(&n)
	return n > 0, err
}

// canManageCashbook meldet, ob der Nutzer Kassenbuchungen anlegen/löschen darf:
// Trainer ODER Kassenwart dieses Kaders (admin passt in beiden Sub-Checks). Der
// Trainer soll die Kasse auch ohne ernannten Kassenwart pflegen können.
func (h *Handler) canManageCashbook(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if ok, err := h.isTrainerOfTeam(ctx, claims, teamID); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return h.isKassenwartOfTeam(ctx, claims, teamID)
}

// canReadCashbook meldet, ob der Nutzer das Kassenbuch lesen darf. Bewusst
// identisch zum Read-Gate für Strafen (Spieler ∨ Trainer ∨ Erw. Kader; keine
// Eltern, keine Außenstehenden) — benannter Alias, damit die Sichtbarkeit von
// Kasse und Strafen nie auseinanderläuft.
func (h *Handler) canReadCashbook(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	return h.canReadPenalties(ctx, claims, teamID)
}

// canReadPenalties meldet, ob der Nutzer die Strafenliste des Teams lesen darf:
// Spieler (kader_members), Trainer (kader_trainers) ODER Erweiterter Kader
// (kader_extended_members) des Kaders der aktiven Saison. Eltern (family_links)
// und Außenstehende bekommen false — das ist die harte Sichtbarkeitsgrenze.
func (h *Handler) canReadPenalties(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims.Role == "admin" {
		return true, nil
	}
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT m.user_id FROM kader_members km
			JOIN kader k   ON k.id = km.kader_id
			JOIN members m ON m.id = km.member_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id = ?
			UNION
			SELECT m.user_id FROM kader_trainers kt
			JOIN kader k   ON k.id = kt.kader_id
			JOIN members m ON m.id = kt.member_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id = ?
			UNION
			SELECT m.user_id FROM kader_extended_members kem
			JOIN kader k   ON k.id = kem.kader_id
			JOIN members m ON m.id = kem.member_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id = ?
		)`,
		teamID, claims.UserID, teamID, claims.UserID, teamID, claims.UserID).Scan(&n)
	return n > 0, err
}

// resolveKaderID liefert die kader_id des Teams in der aktiven Saison. Bei mehreren
// Kadern pro Team/Saison (team_number) wird der kleinste genommen — Aufgaben/Strafen
// hängen am Team, nicht an einer einzelnen Kader-Nummer. Liefert (0, nil) wenn kein
// Kader existiert. Schreib-Handler nutzen das, um die kader_id für INSERTs zu finden.
func (h *Handler) resolveKaderID(ctx context.Context, teamID int) (int, error) {
	var kaderID int
	err := h.db.QueryRowContext(ctx, `
		SELECT k.id FROM kader k
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ORDER BY k.id
		LIMIT 1`, teamID).Scan(&kaderID)
	if err != nil {
		return 0, err
	}
	return kaderID, nil
}
