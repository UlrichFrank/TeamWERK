package trainingdiary

import (
	"context"
	"database/sql"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Die Zugriffsregeln spiegeln bewusst attendance.canSeeMemberStats /
// canSeeTeamStats. Kopiert statt importiert, weil der Architektur-Test
// Domain→Domain-Importe verbietet — die Auflösung von Trainern gegen den Kader
// der AKTIVEN Saison ist derselbe Mechanismus, damit ein Kaderwechsel den
// Zugriff sofort und ohne Nachpflege entzieht.
//
// Bewusst NICHT enthalten: `vorstand` und `kassierer`. Das Tagebuch ist
// persönlich; Vereinsverwaltung begründet keinen Lesezugriff.

// resolveOwnMember liefert die member_id des aufrufenden Nutzers. Nutzer ohne
// Mitglieds-Datensatz (z. B. reine Elternkonten) können nichts erfassen.
func (h *Handler) resolveOwnMember(ctx context.Context, claims *auth.Claims) (int, error) {
	if claims == nil {
		return 0, sql.ErrNoRows
	}
	var id int
	err := h.db.QueryRowContext(ctx,
		`SELECT id FROM members WHERE user_id = ?`, claims.UserID).Scan(&id)
	return id, err
}

// canReadMemberDiary prüft den Lesezugriff auf das Tagebuch eines Mitglieds:
// das Mitglied selbst, ein Elternteil via family_links, ein Trainer des Kaders
// in der aktiven Saison (Stamm- oder erweiterter Kader), sportliche_leitung
// oder admin.
func (h *Handler) canReadMemberDiary(ctx context.Context, claims *auth.Claims, memberID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("sportliche_leitung") {
		return true, nil
	}

	var ownUserID sql.NullInt64
	err := h.db.QueryRowContext(ctx,
		`SELECT user_id FROM members WHERE id = ?`, memberID).Scan(&ownUserID)
	if err == sql.ErrNoRows {
		// Mitglied existiert nicht → kein Zugriff, kein Serverfehler. Die
		// Aufrufer antworten damit 403 statt 500 und geben zugleich nicht
		// preis, ob die ID existiert.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if ownUserID.Valid && int(ownUserID.Int64) == claims.UserID {
		return true, nil
	}

	var familyN int
	if err := h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? AND member_id = ?`,
		claims.UserID, memberID).Scan(&familyN); err != nil {
		return false, err
	}
	if familyN > 0 {
		return true, nil
	}

	if !claims.HasFunction("trainer") {
		return false, nil
	}
	var trainerN int
	if err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members tm ON tm.id = trm.member_id AND tm.user_id = ?
		WHERE trm.team_id IN (
			SELECT k.team_id FROM kader k
			WHERE k.season_id = trm.season_id AND (
				EXISTS (SELECT 1 FROM kader_members km           WHERE km.kader_id  = k.id AND km.member_id  = ?)
				OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = ?)
			)
		)`,
		claims.UserID, memberID, memberID).Scan(&trainerN); err != nil {
		return false, err
	}
	return trainerN > 0, nil
}

// canSeeTeamDiary prüft den Zugriff auf die Mannschaftsübersicht: admin,
// sportliche_leitung oder Trainer dieser Mannschaft in der aktiven Saison.
func (h *Handler) canSeeTeamDiary(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("sportliche_leitung") {
		return true, nil
	}
	if !claims.HasFunction("trainer") {
		return false, nil
	}
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trainer_memberships trm
		JOIN seasons s ON s.id = trm.season_id AND s.is_active = 1
		JOIN members m ON m.id = trm.member_id AND m.user_id = ?
		WHERE trm.team_id = ?`,
		claims.UserID, teamID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// activeSeasonID liefert die ID der aktiven Saison oder (0, nil), wenn keine
// gesetzt ist. Der Nullwert ist ein gültiger Zustand: Einträge werden dann mit
// season_id = NULL gespeichert und von der Retention nie angefasst.
func (h *Handler) activeSeasonID(ctx context.Context) (int, error) {
	var id int
	err := h.db.QueryRowContext(ctx, `SELECT id FROM seasons WHERE is_active = 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// resolveSeasonWindow ermittelt Saison-ID und Zeitraum für die Auswertung.
// Ohne Parameter greift die aktive Saison; existiert auch die nicht, liefert
// die Funktion (0, "", "", nil) und die Aufrufer antworten mit einer leeren
// Liste statt einem Fehler.
func (h *Handler) resolveSeasonWindow(ctx context.Context, seasonParam string) (id int, startDate, endDate string, err error) {
	var param any
	if seasonParam != "" {
		param = seasonParam
	}
	row := h.db.QueryRowContext(ctx, `
		SELECT id, start_date, end_date
		FROM seasons
		WHERE id = COALESCE(?, (SELECT id FROM seasons WHERE is_active = 1))`, param)
	if scanErr := row.Scan(&id, &startDate, &endDate); scanErr == sql.ErrNoRows {
		return 0, "", "", nil
	} else if scanErr != nil {
		return 0, "", "", scanErr
	}
	return id, startDate, endDate, nil
}
