package notify

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

// maxReasonRunes is the cap for a user-supplied deletion reason. Chosen to match
// the existing CHECK (length(note) <= 200) on games.note, and because push bodies
// get truncated by Android/iOS anyway. Counting runes (not bytes) keeps umlauts
// from being cut mid-sequence.
const maxReasonRunes = 200

// fallbackActor is used when the acting user has no name on file. Deliberately
// generic: users.email would leak a private address to the whole team.
const fallbackActor = "einem Trainer"

// fallbackSubject keeps the body readable if a caller has no name for the event.
const fallbackSubject = "Ein Termin"

// CancellationBody composes the body of a cancellation notification.
//
//	CancellationBody("HSG Ostfildern", "am 14.09.2026", "Tim Meier", "Halle gesperrt")
//	  → "HSG Ostfildern am 14.09.2026 entfällt. Abgesagt von Tim Meier: Halle gesperrt."
//
// Omitting the reason drops the trailing clause; omitting the actor falls back to
// a generic phrase. `when` is a complete temporal phrase including its preposition
// ("am 14.09.2026", "ab 01.09.2026") so that both single dates and ranges read
// correctly — see FormatDateDMY for building the date part.
//
// Every cancellation notification in the system goes through this function, which
// is what makes the invariant "no cancellation message without a subject and a
// date" checkable in one place.
func CancellationBody(subject, when, actor, reason string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = fallbackSubject
	}
	var b strings.Builder
	b.WriteString(subject)
	if when = strings.TrimSpace(when); when != "" {
		b.WriteString(" ")
		b.WriteString(when)
	}
	b.WriteString(" entfällt. ")
	b.WriteString(ActorClause(actor, reason))
	return b.String()
}

// ActorClause builds the trailing "Abgesagt von {actor}[: {reason}]."-sentence
// that every cancellation body ends with. It is exported separately because
// DeleteGame's duty message keeps its own leading sentence — that wording is
// pinned by a scenario in the push-duties spec — and only appends actor and
// reason. Sharing this clause keeps the name fallback and the punctuation
// rules in one place regardless.
func ActorClause(actor, reason string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = fallbackActor
	}

	var b strings.Builder
	b.WriteString("Abgesagt von ")
	b.WriteString(actor)

	if reason = TrimReason(reason); reason != "" {
		b.WriteString(": ")
		b.WriteString(reason)
		if !endsWithSentencePunctuation(reason) {
			b.WriteString(".")
		}
		return b.String()
	}
	b.WriteString(".")
	return b.String()
}

// endsWithSentencePunctuation reports whether s already closes a sentence, so a
// reason like "Halle gesperrt!" does not end up as "Halle gesperrt!.".
func endsWithSentencePunctuation(s string) bool {
	if s == "" {
		return false
	}
	switch s[len(s)-1] {
	case '.', '!', '?':
		return true
	}
	return false
}

// TrimReason trims surrounding whitespace and caps the reason at maxReasonRunes.
// Truncation is silent by design: the user just clicked delete, and an
// over-long reason must not turn that into an error.
func TrimReason(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= maxReasonRunes {
		return s
	}
	return strings.TrimSpace(string(r[:maxReasonRunes]))
}

// ActorName returns "Vorname Nachname" for the given user, or "" when no name is
// on file. It deliberately never falls back to users.email — the result is sent
// to a whole team.
func ActorName(db *sql.DB, userID int) string {
	var first, last string
	if err := db.QueryRow(
		`SELECT COALESCE(first_name, ''), COALESCE(last_name, '') FROM users WHERE id = ?`,
		userID).Scan(&first, &last); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
}

// FormatDateDMY turns "2026-06-14" (or an ISO timestamp) into "14.06.2026".
// Shared by every domain that builds a cancellation body.
func FormatDateDMY(s string) string {
	if len(s) < 10 {
		return s
	}
	d := s[:10]
	return d[8:10] + "." + d[5:7] + "." + d[0:4]
}

// DecodeCancellation reads the optional {reason, silent} body of a DELETE request.
//
// It is deliberately tolerant: a missing body (io.EOF), malformed JSON or unknown
// fields all yield ("", false) rather than an error. Older PWA installs sitting in
// the service worker cache send no body at all, and a 400 there would break
// deletion for them. The reason comes back already trimmed and capped.
//
// `silent` is only a request — the caller must still check the acting user's
// suppress capability before honouring it.
func DecodeCancellation(r *http.Request) (reason string, silent bool) {
	if r == nil || r.Body == nil {
		return "", false
	}
	var req struct {
		Reason string `json:"reason"`
		Silent bool   `json:"silent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", false
	}
	return TrimReason(req.Reason), req.Silent
}
