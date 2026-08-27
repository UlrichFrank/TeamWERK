package duties

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/settings"
)

// dayContext bündelt die Tageskonstellation eines Spieltags: wie viele Spiele
// insgesamt an diesem Tag liegen (same_day_behavior der Diensttypen reagiert
// darauf) und ob am Vor-/Folgetag ein Heimspiel liegt (adjacent_day_behavior).
// Spiegelt exakt loadSameDayContextTx (internal/games/regen.go), aber
// eigenständig implementiert — duties darf games (Domain) nicht importieren.
type dayContext struct {
	gamesSameDay    int
	homeGamePrevDay bool
	homeGameNextDay bool
}

// GET /api/duty-slots/export
// Exportiert Dienst-Slots im gewählten Datumsbereich als CSV inklusive
// Start-/Endzeit (aus event_time + hours_value berechnet). Für spielgebundene
// Slots kommen zusätzlich der Ausrichter des Spieltages (settings.ResolveAusrichterForDay,
// nur bei Heimspielen sinnvoll) sowie die Tageskonstellation dazu, die die
// same_day_behavior/adjacent_day_behavior-Logik der Diensttypen steuert
// (docs/agent/06-gotchas.md „Auto-Duty-Regen"). Ausrichter/Konstellation sind
// je (Datum, Saison) identisch für alle Slots desselben Spieltags — gecacht,
// damit ein Saisonexport nicht pro Zeile denselben Spieltag neu auflöst.
func (h *Handler) ExportSlots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	where := ` WHERE 1=1`
	var args []any
	if dateFrom := q.Get("date_from"); dateFrom != "" {
		where += ` AND ds.event_date >= ?`
		args = append(args, dateFrom)
	}
	if dateTo := q.Get("date_to"); dateTo != "" {
		where += ` AND ds.event_date <= ?`
		args = append(args, dateTo)
	}

	rows, err := h.db.QueryContext(ctx,
		`SELECT ds.event_name, ds.event_date, COALESCE(ds.event_time,''), ds.hours_value,
		        dt.name, COALESCE(ds.role_desc,''), ds.slots_total, ds.slots_filled, ds.season_id,
		        ds.game_id, COALESCE(g.opponent,''), COALESCE(g.is_home,0), COALESCE(g.event_type,''), COALESCE(g.date,''),
		        COALESCE(t.name,''),
		        COALESCE((SELECT GROUP_CONCAT(tm.name, ', ') FROM game_teams gt JOIN teams tm ON tm.id=gt.team_id WHERE gt.game_id = ds.game_id), ''),
		        dt.same_day_behavior, dt.adjacent_day_behavior
		 FROM duty_slots ds
		 JOIN duty_types dt ON dt.id = ds.duty_type_id
		 LEFT JOIN games g ON g.id = ds.game_id
		 LEFT JOIN teams t ON t.id = ds.team_id`+where+`
		 ORDER BY ds.event_date, ds.event_time, ds.id`,
		args...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type slotRow struct {
		eventName, eventDate, eventTime      string
		hoursValue                           float64
		dutyType, roleDesc                   string
		slotsTotal, slotsFilled              int
		seasonID                             int
		gameID                               sql.NullInt64
		opponent                             string
		isHome                               bool
		eventType                            string
		gameDate                             string
		teamName, gameTeamNames              string
		sameDayBehavior, adjacentDayBehavior string
	}
	var slotRows []slotRow
	for rows.Next() {
		var s slotRow
		var isHomeInt int
		if err := rows.Scan(&s.eventName, &s.eventDate, &s.eventTime, &s.hoursValue,
			&s.dutyType, &s.roleDesc, &s.slotsTotal, &s.slotsFilled, &s.seasonID,
			&s.gameID, &s.opponent, &isHomeInt, &s.eventType, &s.gameDate,
			&s.teamName, &s.gameTeamNames, &s.sameDayBehavior, &s.adjacentDayBehavior); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		s.isHome = isHomeInt != 0
		slotRows = append(slotRows, s)
	}

	type dayKey struct {
		date     string
		seasonID int
	}
	ausrichterCache := map[dayKey]string{}
	contextCache := map[dayKey]dayContext{}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="dienste.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{
		"Datum", "Start", "Ende", "Diensttyp", "Rolle", "Team", "Termin", "Gegner", "Terminart",
		"Plätze besetzt", "Plätze gesamt", "Ausrichter", "Spiele am Tag",
		"Heimspiel Vortag", "Heimspiel Folgetag",
		"Verhalten Mehrfachspieltag", "Verhalten Nachbartag",
	})

	for _, s := range slotRows {
		team := s.gameTeamNames
		if team == "" {
			team = s.teamName
		}

		var ausrichter, spieleAmTag, heimVortag, heimFolgetag string
		if s.gameID.Valid {
			key := dayKey{date: s.gameDate, seasonID: s.seasonID}
			if s.isHome {
				name, cached := ausrichterCache[key]
				if !cached {
					name = h.resolveAusrichterName(ctx, s.gameDate, s.seasonID)
					ausrichterCache[key] = name
				}
				ausrichter = name
			}
			dc, cached := contextCache[key]
			if !cached {
				dc = h.loadDayContext(ctx, s.gameDate, s.seasonID)
				contextCache[key] = dc
			}
			spieleAmTag = fmt.Sprintf("%d", dc.gamesSameDay)
			heimVortag = yesNo(dc.homeGamePrevDay)
			heimFolgetag = yesNo(dc.homeGameNextDay)
		}

		cw.Write([]string{
			displayDate(s.eventDate), s.eventTime, endTime(s.eventTime, s.hoursValue),
			s.dutyType, s.roleDesc, team, s.eventName, s.opponent, terminartLabel(s.eventType),
			fmt.Sprintf("%d", s.slotsFilled), fmt.Sprintf("%d", s.slotsTotal),
			ausrichter, spieleAmTag, heimVortag, heimFolgetag,
			behaviorLabel(s.sameDayBehavior), behaviorLabel(s.adjacentDayBehavior),
		})
	}
	cw.Flush()
}

// resolveAusrichterName liefert den Namen des für diesen Spieltag geltenden
// Ausrichters (leerer String bei Auflösungsfehler, z. B. fehlende
// Default-Zeile — der Export bricht dafür nicht ab).
func (h *Handler) resolveAusrichterName(ctx context.Context, date string, seasonID int) string {
	id, _, err := settings.ResolveAusrichterForDayDetailed(ctx, h.db, date, seasonID)
	if err != nil {
		return ""
	}
	a, err := settings.GetAusrichter(ctx, h.db, id)
	if err != nil {
		return ""
	}
	return a.Name
}

// loadDayContext zählt Spiele am selben Tag und prüft Heimspiele am
// Vor-/Folgetag — dieselbe Konstellation, gegen die same_day_behavior/
// adjacent_day_behavior beim Regen greifen. date() normalisiert beide Seiten
// des Vergleichs (SQLite-DATE-Gotcha, docs/agent/06-gotchas.md): games.date
// kann je nach Lesepfad mit oder ohne Zeitanteil vorliegen, date() versteht
// beide Formen gleich.
func (h *Handler) loadDayContext(ctx context.Context, date string, seasonID int) dayContext {
	var dc dayContext
	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM games WHERE date(date)=date(?) AND season_id=?`,
		date, seasonID).Scan(&dc.gamesSameDay)
	var prev, next int
	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM games WHERE date(date)=date(?, '-1 days') AND is_home=1 AND season_id=?`,
		date, seasonID).Scan(&prev)
	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM games WHERE date(date)=date(?, '+1 days') AND is_home=1 AND season_id=?`,
		date, seasonID).Scan(&next)
	dc.homeGamePrevDay = prev > 0
	dc.homeGameNextDay = next > 0
	return dc
}

// displayDate normalisiert ein gescanntes DATE-Feld auf die reine
// "2006-01-02"-Form (SQLite-DATE-Gotcha, docs/agent/06-gotchas.md).
func displayDate(s string) string {
	if len(s) > 10 {
		return s[:10]
	}
	return s
}

func endTime(start string, hoursValue float64) string {
	if start == "" {
		return ""
	}
	t, err := time.Parse("15:04", start)
	if err != nil {
		return ""
	}
	return t.Add(time.Duration(hoursValue * float64(time.Hour))).Format("15:04")
}

func yesNo(b bool) string {
	if b {
		return "Ja"
	}
	return "Nein"
}

func terminartLabel(eventType string) string {
	switch eventType {
	case "heim":
		return "Heimspiel"
	case "auswärts":
		return "Auswärtsspiel"
	case "generisch":
		return "Termin"
	default:
		return ""
	}
}

func behaviorLabel(behavior string) string {
	switch behavior {
	case "skip":
		return "entfällt"
	case "reduced":
		return "reduziert"
	default:
		return "normal"
	}
}
