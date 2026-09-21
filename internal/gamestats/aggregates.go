package gamestats

import (
	"context"
	"sort"
)

// Dieses File trägt die Auswertungen, die aus den bereits gespeicherten
// Ergebnissen und Spielberichten einer Staffel gerechnet werden: Kreuztabelle,
// Platzierungsverlauf, Mannschafts- und Schiedsrichter-Ranglisten.
//
// Entscheidung 1 aus design.md zieht sich durch alle: Tor-Werte kommen aus
// bwhv_games (den Ergebnissen), NICHT aus der Summe der Spielerzeilen. Eine
// Begegnung ohne freigegebenes PDF fehlte sonst in jeder Statistik, und ein
// Bericht mit abweichender Mannschaftsliste (ein gespeicherter Fall mit
// Warnung) lieferte eine andere Summe als der amtliche Endstand. Fair-Play und
// Torverteilung hängen zwangsläufig an den Berichten — deshalb stehen in
// derselben Ansicht zwei verschiedene Abdeckungen nebeneinander, und jede
// Mannschafts-Zeile weist beide Spielzahlen aus.

// trimDate schneidet einen gespeicherten DATE-Wert auf "2006-01-02".
//
// SQLite liefert DATE-Spalten je nach Schreibweg als reines Datum oder als
// ISO-Timestamp ("2026-09-20T00:00:00Z"). Beides landet hier als
// Gruppierungsschlüssel und als Anzeigewert; ohne den Schnitt bildeten
// dieselben Spieltage zwei Gruppen (docs/agent/06-gotchas.md).
func trimDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// CrossCell ist eine Zelle der Kreuztabelle.
//
// Played unterscheidet die beiden Fälle ausdrücklich, statt sie an einem
// Nullwert abzulesen: ein 0:0 ist ein gespieltes Spiel. Das Vorbild muss
// "beide Zahlen 0" als "nicht gespielt" behandeln und verliert damit jedes
// echte torlose Spiel — mit der NULL-Prüfung auf home_goals braucht es diese
// Krücke nicht.
type CrossCell struct {
	BwhvGameID int    `json:"bwhvGameId"`
	Played     bool   `json:"played"`
	HomeGoals  *int   `json:"homeGoals"`
	GuestGoals *int   `json:"guestGoals"`
	Date       string `json:"date"`
}

// CrossRow ist eine Zeile der Kreuztabelle: eine Heimmannschaft und ihre
// Zellen, parallel zu CrossTable.Teams. Ein nil-Eintrag ist die Diagonale oder
// eine Paarung ohne angesetzte Begegnung.
type CrossRow struct {
	Team  string       `json:"team"`
	Cells []*CrossCell `json:"cells"`
}

// CrossTable ist die Matrix aller Mannschaften einer Staffel: Heim in der
// Zeile, Gast in der Spalte.
type CrossTable struct {
	Teams []string   `json:"teams"`
	Rows  []CrossRow `json:"rows"`
}

// crossGame ist eine Begegnung, soweit die Kreuztabelle und der Verlauf sie
// brauchen.
type crossGame struct {
	id         int
	date       string
	home       string
	guest      string
	homeGoals  *int
	guestGoals *int
}

// played meldet, ob ein Ergebnis erfasst ist. Das ist die Definition von
// "gespielt" in diesem Paket — und der Grund, warum ein 0:0 nicht verschwindet.
func (g crossGame) played() bool { return g.homeGoals != nil && g.guestGoals != nil }

// loadGames liest die Begegnungen einer Staffel nach Datum sortiert.
func (s *Store) loadGames(ctx context.Context, staffelID int) ([]crossGame, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, date, home_team, guest_team, home_goals, guest_goals
		  FROM bwhv_games
		 WHERE staffel_id = ?
		 ORDER BY date, time, game_no`, staffelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []crossGame
	for rows.Next() {
		var g crossGame
		if err := rows.Scan(&g.id, &g.date, &g.home, &g.guest, &g.homeGoals, &g.guestGoals); err != nil {
			return nil, err
		}
		g.date = trimDate(g.date)
		out = append(out, g)
	}
	return out, rows.Err()
}

// teamsOf sammelt alle Mannschaften einer Staffel in stabiler Reihenfolge.
//
// Sortiert wird alphabetisch und nicht nach dem Tabellenstand: der Snapshot in
// bwhv_staffeln.table_json kann fehlen (vor dem ersten Abruf) oder älter sein
// als der Spielplan, und eine Achse, die je nach Abrufzustand die Reihenfolge
// wechselt, macht die Matrix unlesbar.
func teamsOf(games []crossGame) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range games {
		for _, t := range []string{g.home, g.guest} {
			if t != "" && !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

// CrossTable liefert die Kreuztabelle einer Staffel.
//
// Sie braucht KEINEN ausgewerteten Spielbericht: Ergebnis und Datum stehen
// beide in bwhv_games.
func (s *Store) CrossTable(ctx context.Context, staffelID int) (*CrossTable, error) {
	games, err := s.loadGames(ctx, staffelID)
	if err != nil {
		return nil, err
	}
	teams := teamsOf(games)
	idx := make(map[string]int, len(teams))
	for i, t := range teams {
		idx[t] = i
	}

	ct := &CrossTable{Teams: teams, Rows: make([]CrossRow, len(teams))}
	for i, t := range teams {
		ct.Rows[i] = CrossRow{Team: t, Cells: make([]*CrossCell, len(teams))}
	}
	for _, g := range games {
		row, ok := idx[g.home]
		if !ok {
			continue
		}
		col, ok := idx[g.guest]
		if !ok || row == col {
			// Die Diagonale bleibt leer: eine Mannschaft spielt nicht gegen
			// sich selbst.
			continue
		}
		c := &CrossCell{BwhvGameID: g.id, Played: g.played()}
		if c.Played {
			c.HomeGoals, c.GuestGoals = g.homeGoals, g.guestGoals
		} else {
			c.Date = g.date
		}
		ct.Rows[row].Cells[col] = c
	}
	return ct, nil
}
