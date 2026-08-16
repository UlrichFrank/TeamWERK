package matchreports

import (
	"database/sql"
	"log/slog"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
)

// reviewerUserIDs sammelt die aktuellen Freigeber (Vereinsfunktion 'medien' ODER
// 'vorstand') als Slice von user_ids. Wird sowohl vom Submit-Handler als auch
// vom Scheduler-Reminder-Job verwendet.
//
// Ein User erscheint höchstens einmal, auch wenn er beide Funktionen hat.
func reviewerUserIDs(db *sql.DB) ([]int, error) {
	rows, err := db.Query(
		`SELECT DISTINCT u.id
		 FROM users u
		 JOIN members m ON m.user_id = u.id
		 JOIN member_club_functions mcf ON mcf.member_id = m.id
		 WHERE mcf.function IN ('medien','vorstand')`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// notifyReviewers sendet Push an alle aktuellen Freigeber. Läuft als Goroutine
// (blocking wäre für einen HTTP-Response verheerend). Bei Query-Fehlern nur
// loggen — Notification-Fehler dürfen keinen Submit killen. Bewusst push-only
// (notify.NoEmail()) — die Fassade übernimmt Log-Fan-out + Push-Präferenz.
func notifyReviewers(db *sql.DB, cfg *appconfig.Config, title, body, url string) {
	ids, err := reviewerUserIDs(db)
	if err != nil {
		slog.Error("matchreports.notifyReviewers query", "err", err)
		return
	}
	notify.Send(db, cfg, ids, "operativ", title, body, url, notify.NoEmail())
}
