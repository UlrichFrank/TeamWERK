package bwhv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// defaultBase ist der Handball4All-Dienst. Konstante mit https-Zwang (kein Downgrade).
const defaultBase = "https://spo.handball4all.de"

// UserAgent identifiziert TeamWERK gegenüber Handball4All. Der Abruf braucht
// zwar keine Zugangsdaten, pollt aber automatisiert einen fremden Dienst — der
// Betreiber soll erkennen können, wer da klopft (design.md §3).
const UserAgent = "TeamWERK/1.0 (+https://teamwerk.team-stuttgart.org; vorstand@team-stuttgart.org)"

// maxJSONBytes begrenzt die gelesene Antwortgröße. Ein Staffel-Spielplan ist
// ~57 KB, ein Katalog ~150 KB; 8 MB Puffer schützt vor Speicher-Ausreißern auf
// dem 1-GB-VPS.
const maxJSONBytes = 8 << 20

// maxPDFBytes begrenzt einen Berichts-Download. Beobachtet wurden ~140 KB bei
// vier Seiten; 32 MB lassen auch ungewöhnlich lange Berichte zu.
const maxPDFBytes = 32 << 20

// politePause ist die Pause zwischen zwei aufeinanderfolgenden Abrufen desselben
// Clients. Sie macht aus einem Berichts-Nachlauf über eine ganze Staffel eine
// serielle, gutmütige Last statt eines Bursts (design.md §3).
const politePause = 750 * time.Millisecond

// Client spricht die öffentliche Handball4All-Schnittstelle an.
//
// Bewusst OHNE Cookie-Jar: es gibt keine Session und keine Anmeldung. Der
// Client hält nur den Zeitpunkt des letzten Abrufs, um die Pause einzuhalten.
type Client struct {
	http     *http.Client
	base     string
	pause    time.Duration
	lastCall time.Time
}

// NewClient liefert einen Client mit 30s-Timeout.
func NewClient() *Client {
	return &Client{
		http:  &http.Client{Timeout: 30 * time.Second},
		base:  defaultBase,
		pause: politePause,
	}
}

// NewClientWithBase erlaubt es Tests, gegen einen httptest-Server zu laufen.
// Die https-Pflicht und die Höflichkeitspause entfallen dabei bewusst: ein
// Testserver spricht http, und eine Pause je Abruf machte die Testlaufzeit
// sekundenweise länger, ohne etwas zu beweisen.
func NewClientWithBase(base string) *Client {
	c := NewClient()
	c.base = strings.TrimRight(base, "/")
	c.pause = 0
	return c
}

// get führt einen GET aus und hält dabei die Pause zwischen zwei Abrufen ein.
func (c *Client) get(ctx context.Context, rawURL string, limit int64) ([]byte, string, error) {
	if wait := c.pause - time.Since(c.lastCall); wait > 0 && !c.lastCall.IsZero() {
		select {
		case <-ctx.Done():
			return nil, "", ctx.Err()
		case <-time.After(wait):
		}
	}
	c.lastCall = time.Now()

	if !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(c.base, "http://127.0.0.1") &&
		!strings.HasPrefix(c.base, "http://localhost") {
		return nil, "", fmt.Errorf("BWHV-Zugriff nur über HTTPS erlaubt: %s", rawURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("BWHV nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("BWHV antwortete mit HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, "", fmt.Errorf("BWHV-Antwort nicht lesbar: %w", err)
	}
	return b, resp.Header.Get("Content-Type"), nil
}

// serviceURL baut einen if_g_json.php-Aufruf.
//
// WICHTIG — die og/o-Falle: og ist die Organisation der WEBSITE, nicht die
// gewünschte Auswahl. Bezirks-Staffeln werden über o selektiert, während og auf
// der Verbands-Organisation stehen bleibt. Ein direkter Zugriff mit og=<bezirk>
// wird serverseitig mit HTTP 401 abgelehnt, unabhängig von Referer, Origin und
// User-Agent (design.md §1.2).
func (c *Client) serviceURL(cmd string, orgID, subOrgID int, extra url.Values) string {
	q := url.Values{}
	q.Set("cmd", cmd)
	q.Set("og", strconv.Itoa(orgID))
	if subOrgID > 0 {
		q.Set("o", strconv.Itoa(subOrgID))
	}
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	return c.base + "/service/if_g_json.php?" + q.Encode()
}

// envelope ist die gemeinsame Hülle aller if_g_json.php-Antworten. Der Dienst
// liefert ein Array mit genau einem Element.
type envelope struct {
	Menu struct {
		Org struct {
			List map[string]string `json:"list"`
		} `json:"org"`
		Period struct {
			List       map[string]string `json:"list"`
			SelectedID any               `json:"selectedID"`
		} `json:"period"`
	} `json:"menu"`
	Head struct {
		RepURL string `json:"repURL"`
	} `json:"head"`
	Content struct {
		Classes []struct {
			GClassID    string `json:"gClassID"`
			GClassSname string `json:"gClassSname"`
			GClassLname string `json:"gClassLname"`
		} `json:"classes"`
		FutureGames struct {
			Games []rawGame `json:"games"`
		} `json:"futureGames"`
		Score []rawScore `json:"score"`
	} `json:"content"`
}

// decodeEnvelope liest die Hülle. Ein Fehlerobjekt (permission denied) kommt
// als Objekt statt als Array und wird hier als klarer Fehler gemeldet, statt
// als leeres Ergebnis durchzurutschen.
func decodeEnvelope(b []byte) (*envelope, error) {
	trimmed := strings.TrimSpace(string(b))
	if strings.HasPrefix(trimmed, "{") {
		var e struct {
			Status     int    `json:"status"`
			StatusText string `json:"statusText"`
		}
		if json.Unmarshal(b, &e) == nil && e.StatusText != "" {
			return nil, fmt.Errorf("BWHV-Schnittstelle lehnte den Abruf ab: %s", e.StatusText)
		}
		return nil, fmt.Errorf("unerwartete Antwortform der BWHV-Schnittstelle")
	}
	var arr []envelope
	if err := json.Unmarshal(b, &arr); err != nil {
		return nil, fmt.Errorf("BWHV-Antwort nicht lesbar: %w", err)
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("BWHV-Antwort war leer")
	}
	return &arr[0], nil
}

// FetchCatalog liefert die Staffeln einer Organisation für eine Spielzeit.
// subOrgID 0 bedeutet Verbandsebene; ein Bezirk wird über subOrgID gewählt.
// periodID darf leer sein — der Dienst nimmt dann die laufende Spielzeit.
func (c *Client) FetchCatalog(ctx context.Context, orgID, subOrgID int, periodID string) ([]Class, error) {
	extra := url.Values{}
	if periodID != "" {
		extra.Set("p", periodID)
	}
	b, _, err := c.get(ctx, c.serviceURL("po", orgID, subOrgID, extra), maxJSONBytes)
	if err != nil {
		return nil, err
	}
	env, err := decodeEnvelope(b)
	if err != nil {
		return nil, err
	}
	out := make([]Class, 0, len(env.Content.Classes))
	for _, cl := range env.Content.Classes {
		if cl.GClassSname == "" {
			continue
		}
		out = append(out, Class{ID: cl.GClassID, Sname: cl.GClassSname, Lname: cl.GClassLname})
	}
	return out, nil
}

// FetchPeriods liefert die wählbaren Spielzeiten und die vom Dienst als aktuell
// markierte. Die laufende Spielzeit wird daraus gelesen statt verdrahtet.
func (c *Client) FetchPeriods(ctx context.Context, orgID int) (periods []Period, selected string, err error) {
	b, _, err := c.get(ctx, c.serviceURL("po", orgID, 0, nil), maxJSONBytes)
	if err != nil {
		return nil, "", err
	}
	env, err := decodeEnvelope(b)
	if err != nil {
		return nil, "", err
	}
	for id, name := range env.Menu.Period.List {
		periods = append(periods, Period{ID: id, Name: name})
	}
	switch v := env.Menu.Period.SelectedID.(type) {
	case string:
		selected = v
	case float64:
		selected = strconv.Itoa(int(v))
	}
	return periods, selected, nil
}

// FetchOrgs liefert die wählbaren Organisationen: den Verband selbst und
// seine Bezirke. Daraus leitet der Aufrufer ab, wo er eine Staffel suchen muss,
// statt das Suffix des Staffelcodes ("-SRM") fest zu verdrahten.
//
// Die eigene Organisation (orgID) ist in der Liste enthalten und wird vom
// Aufrufer als Verbandsebene behandelt; alle übrigen sind Unter-Organisationen
// und gehören in den o-Parameter, nicht in og.
func (c *Client) FetchOrgs(ctx context.Context, orgID int) (map[string]string, error) {
	b, _, err := c.get(ctx, c.serviceURL("po", orgID, 0, nil), maxJSONBytes)
	if err != nil {
		return nil, err
	}
	env, err := decodeEnvelope(b)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for id, name := range env.Menu.Org.List {
		if id == "-1" || strings.TrimSpace(name) == "" {
			continue
		}
		out[id] = name
	}
	return out, nil
}

// FetchSchedule liefert alle Begegnungen einer Staffel über die ganze Saison
// samt Tabellenstand. ca=1 schaltet "alle Spiele" statt nur der kommenden frei.
func (c *Client) FetchSchedule(ctx context.Context, orgID, subOrgID int, periodID, classID string) (*Schedule, error) {
	extra := url.Values{}
	extra.Set("cl", classID)
	extra.Set("ca", "1")
	if periodID != "" {
		extra.Set("p", periodID)
	}
	b, _, err := c.get(ctx, c.serviceURL("ps", orgID, subOrgID, extra), maxJSONBytes)
	if err != nil {
		return nil, err
	}
	env, err := decodeEnvelope(b)
	if err != nil {
		return nil, err
	}
	sch := &Schedule{ReportURL: env.Head.RepURL}
	for _, rg := range env.Content.FutureGames.Games {
		g, ok := rg.toGame()
		if !ok {
			continue
		}
		sch.Games = append(sch.Games, g)
	}
	for i, rs := range env.Content.Score {
		sch.Table = append(sch.Table, rs.toRow(i+1))
	}
	return sch, nil
}

// FetchReport lädt ein Spielbericht-PDF. reportURL ist das Präfix aus
// Schedule.ReportURL, an das die sGID gehängt wird.
//
// Geprüft wird sowohl der Content-Type als auch die %PDF-Signatur: eine
// HTML-Fehlerseite mit HTTP 200 darf nicht als Bericht durchgehen und später
// im Parser als unverständliches Dokument auflaufen.
func (c *Client) FetchReport(ctx context.Context, reportURL, sgid string) ([]byte, error) {
	if sgid == "" || sgid == "0" {
		return nil, fmt.Errorf("kein Spielbericht verfügbar (leere sGID)")
	}
	if reportURL == "" {
		reportURL = c.base + "/misc/sboPublicReports.php?sGID="
	}
	b, ctype, err := c.get(ctx, reportURL+url.QueryEscape(sgid), maxPDFBytes)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(strings.ToLower(ctype), "pdf") && !strings.HasPrefix(string(b), "%PDF") {
		return nil, fmt.Errorf("antwort ist kein PDF (Content-Type %q, %d Bytes)", ctype, len(b))
	}
	if !strings.HasPrefix(string(b), "%PDF") {
		return nil, fmt.Errorf("antwort trägt keine PDF-Signatur (%d Bytes)", len(b))
	}
	return b, nil
}

// ReportURLPrefix liefert das Präfix für Spielbericht-Downloads. Der Dienst
// liefert es in jeder Antwort als head.repURL mit; der hier gebildete Wert ist
// der Rückfall, wenn kein Spielplan zur Hand ist.
func (c *Client) ReportURLPrefix() string {
	return c.base + "/misc/sboPublicReports.php?sGID="
}
