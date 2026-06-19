package calendar

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/teamstuttgart/teamwerk/internal/auth"
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
}

// GET /api/calendar/token
func (h *Handler) GetToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var s tokenSettings
	err := h.db.QueryRowContext(r.Context(),
		`SELECT token, include_heim, include_auswaerts, include_training, include_generisch, include_duty
		 FROM calendar_tokens WHERE user_id = ?`, claims.UserID).
		Scan(&s.Token, &s.IncludeHeim, &s.IncludeAuswaerts, &s.IncludeTraining, &s.IncludeGenerisch, &s.IncludeDuty)
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
			`INSERT INTO calendar_tokens (user_id, token, include_heim, include_auswaerts, include_training, include_generisch, include_duty)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			claims.UserID, token,
			boolToInt(req.IncludeHeim), boolToInt(req.IncludeAuswaerts),
			boolToInt(req.IncludeTraining), boolToInt(req.IncludeGenerisch),
			boolToInt(req.IncludeDuty))
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
			`UPDATE calendar_tokens SET include_heim=?, include_auswaerts=?, include_training=?, include_generisch=?, include_duty=?
			 WHERE user_id=?`,
			boolToInt(req.IncludeHeim), boolToInt(req.IncludeAuswaerts),
			boolToInt(req.IncludeTraining), boolToInt(req.IncludeGenerisch),
			boolToInt(req.IncludeDuty), claims.UserID)
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

// GET /api/calendar/feed/{token}.ics
func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	// Strip .ics suffix if present (Chi may include it in the path value).
	token = strings.TrimSuffix(token, ".ics")

	var s tokenSettings
	var userID int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT user_id, token, include_heim, include_auswaerts, include_training, include_generisch, include_duty
		 FROM calendar_tokens WHERE token = ?`, token).
		Scan(&userID, &s.Token, &s.IncludeHeim, &s.IncludeAuswaerts, &s.IncludeTraining, &s.IncludeGenerisch, &s.IncludeDuty)
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

	// Trainings (training_sessions table — separate from games).
	if s.IncludeTraining {
		trainings, err := h.fetchTrainings(r, userID)
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

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"teamwerk.ics\"")
	fmt.Fprint(w, renderICal(events))
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
}

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
		    g.opponent, g.event_type, g.is_home,
		    COALESCE(v.name,''), COALESCE(v.street,''), COALESCE(v.postal_code,''), COALESCE(v.city,'')
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id
		JOIN kader k ON k.team_id = gt.team_id AND k.season_id = g.season_id
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		LEFT JOIN venues v ON v.id = g.venue_id
		WHERE m.user_id = ? AND g.event_type IN (`+placeholders+`)
		ORDER BY g.date, g.time`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Europe/Berlin")
	var events []calEvent
	for rows.Next() {
		var id int
		var date, startTime, opponent, eventType string
		var endTime, endDate sql.NullString
		var isHome bool
		var vName, vStreet, vPostal, vCity string
		if err := rows.Scan(&id, &date, &startTime, &endTime, &endDate,
			&opponent, &eventType, &isHome,
			&vName, &vStreet, &vPostal, &vCity); err != nil {
			continue
		}

		summary := gameTitle(eventType, isHome, opponent)

		var location string
		if vName != "" {
			parts := []string{vName}
			if vStreet != "" {
				parts = append(parts, vStreet)
			}
			if vPostal != "" || vCity != "" {
				parts = append(parts, strings.TrimSpace(vPostal+" "+vCity))
			}
			location = strings.Join(parts, ", ")
		}

		startDT := parseDT(date, startTime, loc)

		var endDT time.Time
		hasEnd := false
		if endTime.Valid && endTime.String != "" {
			endDateStr := date
			if endDate.Valid && endDate.String != "" {
				endDateStr = endDate.String
			}
			endDT = parseDT(endDateStr, endTime.String, loc)
			hasEnd = true
		} else {
			endDT = startDT.Add(2 * time.Hour)
			hasEnd = true
		}

		events = append(events, calEvent{
			UID:      fmt.Sprintf("game-%d@teamwerk", id),
			Summary:  summary,
			Location: location,
			Start:    startDT,
			End:      endDT,
			HasEnd:   hasEnd,
		})
	}
	return events, rows.Err()
}

func (h *Handler) fetchTrainings(r *http.Request, userID int) ([]calEvent, error) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT DISTINCT
		    ts.id, ts.date, ts.start_time, ts.end_time,
		    ts.location, COALESCE(t.name, '')
		FROM training_sessions ts
		JOIN teams t ON t.id = ts.team_id
		JOIN kader k ON k.team_id = ts.team_id AND k.season_id = ts.season_id
		JOIN kader_members km ON km.kader_id = k.id
		JOIN members m ON m.id = km.member_id
		WHERE m.user_id = ? AND ts.status = 'active'
		ORDER BY ts.date, ts.start_time`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loc, _ := time.LoadLocation("Europe/Berlin")
	var events []calEvent
	for rows.Next() {
		var id int
		var date, startTime, endTime, location, teamName string
		if err := rows.Scan(&id, &date, &startTime, &endTime, &location, &teamName); err != nil {
			continue
		}
		summary := "Training"
		if teamName != "" {
			summary = "Training: " + teamName
		}
		start := parseDT(date, startTime, loc)
		end := parseDT(date, endTime, loc)
		events = append(events, calEvent{
			UID:      fmt.Sprintf("training-%d@teamwerk", id),
			Summary:  summary,
			Location: location,
			Start:    start,
			End:      end,
			HasEnd:   true,
		})
	}
	return events, rows.Err()
}

func (h *Handler) fetchDuties(r *http.Request, userID int) ([]calEvent, error) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT ds.id, ds.event_name, ds.event_date, COALESCE(ds.event_time,''), dt.name
		FROM duty_slots ds
		JOIN duty_assignments da ON da.duty_slot_id = ds.id
		JOIN duty_types dt ON dt.id = ds.duty_type_id
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
		var eventName, eventDate, eventTime, dutyTypeName string
		if err := rows.Scan(&id, &eventName, &eventDate, &eventTime, &dutyTypeName); err != nil {
			continue
		}
		if eventTime == "" {
			eventTime = "00:00"
		}
		start := parseDT(eventDate, eventTime, loc)
		events = append(events, calEvent{
			UID:     fmt.Sprintf("duty-%d@teamwerk", id),
			Summary: "Dienst: " + dutyTypeName + " – " + eventName,
			Start:   start,
			End:     start.Add(time.Hour),
			HasEnd:  true,
		})
	}
	return events, rows.Err()
}

// ── iCal rendering ───────────────────────────────────────────────────────────

func renderICal(events []calEvent) string {
	var sb strings.Builder
	writeLine(&sb, "BEGIN:VCALENDAR")
	writeLine(&sb, "VERSION:2.0")
	writeLine(&sb, "PRODID:-//TeamWERK//Team Stuttgart//DE")
	writeLine(&sb, "X-WR-CALNAME:TeamWERK")
	writeLine(&sb, "CALSCALE:GREGORIAN")
	writeLine(&sb, "METHOD:PUBLISH")
	for _, e := range events {
		sb.WriteString("BEGIN:VEVENT\r\n")
		writeField(&sb, "UID", e.UID)
		writeField(&sb, "SUMMARY", escapeText(e.Summary))
		if e.Location != "" {
			writeField(&sb, "LOCATION", escapeText(e.Location))
		}
		if e.Description != "" {
			writeField(&sb, "DESCRIPTION", escapeText(e.Description))
		}
		writeLine(&sb, "DTSTART;TZID=Europe/Berlin:"+formatDT(e.Start))
		if e.HasEnd {
			writeLine(&sb, "DTEND;TZID=Europe/Berlin:"+formatDT(e.End))
		}
		sb.WriteString("END:VEVENT\r\n")
	}
	sb.WriteString("END:VCALENDAR\r\n")
	return sb.String()
}

// writeLine folds at 75 octets per RFC 5545.
func writeLine(sb *strings.Builder, line string) {
	b := []byte(line)
	const max = 75
	if len(b) <= max {
		sb.Write(b)
		sb.WriteString("\r\n")
		return
	}
	sb.Write(b[:max])
	sb.WriteString("\r\n")
	b = b[max:]
	for len(b) > 0 {
		sb.WriteByte(' ')
		n := 74 // continuation lines: 1 space + 74 chars = 75
		if n > len(b) {
			n = len(b)
		}
		sb.Write(b[:n])
		sb.WriteString("\r\n")
		b = b[n:]
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

func parseDT(date, timeStr string, loc *time.Location) time.Time {
	// modernc.org/sqlite returns DATE columns as full ISO timestamps
	// ("2026-08-15T00:00:00Z"), not "2026-08-15" — normalize to the date part.
	if len(date) > 10 {
		date = date[:10]
	}
	if timeStr == "" {
		timeStr = "00:00"
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, loc)
	if err != nil {
		t, _ = time.ParseInLocation("2006-01-02", date, loc)
	}
	return t
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

func gameTitle(eventType string, isHome bool, opponent string) string {
	switch eventType {
	case "heim":
		return "Heim: SG Stuttgart – " + opponent
	case "auswärts":
		return "Auswärts: " + opponent + " – SG Stuttgart"
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
