package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// newlyReadPerSender liefert pro Absender die höchste message_id in der Konversation,
// die für readerID noch NICHT gelesen ist (also durch den folgenden MarkRead-INSERT
// neu markiert wird). Wird VOR dem INSERT OR IGNORE aufgerufen. Fehler werden
// bewusst verschluckt (Read-Receipt-Fanout ist best-effort, kein Nutzer-facing Pfad).
func (h *Handler) newlyReadPerSender(ctx context.Context, convID, readerID int) map[int]int {
	rows, err := h.db.QueryContext(ctx, `
		SELECT m.sender_id, MAX(m.id)
		FROM messages m
		WHERE m.conversation_id = ? AND m.sender_id != ?
		  AND NOT EXISTS (
		      SELECT 1 FROM message_reads mr
		      WHERE mr.message_id = m.id AND mr.user_id = ?)
		GROUP BY m.sender_id`, convID, readerID, readerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := map[int]int{}
	for rows.Next() {
		var sender, upTo int
		if err := rows.Scan(&sender, &upTo); err == nil {
			out[sender] = upTo
		}
	}
	return out
}

// messageReader ist ein Eintrag der Reader-Liste einer Nachricht.
type messageReader struct {
	UserID int    `json:"userId"`
	Name   string `json:"name"`
	ReadAt string `json:"readAt"`
}

// GET /api/chat/messages/{id}/reads
// Liefert die Leser-Liste (ohne den Absender) einer eigenen Nachricht, sortiert
// nach read_at aufsteigend. Nur der Absender darf sie sehen (403 sonst); für
// fehlende oder gelöschte Nachrichten 404.
func (h *Handler) MessageReads(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	msgID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var senderID int
	var deletedAt sql.NullString
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT sender_id, deleted_at FROM messages WHERE id = ?`, msgID).
		Scan(&senderID, &deletedAt); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if deletedAt.Valid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// Reads Dritter sind nur für den Absender sichtbar (WhatsApp-Semantik).
	if senderID != claims.UserID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT u.id, u.first_name || ' ' || u.last_name, mr.read_at
		FROM message_reads mr
		JOIN users u ON u.id = mr.user_id
		WHERE mr.message_id = ? AND mr.user_id != ?
		ORDER BY mr.read_at ASC`, msgID, senderID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	readers := []messageReader{}
	for rows.Next() {
		var mr messageReader
		if err := rows.Scan(&mr.UserID, &mr.Name, &mr.ReadAt); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		readers = append(readers, mr)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(readers)
}
