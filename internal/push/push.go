package push

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
)

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
		log.Printf("push: query subscriptions: %v", err)
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
			log.Printf("push: send to subscription %d: %v", s.id, err)
			continue
		}
		resp.Body.Close()
		// Web Push Spec §7: remove subscriptions on permanent failure codes.
		switch resp.StatusCode {
		case http.StatusGone, http.StatusNotFound,
			http.StatusUnauthorized, http.StatusBadRequest:
			db.Exec(`DELETE FROM push_subscriptions WHERE id = ?`, s.id)
		}
	}
}
