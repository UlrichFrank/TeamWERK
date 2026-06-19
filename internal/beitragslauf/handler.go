package beitragslauf

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/sepa"
)

// Exclusion-/Warnungs-Codes
const (
	exclStatusInaktiv  = "status_inaktiv"
	exclBeitragsfrei   = "beitragsfrei"
	exclKeinMandat     = "kein_sepa_mandat"
	exclIBANFehlt      = "iban_fehlt"
	exclIBANUngueltig  = "iban_ungueltig"
	exclKeineMitglNr   = "mitgliedsnummer_fehlt"
	exclAdresse        = "adresse_unvollstaendig"
	exclKeinSatz       = "kein_beitragssatz"
	warnHomeClubUnklar = "home_club_unklar"
)

var kategorieLabel = map[string]string{
	"aktiv_mit":  "Aktiv (mit Stammverein)",
	"aktiv_ohne": "Aktiv (ohne Stammverein)",
	"passiv":     "Passiv",
}

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
	dir string // BeitragslaufDir für Protokolle
}

func NewHandler(db *sql.DB, h *hub.EventHub, dir string) *Handler {
	return &Handler{db: db, hub: h, dir: dir}
}

// PreviewItem entspricht einer Zeile der Vorschau.
type PreviewItem struct {
	MemberID       int      `json:"member_id"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	Kategorie      string   `json:"kategorie,omitempty"`
	KategorieLabel string   `json:"kategorie_label,omitempty"`
	BetragCent     int      `json:"betrag_cent,omitempty"`
	Included       bool     `json:"included"`
	Warnings       []string `json:"warnings"`
	Exclusions     []string `json:"exclusions"`

	// intern für Export (nicht serialisiert)
	row MemberRow
}

type previewResult struct {
	SaisonID    int
	SaisonLabel string
	SaisonKurz  string
	Faelligkeit time.Time
	Items       []PreviewItem
}

func (h *Handler) buildPreview(ctx context.Context, saisonID int) (*previewResult, error) {
	label, saisonStart, err := h.loadSeason(ctx, saisonID)
	if err != nil {
		return nil, err
	}
	members, err := LoadMembersForLauf(h.db)
	if err != nil {
		return nil, err
	}
	saetze, err := LoadSaetzeMap(h.db)
	if err != nil {
		return nil, err
	}
	res := &previewResult{SaisonID: saisonID, SaisonLabel: label, SaisonKurz: label, Faelligkeit: saisonStart}
	for _, m := range members {
		res.Items = append(res.Items, computeItem(m, saetze, saisonStart))
	}
	return res, nil
}

func computeItem(m MemberRow, saetze map[string][]Satz, saisonStart time.Time) PreviewItem {
	it := PreviewItem{
		MemberID:   m.ID,
		Name:       m.FirstName + " " + m.LastName,
		Status:     m.Status,
		Warnings:   []string{},
		Exclusions: []string{},
		row:        m,
	}
	gruppe := BeitragsGruppe(m.Status)
	if gruppe == "" {
		it.Exclusions = append(it.Exclusions, exclStatusInaktiv)
	}
	if m.Beitragsfrei {
		it.Exclusions = append(it.Exclusions, exclBeitragsfrei)
	}
	if m.MemberNumber == "" {
		it.Exclusions = append(it.Exclusions, exclKeineMitglNr)
	}
	if !m.SepaMandat || m.SepaMandatPath == "" {
		it.Exclusions = append(it.Exclusions, exclKeinMandat)
	}
	if m.IBAN == "" {
		it.Exclusions = append(it.Exclusions, exclIBANFehlt)
	} else if !sepa.IsValidIBAN(m.IBAN) {
		it.Exclusions = append(it.Exclusions, exclIBANUngueltig)
	}
	if m.Street == "" || m.Zip == "" || m.City == "" {
		it.Exclusions = append(it.Exclusions, exclAdresse)
	}

	if len(it.Exclusions) > 0 {
		it.Included = false
		return it
	}

	// Kategorie bestimmen
	var kategorie string
	if gruppe == "passiv" {
		kategorie = "passiv"
	} else {
		match := MatchHomeClub(m.HomeClub)
		if match.Warning != "" {
			it.Warnings = append(it.Warnings, warnHomeClubUnklar)
		}
		kategorie = AktivKategorie(match.Matched)
	}

	betrag, err := LookupBetragCent(saetze, kategorie, saisonStart)
	if err != nil {
		it.Included = false
		it.Exclusions = append(it.Exclusions, exclKeinSatz)
		return it
	}
	it.Included = true
	it.Kategorie = kategorie
	it.KategorieLabel = kategorieLabel[kategorie]
	it.BetragCent = betrag
	return it
}

// loadSeason liefert Label (name) und den Abrechnungs-Stichtag 01.07. des
// Saison-Startjahres.
func (h *Handler) loadSeason(ctx context.Context, id int) (label string, stichtag time.Time, err error) {
	var name, startDate string
	err = h.db.QueryRowContext(ctx, `SELECT name, start_date FROM seasons WHERE id=?`, id).Scan(&name, &startDate)
	if err != nil {
		return "", time.Time{}, err
	}
	start, perr := time.Parse("2006-01-02", startDate[:min(10, len(startDate))])
	if perr != nil {
		return "", time.Time{}, perr
	}
	return name, time.Date(start.Year(), time.July, 1, 0, 0, 0, 0, time.UTC), nil
}

// GET /api/fee-run/preview?saison_id=
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	saisonID, err := strconv.Atoi(r.URL.Query().Get("saison_id"))
	if err != nil {
		http.Error(w, "saison_id fehlt oder ungültig", http.StatusBadRequest)
		return
	}
	pr, err := h.buildPreview(r.Context(), saisonID)
	if err != nil {
		http.Error(w, "Saison nicht gefunden", http.StatusNotFound)
		return
	}
	var inc, exc, warn, total int
	for _, it := range pr.Items {
		if it.Included {
			inc++
			total += it.BetragCent
			if len(it.Warnings) > 0 {
				warn++
			}
		} else {
			exc++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"saison_id":    pr.SaisonID,
		"saison_label": pr.SaisonLabel,
		"faelligkeit":  pr.Faelligkeit.Format("2006-01-02"),
		"items":        pr.Items,
		"summary": map[string]any{
			"included_count": inc,
			"excluded_count": exc,
			"warned_count":   warn,
			"total_cent":     total,
		},
	})
}

// POST /api/fee-run/export  {saison_id, member_ids}
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SaisonID  int   `json:"saison_id"`
		MemberIDs []int `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	club, err := h.loadClubSepa(r.Context())
	if err != nil || club.GlaeubigerID == "" || club.IBAN == "" || club.BIC == "" || club.Kontoinhaber == "" {
		http.Error(w, "Vereins-SEPA-Stammdaten unvollständig", http.StatusBadRequest)
		return
	}
	pr, err := h.buildPreview(r.Context(), req.SaisonID)
	if err != nil {
		http.Error(w, "Saison nicht gefunden", http.StatusNotFound)
		return
	}
	byID := map[int]PreviewItem{}
	for _, it := range pr.Items {
		byID[it.MemberID] = it
	}
	items := make([]ExportItem, 0, len(req.MemberIDs))
	for _, id := range req.MemberIDs {
		it, ok := byID[id]
		if !ok || !it.Included {
			http.Error(w, fmt.Sprintf("Mitglied %d ist ausgeschlossen oder unbekannt", id), http.StatusBadRequest)
			return
		}
		name := it.row.AccountHolder
		if name == "" {
			name = it.Name
		}
		items = append(items, ExportItem{
			MemberID: it.MemberID, Name: name,
			Street: it.row.Street, Zip: it.row.Zip, City: it.row.City,
			IBAN: sepa.NormalizeIBAN(it.row.IBAN), BetragCent: it.BetragCent,
			MandatRef: it.row.MemberNumber, MandatDatum: it.row.SepaMandatDate, MemberNumber: it.row.MemberNumber,
		})
	}
	xmlBytes, err := BuildXML(BuildInput{
		SaisonKurz: pr.SaisonKurz, ClubName: club.Name, GlaeubigerID: club.GlaeubigerID,
		ClubIBAN: club.IBAN, BIC: club.BIC, Kontoinhaber: club.Kontoinhaber,
		Faelligkeit: nextBusinessDay(pr.Faelligkeit), CreatedAt: time.Now(), Items: items,
	})
	if err != nil {
		http.Error(w, "XML-Erzeugung fehlgeschlagen", http.StatusInternalServerError)
		return
	}
	filename := "beitragslauf_" + saisonStamp(pr.SaisonKurz) + ".xml"
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Write(xmlBytes)
}

// POST /api/fee-run/confirm  {saison_id, results}
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SaisonID int `json:"saison_id"`
		Results  []struct {
			MemberID   int  `json:"member_id"`
			BetragCent int  `json:"betrag_cent"`
			Success    bool `json:"success"`
		} `json:"results"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	label, _, err := h.loadSeason(r.Context(), req.SaisonID)
	if err != nil {
		http.Error(w, "Saison nicht gefunden", http.StatusNotFound)
		return
	}
	results := make([]ProtokollResult, 0, len(req.Results))
	var okCount, failCount, sumOK int
	for _, rr := range req.Results {
		var first, last, memberNr string
		h.db.QueryRowContext(r.Context(),
			`SELECT first_name, last_name, COALESCE(member_number,'') FROM members WHERE id=?`, rr.MemberID).
			Scan(&first, &last, &memberNr)
		results = append(results, ProtokollResult{
			MemberNumber: memberNr, Name: first + " " + last, BetragCent: rr.BetragCent, Success: rr.Success,
		})
		if rr.Success {
			okCount++
			sumOK += rr.BetragCent
		} else {
			failCount++
		}
	}
	user := "unbekannt"
	if claims := auth.ClaimsFromCtx(r.Context()); claims != nil {
		user = fmt.Sprintf("%s (User #%d)", claims.Email, claims.UserID)
	}
	if err := AppendProtokoll(h.dir, label, user, time.Now(), results); err != nil {
		http.Error(w, "Protokoll konnte nicht geschrieben werden", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"saison_label":           label,
		"erfolgreich":            okCount,
		"nicht_erfolgreich":      failCount,
		"summe_erfolgreich_cent": sumOK,
	})
}

// GET /api/fee-run/protocol?saison_id=
func (h *Handler) Protocol(w http.ResponseWriter, r *http.Request) {
	saisonID, err := strconv.Atoi(r.URL.Query().Get("saison_id"))
	if err != nil {
		http.Error(w, "saison_id fehlt oder ungültig", http.StatusBadRequest)
		return
	}
	label, _, err := h.loadSeason(r.Context(), saisonID)
	if err != nil {
		http.Error(w, "Saison nicht gefunden", http.StatusNotFound)
		return
	}
	data, err := ReadProtokoll(h.dir, label)
	if err != nil {
		http.Error(w, "Protokoll konnte nicht gelesen werden", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

type clubSepa struct {
	Name         string
	GlaeubigerID string
	IBAN         string
	BIC          string
	Kontoinhaber string
}

func (h *Handler) loadClubSepa(ctx context.Context) (clubSepa, error) {
	var c clubSepa
	var name string
	var g, i, b, k sql.NullString
	err := h.db.QueryRowContext(ctx,
		`SELECT name, COALESCE(glaeubiger_id,''), COALESCE(iban,''), COALESCE(bic,''), COALESCE(kontoinhaber,'') FROM clubs LIMIT 1`).
		Scan(&name, &g, &i, &b, &k)
	c.Name, c.GlaeubigerID, c.IBAN, c.BIC, c.Kontoinhaber = name, g.String, i.String, b.String, k.String
	return c, err
}
