package calendar

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

type tokenSettings struct {
	Token            string `json:"token"`
	IncludeHeim      bool   `json:"include_heim"`
	IncludeAuswaerts bool   `json:"include_auswaerts"`
	IncludeTraining  bool   `json:"include_training"`
	IncludeGenerisch bool   `json:"include_generisch"`
	IncludeDuty      bool   `json:"include_duty"`
	// IncludePracticeGroups steuert Übungsgruppen-Termine (kader.kind='practice',
	// erkennbar an training_sessions.team_id IS NULL) und ist von
	// IncludeTraining unabhängig — das Mannschaftstraining ist der Pflichttermin
	// der eigenen Mannschaft, die Übungsgruppe ein Zusatzangebot mit eigenem
	// Rhythmus.
	IncludePracticeGroups bool `json:"include_practice_groups"`
}

// GET /api/calendar/token
func (h *Handler) GetToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var s tokenSettings
	err := h.db.QueryRowContext(r.Context(),
		`SELECT token, include_heim, include_auswaerts, include_training, include_generisch, include_duty, include_practice_groups
		 FROM calendar_tokens WHERE user_id = ?`, claims.UserID).
		Scan(&s.Token, &s.IncludeHeim, &s.IncludeAuswaerts, &s.IncludeTraining, &s.IncludeGenerisch, &s.IncludeDuty, &s.IncludePracticeGroups)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// POST /api/calendar/token
func (h *Handler) UpsertToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req tokenSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Check if token exists for user.
	var existing string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT token FROM calendar_tokens WHERE user_id = ?`, claims.UserID).Scan(&existing)

	if err == sql.ErrNoRows {
		// New token: generate UUID v4.
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		b[6] = (b[6] & 0x0f) | 0x40 // version 4
		b[8] = (b[8] & 0x3f) | 0x80 // variant bits
		token := hex.EncodeToString(b[:4]) + "-" +
			hex.EncodeToString(b[4:6]) + "-" +
			hex.EncodeToString(b[6:8]) + "-" +
			hex.EncodeToString(b[8:10]) + "-" +
			hex.EncodeToString(b[10:])
		_, err = h.db.ExecContext(r.Context(),
			`INSERT INTO calendar_tokens (user_id, token, include_heim, include_auswaerts, include_training, include_generisch, include_duty, include_practice_groups)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			claims.UserID, token,
			boolToInt(req.IncludeHeim), boolToInt(req.IncludeAuswaerts),
			boolToInt(req.IncludeTraining), boolToInt(req.IncludeGenerisch),
			boolToInt(req.IncludeDuty), boolToInt(req.IncludePracticeGroups))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		req.Token = token
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else {
		// Update settings, keep existing token.
		_, err = h.db.ExecContext(r.Context(),
			`UPDATE calendar_tokens SET include_heim=?, include_auswaerts=?, include_training=?, include_generisch=?, include_duty=?, include_practice_groups=?
			 WHERE user_id=?`,
			boolToInt(req.IncludeHeim), boolToInt(req.IncludeAuswaerts),
			boolToInt(req.IncludeTraining), boolToInt(req.IncludeGenerisch),
			boolToInt(req.IncludeDuty), boolToInt(req.IncludePracticeGroups), claims.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		req.Token = existing
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

// DELETE /api/calendar/token
func (h *Handler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	h.db.ExecContext(r.Context(), `DELETE FROM calendar_tokens WHERE user_id = ?`, claims.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// childUserID liefert die user_id des Kindes, wenn der anfragende Nutzer
// Elternteil dieses Mitglieds ist (via family_links). 0 = kein Zugriff.
func (h *Handler) childUserID(r *http.Request, claims *auth.Claims) (int, bool) {
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		return 0, false
	}
	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id = ? AND member_id = ?`,
		claims.UserID, memberID).Scan(&count)
	if count == 0 {
		return 0, false
	}
	var userID int
	err = h.db.QueryRowContext(r.Context(),
		`SELECT user_id FROM members WHERE id = ? AND user_id IS NOT NULL`, memberID).Scan(&userID)
	if err != nil {
		return 0, false
	}
	return userID, true
}

// GET /api/profile/kind/{memberId}/calendar-token
func (h *Handler) GetChildToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	childUID, ok := h.childUserID(r, claims)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var s tokenSettings
	err := h.db.QueryRowContext(r.Context(),
		`SELECT token, include_heim, include_auswaerts, include_training, include_generisch, include_duty, include_practice_groups
		 FROM calendar_tokens WHERE user_id = ?`, childUID).
		Scan(&s.Token, &s.IncludeHeim, &s.IncludeAuswaerts, &s.IncludeTraining, &s.IncludeGenerisch, &s.IncludeDuty, &s.IncludePracticeGroups)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// POST /api/profile/kind/{memberId}/calendar-token
func (h *Handler) UpsertChildToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	childUID, ok := h.childUserID(r, claims)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req tokenSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var existing string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT token FROM calendar_tokens WHERE user_id = ?`, childUID).Scan(&existing)
	if err == sql.ErrNoRows {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		token := hex.EncodeToString(b[:4]) + "-" +
			hex.EncodeToString(b[4:6]) + "-" +
			hex.EncodeToString(b[6:8]) + "-" +
			hex.EncodeToString(b[8:10]) + "-" +
			hex.EncodeToString(b[10:])
		_, err = h.db.ExecContext(r.Context(),
			`INSERT INTO calendar_tokens (user_id, token, include_heim, include_auswaerts, include_training, include_generisch, include_duty, include_practice_groups)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			childUID, token,
			boolToInt(req.IncludeHeim), boolToInt(req.IncludeAuswaerts),
			boolToInt(req.IncludeTraining), boolToInt(req.IncludeGenerisch),
			boolToInt(req.IncludeDuty), boolToInt(req.IncludePracticeGroups))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		req.Token = token
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else {
		_, err = h.db.ExecContext(r.Context(),
			`UPDATE calendar_tokens SET include_heim=?, include_auswaerts=?, include_training=?, include_generisch=?, include_duty=?, include_practice_groups=?
			 WHERE user_id=?`,
			boolToInt(req.IncludeHeim), boolToInt(req.IncludeAuswaerts),
			boolToInt(req.IncludeTraining), boolToInt(req.IncludeGenerisch),
			boolToInt(req.IncludeDuty), boolToInt(req.IncludePracticeGroups), childUID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		req.Token = existing
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

// DELETE /api/profile/kind/{memberId}/calendar-token
func (h *Handler) DeleteChildToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	childUID, ok := h.childUserID(r, claims)
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h.db.ExecContext(r.Context(), `DELETE FROM calendar_tokens WHERE user_id = ?`, childUID)
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/calendar/feed/{token}.ics
func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	// Strip .ics suffix if present (Chi may include it in the path value).
	token = strings.TrimSuffix(token, ".ics")

	var s tokenSettings
	var userID int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT user_id, token, include_heim, include_auswaerts, include_training, include_generisch, include_duty, include_practice_groups
		 FROM calendar_tokens WHERE token = ?`, token).
		Scan(&userID, &s.Token, &s.IncludeHeim, &s.IncludeAuswaerts, &s.IncludeTraining, &s.IncludeGenerisch, &s.IncludeDuty, &s.IncludePracticeGroups)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var events []calEvent

	// Games (heim/auswärts/generisch).
	gameTypes := buildGameTypeFilter(s)
	if len(gameTypes) > 0 {
		games, err := h.fetchGames(r, userID, gameTypes)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		events = append(events, games...)
	}

	// Trainings (training_sessions table — separate from games). Mannschafts-
	// und Übungsgruppentermine liegen in derselben Tabelle und werden von zwei
	// unabhängigen Toggles gesteuert; ist keiner gesetzt, entfällt die Abfrage.
	if s.IncludeTraining || s.IncludePracticeGroups {
		trainings, err := h.fetchTrainings(r, userID, s.IncludeTraining, s.IncludePracticeGroups)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		events = append(events, trainings...)
	}

	// Duties.
	if s.IncludeDuty {
		duties, err := h.fetchDuties(r, userID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		events = append(events, duties...)
	}

	// Append the person's first name to the calendar name so families can
	// subscribe to several members' feeds and tell them apart.
	calName := "TeamWERK"
	var firstName string
	h.db.QueryRowContext(r.Context(),
		`SELECT first_name FROM users WHERE id = ?`, userID).Scan(&firstName)
	if firstName != "" {
		calName = "TeamWERK – " + firstName
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"teamwerk.ics\"")
	fmt.Fprint(w, renderICal(events, calName))
}

// ── Queries ──────────────────────────────────────────────────────────────────

type calEvent struct {
	UID         string
	Summary     string
	Location    string
	Description string
	Start       time.Time
	End         time.Time
	HasEnd      bool
	// Stamp wird zum DTSTAMP. Er stammt aus created_at des Termins, damit
	// derselbe Datenstand bei jedem Abruf denselben Body liefert.
	Stamp time.Time
	// AllDay schreibt DTSTART/DTEND als VALUE=DATE statt mit TZID — für
	// Dienste ohne event_time, die sonst als Termin um Mitternacht erschienen.
	AllDay bool
}

// venueLocation bildet die LOCATION-Zeile „Name, Straße, PLZ Ort". Ohne Namen
// bleibt sie leer — eine Adresse ohne Hallennamen wäre nicht wiedererkennbar.
func venueLocation(name, street, postal, city string) string {
	if name == "" {
		return ""
	}
	parts := []string{name}
	if street != "" {
		parts = append(parts, street)
	}
	if postal != "" || city != "" {
		parts = append(parts, strings.TrimSpace(postal+" "+city))
	}
	return strings.Join(parts, ", ")
}

// parseStamp liest ein created_at als UTC. Die Spalte trägt je nach Treiberweg
// RFC3339 oder die zonenlose SQLite-Form „2006-01-02 15:04:05"; letztere ist
// UTC ohne Kennung und darf nicht als Ortszeit gelesen werden (Gotcha
// „SQLite DATETIME-Felder"). Nicht parsebar → jetzt, ein VEVENT ohne DTSTAMP
// darf nicht entstehen.
func parseStamp(s string) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC()
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}

// kaderMembership vereint die drei Arten, auf die ein Nutzer an einem Kader
// hängt — dieselbe Menge, die auth.GameVisibilityClause für den Spielplan
// auflöst (dort zusätzlich über family_links für Eltern; der Feed braucht das
// nicht, weil Eltern einen eigenen Kind-Token bekommen). Der
// Funktionsträger-Bypass aus auth wird bewusst NICHT übernommen: ein Vorstand
// will nicht alle Spiele des Vereins im Kalender haben.
//
// is_extended=1 nur beim erweiterten Kader — es steuert allein die
// Kennzeichnung im Titel. Trainer zählen wie reguläre Mitglieder.
const kaderMembership = `
	SELECT kader_id, member_id, 0 AS is_extended FROM kader_members
	UNION ALL
	SELECT kader_id, member_id, 0 FROM kader_trainers
	UNION ALL
	SELECT kader_id, member_id, 1 FROM kader_extended_members`

func (h *Handler) fetchGames(r *http.Request, userID int, eventTypes []string) ([]calEvent, error) {
	placeholders := strings.Repeat("?,", len(eventTypes))
	placeholders = placeholders[:len(placeholders)-1]
	args := []any{userID}
	for _, t := range eventTypes {
		args = append(args, t)
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT DISTINCT
		    g.id, g.date, g.time, g.end_time, g.end_date,
		    g.opponent, g.event_type, g.is_home, g.note,
		    COALESCE(v.name,''), COALESCE(v.street,''), COALESCE(v.postal_code,''), COALESCE(v.city,''),
		    t.name, mem.is_extended, g.created_at,
		    EXISTS(SELECT 1 FROM game_lineup gl WHERE gl.game_id = g.id) AS lineup_exists,
		    EXISTS(SELECT 1 FROM game_lineup gl2
		           WHERE gl2.game_id = g.id AND gl2.member_id = mem.member_id) AS in_lineup
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id
		JOIN teams t ON t.id = gt.team_id
		JOIN kader k ON k.team_id = gt.team_id AND k.season_id = g.season_id
		JOIN (`+kaderMembership+`) mem ON mem.kader_id = k.id
		JOIN members m ON m.id = mem.member_id
		LEFT JOIN venues v ON v.id = g.venue_id
		WHERE m.user_id = ? AND g.event_type IN (`+placeholders+`)
		ORDER BY g.date, g.time, mem.is_extended`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Europe/Berlin")
	var events []calEvent
	// Der Team-Join kann dasselbe Spiel mehrfach liefern, wenn der Nutzer über
	// zwei Kader daran hängt. Ein Spiel bleibt ein Event (UID game-<id>); die
	// ORDER BY sorgt dafür, dass dabei die reguläre Zugehörigkeit die
	// erweiterte schlägt.
	seen := map[int]bool{}
	for rows.Next() {
		var id int
		var date, startTime, opponent, eventType string
		var endTime, endDate sql.NullString
		var isHome, isExtended bool
		var note string
		var vName, vStreet, vPostal, vCity, teamName string
		var lineupExists, inLineup bool
		var createdAt string
		if err := rows.Scan(&id, &date, &startTime, &endTime, &endDate,
			&opponent, &eventType, &isHome, &note,
			&vName, &vStreet, &vPostal, &vCity, &teamName, &isExtended, &createdAt,
			&lineupExists, &inLineup); err != nil {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true

		state := resolveLineupState(isExtended, eventType, lineupExists, inLineup)
		summary := gameTitle(eventType, isHome, opponent,
			kaderLabel(teamName, isExtended, state))

		startDT := timez.ParseDT(date, startTime, loc)

		var endDT time.Time
		hasEnd := false
		if endTime.Valid && endTime.String != "" {
			endDateStr := date
			if endDate.Valid && endDate.String != "" {
				endDateStr = endDate.String
			}
			endDT = timez.ParseDT(endDateStr, endTime.String, loc)
			hasEnd = true
		} else {
			endDT = startDT.Add(2 * time.Hour)
			hasEnd = true
		}

		events = append(events, calEvent{
			UID:         fmt.Sprintf("game-%d@teamwerk", id),
			Summary:     summary,
			Location:    venueLocation(vName, vStreet, vPostal, vCity),
			Description: joinDescription(note, state.sentence()),
			Start:       startDT,
			End:         endDT,
			HasEnd:      hasEnd,
			Stamp:       parseStamp(createdAt),
		})
	}
	return events, rows.Err()
}

// fetchTrainings liefert Mannschafts- und Übungsgruppentermine. Der Kader wird
// über `ts.kader_id` aufgelöst, nicht über `team_id` + `season_id`: die frühere
// Doppelbedingung rekonstruierte den Kader aus Team und Saison, obwohl er
// direkt an der Zeile hängt — und bei einer Übungsgruppe ist `team_id` NULL, der
// JOIN matchte nie. Damit folgt der Feed derselben Auflösung wie
// `trainings.ListSessions`.
//
// includeTeams/includePractice müssen nicht beide gesetzt sein; ist keiner
// gesetzt, ruft `Feed` die Funktion gar nicht erst auf.
func (h *Handler) fetchTrainings(r *http.Request, userID int, includeTeams, includePractice bool) ([]calEvent, error) {
	// team_id IS NULL ist das Kennzeichen der Übungsgruppe (Gotcha
	// „Übungsgruppen": die Spalte ist eine nullable Projektion von
	// kader.team_id). Sind beide Arten gewünscht, entfällt der Filter.
	kindFilter := ""
	switch {
	case includeTeams && !includePractice:
		kindFilter = "AND ts.team_id IS NOT NULL"
	case !includeTeams && includePractice:
		kindFilter = "AND ts.team_id IS NULL"
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT DISTINCT
		    ts.id, ts.date, ts.start_time, ts.end_time,
		    COALESCE(t.name, k.name, ''), ts.note,
		    COALESCE(v.name,''), COALESCE(v.street,''), COALESCE(v.postal_code,''), COALESCE(v.city,''),
		    mem.is_extended, ts.created_at
		FROM training_sessions ts
		JOIN kader k ON k.id = ts.kader_id
		LEFT JOIN teams t ON t.id = ts.team_id
		JOIN (`+kaderMembership+`) mem ON mem.kader_id = k.id
		JOIN members m ON m.id = mem.member_id
		LEFT JOIN venues v ON v.id = ts.venue_id
		WHERE m.user_id = ? AND ts.status = 'active' `+kindFilter+`
		ORDER BY ts.date, ts.start_time, mem.is_extended`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Europe/Berlin")
	var events []calEvent
	// Wie bei den Spielen: ein Training bleibt ein Event (UID training-<id>),
	// auch wenn der Nutzer über mehrere Kader-Zugehörigkeiten daran hängt.
	seen := map[int]bool{}
	for rows.Next() {
		var id int
		// groupName ist der Mannschaftsname, bei einer Übungsgruppe (kein Team)
		// deren kader.name. Bewusst nicht ts.title: der Titel ist pro Termin
		// frei und driftet innerhalb derselben Serie, ein Kalendereintrag soll
		// wiedererkennbar bleiben.
		var date, startTime, endTime, groupName, note string
		var vName, vStreet, vPostal, vCity string
		var isExtended bool
		var createdAt string
		if err := rows.Scan(&id, &date, &startTime, &endTime, &groupName, &note,
			&vName, &vStreet, &vPostal, &vCity, &isExtended, &createdAt); err != nil {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		summary := "Training"
		if groupName != "" {
			summary = "Training: " + kaderLabel(groupName, isExtended, lineupNone)
		}
		start := timez.ParseDT(date, startTime, loc)
		end := timez.ParseDT(date, endTime, loc)
		events = append(events, calEvent{
			UID:         fmt.Sprintf("training-%d@teamwerk", id),
			Summary:     summary,
			Location:    venueLocation(vName, vStreet, vPostal, vCity),
			Description: note,
			Start:       start,
			End:         end,
			HasEnd:      true,
			Stamp:       parseStamp(createdAt),
		})
	}
	return events, rows.Err()
}

func (h *Handler) fetchDuties(r *http.Request, userID int) ([]calEvent, error) {
	// Der Ort kommt über den Spielbezug des Slots; beide Joins sind LEFT, damit
	// Dienste ohne Spiel (team_id statt game_id) und Spiele ohne Venue ohne
	// LOCATION im Feed bleiben statt herauszufallen.
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT ds.id, ds.event_name, ds.event_date, COALESCE(ds.event_time,''), dt.name,
		       ds.hours_value, COALESCE(ds.role_desc,''), ds.created_at,
		       COALESCE(v.name,''), COALESCE(v.street,''), COALESCE(v.postal_code,''), COALESCE(v.city,'')
		FROM duty_slots ds
		JOIN duty_assignments da ON da.duty_slot_id = ds.id
		JOIN duty_types dt ON dt.id = ds.duty_type_id
		LEFT JOIN games g ON g.id = ds.game_id
		LEFT JOIN venues v ON v.id = g.venue_id
		WHERE da.user_id = ? AND da.status IN ('assigned','fulfilled')
		ORDER BY ds.event_date, ds.event_time`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Europe/Berlin")
	var events []calEvent
	for rows.Next() {
		var id int
		var eventName, eventDate, eventTime, dutyTypeName, roleDesc, createdAt string
		var hours float64
		var vName, vStreet, vPostal, vCity string
		if err := rows.Scan(&id, &eventName, &eventDate, &eventTime, &dutyTypeName,
			&hours, &roleDesc, &createdAt,
			&vName, &vStreet, &vPostal, &vCity); err != nil {
			continue
		}
		e := calEvent{
			UID:         fmt.Sprintf("duty-%d@teamwerk", id),
			Summary:     "Dienst: " + dutyTypeName + " – " + eventName,
			Location:    venueLocation(vName, vStreet, vPostal, vCity),
			Description: roleDesc,
			HasEnd:      true,
			Stamp:       parseStamp(createdAt),
		}
		if eventTime == "" {
			// Ohne Uhrzeit ist die Aussage „irgendwann an diesem Tag" — ein
			// Termin ab Mitternacht über hours_value wäre erfunden.
			e.AllDay = true
			e.Start = timez.ParseDT(eventDate, "", loc)
			e.End = e.Start.AddDate(0, 0, 1)
		} else {
			e.Start = timez.ParseDT(eventDate, eventTime, loc)
			e.End = e.Start.Add(dutyDuration(hours))
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// dutyDuration rechnet duty_slots.hours_value (bereits durch die Ablösekette
// gekappt) in eine Dauer um. Nicht-positive Werte stammen aus kaputten
// Bestandszeilen und fallen auf eine Stunde zurück, damit kein Event mit
// DTEND <= DTSTART entsteht. Gerundet auf Minuten gegen Float-Rauschen.
func dutyDuration(hours float64) time.Duration {
	d := time.Duration(math.Round(hours*60)) * time.Minute
	if d <= 0 {
		return time.Hour
	}
	return d
}

// ── iCal rendering ───────────────────────────────────────────────────────────

func renderICal(events []calEvent, calName string) string {
	var sb strings.Builder
	writeLine(&sb, "BEGIN:VCALENDAR")
	writeLine(&sb, "VERSION:2.0")
	writeLine(&sb, "PRODID:-//TeamWERK//Team Stuttgart//DE")
	writeLine(&sb, "X-WR-CALNAME:"+escapeText(calName))
	writeLine(&sb, "CALSCALE:GREGORIAN")
	writeLine(&sb, "METHOD:PUBLISH")
	for _, l := range vtimezoneBerlin {
		writeLine(&sb, l)
	}
	for _, e := range events {
		sb.WriteString("BEGIN:VEVENT\r\n")
		writeField(&sb, "UID", e.UID)
		stamp := e.Stamp
		if stamp.IsZero() {
			stamp = time.Now()
		}
		writeLine(&sb, "DTSTAMP:"+stamp.UTC().Format("20060102T150405Z"))
		writeField(&sb, "SUMMARY", escapeText(e.Summary))
		if e.Location != "" {
			writeField(&sb, "LOCATION", escapeText(e.Location))
		}
		if e.Description != "" {
			writeField(&sb, "DESCRIPTION", escapeText(e.Description))
		}
		switch {
		case e.AllDay:
			writeLine(&sb, "DTSTART;VALUE=DATE:"+e.Start.Format("20060102"))
			writeLine(&sb, "DTEND;VALUE=DATE:"+e.End.Format("20060102"))
		default:
			writeLine(&sb, "DTSTART;TZID=Europe/Berlin:"+formatDT(e.Start))
			if e.HasEnd {
				writeLine(&sb, "DTEND;TZID=Europe/Berlin:"+formatDT(e.End))
			}
		}
		sb.WriteString("END:VEVENT\r\n")
	}
	sb.WriteString("END:VCALENDAR\r\n")
	return sb.String()
}

// vtimezoneBerlin definiert die TZID, auf die DTSTART/DTEND verweisen (RFC 5545
// verlangt die Komponente im Kalender). Die Regeln sind die EU-Sommerzeit
// (letzter Sonntag März/Oktober) als Literal, während die Zeitstempel selbst
// über time/tzdata gerechnet werden: schafft die EU die Umstellung ab, zieht
// tzdata nach, dieser Block aber nicht — dann hier mit anpassen.
var vtimezoneBerlin = []string{
	"BEGIN:VTIMEZONE",
	"TZID:Europe/Berlin",
	"BEGIN:DAYLIGHT",
	"TZOFFSETFROM:+0100",
	"TZOFFSETTO:+0200",
	"TZNAME:CEST",
	"DTSTART:19700329T020000",
	"RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU",
	"END:DAYLIGHT",
	"BEGIN:STANDARD",
	"TZOFFSETFROM:+0200",
	"TZOFFSETTO:+0100",
	"TZNAME:CET",
	"DTSTART:19701025T030000",
	"RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU",
	"END:STANDARD",
	"END:VTIMEZONE",
}

// writeLine faltet nach RFC 5545 bei 75 Oktetten (Fortsetzungszeilen 74 plus
// führendes Leerzeichen), schneidet aber nur an Rune-Grenzen — ein Schnitt in
// einer UTF-8-Sequenz macht beide Zeilen ungültig. Ein Codepoint hat höchstens
// 4 Byte, der Schnittpunkt liegt also immer > 0: kein Stillstand möglich.
func writeLine(sb *strings.Builder, line string) {
	limit := 75
	for {
		if len(line) <= limit {
			sb.WriteString(line)
			sb.WriteString("\r\n")
			return
		}
		cut := limit
		for !utf8.RuneStart(line[cut]) {
			cut--
		}
		sb.WriteString(line[:cut])
		sb.WriteString("\r\n ")
		line = line[cut:]
		limit = 74
	}
}

func writeField(sb *strings.Builder, name, value string) {
	writeLine(sb, name+":"+value)
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func formatDT(t time.Time) string {
	return t.Format("20060102T150405")
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func buildGameTypeFilter(s tokenSettings) []string {
	var types []string
	if s.IncludeHeim {
		types = append(types, "heim")
	}
	if s.IncludeAuswaerts {
		types = append(types, "auswärts")
	}
	if s.IncludeGenerisch {
		types = append(types, "generisch")
	}
	return types
}

// joinDescription setzt die Beschreibung aus der Notiz des Termins und dem
// generierten Aufstellungssatz zusammen, getrennt durch eine Leerzeile. Leere
// Teile entfallen. Die Notiz steht vorn: sie ist die Aussage des Trainers zum
// Termin, der Satz nur eine Ergänzung — umgekehrt schöbe sich generierter Text
// vor den redaktionellen.
func joinDescription(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, "\n\n")
}

// lineupState ist der Aufstellungsstatus eines Feed-Nutzers für ein Spiel. Der
// leere Wert heißt „gilt hier nicht" — er steht für alle Termine, an denen kein
// Status auszuweisen ist (regulärer Kader, Trainer, generische Events).
type lineupState string

const (
	lineupNone      lineupState = ""
	lineupIn        lineupState = "aufgestellt"
	lineupOut       lineupState = "nicht aufgestellt"
	lineupUndecided lineupState = "Aufstellung offen"
)

// Die Sätze für die Beschreibung, je Zustand. Wortlaut ist Zusage an den
// Empfänger und steht so in specs/ical-feed.
const (
	lineupSentenceIn        = "Du bist für das Spiel aufgestellt."
	lineupSentenceOut       = "Du bist für das Spiel NICHT aufgestellt. Bitte mit der Trainerin/dem Trainer absprechen, ob eine Anwesenheit trotzdem erwünscht ist."
	lineupSentenceUndecided = "Die Aufstellung für dieses Spiel steht noch nicht fest."
)

// resolveLineupState leitet den Status EINMAL aus den beiden EXISTS-Spalten ab.
// Die Regel „keine Zeile für das Spiel ≠ nicht nominiert" lebt allein hier: an
// zwei Stellen ausgewertet würde aus einer ungepflegten Aufstellung früher oder
// später eine Absage, die niemand ausgesprochen hat.
//
// Der Status gilt nur für Spieler des erweiterten Kaders an Heim-/Auswärtsspielen.
// Für den Stammkader ist die Teilnahme der Regelfall, generische Events haben
// keine Aufstellung.
func resolveLineupState(isExtended bool, eventType string, lineupExists, inLineup bool) lineupState {
	if !isExtended || (eventType != "heim" && eventType != "auswärts") {
		return lineupNone
	}
	switch {
	case inLineup:
		return lineupIn
	case lineupExists:
		return lineupOut
	default:
		return lineupUndecided
	}
}

// sentence liefert den Satz für die Beschreibung; leer, wenn kein Status gilt.
func (l lineupState) sentence() string {
	switch l {
	case lineupIn:
		return lineupSentenceIn
	case lineupOut:
		return lineupSentenceOut
	case lineupUndecided:
		return lineupSentenceUndecided
	default:
		return ""
	}
}

// kaderLabel benennt die Mannschaft, über die der Feed-Nutzer am Termin hängt
// (z. B. "mA1"). Hängt er nur über den erweiterten Kader daran, sagt das Label
// das dazu — der Termin gehört dann nicht zur eigenen Stammmannschaft — und bei
// Spielen zusätzlich, ob er aufgestellt ist.
//
// Der Zusatz ist auf "erw. Kader" gekürzt, damit Mannschaft, Kader und Status
// zusammen in die Titel-Klammer passen; der Mittelpunkt trennt sie stärker als
// ein weiterer Bindestrich, von dem der Spieltitel schon einen trägt.
// Trainings rufen denselben Helfer mit lineupNone auf — derselbe Zusatz darf im
// Kalender nicht in zwei Schreibweisen auftauchen.
func kaderLabel(teamName string, isExtended bool, state lineupState) string {
	if teamName == "" || !isExtended {
		return teamName
	}
	label := teamName + " · erw. Kader"
	if state != lineupNone {
		label += " · " + string(state)
	}
	return label
}

// ownTeamLabel setzt das Kader-Label in den Spieltitel; ohne auflösbaren
// Teamnamen bleibt es beim generischen "Team".
func ownTeamLabel(label string) string {
	if label == "" {
		return "Team"
	}
	return "Team (" + label + ")"
}

func gameTitle(eventType string, isHome bool, opponent, teamLabel string) string {
	switch eventType {
	case "heim":
		return "Heim: " + ownTeamLabel(teamLabel) + " – " + opponent
	case "auswärts":
		return "Auswärts: " + opponent + " – " + ownTeamLabel(teamLabel)
	default:
		return opponent
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
