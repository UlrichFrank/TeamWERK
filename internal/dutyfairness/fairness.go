// Package dutyfairness berechnet die Dienst-Bilanz je Kind der aktiven Saison:
// wie viele Dienste bereits geleistet bzw. eingeplant sind, und welcher faire
// Anteil aus den tatsächlich bekannten Dienst-Slots des Kaders folgt.
//
// Alles wird live aus duty_slots/duty_assignments gerechnet — bewusst ohne
// duty_accounts, dessen ist-Wert nur beim Löschen eines Termins nachgezogen
// wird (offener Change dienstkonto-ist-buchung). Ein Codepfad bedient drei
// Sichten: Dashboard-Kachel, anonymisierte Rangliste, Vorstands-Rangliste.
//
// Foundation statt Domain, weil internal/dashboard (Domain) dieselbe Berechnung
// braucht und Domains sich nicht gegenseitig importieren dürfen.
package dutyfairness

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// Member ist ein Kader-Mitglied der aktiven Saison samt seiner Dienst-Zählung.
// Die Zählung hängt am Mitglied, nicht am Kader: steht ein Kind in zwei
// Mannschaften, erscheint es in beiden Ranglisten mit denselben Zahlen.
type Member struct {
	MemberID int
	Name     string
	// UserID ist der eigene Account des Mitglieds (0 = keiner). Kinder-
	// Proxy-Accounts sind ebenfalls eigene Accounts.
	UserID     int
	Geleistet  float64
	Vorhersage float64
}

// Team ist eine Mannschaft mit Kader in der aktiven Saison. Übungsgruppen
// (kader ohne team_id) fehlen strukturell — sie haben keine Dienste.
type Team struct {
	TeamID      int
	Label       string
	PlayerCount int
	// Total ist die Gesamtsumme: team-gebundene Slots voll plus der nach
	// Spieleranzahl anteilige Teil der generischen Slots (Bruchzahl möglich).
	Total float64
	// Soll ist der Fair-Anteil je Kind = Total / PlayerCount (0 ohne Spieler).
	Soll    float64
	Members []*Member
}

// Snapshot ist der vollständige Rechenstand einer Saison. Er wird pro Request
// neu gebaut (wenige, gebatchte Queries, kein N+1) und nie gespeichert.
type Snapshot struct {
	Teams map[int]*Team
	// TeamOrder listet alle Team-IDs nach Label sortiert.
	TeamOrder []int

	members     map[int]*Member
	memberTeams map[int][]int
	// linked: user_id → Mitglieder, mit denen der Account verbunden ist (eigenes
	// Mitglied oder Kind via family_links), jeweils nur Mitglieder mit Kader.
	ownByUser      map[int][]int
	childrenByUser map[int][]int
}

// today liefert das heutige Datum in Vereinszeit. Ein Termin von heute zählt als
// Vorhersage, nicht als geleistet (Spec: „heute eingeschlossen").
func today() string {
	return time.Now().In(timez.Berlin()).Format("2006-01-02")
}

type slotInfo struct {
	date    string
	teams   []int // leer = generisch
	generic bool
}

// Compute baut den Snapshot der Saison.
func Compute(ctx context.Context, db *sql.DB, seasonID int) (*Snapshot, error) {
	s := &Snapshot{
		Teams:          map[int]*Team{},
		members:        map[int]*Member{},
		memberTeams:    map[int][]int{},
		ownByUser:      map[int][]int{},
		childrenByUser: map[int][]int{},
	}
	if err := s.loadTeams(ctx, db, seasonID); err != nil {
		return nil, err
	}
	if err := s.loadMembers(ctx, db, seasonID); err != nil {
		return nil, err
	}
	if err := s.loadFamilyLinks(ctx, db); err != nil {
		return nil, err
	}
	slots, err := s.loadSlots(ctx, db, seasonID)
	if err != nil {
		return nil, err
	}
	if err := s.countAssignments(ctx, db, seasonID, slots); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Snapshot) loadTeams(ctx context.Context, db *sql.DB, seasonID int) error {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT k.team_id, COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name)
		FROM kader k
		JOIN teams t ON t.id = k.team_id
		WHERE k.season_id = ? AND k.team_id IS NOT NULL`, seasonID)
	if err != nil {
		return fmt.Errorf("dutyfairness teams: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		t := &Team{Members: []*Member{}}
		if err := rows.Scan(&t.TeamID, &t.Label); err != nil {
			return fmt.Errorf("dutyfairness teams scan: %w", err)
		}
		s.Teams[t.TeamID] = t
		s.TeamOrder = append(s.TeamOrder, t.TeamID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	slices.SortFunc(s.TeamOrder, func(a, b int) int {
		return cmp.Or(cmp.Compare(s.Teams[a].Label, s.Teams[b].Label), cmp.Compare(a, b))
	})
	return nil
}

func (s *Snapshot) loadMembers(ctx context.Context, db *sql.DB, seasonID int) error {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT k.team_id, m.id, m.first_name || ' ' || m.last_name, COALESCE(m.user_id, 0)
		FROM kader_members km
		JOIN kader k ON k.id = km.kader_id
		JOIN members m ON m.id = km.member_id
		WHERE k.season_id = ? AND k.team_id IS NOT NULL`, seasonID)
	if err != nil {
		return fmt.Errorf("dutyfairness members: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var teamID, memberID, userID int
		var name string
		if err := rows.Scan(&teamID, &memberID, &name, &userID); err != nil {
			return fmt.Errorf("dutyfairness members scan: %w", err)
		}
		t := s.Teams[teamID]
		if t == nil {
			continue
		}
		m := s.members[memberID]
		if m == nil {
			m = &Member{MemberID: memberID, Name: name, UserID: userID}
			s.members[memberID] = m
			if userID > 0 {
				s.ownByUser[userID] = append(s.ownByUser[userID], memberID)
			}
		}
		t.Members = append(t.Members, m)
		t.PlayerCount++
		s.memberTeams[memberID] = append(s.memberTeams[memberID], teamID)
	}
	return rows.Err()
}

func (s *Snapshot) loadFamilyLinks(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT parent_user_id, member_id FROM family_links`)
	if err != nil {
		return fmt.Errorf("dutyfairness family_links: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var parentUserID, memberID int
		if err := rows.Scan(&parentUserID, &memberID); err != nil {
			return fmt.Errorf("dutyfairness family_links scan: %w", err)
		}
		if s.members[memberID] != nil {
			s.childrenByUser[parentUserID] = append(s.childrenByUser[parentUserID], memberID)
		}
	}
	return rows.Err()
}

// loadSlots summiert die Gesamtsumme je Team und liefert die Slot-Zuordnung für
// die Zählung der Zuweisungen. Ein Slot gehört über game_id → game_teams zu den
// Mannschaften des Spiels, sonst über seine direkte team_id; ohne beides ist er
// generisch (z. B. Vereinsfest) und wird nach Spieleranzahl auf alle Kader
// verteilt. Ein Spiel mehrerer Mannschaften zählt für jede davon voll.
func (s *Snapshot) loadSlots(ctx context.Context, db *sql.DB, seasonID int) (map[int]slotInfo, error) {
	gameTeams := map[int][]int{}
	gtRows, err := db.QueryContext(ctx, `
		SELECT gt.game_id, gt.team_id
		FROM game_teams gt
		JOIN games g ON g.id = gt.game_id
		WHERE g.season_id = ?`, seasonID)
	if err != nil {
		return nil, fmt.Errorf("dutyfairness game_teams: %w", err)
	}
	for gtRows.Next() {
		var gameID, teamID int
		if err := gtRows.Scan(&gameID, &teamID); err != nil {
			gtRows.Close()
			return nil, fmt.Errorf("dutyfairness game_teams scan: %w", err)
		}
		gameTeams[gameID] = append(gameTeams[gameID], teamID)
	}
	gtRows.Close()
	if err := gtRows.Err(); err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, event_date, slots_total, game_id, team_id
		FROM duty_slots
		WHERE season_id = ?`, seasonID)
	if err != nil {
		return nil, fmt.Errorf("dutyfairness slots: %w", err)
	}
	defer rows.Close()

	slots := map[int]slotInfo{}
	var genericTotal float64
	for rows.Next() {
		var id, slotsTotal int
		var date string
		var gameID, teamID sql.NullInt64
		if err := rows.Scan(&id, &date, &slotsTotal, &gameID, &teamID); err != nil {
			return nil, fmt.Errorf("dutyfairness slots scan: %w", err)
		}
		var teams []int
		switch {
		case gameID.Valid:
			teams = gameTeams[int(gameID.Int64)]
		case teamID.Valid:
			teams = []int{int(teamID.Int64)}
		}
		info := slotInfo{date: dateOnly(date), teams: teams, generic: len(teams) == 0}
		slots[id] = info
		if info.generic {
			genericTotal += float64(slotsTotal)
			continue
		}
		for _, tid := range teams {
			if t := s.Teams[tid]; t != nil {
				t.Total += float64(slotsTotal)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var totalPlayers int
	for _, t := range s.Teams {
		totalPlayers += t.PlayerCount
	}
	for _, t := range s.Teams {
		if totalPlayers > 0 {
			t.Total += genericTotal * float64(t.PlayerCount) / float64(totalPlayers)
		}
		if t.PlayerCount > 0 {
			t.Soll = t.Total / float64(t.PlayerCount)
		}
	}
	return slots, nil
}

// countAssignments rechnet jede Zuweisung genau einem Mitglied (oder anteilig
// mehreren) zu. Gezählt wird nur die Existenz — der status ist bedeutungslos,
// weil „Erfüllt" nicht zuverlässig geklickt wird (design.md Entscheidung 1).
//
// Eine Zuweisung trägt nur eine user_id. Zurechnung in dieser Reihenfolge,
// die erste nicht-leere Stufe gewinnt und wird gleichmäßig geteilt:
//  1. eigene Mitglieder des Accounts, deren Team zum Slot passt
//  2. Kinder des Accounts (family_links), deren Team zum Slot passt
//  3. eigene Mitglieder des Accounts, unabhängig vom Team
//
// Ein generischer Slot passt zu jedem Team. Eine Eltern-Zuweisung an einem
// Slot, zu dem kein Kind passt, zählt für niemanden (Team-Match); geteilt wird
// nur zwischen Geschwistern, zu denen der Slot passt — so bleibt die Summe
// einer Familie gleich der Zahl ihrer tatsächlichen Dienste.
func (s *Snapshot) countAssignments(ctx context.Context, db *sql.DB, seasonID int, slots map[int]slotInfo) error {
	rows, err := db.QueryContext(ctx, `
		SELECT da.duty_slot_id, da.user_id
		FROM duty_assignments da
		JOIN duty_slots ds ON ds.id = da.duty_slot_id
		WHERE ds.season_id = ?`, seasonID)
	if err != nil {
		return fmt.Errorf("dutyfairness assignments: %w", err)
	}
	defer rows.Close()

	now := today()
	for rows.Next() {
		var slotID, userID int
		if err := rows.Scan(&slotID, &userID); err != nil {
			return fmt.Errorf("dutyfairness assignments scan: %w", err)
		}
		slot, ok := slots[slotID]
		if !ok {
			continue
		}
		targets := s.matching(s.ownByUser[userID], slot)
		if len(targets) == 0 {
			targets = s.matching(s.childrenByUser[userID], slot)
		}
		if len(targets) == 0 {
			targets = s.ownByUser[userID]
		}
		if len(targets) == 0 {
			continue
		}
		weight := 1.0 / float64(len(targets))
		for _, memberID := range targets {
			m := s.members[memberID]
			if slot.date < now {
				m.Geleistet += weight
			} else {
				m.Vorhersage += weight
			}
		}
	}
	return rows.Err()
}

func (s *Snapshot) matching(memberIDs []int, slot slotInfo) []int {
	var out []int
	for _, id := range memberIDs {
		if slot.generic || slices.ContainsFunc(s.memberTeams[id], func(tid int) bool {
			return slices.Contains(slot.teams, tid)
		}) {
			out = append(out, id)
		}
	}
	return out
}

// LinkedMembers liefert die Mitglieder, deren Zeile ein Account als eigene sieht:
// das eigene Mitglied und die Kinder via family_links (nur Mitglieder mit Kader).
func (s *Snapshot) LinkedMembers(userID int) map[int]bool {
	out := map[int]bool{}
	for _, id := range s.ownByUser[userID] {
		out[id] = true
	}
	for _, id := range s.childrenByUser[userID] {
		out[id] = true
	}
	return out
}

// TeamsFor liefert die Teams (in TeamOrder), in deren Kader mindestens eines der
// gegebenen Mitglieder steht — der „eigene Teams"-Scope eines Standard-Nutzers.
func (s *Snapshot) TeamsFor(memberIDs map[int]bool) []int {
	var out []int
	for _, tid := range s.TeamOrder {
		if slices.ContainsFunc(s.Teams[tid].Members, func(m *Member) bool { return memberIDs[m.MemberID] }) {
			out = append(out, tid)
		}
	}
	return out
}

// Position ist eine Zeile der Dashboard-Kachel: ein verbundenes Mitglied in
// einem seiner Teams.
type Position struct {
	Member *Member
	Team   *Team
}

// PositionsFor liefert eine Position je (verbundenes Mitglied × Team): das
// eigene Mitglied zuerst, dann die Kinder, jeweils nach Name und Team-Label.
func (s *Snapshot) PositionsFor(userID int) []Position {
	var out []Position
	add := func(memberIDs []int) {
		ids := slices.Clone(memberIDs)
		slices.SortFunc(ids, func(a, b int) int {
			return cmp.Or(cmp.Compare(s.members[a].Name, s.members[b].Name), cmp.Compare(a, b))
		})
		for _, id := range slices.Compact(ids) {
			for _, tid := range s.TeamOrder {
				if slices.Contains(s.memberTeams[id], tid) {
					out = append(out, Position{Member: s.members[id], Team: s.Teams[tid]})
				}
			}
		}
	}
	own := s.ownByUser[userID]
	add(own)
	add(slices.DeleteFunc(slices.Clone(s.childrenByUser[userID]), func(id int) bool {
		return slices.Contains(own, id)
	}))
	return out
}

// Ranked liefert die Mitglieder eines Teams in Ranglisten-Reihenfolge:
// geleistet+vorhersage absteigend, bei Gleichstand member_id aufsteigend —
// jede Zeile bekommt einen eindeutigen Platz (design.md Entscheidung 7).
// Verglichen wird auf zwei Nachkommastellen gerundet, damit geteilte Anteile
// (1/3 + 2/3) nicht an Float-Rauschen einen Gleichstand verlieren.
func (t *Team) Ranked() []*Member {
	out := slices.Clone(t.Members)
	slices.SortFunc(out, func(a, b *Member) int {
		return cmp.Or(
			cmp.Compare(Round2(b.Geleistet+b.Vorhersage), Round2(a.Geleistet+a.Vorhersage)),
			cmp.Compare(a.MemberID, b.MemberID),
		)
	})
	return out
}

// Round2 rundet auf zwei Nachkommastellen — für JSON und Sortierung.
func Round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// dateOnly schneidet SQLite-DATE-Werte auf "2006-01-02" (Gotcha: der Treiber
// liefert DATE-Spalten als "…T00:00:00Z").
func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
