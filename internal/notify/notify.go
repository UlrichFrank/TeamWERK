// Package notify provides a category-aware notification fan-out facade.
// Callers pass a category (e.g. "duties") and the facade dispatches push and
// email per user according to notification_preferences.
package notify

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/push"
)

// Send fans out a notification for the given category to the given users.
// Push goes to users with push_enabled=1 (default true).
// Email goes to users with email_enabled=1 (default false).
// Both default values match notification_preferences semantics.
//
// Send is fire-and-forget for email (one goroutine per recipient).
// Push runs synchronously inside push.SendToUsers, matching prior behavior.
func Send(db *sql.DB, cfg *appconfig.Config, userIDs []int, category, title, body, url string) {
	if len(userIDs) == 0 {
		return
	}

	pushUIDs := push.FilterByPushPref(db, userIDs, category)
	emailUIDs := filterByEmailPref(db, userIDs, category)

	if len(pushUIDs) > 0 {
		push.SendToUsers(db, cfg, pushUIDs, title, body, url)
	}

	for _, uid := range emailUIDs {
		go sendCategoryEmail(db, cfg, uid, title, body, url)
	}
}

// filterByEmailPref returns the subset of userIDs that have email_enabled=1
// for the given category. Users without a row default to false.
func filterByEmailPref(db *sql.DB, userIDs []int, category string) []int {
	if len(userIDs) == 0 {
		return nil
	}
	args := make([]any, len(userIDs)+1)
	args[0] = category
	placeholders := make([]string, len(userIDs))
	for i, id := range userIDs {
		args[i+1] = id
		placeholders[i] = "?"
	}
	query := fmt.Sprintf(
		`SELECT user_id FROM notification_preferences
		 WHERE category = ? AND email_enabled = 1 AND user_id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		result = append(result, id)
	}
	return result
}

// sendCategoryEmail loads the user's email and sends a plain-text notification mail.
// The body gets a direct-link line appended pointing into the app.
func sendCategoryEmail(db *sql.DB, cfg *appconfig.Config, userID int, title, body, url string) {
	var email string
	if err := db.QueryRow(`SELECT email FROM users WHERE id = ?`, userID).Scan(&email); err != nil || email == "" {
		return
	}
	fullBody := body
	if url != "" {
		fullBody = body + "\n\nDirektlink: " + cfg.BaseURL + url
	}
	m := mailer.New(cfg.SMTP)
	if err := m.Send(email, title, fullBody); err != nil {
		log.Printf("notifications: send mail to user %d: %v", userID, err)
	}
}
