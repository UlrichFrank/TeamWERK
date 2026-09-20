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
	"errors"
	"fmt"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
)

// ErrStaffelUnbekannt meldet einen Staffelcode, den kein Katalog kennt.
// Er wird sichtbar protokolliert und die Staffel im Lauf übersprungen —
// ein stiller Leerlauf wäre die schlechtere Antwort.
var ErrStaffelUnbekannt = errors.New("Staffelcode in keinem Katalog gefunden")

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

// ResolveStaffel löst einen Staffelcode über den Katalog auf und legt die
// Staffel an bzw. aktualisiert sie.
//
// Gespeichert wird der Code, nicht die Klassen-ID: die wechselt mit der
// Saison. Gesucht wird zuerst auf Verbandsebene, dann in den Bezirken — die
// Bezirksliste kommt vom Dienst selbst, statt sie aus dem Code-Suffix zu
// raten. Bezirke sind dabei nur über o erreichbar, og bleibt der Verband
// (design.md §1.2).
func (s *Store) ResolveStaffel(ctx context.Context, c *bwhv.Client, seasonID, orgID int, code string) (*Staffel, error) {
	periods, selected, err := c.FetchPeriods(ctx, orgID)
	if err != nil {
		return nil, err
	}
	_ = periods
	if selected == "" {
		return nil, fmt.Errorf("Spielzeit des Verbands nicht ermittelbar")
	}

	candidates := []int{0} // 0 = Verbandsebene (kein o-Parameter)
	orgs, err := c.FetchOrgs(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for id := range orgs {
		var n int
		if _, err := fmt.Sscan(id, &n); err != nil || n == orgID || n <= 0 {
			continue
		}
		candidates = append(candidates, n)
	}

	want := strings.EqualFold
	for _, sub := range candidates {
		classes, err := c.FetchCatalog(ctx, orgID, sub, selected)
		if err != nil {
			return nil, err
		}
		for _, cl := range classes {
			if !want(cl.Sname, code) {
				continue
			}
			return s.upsertStaffel(ctx, Staffel{
				SeasonID: seasonID, Code: cl.Sname, Name: cl.Lname,
				OrgID: orgID, SubOrgID: sub, PeriodID: selected,
			})
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrStaffelUnbekannt, code)
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
