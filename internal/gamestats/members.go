package gamestats

import (
	"context"
	"database/sql"

	"strings"

	"github.com/agnivade/levenshtein"
)

// LinkOwnPlayers ordnet die Spieler einer Staffel den Mitgliedern des eigenen
// Vereins zu, soweit das eindeutig möglich ist.
//
// Herangezogen werden Name (unscharf), Geburtsjahrgang, Trikotnummer und die
// Kaderzugehörigkeit der Saison. Ist die Zuordnung nicht eindeutig, bleibt sie
// OFFEN — geraten wird nicht; die Oberfläche bietet die manuelle Zuordnung an
// (design.md §6.2).
//
// Eine bereits gesetzte member_id wird nie überschrieben: eine manuelle
// Zuordnung hält für die Saison.
func (s *Store) LinkOwnPlayers(ctx context.Context, staffelID, kaderID int) (linked int, err error) {
	members, err := s.kaderMembers(ctx, kaderID)
	if err != nil {
		return 0, err
	}
	if len(members) == 0 {
		return 0, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT p.id, p.name, COALESCE(p.birth_year, 0),
		       COALESCE((SELECT pg.jersey_number FROM bwhv_player_games pg
		                  WHERE pg.player_id = p.id AND pg.jersey_number IS NOT NULL
		                  ORDER BY pg.report_id DESC LIMIT 1), 0)
		  FROM bwhv_players p
		 WHERE p.staffel_id = ? AND p.member_id IS NULL`, staffelID)
	if err != nil {
		return 0, err
	}
	type pending struct {
		id     int
		name   string
		year   int
		number int
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.name, &p.year, &p.number); err != nil {
			rows.Close()
			return 0, err
		}
		todo = append(todo, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, p := range todo {
		m, ok := matchMember(p.name, p.year, p.number, members)
		if !ok {
			continue
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE bwhv_players SET member_id = ? WHERE id = ? AND member_id IS NULL`, m.id, p.id); err != nil {
			return linked, err
		}
		linked++
	}
	return linked, nil
}

type kaderMember struct {
	id     int
	name   string
	year   int
	number int
}

// kaderMembers liest Stamm- und erweiterten Kader einer Mannschaft.
func (s *Store) kaderMembers(ctx context.Context, kaderID int) ([]kaderMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.first_name || ' ' || m.last_name,
		       COALESCE(CAST(substr(m.date_of_birth, 1, 4) AS INTEGER), 0),
		       COALESCE(m.jersey_number, 0)
		  FROM members m
		 WHERE m.id IN (SELECT member_id FROM kader_members WHERE kader_id = ?
		                UNION
		                SELECT member_id FROM kader_extended_members WHERE kader_id = ?)`,
		kaderID, kaderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []kaderMember
	for rows.Next() {
		var m kaderMember
		if err := rows.Scan(&m.id, &m.name, &m.year, &m.number); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// matchMember liefert das eindeutig passende Mitglied.
//
// Der Name entscheidet; Jahrgang und Trikotnummer brechen Gleichstand.
// Bleiben mehrere Kandidaten, gibt es KEINE Zuordnung.
func matchMember(name string, year, number int, members []kaderMember) (kaderMember, bool) {
	var cands []kaderMember
	for _, m := range members {
		if nameClose(name, m.name) {
			cands = append(cands, m)
		}
	}
	if len(cands) == 1 {
		return cands[0], true
	}
	if len(cands) == 0 {
		return kaderMember{}, false
	}
	for _, filter := range []func(kaderMember) bool{
		func(m kaderMember) bool { return year > 0 && m.year == year },
		func(m kaderMember) bool { return number > 0 && m.number == number },
	} {
		var narrowed []kaderMember
		for _, m := range cands {
			if filter(m) {
				narrowed = append(narrowed, m)
			}
		}
		if len(narrowed) == 1 {
			return narrowed[0], true
		}
		if len(narrowed) > 1 {
			cands = narrowed
		}
	}
	return kaderMember{}, false
}

// nameClose vergleicht zwei Personennamen unscharf. Zusätzlich zur
// Levenshtein-Distanz gilt die umgedrehte Reihenfolge ("Nachname Vorname")
// als Treffer — Spielberichte und Mitgliederverwaltung sind sich darüber
// nicht immer einig.
func nameClose(a, b string) bool {
	na, nb := normalizeName(a), normalizeName(b)
	if na == nb {
		return true
	}
	if len([]rune(na)) >= NameMinLengthForFuzzy && len([]rune(nb)) >= NameMinLengthForFuzzy &&
		levenshtein.ComputeDistance(na, nb) <= NameDistanceMax {
		return true
	}
	return sortedTokens(na) == sortedTokens(nb)
}

func sortedTokens(s string) string {
	f := strings.Fields(s)
	for i := 0; i < len(f); i++ {
		for j := i + 1; j < len(f); j++ {
			if f[j] < f[i] {
				f[i], f[j] = f[j], f[i]
			}
		}
	}
	return strings.Join(f, " ")
}

// KaderIDForStaffel liefert den Kader, dessen Staffelcode auf die Staffel zeigt.
func (s *Store) KaderIDForStaffel(ctx context.Context, staffelID int) (int, error) {
	var id int
	err := s.db.QueryRowContext(ctx, `
		SELECT k.id FROM kader k
		  JOIN bwhv_staffeln st ON st.code = k.staffel AND st.season_id = k.season_id
		 WHERE st.id = ? AND k.kind = 'team' LIMIT 1`, staffelID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}
