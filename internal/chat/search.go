package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// SearchHit ist ein einzelner Treffer der übergreifenden Volltextsuche
// (design.md, Requirement „Übergreifende Volltextsuche"): entweder eine
// Nachricht (kind="message", ConversationID gesetzt) oder eine Mitteilung
// (kind="broadcast", ConversationID nil, ConversationName fest "Mitteilung").
type SearchHit struct {
	Kind             string `json:"kind"`
	ID               int    `json:"id"`
	ConversationID   *int   `json:"conversationId,omitempty"`
	ConversationName string `json:"conversationName"`
	SenderName       string `json:"senderName"`
	Snippet          string `json:"snippet"`
	SentAt           string `json:"sentAt"`
}

// searchSnippetLen/-Half bestimmen das Textfenster um die erste Fundstelle
// (design.md Entscheidung 5: ca. 140 Runen statt Volltext).
const (
	searchSnippetLen  = 140
	searchSnippetHalf = 60
)

// likePattern escaped die SQLite-LIKE-Wildcards (`%`, `_`) sowie das
// Escape-Zeichen selbst im Suchbegriff und umschließt ihn mit `%` für eine
// Teilstring-Suche (design.md Entscheidung 4). Immer zusammen mit
// `LIKE ? ESCAPE '\'` verwenden.
func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// snippet liefert einen ca. searchSnippetLen Runen breiten, rune-genauen
// Ausschnitt um die erste (case-insensitive) Fundstelle von q in body, mit
// Ellipsen an abgeschnittenen Enden. Ohne Fundstelle: die ersten
// searchSnippetLen Runen. Groß-/Kleinschreibung wird über strings.ToLower
// gefaltet — das bildet jede Rune 1:1 auf eine Rune ab (kein Fall wie bei
// unicode.SpecialCase), der Byte-Index im gelowerten String lässt sich daher
// über utf8.RuneCountInString direkt in den Rune-Index der ORIGINAL-Sequenz
// übersetzen, ohne eine Rune zu zerschneiden.
func snippet(body, q string) string {
	runes := []rune(body)
	total := len(runes)

	lowerBody := strings.ToLower(body)
	lowerQ := strings.ToLower(q)
	byteIdx := strings.Index(lowerBody, lowerQ)

	if byteIdx < 0 {
		if total <= searchSnippetLen {
			return string(runes)
		}
		return string(runes[:searchSnippetLen]) + "…"
	}

	idx := utf8.RuneCountInString(lowerBody[:byteIdx])
	start := idx - searchSnippetHalf
	if start < 0 {
		start = 0
	}
	end := start + searchSnippetLen
	if end > total {
		end = total
	}

	out := string(runes[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < total {
		out += "…"
	}
	return out
}

// parseSentAt normalisiert einen DB-Zeitstempel für den Sortiervergleich
// zwischen messages.sent_at und broadcasts.sent_at. Beide Spalten nutzen
// denselben SQLite-DEFAULT-CURRENT_TIMESTAMP-Format ("2006-01-02 15:04:05");
// ein RFC3339-Wert wird zusätzlich erkannt, falls eine der beiden Quellen
// künftig davon abweicht. Bei Parse-Fehler: Zero-Value (rutscht ans Ende,
// bricht die Antwort aber nicht ab).
func parseSentAt(s string) time.Time {
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GET /api/chat/search?q=&limit=&offset=
//
// Durchsucht messages.body aller Konversationen, in denen der Nutzer aktives
// Mitglied ist (conversation_members.left_at IS NULL — bewusst enger als die
// isMember-Regel von ListMessages, design.md Entscheidung 3), sowie
// broadcasts.body aller für ihn sichtbaren Mitteilungen (broadcast_reads
// vorhanden, hidden_at IS NULL). Gelöschte Nachrichten (deleted_at IS NOT
// NULL) und System-Nachrichten bleiben ausgeschlossen. Zwei getrennte
// Teilqueries statt SQL-UNION (design.md Entscheidung 2): beide Mengen werden
// geladen, in Go nach sent_at DESC (id DESC als Tie-Breaker) sortiert und
// dann per limit/offset geschnitten — total ist die Länge vor dem Schnitt.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "q required", http.StatusBadRequest)
		return
	}

	// Gleiches Parsing-/Clamping-Muster wie internal/members/handler.go List.
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	pattern := likePattern(q)
	hits := make([]SearchHit, 0)

	// Nachrichten: Sichtbarkeit über aktive Konversationsmitgliedschaft.
	// Direkt-Chats (c.name IS NULL) lösen ihren Anzeigenamen über die anderen
	// aktiven Mitglieder auf (group_concat), Fallback "Konversation" bei einer
	// verwaisten Direktkonversation ohne verbliebenes Gegenüber.
	msgRows, err := h.db.QueryContext(r.Context(), `
		SELECT m.id, m.conversation_id,
		       COALESCE(c.name, (
		         SELECT GROUP_CONCAT(u2.first_name || ' ' || u2.last_name, ', ')
		         FROM conversation_members cm2
		         JOIN users u2 ON u2.id = cm2.user_id
		         WHERE cm2.conversation_id = c.id AND cm2.user_id != ? AND cm2.left_at IS NULL
		       ), 'Konversation') AS conv_name,
		       u.first_name || ' ' || u.last_name, m.body, m.sent_at
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		JOIN conversation_members cm ON cm.conversation_id = c.id AND cm.user_id = ? AND cm.left_at IS NULL
		JOIN users u ON u.id = m.sender_id
		WHERE m.deleted_at IS NULL AND m.is_system = 0 AND m.body LIKE ? ESCAPE '\'`,
		claims.UserID, claims.UserID, pattern)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer msgRows.Close()
	for msgRows.Next() {
		var id, convID int
		var convName, senderName, body, sentAt string
		if err := msgRows.Scan(&id, &convID, &convName, &senderName, &body, &sentAt); err != nil {
			continue
		}
		cID := convID
		hits = append(hits, SearchHit{
			Kind:             "message",
			ID:               id,
			ConversationID:   &cID,
			ConversationName: convName,
			SenderName:       senderName,
			Snippet:          snippet(body, q),
			SentAt:           sentAt,
		})
	}
	msgRows.Close()

	// Mitteilungen: Sichtbarkeit über eine vorhandene, nicht ausgeblendete
	// broadcast_reads-Zeile. broadcasts trägt kein deleted_at (Schema-Stand
	// Migration 055) — hidden_at ist hier der einzige Ausschlussfilter.
	bRows, err := h.db.QueryContext(r.Context(), `
		SELECT b.id, u.first_name || ' ' || u.last_name, b.body, b.sent_at
		FROM broadcasts b
		JOIN users u ON u.id = b.sender_id
		JOIN broadcast_reads br ON br.broadcast_id = b.id AND br.user_id = ?
		WHERE br.hidden_at IS NULL AND b.body LIKE ? ESCAPE '\'`,
		claims.UserID, pattern)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer bRows.Close()
	for bRows.Next() {
		var id int
		var senderName, body, sentAt string
		if err := bRows.Scan(&id, &senderName, &body, &sentAt); err != nil {
			continue
		}
		hits = append(hits, SearchHit{
			Kind:             "broadcast",
			ID:               id,
			ConversationName: "Mitteilung",
			SenderName:       senderName,
			Snippet:          snippet(body, q),
			SentAt:           sentAt,
		})
	}
	bRows.Close()

	sort.Slice(hits, func(i, j int) bool {
		ti, tj := parseSentAt(hits[i].SentAt), parseSentAt(hits[j].SentAt)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return hits[i].ID > hits[j].ID
	})

	total := len(hits)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	items := hits[start:end]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
	})
}
