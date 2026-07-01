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
)

// Exclusion-/Warnungs-Codes
const (
	exclStatusInaktiv = "status_inaktiv"
	exclBeitragsfrei  = "beitragsfrei"
	exclKeinMandat    = "kein_sepa_mandat"
	exclIBANFehlt     = "iban_fehlt"
	exclKeineMitglNr  = "mitgliedsnummer_fehlt"
	exclAdresse       = "adresse_unvollstaendig"
	exclKeinSatz      = "kein_beitragssatz"
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
	Half           bool     `json:"half"`
	HalfReason     string   `json:"half_reason,omitempty"` // erstjahr | eintritt | austritt
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
	season, err := h.loadSeason(ctx, saisonID)
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
	res := &previewResult{SaisonID: saisonID, SaisonLabel: season.Label, SaisonKurz: season.Label, Faelligkeit: season.Stichtag}
	for _, m := range members {
		// Früher (oder ohne Austrittsdatum) ausgetretene Mitglieder gar nicht
		// anzeigen — wie honorar/anwaerter. Nur unterjährige Austritte
		// (exit_date im Saisonfenster) werden einbezogen und halbiert.
		if m.Status == "ausgetreten" && !inWindow(m.ExitDate, season.Start, season.End) {
			continue
		}
		res.Items = append(res.Items, computeItem(m, saetze, season))
	}
	return res, nil
}

func computeItem(m MemberRow, saetze map[string][]Satz, season SeasonInfo) PreviewItem {
	it := PreviewItem{
		MemberID:   m.ID,
		Name:       m.FirstName + " " + m.LastName,
		Status:     m.Status,
		Warnings:   []string{},
		Exclusions: []string{},
		row:        m,
	}
	gruppe := BeitragsGruppe(m.Status)
	// Unterjähriger Austritt: ausgetreten + exit_date im Saisonfenster → wie ein
	// aktives Mitglied behandeln (Kategorie aus home_club_id), Beitrag halbiert.
	if gruppe == "" && m.Status == "ausgetreten" && inWindow(m.ExitDate, season.Start, season.End) {
		gruppe = "aktiv"
	}
	if gruppe == "" {
		it.Exclusions = append(it.Exclusions, exclStatusInaktiv)
	}
	if m.Beitragsfrei {
		it.Exclusions = append(it.Exclusions, exclBeitragsfrei)
	}

	// Beitrag bereits hier berechnen — auch für Mitglieder, die später wegen
	// fehlender SEPA-Daten ausgeschlossen werden. So bleibt im Frontend
	// sichtbar, welcher Betrag NICHT abgebucht werden kann.
	if gruppe != "" && !m.Beitragsfrei {
		var kategorie string
		if gruppe == "passiv" {
			kategorie = "passiv"
		} else {
			kategorie = AktivKategorie(m.HasHomeClub)
		}
		betrag, err := LookupBetragCent(saetze, kategorie, season.Stichtag)
		if err != nil {
			it.Exclusions = append(it.Exclusions, exclKeinSatz)
		} else {
			it.Kategorie = kategorie
			it.KategorieLabel = kategorieLabel[kategorie]
			// Exakte Halbierung (Integer-Division) bei unterjährigem Ein-/Austritt
			// oder im ersten Abrechnungsjahr. Ermäßigungen stapeln nicht.
			if half, reason := halfFee(m, season); half {
				betrag /= 2
				it.Half = true
				it.HalfReason = reason
			}
			it.BetragCent = betrag
		}
	}

	if m.MemberNumber == "" {
		it.Exclusions = append(it.Exclusions, exclKeineMitglNr)
	}
	if !m.SepaMandat || m.SepaMandatPath == "" {
		it.Exclusions = append(it.Exclusions, exclKeinMandat)
	}
	// Server kennt nur, OB Bankdaten vorliegen (Envelope vorhanden). Die IBAN-Gültigkeit
	// (exclIBANUngueltig) prüft der Client nach dem Entschlüsseln.
	if !m.HasBank {
		it.Exclusions = append(it.Exclusions, exclIBANFehlt)
	}
	if m.Street == "" || m.Zip == "" || m.City == "" {
		it.Exclusions = append(it.Exclusions, exclAdresse)
	}

	it.Included = len(it.Exclusions) == 0
	return it
}

// loadSeason liefert das Abrechnungsjahr: Label, Saisonfenster (start/end),
// den Stichtag 01.07. des Startjahres (Fälligkeit + Satz-Stichtag) und das
// is_inaugural-Flag (erstes Abrechnungsjahr → alle zahlen halb).
func (h *Handler) loadSeason(ctx context.Context, id int) (SeasonInfo, error) {
	var name, startDate, endDate string
	var inaugural int
	err := h.db.QueryRowContext(ctx,
		`SELECT name, start_date, end_date, COALESCE(is_inaugural,0) FROM seasons WHERE id=?`, id).
		Scan(&name, &startDate, &endDate, &inaugural)
	if err != nil {
		return SeasonInfo{}, err
	}
	start, perr := time.Parse("2006-01-02", startDate[:min(10, len(startDate))])
	if perr != nil {
		return SeasonInfo{}, perr
	}
	end, perr := time.Parse("2006-01-02", endDate[:min(10, len(endDate))])
	if perr != nil {
		return SeasonInfo{}, perr
	}
	return SeasonInfo{
		Label:     name,
		Start:     start,
		End:       end,
		Stichtag:  time.Date(start.Year(), time.July, 1, 0, 0, 0, 0, time.UTC),
		Inaugural: inaugural != 0,
	}, nil
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
	var inc, exc, warn, sepaTotal, exclTotal int
	for _, it := range pr.Items {
		if it.Included {
			inc++
			sepaTotal += it.BetragCent
			if len(it.Warnings) > 0 {
				warn++
			}
		} else {
			exc++
			exclTotal += it.BetragCent
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"saison_id":    pr.SaisonID,
		"saison_label": pr.SaisonLabel,
		"faelligkeit":  pr.Faelligkeit.Format("2006-01-02"),
		"items":        pr.Items,
		"summary": map[string]any{
			"included_count":   inc,
			"excluded_count":   exc,
			"warned_count":     warn,
			"total_cent":       sepaTotal,
			"excluded_cent":    exclTotal,
			"gesamtsumme_cent": sepaTotal + exclTotal,
		},
	})
}

// POST /api/fee-run/export-data  {saison_id, member_ids, faelligkeit?}
//
// Liefert die für die clientseitige pain.008-Erzeugung nötigen Daten: NUR Ciphertext +
// Wraps (Mitglieds-Bankdaten + Vereins-SEPA) sowie nicht-geheime Felder. Der Server sieht
// keine Klartext-IBAN; das XML wird im Browser des Kassierers gebaut (Zero-Knowledge).
//
// `faelligkeit` (YYYY-MM-DD) ist optional und überschreibt für den SEPA-Einzugstermin
// (ReqdColltnDt) den Default 01.07. der Saison. Muss heute oder in der Zukunft liegen —
// SEPA-XSD lehnt Vergangenheits-Daten ab. Der Beitragssatz-Stichtag bleibt unverändert
// bei 01.07., damit ein anderes Fälligkeitsdatum keinen anderen Satz auslöst.
func (h *Handler) ExportData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SaisonID    int    `json:"saison_id"`
		MemberIDs   []int  `json:"member_ids"`
		Faelligkeit string `json:"faelligkeit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	club, err := h.loadClubSepa(r.Context())
	if err != nil || club.Ciphertext == "" {
		http.Error(w, "Vereins-SEPA-Stammdaten nicht eingerichtet", http.StatusBadRequest)
		return
	}
	pr, err := h.buildPreview(r.Context(), req.SaisonID)
	if err != nil {
		http.Error(w, "Saison nicht gefunden", http.StatusNotFound)
		return
	}
	faelligkeitStr := pr.Faelligkeit.Format("2006-01-02")
	if req.Faelligkeit != "" {
		fd, ferr := time.Parse("2006-01-02", req.Faelligkeit)
		if ferr != nil {
			http.Error(w, "faelligkeit: Format YYYY-MM-DD erwartet", http.StatusBadRequest)
			return
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if fd.Before(today) {
			http.Error(w, "faelligkeit liegt in der Vergangenheit — SEPA lehnt das ab", http.StatusBadRequest)
			return
		}
		faelligkeitStr = req.Faelligkeit
	}
	byID := map[int]PreviewItem{}
	for _, it := range pr.Items {
		byID[it.MemberID] = it
	}
	type exportDataItem struct {
		MemberID       int    `json:"member_id"`
		Name           string `json:"name"`
		MemberNumber   string `json:"member_number"`
		BetragCent     int    `json:"betrag_cent"`
		Street         string `json:"street"`
		Zip            string `json:"zip"`
		City           string `json:"city"`
		MandatDatum    string `json:"mandat_datum"`
		BankCiphertext string `json:"bank_ciphertext"`
		BankDekEnc     string `json:"bank_dek_enc"`
	}
	items := make([]exportDataItem, 0, len(req.MemberIDs))
	for _, id := range req.MemberIDs {
		it, ok := byID[id]
		if !ok || !it.Included {
			http.Error(w, fmt.Sprintf("Mitglied %d ist ausgeschlossen oder unbekannt", id), http.StatusBadRequest)
			return
		}
		items = append(items, exportDataItem{
			MemberID: it.MemberID, Name: it.Name, MemberNumber: it.row.MemberNumber,
			BetragCent: it.BetragCent, Street: it.row.Street, Zip: it.row.Zip, City: it.row.City,
			MandatDatum:    it.row.SepaMandatDate,
			BankCiphertext: it.row.BankCiphertext, BankDekEnc: it.row.BankDekEnc,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"saison_kurz": pr.SaisonKurz,
		"faelligkeit": faelligkeitStr,
		"club_name":   club.Name,
		"club_sepa":   map[string]string{"ciphertext": club.Ciphertext, "dek_enc": club.DekEnc},
		"items":       items,
	})
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
	season, err := h.loadSeason(r.Context(), req.SaisonID)
	if err != nil {
		http.Error(w, "Saison nicht gefunden", http.StatusNotFound)
		return
	}
	label := season.Label
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
	season, err := h.loadSeason(r.Context(), saisonID)
	if err != nil {
		http.Error(w, "Saison nicht gefunden", http.StatusNotFound)
		return
	}
	data, err := ReadProtokoll(h.dir, season.Label)
	if err != nil {
		http.Error(w, "Protokoll konnte nicht gelesen werden", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// clubSepa hält Vereinsname (Klartext) + den Zero-Knowledge-Envelope der SEPA-Stammdaten
// (glaeubiger_id/iban/bic/kontoinhaber). Der Server entschlüsselt nicht.
type clubSepa struct {
	Name       string
	Ciphertext string
	DekEnc     string
}

func (h *Handler) loadClubSepa(ctx context.Context) (clubSepa, error) {
	var c clubSepa
	var ct, dek sql.NullString
	err := h.db.QueryRowContext(ctx,
		`SELECT name, COALESCE(sepa_ciphertext,''), COALESCE(sepa_dek_enc,'') FROM clubs LIMIT 1`).
		Scan(&c.Name, &ct, &dek)
	c.Ciphertext = ct.String
	c.DekEnc = dek.String
	return c, err
}
