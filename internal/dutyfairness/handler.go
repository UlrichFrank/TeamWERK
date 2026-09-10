package dutyfairness

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

type teamOption struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type ranglisteRow struct {
	Rank int `json:"rank"`
	// MemberID und Name sind für anonymisierte Zeilen null — die Oberfläche
	// zeigt dann ausschließlich die Platzierung.
	MemberID   *int    `json:"memberId"`
	Name       *string `json:"name"`
	IsOwn      bool    `json:"isOwn"`
	Geleistet  float64 `json:"geleistet"`
	Vorhersage float64 `json:"vorhersage"`
}

type ranglisteBlock struct {
	TeamID    int            `json:"teamId"`
	TeamLabel string         `json:"teamLabel"`
	Soll      float64        `json:"soll"`
	Rows      []ranglisteRow `json:"rows"`
}

type ranglisteResponse struct {
	// Teams ist der Scope des Nutzers = die Optionen des Team-Filters.
	Teams  []teamOption     `json:"teams"`
	Blocks []ranglisteBlock `json:"blocks"`
}

// Rangliste — GET /api/duty-fairness/rangliste?team=<id,id>
//
// Ein Endpoint für alle Sichten (design.md Entscheidung 5): Standard-Nutzer
// sehen nur Teams, in deren Kader ihr eigenes Mitglied oder ein Kind steht, und
// nur die verbundenen Zeilen mit Namen; admin/vorstand sehen alle Teams und
// alle Namen. Ohne `team` kommen alle Teams des Scopes; ein Team außerhalb des
// Scopes ist 403. Ungültige Teile der ID-Liste werden verworfen wie im
// Frontend-Filter (lib/teamFilter.ts).
func (h *Handler) Rangliste(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := auth.ClaimsFromCtx(ctx)
	resp := ranglisteResponse{Teams: []teamOption{}, Blocks: []ranglisteBlock{}}

	var seasonID int
	err := h.db.QueryRowContext(ctx, `SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`).Scan(&seasonID)
	if err == sql.ErrNoRows {
		writeJSON(w, resp)
		return
	}
	if err != nil {
		slog.Error("dutyfairness rangliste: active season", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	snap, err := Compute(ctx, h.db, seasonID)
	if err != nil {
		slog.Error("dutyfairness rangliste: compute", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	privileged := claims.Role == "admin" || claims.HasFunction("vorstand")
	linked := snap.LinkedMembers(claims.UserID)
	scope := snap.TeamOrder
	if !privileged {
		scope = snap.TeamsFor(linked)
	}

	requested := parseTeamIDs(r.URL.Query().Get("team"))
	for _, tid := range requested {
		if !slices.Contains(scope, tid) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	for _, tid := range scope {
		t := snap.Teams[tid]
		resp.Teams = append(resp.Teams, teamOption{ID: t.TeamID, Label: t.Label})
		if len(requested) > 0 && !slices.Contains(requested, tid) {
			continue
		}
		block := ranglisteBlock{TeamID: t.TeamID, TeamLabel: t.Label, Soll: Round2(t.Soll), Rows: []ranglisteRow{}}
		for i, m := range t.Ranked() {
			row := ranglisteRow{
				Rank:       i + 1,
				IsOwn:      linked[m.MemberID],
				Geleistet:  Round2(m.Geleistet),
				Vorhersage: Round2(m.Vorhersage),
			}
			if privileged || row.IsOwn {
				row.MemberID = &m.MemberID
				row.Name = &m.Name
			}
			block.Rows = append(block.Rows, row)
		}
		resp.Blocks = append(resp.Blocks, block)
	}

	writeJSON(w, resp)
}

func parseTeamIDs(raw string) []int {
	var out []int
	for _, part := range strings.Split(raw, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
