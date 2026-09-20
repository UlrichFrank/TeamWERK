package gamestats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/agnivade/levenshtein"
)

// NameDistanceMax ist die zulässige Levenshtein-Distanz zwischen zwei
// Schreibweisen desselben Namens. Zwei Zeichen decken die beobachteten
// Varianten ab ("Haßlöcher"/"Hasslocher"), ohne verschiedene Personen
// zusammenzuziehen.
const NameDistanceMax = 2

// NameMinLengthForFuzzy ist die Mindestlänge, ab der überhaupt unscharf
// verglichen wird. Bei kurzen Namen wären zwei Zeichen Abstand ein anderer
// Mensch.
const NameMinLengthForFuzzy = 5

// playerKey ist der Schlüssel, unter dem ein Spieler innerhalb eines Berichts
// wiedergefunden wird.
func playerKey(team, name string) string { return team + "\x00" + name }

// resolvePlayer findet oder erzeugt die Spieler-Identität.
//
// Der NAME ist der Schlüssel, die Trikotnummer bestätigt nur und bricht
// Gleichstand. Grund: ein vergessenes Trikot führt zu einer geliehenen Nummer —
// die Nummer wäre dann der zuverlässig falsche Schlüssel. Namen tragen
// Schreibvarianten, bezeichnen über die Saison aber dieselbe Person
// (design.md §6.1).
//
// Bleibt die Zuordnung mehrdeutig, entsteht ein NEUER Spieler statt einer
// geratenen Verschmelzung, und der Widerspruch wird am Datensatz vermerkt.
func resolvePlayer(ctx context.Context, tx *sql.Tx, staffelID int, team, name string, birthYear, number int) (int, error) {
	name = strings.TrimSpace(name)
	if id, ok, err := exactPlayer(ctx, tx, staffelID, team, name); err != nil || ok {
		return id, err
	}

	cands, err := fuzzyCandidates(ctx, tx, staffelID, team, name)
	if err != nil {
		return 0, err
	}
	if len(cands) == 0 {
		return insertPlayer(ctx, tx, staffelID, team, name, birthYear, "")
	}

	// Die Trikotnummer bestätigt: ein Kandidat, der diese Nummer schon getragen
	// hat, ist die gesuchte Person.
	var confirmed []candidate
	var known bool
	for _, c := range cands {
		wore, err := playerWoreNumber(ctx, tx, c.id, number)
		if err != nil {
			return 0, err
		}
		any, err := playerHasAnyNumber(ctx, tx, c.id)
		if err != nil {
			return 0, err
		}
		known = known || any
		if wore {
			confirmed = append(confirmed, c)
		}
	}
	if len(confirmed) == 1 {
		return confirmed[0].id, noteConflict(ctx, tx, confirmed[0].id, fmt.Sprintf(
			"Schreibvariante %q mit %q zusammengeführt", name, confirmed[0].name))
	}

	// Genau ein ähnlicher Name, und die Nummer widerspricht nicht (weil sie noch
	// unbekannt ist): zusammenführen.
	if len(cands) == 1 && (!known || number <= 0) {
		return cands[0].id, noteConflict(ctx, tx, cands[0].id, fmt.Sprintf(
			"Schreibvariante %q mit %q zusammengeführt", name, cands[0].name))
	}

	// Unscharfer Name UND abweichende Nummer: kein Signal bestätigt die
	// Identität. Hier zu verschmelzen hieße raten — es entsteht eine neue
	// Person mit vermerktem Widerspruch. Ein EXAKTER Name mit anderer Nummer
	// wurde oben längst zusammengeführt; das ist der Fall "Trikot vergessen",
	// den die Namensregel ausdrücklich abdecken soll (design.md §6.1).
	names := make([]string, 0, len(cands))
	for _, c := range cands {
		names = append(names, c.name)
	}
	return insertPlayer(ctx, tx, staffelID, team, name, birthYear, fmt.Sprintf(
		"nicht zusammengeführt mit %s — Name ähnlich, Trikotnummer abweichend",
		strings.Join(names, ", ")))
}

type candidate struct {
	id   int
	name string
}

func exactPlayer(ctx context.Context, tx *sql.Tx, staffelID int, team, name string) (int, bool, error) {
	var id int
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM bwhv_players WHERE staffel_id = ? AND team_name = ? AND name = ?`,
		staffelID, team, name).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		return 0, false, nil
	case err != nil:
		return 0, false, err
	}
	return id, true, nil
}

// fuzzyCandidates liefert die Spieler derselben Mannschaft, deren Name nah
// genug an name liegt.
func fuzzyCandidates(ctx context.Context, tx *sql.Tx, staffelID int, team, name string) ([]candidate, error) {
	if len([]rune(name)) < NameMinLengthForFuzzy {
		return nil, nil
	}
	rows, err := tx.QueryContext(ctx,
		`SELECT id, name FROM bwhv_players WHERE staffel_id = ? AND team_name = ?`, staffelID, team)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.name); err != nil {
			return nil, err
		}
		if len([]rune(c.name)) < NameMinLengthForFuzzy {
			continue
		}
		if levenshtein.ComputeDistance(normalizeName(name), normalizeName(c.name)) <= NameDistanceMax {
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

// normalizeName macht den Vergleich unabhängig von Groß-/Kleinschreibung und
// von den üblichen Umlaut-Umschreibungen, damit "Haßlöcher" und "Hasslocher"
// nicht schon durch die Ersetzung die ganze Distanz aufbrauchen.
func normalizeName(s string) string {
	r := strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
		"Ä", "ae", "Ö", "oe", "Ü", "ue",
	)
	return strings.ToLower(strings.Join(strings.Fields(r.Replace(s)), " "))
}

func playerWoreNumber(ctx context.Context, tx *sql.Tx, playerID, number int) (bool, error) {
	if number <= 0 {
		return false, nil
	}
	var n int
	err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bwhv_player_games WHERE player_id = ? AND jersey_number = ?`,
		playerID, number).Scan(&n)
	return n > 0, err
}

// playerHasAnyNumber meldet, ob zu diesem Spieler überhaupt schon eine
// Trikotnummer bekannt ist. Ohne diese Unterscheidung wäre "Nummer unbekannt"
// von "Nummer widerspricht" nicht zu trennen, und der erste Bericht einer
// Person erzeugte stets einen Widerspruch.
func playerHasAnyNumber(ctx context.Context, tx *sql.Tx, playerID int) (bool, error) {
	var n int
	err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bwhv_player_games WHERE player_id = ? AND jersey_number IS NOT NULL`,
		playerID).Scan(&n)
	return n > 0, err
}

func insertPlayer(ctx context.Context, tx *sql.Tx, staffelID int, team, name string, birthYear int, conflict string) (int, error) {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO bwhv_players (staffel_id, team_name, name, birth_year, conflict)
		 VALUES (?, ?, ?, ?, ?)`,
		staffelID, team, name, nullableInt(birthYear), conflict)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func noteConflict(ctx context.Context, tx *sql.Tx, playerID int, note string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE bwhv_players SET conflict = CASE
			WHEN conflict = '' THEN ?
			WHEN instr(conflict, ?) > 0 THEN conflict
			ELSE conflict || ' | ' || ?
		END WHERE id = ?`, note, note, note, playerID)
	return err
}
