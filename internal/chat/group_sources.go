package chat

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Standard-Gruppen-Kacheln als Datenwert.
//
// Eine Kachel aus GET /api/chat/team-groups ist das Tripel (groupType, refId,
// kind). Dieselbe Auflösung bedient die Kachel-Liste (count), die
// …/members-Endpoints, die Herkunft einer Gruppe (conversation_sources) und den
// Abgleich „Aktualisieren" — sonst schlüge der erste Abgleich vor, was der
// Anlege-Dialog eben erst anders aufgelöst hat.

// GroupSource ist eine Standard-Gruppen-Kachel. Bei groupType "practice" trägt
// RefID die kader.id, bei "team" die teams.id (0 bei kind "alle_trainer").
type GroupSource struct {
	GroupType string `json:"groupType"`
	RefID     int    `json:"refId"`
	Kind      string `json:"kind"`
}

// queryer ist *sql.DB oder *sql.Tx — der Abgleich rechnet in der Transaktion.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// validSource prüft nur die Form der Kachel, nicht ihre Existenz oder
// Sichtbarkeit.
func validSource(s GroupSource) bool {
	switch s.GroupType {
	case "team":
		if s.Kind == "alle_trainer" {
			return s.RefID == 0
		}
		return s.RefID > 0 && (s.Kind == "trainer" || s.Kind == "spieler" || s.Kind == "eltern")
	case "practice":
		return s.RefID > 0 && (s.Kind == "trainer" || s.Kind == "spieler" || s.Kind == "eltern")
	}
	return false
}

// sourceQuery liefert das SQL-Fragment (DISTINCT user_id, name) der Kachel samt
// seiner Argumente. Die Platzhalter-Anzahl hängt am kind — die Fallunterscheidung
// steht nur hier.
func sourceQuery(s GroupSource) (string, []any) {
	switch {
	case s.GroupType == "team" && s.Kind == "alle_trainer":
		return allTrainersMemberQuery(), nil
	case s.GroupType == "team" && s.Kind == "trainer":
		return teamGroupMemberQuery("trainer"), []any{s.RefID}
	case s.GroupType == "team":
		return teamGroupMemberQuery(s.Kind), []any{s.RefID, s.RefID}
	default:
		return practiceGroupMemberQuery(s.Kind), []any{s.RefID}
	}
}

// resolveSource löst eine (formal gültige) Kachel in ihre Mitglieder auf,
// excludeUserID ausgenommen, nach Namen sortiert.
func resolveSource(ctx context.Context, q queryer, s GroupSource, excludeUserID int) ([]TeamGroupMember, error) {
	frag, args := sourceQuery(s)
	rows, err := q.QueryContext(ctx,
		`SELECT user_id, name FROM (`+frag+`) WHERE user_id != ? ORDER BY name`,
		append(args, excludeUserID)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []TeamGroupMember{}
	for rows.Next() {
		var m TeamGroupMember
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

// countSource zählt dieselbe Menge wie resolveSource.
func countSource(ctx context.Context, q queryer, s GroupSource, excludeUserID int) (int, error) {
	frag, args := sourceQuery(s)
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM (`+frag+`) WHERE user_id != ?`,
		append(args, excludeUserID)...).Scan(&n)
	return n, err
}

// canSeeSource delegiert an die bestehenden Sichtbarkeitsregeln der Kachel-Arten.
// Eine Übungsgruppe, die (in der aktiven Saison) nicht existiert, ist nicht
// sichtbar.
func (h *Handler) canSeeSource(r *http.Request, claims *auth.Claims, s GroupSource) (bool, error) {
	switch {
	case s.GroupType == "team" && s.Kind == "alle_trainer":
		return h.callerInTrainerCircle(r.Context(), claims)
	case s.GroupType == "team":
		return h.canSeeTeamGroup(r, claims, s.RefID)
	default:
		ok, err := h.isPracticeGroup(r.Context(), s.RefID)
		if err != nil || !ok {
			return false, err
		}
		return h.canSeePracticeGroup(r.Context(), claims, s.RefID)
	}
}

// dedupSources fasst doppelte Tripel zusammen, Reihenfolge bleibt erhalten.
func dedupSources(in []GroupSource) []GroupSource {
	seen := make(map[GroupSource]bool, len(in))
	out := make([]GroupSource, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// sourceExists meldet, ob die Kachel in der aktiven Saison existiert:
// Mannschaft mit Kader der aktiven Saison bzw. Übungsgruppe der aktiven Saison.
// „Alle Trainer" existiert immer.
func sourceExists(ctx context.Context, q queryer, s GroupSource) (bool, error) {
	var exists bool
	var err error
	switch {
	case s.GroupType == "team" && s.Kind == "alle_trainer":
		return true, nil
	case s.GroupType == "team":
		err = q.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM kader k JOIN seasons se ON se.id = k.season_id
				WHERE k.team_id = ? AND se.is_active = 1
			)`, s.RefID).Scan(&exists)
	default:
		err = q.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM kader k JOIN seasons se ON se.id = k.season_id
				WHERE k.id = ? AND k.kind = 'practice' AND se.is_active = 1
			)`, s.RefID).Scan(&exists)
	}
	return exists, err
}

// checkSources prüft Kacheln für das Anlegen einer Gruppe: Form und Existenz
// (400), dann Sichtbarkeit für den Caller (403). Rückgabe 0 = in Ordnung.
func (h *Handler) checkSources(r *http.Request, claims *auth.Claims, sources []GroupSource) (int, error) {
	for _, s := range sources {
		if !validSource(s) {
			return http.StatusBadRequest, nil
		}
		ok, err := sourceExists(r.Context(), h.db, s)
		if err != nil {
			return 0, err
		}
		if !ok {
			return http.StatusBadRequest, nil
		}
	}
	for _, s := range sources {
		ok, err := h.canSeeSource(r, claims, s)
		if err != nil {
			return 0, err
		}
		if !ok {
			return http.StatusForbidden, nil
		}
	}
	return 0, nil
}

// loadSources liest die gespeicherte Herkunft einer Gruppe.
func loadSources(ctx context.Context, q queryer, convID int) ([]GroupSource, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT group_type, ref_id, kind FROM conversation_sources
		WHERE conversation_id = ?
		ORDER BY group_type, ref_id, kind`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GroupSource{}
	for rows.Next() {
		var s GroupSource
		if err := rows.Scan(&s.GroupType, &s.RefID, &s.Kind); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// writeSources ersetzt die Herkunft einer Gruppe vollständig.
func writeSources(ctx context.Context, tx *sql.Tx, convID int, sources []GroupSource) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_sources WHERE conversation_id = ?`, convID); err != nil {
		return err
	}
	for _, s := range sources {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO conversation_sources (conversation_id, group_type, ref_id, kind)
			VALUES (?, ?, ?, ?)`, convID, s.GroupType, s.RefID, s.Kind); err != nil {
			return err
		}
	}
	return nil
}
