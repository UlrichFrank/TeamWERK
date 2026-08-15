package notify

import (
	"strings"
	"testing"
)

func TestChangeBody_OhneVerschiebung(t *testing.T) {
	got := ChangeBody("Ferientraining mB", "am 20.08.2026 um 18:00 Uhr", "", "Tim Meier")
	want := "Ferientraining mB am 20.08.2026 um 18:00 Uhr wurde geändert. Geändert von Tim Meier."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if strings.Contains(got, "vorher") {
		t.Errorf("ohne Verschiebung darf keine Vorher-Klammer erscheinen: %q", got)
	}
}

func TestChangeBody_MitVerschiebung(t *testing.T) {
	got := ChangeBody("Ferientraining mB", "am 20.08.2026 um 18:00 Uhr", "19.08.2026, 17:00 Uhr", "Tim Meier")
	want := "Ferientraining mB am 20.08.2026 um 18:00 Uhr wurde geändert (vorher 19.08.2026, 17:00 Uhr). Geändert von Tim Meier."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChangeBody_OhneAktor(t *testing.T) {
	got := ChangeBody("HSG Ostfildern", "am 14.09.2026 um 18:00 Uhr", "", "")
	if !strings.Contains(got, fallbackActor) {
		t.Errorf("erwartete generische Aktor-Formulierung, got %q", got)
	}
}

func TestChangeBody_OhneSubjektBleibtLesbar(t *testing.T) {
	got := ChangeBody("", "am 14.09.2026", "", "Tim Meier")
	if !strings.HasPrefix(got, fallbackSubject) {
		t.Errorf("erwarteten Fallback-Betreff, got %q", got)
	}
}

// Ohne Zeitangabe bleibt der Satz grammatikalisch heil (keine doppelten
// Leerzeichen, kein hängendes "am").
func TestChangeBody_OhneZeitangabe(t *testing.T) {
	got := ChangeBody("Sommerfest", "", "", "Tim Meier")
	want := "Sommerfest wurde geändert. Geändert von Tim Meier."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Zeitraum-Variante: `when` trägt seine eigene Präposition, damit Serien
// ("ab … bis …") und Einzeltermine ("am …") denselben Helfer nutzen.
func TestChangeBody_ZeitraumPhrase(t *testing.T) {
	got := ChangeBody("Trainingsserie Dienstag", "ab 01.09.2026 bis 30.06.2027", "montags 18:00 Uhr", "Tim Meier")
	want := "Trainingsserie Dienstag ab 01.09.2026 bis 30.06.2027 wurde geändert (vorher montags 18:00 Uhr). Geändert von Tim Meier."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Der Aktor-Name darf niemals auf die E-Mail-Adresse zurückfallen — der Text
// geht an ein ganzes Team.
func TestChangeBody_GibtNiemalsDieEmailPreis(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "geheim@test.local")

	body := ChangeBody("HSG", "am 14.09.2026 um 18:00 Uhr", "", ActorName(db, uid))
	if strings.Contains(body, "@") {
		t.Errorf("Body enthält die E-Mail-Adresse: %q", body)
	}
}

func TestCreationBody(t *testing.T) {
	got := CreationBody("Heimspiel vs. HSG Ostfildern", "am 14.09.2026 um 18:00 Uhr", "Tim Meier")
	want := "Heimspiel vs. HSG Ostfildern am 14.09.2026 um 18:00 Uhr. Angelegt von Tim Meier."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCreationBody_OhneAktorUndOhneZeitangabe(t *testing.T) {
	got := CreationBody("Sommerfest", "", "")
	if !strings.Contains(got, fallbackActor) {
		t.Errorf("erwartete generische Aktor-Formulierung, got %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("fehlende Zeitangabe darf kein doppeltes Leerzeichen hinterlassen: %q", got)
	}
}

func TestEventWhen(t *testing.T) {
	cases := []struct {
		date, clock, want string
	}{
		{"2026-08-20", "18:00", "am 20.08.2026 um 18:00 Uhr"},
		// SQLite liefert DATE-Spalten oft als ISO-Timestamp zurück.
		{"2026-08-20T00:00:00Z", "18:00", "am 20.08.2026 um 18:00 Uhr"},
		{"2026-08-20", "18:00:00", "am 20.08.2026 um 18:00 Uhr"},
		{"2026-08-20", "", "am 20.08.2026"},
		{"", "18:00", ""},
	}
	for _, c := range cases {
		if got := EventWhen(c.date, c.clock); got != c.want {
			t.Errorf("EventWhen(%q, %q) = %q, want %q", c.date, c.clock, got, c.want)
		}
	}
}

func TestEventMoment(t *testing.T) {
	cases := []struct {
		date, clock, want string
	}{
		{"2026-08-19", "17:00", "19.08.2026, 17:00 Uhr"},
		{"2026-08-19T00:00:00Z", "17:00", "19.08.2026, 17:00 Uhr"},
		{"2026-08-19", "", "19.08.2026"},
		{"", "17:00", ""},
	}
	for _, c := range cases {
		if got := EventMoment(c.date, c.clock); got != c.want {
			t.Errorf("EventMoment(%q, %q) = %q, want %q", c.date, c.clock, got, c.want)
		}
	}
}

func TestFormatTimeHM(t *testing.T) {
	cases := map[string]string{
		"18:00":    "18:00",
		"18:00:00": "18:00",
		" 18:00 ":  "18:00",
		"18:0":     "",
		"":         "",
	}
	for in, want := range cases {
		if got := FormatTimeHM(in); got != want {
			t.Errorf("FormatTimeHM(%q) = %q, want %q", in, got, want)
		}
	}
}
