package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/httpx"
)

// Abgleich einer Chat-Gruppe gegen ihre Standard-Gruppen-Herkunft
// (chat-gruppe-aktualisieren).
//
// Zwei Phasen wie beim H4A-Import und beim Massen-Dienstregen: preview rechnet
// und schreibt nichts, apply rechnet in der Transaktion NEU und nimmt nur IDs
// an, die im neuen Diff stehen — sonst wäre apply ein verkapptes
// Massen-AddMember für beliebige IDs.
//
// Sperre statt Massenentfernung: eine Kachel, die zu null Personen auflöst
// (Saisonwechsel, neuer Kader noch leer) oder für den Ersteller nicht sichtbar
// ist (Übergabe an jemanden ohne Zugriff), sperrt den Abgleich. Überspringen
// hieße, alle ihre Mitglieder in „Entfernen" zu führen.

const (
	syncProblemEmpty      = "empty"
	syncProblemNotVisible = "not_visible"
)

// SyncSource ist eine Kachel in der Vorschau, samt Überlappung mit der Gruppe.
type SyncSource struct {
	GroupSource
	Label     string `json:"label"`
	Total     int    `json:"total"`
	AlreadyIn int    `json:"alreadyIn"`
	Problem   string `json:"problem,omitempty"`
}

// SyncPreview ist die Antwort von POST …/sync/preview.
type SyncPreview struct {
	Sources     []SyncSource      `json:"sources"`
	Add         []TeamGroupMember `json:"add"`
	Remove      []TeamGroupMember `json:"remove"`
	Blocked     bool              `json:"blocked"`
	Suggestions []SyncSource      `json:"suggestions,omitempty"`
}

var kindLabel = map[string]string{
	"trainer":      "Trainer",
	"spieler":      "Spieler",
	"eltern":       "Eltern",
	"alle_trainer": "Alle Trainer",
}

// sourceLabel baut die Anzeige wie im Dialog „Neues Gespräch" („Spieler mC1").
// Auch für nicht sichtbare Kacheln — der Ersteller muss sie erkennen und
// abwählen können; ein Teamkürzel ist kein Geheimnis.
func sourceLabel(ctx context.Context, q queryer, s GroupSource) string {
	if s.Kind == "alle_trainer" {
		return kindLabel[s.Kind]
	}
	var name sql.NullString
	if s.GroupType == "team" {
		q.QueryRowContext(ctx,
			`SELECT COALESCE(`+db.TeamDisplayShort("t")+`, t.name) FROM teams t WHERE t.id = ?`,
			s.RefID).Scan(&name)
		if !name.Valid || name.String == "" {
			name.String = "unbekannte Mannschaft"
		}
	} else {
		q.QueryRowContext(ctx, `SELECT COALESCE(name, '') FROM kader WHERE id = ?`, s.RefID).Scan(&name)
		if !name.Valid || name.String == "" {
			name.String = "unbekannte Übungsgruppe"
		}
	}
	return kindLabel[s.Kind] + " " + name.String
}

// activeConvMembers liefert die aktiven Mitglieder (left_at IS NULL) mit Namen.
func activeConvMembers(ctx context.Context, q queryer, convID int) (map[int]TeamGroupMember, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT cm.user_id, u.first_name || ' ' || u.last_name
		FROM conversation_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.conversation_id = ? AND cm.left_at IS NULL`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]TeamGroupMember{}
	for rows.Next() {
		var m TeamGroupMember
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, err
		}
		out[m.ID] = m
	}
	return out, rows.Err()
}

// describeSource löst eine Kachel für die Vorschau auf: Label, Mitglieder,
// Überlappung mit der Gruppe, Problem. members ist nil bei einem Problem.
func (h *Handler) describeSource(r *http.Request, q queryer, claims *auth.Claims, s GroupSource, active map[int]TeamGroupMember) (SyncSource, []TeamGroupMember, error) {
	out := SyncSource{GroupSource: s, Label: sourceLabel(r.Context(), q, s)}
	exists, err := sourceExists(r.Context(), q, s)
	if err != nil {
		return out, nil, err
	}
	if !exists {
		out.Problem = syncProblemEmpty
		return out, nil, nil
	}
	visible, err := h.canSeeSource(r, claims, s)
	if err != nil {
		return out, nil, err
	}
	if !visible {
		out.Problem = syncProblemNotVisible
		return out, nil, nil
	}
	members, err := resolveSource(r.Context(), q, s, claims.UserID)
	if err != nil {
		return out, nil, err
	}
	if len(members) == 0 {
		out.Problem = syncProblemEmpty
		return out, nil, nil
	}
	out.Total = len(members)
	for _, m := range members {
		if _, ok := active[m.ID]; ok {
			out.AlreadyIn++
		}
	}
	return out, members, nil
}

// computeSyncDiff rechnet den Abgleich der Gruppe convID gegen sources. Der
// Caller ist der Ersteller; er steht nie in remove.
func (h *Handler) computeSyncDiff(r *http.Request, q queryer, convID int, claims *auth.Claims, sources []GroupSource) (*SyncPreview, error) {
	active, err := activeConvMembers(r.Context(), q, convID)
	if err != nil {
		return nil, err
	}
	p := &SyncPreview{Sources: []SyncSource{}, Add: []TeamGroupMember{}, Remove: []TeamGroupMember{}}
	soll := map[int]TeamGroupMember{}
	for _, s := range sources {
		desc, members, err := h.describeSource(r, q, claims, s, active)
		if err != nil {
			return nil, err
		}
		if desc.Problem != "" {
			p.Blocked = true
		}
		p.Sources = append(p.Sources, desc)
		for _, m := range members {
			soll[m.ID] = m
		}
	}
	if p.Blocked || len(sources) == 0 {
		return p, nil
	}
	for id, m := range soll {
		if _, ok := active[id]; !ok {
			p.Add = append(p.Add, m)
		}
	}
	for id, m := range active {
		if _, ok := soll[id]; !ok && id != claims.UserID {
			p.Remove = append(p.Remove, m)
		}
	}
	byName := func(list []TeamGroupMember) {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Name != list[j].Name {
				return list[i].Name < list[j].Name
			}
			return list[i].ID < list[j].ID
		})
	}
	byName(p.Add)
	byName(p.Remove)
	return p, nil
}

// suggestSources liefert alle für den Caller sichtbaren Kacheln mit ihrer
// Überlappung — nur Auswahlhilfe, nie vorausgewählt.
func (h *Handler) suggestSources(r *http.Request, claims *auth.Claims, convID int) ([]SyncSource, error) {
	active, err := activeConvMembers(r.Context(), h.db, convID)
	if err != nil {
		return nil, err
	}
	out := []SyncSource{}
	for _, g := range h.visibleTeamGroups(r, claims) {
		s := GroupSource{GroupType: g.GroupType, RefID: g.TeamID, Kind: g.Kind}
		members, err := resolveSource(r.Context(), h.db, s, claims.UserID)
		if err != nil {
			return nil, err
		}
		desc := SyncSource{GroupSource: s, Label: sourceLabel(r.Context(), h.db, s), Total: len(members)}
		for _, m := range members {
			if _, ok := active[m.ID]; ok {
				desc.AlreadyIn++
			}
		}
		out = append(out, desc)
	}
	return out, nil
}

// loadSyncTarget prüft die Konversation für beide Abgleich-Routen:
// 404 → 403 (nicht Ersteller) → 400 (keine Gruppe). ok=false heißt: Antwort
// ist geschrieben.
func (h *Handler) loadSyncTarget(w http.ResponseWriter, r *http.Request, claims *auth.Claims) (int, bool) {
	convID, ok := httpx.PathID(r, "id")
	if !ok {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, nil)
		return 0, false
	}
	var convType string
	var createdBy int
	err := h.db.QueryRowContext(r.Context(),
		`SELECT type, created_by FROM conversations WHERE id = ?`, convID).Scan(&convType, &createdBy)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, nil)
		return 0, false
	}
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return 0, false
	}
	if createdBy != claims.UserID {
		httpx.WriteError(w, r, http.StatusForbidden, httpx.CodeForbidden, nil)
		return 0, false
	}
	if convType != "group" {
		httpx.WriteError(w, r, http.StatusBadRequest, "not_a_group", nil)
		return 0, false
	}
	return convID, true
}

// syncSources bestimmt die Kacheln einer Abgleich-Anfrage: übergeben (dedupliziert,
// Form geprüft) oder — fehlt das Feld — die gespeicherte Herkunft.
func (h *Handler) syncSources(w http.ResponseWriter, r *http.Request, q queryer, convID int, given *[]GroupSource) ([]GroupSource, bool) {
	if given == nil {
		stored, err := loadSources(r.Context(), q, convID)
		if err != nil {
			httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
			return nil, false
		}
		return stored, true
	}
	sources := dedupSources(*given)
	for _, s := range sources {
		if !validSource(s) {
			httpx.WriteError(w, r, http.StatusBadRequest, "invalid_source", nil)
			return nil, false
		}
	}
	return sources, true
}

// decodeOptionalBody akzeptiert einen leeren Body als „keine Angaben".
func decodeOptionalBody(r *http.Request, v any) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// SyncPreview — POST /api/chat/conversations/{id}/sync/preview
func (h *Handler) SyncPreview(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, ok := h.loadSyncTarget(w, r, claims)
	if !ok {
		return
	}
	var body struct {
		Sources *[]GroupSource `json:"sources"`
	}
	if err := decodeOptionalBody(r, &body); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidBody, nil)
		return
	}
	sources, ok := h.syncSources(w, r, h.db, convID, body.Sources)
	if !ok {
		return
	}
	preview, err := h.computeSyncDiff(r, h.db, convID, claims, sources)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if preview.Suggestions, err = h.suggestSources(r, claims, convID); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, preview)
}

// SyncApply — POST /api/chat/conversations/{id}/sync/apply
func (h *Handler) SyncApply(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, ok := h.loadSyncTarget(w, r, claims)
	if !ok {
		return
	}
	var body struct {
		Sources       *[]GroupSource `json:"sources"`
		AddUserIDs    []int          `json:"addUserIds"`
		RemoveUserIDs []int          `json:"removeUserIds"`
	}
	if err := decodeOptionalBody(r, &body); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidBody, nil)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	defer tx.Rollback()

	sources, ok := h.syncSources(w, r, tx, convID, body.Sources)
	if !ok {
		return
	}
	diff, err := h.computeSyncDiff(r, tx, convID, claims, sources)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if diff.Blocked {
		httpx.WriteError(w, r, http.StatusConflict, "sync_blocked", nil)
		return
	}
	add, okAdd := pickFromDiff(body.AddUserIDs, diff.Add)
	remove, okRemove := pickFromDiff(body.RemoveUserIDs, diff.Remove)
	if !okAdd || !okRemove {
		httpx.WriteError(w, r, http.StatusConflict, "sync_stale", nil)
		return
	}
	for _, uid := range add {
		can, err := h.canContactUser(r, claims, uid)
		if err != nil {
			httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
			return
		}
		if !can {
			httpx.WriteError(w, r, http.StatusForbidden, httpx.CodeForbidden, nil)
			return
		}
	}

	if err := writeSources(r.Context(), tx, convID, sources); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	for _, uid := range add {
		if err := addMemberTx(r.Context(), tx, convID, uid); err != nil {
			httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
			return
		}
	}
	for _, uid := range remove {
		if err := removeMemberTx(r.Context(), tx, convID, uid); err != nil {
			httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}

	// Events gebündelt nach dem Commit: je Empfänger höchstens eins je Typ.
	switch {
	case len(add) > 0 || len(remove) > 0:
		if len(add) > 0 {
			event := fmt.Sprintf("chat:new-message:%d", convID)
			for _, uid := range h.activeMembers(r, convID, 0) {
				h.hub.BroadcastToUser(uid, event)
			}
		}
		if len(remove) > 0 {
			event := fmt.Sprintf("chat:member-left:%d", convID)
			for _, uid := range h.activeMembers(r, convID, 0) {
				h.hub.BroadcastToUser(uid, event)
			}
			for _, uid := range remove {
				h.hub.BroadcastToUser(uid, event)
			}
		}
	default:
		// Nur die Herkunft geändert: die übrigen Sitzungen des Erstellers laden nach.
		h.hub.BroadcastToUser(claims.UserID, fmt.Sprintf("chat:new-message:%d", convID))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]int{"added": len(add), "removed": len(remove)})
}

// pickFromDiff prüft, dass jede gewünschte ID im Diff steht, und dedupliziert.
func pickFromDiff(want []int, diff []TeamGroupMember) ([]int, bool) {
	allowed := make(map[int]bool, len(diff))
	for _, m := range diff {
		allowed[m.ID] = true
	}
	seen := map[int]bool{}
	out := []int{}
	for _, id := range want {
		if !allowed[id] {
			return nil, false
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, true
}

// addMemberTx und removeMemberTx schreiben dasselbe wie AddMember/RemoveMember
// (inklusive Systemnachricht), nur in der Abgleich-Transaktion.
func addMemberTx(ctx context.Context, tx *sql.Tx, convID, uid int) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE conversation_members SET left_at = NULL WHERE conversation_id = ? AND user_id = ?`,
		convID, uid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`,
			convID, uid); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, 'wurde hinzugefügt', 1)`,
		convID, uid)
	return err
}

func removeMemberTx(ctx context.Context, tx *sql.Tx, convID, uid int) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP
		 WHERE conversation_id = ? AND user_id = ? AND left_at IS NULL`,
		convID, uid); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, 'wurde entfernt', 1)`,
		convID, uid)
	return err
}
