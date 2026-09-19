package videos

import (
	"database/sql"
	"net/http"
	"sort"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
)

// eligibleTeam ist eine Mannschaft, für die der Nutzer hochladen darf.
// UploadWithoutGame ist genau dann wahr, wenn die Berechtigung aus dem
// Rollen-Pfad kommt (CanUploadToTeam) — nur dann akzeptiert CreateUpload
// einen Upload ohne game_id („Freier Titel").
type eligibleTeam struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	UploadWithoutGame bool   `json:"upload_without_game"`
}

// eligibleGame ist ein upload-berechtigtes Spiel der aktiven Saison. TeamIDs
// sind die Mannschaften des Spiels, unter denen der Upload erlaubt ist —
// beim Rollen-Pfad der Schnitt mit den Rollen-Teams, beim Dienst-Pfad alle
// Mannschaften des Spiels (CreateUpload verlangt dort team_id ∈ game_teams).
type eligibleGame struct {
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Opponent  string `json:"opponent"`
	EventType string `json:"event_type"`
	SeasonID  int    `json:"season_id"`
	TeamIDs   []int  `json:"team_ids"`
}

// EligibleGames beschreibt, wofür der aufrufende Nutzer aktuell ein Video
// hochladen darf. GET /api/videos/upload-eligible-games (Authenticated-Tier).
//
// Drei Mengen (video-upload-eligible-teams):
//   - game_ids: Spiele der Teams mit Rollen-Berechtigung (CanUploadToTeam —
//     Trainer des Teams, sportliche Leitung, Vorstand, Admin) vereinigt mit
//     Spielen, für die eine Video-Upload-Dienst-Zuweisung existiert
//     (video-download-duty-upload). Saisonübergreifend, unverändert seit dem
//     ersten Vertrag — ältere Tool-Versionen lesen nur dieses Feld.
//   - games: dieselbe Vereinigung, beschränkt auf die aktive Saison, als
//     Datensätze mit Datum/Gegner/Team-IDs. Das Desktop-Tool braucht sie, weil
//     GET /api/games über GameVisibilityClause nur Kader-Zugehörigkeit kennt
//     und ein Dienst-Spiel bei einer fremden Mannschaft dort unsichtbar ist.
//   - teams: Rollen-Teams mit Kader in der aktiven Saison (upload_without_game
//     = true) vereinigt mit den Mannschaften der berechtigten Spiele (false,
//     sofern nicht zugleich Rollen-Team). Bewusst NICHT /api/teams: das ist
//     Kader-Zugehörigkeit (Spieler/Eltern eingeschlossen), hier zählt nur das
//     Upload-Recht — ein Spieler ohne Funktion bekommt drei leere Listen.
//
// Dies ist ein reiner Vorauswahl-Endpoint für Client-UIs (Desktop-Tool). Die
// eigentliche Autorisierung bleibt serverseitig in CreateUpload — diese Listen
// dürfen nicht großzügiger sein als jene Prüfung, sonst zeigt der Client ein
// Spiel an, für das der Upload dann mit 403 scheitert.
func (h *Handler) EligibleGames(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	bypass := claims.Role == "admin" || claims.HasAnyFunction("vorstand", "sportliche_leitung")
	isTrainer := claims.HasFunction("trainer")

	ids := map[int]bool{}

	switch {
	case bypass:
		// Teamübergreifend berechtigt (siehe CanUploadToTeam) → alle Spiele.
		rows, err := h.db.Query(`SELECT id FROM games`)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := collectGameIDs(rows, ids); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case isTrainer:
		rows, err := h.db.Query(`
			SELECT DISTINCT gt.game_id
			FROM game_teams gt
			JOIN trainer_memberships tm ON tm.team_id = gt.team_id
			JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
			JOIN members m ON m.id = tm.member_id
			WHERE m.user_id = ?`, claims.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if err := collectGameIDs(rows, ids); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	// Dienst-Pfad: unabhängig vom Rollen-Pfad, immer zusätzlich geprüft (ein
	// Spieler ohne jede Vereinsfunktion landet sonst nie in obigen Zweigen).
	// Separat gehalten, weil die Team-Auflösung unten für Dienst-Spiele anders
	// ist als für Rollen-Spiele.
	dutyGames := map[int]bool{}
	dutyRows, err := h.db.Query(`
		SELECT DISTINCT ds.game_id
		FROM duty_assignments da
		JOIN duty_slots ds ON ds.id = da.duty_slot_id AND ds.game_id IS NOT NULL
		JOIN duty_types dt ON dt.id = ds.duty_type_id
		WHERE da.user_id = ? AND dt.grants_video_upload = 1`, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := collectGameIDs(dutyRows, dutyGames); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for id := range dutyGames {
		ids[id] = true
	}

	// Rollen-Teams der aktiven Saison — dieselbe Menge, für die CanUploadToTeam
	// true liefert, beschränkt auf Teams mit Kader (wie die frühere
	// /api/teams-Liste des Tools: ein Team ohne Kader hat keine Spiele).
	roleTeams, roleOrder, err := h.roleUploadTeams(claims.UserID, bypass, isTrainer)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	games, gameTeamNames, err := h.eligibleSeasonGames(bypass, roleTeams, dutyGames)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// teams: Rollen-Teams in Saisonreihenfolge, danach die reinen Dienst-Teams
	// (alphabetisch) — Dedup über die ID, Rollen-Recht gewinnt.
	teams := make([]eligibleTeam, 0, len(roleOrder))
	seen := map[int]bool{}
	for _, id := range roleOrder {
		teams = append(teams, eligibleTeam{ID: id, Name: roleTeams[id], UploadWithoutGame: true})
		seen[id] = true
	}
	var dutyOnly []eligibleTeam
	for _, g := range games {
		for _, tid := range g.TeamIDs {
			if !seen[tid] {
				seen[tid] = true
				// bypass: CanUploadToTeam ist für jede Team-ID wahr, auch für ein
				// Team ohne Kader in der aktiven Saison, das nur über ein Spiel
				// hier auftaucht.
				dutyOnly = append(dutyOnly, eligibleTeam{ID: tid, Name: gameTeamNames[tid], UploadWithoutGame: bypass})
			}
		}
	}
	sort.Slice(dutyOnly, func(i, j int) bool {
		if dutyOnly[i].Name != dutyOnly[j].Name {
			return dutyOnly[i].Name < dutyOnly[j].Name
		}
		return dutyOnly[i].ID < dutyOnly[j].ID
	})
	teams = append(teams, dutyOnly...)

	result := make([]int, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Ints(result)
	writeJSON(w, map[string]any{
		"game_ids": result,
		"teams":    teams,
		"games":    games,
	})
}

// roleUploadTeams liefert die Teams mit Kader in der aktiven Saison, für die
// CanUploadToTeam true liefert: alle (bypass) bzw. die Trainer-Teams des
// Nutzers. Reihenfolge wie /api/teams (Altersklasse, Geschlecht, Nummer).
func (h *Handler) roleUploadTeams(userID int, bypass, isTrainer bool) (map[int]string, []int, error) {
	names := map[int]string{}
	var order []int
	if !bypass && !isTrainer {
		return names, order, nil
	}
	const activeSeasonSub = `(SELECT id FROM seasons WHERE is_active = 1 LIMIT 1)`
	q := `SELECT DISTINCT t.id, t.name, ` + appdb.AgeClassSortKey("t.age_class") + ` AS ak, t.gender, k.team_number
	      FROM teams t
	      JOIN kader k ON k.team_id = t.id AND k.season_id = ` + activeSeasonSub
	var args []any
	if !bypass {
		q += `
	      JOIN kader_trainers kt ON kt.kader_id = k.id
	      JOIN members m ON m.id = kt.member_id AND m.user_id = ?`
		args = append(args, userID)
	}
	q += ` ORDER BY ak, t.gender, k.team_number, t.id`
	rows, err := h.db.Query(q, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, teamNumber int
		var name, ak, gender string
		if err := rows.Scan(&id, &name, &ak, &gender, &teamNumber); err != nil {
			return nil, nil, err
		}
		if _, dup := names[id]; dup {
			continue
		}
		names[id] = name
		order = append(order, id)
	}
	return names, order, rows.Err()
}

// eligibleSeasonGames liest alle Spiele der aktiven Saison samt Teams und
// behält die upload-berechtigten: Rollen-Spiele (bypass → alle; sonst Spiele
// mit mindestens einem Rollen-Team, TeamIDs = Schnitt mit den Rollen-Teams)
// und Dienst-Spiele (TeamIDs = alle Teams des Spiels). Ein Spiel, das über
// beide Pfade berechtigt ist, trägt die Vereinigung. Rückgabe zusätzlich
// die Namen aller beteiligten Teams für die teams-Liste.
func (h *Handler) eligibleSeasonGames(bypass bool, roleTeams map[int]string, dutyGames map[int]bool) ([]eligibleGame, map[int]string, error) {
	rows, err := h.db.Query(`
		SELECT g.id, g.date, g.opponent, g.event_type, g.season_id, gt.team_id, t.name
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id
		JOIN teams t ON t.id = gt.team_id
		WHERE g.season_id = (SELECT id FROM seasons WHERE is_active = 1 LIMIT 1)
		ORDER BY g.date, g.time, g.id, gt.team_id`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	games := []eligibleGame{}
	names := map[int]string{}
	idx := map[int]int{} // game id → Position in games
	for rows.Next() {
		var g eligibleGame
		var teamID int
		var teamName string
		var opponent sql.NullString
		if err := rows.Scan(&g.ID, &g.Date, &opponent, &g.EventType, &g.SeasonID, &teamID, &teamName); err != nil {
			return nil, nil, err
		}
		g.Opponent = opponent.String
		_, isRoleTeam := roleTeams[teamID]
		allowed := bypass || isRoleTeam || dutyGames[g.ID]
		if !allowed {
			continue
		}
		names[teamID] = teamName
		i, ok := idx[g.ID]
		if !ok {
			g.TeamIDs = []int{}
			games = append(games, g)
			i = len(games) - 1
			idx[g.ID] = i
		}
		games[i].TeamIDs = append(games[i].TeamIDs, teamID)
	}
	return games, names, rows.Err()
}

// collectGameIDs scannt eine Query mit einer einzelnen int-Spalte in die
// gegebene Menge und schließt die rows.
func collectGameIDs(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}, into map[int]bool) error {
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err
		}
		into[id] = true
	}
	return rows.Err()
}
