package chat

import (
	"context"
	"database/sql"
	"fmt"
)

// Ziele einer Mitteilung. Zwei Familien: vereinsweite Zielgruppen (kein Team)
// und die Standardgruppen eines Teams, die dasselbe Vokabular tragen wie der
// Gruppen-Picker des Chats (team_groups.go).
const (
	TargetUsers   = "users"   // alle Zeilen in users
	TargetMembers = "members" // alle User mit Mitglieds-Datensatz
	TargetSpieler = "spieler" // alle User, deren Mitglied die Vereinsfunktion 'spieler' trägt
	TargetEltern  = "eltern"  // alle Eltern laut family_links

	TargetTeamSpieler = "team_spieler" // Spieler eines Teams (regulärer + erweiterter Kader)
	TargetTeamEltern  = "team_eltern"  // Eltern der Spieler eines Teams
	TargetTeamTrainer = "team_trainer" // Kader-Trainer eines Teams
	TargetAlleTrainer = "alle_trainer" // Kader-Trainer aller Teams der aktiven Saison
)

// Target ist ein einzelnes Ziel. TeamID ist bei den team_*-Kinds Pflicht und
// bei allen anderen verboten — die Tabelle broadcast_targets erzwingt dieselbe
// Bindung per CHECK.
type Target struct {
	Kind   string `json:"kind"`
	TeamID *int   `json:"teamId"`
}

// audienceQueries hält je vereinsweiter Zielgruppe die Query, die ihre
// User-IDs liefert.
//
// Die Map ist zugleich die Whitelist: was hier (oder unter teamTargetKinds)
// nicht steht, ist nicht setzbar. Das gilt insbesondere für 'legacy' — den
// Wert, auf den Migration 049 die Bestandszeilen der alten Zielgruppen
// ('team', 'role') abgebildet hat. Er ist persistierbar und lesbar, aber nie
// schreibbar.
//
// Alle vier liefern distinkte IDs: bei 'spieler' erzeugte ein Join auf
// member_club_functions sonst eine Zeile je Vereinsfunktion, bei 'eltern' eine
// je verknüpftem Kind. Der Join über members.user_id schließt Mitglieder ohne
// Zugang strukturell aus — sie sind über eine Mitteilung nicht erreichbar.
//
// Bewusst kein Rückgriff auf internal/policy: die Ordner-ACL beantwortet ein
// Prädikat für ein bekanntes Subjekt ("darf DIESER Nutzer?"), hier wird die
// Umkehrung gebraucht ("wer sind alle?"). Insbesondere erben Eltern hier NICHT
// die Vereinsfunktionen ihrer Kinder — sonst wäre 'eltern' eine Teilmenge von
// 'spieler' und die Auswahl irreführend (mitteilung-zielgruppen/design.md §3).
var audienceQueries = map[string]string{
	TargetUsers: `SELECT id FROM users`,

	TargetMembers: `SELECT DISTINCT u.id
	                  FROM users u
	                  JOIN members m ON m.user_id = u.id`,

	TargetSpieler: `SELECT DISTINCT u.id
	                  FROM users u
	                  JOIN members m ON m.user_id = u.id
	                  JOIN member_club_functions mcf
	                    ON mcf.member_id = m.id AND mcf.function = 'spieler'`,

	TargetEltern: `SELECT DISTINCT parent_user_id FROM family_links`,
}

// teamTargetKinds bildet die teambezogenen Ziele auf die Kind-Bezeichner der
// Chat-Standardgruppen ab. Die Auflösung läuft dadurch durch dieselben Queries
// wie der Gruppen-Picker (teamGroupMemberQuery) — es gibt bewusst keine zweite
// Definition von "Spieler eines Teams", die daneben driften könnte.
var teamTargetKinds = map[string]string{
	TargetTeamSpieler: "spieler",
	TargetTeamEltern:  "eltern",
	TargetTeamTrainer: "trainer",
}

// IsClubWideTarget meldet, ob das Kind eine vereinsweite Zielgruppe ist. Diese
// bleiben admin/vorstand/sportliche_leitung vorbehalten — anders als
// alle_trainer, das zwar auch teamübergreifend ist, aber das Kollegium meint
// und nicht ein Publikum (design.md §5).
func IsClubWideTarget(kind string) bool {
	_, ok := audienceQueries[kind]
	return ok
}

// IsTeamTarget meldet, ob das Kind eine Team-Standardgruppe adressiert und
// damit eine TeamID braucht.
func IsTeamTarget(kind string) bool {
	_, ok := teamTargetKinds[kind]
	return ok
}

// ValidTarget meldet, ob das Ziel über die API setzbar ist: bekanntes Kind und
// eine TeamID genau dann, wenn das Kind teambezogen ist. 'legacy' und alle
// Altwerte ('all', 'team', 'role') sind nicht setzbar.
func ValidTarget(t Target) bool {
	if IsTeamTarget(t.Kind) {
		return t.TeamID != nil
	}
	if IsClubWideTarget(t.Kind) || t.Kind == TargetAlleTrainer {
		return t.TeamID == nil
	}
	return false
}

// targetQuery liefert die Query, die die User-IDs eines Ziels ergibt, plus ihre
// Bind-Args.
func targetQuery(t Target) (string, []any, error) {
	if q, ok := audienceQueries[t.Kind]; ok {
		return q, nil, nil
	}
	if t.Kind == TargetAlleTrainer {
		return `SELECT DISTINCT user_id FROM (` + allTrainersMemberQuery() + `)`, nil, nil
	}
	groupKind, ok := teamTargetKinds[t.Kind]
	if !ok {
		return "", nil, fmt.Errorf("unbekanntes Ziel %q", t.Kind)
	}
	if t.TeamID == nil {
		return "", nil, fmt.Errorf("ziel %q ohne teamId", t.Kind)
	}
	q := teamGroupMemberQuery(groupKind)
	if q == "" {
		return "", nil, fmt.Errorf("keine Query für Ziel %q", t.Kind)
	}
	// teamGroupMemberQuery bindet team_id einmal für 'trainer' und zweimal für
	// 'spieler'/'eltern' (dort steht je ein Zweig für regulären und erweiterten
	// Kader in der UNION).
	args := []any{*t.TeamID}
	if groupKind != "trainer" {
		args = append(args, *t.TeamID)
	}
	return `SELECT DISTINCT user_id FROM (` + q + `)`, args, nil
}

// resolveTargets liefert die Vereinigung der User-IDs aller Ziele, dedupliziert
// und in stabiler Reihenfolge (erste Fundstelle gewinnt). Ein Empfänger, den
// mehrere Ziele treffen, erscheint genau einmal — daran hängt, dass er nur eine
// broadcast_reads-Zeile und einen Push bekommt.
func resolveTargets(ctx context.Context, db *sql.DB, targets []Target) ([]int, error) {
	seen := map[int]bool{}
	ids := []int{}
	for _, t := range targets {
		query, args, err := targetQuery(t)
		if err != nil {
			return nil, err
		}
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return ids, nil
}
