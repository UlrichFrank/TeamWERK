package attendance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/httpx"
	"github.com/teamstuttgart/teamwerk/internal/policy"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// matrixMaxDays deckelt den Zeitraum der Rückmelde-Matrix: eine Saison plus
// Puffer, analog zum Ladefenster der Terminliste (web/src/lib/terminWindow.ts).
const matrixMaxDays = 400

// matrixEvent ist eine Spalte der Matrix (ein Training oder ein Spiel/Event).
type matrixEvent struct {
	Kind      string `json:"kind"` // training | game
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Time      string `json:"time"`
	EventType string `json:"event_type"` // training | heim | auswärts | generisch
	Title     string `json:"title"`
	Cancelled bool   `json:"cancelled"`
	// Ab hier dürfen Spieler/Eltern nicht mehr umsagen (policy.*RSVPCutoff
	// vor Beginn) — dieselbe Frist, die die Liste als rsvp_locks_at bekommt.
	RsvpLocksAt       string `json:"rsvp_locks_at,omitempty"`
	RsvpRequireReason bool   `json:"rsvp_require_reason"`

	defPlayers  string
	defExtended string
}

// matrixCell ist der Rückmeldestatus eines Spielers zu einem Termin. Present
// ist nur für Trainer-Sicht gesetzt (siehe design.md §2).
type matrixCell struct {
	Status      *string `json:"status"`
	IsDefault   bool    `json:"is_default"`
	Unavailable bool    `json:"unavailable,omitempty"`
	Present     *bool   `json:"present,omitempty"`
	// Locked: Antwort stammt aus einer erfassten Abwesenheit und ist nur über
	// die Abwesenheit änderbar (Respond antwortet sonst rsvp_locked_absence).
	Locked bool `json:"locked,omitempty"`
	// Reason nur in Zeilen mit CanRespond (eigene, Kinder) — design.md §2.
	Reason *string `json:"reason,omitempty"`
}

type matrixMember struct {
	MemberID int    `json:"member_id"`
	Name     string `json:"name"`
	Extended bool   `json:"extended"`
	// IsSelf: Mitglied des Aufrufers. CanRespond: der Aufrufer bietet für diese
	// Zeile Zu-/Absage an — eigenes Mitglied oder Kind (family_links), genau
	// die Zeilen, die die Liste als „Ich" bzw. Kindername zeigt.
	IsSelf     bool         `json:"is_self"`
	CanRespond bool         `json:"can_respond"`
	Cells      []matrixCell `json:"cells"`
}

type matrixResponse struct {
	TeamID   int            `json:"team_id"`
	TeamName string         `json:"team_name"`
	Events   []matrixEvent  `json:"events"`
	Members  []matrixMember `json:"members"`
}

// eventKey adressiert einen Termin über Art und ID (Trainings- und Spiel-IDs
// kommen aus getrennten Tabellen und überlappen).
type eventKey struct {
	kind string
	id   int
}

type memberEventKey struct {
	ev       eventKey
	memberID int
}

// canSeeTeamMatrix: wer die Termine der Mannschaft sehen darf, sieht auch die
// Rückmeldungen — dieselbe Menge wie auf der Detailseite (GetParticipants).
func (h *Handler) canSeeTeamMatrix(ctx context.Context, claims *auth.Claims, teamID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("vorstand") ||
		claims.HasFunction("sportliche_leitung") || claims.HasFunction("trainer") {
		return true, nil
	}
	var n int
	err := h.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_accessible_teams uat
		JOIN seasons s ON s.id = uat.season_id AND s.is_active = 1
		WHERE uat.user_id = ? AND uat.team_id = ?`,
		claims.UserID, teamID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// parseMatrixRange validiert from/to (beide Pflicht, from <= to, höchstens
// matrixMaxDays Tage).
func parseMatrixRange(r *http.Request) (from, to string, ok bool) {
	from, to = r.URL.Query().Get("from"), r.URL.Query().Get("to")
	f, err1 := time.Parse("2006-01-02", from)
	t, err2 := time.Parse("2006-01-02", to)
	if err1 != nil || err2 != nil || t.Before(f) || t.Sub(f) > matrixMaxDays*24*time.Hour {
		return "", "", false
	}
	return from, to, true
}

// GetTeamRSVPMatrix — GET /api/teams/{id}/rsvp-matrix?from=&to=
// Termine der Mannschaft als Spalten, Spieler des Kaders der aktiven Saison
// als Zeilen, je Zelle der Rückmeldestatus. Details:
// openspec/changes/termin-matrix/design.md.
func (h *Handler) GetTeamRSVPMatrix(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	teamID, ok := httpx.PathID(r, "id")
	if !ok {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, nil)
		return
	}

	resp := matrixResponse{TeamID: teamID, Events: []matrixEvent{}, Members: []matrixMember{}}
	err := h.db.QueryRowContext(r.Context(), `SELECT name FROM teams WHERE id = ?`, teamID).Scan(&resp.TeamName)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, nil)
		return
	}
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}

	allowed, err := h.canSeeTeamMatrix(r.Context(), claims, teamID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if !allowed {
		httpx.WriteError(w, r, http.StatusForbidden, httpx.CodeForbidden, nil)
		return
	}

	from, to, ok := parseMatrixRange(r)
	if !ok {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeValidation, nil)
		return
	}

	withPresence, err := h.canSeeTeamStats(r.Context(), claims, teamID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}

	resp.Events, err = h.loadMatrixEvents(r.Context(), teamID, from, to)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	members, err := h.loadMatrixMembers(r.Context(), teamID, claims.UserID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	facts, err := h.loadMatrixFacts(r.Context(), teamID, from, to, withPresence)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}

	for i := range members {
		m := &members[i]
		m.Cells = make([]matrixCell, len(resp.Events))
		for j, ev := range resp.Events {
			m.Cells[j] = facts.cell(ev, m.MemberID, m.Extended, m.CanRespond)
		}
	}
	resp.Members = members
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// loadMatrixEvents liefert Trainings (inkl. abgesagter) und Spiele der
// Mannschaft im Zeitraum, chronologisch.
func (h *Handler) loadMatrixEvents(ctx context.Context, teamID int, from, to string) ([]matrixEvent, error) {
	events := []matrixEvent{}

	rows, err := h.db.QueryContext(ctx, `
		SELECT ts.id, date(ts.date), ts.start_time,
		       COALESCE(NULLIF(ts.title, ''), 'Training'),
		       ts.status = 'cancelled',
		       ts.rsvp_default_players, ts.rsvp_default_extended,
		       ts.rsvp_require_reason
		FROM training_sessions ts
		WHERE ts.team_id = ?
		  AND date(ts.date) BETWEEN date(?) AND date(?)`,
		teamID, from, to)
	if err != nil {
		return nil, fmt.Errorf("matrix trainings: %w", err)
	}
	for rows.Next() {
		ev := matrixEvent{Kind: "training", EventType: "training"}
		if err := rows.Scan(&ev.ID, &ev.Date, &ev.Time, &ev.Title, &ev.Cancelled,
			&ev.defPlayers, &ev.defExtended, &ev.RsvpRequireReason); err != nil {
			rows.Close()
			return nil, fmt.Errorf("matrix trainings scan: %w", err)
		}
		events = append(events, ev)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("matrix trainings rows: %w", err)
	}

	rows, err = h.db.QueryContext(ctx, `
		SELECT g.id, date(g.date), g.time, g.opponent, g.event_type,
		       g.rsvp_default_players, g.rsvp_default_extended,
		       g.rsvp_require_reason
		FROM games g
		JOIN game_teams gt ON gt.game_id = g.id AND gt.team_id = ?
		WHERE date(g.date) BETWEEN date(?) AND date(?)`,
		teamID, from, to)
	if err != nil {
		return nil, fmt.Errorf("matrix games: %w", err)
	}
	for rows.Next() {
		ev := matrixEvent{Kind: "game"}
		if err := rows.Scan(&ev.ID, &ev.Date, &ev.Time, &ev.Title, &ev.EventType,
			&ev.defPlayers, &ev.defExtended, &ev.RsvpRequireReason); err != nil {
			rows.Close()
			return nil, fmt.Errorf("matrix games scan: %w", err)
		}
		events = append(events, ev)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("matrix games rows: %w", err)
	}

	for i := range events {
		cutoff := policy.GameRSVPCutoff
		if events[i].Kind == "training" {
			cutoff = policy.TrainingRSVPCutoff
		}
		events[i].RsvpLocksAt = rsvpLocksAt(events[i].Date, events[i].Time, cutoff)
	}

	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Date != events[j].Date {
			return events[i].Date < events[j].Date
		}
		if events[i].Time != events[j].Time {
			return events[i].Time < events[j].Time
		}
		if events[i].Kind != events[j].Kind {
			return events[i].Kind < events[j].Kind
		}
		return events[i].ID < events[j].ID
	})
	return events, nil
}

// rsvpLocksAt rechnet Beginn (Europe/Berlin) minus Frist als RFC3339 in UTC —
// dieselbe Rechnung wie games.gameLocksAt/trainings.trainingLocksAt. Leer bei
// unlesbarem Datum/Zeit, dann greift nur die serverseitige Prüfung.
func rsvpLocksAt(date, hhmm string, cutoff time.Duration) string {
	if len(hhmm) > 5 {
		hhmm = hhmm[:5]
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+hhmm, timez.Berlin())
	if err != nil {
		return ""
	}
	return t.Add(-cutoff).UTC().Format(time.RFC3339)
}

// loadMatrixMembers liefert Stammkader, dann erweiterten Kader (ohne Doppelte)
// der aktiven Saison, jeweils nach Name sortiert. Ohne aktive Saison leer.
func (h *Handler) loadMatrixMembers(ctx context.Context, teamID, callerUserID int) ([]matrixMember, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT roster.id, roster.name, roster.extended,
		       COALESCE(roster.user_id, 0) = ? AS is_self,
		       EXISTS (SELECT 1 FROM family_links fl
		               WHERE fl.member_id = roster.id AND fl.parent_user_id = ?) AS is_child
		FROM (
		SELECT m.id, m.first_name || ' ' || m.last_name AS name, 0 AS extended, m.user_id
		FROM members m
		JOIN kader_members km ON km.member_id = m.id
		JOIN kader k ON k.id = km.kader_id AND k.team_id = ?
		JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		UNION
		SELECT m.id, m.first_name || ' ' || m.last_name, 1 AS extended, m.user_id
		FROM members m
		JOIN kader_extended_members kem ON kem.member_id = m.id
		JOIN kader k ON k.id = kem.kader_id AND k.team_id = ?
		JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
		WHERE NOT EXISTS (
			SELECT 1 FROM kader_members km2
			WHERE km2.member_id = m.id AND km2.kader_id = k.id
		)
		) AS roster
		ORDER BY roster.extended, roster.name`,
		callerUserID, callerUserID, teamID, teamID)
	if err != nil {
		return nil, fmt.Errorf("matrix members: %w", err)
	}
	defer rows.Close()
	members := []matrixMember{}
	for rows.Next() {
		var m matrixMember
		var isChild bool
		if err := rows.Scan(&m.MemberID, &m.Name, &m.Extended, &m.IsSelf, &isChild); err != nil {
			return nil, fmt.Errorf("matrix members scan: %w", err)
		}
		m.CanRespond = m.IsSelf || isChild
		members = append(members, m)
	}
	return members, rows.Err()
}

// matrixFacts sammelt Antworten, Anwesenheit und Serien-Abmeldungen aller
// Termine im Zeitraum — je Tabelle eine Abfrage, unabhängig von der Spaltenzahl.
type matrixFacts struct {
	responses   map[memberEventKey]matrixResponseRow
	present     map[memberEventKey]bool
	unavailable map[memberEventKey]bool
}

// cell leitet den Zellwert ab — dieselbe Regel wie GetParticipants bzw.
// trainings.GetAttendances: Antwort vor Rollen-Voreinstellung, 'none' bleibt leer.
func (f matrixFacts) cell(ev matrixEvent, memberID int, extended, withReason bool) matrixCell {
	k := memberEventKey{eventKey{ev.Kind, ev.ID}, memberID}
	var c matrixCell
	if resp, ok := f.responses[k]; ok {
		st := resp.status
		c.Status = &st
		c.Locked = resp.fromAbsence
		if withReason && resp.reason != "" {
			reason := resp.reason
			c.Reason = &reason
		}
	} else {
		def := ev.defPlayers
		if extended {
			def = ev.defExtended
		}
		if def == "confirmed" || def == "declined" {
			c.Status = &def
			c.IsDefault = true
		}
	}
	if p, ok := f.present[k]; ok {
		b := p
		c.Present = &b
	}
	c.Unavailable = f.unavailable[k]
	return c
}

// matrixResponseRow ist eine gespeicherte Rückmeldung.
type matrixResponseRow struct {
	status      string
	reason      string
	fromAbsence bool
}

func (h *Handler) loadMatrixFacts(ctx context.Context, teamID int, from, to string, withPresence bool) (matrixFacts, error) {
	f := matrixFacts{
		responses:   map[memberEventKey]matrixResponseRow{},
		present:     map[memberEventKey]bool{},
		unavailable: map[memberEventKey]bool{},
	}

	// Jede Abfrage liefert (Termin-ID, Mitglied, Wert) und bindet dieselben drei
	// Parameter (team, from, to).
	type query struct {
		kind string
		sql  string
		into func(k memberEventKey, v string)
	}
	queries := []query{
		{"training", `
			SELECT DISTINCT ts.id, msu.member_id, ''
			FROM training_sessions ts
			JOIN member_series_unavailabilities msu
			  ON msu.training_series_id = ts.series_id
			 AND (msu.start_date IS NULL OR msu.start_date <= date(ts.date))
			 AND (msu.end_date   IS NULL OR msu.end_date   >= date(ts.date))
			WHERE ts.team_id = ? AND date(ts.date) BETWEEN date(?) AND date(?)`,
			func(k memberEventKey, _ string) { f.unavailable[k] = true }},
	}
	if withPresence {
		queries = append(queries,
			query{"training", `
				SELECT ta.training_id, ta.member_id, CAST(ta.present AS TEXT)
				FROM training_attendances ta
				JOIN training_sessions ts ON ts.id = ta.training_id AND ts.attendance_tracked = 1
				WHERE ts.team_id = ? AND date(ts.date) BETWEEN date(?) AND date(?)`,
				func(k memberEventKey, v string) { f.present[k] = v == "1" }},
			query{"game", `
				SELECT ga.game_id, ga.member_id, CAST(ga.present AS TEXT)
				FROM game_attendances ga
				JOIN game_teams gt ON gt.game_id = ga.game_id AND gt.team_id = ?
				JOIN games g ON g.id = ga.game_id AND g.attendance_tracked = 1
				WHERE date(g.date) BETWEEN date(?) AND date(?)`,
				func(k memberEventKey, v string) { f.present[k] = v == "1" }},
		)
	}

	responseQueries := []struct{ kind, sql string }{
		{"training", `
			SELECT tr.training_id, tr.member_id, tr.status, tr.reason, tr.absence_id IS NOT NULL
			FROM training_responses tr
			JOIN training_sessions ts ON ts.id = tr.training_id
			WHERE ts.team_id = ? AND date(ts.date) BETWEEN date(?) AND date(?)`},
		{"game", `
			SELECT gr.game_id, gr.member_id, gr.status, gr.reason, gr.absence_id IS NOT NULL
			FROM game_responses gr
			JOIN game_teams gt ON gt.game_id = gr.game_id AND gt.team_id = ?
			JOIN games g ON g.id = gr.game_id
			WHERE date(g.date) BETWEEN date(?) AND date(?)`},
	}
	for _, q := range responseQueries {
		rows, err := h.db.QueryContext(ctx, q.sql, teamID, from, to)
		if err != nil {
			return f, fmt.Errorf("matrix responses (%s): %w", q.kind, err)
		}
		for rows.Next() {
			var evID, memberID int
			var row matrixResponseRow
			if err := rows.Scan(&evID, &memberID, &row.status, &row.reason, &row.fromAbsence); err != nil {
				rows.Close()
				return f, fmt.Errorf("matrix responses scan (%s): %w", q.kind, err)
			}
			f.responses[memberEventKey{eventKey{q.kind, evID}, memberID}] = row
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return f, fmt.Errorf("matrix responses rows (%s): %w", q.kind, err)
		}
	}

	for _, q := range queries {
		rows, err := h.db.QueryContext(ctx, q.sql, teamID, from, to)
		if err != nil {
			return f, fmt.Errorf("matrix facts (%s): %w", q.kind, err)
		}
		for rows.Next() {
			var evID, memberID int
			var v string
			if err := rows.Scan(&evID, &memberID, &v); err != nil {
				rows.Close()
				return f, fmt.Errorf("matrix facts scan (%s): %w", q.kind, err)
			}
			q.into(memberEventKey{eventKey{q.kind, evID}, memberID}, v)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return f, fmt.Errorf("matrix facts rows (%s): %w", q.kind, err)
		}
	}
	return f, nil
}
