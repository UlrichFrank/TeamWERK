package gamestats

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
)

// PendingReport ist eine Begegnung mit freigegebenem Bericht, der noch nicht
// verarbeitet ist.
type PendingReport struct {
	BwhvGameID int
	StaffelID  int
	GameNo     string
	SGID       string
	Attempts   int
}

// PendingReports liefert Begegnungen mit sGID, zu denen noch kein
// ausgewerteter Bericht vorliegt.
//
// Berichte im Zustand parse_failed werden MIT geliefert: ihr PDF liegt bereits
// auf der Platte, und ein Parser-Fix soll sie ohne erneuten Fremdabruf
// nachverarbeiten können.
func (s *Store) PendingReports(ctx context.Context, staffelID int) ([]PendingReport, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.staffel_id, g.game_no, g.sgid, COALESCE(r.attempts, 0)
		  FROM bwhv_games g
		  LEFT JOIN bwhv_reports r ON r.bwhv_game_id = g.id
		 WHERE g.staffel_id = ?
		   AND TRIM(g.sgid) <> ''
		   AND (r.id IS NULL OR r.state <> 'parsed')
		 ORDER BY g.date, g.game_no`, staffelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingReport
	for rows.Next() {
		var p PendingReport
		if err := rows.Scan(&p.BwhvGameID, &p.StaffelID, &p.GameNo, &p.SGID, &p.Attempts); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ReportStore legt die PDFs ab.
type ReportStore struct{ dir string }

// NewReportStore liefert eine Ablage unter dir.
func NewReportStore(dir string) *ReportStore { return &ReportStore{dir: dir} }

// Save schreibt das PDF und liefert den relativen Pfad.
func (rs *ReportStore) Save(seasonID int, gameNo string, raw []byte) (string, error) {
	rel := filepath.Join(fmt.Sprintf("%d", seasonID), gameNo+".pdf")
	full := filepath.Join(rs.dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, raw, 0o640); err != nil {
		return "", err
	}
	return rel, nil
}

// Open öffnet ein abgelegtes PDF.
func (rs *ReportStore) Open(rel string) (*os.File, error) {
	if rel == "" {
		return nil, os.ErrNotExist
	}
	return os.Open(filepath.Join(rs.dir, filepath.Clean("/"+rel)))
}

// MarkAttempt hält einen Abrufversuch fest, ohne den Bericht zu verwerten.
// Ein Transportfehler bleibt damit wiederholbar (Zustand pending).
func (s *Store) MarkAttempt(ctx context.Context, bwhvGameID int, sgid string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bwhv_reports (bwhv_game_id, sgid, state, attempts, fetched_at)
		VALUES (?, ?, 'pending', 1, ?)
		ON CONFLICT (bwhv_game_id) DO UPDATE SET
			attempts = bwhv_reports.attempts + 1, fetched_at = excluded.fetched_at`,
		bwhvGameID, sgid, nowUTC())
	return err
}

// MarkFailed setzt einen Bericht auf parse_failed.
//
// Es werden KEINE Spieler- oder Verlaufszeilen geschrieben, und vorhandene
// werden entfernt: ein halb ausgewerteter Bericht in der Datenbank wäre
// schlimmer als gar keiner, weil er plausibel aussieht (design.md §5.3).
func (s *Store) MarkFailed(ctx context.Context, bwhvGameID int, sgid, pdfPath, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO bwhv_reports (bwhv_game_id, sgid, state, pdf_path, failure_reason, attempts, fetched_at)
		VALUES (?, ?, 'parse_failed', ?, ?, 1, ?)
		ON CONFLICT (bwhv_game_id) DO UPDATE SET
			state = 'parse_failed', pdf_path = excluded.pdf_path,
			failure_reason = excluded.failure_reason,
			attempts = bwhv_reports.attempts + 1, fetched_at = excluded.fetched_at,
			parsed_at = NULL`,
		bwhvGameID, sgid, pdfPath, reason, nowUTC()); err != nil {
		return err
	}
	var reportID int
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM bwhv_reports WHERE bwhv_game_id = ?`, bwhvGameID).Scan(&reportID); err != nil {
		return err
	}
	for _, stmt := range []string{
		`DELETE FROM bwhv_player_games WHERE report_id = ?`,
		`DELETE FROM bwhv_events WHERE report_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, reportID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SaveReport persistiert einen ausgewerteten Bericht samt Spielern und
// Spielverlauf. Die Spieler-Identität löst resolvePlayer auf.
func (s *Store) SaveReport(ctx context.Context, p PendingReport, rep *bwhv.Report, pdfPath string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	warnings, err := json.Marshal(rep.Warnings)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO bwhv_reports (bwhv_game_id, sgid, state, pdf_path, spectators, referees,
			warnings_json, failure_reason, attempts, fetched_at, parsed_at)
		VALUES (?, ?, 'parsed', ?, ?, ?, ?, '', 1, ?, ?)
		ON CONFLICT (bwhv_game_id) DO UPDATE SET
			state = 'parsed', pdf_path = excluded.pdf_path,
			spectators = excluded.spectators, referees = excluded.referees,
			warnings_json = excluded.warnings_json, failure_reason = '',
			attempts = bwhv_reports.attempts + 1,
			fetched_at = excluded.fetched_at, parsed_at = excluded.parsed_at`,
		p.BwhvGameID, p.SGID, pdfPath, rep.Header.Spectators, rep.Header.Referees,
		string(warnings), nowUTC(), nowUTC()); err != nil {
		return err
	}

	var reportID int
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM bwhv_reports WHERE bwhv_game_id = ?`, p.BwhvGameID).Scan(&reportID); err != nil {
		return err
	}
	for _, stmt := range []string{
		`DELETE FROM bwhv_player_games WHERE report_id = ?`,
		`DELETE FROM bwhv_events WHERE report_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, reportID); err != nil {
			return err
		}
	}

	// Spieler auflösen: Schlüssel ist (Staffel, Mannschaft, Name).
	playerIDs := map[string]int{}
	for _, ros := range []bwhv.Roster{rep.Home, rep.Guest} {
		for _, pl := range ros.Players {
			id, err := resolvePlayer(ctx, tx, p.StaffelID, ros.TeamName, pl.Name, pl.BirthYear, pl.Number)
			if err != nil {
				return err
			}
			playerIDs[playerKey(ros.TeamName, pl.Name)] = id
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO bwhv_player_games (report_id, player_id, side, jersey_number,
					goals, seven_m_attempts, seven_m_goals, two_min, yellow, red, blue)
				VALUES (?,?,?,?,?,?,?,?,?,?,0)`,
				reportID, id, ros.Side, pl.Number, pl.Goals, pl.SevenMAtt, pl.SevenMGoals,
				pl.TwoMin, pl.Warnings, pl.Disq); err != nil {
				return err
			}
		}
	}

	for _, e := range rep.Events {
		var playerID any
		if e.PlayerName != "" {
			if id, ok := playerIDs[playerKey(teamOf(e, rep), e.PlayerName)]; ok {
				playerID = id
			}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO bwhv_events (report_id, seq, clock_time, game_second,
				score_home, score_guest, kind, side, player_id, jersey_number, raw_text)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			reportID, e.Seq, e.ClockTime, e.GameSecond, e.ScoreHome, e.ScoreGuest,
			string(e.Kind), e.Side, playerID, nullableInt(e.Number), e.RawText); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// teamOf liefert den Mannschaftsnamen eines Ereignisses in der Schreibweise
// der Mannschaftsliste. Der Verlauf nutzt Kurzformen, die Liste die Langform —
// die Seite ist die verlässliche Brücke.
func teamOf(e bwhv.Event, rep *bwhv.Report) string {
	switch e.Side {
	case "home":
		return rep.Home.TeamName
	case "guest":
		return rep.Guest.TeamName
	}
	return e.TeamName
}

var errNoSeason = errors.New("keine aktive Saison")

// ActiveSeason liefert die aktive Saison.
func (s *Store) ActiveSeason(ctx context.Context) (int, error) {
	var id int
	err := s.db.QueryRowContext(ctx, `SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, errNoSeason
	}
	return id, err
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }
