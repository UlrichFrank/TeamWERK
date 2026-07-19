package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/push"
)

// PushWithBadgeFn matches push.SendToUserWithBadge; injectable for tests.
type PushWithBadgeFn func(db *sql.DB, cfg *appconfig.Config, userID int, title, body, url string, badge int)

type Handler struct {
	db     *sql.DB
	hub    *hub.EventHub
	cfg    *appconfig.Config
	pushFn PushWithBadgeFn
}

var allowedEmojiOrder = []string{"👍", "👎", "❤️", "😂", "😮", "😢", "🙌", "🔥"}

var allowedEmojis = func() map[string]bool {
	m := map[string]bool{}
	for _, e := range allowedEmojiOrder {
		m[e] = true
	}
	return m
}()

type messageReaction struct {
	Emoji      string   `json:"emoji"`
	Count      int      `json:"count"`
	UserNames  []string `json:"userNames"`
	MyReaction bool     `json:"myReaction"`
}

func NewHandler(db *sql.DB, h *hub.EventHub, cfg *appconfig.Config) *Handler {
	return &Handler{db: db, hub: h, cfg: cfg, pushFn: push.SendToUserWithBadge}
}

// SetPushFn replaces the push function (test seam).
func (h *Handler) SetPushFn(fn PushWithBadgeFn) {
	h.pushFn = fn
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

	inCircle := false
	if claims.Role != "admin" && !claims.HasFunction("vorstand") {
		inCircle, err = h.callerInTrainerCircle(r.Context(), claims)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	switch {
	case claims.Role == "admin" || claims.HasFunction("vorstand"):
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT id, first_name || ' ' || last_name AS name FROM users
			WHERE id != ? AND (first_name || ' ' || last_name LIKE ? OR email LIKE ?)
			ORDER BY first_name, last_name LIMIT 50`, claims.UserID, q, q)
	case inCircle:
		// Zugriffskreis-Mitglied: User mit gemeinsamem Team ∪ gesamter Zugriffskreis.
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT id, name FROM (
				SELECT u.id AS id, u.first_name || ' ' || u.last_name AS name
				FROM users u
				JOIN user_accessible_teams uat ON uat.user_id = u.id
				WHERE u.id != ?
				  AND (u.first_name || ' ' || u.last_name LIKE ? OR u.email LIKE ?)
				  AND uat.team_id IN (
				    SELECT team_id FROM user_accessible_teams WHERE user_id = ?
				  )
				UNION
				SELECT user_id AS id, name FROM (`+trainerCircleMemberQuery()+`)
				WHERE user_id != ? AND name LIKE ?
			)
			ORDER BY name LIMIT 50`, claims.UserID, q, q, claims.UserID, claims.UserID, q)
	default:
		rows, err = h.db.QueryContext(r.Context(), `
			SELECT DISTINCT u.id, u.first_name || ' ' || u.last_name AS name FROM users u
			JOIN user_accessible_teams uat ON uat.user_id = u.id
			WHERE u.id != ?
			  AND (u.first_name || ' ' || u.last_name LIKE ? OR u.email LIKE ?)
			  AND uat.team_id IN (
			    SELECT team_id FROM user_accessible_teams WHERE user_id = ?
			  )
			ORDER BY u.first_name, u.last_name LIMIT 50`, claims.UserID, q, q, claims.UserID)
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
		SELECT u.id, u.first_name || ' ' || u.last_name FROM users u
		JOIN conversation_members cm ON cm.user_id = u.id
		WHERE cm.conversation_id = ? AND cm.left_at IS NULL
		ORDER BY u.first_name, u.last_name`, convID)
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

	// Find existing conversation where target is still active (target.left_at IS NULL).
	// Caller may have left_at set — no constraint on caller's left_at.
	var existingID int
	err = h.db.QueryRowContext(r.Context(), `
		SELECT c.id FROM conversations c
		JOIN conversation_members m1 ON m1.conversation_id = c.id AND m1.user_id = ?
		JOIN conversation_members m2 ON m2.conversation_id = c.id AND m2.user_id = ? AND m2.left_at IS NULL
		WHERE c.type = 'direct' LIMIT 1`,
		claims.UserID, targetUserID).Scan(&existingID)
	if err == nil {
		// Re-join if caller had previously left
		res, _ := h.db.ExecContext(r.Context(),
			`UPDATE conversation_members SET left_at = NULL
			 WHERE conversation_id = ? AND user_id = ? AND left_at IS NOT NULL`,
			existingID, claims.UserID)
		if n, _ := res.RowsAffected(); n > 0 {
			h.hub.BroadcastToUser(targetUserID, fmt.Sprintf("chat:new-message:%d", existingID))
		}
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

	h.hub.BroadcastToUser(targetUserID, fmt.Sprintf("chat:new-message:%d", convID))

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
	// Zugriffskreis (Trainer/Vorstand/sL/Beisitzer): zwei Mitglieder dürfen sich
	// teamübergreifend kontaktieren. Caller-Funktionen aus Claims, Ziel aus DB.
	callerInCircle, err := h.callerInTrainerCircle(r.Context(), claims)
	if err != nil {
		return false, err
	}
	if callerInCircle {
		targetInCircle, err := h.isInTrainerCircle(r.Context(), targetUserID)
		if err != nil {
			return false, err
		}
		if targetInCircle {
			return true, nil
		}
	}
	var count int
	err = h.db.QueryRowContext(r.Context(), `
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

// messagePreviewLen ist die maximale Zeichenzahl (runes) des Body-Previews in
// der Nachrichtenliste. Der Volltext wird bei Bedarf über den Einzel-Pfad
// (GET /api/chat/messages/{id}) nachgeladen.
const messagePreviewLen = 280

// previewBody kürzt body rune-genau auf messagePreviewLen und meldet, ob
// gekürzt wurde. Leerer body (u. a. gelöschte Nachrichten) → ("", false).
func previewBody(body string) (string, bool) {
	if utf8.RuneCountInString(body) <= messagePreviewLen {
		return body, false
	}
	runes := []rune(body)
	return string(runes[:messagePreviewLen]), true
}

// messageSelect ist der gemeinsame SELECT-Rumpf für alle ListMessages-Varianten
// (voll, ?after=, ?before=); nur WHERE/ORDER/LIMIT unterscheiden sich. Die
// vierte Spalte ist der (bei gelöschten Nachrichten leere) Body, aus dem der
// Handler rune-genau den Preview (messagePreviewLen) ableitet.
// mediaURL baut den relativen Abrufpfad eines Bildes (ohne /api-Prefix; der
// axios-Client im Frontend ergänzt ihn — Konvention wie bei Match-Report-Bildern).
func mediaURL(id int) string {
	return fmt.Sprintf("/media/%d", id)
}

const messageSelect = `
	SELECT m.id, m.sender_id, u.first_name || ' ' || u.last_name,
	       CASE WHEN m.deleted_at IS NOT NULL THEN '' ELSE m.body END,
	       m.sent_at,
	       m.reply_to_id,
	       CASE
	         WHEN m.reply_to_id IS NULL THEN NULL
	         WHEN rm.deleted_at IS NOT NULL THEN '[Nachricht gelöscht]'
	         ELSE rm.body
	       END AS reply_to_body,
	       CASE WHEN m.reply_to_id IS NOT NULL THEN ru.first_name || ' ' || ru.last_name ELSE NULL END,
	       m.edited_at,
	       m.deleted_at,
	       m.is_system,
	       m.media_id,
	       med.width,
	       med.height
	FROM messages m
	JOIN users u ON u.id = m.sender_id
	LEFT JOIN messages rm ON rm.id = m.reply_to_id
	LEFT JOIN users ru ON ru.id = rm.sender_id
	LEFT JOIN media med ON med.id = m.media_id
	WHERE m.conversation_id = ?`

// messagePageSize begrenzt jede ListMessages-Antwort (voll, after, before).
const messagePageSize = 100

// GET /api/chat/conversations/{id}/messages[?after=<msgId>|?before=<msgId>]
//
// Die Liste liefert je Nachricht nur einen gekürzten Preview (≤ messagePreviewLen
// Zeichen) plus truncated-Flag; der Volltext wird bei Bedarf über
// GET /api/chat/messages/{id} nachgeladen.
//
// Inkrementeller Sync (id-Cursor, append-only — kein updated_at nötig):
//   - ?after=<msgId>  → nur Nachrichten mit id > msgId, aufsteigend (Delta-Nachladen).
//   - ?before=<msgId> → Seite der Nachrichten unmittelbar vor msgId (Verlaufs-Scroll).
//   - ohne Parameter  → letzte messagePageSize Nachrichten, älteste zuerst.
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	afterStr := r.URL.Query().Get("after")
	beforeStr := r.URL.Query().Get("before")
	if afterStr != "" && beforeStr != "" {
		http.Error(w, "after and before are mutually exclusive", http.StatusBadRequest)
		return
	}
	parseCursor := func(s string) (int, bool) {
		v, convErr := strconv.Atoi(s)
		return v, convErr == nil && v >= 0
	}

	if !h.isMember(r, convID, claims.UserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Preview statt Volltext: die Liste liefert höchstens messagePreviewLen
	// Zeichen (rune-genau) plus truncated-Flag. Der Volltext wird bei Bedarf
	// über GET /api/chat/messages/{id} nachgeladen. Gelöschte Nachrichten
	// liefern weder Preview noch Body (leerer Preview, truncated=false).
	type Message struct {
		ID                int               `json:"id"`
		SenderID          int               `json:"senderId"`
		SenderName        string            `json:"senderName"`
		Preview           string            `json:"preview"`
		Truncated         bool              `json:"truncated"`
		SentAt            string            `json:"sentAt"`
		ReplyToID         *int              `json:"replyToId"`
		ReplyToBody       *string           `json:"replyToBody"`
		ReplyToSenderName *string           `json:"replyToSenderName"`
		EditedAt          *string           `json:"editedAt"`
		DeletedAt         *string           `json:"deletedAt"`
		IsSystem          bool              `json:"isSystem"`
		MediaID           *int              `json:"mediaId"`
		MediaURL          *string           `json:"mediaUrl"`
		MediaWidth        *int              `json:"mediaWidth,omitempty"`
		MediaHeight       *int              `json:"mediaHeight,omitempty"`
		Reactions         []messageReaction `json:"reactions"`
	}

	var rows *sql.Rows
	newestFirst := false // true, wenn die Query absteigend sortiert → vor Antwort umdrehen
	switch {
	case afterStr != "":
		after, ok := parseCursor(afterStr)
		if !ok {
			http.Error(w, "invalid after", http.StatusBadRequest)
			return
		}
		rows, err = h.db.QueryContext(r.Context(),
			messageSelect+` AND m.id > ? ORDER BY m.id ASC LIMIT ?`,
			convID, after, messagePageSize)
	case beforeStr != "":
		before, ok := parseCursor(beforeStr)
		if !ok {
			http.Error(w, "invalid before", http.StatusBadRequest)
			return
		}
		newestFirst = true
		rows, err = h.db.QueryContext(r.Context(),
			messageSelect+` AND m.id < ? ORDER BY m.id DESC LIMIT ?`,
			convID, before, messagePageSize)
	default:
		newestFirst = true
		// id als Tie-Breaker: sent_at hat Sekundengranularität, gleiche
		// Timestamps wären sonst instabil sortiert.
		rows, err = h.db.QueryContext(r.Context(),
			messageSelect+` ORDER BY m.sent_at DESC, m.id DESC LIMIT ?`,
			convID, messagePageSize)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	msgs := []Message{}
	for rows.Next() {
		var msg Message
		var body string
		var replyToID, mediaID, mediaWidth, mediaHeight sql.NullInt64
		var replyToBody, replyToSenderName, editedAt, deletedAt sql.NullString
		rows.Scan(&msg.ID, &msg.SenderID, &msg.SenderName, &body, &msg.SentAt,
			&replyToID, &replyToBody, &replyToSenderName, &editedAt, &deletedAt, &msg.IsSystem, &mediaID, &mediaWidth, &mediaHeight)
		if mediaID.Valid {
			id := int(mediaID.Int64)
			msg.MediaID = &id
			url := mediaURL(id)
			msg.MediaURL = &url
			if mediaWidth.Valid && mediaHeight.Valid {
				w := int(mediaWidth.Int64)
				h := int(mediaHeight.Int64)
				msg.MediaWidth = &w
				msg.MediaHeight = &h
			}
		}
		// Body ist bei gelöschten Nachrichten bereits '' (SQL-CASE) → Preview leer,
		// truncated=false. Sonst rune-genau auf messagePreviewLen kürzen.
		msg.Preview, msg.Truncated = previewBody(body)
		if replyToID.Valid {
			id := int(replyToID.Int64)
			msg.ReplyToID = &id
		}
		if replyToBody.Valid {
			msg.ReplyToBody = &replyToBody.String
		}
		if replyToSenderName.Valid {
			msg.ReplyToSenderName = &replyToSenderName.String
		}
		if editedAt.Valid {
			msg.EditedAt = &editedAt.String
		}
		if deletedAt.Valid {
			msg.DeletedAt = &deletedAt.String
		}
		msgs = append(msgs, msg)
	}

	// Reverse so oldest first (nur nötig, wenn absteigend gelesen wurde)
	if newestFirst {
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
	}

	// Attach emoji reactions
	if len(msgs) > 0 {
		placeholders := make([]string, len(msgs))
		ids := make([]any, len(msgs))
		for i, m := range msgs {
			placeholders[i] = "?"
			ids[i] = m.ID
		}
		type reactionRow struct {
			msgID  int
			emoji  string
			name   string
			userID int
		}
		rrows, rerr := h.db.QueryContext(r.Context(), fmt.Sprintf(
			`SELECT mr.message_id, mr.emoji, u.first_name||' '||u.last_name, mr.user_id
			 FROM message_reactions mr JOIN users u ON u.id = mr.user_id
			 WHERE mr.message_id IN (%s)
			 ORDER BY mr.message_id, mr.created_at`, strings.Join(placeholders, ",")), ids...)
		if rerr == nil {
			defer rrows.Close()
			type reactionKey struct {
				msgID int
				emoji string
			}
			type reactionAcc struct {
				names []string
				hasMe bool
			}
			rmap := map[reactionKey]*reactionAcc{}
			for rrows.Next() {
				var rr reactionRow
				rrows.Scan(&rr.msgID, &rr.emoji, &rr.name, &rr.userID)
				key := reactionKey{rr.msgID, rr.emoji}
				if rmap[key] == nil {
					rmap[key] = &reactionAcc{}
				}
				rmap[key].names = append(rmap[key].names, rr.name)
				if rr.userID == claims.UserID {
					rmap[key].hasMe = true
				}
			}
			for i := range msgs {
				for _, emoji := range allowedEmojiOrder {
					key := reactionKey{msgs[i].ID, emoji}
					if acc, ok := rmap[key]; ok {
						msgs[i].Reactions = append(msgs[i].Reactions, messageReaction{
							Emoji:      emoji,
							Count:      len(acc.names),
							UserNames:  acc.names,
							MyReaction: acc.hasMe,
						})
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

// GET /api/chat/messages/{id}
//
// Einzel-Pfad für den Nachrichten-Volltext (die Liste liefert nur den Preview).
// Sichtbarkeit unverändert: nur Mitglieder der Konversation lesen; gelöschte
// Nachrichten liefern keinen Body.
func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	msgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var convID int
	var body string
	var deletedAt sql.NullString
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT conversation_id, body, deleted_at FROM messages WHERE id = ?`, msgID).
		Scan(&convID, &body, &deletedAt); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if !h.isMember(r, convID, claims.UserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Gelöschte Nachricht → kein Body.
	if deletedAt.Valid {
		body = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":      msgID,
		"body":    body,
		"deleted": deletedAt.Valid,
	})
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
		Body      string `json:"body"`
		ReplyToID *int   `json:"replyToId"`
		MediaID   *int   `json:"mediaId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	body.Body = strings.TrimSpace(body.Body)
	// Mindestens nicht-leerer Text ODER ein Bild.
	if body.Body == "" && body.MediaID == nil {
		http.Error(w, "body or mediaId required", http.StatusBadRequest)
		return
	}

	var replyToID sql.NullInt64
	if body.ReplyToID != nil {
		var count int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM messages WHERE id = ? AND conversation_id = ?`,
			*body.ReplyToID, convID).Scan(&count)
		if count == 0 {
			http.Error(w, "invalid replyToId", http.StatusBadRequest)
			return
		}
		replyToID = sql.NullInt64{Int64: int64(*body.ReplyToID), Valid: true}
	}

	var mediaID sql.NullInt64
	if body.MediaID != nil {
		var count int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM media WHERE id = ?`, *body.MediaID).Scan(&count)
		if count == 0 {
			http.Error(w, "invalid mediaId", http.StatusBadRequest)
			return
		}
		mediaID = sql.NullInt64{Int64: int64(*body.MediaID), Valid: true}
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO messages (conversation_id, sender_id, body, reply_to_id, media_id) VALUES (?, ?, ?, ?, ?)`,
		convID, claims.UserID, body.Body, replyToID, mediaID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	msgID, _ := res.LastInsertId()

	// For direct chats: restore any member who had left so they receive the SSE
	var convType string
	var convName sql.NullString
	h.db.QueryRowContext(r.Context(), `SELECT type, name FROM conversations WHERE id = ?`, convID).Scan(&convType, &convName)
	if convType == "direct" {
		h.db.ExecContext(r.Context(),
			`UPDATE conversation_members SET left_at = NULL WHERE conversation_id = ? AND left_at IS NOT NULL`,
			convID)
	}

	recipientIDs := h.activeMembers(r, convID, 0)
	event := fmt.Sprintf("chat:new-message:%d", convID)
	for _, uid := range recipientIDs {
		h.hub.BroadcastToUser(uid, event)
	}

	pushRecipients := push.FilterByPushPref(h.db, h.activeMembers(r, convID, claims.UserID), "chat")
	title := h.senderName(r, claims.UserID, claims.Email)
	preview := truncate(body.Body, 80)
	if preview == "" {
		preview = "Bild"
	}
	if convType == "group" && strings.TrimSpace(convName.String) != "" {
		preview = strings.TrimSpace(convName.String) + "\n" + preview
	}
	for _, uid := range pushRecipients {
		badge, err := ComputeUnreadForUser(h.db, uid)
		if err != nil {
			slog.Error("chat compute unread failed", "user", uid, "error", err)
			badge = 0
		}
		go h.pushFn(h.db, h.cfg, uid, title, preview, fmt.Sprintf("/chat?conv=%d", convID), badge)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": msgID})
}

// PUT /api/chat/messages/{id}
func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	msgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Body) == "" {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}

	var convID int
	h.db.QueryRowContext(r.Context(), `SELECT conversation_id FROM messages WHERE id = ?`, msgID).Scan(&convID)

	res, err := h.db.ExecContext(r.Context(),
		`UPDATE messages SET body = ?, edited_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND sender_id = ? AND deleted_at IS NULL`,
		body.Body, msgID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if convID != 0 {
		event := fmt.Sprintf("chat:new-message:%d", convID)
		for _, uid := range h.activeMembers(r, convID, 0) {
			h.hub.BroadcastToUser(uid, event)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/chat/messages/{id}
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	msgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var senderID, convID int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT sender_id, conversation_id FROM messages WHERE id = ?`, msgID).Scan(&senderID, &convID); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if senderID != claims.UserID && claims.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	h.db.ExecContext(r.Context(),
		`UPDATE messages SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL`, msgID)

	event := fmt.Sprintf("chat:new-message:%d", convID)
	for _, uid := range h.activeMembers(r, convID, 0) {
		h.hub.BroadcastToUser(uid, event)
	}

	w.WriteHeader(http.StatusNoContent)
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

	h.hub.BroadcastToUser(claims.UserID, "chat:conversation-read")
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

	h.db.ExecContext(r.Context(),
		`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, 'hat die Gruppe verlassen', 1)`,
		convID, claims.UserID)

	event := fmt.Sprintf("chat:member-left:%d", convID)
	for _, uid := range h.activeMembers(r, convID, 0) {
		h.hub.BroadcastToUser(uid, event)
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/chat/broadcasts
func (h *Handler) ListBroadcasts(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	type Broadcast struct {
		ID          int     `json:"id"`
		SenderName  string  `json:"senderName"`
		Body        string  `json:"body"`
		SentAt      string  `json:"sentAt"`
		IsRead      bool    `json:"isRead"`
		IsSent      bool    `json:"isSent"`
		EditedAt    *string `json:"editedAt"`
		MediaID     *int    `json:"mediaId"`
		MediaURL    *string `json:"mediaUrl"`
		MediaWidth  *int    `json:"mediaWidth,omitempty"`
		MediaHeight *int    `json:"mediaHeight,omitempty"`
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT b.id, u.first_name || ' ' || u.last_name, b.body, b.sent_at,
		       CASE WHEN br.read_at IS NOT NULL THEN 1 ELSE 0 END AS is_read,
		       CASE WHEN b.sender_id = ? THEN 1 ELSE 0 END AS is_sent,
		       b.edited_at, b.media_id, med.width, med.height
		FROM broadcasts b
		JOIN users u ON u.id = b.sender_id
		JOIN broadcast_reads br ON br.broadcast_id = b.id AND br.user_id = ?
		LEFT JOIN media med ON med.id = b.media_id
		WHERE br.hidden_at IS NULL
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
		var editedAt sql.NullString
		var mediaID, mediaWidth, mediaHeight sql.NullInt64
		rows.Scan(&b.ID, &b.SenderName, &b.Body, &b.SentAt, &isRead, &isSent, &editedAt, &mediaID, &mediaWidth, &mediaHeight)
		b.IsRead = isRead == 1
		b.IsSent = isSent == 1
		if editedAt.Valid {
			b.EditedAt = &editedAt.String
		}
		if mediaID.Valid {
			id := int(mediaID.Int64)
			b.MediaID = &id
			url := mediaURL(id)
			b.MediaURL = &url
			if mediaWidth.Valid && mediaHeight.Valid {
				w := int(mediaWidth.Int64)
				h := int(mediaHeight.Int64)
				b.MediaWidth = &w
				b.MediaHeight = &h
			}
		}
		broadcasts = append(broadcasts, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(broadcasts)
}

// PUT /api/chat/broadcasts/{id}
func (h *Handler) EditBroadcast(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	broadcastID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
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
		`UPDATE broadcasts SET body = ?, edited_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND sender_id = ?`,
		body.Body, broadcastID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		MediaID    *int   `json:"mediaId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	body.Body = strings.TrimSpace(body.Body)
	// Mindestens nicht-leerer Text ODER ein Bild.
	if body.Body == "" && body.MediaID == nil {
		http.Error(w, "body or mediaId required", http.StatusBadRequest)
		return
	}
	if body.TargetType != "all" && body.TargetType != "team" && body.TargetType != "role" {
		http.Error(w, "targetType must be all, team or role", http.StatusBadRequest)
		return
	}

	var mediaID sql.NullInt64
	if body.MediaID != nil {
		var count int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM media WHERE id = ?`, *body.MediaID).Scan(&count)
		if count == 0 {
			http.Error(w, "invalid mediaId", http.StatusBadRequest)
			return
		}
		mediaID = sql.NullInt64{Int64: int64(*body.MediaID), Valid: true}
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
		`INSERT INTO broadcasts (sender_id, target_type, target_id, target_role, body, media_id) VALUES (?, ?, ?, ?, ?, ?)`,
		claims.UserID, body.TargetType, targetID, targetRole, body.Body, mediaID)
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
	pushRecipients := push.FilterByPushPref(h.db, pushIDs, "chat")
	title := h.senderName(r, claims.UserID, claims.Email)
	preview := truncate(body.Body, 80)
	if preview == "" {
		preview = "Bild"
	}
	for _, uid := range pushRecipients {
		badge, err := ComputeUnreadForUser(h.db, uid)
		if err != nil {
			slog.Error("chat compute unread failed", "user", uid, "error", err)
			badge = 0
		}
		go h.pushFn(h.db, h.cfg, uid, title, preview, "/chat?tab=broadcasts", badge)
	}

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

	h.hub.BroadcastToUser(claims.UserID, "chat:conversation-read")
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/chat/conversations/{id}
func (h *Handler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if !h.isMember(r, convID, claims.UserID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var convType string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT type FROM conversations WHERE id = ?`, convID).Scan(&convType); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	h.db.ExecContext(r.Context(),
		`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP
		 WHERE conversation_id = ? AND user_id = ? AND left_at IS NULL`,
		convID, claims.UserID)

	if convType == "direct" {
		h.db.ExecContext(r.Context(),
			`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, 'hat diesen Chat verlassen', 1)`,
			convID, claims.UserID)
		for _, uid := range h.activeMembers(r, convID, 0) {
			h.hub.BroadcastToUser(uid, fmt.Sprintf("chat:member-left:%d", convID))
		}
	}

	var remaining int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ? AND left_at IS NULL`, convID).Scan(&remaining)
	if remaining == 0 {
		h.db.ExecContext(r.Context(), `DELETE FROM conversations WHERE id = ?`, convID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/chat/broadcasts/{id}
func (h *Handler) DeleteBroadcast(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	broadcastID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`UPDATE broadcast_reads SET hidden_at = CURRENT_TIMESTAMP
		 WHERE broadcast_id = ? AND user_id = ? AND hidden_at IS NULL`,
		broadcastID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var remaining int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM broadcast_reads WHERE broadcast_id = ? AND hidden_at IS NULL`, broadcastID).Scan(&remaining)
	if remaining == 0 {
		h.db.ExecContext(r.Context(), `DELETE FROM broadcasts WHERE id = ?`, broadcastID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/chat/conversations/{id}/members
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		UserID int `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == 0 {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}

	var convType string
	var createdBy int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT type, created_by FROM conversations WHERE id = ?`, convID).Scan(&convType, &createdBy); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if convType != "group" {
		http.Error(w, "only group conversations support adding members", http.StatusBadRequest)
		return
	}
	if createdBy != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ok, err := h.canContactUser(r, claims, body.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`UPDATE conversation_members SET left_at = NULL WHERE conversation_id = ? AND user_id = ?`,
		convID, body.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := h.db.ExecContext(r.Context(),
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`,
			convID, body.UserID); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	h.db.ExecContext(r.Context(),
		`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, 'wurde hinzugefügt', 1)`,
		convID, body.UserID)

	event := fmt.Sprintf("chat:new-message:%d", convID)
	for _, uid := range h.activeMembers(r, convID, 0) {
		h.hub.BroadcastToUser(uid, event)
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/chat/conversations/{id}/members/{uid}
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	targetUserID, err := strconv.Atoi(chi.URLParam(r, "uid"))
	if err != nil {
		http.Error(w, "invalid uid", http.StatusBadRequest)
		return
	}

	var convType string
	var createdBy int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT type, created_by FROM conversations WHERE id = ?`, convID).Scan(&convType, &createdBy); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if createdBy != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if convType != "group" {
		http.Error(w, "only group conversations support removing members", http.StatusBadRequest)
		return
	}
	if targetUserID == claims.UserID {
		http.Error(w, "creator cannot remove themselves via this endpoint", http.StatusBadRequest)
		return
	}

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP
		 WHERE conversation_id = ? AND user_id = ? AND left_at IS NULL`,
		convID, targetUserID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.db.ExecContext(r.Context(),
		`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, 'wurde entfernt', 1)`,
		convID, targetUserID)

	event := fmt.Sprintf("chat:member-left:%d", convID)
	for _, uid := range h.activeMembers(r, convID, 0) {
		h.hub.BroadcastToUser(uid, event)
	}
	h.hub.BroadcastToUser(targetUserID, event)

	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/chat/conversations/{id}
func (h *Handler) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len([]rune(name)) > 100 {
		http.Error(w, "name must be 1..100 chars", http.StatusBadRequest)
		return
	}

	var convType string
	var createdBy int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT type, created_by FROM conversations WHERE id = ?`, convID).Scan(&convType, &createdBy); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if createdBy != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if convType != "group" {
		http.Error(w, "only group conversations can be renamed", http.StatusBadRequest)
		return
	}

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE conversations SET name = ? WHERE id = ?`, name, convID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	sysBody := fmt.Sprintf("hat die Gruppe in '%s' umbenannt", name)
	h.db.ExecContext(r.Context(),
		`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, ?, 1)`,
		convID, claims.UserID, sysBody)

	event := fmt.Sprintf("chat:conv-updated:%d", convID)
	for _, uid := range h.activeMembers(r, convID, 0) {
		h.hub.BroadcastToUser(uid, event)
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/chat/conversations/{id}/transfer-ownership
func (h *Handler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		NewOwnerID int `json:"newOwnerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NewOwnerID == 0 {
		http.Error(w, "newOwnerId required", http.StatusBadRequest)
		return
	}

	var convType string
	var createdBy int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT type, created_by FROM conversations WHERE id = ?`, convID).Scan(&convType, &createdBy); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if createdBy != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if convType != "group" {
		http.Error(w, "only group conversations support ownership transfer", http.StatusBadRequest)
		return
	}
	if body.NewOwnerID == claims.UserID {
		http.Error(w, "cannot transfer to self", http.StatusBadRequest)
		return
	}
	if !h.isActiveMember(r, convID, body.NewOwnerID) {
		http.Error(w, "new owner must be an active member", http.StatusBadRequest)
		return
	}

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE conversations SET created_by = ? WHERE id = ?`, body.NewOwnerID, convID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	newOwnerName := h.senderName(r, body.NewOwnerID, "")
	sysBody := fmt.Sprintf("hat die Verwaltung an %s übergeben", newOwnerName)
	h.db.ExecContext(r.Context(),
		`INSERT INTO messages (conversation_id, sender_id, body, is_system) VALUES (?, ?, ?, 1)`,
		convID, claims.UserID, sysBody)

	event := fmt.Sprintf("chat:conv-updated:%d", convID)
	for _, uid := range h.activeMembers(r, convID, 0) {
		h.hub.BroadcastToUser(uid, event)
	}

	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/chat/conversations/{id}/everyone
func (h *Handler) DeleteConversationForEveryone(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	convID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var convType string
	var createdBy int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT type, created_by FROM conversations WHERE id = ?`, convID).Scan(&convType, &createdBy); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if createdBy != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if convType != "group" {
		http.Error(w, "only group conversations can be deleted for everyone", http.StatusBadRequest)
		return
	}

	recipients := h.activeMembers(r, convID, 0)

	if _, err := h.db.ExecContext(r.Context(),
		`DELETE FROM conversations WHERE id = ?`, convID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	event := fmt.Sprintf("chat:conv-deleted:%d", convID)
	for _, uid := range recipients {
		h.hub.BroadcastToUser(uid, event)
	}
	h.hub.BroadcastToUser(claims.UserID, event)

	w.WriteHeader(http.StatusNoContent)
}

// POST /api/chat/messages/{id}/reactions
func (h *Handler) ToggleReaction(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	msgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !allowedEmojis[body.Emoji] {
		http.Error(w, "invalid emoji", http.StatusBadRequest)
		return
	}

	var convID int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT conversation_id FROM messages WHERE id = ? AND deleted_at IS NULL`, msgID).Scan(&convID); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !h.isMember(r, convID, claims.UserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM message_reactions WHERE message_id=? AND user_id=? AND emoji=?`,
		msgID, claims.UserID, body.Emoji).Scan(&count)
	if count > 0 {
		h.db.ExecContext(r.Context(),
			`DELETE FROM message_reactions WHERE message_id=? AND user_id=? AND emoji=?`,
			msgID, claims.UserID, body.Emoji)
	} else {
		h.db.ExecContext(r.Context(),
			`DELETE FROM message_reactions WHERE message_id=? AND user_id=?`,
			msgID, claims.UserID)
		h.db.ExecContext(r.Context(),
			`INSERT INTO message_reactions (message_id, user_id, emoji) VALUES (?,?,?)`,
			msgID, claims.UserID, body.Emoji)
	}

	event := fmt.Sprintf("chat:new-message:%d", convID)
	for _, uid := range h.activeMembers(r, convID, 0) {
		h.hub.BroadcastToUser(uid, event)
	}
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

func (h *Handler) senderName(r *http.Request, userID int, fallback string) string {
	var first, last string
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT first_name, last_name FROM users WHERE id = ?`, userID).Scan(&first, &last); err != nil {
		return fallback
	}
	name := strings.TrimSpace(first + " " + last)
	if name == "" {
		return fallback
	}
	return name
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
