package chat

import (
	"context"
	"database/sql"
	"fmt"
)

// Zielgruppen einer Mitteilung. Vereinsweit und bewusst nur vier — Team-Ansagen
// laufen über die Team-Standardgruppen des Chats (team_groups.go), die denselben
// Kreis mit Rückkanal erreichen.
const (
	TargetUsers   = "users"   // alle Zeilen in users
	TargetMembers = "members" // alle User mit Mitglieds-Datensatz
	TargetSpieler = "spieler" // alle User, deren Mitglied die Vereinsfunktion 'spieler' trägt
	TargetEltern  = "eltern"  // alle Eltern laut family_links
)

// audienceQueries hält je Zielgruppe die Query, die ihre User-IDs liefert.
//
// Die Map ist zugleich die Whitelist: was hier nicht steht, ist nicht setzbar.
// Das gilt insbesondere für 'legacy' — den Wert, auf den Migration 049 die
// Bestandszeilen der alten Zielgruppen ('team', 'role') abgebildet hat. Er ist
// persistierbar und lesbar, aber nie schreibbar.
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
// 'spieler' und die Auswahl irreführend (design.md §3).
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

// ValidTarget meldet, ob target über die API als Zielgruppe setzbar ist.
// 'legacy' und alle Altwerte ('all', 'team', 'role') sind es nicht.
func ValidTarget(target string) bool {
	_, ok := audienceQueries[target]
	return ok
}

// resolveAudience liefert die User-IDs der Zielgruppe.
func resolveAudience(ctx context.Context, db *sql.DB, target string) ([]int, error) {
	query, ok := audienceQueries[target]
	if !ok {
		return nil, fmt.Errorf("unbekannte Zielgruppe %q", target)
	}

	rows, err := db.QueryContext(ctx, query)
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
