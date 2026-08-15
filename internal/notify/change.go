package notify

import "strings"

// ChangeBody composes the body of a change notification — the counterpart to
// CancellationBody for events that were edited rather than deleted.
//
//	ChangeBody("Ferientraining mB", "am 20.08.2026 um 18:00 Uhr", "", "Tim Meier")
//	  → "Ferientraining mB am 20.08.2026 um 18:00 Uhr wurde geändert. Geändert von Tim Meier."
//
//	ChangeBody("Ferientraining mB", "am 20.08.2026 um 18:00 Uhr", "19.08.2026, 17:00 Uhr", "Tim Meier")
//	  → "Ferientraining mB am 20.08.2026 um 18:00 Uhr wurde geändert (vorher 19.08.2026, 17:00 Uhr). Geändert von Tim Meier."
//
// `when` is a complete temporal phrase including its preposition ("am 20.08.2026
// um 18:00 Uhr", "ab 01.09.2026 bis 30.06.2027") so single dates and series
// periods share one helper — same contract as CancellationBody. `previously` is
// the prepositionless form (EventMoment) and MUST only be passed when the event
// actually moved: on an unchanged date it would read as a contradiction.
//
// Every change notification in the system goes through this function, which is
// what makes the invariant "no change message without a subject, a moment and an
// actor" checkable in one place.
func ChangeBody(subject, when, previously, actor string) string {
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
	b.WriteString(" wurde geändert")
	if previously = strings.TrimSpace(previously); previously != "" {
		b.WriteString(" (vorher ")
		b.WriteString(previously)
		b.WriteString(")")
	}
	b.WriteString(". Geändert von ")
	if actor = strings.TrimSpace(actor); actor == "" {
		actor = fallbackActor
	}
	b.WriteString(actor)
	b.WriteString(".")
	return b.String()
}

// CreationBody composes the body of a "new event" notification. It shares the
// moment formatters below with ChangeBody so a termin reads the same whether it
// was just created, edited or cancelled.
//
//	CreationBody("Heimspiel vs. HSG Ostfildern", "am 14.09.2026 um 18:00 Uhr", "Tim Meier")
//	  → "Heimspiel vs. HSG Ostfildern am 14.09.2026 um 18:00 Uhr. Angelegt von Tim Meier."
//
// There is no "previously" counterpart: a new event has no past.
func CreationBody(subject, when, actor string) string {
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
	b.WriteString(". Angelegt von ")
	if actor = strings.TrimSpace(actor); actor == "" {
		actor = fallbackActor
	}
	b.WriteString(actor)
	b.WriteString(".")
	return b.String()
}

// EventWhen builds the prepositional moment of a single event: "am 20.08.2026 um
// 18:00 Uhr". Without a clock it degrades to "am 20.08.2026", without a date to
// the empty string — ChangeBody then simply omits the phrase.
func EventWhen(date, clock string) string {
	d := FormatDateDMY(strings.TrimSpace(date))
	if d == "" {
		return ""
	}
	if t := FormatTimeHM(clock); t != "" {
		return "am " + d + " um " + t + " Uhr"
	}
	return "am " + d
}

// EventMoment builds the prepositionless form used inside the "(vorher …)"
// clause: "20.08.2026, 18:00 Uhr".
func EventMoment(date, clock string) string {
	d := FormatDateDMY(strings.TrimSpace(date))
	if d == "" {
		return ""
	}
	if t := FormatTimeHM(clock); t != "" {
		return d + ", " + t + " Uhr"
	}
	return d
}

// PreviousMoment returns the old moment for a ChangeBody "(vorher …)" clause —
// but only when the event actually moved. Both domains that edit events (games
// and trainings) need exactly this comparison, and they may not import each
// other, so it lives here next to the formatters it depends on.
//
// The comparison runs on the normalised forms on purpose: SQLite hands DATE
// columns back as ISO timestamps ("2026-08-19T00:00:00Z"), and a raw string
// compare against the request's "2026-08-19" would report every edit as a
// reschedule.
func PreviousMoment(oldDate, oldClock, newDate, newClock string) string {
	if FormatDateDMY(strings.TrimSpace(oldDate)) == FormatDateDMY(strings.TrimSpace(newDate)) &&
		FormatTimeHM(oldClock) == FormatTimeHM(newClock) {
		return ""
	}
	return EventMoment(oldDate, oldClock)
}

// FormatTimeHM normalises a stored time to "HH:MM". The columns are TEXT and in
// practice hold "18:00", but SQLite does not enforce that — a "18:00:00" from a
// hand-edited row must not leak seconds into a push body. Anything shorter than
// "HH:MM" yields the empty string, which callers treat as "no time on file".
func FormatTimeHM(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 5 {
		return ""
	}
	return s[:5]
}
