package gamestats

import (
	"context"
	"database/sql"
)

// PlayerStat ist die Saisonbilanz eines Spielers.
type PlayerStat struct {
	PlayerID    int     `json:"playerId"`
	MemberID    *int    `json:"memberId"`
	Name        string  `json:"name"`
	TeamName    string  `json:"teamName"`
	Games       int     `json:"games"`
	Goals       int     `json:"goals"`
	SevenMAtt   int     `json:"sevenMAttempts"`
	SevenMGoals int     `json:"sevenMGoals"`
	TwoMin      int     `json:"twoMin"`
	Warnings    int     `json:"warnings"`
	Disq        int     `json:"disq"`
	FairPlay    float64 `json:"fairPlayScore"`
}

// fairPlayScore gewichtet Disziplinarmaßnahmen: eine Disqualifikation wiegt
// schwerer als eine Zeitstrafe, eine Verwarnung am wenigsten. Kleiner ist
// besser.
func fairPlayScore(twoMin, warnings, disq int) float64 {
	return float64(warnings)*0.5 + float64(twoMin)*1 + float64(disq)*3
}

// statSelect summiert die Mannschaftslisten-Zeilen ausgewerteter Berichte.
// Berichte im Zustand parse_failed tragen keine Zeilen und fallen damit von
// selbst heraus — der Filter steht trotzdem explizit da.
const statSelect = `
	SELECT p.id, p.member_id, p.name, p.team_name,
	       COUNT(pg.report_id),
	       COALESCE(SUM(pg.goals), 0),
	       COALESCE(SUM(pg.seven_m_attempts), 0),
	       COALESCE(SUM(pg.seven_m_goals), 0),
	       COALESCE(SUM(pg.two_min), 0),
	       COALESCE(SUM(pg.yellow), 0),
	       COALESCE(SUM(pg.red), 0)
	  FROM bwhv_players p
	  JOIN bwhv_player_games pg ON pg.player_id = p.id
	  JOIN bwhv_reports r ON r.id = pg.report_id AND r.state = 'parsed'`

func scanStats(rows *sql.Rows) ([]PlayerStat, error) {
	defer rows.Close()
	var out []PlayerStat
	for rows.Next() {
		var s PlayerStat
		var member sql.NullInt64
		if err := rows.Scan(&s.PlayerID, &member, &s.Name, &s.TeamName, &s.Games,
			&s.Goals, &s.SevenMAtt, &s.SevenMGoals, &s.TwoMin, &s.Warnings, &s.Disq); err != nil {
			return nil, err
		}
		if member.Valid {
			id := int(member.Int64)
			s.MemberID = &id
		}
		s.FairPlay = fairPlayScore(s.TwoMin, s.Warnings, s.Disq)
		out = append(out, s)
	}
	return out, rows.Err()
}

// StaffelStats liefert die Saisonbilanz aller Spieler einer Staffel —
// eigene wie fremde.
func (s *Store) StaffelStats(ctx context.Context, staffelID int) ([]PlayerStat, error) {
	rows, err := s.db.QueryContext(ctx, statSelect+`
		 WHERE p.staffel_id = ?
		 GROUP BY p.id
		 ORDER BY 6 DESC, p.name`, staffelID)
	if err != nil {
		return nil, err
	}
	return scanStats(rows)
}

// MemberStats liefert die Saisonbilanz eines Mitglieds über alle Staffeln der
// Saison. Ein Mitglied kann in mehreren Staffeln auflaufen (Doppelspielrecht),
// deshalb je Staffel eine Zeile.
func (s *Store) MemberStats(ctx context.Context, memberID, seasonID int) ([]PlayerStat, error) {
	rows, err := s.db.QueryContext(ctx, statSelect+`
		  JOIN bwhv_staffeln st ON st.id = p.staffel_id
		 WHERE p.member_id = ? AND st.season_id = ?
		 GROUP BY p.id
		 ORDER BY st.code`, memberID, seasonID)
	if err != nil {
		return nil, err
	}
	return scanStats(rows)
}

// ReportDetail ist ein Bericht mit Mannschaftslisten und Spielverlauf.
type ReportDetail struct {
	ReportID   int          `json:"reportId"`
	State      string       `json:"state"`
	Spectators string       `json:"spectators"`
	Referees   string       `json:"referees"`
	Warnings   []string     `json:"warnings"`
	HasPDF     bool         `json:"hasPdf"`
	Players    []PlayerLine `json:"players"`
	Events     []EventLine  `json:"events"`
}

// PlayerLine ist eine Spielerzeile eines Berichts.
type PlayerLine struct {
	PlayerID    int    `json:"playerId"`
	MemberID    *int   `json:"memberId"`
	Name        string `json:"name"`
	Side        string `json:"side"`
	Number      *int   `json:"number"`
	Goals       int    `json:"goals"`
	SevenMAtt   int    `json:"sevenMAttempts"`
	SevenMGoals int    `json:"sevenMGoals"`
	TwoMin      int    `json:"twoMin"`
	Warnings    int    `json:"warnings"`
	Disq        int    `json:"disq"`
	Conflict    string `json:"conflict"`
}

// EventLine ist eine Zeile des Spielverlaufs.
type EventLine struct {
	Seq        int     `json:"seq"`
	ClockTime  string  `json:"clockTime"`
	GameSecond int     `json:"gameSecond"`
	ScoreHome  *int    `json:"scoreHome"`
	ScoreGuest *int    `json:"scoreGuest"`
	Kind       string  `json:"kind"`
	Side       string  `json:"side"`
	PlayerID   *int    `json:"playerId"`
	PlayerName *string `json:"playerName"`
	Number     *int    `json:"number"`
	RawText    string  `json:"rawText"`
}

// ReportForGame liefert den ausgewerteten Bericht einer Begegnung.
// Fehlt er oder ist er gescheitert, meldet die Funktion sql.ErrNoRows.
func (s *Store) ReportForGame(ctx context.Context, bwhvGameID int) (*ReportDetail, error) {
	var d ReportDetail
	var warnings, pdfPath string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, state, spectators, referees, warnings_json, pdf_path
		  FROM bwhv_reports WHERE bwhv_game_id = ? AND state = 'parsed'`, bwhvGameID).
		Scan(&d.ReportID, &d.State, &d.Spectators, &d.Referees, &warnings, &pdfPath)
	if err != nil {
		return nil, err
	}
	d.HasPDF = pdfPath != ""
	d.Warnings = decodeWarnings(warnings)

	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.member_id, p.name, pg.side, pg.jersey_number,
		       pg.goals, pg.seven_m_attempts, pg.seven_m_goals,
		       pg.two_min, pg.yellow, pg.red, p.conflict
		  FROM bwhv_player_games pg
		  JOIN bwhv_players p ON p.id = pg.player_id
		 WHERE pg.report_id = ?
		 ORDER BY pg.side, pg.jersey_number`, d.ReportID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p PlayerLine
		var member sql.NullInt64
		if err := rows.Scan(&p.PlayerID, &member, &p.Name, &p.Side, &p.Number,
			&p.Goals, &p.SevenMAtt, &p.SevenMGoals, &p.TwoMin, &p.Warnings, &p.Disq, &p.Conflict); err != nil {
			rows.Close()
			return nil, err
		}
		if member.Valid {
			id := int(member.Int64)
			p.MemberID = &id
		}
		d.Players = append(d.Players, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	erows, err := s.db.QueryContext(ctx, `
		SELECT e.seq, e.clock_time, e.game_second, e.score_home, e.score_guest,
		       e.kind, e.side, e.player_id, p.name, e.jersey_number, e.raw_text
		  FROM bwhv_events e
		  LEFT JOIN bwhv_players p ON p.id = e.player_id
		 WHERE e.report_id = ? ORDER BY e.seq`, d.ReportID)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var e EventLine
		var pid sql.NullInt64
		var pname sql.NullString
		if err := erows.Scan(&e.Seq, &e.ClockTime, &e.GameSecond, &e.ScoreHome, &e.ScoreGuest,
			&e.Kind, &e.Side, &pid, &pname, &e.Number, &e.RawText); err != nil {
			return nil, err
		}
		if pid.Valid {
			id := int(pid.Int64)
			e.PlayerID = &id
		}
		if pname.Valid {
			n := pname.String
			e.PlayerName = &n
		}
		d.Events = append(d.Events, e)
	}
	return &d, erows.Err()
}

// ReportPDFPath liefert den Ablagepfad eines Berichts.
func (s *Store) ReportPDFPath(ctx context.Context, reportID int) (string, error) {
	var p string
	err := s.db.QueryRowContext(ctx, `SELECT pdf_path FROM bwhv_reports WHERE id = ?`, reportID).Scan(&p)
	return p, err
}
