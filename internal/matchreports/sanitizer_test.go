package matchreports

import (
	"strings"
	"testing"
)

func TestSanitizeBody_AllowedTagsPassThrough(t *testing.T) {
	body := "## Erste Halbzeit\n\nDer Auftakt war **zäh**, dann Torfestival.\n\n- Ein\n- Zwei"
	html, err := SanitizeBody(body)
	if err != nil {
		t.Fatalf("SanitizeBody: %v", err)
	}
	for _, want := range []string{"<h2>", "<strong>zäh</strong>", "<ul>", "<li>Ein"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output, got: %s", want, html)
		}
	}
}

func TestSanitizeBody_StripsScriptAndOnHandlers(t *testing.T) {
	// Goldmark rendert rohes HTML per Default nicht — bluemonday darüber
	// sichert zusätzlich ab. Hier ein Angriff über inline-HTML.
	body := "Hallo <script>alert(1)</script> <a href=\"javascript:alert(2)\" onclick=\"bad()\">Klick</a>"
	html, err := SanitizeBody(body)
	if err != nil {
		t.Fatalf("SanitizeBody: %v", err)
	}
	if strings.Contains(html, "<script>") || strings.Contains(html, "onclick") ||
		strings.Contains(html, "javascript:") {
		t.Errorf("dangerous content leaked: %s", html)
	}
}

func TestSanitizeBody_AllowsSafeLinks(t *testing.T) {
	body := "Siehe [Homepage](https://team-stuttgart.org)."
	html, err := SanitizeBody(body)
	if err != nil {
		t.Fatalf("SanitizeBody: %v", err)
	}
	if !strings.Contains(html, `href="https://team-stuttgart.org"`) {
		t.Errorf("expected https link in output, got: %s", html)
	}
}
