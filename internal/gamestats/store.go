// Package gamestats hält Spielpläne, Tabellen, Spielberichte und
// Spielerstatistik des BWHV in der Datenbank und stellt sie über HTTP bereit.
//
// Domänen-Paket: es nutzt internal/bwhv (Foundation) für Abruf und Parsing und
// importiert kein anderes Domänen-Paket. Die Vorbefüllung der Ergebnisfelder
// des redaktionellen Spielberichts läuft deshalb über das Frontend, nicht über
// einen Import aus internal/matchreports (design.md §8).
package gamestats

import (
	"context"
	"database/sql"
	"strings"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
)

// Staffel ist eine gespeicherte Staffel-Zuordnung samt aufgelöster Herkunft.
type Staffel struct {
	ID       int
	SeasonID int
	Code     string
	Name     string
	OrgID    int
	SubOrgID int
	PeriodID string
}

// Store kapselt den Datenbankzugriff dieses Pakets.
type Store struct{ db *sql.DB }

// NewStore liefert einen Store.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// StaffelCodes liest die Staffelcodes aller Mannschafts-Kader einer Saison.
//
// Übungsgruppen (kind='practice') sind ausgeschlossen: sie tragen keine
// Staffel, und die Einschränkung steht hier statt als CHECK in der Migration,
// weil SQLite dafür einen Tabellen-Rebuild bräuchte.
func (s *Store) StaffelCodes(ctx context.Context, seasonID int) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT k.staffel, k.id
		  FROM kader k
		 WHERE k.season_id = ?
		   AND k.kind = 'team'
		   AND k.staffel IS NOT NULL
		   AND TRIM(k.staffel) <> ''`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var code string
		var kaderID int
		if err := rows.Scan(&code, &kaderID); err != nil {
			return nil, err
		}
		out[strings.TrimSpace(code)] = kaderID
	}
	return out, rows.Err()
}

func (s *Store) upsertStaffel(ctx context.Context, st Staffel) (*Staffel, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO bwhv_staffeln (season_id, code, name, org_id, sub_org_id, period_id)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (season_id, code) DO UPDATE SET
			name = excluded.name, org_id = excluded.org_id,
			sub_org_id = excluded.sub_org_id, period_id = excluded.period_id`,
		st.SeasonID, st.Code, st.Name, st.OrgID, nullableInt(st.SubOrgID), st.PeriodID)
	if err != nil {
		return nil, err
	}
	return s.staffelByCode(ctx, st.SeasonID, st.Code)
}

func (s *Store) staffelByCode(ctx context.Context, seasonID int, code string) (*Staffel, error) {
	var st Staffel
	var sub sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, season_id, code, name, org_id, sub_org_id, period_id
		  FROM bwhv_staffeln WHERE season_id = ? AND code = ?`, seasonID, code).
		Scan(&st.ID, &st.SeasonID, &st.Code, &st.Name, &st.OrgID, &sub, &st.PeriodID)
	if err != nil {
		return nil, err
	}
	if sub.Valid {
		st.SubOrgID = int(sub.Int64)
	}
	return &st, nil
}

// ListStaffeln liefert alle gespeicherten Staffeln einer Saison.
func (s *Store) ListStaffeln(ctx context.Context, seasonID int) ([]Staffel, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, season_id, code, name, org_id, sub_org_id, period_id
		  FROM bwhv_staffeln WHERE season_id = ? ORDER BY code`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Staffel
	for rows.Next() {
		var st Staffel
		var sub sql.NullInt64
		if err := rows.Scan(&st.ID, &st.SeasonID, &st.Code, &st.Name, &st.OrgID, &sub, &st.PeriodID); err != nil {
			return nil, err
		}
		if sub.Valid {
			st.SubOrgID = int(sub.Int64)
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func nullableInt(v int) any {
	if v <= 0 {
		return nil
	}
	return v
}

// StaffelByID liefert eine Staffel der angegebenen Saison.
func (s *Store) StaffelByID(ctx context.Context, id, seasonID int) (*Staffel, error) {
	var st Staffel
	var sub sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, season_id, code, name, org_id, sub_org_id, period_id
		  FROM bwhv_staffeln WHERE id = ? AND season_id = ?`, id, seasonID).
		Scan(&st.ID, &st.SeasonID, &st.Code, &st.Name, &st.OrgID, &sub, &st.PeriodID)
	if err != nil {
		return nil, err
	}
	if sub.Valid {
		st.SubOrgID = int(sub.Int64)
	}
	return &st, nil
}

// ListStaffelnWithTeam liefert die Staffel-Zuordnungen der Saison.
//
// Die Abfrage geht FROM kader, nicht FROM bwhv_staffeln: die Zuordnung ist das,
// was der Vorstand pflegt (kader.staffel), der Snapshot in bwhv_staffeln
// entsteht erst beim ersten Abruf. Andersherum — und genau das war der Fehler —
// bleibt eine frisch gepflegte Staffel unsichtbar, bis zufällig ein Poll lief,
// und die Seite behauptet "noch keiner Mannschaft zugeordnet".
//
// Polled=false heißt deshalb "zugeordnet, aber noch nichts abgerufen" und ist
// ein anzeigbarer Zustand, kein Fehler.
func (s *Store) ListStaffelnWithTeam(ctx context.Context, seasonID int) ([]staffelResponse, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(st.id, 0), k.staffel, COALESCE(st.name, ''),
		       COALESCE(t.name, ''), COALESCE(`+appdb.TeamDisplayShort("t")+`, t.name, ''), k.id,
		       CASE WHEN st.polled_at IS NULL THEN 0 ELSE 1 END
		  FROM kader k
		  LEFT JOIN teams t ON t.id = k.team_id
		  LEFT JOIN bwhv_staffeln st ON st.season_id = k.season_id AND st.code = k.staffel
		 WHERE k.season_id = ?
		   AND k.kind = 'team'
		   AND k.staffel IS NOT NULL
		   AND TRIM(k.staffel) <> ''
		 ORDER BY t.name, k.staffel`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []staffelResponse
	for rows.Next() {
		var r staffelResponse
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.TeamName, &r.TeamShort, &r.KaderID, &r.Polled); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
