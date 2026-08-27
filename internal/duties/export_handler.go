package duties

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/settings"
)

// Dienst-CSV-Export (GET /api/duty-slots/export).
//
// Zweck ist die Planungssicht des Dienst-Verantwortlichen: welche Dienste liegen
// im Zeitraum, wann beginnen und enden sie, und in welcher Tageskonstellation
// stehen sie. Bewusst OHNE Belegung/Zuweisungen — kein Name, keine
// slots_filled-Spalte; das Blatt trägt damit keine personenbezogenen Daten und
// darf frei weitergegeben werden (etwa an einen Ausrichter, der nicht im System
// ist).
//
// Der Export BESCHREIBT die Tageskonstellation, er RECHNET sie nicht nach: die
// Entscheidung „mehrere Spiele am Tag → Dienst entfällt/reduziert" trifft
// ausschließlich die Regen-Engine (internal/games/regen.go, applyBehavior). Hier
// stehen die Eingangsgrößen (Spiele am Tag, Anwurfzeiten, Heimspiel am
// Nachbartag) neben der am Diensttyp konfigurierten Regel — kein zweiter,
// driftender Nachbau der Entscheidung. Die Zahlen selbst sind exakt die, die
// loadSameDayContextTx liest (alle Spiele des Tages für die Zeiten, nur
// is_home=1 für Vor-/Folgetag).

// audienceLabels spiegelt AUDIENCE_LABELS aus web/src/lib/constants.ts. Bewusst
// eine zweite, kleine Kopie: die CSV ist die einzige serverseitig gerenderte
// Ausgabe dieser Werte, ein geteiltes Label-Register dafür wäre mehr Apparat als
// Nutzen. Unbekannte Werte fallen auf den Rohwert zurück.
var audienceLabels = map[string]string{
	"spieler":            "Spieler",
	"trainer":            "Trainer",
	"sportliche_leitung": "Sportliche Leitung",
	"vorstand":           "Vorstand",
	"vorstand_beisitzer": "Vorstands-Beisitzer",
	"kassierer":          "Kassierer",
	"medien":             "Medien",
	"eltern":             "Eltern",
}

var weekdayShort = map[time.Weekday]string{
	time.Monday: "Mo", time.Tuesday: "Di", time.Wednesday: "Mi", time.Thursday: "Do",
	time.Friday: "Fr", time.Saturday: "Sa", time.Sunday: "So",
}

var eventTypeLabels = map[string]string{
	"heim":      "Heimspiel",
	"auswärts":  "Auswärtsspiel",
	"generisch": "Sonstiges",
}

var exportHeader = []string{
	"Datum", "Wochentag",
	"Ausrichter", "Ausrichter für Tag gesetzt",
	"Spiele am Tag", "Anwurfzeiten am Tag", "Heimspiel Vortag", "Heimspiel Folgetag",
	"Termin", "Termin-Art", "Gegner", "Mannschaften", "Halle", "Anwurf", "Termin-Ende",
	"Dienst", "Beschreibung", "Dienst-Beginn", "Dienst-Ende", "Dauer (Std.)", "Plätze",
	"Herkunft", "Dienst-Vorlage", "Zielgruppe",
	"Regel bei mehreren Spielen am Tag", "Regel bei Spiel am Nachbartag",
}

// dayContext sind die tagesweiten Angaben — je (Datum, Saison) einmal geladen
// und dann über alle Dienste dieses Tages wiederverwendet.
type dayContext struct {
	ausrichter string
	explicit   bool
	gameCount  int
	times      []string
	prevHome   bool
	nextHome   bool
}

// GET /api/duty-slots/export?from=YYYY-MM-DD&to=YYYY-MM-DD
//
// Beide Grenzen sind Pflicht: ein Export ohne Zeitraum wäre ein Voll-Dump der
// Tabelle und in der UI nie gewollt.
func (h *Handler) ExportSlots(w http.ResponseWriter, r *http.Request) {
	from, okFrom := parseExportDate(r.URL.Query().Get("from"))
	to, okTo := parseExportDate(r.URL.Query().Get("to"))
	if !okFrom || !okTo {
		http.Error(w, `{"error":"invalid_range"}`, http.StatusBadRequest)
		return
	}
	if from > to {
		http.Error(w, `{"error":"invalid_range"}`, http.StatusBadRequest)
		return
	}

	ausrichterNames, err := h.loadAusrichterNames(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// substr(...) auf beiden Seiten des Vergleichs: event_date ist DATE, kann
	// aber als ISO-Timestamp in der Zeile stehen (SQLite-DATE-Gotcha) — dann
	// würde ein nacktes BETWEEN den Tag am Bereichsende verlieren.
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT substr(ds.event_date, 1, 10),
		       ds.season_id,
		       COALESCE(ds.event_time, ''),
		       ds.hours_value,
		       ds.slots_total,
		       ds.is_custom,
		       COALESCE(ds.event_name, ''),
		       dt.name,
		       COALESCE(ds.role_desc, ''),
		       COALESCE(ds.audiences, dt.audiences),
		       dt.same_day_behavior, COALESCE(sdv.name, ''),
		       dt.adjacent_day_behavior, COALESCE(adv.name, ''),
		       COALESCE(g.event_type, ''),
		       COALESCE(g.opponent, ''),
		       COALESCE(g.time, ''),
		       COALESCE(g.end_time, ''),
		       COALESCE(v.name, ''),
		       COALESCE(gtpl.name, ''),
		       COALESCE(
		         (SELECT group_concat(n, ', ') FROM (
		            SELECT tt.name AS n FROM game_teams gt
		            JOIN teams tt ON tt.id = gt.team_id
		            WHERE gt.game_id = ds.game_id ORDER BY tt.name)),
		         COALESCE(t.name, ''))
		 FROM duty_slots ds
		 JOIN duty_types dt ON dt.id = ds.duty_type_id
		 LEFT JOIN duty_types sdv ON sdv.id = dt.same_day_variant_id
		 LEFT JOIN duty_types adv ON adv.id = dt.adjacent_day_variant_id
		 LEFT JOIN games g ON g.id = ds.game_id
		 LEFT JOIN venues v ON v.id = g.venue_id
		 LEFT JOIN game_templates gtpl ON gtpl.id = g.template_id
		 LEFT JOIN teams t ON t.id = ds.team_id
		 WHERE substr(ds.event_date, 1, 10) BETWEEN ? AND ?
		 ORDER BY substr(ds.event_date, 1, 10), COALESCE(ds.event_time, ''), ds.id`,
		from, to)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="dienste_%s_%s.csv"`, from, to))
	// UTF-8-BOM: ohne ihn zeigt Excel (Windows/macOS) die Umlaute in
	// „Auswärtsspiel"/„Plätze" als Mojibake an — der Content-Type-Charset
	// erreicht den Datei-Import nicht.
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cw.Write(exportHeader)

	dayCache := map[string]dayContext{}
	for rows.Next() {
		var date, eventTime, eventName, dutyType, roleDesc string
		var sameDay, sameDayVariant, adjDay, adjDayVariant string
		var eventType, opponent, gameTime, gameEnd, venueName, templateName, teamNames string
		var audiences sql.NullString
		var seasonID, slotsTotal, isCustom int
		var hours float64
		if err := rows.Scan(&date, &seasonID, &eventTime, &hours, &slotsTotal, &isCustom,
			&eventName, &dutyType, &roleDesc, &audiences,
			&sameDay, &sameDayVariant, &adjDay, &adjDayVariant,
			&eventType, &opponent, &gameTime, &gameEnd, &venueName, &templateName, &teamNames); err != nil {
			// Mitten im Stream ist kein sauberer HTTP-Status mehr möglich; die
			// Zeile wird übersprungen, statt die Datei zu zerreißen.
			continue
		}

		key := date + "|" + strconv.Itoa(seasonID)
		ctxDay, ok := dayCache[key]
		if !ok {
			ctxDay = h.loadDayContext(r.Context(), date, seasonID, ausrichterNames)
			dayCache[key] = ctxDay
		}

		cw.Write([]string{
			formatGermanDate(date),
			weekdayOf(date),
			ctxDay.ausrichter,
			jaNein(ctxDay.explicit),
			strconv.Itoa(ctxDay.gameCount),
			strings.Join(ctxDay.times, ", "),
			jaNein(ctxDay.prevHome),
			jaNein(ctxDay.nextHome),
			eventName,
			eventTypeLabel(eventType),
			opponent,
			teamNames,
			venueName,
			gameTime,
			gameEnd,
			dutyType,
			roleDesc,
			eventTime,
			endTime(eventTime, hours),
			formatHours(hours),
			strconv.Itoa(slotsTotal),
			herkunft(isCustom),
			templateName,
			formatAudiences(audiences),
			behaviorLabel(sameDay, sameDayVariant),
			behaviorLabel(adjDay, adjDayVariant),
		})
	}
	cw.Flush()
}

// loadAusrichterNames liest die Ausrichter-Namen (inkl. deaktivierter — ein
// Spieltag kann noch auf einem inaktiven Eintrag stehen) einmal pro Export.
func (h *Handler) loadAusrichterNames(ctx context.Context) (map[int]string, error) {
	list, err := settings.ListAusrichter(ctx, h.db, true)
	if err != nil {
		return nil, err
	}
	names := make(map[int]string, len(list))
	for _, a := range list {
		names[a.ID] = a.Name
	}
	return names, nil
}

// loadDayContext sammelt die tagesweiten Angaben. Fehler sind hier bewusst
// nicht fatal: ein Report soll eine leere Zelle zeigen und nicht als Ganzes
// scheitern, weil ein einzelner Tag nicht auflösbar war.
func (h *Handler) loadDayContext(ctx context.Context, date string, seasonID int, names map[int]string) dayContext {
	var d dayContext

	if id, explicit, err := settings.ResolveAusrichterForDayDetailed(ctx, h.db, date, seasonID); err == nil {
		d.ausrichter = names[id]
		d.explicit = explicit
	} else if !errors.Is(err, settings.ErrNoDefaultAusrichter) {
		// Ein echter DB-Fehler bleibt genauso still wie der fehlende Default —
		// beides endet in einer leeren Spalte, siehe Kommentar oben.
		d.ausrichter = ""
	}

	// Dieselben Größen wie loadSameDayContextTx: die Zeiten kommen aus ALLEN
	// Spielen des Tages, Vor-/Folgetag zählen nur Heimspiele.
	var prev, next int
	h.db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM games
		         WHERE substr(date,1,10) = ? AND season_id = ?),
		       (SELECT COUNT(*) FROM games
		         WHERE substr(date,1,10) = date(?, '-1 days') AND is_home = 1 AND season_id = ?),
		       (SELECT COUNT(*) FROM games
		         WHERE substr(date,1,10) = date(?, '+1 days') AND is_home = 1 AND season_id = ?)`,
		date, seasonID, date, seasonID, date, seasonID,
	).Scan(&d.gameCount, &prev, &next)
	d.prevHome = prev > 0
	d.nextHome = next > 0

	rows, err := h.db.QueryContext(ctx,
		`SELECT DISTINCT time FROM games
		  WHERE substr(date,1,10) = ? AND season_id = ? ORDER BY time`, date, seasonID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			if rows.Scan(&t) == nil {
				d.times = append(d.times, t)
			}
		}
	}
	return d
}

// parseExportDate akzeptiert nur die reine "2006-01-02"-Form — sie ist das, was
// die Datumsfelder des Frontends liefern, und hält den Vergleich in SQL exakt.
func parseExportDate(s string) (string, bool) {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return "", false
	}
	return s, true
}

func formatGermanDate(iso string) string {
	if len(iso) < 10 {
		return iso
	}
	return iso[8:10] + "." + iso[5:7] + "." + iso[0:4]
}

func weekdayOf(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return ""
	}
	return weekdayShort[t.Weekday()]
}

func jaNein(b bool) string {
	if b {
		return "ja"
	}
	return "nein"
}

func eventTypeLabel(eventType string) string {
	if eventType == "" {
		return "ohne Termin"
	}
	if l, ok := eventTypeLabels[eventType]; ok {
		return l
	}
	return eventType
}

func herkunft(isCustom int) string {
	if isCustom != 0 {
		return "manuell"
	}
	return "Vorlage"
}

// endTime addiert die Dienst-Dauer (Stunden, REAL) auf die Startzeit. Ohne
// Startzeit gibt es keine Endzeit — ein Slot ohne event_time ist zulässig
// (ganztägiger Dienst).
func endTime(start string, hours float64) string {
	if start == "" || hours <= 0 {
		return ""
	}
	return addMinutesHHMM(start, int(math.Round(hours*60)))
}

// addMinutesHHMM ist die duties-lokale Kopie von games.addMinutes. Domain-Pakete
// dürfen sich nicht gegenseitig importieren (internal/arch), und ein
// Foundation-Paket für neun Zeilen Zeitarithmetik wäre unverhältnismäßig.
func addMinutesHHMM(t string, offset int) string {
	if len(t) < 5 {
		return t
	}
	hh, err1 := strconv.Atoi(t[:2])
	mm, err2 := strconv.Atoi(t[3:5])
	if err1 != nil || err2 != nil {
		return t
	}
	total := ((hh*60+mm+offset)%1440 + 1440) % 1440
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

// formatHours schreibt die Dauer mit Dezimalkomma — die Datei nutzt ';' als
// Trenner (deutsche Excel-Konvention), ein Dezimalpunkt käme dort als Text an.
func formatHours(h float64) string {
	return strings.Replace(fmt.Sprintf("%.2f", h), ".", ",", 1)
}

func formatAudiences(ns sql.NullString) string {
	values := audiencesFromDB(ns)
	if len(values) == 0 {
		return "alle"
	}
	labels := make([]string, 0, len(values))
	for _, v := range values {
		if l, ok := audienceLabels[v]; ok {
			labels = append(labels, l)
		} else {
			labels = append(labels, v)
		}
	}
	return strings.Join(labels, ", ")
}

// behaviorLabel beschreibt die am Diensttyp konfigurierte Regel, nicht ihr
// Ergebnis für diesen Slot. Ob sie greift, hängt zusätzlich von der Lage des
// Dienstes zwischen den Anwurfzeiten ab (classifySlotPosition) — das entscheidet
// die Engine, und der Export gibt bewusst nur die Regel und die Tagesdaten aus,
// aus denen der Leser es ableiten kann.
func behaviorLabel(behavior, variant string) string {
	switch behavior {
	case "skip":
		return "entfällt"
	case "reduced":
		if variant != "" {
			return "reduziert → " + variant
		}
		return "reduziert"
	default:
		return "normal"
	}
}
