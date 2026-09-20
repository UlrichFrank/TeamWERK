package gamestats

import (
	"context"
	"database/sql"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// PollDelay ist der Abstand zwischen frühestem Anwurf eines Spieltags und dem
// ersten Abrufversuch. Zwei Stunden decken Spiel plus Halbzeit plus die Zeit
// ab, die der Zeitnehmer für die Freigabe braucht.
const PollDelay = 2 * time.Hour

// PollInterval ist die Kadenz innerhalb des Fensters.
const PollInterval = 15 * time.Minute

// CatchUpDays ist das Fenster, in dem Nachzügler früherer Tage nachgeholt
// werden. Ein Bericht, der am Spieltag nicht freigegeben wurde, soll nicht
// endgültig verloren sein.
const CatchUpDays = 3

// DueStaffel ist eine zum Abruf fällige Staffel samt Begründung.
type DueStaffel struct {
	Staffel Staffel
	Open    int    // Begegnungen ohne Bericht im betrachteten Zeitraum
	Reason  string // "spieltag" | "nachzug"
}

// DueStaffeln liefert die Staffeln, die jetzt abgerufen werden sollen.
//
// Eine Staffel ist fällig, wenn sie heute mindestens eine Begegnung hat, der
// früheste Anwurf des Tages mindestens PollDelay zurückliegt und mindestens
// eine Begegnung des Tages noch kein sGID trägt.
//
// Der Zustand wird vollständig aus bwhv_games abgeleitet — es gibt bewusst
// KEIN Poll-Zustandsfeld, das von der Wirklichkeit abweichen könnte
// (design.md §4). polled_at begrenzt nur die Kadenz.
func (s *Store) DueStaffeln(ctx context.Context, seasonID int, now time.Time) ([]DueStaffel, error) {
	staffeln, err := s.ListStaffeln(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	local := now.In(timez.Berlin())
	today := local.Format("2006-01-02")

	var out []DueStaffel
	for _, st := range staffeln {
		due, err := s.dueFor(ctx, st, local, today)
		if err != nil {
			return nil, err
		}
		if due != nil {
			out = append(out, *due)
		}
	}
	return out, nil
}

func (s *Store) dueFor(ctx context.Context, st Staffel, local time.Time, today string) (*DueStaffel, error) {
	if recent, err := s.polledWithin(ctx, st.ID, local, PollInterval); err != nil || recent {
		return nil, err
	}

	// COALESCE um das SUM: über null Zeilen liefert SUM NULL, nicht 0 — ein
	// Tag ohne Begegnung ließe den Scan sonst mit einem Konvertierungsfehler
	// auflaufen statt die Staffel schlicht zu überspringen.
	var earliest sql.NullString
	var open int
	err := s.db.QueryRowContext(ctx, `
		SELECT MIN(NULLIF(TRIM(time), '')),
		       COALESCE(SUM(CASE WHEN TRIM(sgid) = '' THEN 1 ELSE 0 END), 0)
		  FROM bwhv_games WHERE staffel_id = ? AND date = ?`, st.ID, today).Scan(&earliest, &open)
	if err != nil {
		return nil, err
	}
	if open > 0 && earliest.Valid {
		anwurf := timez.ParseDT(today, earliest.String, timez.Berlin())
		if !local.Before(anwurf.Add(PollDelay)) {
			return &DueStaffel{Staffel: st, Open: open, Reason: "spieltag"}, nil
		}
		return nil, nil
	}

	// Nachzug: offene Begegnungen der Vortage innerhalb des Nachholfensters.
	from := local.AddDate(0, 0, -CatchUpDays).Format("2006-01-02")
	var stale int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM bwhv_games
		 WHERE staffel_id = ? AND date >= ? AND date < ? AND TRIM(sgid) = ''`,
		st.ID, from, today).Scan(&stale); err != nil {
		return nil, err
	}
	if stale > 0 {
		return &DueStaffel{Staffel: st, Open: stale, Reason: "nachzug"}, nil
	}
	return nil, nil
}

// polledWithin meldet, ob die Staffel innerhalb der Kadenz schon abgerufen wurde.
func (s *Store) polledWithin(ctx context.Context, staffelID int, now time.Time, d time.Duration) (bool, error) {
	var raw sql.NullString
	if err := s.db.QueryRowContext(ctx,
		`SELECT polled_at FROM bwhv_staffeln WHERE id = ?`, staffelID).Scan(&raw); err != nil {
		return false, err
	}
	if !raw.Valid || raw.String == "" {
		return false, nil
	}
	t, err := time.Parse(time.RFC3339, raw.String)
	if err != nil {
		return false, nil
	}
	return now.Sub(t) < d, nil
}
