package videos

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// teamRef ist eine Team-Kurzreferenz in der Video-Antwort.
type teamRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// videoListItem ist ein Eintrag der Listen-Antwort (GET /api/videos).
type videoListItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	TeamID      int       `json:"team_id"`
	TeamName    string    `json:"team_name"`
	Teams       []teamRef `json:"teams"`
	SeasonID    int       `json:"season_id"`
	GameID      *int      `json:"game_id"`
	Status      string    `json:"status"`
	DurationSec *int      `json:"duration_sec"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   string    `json:"created_at"`
	ReadyAt     *string   `json:"ready_at"`
}

// videoDetail ist die Detail-Antwort (GET /api/videos/{id}). Sie erweitert den
// Listen-Eintrag um die selten gebrauchten Felder.
type videoDetail struct {
	videoListItem
	UploadID      *string `json:"upload_id"`
	SizeBytes     *int64  `json:"size_bytes"`
	FailureReason *string `json:"failure_reason"`
}

// visibilityFilter liefert ein SQL-Fragment (ohne führendes AND/WHERE) plus die
// zugehörigen Argumente, das die Sichtbarkeit über video_teams prüft.
// admin/vorstand sehen alles → ("", nil). Andernfalls: Video sichtbar, wenn
// mindestens eines der zugeordneten Teams zum Nutzer gehört (aktiver Spieler /
// Trainer / Elternteil eines aktiven Spielers, jeweils in der aktiven Saison).
func visibilityFilter(claims *auth.Claims) (string, []any) {
	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		return "", nil
	}
	frag := `EXISTS (
		SELECT 1 FROM video_teams vt_vis
		WHERE vt_vis.video_id = v.id AND vt_vis.team_id IN (
			SELECT pm.team_id FROM player_memberships pm
			JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
			JOIN members m ON m.id = pm.member_id AND m.status = 'aktiv'
			WHERE m.user_id = ?
			UNION
			SELECT tm.team_id FROM trainer_memberships tm
			JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
			JOIN members m ON m.id = tm.member_id
			WHERE m.user_id = ?
			UNION
			SELECT pm.team_id FROM family_links fl
			JOIN members m ON m.id = fl.member_id AND m.status = 'aktiv'
			JOIN player_memberships pm ON pm.member_id = m.id
			JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
			WHERE fl.parent_user_id = ?
		)
	)`
	return frag, []any{claims.UserID, claims.UserID, claims.UserID}
}

// loadTeamsByVideoIDs lädt die zugeordneten Teams für eine Menge von Video-IDs
// in einer einzigen Abfrage und gibt eine Map video_id → []teamRef zurück.
func (h *Handler) loadTeamsByVideoIDs(videoIDs []int) map[int][]teamRef {
	result := make(map[int][]teamRef, len(videoIDs))
	if len(videoIDs) == 0 {
		return result
	}
	placeholders := strings.Repeat("?,", len(videoIDs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(videoIDs))
	for i, id := range videoIDs {
		args[i] = id
	}
	rows, err := h.db.Query(`
		SELECT vt.video_id, vt.team_id, t.name
		FROM video_teams vt JOIN teams t ON t.id = vt.team_id
		WHERE vt.video_id IN (`+placeholders+`)
		ORDER BY vt.video_id, t.name`, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var vid int
		var ref teamRef
		if err := rows.Scan(&vid, &ref.ID, &ref.Name); err != nil {
			continue
		}
		result[vid] = append(result[vid], ref)
	}
	return result
}

// List liefert die für den Aufrufer sichtbaren Videos, paginiert.
// GET /api/videos?team_id=&status=&limit=&offset= (Authenticated-Tier).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var where []string
	var args []any

	if frag, fragArgs := visibilityFilter(claims); frag != "" {
		where = append(where, frag)
		args = append(args, fragArgs...)
	}
	if v := r.URL.Query().Get("team_id"); v != "" {
		teamID, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid team_id", http.StatusBadRequest)
			return
		}
		where = append(where, "EXISTS (SELECT 1 FROM video_teams vt_f WHERE vt_f.video_id = v.id AND vt_f.team_id = ?)")
		args = append(args, teamID)
	}
	if v := r.URL.Query().Get("status"); v != "" {
		where = append(where, "v.status = ?")
		args = append(args, v)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM videos v `+whereClause, args...).Scan(&total); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := h.db.Query(`
		SELECT v.id, v.title, v.description, v.team_id, t.name, v.season_id,
		       v.game_id, v.status, v.duration_sec, v.created_by, v.created_at, v.ready_at
		FROM videos v
		JOIN teams t ON t.id = v.team_id
		`+whereClause+`
		ORDER BY v.created_at DESC, v.id DESC
		LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []videoListItem{}
	for rows.Next() {
		var it videoListItem
		var desc sql.NullString
		var gameID, durationSec sql.NullInt64
		var readyAt sql.NullString
		if err := rows.Scan(&it.ID, &it.Title, &desc, &it.TeamID, &it.TeamName,
			&it.SeasonID, &gameID, &it.Status, &durationSec, &it.CreatedBy,
			&it.CreatedAt, &readyAt); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if desc.Valid {
			it.Description = &desc.String
		}
		if gameID.Valid {
			g := int(gameID.Int64)
			it.GameID = &g
		}
		if durationSec.Valid {
			d := int(durationSec.Int64)
			it.DurationSec = &d
		}
		if readyAt.Valid {
			it.ReadyAt = &readyAt.String
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Teams für alle geladenen Videos in einer Abfrage nachladen.
	if len(items) > 0 {
		ids := make([]int, len(items))
		for i, it := range items {
			ids[i] = it.ID
		}
		teamsMap := h.loadTeamsByVideoIDs(ids)
		for i := range items {
			if ts := teamsMap[items[i].ID]; ts != nil {
				items[i].Teams = ts
			} else {
				items[i].Teams = []teamRef{}
			}
		}
	}

	writeJSON(w, map[string]any{"items": items, "total": total})
}

// Get liefert die Detail-Daten eines Videos. 404 sowohl wenn das Video nicht
// existiert als auch wenn es für den Aufrufer nicht sichtbar ist (keine
// Existenz-Leakage). GET /api/videos/{id} (Authenticated-Tier).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var d videoDetail
	var desc, readyAt, uploadID, failureReason sql.NullString
	var gameID, durationSec, sizeBytes sql.NullInt64
	err = h.db.QueryRow(`
		SELECT v.id, v.title, v.description, v.team_id, t.name, v.season_id,
		       v.game_id, v.status, v.duration_sec, v.created_by, v.created_at, v.ready_at,
		       v.upload_id, v.size_bytes, v.failure_reason
		FROM videos v
		JOIN teams t ON t.id = v.team_id
		WHERE v.id = ?`, id).Scan(
		&d.ID, &d.Title, &desc, &d.TeamID, &d.TeamName, &d.SeasonID,
		&gameID, &d.Status, &durationSec, &d.CreatedBy, &d.CreatedAt, &readyAt,
		&uploadID, &sizeBytes, &failureReason)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Teams für Sichtbarkeitsprüfung laden.
	teamsMap := h.loadTeamsByVideoIDs([]int{id})
	d.Teams = teamsMap[id]
	if d.Teams == nil {
		d.Teams = []teamRef{}
	}

	// video.TeamIDs für CanViewVideo befüllen.
	teamIDs := make([]int, len(d.Teams))
	for i, tr := range d.Teams {
		teamIDs[i] = tr.ID
	}
	ok, err := h.CanViewVideo(claims, &Video{ID: d.ID, TeamID: d.TeamID, TeamIDs: teamIDs})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		// Existenz nicht preisgeben: 404 statt 403.
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if desc.Valid {
		d.Description = &desc.String
	}
	if gameID.Valid {
		g := int(gameID.Int64)
		d.GameID = &g
	}
	if durationSec.Valid {
		dv := int(durationSec.Int64)
		d.DurationSec = &dv
	}
	if readyAt.Valid {
		d.ReadyAt = &readyAt.String
	}
	if uploadID.Valid {
		d.UploadID = &uploadID.String
	}
	if sizeBytes.Valid {
		d.SizeBytes = &sizeBytes.Int64
	}
	if failureReason.Valid {
		d.FailureReason = &failureReason.String
	}

	writeJSON(w, d)
}

// patchVideoReq beschreibt die im PATCH änderbaren Felder. Alle Felder sind
// optional (Pointer/Set-Flag) — nur gesetzte Felder werden aktualisiert.
// game_id ist tri-state: fehlt = unverändert, null = NULL setzen, Zahl = setzen.
// team_ids ersetzt die gesamte video_teams-Zuordnung wenn gesetzt (mind. 1 Team).
type patchVideoReq struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	GameID      *int    `json:"game_id"`
	gameIDSet   bool
	TeamIDs     []int `json:"team_ids"`
	teamIDsSet  bool
}

// Update ändert Titel/Beschreibung/game_id/team_ids eines Videos.
// PATCH /api/videos/{id} (Authenticated-Tier; CanManageTeamVideos erzwungen).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	video, err := h.loadVideoForView(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if video == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	ok, err := h.CanManageTeamVideos(claims, video.TeamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Rohes Decode, um die Anwesenheit von game_id (tri-state) zu erkennen.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	var req patchVideoReq
	if v, ok := raw["title"]; ok {
		if err := json.Unmarshal(v, &req.Title); err != nil {
			http.Error(w, "invalid title", http.StatusBadRequest)
			return
		}
	}
	if v, ok := raw["description"]; ok {
		if err := json.Unmarshal(v, &req.Description); err != nil {
			http.Error(w, "invalid description", http.StatusBadRequest)
			return
		}
	}
	if v, ok := raw["game_id"]; ok {
		req.gameIDSet = true
		if err := json.Unmarshal(v, &req.GameID); err != nil {
			http.Error(w, "invalid game_id", http.StatusBadRequest)
			return
		}
	}
	if v, ok := raw["team_ids"]; ok {
		req.teamIDsSet = true
		if err := json.Unmarshal(v, &req.TeamIDs); err != nil {
			http.Error(w, "invalid team_ids", http.StatusBadRequest)
			return
		}
		if len(req.TeamIDs) == 0 {
			http.Error(w, "team_ids must not be empty", http.StatusBadRequest)
			return
		}
	}

	var sets []string
	var args []any
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			http.Error(w, "title must not be empty", http.StatusBadRequest)
			return
		}
		sets = append(sets, "title = ?")
		args = append(args, title)
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		sets = append(sets, "description = ?")
		args = append(args, desc)
	}
	if req.gameIDSet {
		if req.GameID == nil {
			sets = append(sets, "game_id = NULL")
		} else {
			sets = append(sets, "game_id = ?")
			args = append(args, *req.GameID)
		}
	}

	if len(sets) == 0 && !req.teamIDsSet {
		http.Error(w, "no fields to update", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback() //nolint:errcheck

	if len(sets) > 0 {
		qArgs := append(args, id)
		if _, err := tx.Exec(`UPDATE videos SET `+strings.Join(sets, ", ")+` WHERE id = ?`, qArgs...); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	if req.teamIDsSet {
		if _, err := tx.Exec(`DELETE FROM video_teams WHERE video_id = ?`, id); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		for _, tid := range req.TeamIDs {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO video_teams (video_id, team_id) VALUES (?, ?)`, id, tid); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("video-updated")
	w.WriteHeader(http.StatusOK)
}

// Delete entfernt ein Video samt aller Dateien (raw + processed).
// DELETE /api/videos/{id} (Authenticated-Tier; CanManageTeamVideos erzwungen).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	video, err := h.loadVideoForView(id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if video == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	ok, err := h.CanManageTeamVideos(claims, video.TeamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if _, err := h.db.Exec(`DELETE FROM videos WHERE id = ?`, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Dateien aufräumen (best effort). os.RemoveAll ignoriert nicht-existente Pfade.
	root := h.cfg.VideoStorageDir
	_ = os.RemoveAll(RawPath(root, id))
	_ = os.RemoveAll(ProcessedDir(root, id))

	h.hub.Broadcast("video-deleted")
	w.WriteHeader(http.StatusOK)
}

// writeJSON serialisiert v als JSON-Response mit 200.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// parseIntDefault parst s als int; bei leerem/ungültigem Wert gilt def.
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
