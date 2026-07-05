package matchreports

import (
	"bytes"
	"sync"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

// SanitizeBody rendert Markdown zu HTML und läuft dann durch die
// Allowlist-Policy (siehe policy() unten). Alles, was nicht auf der Allowlist
// steht, wird gestrippt — Script-Tags, iframes, on*-Handler immer.
//
// Diese Funktion ist der Bruch mit dem „trust in author"-Modell: Publisher
// liefert nur strukturell erlaubtes HTML an TYPO3. Falls TYPO3 zusätzlich
// sanitizet (defense in depth), umso besser — hier ist die erste Barriere.
func SanitizeBody(bodyMarkdown string) (string, error) {
	// Markdown → HTML
	var rendered bytes.Buffer
	md := getMarkdownParser()
	if err := md.Convert([]byte(bodyMarkdown), &rendered); err != nil {
		return "", err
	}
	// Allowlist-Filter
	return getPolicy().Sanitize(rendered.String()), nil
}

// getPolicy liefert die bluemonday-Policy. Sync.Once, damit die Erstellung
// (etwas Regex-Kompilierung) einmal passiert.
var (
	policyOnce sync.Once
	policyVal  *bluemonday.Policy

	markdownOnce sync.Once
	markdownVal  goldmark.Markdown
)

func getPolicy() *bluemonday.Policy {
	policyOnce.Do(func() {
		p := bluemonday.NewPolicy()
		// Struktur-Elemente
		p.AllowElements("p", "br")
		// Überschriften — nur h2/h3, weil h1 der Seitentitel bleibt.
		p.AllowElements("h2", "h3")
		// Inline-Auszeichnung
		p.AllowElements("strong", "em", "b", "i")
		// Listen
		p.AllowElements("ul", "ol", "li")
		// Links: nur http/https, kein javascript:
		p.AllowStandardURLs()
		p.AllowAttrs("href").OnElements("a")
		p.RequireParseableURLs(true)
		p.AllowRelativeURLs(false)
		p.AllowURLSchemes("http", "https", "mailto")
		policyVal = p
	})
	return policyVal
}

func getMarkdownParser() goldmark.Markdown {
	markdownOnce.Do(func() {
		markdownVal = goldmark.New(
			goldmark.WithRendererOptions(
				// Kein rohes HTML — spare uns arbeit im Sanitizer.
				html.WithHardWraps(),
			),
		)
	})
	return markdownVal
}
