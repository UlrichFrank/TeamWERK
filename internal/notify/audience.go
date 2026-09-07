package notify

import (
	"database/sql"
	"log/slog"
	"strings"
)

// teamAudienceQuery löst die Empfängermenge einer Terminmeldung auf: Stammkader
// und erweiterter Kader der betroffenen Mannschaften jeweils samt Elternteilen,
// dazu die Trainer des Kaders.
//
// Fünf Zweige, weil jede Gruppe an einer eigenen Tabelle hängt (`kader_members`,
// `kader_extended_members`, `kader_trainers`) und die Eltern über `family_links`
// an der Mitglieds-ID hängen, nicht am Nutzer. Die Deduplizierung macht das
// äußere DISTINCT — ein Mitglied darf in mehreren Listen und in mehreren
// betroffenen Mannschaften stehen, ohne die Meldung doppelt auszulösen.
//
// Trainer bekommen KEINEN Eltern-Zweig: sie stehen hier in ihrer Funktion als
// Trainer, nicht als Kind.
//
// `%[1]s` ist die Platzhalterliste der Mannschafts-IDs; sie steht fünfmal, die
// Argumente werden entsprechend fünfmal angehängt.
const teamAudienceQuery = `
	SELECT DISTINCT user_id FROM (
		SELECT m.user_id AS user_id
		FROM kader k
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		WHERE k.team_id IN (%[1]s) AND m.user_id IS NOT NULL

		UNION

		SELECT fl.parent_user_id AS user_id
		FROM kader k
		JOIN kader_members km ON km.kader_id = k.id
		JOIN family_links fl ON fl.member_id = km.member_id
		JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		WHERE k.team_id IN (%[1]s)

		UNION

		SELECT m.user_id AS user_id
		FROM kader k
		JOIN kader_extended_members kem ON kem.kader_id = k.id
		JOIN members m ON m.id = kem.member_id
		JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		WHERE k.team_id IN (%[1]s) AND m.user_id IS NOT NULL

		UNION

		SELECT fl.parent_user_id AS user_id
		FROM kader k
		JOIN kader_extended_members kem ON kem.kader_id = k.id
		JOIN family_links fl ON fl.member_id = kem.member_id
		JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		WHERE k.team_id IN (%[1]s)

		UNION

		SELECT m.user_id AS user_id
		FROM kader k
		JOIN kader_trainers kt ON kt.kader_id = k.id
		JOIN members m ON m.id = kt.member_id
		JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		WHERE k.team_id IN (%[1]s) AND m.user_id IS NOT NULL
	)`

// TeamAudience liefert die Nutzerkonten, die eine Benachrichtigung zu einem
// Termin der übergebenen Mannschaften bekommen: Stammkader UND erweiterter
// Kader jeweils samt Elternteilen (`family_links`), dazu die Trainer des
// Kaders — bezogen auf die aktive Saison.
//
// Der Auslöser einer Änderung wird NICHT herausgefiltert. Wer einen Termin
// anlegt, verschiebt oder absagt, bekommt die eigene Meldung mit: sie ist die
// Bestätigung, dass sie rausgegangen ist, und zeigt im Wortlaut, was die
// Mannschaft gelesen hat. Eine Ausnahme dafür gäbe es nur an einer Stelle
// (hier), sie wäre also billig — sie ist bewusst nicht da.
//
// Das ist die EINZIGE Auflösung dieser Frage im System. Vorher gab es drei
// Kopien (games, trainings, scheduler), die über die Jahre auseinandergelaufen
// sind: der erweiterte Kader bekam Trainings-, aber keine Spielmeldungen, seine
// Eltern gar keine. Wer die Menge ändern will, ändert sie hier — für alle
// Meldungswege gleichzeitig. `internal/arch` hält das mechanisch fest.
//
// Nicht für Dienste verwenden: die Dienstbörse fragt "wer ist dienstpflichtig?"
// und bleibt deshalb auf dem Stammkader (`player_memberships`) — wer im
// erweiterten Kader aushilft, schuldet dem Verein keine Dienststunden.
//
// Bezugsgröße ist die AKTIVE Saison, nicht die Saison des Termins. Das ist das
// unveränderte Verhalten aller drei Vorgänger; Termine außerhalb der aktiven
// Saison erzeugen damit weiterhin keine Meldung.
//
// Die Signatur trägt bewusst kein `error`: eine fehlgeschlagene Benachrichtigung
// darf die auslösende Mutation nie kippen, und alle Aufrufstellen würden den
// Fehler ohnehin nur loggen. Anders als die drei Vorgänger gibt die Funktion bei
// einem Query-Fehler aber nicht stumm nil zurück, sondern protokolliert ihn.
func TeamAudience(db *sql.DB, teamIDs ...int) []int {
	if len(teamIDs) == 0 {
		return nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(teamIDs)), ",")
	args := make([]any, 0, 5*len(teamIDs))
	for range 5 {
		for _, id := range teamIDs {
			args = append(args, id)
		}
	}

	query := strings.ReplaceAll(teamAudienceQuery, "%[1]s", placeholders)
	rows, err := db.Query(query, args...)
	if err != nil {
		slog.Error("notify.TeamAudience query", "team_ids", teamIDs, "error", err)
		return nil
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			slog.Error("notify.TeamAudience scan", "error", err)
			return nil
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		slog.Error("notify.TeamAudience rows", "error", err)
		return nil
	}
	return ids
}
