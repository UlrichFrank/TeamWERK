package games

// Ausrichter eines Spieltags lesen und ändern (heimspieltag-ausrichter,
// design.md Decision 7). Liegt im games-Package, weil beide Schreibwege den
// unexportierten runAutoRegen brauchen — dieselbe Begründung, aus der auch
// h4aimport_handler.go und bulkregen_handler.go hier liegen statt in ihrem
// fachlichen Heimat-Package.
//
// Vorschau und Anwenden gehen durch EINEN Kern (runGameDayHost) und
// unterscheiden sich ausschließlich im Abschluss: Commit oder das ohnehin
// laufende defer tx.Rollback(). Damit kann die Vorschau nicht vom Ergebnis
// abweichen — sie ist dasselbe Programm, nur ohne Commit. Der geerbte
// Trade-off aus duty-bulk-regen gilt auch hier: der Dry-Run ist eine echte
// Schreibtransaktion und serialisiert kurz gegen andere Schreiber. Bei einem
// einzelnen Tag ist das unkritisch.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/httpx"
	"github.com/teamstuttgart/teamwerk/internal/settings"
)

// --- Request/Response ---------------------------------------------------------------

type gameDayHostRequest struct {
	Date         string `json:"date"`
	AusrichterID int    `json:"ausrichter_id"`
}

// gameDayHostBalance ist die Bilanz eines Wechsels. Created/Deleted sind
// Slot-KAPAZITÄT (slots_total), nicht Zeilenzahl — dieselbe Einheit, die
// GameDelta und die Vor-/Nach-Zählung unten verwenden, damit
// slots_after - slots_before == created - deleted aufgeht und die Zahlen
// gegeneinander lesbar bleiben.
type gameDayHostBalance struct {
	Created           int `json:"created"`
	Deleted           int `json:"deleted"`
	AssignmentsKept   int `json:"assignments_kept"`
	AssignmentsLost   int `json:"assignments_lost"`
	SlotsBefore       int `json:"slots_before"`
	SlotsAfter        int `json:"slots_after"`
	AssignmentsBefore int `json:"assignments_before"`
	AssignmentsAfter  int `json:"assignments_after"`
}

type gameDayHostResponse struct {
	Date           string             `json:"date"`
	AusrichterID   int                `json:"ausrichter_id"`
	AusrichterName string             `json:"ausrichter_name"`
	IsExplicit     bool               `json:"is_explicit"`
	Balance        gameDayHostBalance `json:"balance"`
	Applied        bool               `json:"applied,omitempty"`
}

// hostAPIError trägt HTTP-Status und Maschinen-Code der validierten
// Fehlerfälle; alles andere ist ein 500.
type hostAPIError struct {
	status int
	code   string
}

func hostErr(status int, code string) *hostAPIError {
	return &hostAPIError{status: status, code: code}
}

// --- HTTP-Einstiegspunkte -------------------------------------------------------------

// GET /api/game-days/{date}/host
// Authenticated-Tier: der Kalender zeigt den Wert jedem Eingeloggten an,
// geändert wird er nur mit manage_games (die beiden POST-Routen unten).
func (h *Handler) GetGameDayHost(w http.ResponseWriter, r *http.Request) {
	date := dayKey(r.PathValue("date"))
	if _, err := time.Parse("2006-01-02", date); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "invalid_date", nil)
		return
	}
	ctx := r.Context()

	var seasonID int
	if err := h.db.QueryRowContext(ctx,
		`SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&seasonID); err != nil {
		httpx.WriteError(w, r, http.StatusBadRequest, "no_active_season", nil)
		return
	}

	id, explicit, err := settings.ResolveAusrichterForDayDetailed(ctx, h.db, date, seasonID)
	if err != nil {
		// ErrNoDefaultAusrichter ist ein Datenfehler, kein Betriebszustand
		// (Migration 048 legt die Default-Zeile idempotent an) — deshalb 500
		// statt eines erfundenen Ersatzwerts.
		httpx.WriteError(w, r, http.StatusInternalServerError, "no_default_ausrichter",
			fmt.Errorf("resolve ausrichter for day %s: %w", date, err))
		return
	}
	resolved, err := settings.GetAusrichter(ctx, h.db, id)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal,
			fmt.Errorf("load ausrichter %d: %w", id, err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, gameDayHostResponse{
		Date: date, AusrichterID: id, AusrichterName: resolved.Name, IsExplicit: explicit,
	})
}

// POST /api/game-days/host/preview
// Vollständiger Dry-Run desselben Codepfads wie Apply, abgeschlossen mit
// Rollback. Schreibt nichts — steht deshalb mit Begründung in der
// broadcastAllowlist (internal/arch/broadcast_test.go).
func (h *Handler) PreviewGameDayHost(w http.ResponseWriter, r *http.Request) {
	resp, _, apiErr := h.runGameDayHost(r.Context(), r, false)
	if apiErr != nil {
		httpx.WriteError(w, r, apiErr.status, apiErr.code, nil)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// POST /api/game-days/host/apply
// Identischer Plan wie Preview, aber committet und broadcastet. Der Wechsel
// betrifft alle Termine des Tages und damit potenziell mehrere Mannschaften —
// deshalb der globale Broadcast wie beim Massenlauf, nicht die
// spielbezogene broadcastGame-Variante.
func (h *Handler) ApplyGameDayHost(w http.ResponseWriter, r *http.Request) {
	resp, summary, apiErr := h.runGameDayHost(r.Context(), r, true)
	if apiErr != nil {
		httpx.WriteError(w, r, apiErr.status, apiErr.code, nil)
		return
	}
	resp.Applied = true
	if h.hub != nil {
		h.hub.Broadcast("duties")
		h.hub.Broadcast("games")
	}
	// Wer durch den Wechsel seinen Dienst verliert, erfährt es — wie bei jedem
	// anderen Regen-Pfad auch. Erst nach dem Commit, sonst benachrichtigt ein
	// zurückgerollter Lauf ins Leere.
	h.dispatchRegenNotifications(summary)
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// --- Gemeinsamer Kern ------------------------------------------------------------------

// runGameDayHost setzt den Tages-Ausrichter und regeneriert den Tag innerhalb
// EINER Transaktion; apply=true committet, sonst greift das defer tx.Rollback().
// Liefert zusätzlich das RegenSummary, damit der Apply-Wrapper nach dem Commit
// benachrichtigen kann.
func (h *Handler) runGameDayHost(ctx context.Context, r *http.Request, apply bool) (gameDayHostResponse, RegenSummary, *hostAPIError) {
	var req gameDayHostRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	// SQLite-DATE-Gotcha (docs/agent/06-gotchas.md), hier auf der SCHREIBseite:
	// die Leseseite (ResolveAusrichterForDay) normalisiert zwar ihren
	// Eingabeparameter, vergleicht ihn aber gegen den GESPEICHERTEN Spaltenwert.
	// Landete hier ein ISO-Timestamp ("2026-09-14T00:00:00Z") in der Spalte,
	// matchte die Auflösung nie und fiele still auf den Default zurück — der
	// Wechsel wäre gespeichert und trotzdem wirkungslos. Deshalb wird das Datum
	// VOR jeder Verwendung auf die reine Form gekürzt.
	date := dayKey(req.Date)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusBadRequest, "invalid_date")
	}

	var updatedBy any
	if claims := auth.ClaimsFromCtx(ctx); claims != nil {
		updatedBy = claims.UserID
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
	}
	defer tx.Rollback() //nolint:errcheck // No-Op nach erfolgreichem Commit

	var seasonID int
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM seasons WHERE is_active=1 LIMIT 1`).Scan(&seasonID); err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusBadRequest, "no_active_season")
	}

	// Ein inaktiver Ausrichter ist kein gültiges Ziel: die Liste blendet ihn für
	// Standard-Nutzer gar nicht ein, und ein Tag, der auf einen ausgemusterten
	// Verein zeigt, wäre genau der Zustand, den UpdateAusrichter beim Default
	// mit 409 verhindert. Beides vor dem ersten Schreibvorgang.
	target, err := settings.GetAusrichter(ctx, tx, req.AusrichterID)
	if errors.Is(err, settings.ErrAusrichterNotFound) {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusBadRequest, "unknown_ausrichter")
	}
	if err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
	}
	if !target.Aktiv {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusBadRequest, "inactive_ausrichter")
	}

	slotsBefore, assignmentsBefore, err := countDayDuties(ctx, tx, seasonID, date)
	if err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO spieltag_ausrichter (date, season_id, ausrichter_id, updated_at, updated_by)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(date, season_id) DO UPDATE SET
			ausrichter_id = excluded.ausrichter_id,
			updated_at    = CURRENT_TIMESTAMP,
			updated_by    = excluded.updated_by`,
		date, seasonID, target.ID, updatedBy); err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
	}

	summary, err := h.runAutoRegen(ctx, tx, []string{date}, seasonID, nil)
	if err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
	}

	slotsAfter, assignmentsAfter, err := countDayDuties(ctx, tx, seasonID, date)
	if err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
	}

	balance := gameDayHostBalance{
		SlotsBefore:       slotsBefore,
		SlotsAfter:        slotsAfter,
		AssignmentsBefore: assignmentsBefore,
		AssignmentsAfter:  assignmentsAfter,
	}
	// PerGame statt der Day-Level-Listen aus RegenSummary: die sind bei
	// summaryCap gekappt, PerGame ist es bewusst nicht — eine gekappte Bilanz
	// wäre schlicht falsch.
	for _, pg := range summary.PerGame {
		balance.Created += pg.Created
		balance.Deleted += pg.DeletedAuto
		balance.AssignmentsKept += pg.AssignmentsKept
		balance.AssignmentsLost += pg.AssignmentsLost
	}

	// Innerhalb der Transaktion zurücklesen statt target.ID durchzureichen: das
	// prüft nebenbei, dass der geschriebene Datumswert von der Auflösung auch
	// gefunden wird (siehe DATE-Gotcha oben) — sonst käme hier is_explicit=false
	// zurück, obwohl gerade explizit geschrieben wurde.
	resolvedID, explicit, err := settings.ResolveAusrichterForDayDetailed(ctx, tx, date, seasonID)
	if err != nil {
		return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
	}

	resp := gameDayHostResponse{
		Date:           date,
		AusrichterID:   resolvedID,
		AusrichterName: target.Name,
		IsExplicit:     explicit,
		Balance:        balance,
	}

	if apply {
		if err := tx.Commit(); err != nil {
			return gameDayHostResponse{}, RegenSummary{}, hostErr(http.StatusInternalServerError, httpx.CodeInternal)
		}
	}
	// Preview: das defer tx.Rollback() oben läuft beim Return — kein Schreibvorgang überlebt.

	return resp, summary, nil
}

// --- Helfer -----------------------------------------------------------------------------

// dayKey kürzt ein Datum auf die reine "2006-01-02"-Form (SQLite-DATE-Gotcha).
func dayKey(date string) string {
	if len(date) > 10 {
		return date[:10]
	}
	return date
}

// countDayDuties zählt die Dienste der Termine eines Spieltags: einmal als
// Slot-KAPAZITÄT (Summe slots_total, dieselbe Einheit wie GameDelta.Created)
// und einmal als Zahl der Zuweisungen. Anker sind die games des Tages, weil
// genau deren Slots der Regen anfasst — terminlose Slots bleiben unberührt.
func countDayDuties(ctx context.Context, tx *sql.Tx, seasonID int, date string) (slots, assignments int, err error) {
	if err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(slots_total), 0) FROM duty_slots
		WHERE game_id IN (SELECT id FROM games WHERE season_id=? AND date=?)`,
		seasonID, date).Scan(&slots); err != nil {
		return 0, 0, err
	}
	if err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM duty_assignments
		WHERE duty_slot_id IN (
			SELECT id FROM duty_slots
			WHERE game_id IN (SELECT id FROM games WHERE season_id=? AND date=?))`,
		seasonID, date).Scan(&assignments); err != nil {
		return 0, 0, err
	}
	return slots, assignments, nil
}
