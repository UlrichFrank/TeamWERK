package auth

import (
	"context"
	"database/sql"
	"strings"
)

// UserCanSeeGame liefert true, wenn der User das Game in irgendeiner Liste oder
// Detail-Antwort sehen darf. Die Regel:
//
//   - Funktionsträger (admin/trainer/sportliche_leitung/vorstand) sehen alles.
//   - Sonst muss der User selbst — oder eines seiner Kinder via family_links —
//     im regulären oder erweiterten Kader eines der Teams in game_teams stehen
//     (Saison-bezogen: kader.season_id = games.season_id).
//
// Konsistent mit der "Meine Teams im Event"-Logik in
// games.Handler.myTeamsInEvent (cross-team-Sichtbarkeit innerhalb eines Events).
func UserCanSeeGame(ctx context.Context, db *sql.DB, userID, gameID int) (bool, error) {
	bypass, err := userHasEventVisibilityBypass(ctx, db, userID)
	if err != nil {
		return false, err
	}
	if bypass {
		return true, nil
	}

	var n int
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM kader k
			WHERE k.season_id = (SELECT season_id FROM games WHERE id = ?)
			  AND k.team_id IN (SELECT team_id FROM game_teams WHERE game_id = ?)
			  AND (
				EXISTS (
					SELECT 1 FROM kader_members km
					JOIN members m ON m.id = km.member_id
					WHERE km.kader_id = k.id
					  AND (m.user_id = ?
					       OR m.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
				)
				OR EXISTS (
					SELECT 1 FROM kader_extended_members kem
					JOIN members m ON m.id = kem.member_id
					WHERE kem.kader_id = k.id
					  AND (m.user_id = ?
					       OR m.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
				)
			  )
		)`,
		gameID, gameID, userID, userID, userID, userID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// GameIDsVisibleToUser liefert die game_ids in der Saison, die der User sehen
// darf, plus ein unrestricted-Flag. Bei unrestricted=true ist der User
// Funktionsträger und sieht alle Events der Saison ohne Filter — Aufrufer
// sollten dann gar nicht erst ein `WHERE id IN (…)` anhängen.
//
// Bei unrestricted=false enthält visibleIDs alle sichtbaren game_ids; ist die
// Liste leer, soll der Caller eine leere Result-Liste zurückgeben, ohne die
// Hauptquery überhaupt zu starten.
func GameIDsVisibleToUser(ctx context.Context, db *sql.DB, userID, seasonID int) (visibleIDs []int, unrestricted bool, err error) {
	bypass, err := userHasEventVisibilityBypass(ctx, db, userID)
	if err != nil {
		return nil, false, err
	}
	if bypass {
		return nil, true, nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT g.id
		FROM games g
		WHERE g.season_id = ?
		  AND EXISTS (
			SELECT 1
			FROM game_teams gt
			JOIN kader k ON k.team_id = gt.team_id AND k.season_id = g.season_id
			WHERE gt.game_id = g.id
			  AND (
				EXISTS (
					SELECT 1 FROM kader_members km
					JOIN members m ON m.id = km.member_id
					WHERE km.kader_id = k.id
					  AND (m.user_id = ?
					       OR m.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
				)
				OR EXISTS (
					SELECT 1 FROM kader_extended_members kem
					JOIN members m ON m.id = kem.member_id
					WHERE kem.kader_id = k.id
					  AND (m.user_id = ?
					       OR m.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
				)
			  )
		  )`,
		seasonID, userID, userID, userID, userID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, false, err
		}
		visibleIDs = append(visibleIDs, id)
	}
	return visibleIDs, false, rows.Err()
}

// userHasEventVisibilityBypass ist true für admin und für jeden User, der über
// sein verknüpftes Member mindestens eine der Funktionen trainer,
// sportliche_leitung oder vorstand hat. Kassierer/Beisitzer bewusst NICHT —
// sie haben keinen operativen Bedarf an Event-Sichtbarkeit fremder Teams.
func userHasEventVisibilityBypass(ctx context.Context, db *sql.DB, userID int) (bool, error) {
	var role sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT role FROM users WHERE id=?`, userID).Scan(&role); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if role.Valid && role.String == "admin" {
		return true, nil
	}

	var hasFunc int
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM member_club_functions mcf
			JOIN members m ON m.id = mcf.member_id
			WHERE m.user_id = ?
			  AND mcf.function IN ('trainer','sportliche_leitung','vorstand')
		)`, userID).Scan(&hasFunc)
	if err != nil {
		return false, err
	}
	return hasFunc == 1, nil
}

// PlaceholdersFor liefert "?,?,?,...?" mit n Fragezeichen, zur Verwendung in
// `IN (…)`-Klauseln, wenn ein Caller `GameIDsVisibleToUser` als WHERE-Filter
// einsetzt.
func PlaceholdersFor(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// GameVisibilityClause liefert einen SQL-WHERE-Schnipsel (zusammen mit den
// Bind-Args), der als zusätzliche Filterbedingung an eine games-Query
// angehängt werden kann. Die Query muss `games` als `g` aliasieren.
//
// Für Funktionsträger ist unrestricted=true und der Caller soll keinen Filter
// anhängen. Für alle anderen wird ein EXISTS-Subquery zurückgegeben, der
// prüft, ob das Game ein Team enthält, in dessen Kader der User oder eines
// seiner Kinder steht (regulär oder erweitert).
//
// Achtung: erfordert eine separate Bypass-Prüfung gegen die DB; daher
// Wrapper-Funktion, die das übernimmt:
func GameVisibilityClause(ctx context.Context, db *sql.DB, userID int) (clause string, args []any, unrestricted bool, err error) {
	bypass, err := userHasEventVisibilityBypass(ctx, db, userID)
	if err != nil {
		return "", nil, false, err
	}
	if bypass {
		return "1=1", nil, true, nil
	}
	clause = `EXISTS (
		SELECT 1 FROM game_teams gt_vis
		JOIN kader k_vis ON k_vis.team_id = gt_vis.team_id AND k_vis.season_id = g.season_id
		WHERE gt_vis.game_id = g.id AND (
			EXISTS (
				SELECT 1 FROM kader_members km_vis
				JOIN members m_vis ON m_vis.id = km_vis.member_id
				WHERE km_vis.kader_id = k_vis.id
				  AND (m_vis.user_id = ?
				       OR m_vis.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
			)
			OR EXISTS (
				SELECT 1 FROM kader_extended_members kem_vis
				JOIN members m_vis2 ON m_vis2.id = kem_vis.member_id
				WHERE kem_vis.kader_id = k_vis.id
				  AND (m_vis2.user_id = ?
				       OR m_vis2.id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
			)
		)
	)`
	return clause, []any{userID, userID, userID, userID}, false, nil
}
