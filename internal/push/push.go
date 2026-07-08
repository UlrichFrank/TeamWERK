package push

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
)

// sendNotification is the seam over webpush.SendNotification. Tests override it
// to drive per-status behavior (e.g. the cleanup switch) without real HTTP.
var sendNotification = webpush.SendNotification

// handlePushResponse applies Web Push §7 cleanup: delete the subscription only
// on PERMANENT failures (404 Not Found, 410 Gone). Transient failures
// (400/401 and 5xx) are logged but the subscription is RETAINED — a transient
// VAPID-signing or payload fault must never wipe a still-valid subscription.
func handlePushResponse(db *sql.DB, subID, statusCode int) {
	switch statusCode {
	case http.StatusGone, http.StatusNotFound:
		db.Exec(`DELETE FROM push_subscriptions WHERE id = ?`, subID)
	case http.StatusBadRequest, http.StatusUnauthorized:
		slog.Warn("push transient failure, keeping subscription", "subscription", subID, "status", statusCode)
	}
}

// SendToUsers sends a push notification to all subscriptions of the given users.
// Runs as fire-and-forget — call via `go push.SendToUsers(...)`.
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
		slog.Error("push query subscriptions failed", "error", err)
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
		resp, err := sendNotification(payload, &webpush.Subscription{
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
			slog.Error("push send failed", "subscription", s.id, "error", err)
			continue
		}
		resp.Body.Close()
		handlePushResponse(db, s.id, resp.StatusCode)
	}
}

// BuildBadgePayload encodes the JSON push payload with a `badge` field.
// Exported for tests; SendToUserWithBadge uses it internally.
func BuildBadgePayload(title, body, url string, badge int) []byte {
	payload, _ := json.Marshal(map[string]any{
		"title": title,
		"body":  body,
		"url":   url,
		"badge": badge,
	})
	return payload
}

// SendToUserWithBadge sends a push notification to all subscriptions of a single
// user and includes the absolute app-badge value in the payload.
// Runs as fire-and-forget — call via `go push.SendToUserWithBadge(...)`.
func SendToUserWithBadge(db *sql.DB, cfg *appconfig.Config, userID int, title, body, url string, badge int) {
	if cfg.VAPIDPrivateKey == "" {
		return
	}

	rows, err := db.Query(
		`SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?`,
		userID)
	if err != nil {
		slog.Error("push query subscriptions failed", "user", userID, "error", err)
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

	payload := BuildBadgePayload(title, body, url, badge)

	for _, s := range subs {
		resp, err := sendNotification(payload, &webpush.Subscription{
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
			slog.Error("push send failed", "subscription", s.id, "error", err)
			continue
		}
		resp.Body.Close()
		handlePushResponse(db, s.id, resp.StatusCode)
	}
}
