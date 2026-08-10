package notify

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCancellationBody_MitGrund(t *testing.T) {
	got := CancellationBody("HSG Ostfildern", "am 14.09.2026", "Tim Meier", "Halle gesperrt")
	want := "HSG Ostfildern am 14.09.2026 entfällt. Abgesagt von Tim Meier: Halle gesperrt."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCancellationBody_OhneGrund(t *testing.T) {
	got := CancellationBody("HSG Ostfildern", "am 14.09.2026", "Tim Meier", "")
	want := "HSG Ostfildern am 14.09.2026 entfällt. Abgesagt von Tim Meier."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Kein leerer Grund-Doppelpunkt.
	if strings.Contains(got, ": ") {
		t.Errorf("Body enthält eine leere Grund-Einleitung: %q", got)
	}
}

// Ein Grund, der bereits einen Satz schließt, darf keinen zweiten Punkt bekommen.
func TestCancellationBody_GrundMitSatzzeichen(t *testing.T) {
	got := CancellationBody("HSG Ostfildern", "am 14.09.2026", "Tim Meier", "Halle gesperrt!")
	if strings.HasSuffix(got, "!.") {
		t.Errorf("doppelte Satzzeichen: %q", got)
	}
}

func TestCancellationBody_OhneAktor(t *testing.T) {
	got := CancellationBody("HSG Ostfildern", "am 14.09.2026", "", "Halle gesperrt")
	if !strings.Contains(got, fallbackActor) {
		t.Errorf("erwartete generische Aktor-Formulierung, got %q", got)
	}
}

func TestCancellationBody_OhneSubjektBleibtLesbar(t *testing.T) {
	got := CancellationBody("", "am 14.09.2026", "Tim Meier", "")
	if !strings.HasPrefix(got, fallbackSubject) {
		t.Errorf("erwarteten Fallback-Betreff, got %q", got)
	}
}

// Zeitraum-Variante: `when` trägt seine eigene Präposition, damit Serien
// ("ab …") und Einzeltermine ("am …") denselben Helfer nutzen können.
func TestCancellationBody_ZeitraumPhrase(t *testing.T) {
	got := CancellationBody("Trainingsserie Dienstag", "ab 01.09.2026", "Tim Meier", "")
	want := "Trainingsserie Dienstag ab 01.09.2026 entfällt. Abgesagt von Tim Meier."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTrimReason_KuerztAufRunenNichtBytes(t *testing.T) {
	// 500 Umlaute: 500 Runen, 1000 Bytes.
	long := strings.Repeat("ä", 500)
	got := TrimReason(long)

	if n := utf8.RuneCountInString(got); n != maxReasonRunes {
		t.Errorf("erwartete %d Runen, got %d", maxReasonRunes, n)
	}
	if !utf8.ValidString(got) {
		t.Error("Kürzung hat eine kaputte UTF-8-Sequenz erzeugt")
	}
}

func TestTrimReason_KurzerGrundBleibtUnveraendert(t *testing.T) {
	if got := TrimReason("  Halle gesperrt  "); got != "Halle gesperrt" {
		t.Errorf("got %q", got)
	}
}

func TestCancellationBody_UeberlangerGrundWirdGekuerzt(t *testing.T) {
	got := CancellationBody("HSG", "am 14.09.2026", "Tim Meier", strings.Repeat("x", 500))
	if strings.Count(got, "x") != maxReasonRunes {
		t.Errorf("erwartete %d gekürzte Zeichen im Body, got %d", maxReasonRunes, strings.Count(got, "x"))
	}
}

func TestFormatDateDMY(t *testing.T) {
	cases := map[string]string{
		"2026-06-14":           "14.06.2026",
		"2026-06-14T00:00:00Z": "14.06.2026",
		"kurz":                 "kurz",
	}
	for in, want := range cases {
		if got := FormatDateDMY(in); got != want {
			t.Errorf("FormatDateDMY(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestActorName(t *testing.T) {
	db := newTestDB(t)

	full := insertUser(t, db, "full@test.local")
	if _, err := db.Exec(`UPDATE users SET first_name='Tim', last_name='Meier' WHERE id=?`, full); err != nil {
		t.Fatal(err)
	}
	nameless := insertUser(t, db, "nameless@test.local")
	firstOnly := insertUser(t, db, "first@test.local")
	if _, err := db.Exec(`UPDATE users SET first_name='Tim' WHERE id=?`, firstOnly); err != nil {
		t.Fatal(err)
	}

	if got := ActorName(db, full); got != "Tim Meier" {
		t.Errorf("got %q, want %q", got, "Tim Meier")
	}
	if got := ActorName(db, firstOnly); got != "Tim" {
		t.Errorf("got %q, want %q", got, "Tim")
	}
	if got := ActorName(db, nameless); got != "" {
		t.Errorf("namenloser Nutzer: got %q, want leeren String", got)
	}
	if got := ActorName(db, 999999); got != "" {
		t.Errorf("unbekannter Nutzer: got %q, want leeren String", got)
	}
}

// Der Aktor-Name darf niemals auf die E-Mail-Adresse zurückfallen — der Text
// geht an ein ganzes Team.
func TestActorName_GibtNiemalsDieEmailPreis(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "geheim@test.local")

	body := CancellationBody("HSG", "am 14.09.2026", ActorName(db, uid), "")
	if strings.Contains(body, "geheim@test.local") || strings.Contains(body, "@") {
		t.Errorf("Body enthält die E-Mail-Adresse: %q", body)
	}
}

func TestDecodeCancellation(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		hasBody    bool
		wantReason string
		wantSilent bool
	}{
		{name: "kein Body", hasBody: false},
		{name: "leerer Body", body: "", hasBody: true},
		{name: "kaputtes JSON", body: "{nope", hasBody: true},
		{name: "leeres Objekt", body: `{}`, hasBody: true},
		{name: "nur Grund", body: `{"reason":"Halle gesperrt"}`, hasBody: true, wantReason: "Halle gesperrt"},
		{name: "Grund und silent", body: `{"reason":"Dublette","silent":true}`, hasBody: true, wantReason: "Dublette", wantSilent: true},
		{name: "silent ohne Grund", body: `{"silent":true}`, hasBody: true, wantSilent: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r *http.Request
			if tc.hasBody {
				r = httptest.NewRequest(http.MethodDelete, "/api/games/1", bytes.NewBufferString(tc.body))
			} else {
				r = httptest.NewRequest(http.MethodDelete, "/api/games/1", nil)
			}
			reason, silent := DecodeCancellation(r)
			if reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", reason, tc.wantReason)
			}
			if silent != tc.wantSilent {
				t.Errorf("silent = %v, want %v", silent, tc.wantSilent)
			}
		})
	}
}

func TestDecodeCancellation_KuerztDenGrund(t *testing.T) {
	body := `{"reason":"` + strings.Repeat("x", 500) + `"}`
	r := httptest.NewRequest(http.MethodDelete, "/api/games/1", bytes.NewBufferString(body))
	reason, _ := DecodeCancellation(r)
	if utf8.RuneCountInString(reason) != maxReasonRunes {
		t.Errorf("erwartete %d Runen, got %d", maxReasonRunes, utf8.RuneCountInString(reason))
	}
}
