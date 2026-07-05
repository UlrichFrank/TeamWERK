// Package matchreports implementiert den Autoren-Workflow für Spielberichte
// von der Draft-Erstellung durch einen Presseteam-User bis zum
// fire-and-forget-Publish an die TYPO3-Extension auf team-stuttgart.org.
//
// State-Machine (siehe openspec/changes/spielbericht-typo3-publisher/design.md):
//
//	draft ─publish─▶ publishing ─(2xx)─▶ published
//	                     │
//	                     └─(4xx/5xx)─▶ publish_failed
//	                                          │
//	                                          └─retry (manuell)─▶ publishing
//
// Nach `published` ist der Bericht in TeamWERK read-only. Änderungen und
// Löschungen laufen ausschließlich in der TYPO3-Backend-Redaktion.
package matchreports

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// State-Konstanten der match_reports.state-Spalte. CHECK-Constraint in
// Migration 019 spiegelt diese Menge.
const (
	StateDraft         = "draft"
	StatePublishing    = "publishing"
	StatePublished     = "published"
	StatePublishFailed = "publish_failed"
)

// EventCategory ist die SSE-Event-Kategorie, die Frontend-Seiten via
// useLiveUpdates abonnieren, um auf Draft-/Publish-Änderungen zu reagieren.
const EventCategory = "match-report-event"

// MaxImages ist das harte Limit für Bilder pro Bericht (spiegelt das
// Typo3-seitige Spike-Limit im Nachbar-Repo).
const MaxImages = 10

// Handler bündelt die Abhängigkeiten der matchreports-Routen.
// Publisher ist ein Interface, damit Tests einen In-Memory-Publisher
// injizieren können statt gegen echtes HTTP zu laufen.
type Handler struct {
	db        *sql.DB
	hub       *hub.EventHub
	cfg       *appconfig.Config
	publisher Publisher
}

// NewHandler baut einen Handler mit Default-HTTP-Publisher.
func NewHandler(db *sql.DB, h *hub.EventHub, cfg *appconfig.Config) *Handler {
	return &Handler{
		db:        db,
		hub:       h,
		cfg:       cfg,
		publisher: NewHTTPPublisher(cfg.TYPO3ImportURL, cfg.TYPO3ImportToken),
	}
}

// NewHandlerWithPublisher erlaubt Tests, einen Fake-Publisher einzuhängen.
func NewHandlerWithPublisher(db *sql.DB, h *hub.EventHub, cfg *appconfig.Config, p Publisher) *Handler {
	return &Handler{db: db, hub: h, cfg: cfg, publisher: p}
}

// writeJSON schreibt einen JSON-Body und setzt den Status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeErr ist die Kurzform für { "error": code } / { "error": code, "detail": ... }.
func writeErr(w http.ResponseWriter, status int, code string, detail ...string) {
	body := map[string]string{"error": code}
	if len(detail) > 0 && detail[0] != "" {
		body["detail"] = detail[0]
	}
	writeJSON(w, status, body)
}

// broadcast schickt eine SSE-Nachricht, falls der Hub gesetzt ist.
// (In Tests kann hub nil sein — dann still no-op.)
func (h *Handler) broadcast() {
	if h.hub != nil {
		h.hub.Broadcast(EventCategory)
	}
}

// logErr loggt Publisher-/Storage-Fehler strukturiert, ohne den Fehler zu re-throwen.
func logErr(msg string, err error, kv ...any) {
	slog.Error(msg, append([]any{"err", err}, kv...)...)
}
