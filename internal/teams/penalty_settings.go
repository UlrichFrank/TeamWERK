package teams

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Strafen-Einheiten (Euro | Striche) pro Kader. amount_cent ist überall echte
// Cent; ein Strich wird als 100 Cent gespeichert (feste Rate 1 € = 1 Strich).
// Die Einheit (penalty_settings.unit) steuert nur Anzeige und Ganzzahl-Validierung
// (Striche = amount_cent muss durch 100 teilbar sein). Ein Wechsel rechnet Katalog
// und alle vergebenen Strafen des Teams in einer TX um:
//   - Euro → Striche: aufrunden auf das nächste Vielfache von 100 (niemand kommt
//     durch Rundung billiger weg).
//   - Striche → Euro: verlustfrei (Werte sind bereits Vielfache von 100).

// PenaltySettings ist die aktuelle Einheit eines Kaders.
type PenaltySettings struct {
	Unit string `json:"unit"`
}

// penaltyPreviewEntry beschreibt eine durch den Wechsel betroffene Row (Katalog
// oder Strafe) mit altem/neuem amount_cent.
type penaltyPreviewEntry struct {
	ID        int    `json:"id"`
	Label     string `json:"label"`
	OldAmount int    `json:"oldAmount"`
	NewAmount int    `json:"newAmount"`
}

// PenaltyPreview ist die Vorschau eines Einheiten-Wechsels ohne DB-Mutation.
type PenaltyPreview struct {
	From      string                `json:"from"`
	To        string                `json:"to"`
	Affected  int                   `json:"affected"`
	RoundedUp int                   `json:"roundedUp"`
	Penalties []penaltyPreviewEntry `json:"penalties"`
	Catalog   []penaltyPreviewEntry `json:"catalog"`
}

// currentPenaltyUnit liefert die Einheit des (primären) Kaders dieses Teams in der
// aktiven Saison. Fehlt die penalty_settings-Row (Kader nach der Backfill-Migration
// angelegt), gilt der Default 'euro'. Genutzt vom Read-Endpoint und von der
// Ganzzahl-Validierung in penalties.go.
func (h *Handler) currentPenaltyUnit(ctx context.Context, teamID int) (string, error) {
	kaderID, err := h.resolveKaderID(ctx, teamID)
	if errors.Is(err, sql.ErrNoRows) {
		return "euro", nil
	}
	if err != nil {
		return "", err
	}
	var unit string
	err = h.db.QueryRowContext(ctx,
		`SELECT unit FROM penalty_settings WHERE kader_id = ?`, kaderID).Scan(&unit)
	if errors.Is(err, sql.ErrNoRows) {
		return "euro", nil
	}
	if err != nil {
		return "", err
	}
	return unit, nil
}

// convertCent rechnet einen Cent-Betrag in die Ziel-Einheit um. Euro → Striche
// rundet auf das nächste Vielfache von 100 auf; Striche → Euro normalisiert (bei
// gültigen Bestandsdaten ein No-Op). Beträge sind stets > 0.
func convertCent(amountCent int, toUnit string) int {
	if toUnit == "striche" {
		return ((amountCent + 99) / 100) * 100
	}
	return (amountCent / 100) * 100
}

// GetPenaltySettings — GET /api/teams/{id}/penalty-settings.
// Gate: canReadPenalties (Team-Interne). Liefert die aktuelle Einheit.
func (h *Handler) GetPenaltySettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.canReadPenalties(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	unit, err := h.currentPenaltyUnit(ctx, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeRespJSON(w, http.StatusOK, PenaltySettings{Unit: unit})
}

// PreviewPenaltySettings — GET /api/teams/{id}/penalty-settings/preview?to=<unit>.
// Gate: isTrainerOfTeam. Liefert die Delta-Liste (Katalog + Strafen) mit alten/
// neuen Beträgen und der Anzahl aufgerundeter Rows, OHNE DB-Mutation.
func (h *Handler) PreviewPenaltySettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isTrainerOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	to := r.URL.Query().Get("to")
	if to != "euro" && to != "striche" {
		http.Error(w, "to must be euro or striche", http.StatusBadRequest)
		return
	}

	from, err := h.currentPenaltyUnit(ctx, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	preview := PenaltyPreview{
		From:      from,
		To:        to,
		Penalties: []penaltyPreviewEntry{},
		Catalog:   []penaltyPreviewEntry{},
	}

	// Katalog-Einträge des Teams (aktive Saison).
	catRows, err := h.db.QueryContext(ctx, `
		SELECT pt.id, pt.reason, pt.default_amount_cent
		FROM penalty_types pt
		JOIN kader k   ON k.id = pt.kader_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ORDER BY pt.reason`, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer catRows.Close()
	for catRows.Next() {
		var e penaltyPreviewEntry
		if err := catRows.Scan(&e.ID, &e.Label, &e.OldAmount); err != nil {
			continue
		}
		e.NewAmount = convertCent(e.OldAmount, to)
		if e.NewAmount != e.OldAmount {
			preview.Affected++
			if to == "striche" && e.OldAmount%100 != 0 {
				preview.RoundedUp++
			}
		}
		preview.Catalog = append(preview.Catalog, e)
	}

	// Vergebene Strafen des Teams (aktive Saison).
	penRows, err := h.db.QueryContext(ctx, `
		SELECT tp.id, m.first_name || ' ' || m.last_name, tp.amount_cent
		FROM team_penalties tp
		JOIN members m ON m.id = tp.member_id
		JOIN kader k   ON k.id = tp.kader_id
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ORDER BY tp.created_at, tp.id`, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer penRows.Close()
	for penRows.Next() {
		var e penaltyPreviewEntry
		if err := penRows.Scan(&e.ID, &e.Label, &e.OldAmount); err != nil {
			continue
		}
		e.NewAmount = convertCent(e.OldAmount, to)
		if e.NewAmount != e.OldAmount {
			preview.Affected++
			if to == "striche" && e.OldAmount%100 != 0 {
				preview.RoundedUp++
			}
		}
		preview.Penalties = append(preview.Penalties, e)
	}

	writeRespJSON(w, http.StatusOK, preview)
}

// SetPenaltySettings — PUT /api/teams/{id}/penalty-settings. Body {"unit":"..."}.
// Gate: isTrainerOfTeam. Rechnet Katalog + alle team_penalties des Teams in einer
// TX um und setzt die Einheit auf allen Kadern des Teams. Broadcast
// penalty-settings + penalties (Beträge in Rows mutieren). 404 ohne aktiven Kader.
func (h *Handler) SetPenaltySettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)

	teamID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}
	if ok, err := h.isTrainerOfTeam(ctx, claims, teamID); err != nil || !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var body struct {
		Unit string `json:"unit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Unit != "euro" && body.Unit != "striche" {
		http.Error(w, "unit must be euro or striche", http.StatusBadRequest)
		return
	}

	if _, err := h.resolveKaderID(ctx, teamID); errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "no active kader for team", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// No-Op bei gleicher Einheit: nicht umrechnen (sonst würde die Euro-
	// Normalisierung bereits gültige Beträge unnötig floor'en). Idempotent 200.
	if cur, cErr := h.currentPenaltyUnit(ctx, teamID); cErr != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if cur == body.Unit {
		writeRespJSON(w, http.StatusOK, PenaltySettings{Unit: body.Unit})
		return
	}

	// Ganze Umrechnung atomar: Katalog + Strafen + Einheit. Scheitert ein UPDATE,
	// bleibt alles beim Alten (keine Halb-Umrechnung).
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// SQLite-Ganzzahl-Ausdruck: Euro → Striche rundet auf (+99 vor Div/100),
	// Striche → Euro normalisiert auf das Vielfache von 100. Beträge sind stets > 0,
	// daher ist die trunkierende Ganzzahl-Division korrekt.
	catExpr, penExpr := "(default_amount_cent / 100) * 100", "(amount_cent / 100) * 100"
	if body.Unit == "striche" {
		catExpr, penExpr = "((default_amount_cent + 99) / 100) * 100", "((amount_cent + 99) / 100) * 100"
	}

	teamKaderFilter := `kader_id IN (
		SELECT k.id FROM kader k
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1)`

	if _, err := tx.ExecContext(ctx,
		`UPDATE penalty_types SET default_amount_cent = `+catExpr+` WHERE `+teamKaderFilter, teamID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE team_penalties SET amount_cent = `+penExpr+` WHERE `+teamKaderFilter, teamID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Einheit auf allen Kadern des Teams setzen (Upsert — neue Kader haben evtl.
	// keine Backfill-Row).
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO penalty_settings (kader_id, unit)
		SELECT k.id, ? FROM kader k
		JOIN seasons s ON s.id = k.season_id
		WHERE k.team_id = ? AND s.is_active = 1
		ON CONFLICT(kader_id) DO UPDATE SET unit = excluded.unit`,
		body.Unit, teamID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.hub.Broadcast("penalty-settings")
	h.hub.Broadcast("penalties")
	writeRespJSON(w, http.StatusOK, PenaltySettings{Unit: body.Unit})
}
