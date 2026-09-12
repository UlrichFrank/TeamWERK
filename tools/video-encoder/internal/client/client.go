// Package client spricht die bestehende TeamWERK-API an — so, wie es der
// Browser tut: Login mit E-Mail/Passwort, Access-Token im Speicher,
// Refresh-Token als Cookie im Cookie-Jar, 401 → Refresh → Wiederholung. Das
// Tool braucht dafür keine eigenen Server-Endpunkte.
//
// Hinweis für die Entwicklung: der Server setzt das Refresh-Cookie mit
// `Secure`. Gegen einen lokalen http://-Server schickt der Cookie-Jar es
// deshalb nicht zurück — Login und Upload gehen, ein Refresh nach 15 min nicht.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("E-Mail oder Passwort ist falsch")
	ErrTooManyAttempts    = errors.New("zu viele Anmeldeversuche — bitte später erneut versuchen")
	ErrSessionExpired     = errors.New("Sitzung abgelaufen — bitte erneut anmelden")
	ErrMaintenance        = errors.New("TeamWERK ist gerade im Wartungsmodus — bitte später erneut versuchen")
	ErrNoUploadPermission = errors.New("dieses Konto darf keine Videos hochladen (nur Trainer, sportliche Leitung und Vorstand)")
	ErrNoActiveSeason     = errors.New("in TeamWERK ist keine Saison aktiv — bitte den Vorstand informieren")
)

// StatusError ist eine unerwartete HTTP-Antwort.
type StatusError struct {
	Op     string
	Status int
	Body   string
}

func (e *StatusError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("%s: HTTP %d (%s)", e.Op, e.Status, e.Body)
	}
	return fmt.Sprintf("%s: HTTP %d", e.Op, e.Status)
}

// Client ist eine angemeldete (oder anzumeldende) Verbindung zu einem
// TeamWERK-Server. Nicht für parallele Uploads gedacht; der Token-Zugriff ist
// trotzdem abgesichert, weil UI-Abrufe (Spiele laden) parallel laufen können.
type Client struct {
	base *url.URL
	hc   *http.Client

	mu    sync.Mutex
	token string

	// Upload-Parameter, in Tests verkleinert.
	chunkSize   int64
	retryDelays []time.Duration
	sleep       func(ctx context.Context, d time.Duration) error
}

// New baut einen Client für baseURL (z. B. https://teamwerk.team-stuttgart.org).
// hc darf nil sein; ohne Cookie-Jar bekommt er einen.
func New(baseURL string, hc *http.Client) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, fmt.Errorf("ungültige Server-Adresse %q", baseURL)
	}
	if hc == nil {
		hc = &http.Client{Transport: defaultTransport()}
	}
	if hc.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		hc.Jar = jar
	}
	return &Client{
		base:      u,
		hc:        hc,
		chunkSize: 64 << 20, // wie der frühere Browser-Upload: guter Kompromiss aus Recovery und Overhead
		// Summe ≈ 5 min: überbrückt einen WLAN-Wechsel oder Router-Neustart.
		retryDelays: []time.Duration{time.Second, 3 * time.Second, 5 * time.Second, 10 * time.Second,
			20 * time.Second, 30 * time.Second, 60 * time.Second, 60 * time.Second, 60 * time.Second, 60 * time.Second},
		sleep: ctxSleep,
	}, nil
}

// Kein Gesamt-Timeout am Client — ein 64-MB-Chunk braucht auf einem
// langsamen Upstream Minuten. Abgesichert sind Verbindungsaufbau und das
// Warten auf die Antwort, nachdem der Body raus ist.
func defaultTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	t.ResponseHeaderTimeout = 2 * time.Minute
	return t
}

func ctxSleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// BaseURL liefert die Server-Adresse ohne abschließenden Slash.
func (c *Client) BaseURL() string { return c.base.String() }

// VideoURL liefert den Link auf die Detailseite eines Videos in TeamWERK.
func (c *Client) VideoURL(id int) string { return c.BaseURL() + "/videos/" + strconv.Itoa(id) }

func (c *Client) accessToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token
}

func (c *Client) setAccessToken(t string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = t
}

// resolve löst path (relativ oder absolut) gegen die Server-Adresse auf und
// weist Ziele auf fremden Hosts ab — sonst ginge der Bearer-Token dorthin.
func (c *Client) resolve(path string) (*url.URL, error) {
	ref, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u := c.base.ResolveReference(ref)
	if u.Scheme != c.base.Scheme || u.Host != c.base.Host {
		return nil, fmt.Errorf("Server verweist auf fremde Adresse %s", u.Redacted())
	}
	return u, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u, err := c.resolve(path)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, method, u.String(), body)
}

// Login meldet sich mit E-Mail und Passwort an (POST /api/auth/login). Das
// Refresh-Cookie der Antwort landet im Cookie-Jar.
func (c *Client) Login(ctx context.Context, email, password string) error {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := c.newRequest(ctx, http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("Server nicht erreichbar: %w", err)
	}
	defer drain(resp)
	switch resp.StatusCode {
	case http.StatusOK:
		return c.takeToken(resp, "Anmeldung")
	case http.StatusUnauthorized:
		return ErrInvalidCredentials
	case http.StatusTooManyRequests:
		return ErrTooManyAttempts
	default:
		return statusError("Anmeldung", resp)
	}
}

// refresh holt mit dem Refresh-Cookie einen neuen Access-Token
// (POST /api/auth/refresh). 401 heißt: Refresh-Token abgelaufen.
func (c *Client) refresh(ctx context.Context) error {
	req, err := c.newRequest(ctx, http.MethodPost, "/api/auth/refresh", nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer drain(resp)
	switch resp.StatusCode {
	case http.StatusOK:
		return c.takeToken(resp, "Token-Erneuerung")
	case http.StatusUnauthorized:
		c.setAccessToken("")
		return ErrSessionExpired
	default:
		return statusError("Token-Erneuerung", resp)
	}
}

func (c *Client) takeToken(resp *http.Response, op string) error {
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.AccessToken == "" {
		return fmt.Errorf("%s: unerwartete Antwort des Servers", op)
	}
	c.setAccessToken(out.AccessToken)
	return nil
}

// do schickt den von build erzeugten Request mit dem aktuellen Access-Token.
// Bei 401 wird genau einmal refresht und der Request neu gebaut und
// wiederholt — build muss deshalb bei jedem Aufruf einen frischen Body liefern.
// Ein zweites 401 kommt als Antwort zurück (keine Schleife); scheitert schon
// der Refresh mit 401, kommt ErrSessionExpired.
func (c *Client) do(ctx context.Context, build func() (*http.Request, error)) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		req, err := build()
		if err != nil {
			return nil, err
		}
		if t := c.accessToken(); t != "" {
			req.Header.Set("Authorization", "Bearer "+t)
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusUnauthorized || attempt > 0 {
			return resp, nil
		}
		drain(resp)
		if err := c.refresh(ctx); err != nil {
			return nil, err
		}
	}
}

func (c *Client) getJSON(ctx context.Context, op, path string, v any) error {
	resp, err := c.do(ctx, func() (*http.Request, error) {
		return c.newRequest(ctx, http.MethodGet, path, nil)
	})
	if err != nil {
		return err
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return statusError(op, resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("%s: unerwartete Antwort des Servers: %w", op, err)
	}
	return nil
}

// Team ist eine Mannschaft aus GET /api/teams.
type Team struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

// Teams liefert die aktiven Mannschaften. Ob der Nutzer für eine davon
// hochladen darf, entscheidet der Server beim Anlegen (403).
func (c *Client) Teams(ctx context.Context) ([]Team, error) {
	var all []Team
	if err := c.getJSON(ctx, "Mannschaften laden", "/api/teams", &all); err != nil {
		return nil, err
	}
	active := all[:0]
	for _, t := range all {
		if t.IsActive {
			active = append(active, t)
		}
	}
	return active, nil
}

// ActiveSeasonID liefert die aktive Saison (GET /api/seasons). Die Route steht
// im selben Rechte-Tier wie der Upload (Trainer/sportl. Leitung/Vorstand) —
// ein 403 heißt deshalb schon hier: dieses Konto kann keine Videos hochladen.
func (c *Client) ActiveSeasonID(ctx context.Context) (int, error) {
	var seasons []struct {
		ID       int  `json:"id"`
		IsActive bool `json:"is_active"`
	}
	err := c.getJSON(ctx, "Saisons laden", "/api/seasons", &seasons)
	var se *StatusError
	if errors.As(err, &se) && se.Status == http.StatusForbidden {
		return 0, ErrNoUploadPermission
	}
	if err != nil {
		return 0, err
	}
	for _, s := range seasons {
		if s.IsActive {
			return s.ID, nil
		}
	}
	return 0, ErrNoActiveSeason
}

// Game ist ein Spiel aus GET /api/games.
type Game struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Opponent string `json:"opponent"`
	Teams    []struct {
		ID int `json:"id"`
	} `json:"teams"`
}

// Label bildet die Auswahl-Beschriftung „DD.MM.YYYY · Gegner" (wie im Web).
func (g Game) Label() string {
	d := g.Date
	if len(d) >= 10 {
		d = d[:10] // SQLite-DATE kommt als ISO-Timestamp
	}
	if p := strings.SplitN(d, "-", 3); len(p) == 3 {
		d = p[2] + "." + p[1] + "." + p[0]
	}
	opp := g.Opponent
	if opp == "" {
		opp = "Spiel"
	}
	return d + " · " + opp
}

// Games liefert die Spiele der Saison, an denen teamID beteiligt ist. Die
// Liste ist serverseitig auf 200 Einträge gedeckelt (httpx.Paging).
func (c *Client) Games(ctx context.Context, seasonID, teamID int) ([]Game, error) {
	var page struct {
		Items []Game `json:"items"`
	}
	path := fmt.Sprintf("/api/games?season_id=%d&limit=200", seasonID)
	if err := c.getJSON(ctx, "Spiele laden", path, &page); err != nil {
		return nil, err
	}
	var out []Game
	for _, g := range page.Items {
		for _, t := range g.Teams {
			if t.ID == teamID {
				out = append(out, g)
				break
			}
		}
	}
	return out, nil
}

// NewVideo ist der Body von POST /api/videos.
type NewVideo struct {
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	TeamID      int     `json:"team_id"`
	SeasonID    int     `json:"season_id"`
	GameID      *int    `json:"game_id,omitempty"`
	SizeBytes   int64   `json:"size_bytes"`
}

// CreateVideo legt die Video-Zeile an (POST /api/videos, status='uploading')
// und liefert ihre ID für die tus-Metadaten.
func (c *Client) CreateVideo(ctx context.Context, v NewVideo) (int, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	resp, err := c.do(ctx, func() (*http.Request, error) {
		req, err := c.newRequest(ctx, http.MethodPost, "/api/videos", bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req, err
	})
	if err != nil {
		return 0, err
	}
	defer drain(resp)
	switch resp.StatusCode {
	case http.StatusCreated:
		var out struct {
			VideoID int `json:"video_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.VideoID <= 0 {
			return 0, errors.New("Video anlegen: unerwartete Antwort des Servers")
		}
		return out.VideoID, nil
	case http.StatusForbidden:
		return 0, errors.New("keine Berechtigung, für diese Mannschaft Videos hochzuladen")
	case http.StatusInsufficientStorage:
		return 0, errors.New("der Speicher auf dem Server ist voll — bitte später erneut versuchen oder den Admin informieren")
	default:
		return 0, statusError("Video anlegen", resp)
	}
}

// statusError baut aus einer unerwarteten Antwort einen Fehler; der
// Wartungsmodus bekommt eine eigene, verständliche Meldung.
func statusError(op string, resp *http.Response) error {
	if resp.StatusCode == http.StatusServiceUnavailable && resp.Header.Get("X-Maintenance-Mode") == "1" {
		return ErrMaintenance
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return &StatusError{Op: op, Status: resp.StatusCode, Body: strings.TrimSpace(string(b))}
}

// drain liest den Rest des Bodys und schließt ihn, damit die Verbindung
// wiederverwendet werden kann.
func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
}
