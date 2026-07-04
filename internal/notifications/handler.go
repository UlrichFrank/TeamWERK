package notifications

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/httpcache"
	"github.com/teamstuttgart/teamwerk/internal/push"
)

type Handler struct {
	db  *sql.DB
	cfg *appconfig.Config
}

func NewHandler(db *sql.DB, cfg *appconfig.Config) *Handler {
	return &Handler{db: db, cfg: cfg}
}

// GET /api/profile/notification-preferences
func (h *Handler) GetNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	prefs := push.GetAllPreferences(h.db, claims.UserID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

// PUT /api/profile/notification-preferences
func (h *Handler) UpdateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	var body map[string]struct {
		Push  bool `json:"push"`
		Email bool `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	for category, pref := range body {
		pushVal := 0
		if pref.Push {
			pushVal = 1
		}
		emailVal := 0
		if pref.Email {
			emailVal = 1
		}
		_, err := h.db.ExecContext(r.Context(), `
			INSERT INTO notification_preferences (user_id, category, push_enabled, email_enabled)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(user_id, category) DO UPDATE SET
			  push_enabled  = excluded.push_enabled,
			  email_enabled = excluded.email_enabled`,
			claims.UserID, category, pushVal, emailVal)
		if err != nil {
			slog.Error("update notification preferences failed", "user", claims.UserID, "category", category, "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/push/vapid-public-key
func (h *Handler) GetVAPIDPublicKey(w http.ResponseWriter, r *http.Request) {
	// Der VAPID-Public-Key ist deploy-stabil (aus der Umgebung) und nicht
	// geheim → immutable-Cache; Rotation ändert den ETag und invalidiert.
	etag := httpcache.ETagFor([]byte(h.cfg.VAPIDPublicKey))
	httpcache.Serve(w, r, etag, "public, max-age=31536000, immutable", func() any {
		return map[string]string{"publicKey": h.cfg.VAPIDPublicKey}
	})
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
