package push

import (
	"database/sql"
	"fmt"
	"strings"
)

// FilterByPushPref returns only the user IDs from userIDs that have push enabled
// for the given category. Users without a preference row are included (default: enabled).
func FilterByPushPref(db *sql.DB, userIDs []int, category string) []int {
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
		 WHERE category = ? AND push_enabled = 0 AND user_id IN (%s)`,
		strings.Join(placeholders, ","),
	)

	rows, err := db.Query(query, args...)
	if err != nil {
		return userIDs
	}
	defer rows.Close()

	disabled := map[int]bool{}
	for rows.Next() {
		var id int
		rows.Scan(&id)
		disabled[id] = true
	}

	result := userIDs[:0:len(userIDs)]
	for _, id := range userIDs {
		if !disabled[id] {
			result = append(result, id)
		}
	}
	return result
}

// HasEmailEnabled returns true if the user has email_enabled=1 for the given category.
func HasEmailEnabled(db *sql.DB, userID int, category string) bool {
	var enabled int
	err := db.QueryRow(
		`SELECT email_enabled FROM notification_preferences WHERE user_id = ? AND category = ?`,
		userID, category,
	).Scan(&enabled)
	if err != nil {
		return false
	}
	return enabled == 1
}

// GetAllPreferences returns all notification preferences for a user as a map.
// Categories without a row get their defaults (push=true, email=false).
func GetAllPreferences(db *sql.DB, userID int) map[string]map[string]bool {
	categories := []string{"games", "trainings", "duties", "duty_reminders", "carpooling", "membership"}
	result := make(map[string]map[string]bool, len(categories))
	for _, c := range categories {
		result[c] = map[string]bool{"push": true, "email": false}
	}

	rows, err := db.Query(
		`SELECT category, push_enabled, email_enabled FROM notification_preferences WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var pushEnabled, emailEnabled int
		rows.Scan(&category, &pushEnabled, &emailEnabled)
		if _, ok := result[category]; ok {
			result[category] = map[string]bool{
				"push":  pushEnabled == 1,
				"email": emailEnabled == 1,
			}
		}
	}
	return result
}
