package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/db"
)

// AllowedTarget ist ein Ziel, das der Absender verwenden darf, angereichert um
// alles, was der Composer zum Anzeigen braucht.
type AllowedTarget struct {
	Kind   string `json:"kind"`
	TeamID *int   `json:"teamId"`
	Label  string `json:"label"`
	Count  int    `json:"count"`
}

var clubWideLabels = map[string]string{
	TargetUsers:       "Alle Nutzer",
	TargetMembers:     "Alle Mitglieder",
	TargetSpieler:     "Alle Spieler",
	TargetEltern:      "Alle Eltern",
	TargetAlleTrainer: "Alle Trainer",
}

var teamTargetLabels = map[string]string{
	TargetTeamSpieler: "Spieler",
	TargetTeamEltern:  "Eltern",
	TargetTeamTrainer: "Trainer",
}

// trainerTeamIDs liefert die Teams, deren Kader-Trainer der User in der aktiven
// Saison ist. Das ist der Wirkungsbereich fürs Senden — bewusst NICHT
// user_accessible_teams wie bei den Chat-Standardgruppen: die View enthält auch
// Teams, in denen jemand als Spieler, erweiterter Kader oder Elternteil hängt,
// und ein Trainer, der nebenbei Vater in der mD2 ist, soll dort nicht senden
// dürfen (design.md §4).
func trainerTeamIDs(ctx context.Context, database *sql.DB, userID int) ([]int, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT DISTINCT k.team_id
		FROM kader_trainers kt
		JOIN kader k ON k.id = kt.kader_id
		JOIN seasons s ON s.id = k.season_id
		JOIN members m ON m.id = kt.member_id
		WHERE m.user_id = ? AND s.is_active = 1 AND k.team_id IS NOT NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// activeSeasonTeams liefert alle Teams mit Kader in der aktiven Saison, samt
// kanonischer Kurzform. Reihenfolge wie im Gruppen-Picker.
func activeSeasonTeams(ctx context.Context, database *sql.DB) ([]teamRef, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT DISTINCT t.id, COALESCE(`+db.TeamDisplayShort("t")+`, t.name)
		FROM teams t
		JOIN kader k ON k.team_id = t.id
		JOIN seasons s ON s.id = k.season_id
		WHERE s.is_active = 1
		ORDER BY t.age_class, t.gender, k.team_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	teams := []teamRef{}
	for rows.Next() {
		var t teamRef
		if err := rows.Scan(&t.id, &t.short); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

type teamRef struct {
	id    int
	short string
}

// allowedTargets liefert die Ziele, die der Absender verwenden darf — dieselbe
// Menge, gegen die SendBroadcast prüft. Ist sie leer, hat der User kein
// Senderecht.
func allowedTargets(ctx context.Context, database *sql.DB, claims *auth.Claims) ([]AllowedTarget, error) {
	clubWide := hasClubWideChatReach(claims)

	teams, err := activeSeasonTeams(ctx, database)
	if err != nil {
		return nil, err
	}
	if !clubWide {
		allowed, err := trainerTeamIDs(ctx, database, claims.UserID)
		if err != nil {
			return nil, err
		}
		set := map[int]bool{}
		for _, id := range allowed {
			set[id] = true
		}
		kept := teams[:0]
		for _, t := range teams {
			if set[t.id] {
				kept = append(kept, t)
			}
		}
		teams = kept
		// Ohne einen einzigen Trainer-Kader gibt es kein Senderecht — auch
		// alle_trainer nicht, das an "darf senden" hängt und nicht umgekehrt.
		if len(teams) == 0 {
			return []AllowedTarget{}, nil
		}
	}

	targets := []AllowedTarget{}
	if clubWide {
		for _, kind := range []string{TargetUsers, TargetMembers, TargetSpieler, TargetEltern} {
			targets = append(targets, AllowedTarget{Kind: kind, Label: clubWideLabels[kind]})
		}
	}
	// alle_trainer ist die einzige teamübergreifende Gruppe, die auch einem
	// reinen Trainer offensteht: sie ist das Kollegium, kein Publikum.
	targets = append(targets, AllowedTarget{Kind: TargetAlleTrainer, Label: clubWideLabels[TargetAlleTrainer]})

	for _, t := range teams {
		teamID := t.id
		for _, kind := range []string{TargetTeamSpieler, TargetTeamEltern, TargetTeamTrainer} {
			targets = append(targets, AllowedTarget{
				Kind:   kind,
				TeamID: &teamID,
				Label:  teamTargetLabels[kind] + " " + t.short,
			})
		}
	}

	// Zählung zum Schluss, damit sie für jedes Ziel durch denselben Resolver
	// läuft wie der spätere Fan-out. Ziele mit count 0 bleiben in der Liste:
	// eine leere Gruppe ist eine legitime Auswahl, das Ausfiltern wäre genau
	// das Verschweigen, gegen das der recipients-Zähler existiert.
	for i := range targets {
		ids, err := resolveTargets(ctx, database, []Target{{Kind: targets[i].Kind, TeamID: targets[i].TeamID}})
		if err != nil {
			return nil, err
		}
		targets[i].Count = len(ids)
	}
	return targets, nil
}

// maySendTo meldet, ob jedes einzelne Ziel in der Allowlist des Absenders
// steht. Ein einziger Fehltreffer lehnt den ganzen Request ab — eine
// Teilzustellung sähe für den Absender aus wie eine vollständige.
func maySendTo(allowed []AllowedTarget, targets []Target) bool {
	for _, t := range targets {
		found := false
		for _, a := range allowed {
			if a.Kind != t.Kind {
				continue
			}
			if (a.TeamID == nil) != (t.TeamID == nil) {
				continue
			}
			if a.TeamID != nil && *a.TeamID != *t.TeamID {
				continue
			}
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return true
}

// GET /api/chat/broadcast-targets
func (h *Handler) ListBroadcastTargets(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	targets, err := allowedTargets(r.Context(), h.db, claims)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if len(targets) == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targets)
}
