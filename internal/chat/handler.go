package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
	cfg *appconfig.Config
}

func NewHandler(db *sql.DB, h *hub.EventHub, cfg *appconfig.Config) *Handler {
	return &Handler{db: db, hub: h, cfg: cfg}
}

// GET /api/chat/events
func (h *Handler) ChatEvents(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.hub.SubscribeUser(claims.UserID)
	defer h.hub.UnsubscribeUser(claims.UserID, ch)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// GET /api/chat/users
func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	q := "%" + r.URL.Query().Get("q") + "%"

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var rows *sql.Rows
	var err error

	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT id, name FROM users
			WHERE id != ? AND name LIKE ?
			ORDER BY name LIMIT 50`, claims.UserID, q)
	} else {
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT DISTINCT u.id, u.name FROM users u
			JOIN user_accessible_teams uat ON uat.user_id = u.id
			WHERE u.id != ?
			  AND u.name LIKE ?
			  AND uat.team_id IN (
			    SELECT team_id FROM user_accessible_teams WHERE user_id = ?
			  )
			ORDER BY u.name LIMIT 50`, claims.UserID, q, claims.UserID)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			continue
		}
		users = append(users, u)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

type Member struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type LastMessage struct {
	Body   string `json:"body"`
	SentAt string `json:"sentAt"`
}

type Conversation struct {
	ID          int          `json:"id"`
	Type        string       `json:"type"`
	Name        *string      `json:"name"`
	CreatedBy   int          `json:"createdBy"`
	UnreadCount int          `json:"unreadCount"`
	LastMessage *LastMessage `json:"lastMessage"`
	Members     []Member     `json:"members"`
}

// GET /api/chat/conversations
func (h *Handler) ListConversations(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT c.id, c.type, c.name, c.created_by,
		  (SELECT COUNT(*) FROM messages m
		   WHERE m.conversation_id = c.id
		     AND m.sender_id != ?
		     AND NOT EXISTS (
		       SELECT 1 FROM message_reads mr WHERE mr.message_id = m.id AND mr.user_id = ?
		     )
		  ) AS unread_count,
		  (SELECT m.body FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1) AS last_body,
		  (SELECT m.sent_at FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1) AS last_at
		FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id = c.id
		WHERE cm.user_id = ? AND cm.left_at IS NULL
		ORDER BY COALESCE(
		  (SELECT m.sent_at FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1),
		  c.created_at
		) DESC`,
		claims.UserID, claims.UserID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	convs := []Conversation{}
	for rows.Next() {
		var c Conversation
		var name sql.NullString
		var lastBody, lastAt sql.NullString
		if err := rows.Scan(&c.ID, &c.Type, &name, &c.CreatedBy, &c.UnreadCount, &lastBody, &lastAt); err != nil {
			continue
		}
		if name.Valid {
			c.Name = &name.String
		}
		if lastBody.Valid && lastAt.Valid {
			c.LastMessage = &LastMessage{Body: lastBody.String, SentAt: lastAt.String}
		}
		convs = append(convs, c)
	}
	rows.Close()

	for i := range convs {
		members, err := h.loadMembers(r, convs[i].ID)
		if err == nil {
			convs[i].Members = members
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convs)
}

func (h *Handler) loadMembers(r *http.Request, convID int) ([]Member, error) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT u.id, u.name FROM users u
		JOIN conversation_members cm ON cm.user_id = u.id
		WHERE cm.conversation_id = ? AND cm.left_at IS NULL
		ORDER BY u.name`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []Member{}
	for rows.Next() {
		var m Member
		rows.Scan(&m.ID, &m.Name)
		members = append(members, m)
	}
	return members, nil
}

// POST /api/chat/conversations
func (h *Handler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	var body struct {
		Type      string `json:"type"`
		UserID    int    `json:"userId"`
		Name      string `json:"name"`
		MemberIDs []int  `json:"memberIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch body.Type {
	case "direct":
		h.createDirect(w, r, claims, body.UserID)
	case "group":
		h.createGroup(w, r, claims, body.Name, body.MemberIDs)
	default:
		http.Error(w, "type must be direct or group", http.StatusBadRequest)
	}
}


func (h *Handler) createDirect(w http.ResponseWriter, r *http.Request, claims *auth.Claims, targetUserID int) {
	if targetUserID == 0 {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}

	canContact, err := h.canContactUser(r, claims, targetUserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !canContact {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var existingID int
	err = h.db.QueryRowContext(r.Context(), `
		SELECT c.id FROM conversations c
		JOIN conversation_members m1 ON m1.conversation_id = c.id AND m1.user_id = ? AND m1.left_at IS NULL
		JOIN conversation_members m2 ON m2.conversation_id = c.id AND m2.user_id = ? AND m2.left_at IS NULL
		WHERE c.type = 'direct' LIMIT 1`,
		claims.UserID, targetUserID).Scan(&existingID)
	if err == nil {
		conv, err := h.getConversation(r, existingID, claims.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conv)
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO conversations (type, created_by) VALUES ('direct', ?)`, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	convID, _ := res.LastInsertId()

	h.db.ExecContext(r.Context(),
		`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?), (?, ?)`,
		convID, claims.UserID, convID, targetUserID)

	conv, err := h.getConversation(r, int(convID), claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conv)
}

func (h *Handler) canContactUser(r *http.Request, claims *auth.Claims, targetUserID int) (bool, error) {
	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		return true, nil
	}
	var count int
	err := h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM user_accessible_teams uat1
		JOIN user_accessible_teams uat2 ON uat1.team_id = uat2.team_id
		WHERE uat1.user_id = ? AND uat2.user_id = ?`,
		claims.UserID, targetUserID).Scan(&count)
	return count > 0, err
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request, claims *auth.Claims, name string, memberIDs []int) {
	if strings.TrimSpace(name) == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	for _, mid := range memberIDs {
		ok, err := h.canContactUser(r, claims, mid)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO conversations (type, name, created_by) VALUES ('group', ?, ?)`,
		name, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	convID, _ := res.LastInsertId()

	allMembers := append([]int{claims.UserID}, memberIDs...)
	seen := make(map[int]bool)
	for _, uid := range allMembers {
		if seen[uid] {
			continue
		}
		seen[uid] = true
		h.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`,
			convID, uid)
	}

	conv, err := h.getConversation(r, int(convID), claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conv)
}

func (h *Handler) getConversation(r *http.Request, convID, userID int) (*Conversation, error) {
	var c Conversation
	var name sql.NullString
	var lastBody, lastAt sql.NullString
	err := h.db.QueryRowContext(r.Context(), `
		SELECT c.id, c.type, c.name, c.created_by,
		  (SELECT COUNT(*) FROM messages m
		   WHERE m.conversation_id = c.id
		     AND m.sender_id != ?
		     AND NOT EXISTS (
		       SELECT 1 FROM message_reads mr WHERE mr.message_id = m.id AND mr.user_id = ?
		     )
		  ) AS unread_count,
		  (SELECT m.body FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1),
		  (SELECT m.sent_at FROM messages m WHERE m.conversation_id = c.id ORDER BY m.sent_at DESC LIMIT 1)
		FROM conversations c WHERE c.id = ?`,
		userID, userID, convID).Scan(
		&c.ID, &c.Type, &name, &c.CreatedBy, &c.UnreadCount, &lastBody, &lastAt)
	if err != nil {
		return nil, err
	}
	if name.Valid {
		c.Name = &name.String
	}
	if lastBody.Valid && lastAt.Valid {
		c.LastMessage = &LastMessage{Body: lastBody.String, SentAt: lastAt.String}
	}
	members, _ := h.loadMembers(r, convID)
	c.Members = members
	return &c, nil
}

// GET /api/chat/conversations/{id}/messages
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if !h.isMember(r, convID, claims.UserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	type Message struct {
		ID         int    `json:"id"`
		SenderID   int    `json:"senderId"`
		SenderName string `json:"senderName"`
		Body       string `json:"body"`
		SentAt     string `json:"sentAt"`
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT m.id, m.sender_id, u.name, m.body, m.sent_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.conversation_id = ?
		ORDER BY m.sent_at DESC
		LIMIT 100`, convID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	msgs := []Message{}
	for rows.Next() {
		var msg Message
		rows.Scan(&msg.ID, &msg.SenderID, &msg.SenderName, &msg.Body, &msg.SentAt)
		msgs = append(msgs, msg)
	}

	// Reverse so oldest first
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

// POST /api/chat/conversations/{id}/messages
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if !h.isActiveMember(r, convID, claims.UserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Body) == "" {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`,
		convID, claims.UserID, body.Body)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	msgID, _ := res.LastInsertId()

	recipientIDs := h.activeMembers(r, convID, claims.UserID)
	event := fmt.Sprintf("chat:new-message:%d", convID)
	for _, uid := range recipientIDs {
		h.hub.BroadcastToUser(uid, event)
	}

	go notifications.SendToUsers(h.db, h.cfg, recipientIDs,
		claims.Email, truncate(body.Body, 80), "/chat")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": msgID})
}

// POST /api/chat/conversations/{id}/read
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	h.db.ExecContext(r.Context(), `
		INSERT OR IGNORE INTO message_reads (message_id, user_id)
		SELECT m.id, ? FROM messages m
		WHERE m.conversation_id = ? AND m.sender_id != ?`,
		claims.UserID, convID, claims.UserID)

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/chat/conversations/{id}/members/me
func (h *Handler) LeaveConversation(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var convType string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT type FROM conversations WHERE id = ?`, convID).Scan(&convType); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if convType != "group" {
		http.Error(w, "cannot leave direct conversation", http.StatusBadRequest)
		return
	}

	h.db.ExecContext(r.Context(),
		`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP
		 WHERE conversation_id = ? AND user_id = ? AND left_at IS NULL`,
		convID, claims.UserID)

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/chat/broadcasts
func (h *Handler) ListBroadcasts(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	type Broadcast struct {
		ID         int    `json:"id"`
		SenderName string `json:"senderName"`
		Body       string `json:"body"`
		SentAt     string `json:"sentAt"`
		IsRead     bool   `json:"isRead"`
		IsSent     bool   `json:"isSent"`
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT b.id, u.name, b.body, b.sent_at,
		       CASE WHEN br.read_at IS NOT NULL THEN 1 ELSE 0 END AS is_read,
		       CASE WHEN b.sender_id = ? THEN 1 ELSE 0 END AS is_sent
		FROM broadcasts b
		JOIN users u ON u.id = b.sender_id
		JOIN broadcast_reads br ON br.broadcast_id = b.id AND br.user_id = ?
		ORDER BY b.sent_at DESC
		LIMIT 100`, claims.UserID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	broadcasts := []Broadcast{}
	for rows.Next() {
		var b Broadcast
		var isRead, isSent int
		rows.Scan(&b.ID, &b.SenderName, &b.Body, &b.SentAt, &isRead, &isSent)
		b.IsRead = isRead == 1
		b.IsSent = isSent == 1
		broadcasts = append(broadcasts, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(broadcasts)
}

// POST /api/chat/broadcasts
func (h *Handler) SendBroadcast(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	if !claims.HasFunction("vorstand") && !claims.IsTrainerLike() && claims.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		Body       string `json:"body"`
		TargetType string `json:"targetType"`
		TargetID   int    `json:"targetId"`
		TargetRole string `json:"targetRole"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Body) == "" {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}
	if body.TargetType != "all" && body.TargetType != "team" && body.TargetType != "role" {
		http.Error(w, "targetType must be all, team or role", http.StatusBadRequest)
		return
	}

	// Trainer may only send to their own team
	if claims.IsTrainerLike() && !claims.HasFunction("vorstand") && claims.Role != "admin" {
		if body.TargetType != "team" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var count int
		h.db.QueryRowContext(r.Context(), `
			SELECT COUNT(*) FROM kader_trainers kt
			JOIN kader k ON k.id = kt.kader_id
			JOIN members m ON m.id = kt.member_id
			WHERE m.user_id = ? AND k.team_id = ?`,
			claims.UserID, body.TargetID).Scan(&count)
		if count == 0 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	var targetID sql.NullInt64
	if body.TargetType == "team" && body.TargetID > 0 {
		targetID = sql.NullInt64{Int64: int64(body.TargetID), Valid: true}
	}
	var targetRole sql.NullString
	if body.TargetType == "role" && body.TargetRole != "" {
		targetRole = sql.NullString{String: body.TargetRole, Valid: true}
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO broadcasts (sender_id, target_type, target_id, target_role, body) VALUES (?, ?, ?, ?, ?)`,
		claims.UserID, body.TargetType, targetID, targetRole, body.Body)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	broadcastID, _ := res.LastInsertId()

	recipientIDs := h.resolveBroadcastRecipients(r, body.TargetType, body.TargetID, body.TargetRole)

	// Mark as read for sender immediately, unread for others
	senderIncluded := false
	for _, uid := range recipientIDs {
		var readAt sql.NullString
		if uid == claims.UserID {
			readAt = sql.NullString{String: "now", Valid: true}
			senderIncluded = true
		}
		if readAt.Valid {
			h.db.ExecContext(r.Context(),
				`INSERT OR IGNORE INTO broadcast_reads (broadcast_id, user_id, read_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
				broadcastID, uid)
		} else {
			h.db.ExecContext(r.Context(),
				`INSERT OR IGNORE INTO broadcast_reads (broadcast_id, user_id) VALUES (?, ?)`,
				broadcastID, uid)
		}
	}
	if !senderIncluded {
		h.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO broadcast_reads (broadcast_id, user_id, read_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
			broadcastID, claims.UserID)
	}

	// SSE + Push for non-sender recipients
	pushIDs := []int{}
	for _, uid := range recipientIDs {
		if uid != claims.UserID {
			h.hub.BroadcastToUser(uid, "chat:new-broadcast")
			pushIDs = append(pushIDs, uid)
		}
	}
	go notifications.SendToUsers(h.db, h.cfg, pushIDs, claims.Email, truncate(body.Body, 80), "/chat")

	w.WriteHeader(http.StatusCreated)
}

// POST /api/chat/broadcasts/{id}/read
func (h *Handler) MarkBroadcastRead(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	broadcastID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	h.db.ExecContext(r.Context(), `
		UPDATE broadcast_reads SET read_at = CURRENT_TIMESTAMP
		WHERE broadcast_id = ? AND user_id = ? AND read_at IS NULL`,
		broadcastID, claims.UserID)

	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func (h *Handler) isMember(r *http.Request, convID, userID int) bool {
	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ?`,
		convID, userID).Scan(&count)
	return count > 0
}

func (h *Handler) isActiveMember(r *http.Request, convID, userID int) bool {
	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND user_id = ? AND left_at IS NULL`,
		convID, userID).Scan(&count)
	return count > 0
}

func (h *Handler) activeMembers(r *http.Request, convID, excludeUserID int) []int {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT user_id FROM conversation_members WHERE conversation_id = ? AND left_at IS NULL AND user_id != ?`,
		convID, excludeUserID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func (h *Handler) resolveBroadcastRecipients(r *http.Request, targetType string, targetID int, targetRole string) []int {
	var rows *sql.Rows
	var err error
	switch targetType {
	case "all":
		rows, err = h.db.QueryContext(r.Context(), `SELECT id FROM users`)
	case "team":
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT DISTINCT user_id FROM user_accessible_teams WHERE team_id = ?`, targetID)
	case "role":
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT id FROM users WHERE role = ?`, targetRole)
	default:
		return nil
	}
	if err != nil {
		return nil
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
