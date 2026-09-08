package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Übungsgruppen im Chat.
//
// Eine Übungsgruppe ist ein Kader ohne `teams`-Zeile (kader.kind='practice').
// Die View `user_accessible_teams` — die Sichtbarkeitsquelle der
// Mannschaftsgruppen — filtert `k.team_id IS NOT NULL` und kennt sie deshalb
// nicht. Sichtbarkeit und Auflösung laufen hier stattdessen direkt über
// kader_members / kader_trainers / family_links, adressiert mit der kader.id.

// practiceGroupMemberQuery liefert DISTINCT (user_id, name) für ein kind einer
// Übungsgruppe. Der einzige Platzhalter ist die kader.id.
//
// Das kind `spieler` hat — anders als bei einer Mannschaft — KEINE Union mit
// `kader_extended_members`: Übungsgruppen haben keinen erweiterten Kader
// (PUT /api/kader/{id} lehnt das mit 409 ab).
func practiceGroupMemberQuery(kind string) string {
	switch kind {
	case "trainer":
		return `
			SELECT DISTINCT m.user_id AS user_id,
			       u.first_name || ' ' || u.last_name AS name
			FROM kader_trainers kt
			JOIN members m ON m.id = kt.member_id
			JOIN users u ON u.id = m.user_id
			WHERE kt.kader_id = ? AND m.user_id IS NOT NULL`
	case "spieler":
		return `
			SELECT DISTINCT m.user_id AS user_id,
			       u.first_name || ' ' || u.last_name AS name
			FROM kader_members km
			JOIN members m ON m.id = km.member_id
			JOIN users u ON u.id = m.user_id
			WHERE km.kader_id = ? AND m.user_id IS NOT NULL`
	case "eltern":
		return `
			SELECT DISTINCT fl.parent_user_id AS user_id,
			       u.first_name || ' ' || u.last_name AS name
			FROM family_links fl
			JOIN kader_members km ON km.member_id = fl.member_id
			JOIN users u ON u.id = fl.parent_user_id
			WHERE km.kader_id = ?`
	}
	return ""
}

// isPracticeGroup meldet, ob kaderID eine Übungsgruppe der AKTIVEN Saison ist.
// Ein Mannschaftskader (kind='team') ist über die Übungsgruppen-Route nicht
// adressierbar — der Aufrufer antwortet dann 404.
func (h *Handler) isPracticeGroup(ctx context.Context, kaderID int) (bool, error) {
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.id = ? AND k.kind = 'practice' AND s.is_active = 1
		)`, kaderID).Scan(&exists)
	return exists, err
}

// canSeePracticeGroup: admin/vorstand/sportliche_leitung sehen jede Gruppe,
// sonst nur Mitglieder, Trainer und Eltern von Mitgliedern.
func (h *Handler) canSeePracticeGroup(ctx context.Context, claims *auth.Claims, kaderID int) (bool, error) {
	if hasClubWideChatReach(claims) {
		return true, nil
	}
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM kader_members km
			JOIN members m ON m.id = km.member_id
			WHERE km.kader_id = ? AND m.user_id = ?
			UNION ALL
			SELECT 1 FROM kader_trainers kt
			JOIN members m ON m.id = kt.member_id
			WHERE kt.kader_id = ? AND m.user_id = ?
			UNION ALL
			SELECT 1 FROM family_links fl
			JOIN kader_members km ON km.member_id = fl.member_id
			WHERE km.kader_id = ? AND fl.parent_user_id = ?
		)`, kaderID, claims.UserID, kaderID, claims.UserID, kaderID, claims.UserID).Scan(&exists)
	return exists, err
}

// sharesPracticeGroup meldet, ob zwei Nutzer über dieselbe Übungsgruppe der
// aktiven Saison verbunden sind (als Mitglied, Trainer oder Elternteil).
//
// Nötig, weil canContactUser sonst über `user_accessible_teams` entscheidet:
// zwei Spieler verschiedener Mannschaften, die zusammen im Torwarttraining
// stehen, könnten die aufgelöste Standardgruppe sonst zwar sehen, aber nicht
// anschreiben — die Kachel wäre da und der Gruppenaufbau schlüge fehl.
func (h *Handler) sharesPracticeGroup(ctx context.Context, userA, userB int) (bool, error) {
	const reach = `
		SELECT km.kader_id AS kader_id, m.user_id AS user_id
		FROM kader_members km JOIN members m ON m.id = km.member_id
		WHERE m.user_id IS NOT NULL
		UNION
		SELECT kt.kader_id, m.user_id
		FROM kader_trainers kt JOIN members m ON m.id = kt.member_id
		WHERE m.user_id IS NOT NULL
		UNION
		SELECT km.kader_id, fl.parent_user_id
		FROM family_links fl JOIN kader_members km ON km.member_id = fl.member_id`
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM (`+reach+`) a
			JOIN (`+reach+`) b ON b.kader_id = a.kader_id
			JOIN kader k ON k.id = a.kader_id
			JOIN seasons s ON s.id = k.season_id
			WHERE k.kind = 'practice' AND s.is_active = 1
			  AND a.user_id = ? AND b.user_id = ?
		)`, userA, userB).Scan(&exists)
	return exists, err
}

// listPracticeGroups liefert die sichtbaren Übungsgruppen-Kacheln der aktiven
// Saison (kind × Gruppe, leere Kinds weggelassen).
func (h *Handler) listPracticeGroups(r *http.Request, claims *auth.Claims) []TeamGroup {
	var rows *sql.Rows
	var err error
	if hasClubWideChatReach(claims) {
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT k.id, COALESCE(k.name, '')
			FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.kind = 'practice' AND s.is_active = 1
			ORDER BY k.name`)
	} else {
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT k.id, COALESCE(k.name, '')
			FROM kader k
			JOIN seasons s ON s.id = k.season_id
			WHERE k.kind = 'practice' AND s.is_active = 1
			  AND EXISTS(
				SELECT 1 FROM kader_members km
				JOIN members m ON m.id = km.member_id
				WHERE km.kader_id = k.id AND m.user_id = ?
				UNION ALL
				SELECT 1 FROM kader_trainers kt
				JOIN members m ON m.id = kt.member_id
				WHERE kt.kader_id = k.id AND m.user_id = ?
				UNION ALL
				SELECT 1 FROM family_links fl
				JOIN kader_members km ON km.member_id = fl.member_id
				WHERE km.kader_id = k.id AND fl.parent_user_id = ?
			  )
			ORDER BY k.name`, claims.UserID, claims.UserID, claims.UserID)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	type groupInfo struct {
		id   int
		name string
	}
	var groups []groupInfo
	for rows.Next() {
		var g groupInfo
		if err := rows.Scan(&g.id, &g.name); err != nil {
			continue
		}
		groups = append(groups, g)
	}

	out := []TeamGroup{}
	for _, g := range groups {
		for _, kind := range []string{"trainer", "spieler", "eltern"} {
			var count int
			err := h.db.QueryRowContext(r.Context(),
				`SELECT COUNT(*) FROM (`+practiceGroupMemberQuery(kind)+`) WHERE user_id != ?`,
				g.id, claims.UserID).Scan(&count)
			if err != nil || count == 0 {
				continue
			}
			out = append(out, TeamGroup{
				GroupType:    "practice",
				TeamID:       g.id,
				DisplayShort: g.name,
				Kind:         kind,
				Count:        count,
			})
		}
	}
	return out
}

// ResolvePracticeGroup — GET /api/chat/practice-groups/{id}/{kind}/members
func (h *Handler) ResolvePracticeGroup(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	kaderID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	kind := chi.URLParam(r, "kind")
	q := practiceGroupMemberQuery(kind)
	if q == "" {
		http.Error(w, "invalid kind", http.StatusBadRequest)
		return
	}

	isPractice, err := h.isPracticeGroup(r.Context(), kaderID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !isPractice {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ok, err := h.canSeePracticeGroup(r.Context(), claims, kaderID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT user_id, name FROM (`+q+`) WHERE user_id != ? ORDER BY name`,
		kaderID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := []TeamGroupMember{}
	for rows.Next() {
		var m TeamGroupMember
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			continue
		}
		members = append(members, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}
