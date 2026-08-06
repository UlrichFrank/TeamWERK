package h4aimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// defaultBase ist der H4A-Host. Bewusst als Konstante mit https-Zwang (kein Downgrade).
const defaultBase = "https://meinh4a.handball4all.de"

// maxBodyBytes begrenzt die gelesene Antwortgröße (der volle Saison-Abruf ist ~einige
// hundert KB HTML; 8 MB Puffer schützt vor Speicher-Ausreißern auf dem 1-GB-VPS).
const maxBodyBytes = 8 << 20

// Client spricht die H4A-Weboberfläche (Formular-Login + xajax) über eine Session an.
type Client struct {
	http *http.Client
	base string
}

// NewClient liefert einen Client mit Cookie-Jar (Session-Persistenz) und 30s-Timeout.
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		http: &http.Client{Jar: jar, Timeout: 30 * time.Second},
		base: defaultBase,
	}
}

// requireHTTPS erzwingt TLS für alle ausgehenden H4A-Requests (fremde Credentials
// dürfen nie über Klartext-HTTP wandern). Siehe design.md §2.
func (c *Client) requireHTTPS() error {
	if !strings.HasPrefix(c.base, "https://") {
		return fmt.Errorf("H4A-Zugriff nur über HTTPS erlaubt")
	}
	return nil
}

// readBody liest den Response-Body begrenzt und schließt ihn.
func readBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return "", fmt.Errorf("antwort von Handball4All nicht lesbar: %w", err)
	}
	return string(b), nil
}

// Login meldet sich per Formular-POST an. Erfolg wird an "ABMELDEN" in der Folgeseite
// erkannt. WICHTIG: weder user noch pw werden geloggt oder in Fehler eingebettet.
func (c *Client) Login(ctx context.Context, user, pw string) error {
	if err := c.requireHTTPS(); err != nil {
		return err
	}
	form := url.Values{}
	form.Set("login", user)
	form.Set("pw", pw)
	form.Set("hvwsubmit", "submit")
	form.Set("submit", "Anmelden")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/index.php", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("H4A-Login-Request konnte nicht gebaut werden: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		// http.Client-Fehler enthalten nur URL/Netzinfo, nie den Body/Credentials.
		return fmt.Errorf("anmeldung bei Handball4All fehlgeschlagen: %w", err)
	}
	body, err := readBody(resp)
	if err != nil {
		return err
	}
	if !strings.Contains(body, "ABMELDEN") {
		// Generische Meldung, kein Passwort-Echo, kein Timing-Orakel.
		return errors.New("anmeldung bei Handball4All fehlgeschlagen")
	}
	return nil
}

var reOption = regexp.MustCompile(`(?is)<option\s+value="([^"]*)"[^>]*>(.*?)</option>`)

// FetchPeriods liest die Perioden/Saison-Optionen aus <select id="ge_periods"> von
// /games/edit.php (setzt eine bestehende Session voraus).
func (c *Client) FetchPeriods(ctx context.Context) ([]Period, error) {
	if err := c.requireHTTPS(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/games/edit.php", nil)
	if err != nil {
		return nil, fmt.Errorf("perioden-Request konnte nicht gebaut werden: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perioden von Handball4All nicht abrufbar: %w", err)
	}
	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}
	return parsePeriods(body)
}

// parsePeriods extrahiert das <select id="ge_periods">-Block und dessen <option>s.
func parsePeriods(htmlStr string) ([]Period, error) {
	sel := extractSelect(htmlStr, "ge_periods")
	if sel == "" {
		return nil, fmt.Errorf("H4A-Periodenauswahl (ge_periods) nicht gefunden — Format geändert?")
	}
	var periods []Period
	for _, m := range reOption.FindAllStringSubmatch(sel, -1) {
		id := strings.TrimSpace(m[1])
		name := cleanText(m[2])
		if id == "" {
			continue
		}
		periods = append(periods, Period{ID: id, Name: name})
	}
	if len(periods) == 0 {
		return nil, fmt.Errorf("H4A-Periodenauswahl leer — Format geändert?")
	}
	return periods, nil
}

// extractSelect schneidet den Inhalt eines <select id="<id>">…</select> heraus.
func extractSelect(htmlStr, id string) string {
	anchor := regexp.MustCompile(`(?is)<select[^>]*\bid="` + regexp.QuoteMeta(id) + `"[^>]*>`).FindStringIndex(htmlStr)
	if anchor == nil {
		return ""
	}
	rest := htmlStr[anchor[1]:]
	if end := strings.Index(strings.ToLower(rest), "</select>"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// xajaxResponse ist die JSON-Hülle der xajax-Antwort. Jedes Kommando (cmd) adressiert
// per id ein DOM-Element; "as" = assign HTML.
//
// Data ist bewusst json.RawMessage und nicht string: H4A liefert in derselben
// Kommandoliste auch Kommandos, deren data ein Array ist (z.B. Optionslisten für
// Selects). Mit string-Typisierung scheitert dann das Unmarshal der GESAMTEN
// Antwort — inklusive des gametable_container-Kommandos, das wir brauchen und das
// vollständig vorliegt.
type xajaxResponse struct {
	Xjxobj []struct {
		Cmd  string          `json:"cmd"`
		ID   string          `json:"id"`
		Data json.RawMessage `json:"data"`
	} `json:"xjxobj"`
}

// dataString liefert den String-Inhalt eines xajax-Kommandos. Nicht-String-data
// (Arrays, Objekte, null) ist für den Spielplan-Abruf ohne Belang und wird als
// „nicht verwertbar" gemeldet, statt die Antwort zu verwerfen.
func dataString(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// FetchGamesHTML ruft den Spielplan der Periode ab und extrahiert das HTML aus dem
// "as"-Kommando für #gametable_container (das mit der längsten data-Payload).
func (c *Client) FetchGamesHTML(ctx context.Context, periodID string) (string, error) {
	if err := c.requireHTTPS(); err != nil {
		return "", err
	}
	body := BuildGamesFormBody(periodID, nowUnixMilli())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/games/edit.php", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("Spielabruf-Request konnte nicht gebaut werden: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("spielabruf bei Handball4All fehlgeschlagen: %w", err)
	}
	status := resp.StatusCode
	raw, err := readBody(resp)
	if err != nil {
		return "", err
	}
	// Ohne Status-Prüfung liefe eine HTML-Fehlerseite in den JSON-Parser und
	// produzierte eine irreführende „kein JSON"-Meldung.
	if status != http.StatusOK {
		return "", fmt.Errorf("spielabruf bei Handball4All: HTTP %d, Antwort beginnt mit %s", status, snippet(raw))
	}

	var parsed xajaxResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		// Der Anfang der Antwort ist die entscheidende Information: daran ist zu
		// erkennen, ob H4A eine Login-/Fehlerseite statt der xajax-Hülle liefert.
		return "", fmt.Errorf("xajax-Antwort von Handball4All nicht als JSON lesbar (%d Bytes, beginnt mit %s): %w", len(raw), snippet(raw), err)
	}
	best := ""
	var seen []string
	for _, o := range parsed.Xjxobj {
		data, ok := dataString(o.Data)
		if !ok {
			seen = append(seen, fmt.Sprintf("%s/%s(kein String)", o.Cmd, o.ID))
			continue
		}
		seen = append(seen, fmt.Sprintf("%s/%s(%d)", o.Cmd, o.ID, len(data)))
		if o.Cmd == "as" && o.ID == "gametable_container" && len(data) > 50 && len(data) > len(best) {
			best = data
		}
	}
	if best == "" {
		return "", fmt.Errorf("xajax-Antwort ohne gametable_container-HTML — Format geändert? Enthaltene Kommandos: %s", strings.Join(seen, ", "))
	}
	return best, nil
}

// snippet kürzt eine Antwort auf einen loggbaren Anfang. Der Spielplan enthält
// keine Zugangsdaten; abgeschnitten wird trotzdem, damit das Log nicht volläuft.
func snippet(s string) string {
	const max = 200
	s = strings.TrimSpace(s)
	if len(s) > max {
		return fmt.Sprintf("%q…", s[:max])
	}
	return fmt.Sprintf("%q", s)
}

// Logout beendet die H4A-Session (best effort — die Session verfällt ohnehin serverseitig).
func (c *Client) Logout(ctx context.Context) error {
	if err := c.requireHTTPS(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/logout.php", nil)
	if err != nil {
		return fmt.Errorf("logout-Request konnte nicht gebaut werden: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("abmeldung bei Handball4All fehlgeschlagen: %w", err)
	}
	resp.Body.Close()
	return nil
}
