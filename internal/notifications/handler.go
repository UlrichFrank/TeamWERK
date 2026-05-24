package notifications

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
)

type Handler struct {
	db  *sql.DB
	cfg *appconfig.Config
}

func NewHandler(db *sql.DB, cfg *appconfig.Config) *Handler {
	return &Handler{db: db, cfg: cfg}
}

// GET /api/push/vapid-public-key
func (h *Handler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"publicKey": h.cfg.VAPIDPublicKey})
}

// POST /api/push/subscribe
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	var body struct {
		Endpoint string `json:"endpoint"`
		P256dh   string `json:"p256dh"`
		Auth     string `json:"auth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Endpoint == "" || body.P256dh == "" || body.Auth == "" {
		http.Error(w, "endpoint, p256dh and auth are required", http.StatusBadRequest)
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET
		  user_id = excluded.user_id,
		  p256dh  = excluded.p256dh,
		  auth    = excluded.auth`,
		claims.UserID, body.Endpoint, body.P256dh, body.Auth)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/push/subscribe
func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.db.ExecContext(r.Context(),
		`DELETE FROM push_subscriptions WHERE endpoint = ? AND user_id = ?`,
		body.Endpoint, claims.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// SendToUsers sends a push notification to all subscriptions of the given users.
// Runs as fire-and-forget — call via `go SendToUsers(...)`.
func SendToUsers(db *sql.DB, cfg *appconfig.Config, userIDs []int, title, body, url string) {
	if len(userIDs) == 0 || cfg.VAPIDPrivateKey == "" {
		return
	}

	placeholders := make([]any, len(userIDs))
	inClause := "?"
	for i, id := range userIDs {
		placeholders[i] = id
		if i > 0 {
			inClause += ",?"
		}
	}

	rows, err := db.Query(
		`SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id IN (`+inClause+`)`,
		placeholders...)
	if err != nil {
		log.Printf("notifications: query subscriptions: %v", err)
		return
	}
	defer rows.Close()

	type sub struct {
		id       int
		endpoint string
		p256dh   string
		auth     string
	}
	var subs []sub
	for rows.Next() {
		var s sub
		rows.Scan(&s.id, &s.endpoint, &s.p256dh, &s.auth)
		subs = append(subs, s)
	}

	payload, _ := json.Marshal(map[string]string{
		"title": title,
		"body":  body,
		"url":   url,
	})

	for _, s := range subs {
		resp, err := webpush.SendNotification(payload, &webpush.Subscription{
			Endpoint: s.endpoint,
			Keys: webpush.Keys{
				P256dh: s.p256dh,
				Auth:   s.auth,
			},
		}, &webpush.Options{
			VAPIDPublicKey:  cfg.VAPIDPublicKey,
			VAPIDPrivateKey: cfg.VAPIDPrivateKey,
			Subscriber:      cfg.VAPIDEmail,
			TTL:             3600,
		})
		if err != nil {
			log.Printf("notifications: send to subscription %d: %v", s.id, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusGone {
			db.Exec(`DELETE FROM push_subscriptions WHERE id = ?`, s.id)
		}
	}
}
