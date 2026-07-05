package matchreports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Publisher abstrahiert den TYPO3-Import-Endpoint. Tests injizieren einen
// In-Memory-Publisher.
type Publisher interface {
	Publish(ctx context.Context, req *PublishRequest) (*PublishResult, error)
}

// PublishRequest sammelt alles, was der TYPO3-Endpoint für den Import braucht.
type PublishRequest struct {
	// Meta wird als JSON-Feld `meta` verschickt.
	Meta PublishMeta
	// Images liefert die Bilder-Datei-Pfade (im Storage) samt Caption.
	// Reihenfolge = Reihenfolge im Multipart (image_0, image_1, …).
	Images []PublishImage
}

// PublishMeta ist der JSON-Blob, den der TYPO3-Endpoint erwartet.
// Feld-Namen matchen scripts/spike-match-report-import/fixture-payload.json
// im Nachbar-Repo. Contract-Version nach AC-8: `season` (String
// "YYYY-YYYY") statt `pid` — die Extension legt den Season-Ordner
// /spielberichte/{YYYY-YYYY}/ selbst an, falls er noch nicht existiert.
type PublishMeta struct {
	Title string `json:"title"`
	// Slug ist NUR das letzte Pfad-Segment (title-slug), z. B.
	// "tws-ma-vs-vfl-kirchheim". Den vollen Pfad
	// /spielberichte/{YYYY-YYYY}/{slug} baut die Extension.
	Slug string `json:"slug"`
	// Season ist das Format-Segment "YYYY-YYYY" (z. B. "2026-2027"). Die
	// Extension legt darunter einen Ordner-Knoten an, falls er noch nicht
	// existiert; darunter dann die pages-Zeile (doktype=126).
	Season           string `json:"season"`
	Abstract         string `json:"abstract"`
	MatchDate        int64  `json:"match_date"`
	MatchScore       string `json:"match_score"`
	MatchTeams       string `json:"match_teams"`
	Tournament       bool   `json:"tournament"`
	TeamCategoryUID  int    `json:"team_category_uid"`
	BodyHTML         string `json:"body_html"`
	ExternalReportID string `json:"external_report_id"`

	// Images-Metadaten (Caption pro Bild), in Reihenfolge.
	Images []PublishImageMeta `json:"images"`
}

// PublishImageMeta ist die JSON-Repräsentation der Caption pro Bild.
type PublishImageMeta struct {
	Caption string `json:"caption"`
}

// PublishImage bündelt die tatsächliche Datei und Caption.
type PublishImage struct {
	Path    string
	Caption string
}

// PublishResult ist die Erfolgs-Antwort des TYPO3-Endpoints.
type PublishResult struct {
	PageUID int    `json:"pageUid"`
	URL     string `json:"url"`
}

// HTTPPublisher spricht den TYPO3-Endpoint per multipart-POST an.
type HTTPPublisher struct {
	URL    string
	Token  string
	Client *http.Client
}

// NewHTTPPublisher baut einen Publisher mit sinnvollen HTTP-Defaults.
// Leere URL/Token: Publisher liefert publishConfigError beim ersten Aufruf,
// damit der Handler das sauber melden kann (kein Nil-Deref).
func NewHTTPPublisher(url, token string) *HTTPPublisher {
	return &HTTPPublisher{
		URL:   url,
		Token: token,
		// 60s reicht für 10 Bilder à ~5 MB gegen Mittwald mit ADSL-Upstream.
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

// ErrPublisherNotConfigured wird geliefert, wenn URL/Token leer sind.
var ErrPublisherNotConfigured = errors.New("typo3 publisher not configured (TYPO3_IMPORT_URL/TYPO3_IMPORT_TOKEN missing)")

// Publish setzt den multipart-Request zusammen und feuert ihn ab.
func (p *HTTPPublisher) Publish(ctx context.Context, req *PublishRequest) (*PublishResult, error) {
	if p.URL == "" || p.Token == "" {
		return nil, ErrPublisherNotConfigured
	}

	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)

	// meta-Feld schreiben.
	metaBytes, err := json.Marshal(req.Meta)
	if err != nil {
		return nil, fmt.Errorf("marshal meta: %w", err)
	}
	metaPart, err := mp.CreateFormField("meta")
	if err != nil {
		return nil, fmt.Errorf("create meta part: %w", err)
	}
	if _, err := metaPart.Write(metaBytes); err != nil {
		return nil, fmt.Errorf("write meta: %w", err)
	}

	// Bilder anhängen.
	for i, img := range req.Images {
		fieldName := fmt.Sprintf("image_%d", i)
		filename := filepath.Base(img.Path)
		part, err := mp.CreateFormFile(fieldName, filename)
		if err != nil {
			return nil, fmt.Errorf("create image part %d: %w", i, err)
		}
		f, err := os.Open(img.Path)
		if err != nil {
			return nil, fmt.Errorf("open image %s: %w", img.Path, err)
		}
		if _, err := io.Copy(part, f); err != nil {
			f.Close()
			return nil, fmt.Errorf("copy image %s: %w", img.Path, err)
		}
		f.Close()
	}
	if err := mp.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, &buf)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.Token)
	httpReq.Header.Set("Content-Type", mp.FormDataContentType())

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &PublisherError{
			Status: resp.StatusCode,
			Body:   string(body),
		}
	}

	var result PublishResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w — body=%s", err, string(body))
	}
	return &result, nil
}

// PublisherError bündelt Non-2xx-Antworten vom TYPO3-Endpoint.
type PublisherError struct {
	Status int
	Body   string
}

func (e *PublisherError) Error() string {
	return fmt.Sprintf("typo3 publisher status %d: %s", e.Status, e.Body)
}
