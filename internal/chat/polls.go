package chat

import (
	"context"
	"fmt"
	"strings"
)

// pollVoter ist ein Abstimmender einer Umfrage-Option (design.md §4). Nicht
// anonym — Namen werden immer mitgeliefert, analog reactions[].userNames.
type pollVoter struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// pollOption ist eine Antwortoption mit Zahl, eigener Wahl und Abstimmenden.
type pollOption struct {
	ID     int         `json:"id"`
	Label  string      `json:"label"`
	Count  int         `json:"count"`
	Voted  bool        `json:"voted"`
	Voters []pollVoter `json:"voters"`
}

// pollView ist die JSON-Form einer Umfrage (design.md §4), sowohl im
// ListMessages-Batch-Anhang als auch im Einzelabruf (GET .../poll).
type pollView struct {
	AllowMultiple bool         `json:"allowMultiple"`
	ClosedAt      *string      `json:"closedAt"`
	VoterCount    int          `json:"voterCount"`
	Options       []pollOption `json:"options"`
}

// loadPolls lädt für die gegebenen Nachrichten-IDs (falls sie Umfragen sind)
// den vollständigen Umfrage-Zustand aus Sicht von userID (voted-Flag). Zwei
// Batch-Queries nach dem Reaktions-Muster: Umfragen+Optionen, dann Stimmen
// mit Namen. Nachrichten ohne chat_polls-Zeile fehlen im Ergebnis-Map.
func (h *Handler) loadPolls(ctx context.Context, userID int, msgIDs []int) (map[int]*pollView, error) {
	result := map[int]*pollView{}
	if len(msgIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(msgIDs))
	args := make([]any, len(msgIDs))
	for i, id := range msgIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	// optIndex: messageID -> optionID -> Index in pollView.Options (Reihenfolge
	// nach position, wie beim Anlegen gesetzt).
	optIndex := map[int]map[int]int{}

	rows, err := h.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT cp.message_id, cp.allow_multiple, cp.closed_at, o.id, o.label
		FROM chat_polls cp
		JOIN chat_poll_options o ON o.message_id = cp.message_id
		WHERE cp.message_id IN (%s)
		ORDER BY cp.message_id, o.position`, inClause), args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var msgID, allowMultiple, optID int
		var closedAt *string
		var label string
		if err := rows.Scan(&msgID, &allowMultiple, &closedAt, &optID, &label); err != nil {
			rows.Close()
			return nil, err
		}
		pv, ok := result[msgID]
		if !ok {
			pv = &pollView{AllowMultiple: allowMultiple == 1, ClosedAt: closedAt, Options: []pollOption{}}
			result[msgID] = pv
			optIndex[msgID] = map[int]int{}
		}
		optIndex[msgID][optID] = len(pv.Options)
		pv.Options = append(pv.Options, pollOption{ID: optID, Label: label, Voters: []pollVoter{}})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if len(result) == 0 {
		return result, nil
	}

	voteRows, err := h.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT o.message_id, v.option_id, v.user_id, u.first_name || ' ' || u.last_name
		FROM chat_poll_votes v
		JOIN chat_poll_options o ON o.id = v.option_id
		JOIN users u ON u.id = v.user_id
		WHERE o.message_id IN (%s)
		ORDER BY o.message_id, o.position, v.created_at`, inClause), args...)
	if err != nil {
		return nil, err
	}
	defer voteRows.Close()

	voterSets := map[int]map[int]bool{}
	for voteRows.Next() {
		var msgID, optID, voterID int
		var name string
		if err := voteRows.Scan(&msgID, &optID, &voterID, &name); err != nil {
			return nil, err
		}
		pv := result[msgID]
		idx, ok := optIndex[msgID][optID]
		if !ok {
			continue
		}
		pv.Options[idx].Count++
		pv.Options[idx].Voters = append(pv.Options[idx].Voters, pollVoter{ID: voterID, Name: name})
		if voterID == userID {
			pv.Options[idx].Voted = true
		}
		if voterSets[msgID] == nil {
			voterSets[msgID] = map[int]bool{}
		}
		voterSets[msgID][voterID] = true
	}
	if err := voteRows.Err(); err != nil {
		return nil, err
	}
	for msgID, set := range voterSets {
		result[msgID].VoterCount = len(set)
	}

	return result, nil
}
