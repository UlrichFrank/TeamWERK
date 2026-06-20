package chat

import "database/sql"

// ComputeUnreadForUser liefert die Summe aller ungelesenen 1:1-/Gruppen-Nachrichten
// plus die Anzahl ungelesener Broadcasts (nicht selbst gesendet) für den User.
//
// Semantik bewusst identisch zu:
//   - GET /api/chat/conversations (Feld unreadCount, handler.go Z. 157)
//   - GET /api/chat/broadcasts    (Filter br.read_at IS NULL, b.sender_id != ?, handler.go Z. 783)
func ComputeUnreadForUser(db *sql.DB, userID int) (int, error) {
	rows, err := db.Query(`
		SELECT COUNT(*) FROM messages m
		JOIN conversation_members cm ON cm.conversation_id = m.conversation_id
		WHERE cm.user_id = ? AND cm.left_at IS NULL
		  AND m.sender_id != ?
		  AND NOT EXISTS (
		    SELECT 1 FROM message_reads mr
		    WHERE mr.message_id = m.id AND mr.user_id = ?
		  )
		UNION ALL
		SELECT COUNT(*) FROM broadcast_reads br
		JOIN broadcasts b ON b.id = br.broadcast_id
		WHERE br.user_id = ?
		  AND br.read_at IS NULL
		  AND br.hidden_at IS NULL
		  AND b.sender_id != ?`,
		userID, userID, userID, userID, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return 0, err
		}
		total += n
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return total, nil
}
