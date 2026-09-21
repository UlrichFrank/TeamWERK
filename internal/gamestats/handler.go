package gamestats

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/background"
	"github.com/teamstuttgart/teamwerk/internal/bwhv"
	"github.com/teamstuttgart/teamwerk/internal/httpx"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

// EventStaffeln ist das SSE-Ereignis dieses Pakets. Es wird nach jedem Lauf
// gesendet, der etwas verändert hat — Push gibt es bewusst keine
// (design.md §10).
const EventStaffeln = "bwhv-updated"

// Handler bedient die HTTP-Routen.
type Handler struct {
	store   *Store
	hub     *hub.EventHub
	client  *bwhv.Client
	reports *ReportStore
	orgID   int
}

// NewHandler liefert einen Handler.
func NewHandler(db *sql.DB, h *hub.EventHub, client *bwhv.Client, reports *ReportStore, orgID int) *Handler {
	return &Handler{store: NewStore(db), hub: h, client: client, reports: reports, orgID: orgID}
}

// staffelResponse ist eine Staffel mit ihrer Mannschaft.
type staffelResponse struct {
	ID       int    `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	TeamName string `json:"teamName"`
	KaderID  int    `json:"kaderId"`
}

// ListStaffeln liefert die Staffeln der aktiven Saison.
func (h *Handler) ListStaffeln(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, err := h.store.ActiveSeason(ctx)
	if err != nil {
		if errors.Is(err, errNoSeason) {
			httpx.WriteJSON(w, http.StatusOK, []staffelResponse{})
			return
		}
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	list, err := h.store.ListStaffelnWithTeam(ctx, seasonID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if list == nil {
		list = []staffelResponse{}
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

// staffelOfRequest löst die Staffel-ID aus dem Pfad auf und prüft, dass sie zur
// aktiven Saison gehört. Eine Staffel einer fremden Saison ist 404, nicht 403 —
// ihre Existenz ist nichts, was verborgen werden müsste, aber sie gehört nicht
// zum aktuellen Betrieb.
func (h *Handler) staffelOfRequest(w http.ResponseWriter, r *http.Request) (Staffel, bool) {
	id, ok := httpx.PathID(r, "id")
	if !ok {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, nil)
		return Staffel{}, false
	}
	seasonID, err := h.store.ActiveSeason(r.Context())
	if err != nil {
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, err)
		return Staffel{}, false
	}
	st, err := h.store.StaffelByID(r.Context(), id, seasonID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, err)
		return Staffel{}, false
	case err != nil:
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return Staffel{}, false
	}
	return *st, true
}

// GetTable liefert den Tabellenstand einer Staffel.
func (h *Handler) GetTable(w http.ResponseWriter, r *http.Request) {
	st, ok := h.staffelOfRequest(w, r)
	if !ok {
		return
	}
	table, err := h.store.StaffelTable(r.Context(), st.ID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if table == nil {
		table = []bwhv.TableRow{}
	}
	httpx.WriteJSON(w, http.StatusOK, table)
}

// GetSchedule liefert den kompletten Spielplan einer Staffel.
func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	st, ok := h.staffelOfRequest(w, r)
	if !ok {
		return
	}
	games, err := h.store.StaffelGames(r.Context(), st.ID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if games == nil {
		games = []ScheduleGame{}
	}
	httpx.WriteJSON(w, http.StatusOK, games)
}

// GetRanglisten liefert die Saisonbilanz aller Spieler einer Staffel.
func (h *Handler) GetRanglisten(w http.ResponseWriter, r *http.Request) {
	st, ok := h.staffelOfRequest(w, r)
	if !ok {
		return
	}
	stats, err := h.store.StaffelStats(r.Context(), st.ID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if stats == nil {
		stats = []PlayerStat{}
	}
	httpx.WriteJSON(w, http.StatusOK, stats)
}

// GetReport liefert den ausgewerteten Bericht einer Begegnung.
func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(r, "id")
	if !ok {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, nil)
		return
	}
	detail, err := h.store.ReportForGame(r.Context(), id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, err)
		return
	case err != nil:
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, detail)
}

// GetReportPDF liefert das abgelegte Spielbericht-PDF.
//
// Content-Type wird explizit gesetzt und nicht geraten, Content-Disposition ist
// attachment — dasselbe Muster wie in internal/upload und internal/media nach
// der Sicherheitswelle 1.
func (h *Handler) GetReportPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(r, "id")
	if !ok {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, nil)
		return
	}
	path, err := h.store.ReportPDFPath(r.Context(), id)
	switch {
	case errors.Is(err, sql.ErrNoRows) || path == "":
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, sql.ErrNoRows)
		return
	case err != nil:
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	f, err := h.reports.Open(path)
	if err != nil {
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, err)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="spielbericht-`+strconv.Itoa(id)+`.pdf"`)
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// GetMemberStats liefert die Saisonbilanz eines Mitglieds.
func (h *Handler) GetMemberStats(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.PathID(r, "id")
	if !ok {
		httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, nil)
		return
	}
	ctx := r.Context()
	var exists int
	if err := h.store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM members WHERE id = ?`, id).Scan(&exists); err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if exists == 0 {
		httpx.WriteError(w, r, http.StatusNotFound, httpx.CodeNotFound, sql.ErrNoRows)
		return
	}
	seasonID, err := h.store.ActiveSeason(ctx)
	if err != nil {
		httpx.WriteJSON(w, http.StatusOK, []PlayerStat{})
		return
	}
	stats, err := h.store.MemberStats(ctx, id, seasonID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, err)
		return
	}
	if stats == nil {
		stats = []PlayerStat{}
	}
	httpx.WriteJSON(w, http.StatusOK, stats)
}

// GetKatalog liefert den Staffel-Katalog des Verbands und seiner Bezirke.
// Nur Vorstand/Admin — die Maske zur Kaderpflege braucht ihn, sonst niemand.
func (h *Handler) GetKatalog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgs, err := h.client.FetchOrgs(ctx, h.orgID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusBadGateway, "bwhv_unreachable", err)
		return
	}
	_, period, err := h.client.FetchPeriods(ctx, h.orgID)
	if err != nil {
		httpx.WriteError(w, r, http.StatusBadGateway, "bwhv_unreachable", err)
		return
	}

	type katalogEntry struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Org  string `json:"org"`
	}
	out := []katalogEntry{}
	subOrgs := []int{0}
	for id := range orgs {
		n, convErr := strconv.Atoi(id)
		if convErr != nil || n == h.orgID || n <= 0 {
			continue
		}
		subOrgs = append(subOrgs, n)
	}
	for _, sub := range subOrgs {
		classes, err := h.client.FetchCatalog(ctx, h.orgID, sub, period)
		if err != nil {
			continue // ein nicht erreichbarer Bezirk darf den Katalog nicht kippen
		}
		label := orgs[strconv.Itoa(h.orgID)]
		if sub > 0 {
			label = orgs[strconv.Itoa(sub)]
		}
		for _, c := range classes {
			out = append(out, katalogEntry{Code: c.Sname, Name: c.Lname, Org: label})
		}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// PollNow stößt den Abruf einer Staffel an, unabhängig vom Zeitfenster.
//
// Mutation: sie broadcastet, sonst schlägt das Broadcast-Gate zu. Der Lauf
// selbst läuft über background.Go mit eigenem Context — r.Context() ist mit
// der Antwort gecancelt und der Abruf liefe still ins Leere.
func (h *Handler) PollNow(w http.ResponseWriter, r *http.Request) {
	st, ok := h.staffelOfRequest(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	_ = claims
	seasonID := st.SeasonID
	poller := NewPoller(h.store, h.client, h.reports, h.orgID)

	background.Go("bwhv-poll-manual", func() {
		ctx, cancel := detachedContext()
		defer cancel()
		res, err := poller.PollStaffel(ctx, seasonID, st)
		if err != nil {
			return
		}
		if res.Changed() {
			h.hub.Broadcast(EventStaffeln)
		}
	})
	h.hub.Broadcast(EventStaffeln)
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "gestartet", "staffel": st.Code})
}
