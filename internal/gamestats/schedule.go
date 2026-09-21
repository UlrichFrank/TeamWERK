package gamestats

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
)

// SaveSchedule schreibt Begegnungen und Tabellenstand einer Staffel.
//
// INVARIANTE: Diese Funktion legt NIEMALS eine Zeile in `games` an und ändert
// dort nichts. Der Staffel-Spielplan umfasst ~90 Begegnungen je Staffel,
// überwiegend fremder Vereine; als Termine würden sie in Spielplan,
// Dienst-Regeneration, iCal-Feed, RSVP und Anwesenheit lecken. Verknüpft wird
// nur in die andere Richtung: wo eine BWHV-Spielnummer einer vorhandenen
// games.external_id entspricht, wird bwhv_games.game_id gesetzt (design.md §7).
func (s *Store) SaveSchedule(ctx context.Context, staffelID int, sch *bwhv.Schedule) (changed int, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, g := range sch.Games {
		gameID, err := lookupOwnGame(ctx, tx, g.GameNo)
		if err != nil {
			return 0, err
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO bwhv_games (staffel_id, game_no, sgid, game_id, date, time,
				home_team, guest_team, home_goals, guest_goals,
				home_goals_ht, guest_goals_ht, hall_number)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT (staffel_id, game_no) DO UPDATE SET
				sgid           = excluded.sgid,
				game_id        = COALESCE(excluded.game_id, bwhv_games.game_id),
				date           = excluded.date,
				time           = excluded.time,
				home_team      = excluded.home_team,
				guest_team     = excluded.guest_team,
				home_goals     = excluded.home_goals,
				guest_goals    = excluded.guest_goals,
				home_goals_ht  = excluded.home_goals_ht,
				guest_goals_ht = excluded.guest_goals_ht,
				hall_number    = excluded.hall_number
			WHERE bwhv_games.sgid           IS NOT excluded.sgid
			   OR bwhv_games.date           IS NOT excluded.date
			   OR bwhv_games.time           IS NOT excluded.time
			   OR bwhv_games.home_goals     IS NOT excluded.home_goals
			   OR bwhv_games.guest_goals    IS NOT excluded.guest_goals
			   OR bwhv_games.home_goals_ht  IS NOT excluded.home_goals_ht
			   OR bwhv_games.guest_goals_ht IS NOT excluded.guest_goals_ht
			   OR bwhv_games.home_team      IS NOT excluded.home_team
			   OR bwhv_games.guest_team     IS NOT excluded.guest_team`,
			staffelID, g.GameNo, g.SGID, gameID, g.Date, g.Time,
			g.HomeTeam, g.GuestTeam, g.HomeGoals, g.GuestGoals,
			g.HomeGoalsHT, g.GuestGoalsHT, g.HallNumber)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			changed++
		}
	}

	tableJSON, err := json.Marshal(sch.Table)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE bwhv_staffeln SET table_json = ?, polled_at = ? WHERE id = ?`,
		string(tableJSON), time.Now().UTC().Format(time.RFC3339), staffelID); err != nil {
		return 0, err
	}
	return changed, tx.Commit()
}

// lookupOwnGame sucht ein eigenes Spiel über die BWHV-Spielnummer. Das ist der
// einzige Berührungspunkt mit `games` — und er ist lesend.
func lookupOwnGame(ctx context.Context, tx *sql.Tx, gameNo string) (any, error) {
	var id int
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM games WHERE external_id = ? LIMIT 1`, gameNo).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}
	return id, nil
}

// StaffelTable liest den gespeicherten Tabellenstand.
func (s *Store) StaffelTable(ctx context.Context, staffelID int) ([]bwhv.TableRow, error) {
	var raw string
	err := s.db.QueryRowContext(ctx,
		`SELECT table_json FROM bwhv_staffeln WHERE id = ?`, staffelID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var rows []bwhv.TableRow
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// ScheduleGame ist eine gespeicherte Begegnung.
type ScheduleGame struct {
	ID           int
	StaffelID    int
	GameNo       string
	SGID         string
	GameID       *int
	Date         string
	Time         string
	HomeTeam     string
	GuestTeam    string
	HomeGoals    *int
	GuestGoals   *int
	HomeGoalsHT  *int
	GuestGoalsHT *int
	HallNumber   string
	HasReport    bool
}

const scheduleCols = `g.id, g.staffel_id, g.game_no, g.sgid, g.game_id, g.date, g.time,
	g.home_team, g.guest_team, g.home_goals, g.guest_goals,
	g.home_goals_ht, g.guest_goals_ht, g.hall_number,
	EXISTS (SELECT 1 FROM bwhv_reports r WHERE r.bwhv_game_id = g.id AND r.state = 'parsed')`

func scanScheduleGames(rows *sql.Rows) ([]ScheduleGame, error) {
	defer rows.Close()
	var out []ScheduleGame
	for rows.Next() {
		var g ScheduleGame
		var gameID sql.NullInt64
		if err := rows.Scan(&g.ID, &g.StaffelID, &g.GameNo, &g.SGID, &gameID, &g.Date, &g.Time,
			&g.HomeTeam, &g.GuestTeam, &g.HomeGoals, &g.GuestGoals,
			&g.HomeGoalsHT, &g.GuestGoalsHT, &g.HallNumber, &g.HasReport); err != nil {
			return nil, err
		}
		if gameID.Valid {
			id := int(gameID.Int64)
			g.GameID = &id
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// StaffelGames liefert den kompletten Spielplan einer Staffel.
func (s *Store) StaffelGames(ctx context.Context, staffelID int) ([]ScheduleGame, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+scheduleCols+` FROM bwhv_games g WHERE g.staffel_id = ? ORDER BY g.date, g.time, g.game_no`,
		staffelID)
	if err != nil {
		return nil, err
	}
	return scanScheduleGames(rows)
}

// GameByID liefert eine einzelne Begegnung.
func (s *Store) GameByID(ctx context.Context, id int) (*ScheduleGame, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+scheduleCols+` FROM bwhv_games g WHERE g.id = ?`, id)
	if err != nil {
		return nil, err
	}
	list, err := scanScheduleGames(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, sql.ErrNoRows
	}
	return &list[0], nil
}

// BwhvGameForGame löst einen TeamWERK-Spieltermin auf die zugehörige
// BWHV-Begegnung auf.
func (s *Store) BwhvGameForGame(ctx context.Context, gameID int) (int, error) {
	var id int
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM bwhv_games WHERE game_id = ? LIMIT 1`, gameID).Scan(&id)
	return id, err
}
