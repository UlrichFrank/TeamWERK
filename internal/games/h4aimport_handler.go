package games

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/h4aimport"
)

// h4aFetcher ist die testbare Teilmenge von *h4aimport.Client. In Tests wird eine
// Fake-Implementierung eingesetzt, damit kein echter Netzzugriff auf H4A passiert.
type h4aFetcher interface {
	Login(ctx context.Context, user, pw string) error
	FetchPeriods(ctx context.Context) ([]h4aimport.Period, error)
	FetchGamesHTML(ctx context.Context, periodID string) (string, error)
	Logout(ctx context.Context) error
}

func defaultH4AFetcher() h4aFetcher { return h4aimport.NewClient() }

// ownClubAliases sind die H4A-Namen der eigenen Mannschaften. Über sie wird pro Spiel
// entschieden, welche Seite (Heim/Gast) „wir" sind. Bewusst eine gepflegte Liste statt
// strings.Contains("Team") — „HandballTeam Heckengäu" wäre sonst ein False-Positive.
// (Siehe design.md §4.)
var ownClubAliases = []string{"Team Stuttgart 2", "Team Stuttgart"}

// ownAlias liefert den eigenen H4A-Vereinsnamen eines Spiels (die gefettete Seite) und ob
// wir Heim sind. Fällt auf RawGame.IsHome zurück, wenn kein Alias matcht.
func ownAlias(g h4aimport.RawGame) (alias string, isHome bool, ok bool) {
	for _, a := range ownClubAliases {
		if strings.EqualFold(strings.TrimSpace(g.Home), a) {
			return a, true, true
		}
		if strings.EqualFold(strings.TrimSpace(g.Guest), a) {
			return a, false, true
		}
	}
	// Kein bekannter Alias — Struktur-Signal (<b>) aus dem Parser nutzen.
	if g.IsHome {
		return strings.TrimSpace(g.Home), true, false
	}
	return strings.TrimSpace(g.Guest), false, false
}

// --- Preview -------------------------------------------------------------------

type h4aFieldChange struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

type h4aPlanGame struct {
	GameNo     string `json:"game_no"`
	Staffel    string `json:"staffel"`
	ClubAlias  string `json:"club_alias"`
	Opponent   string `json:"opponent"`
	Date       string `json:"date"`
	Time       string `json:"time"`
	IsHome     bool   `json:"is_home"`
	EventType  string `json:"event_type"`
	HallNumber string `json:"hall_number"`

	TeamID   *int   `json:"team_id"`
	TeamName string `json:"team_name"`
	// TeamSource unterscheidet die bestätigte Zuordnung ("gelernt") vom aus dem
	// Staffelcode abgeleiteten Vorschlag ("vorschlag"). Das Modal markiert
	// Vorschläge sichtbar — sie sind überprüfenswert, eine gelernte Zuordnung nicht.
	TeamSource string `json:"team_source,omitempty"`
	VenueID    *int   `json:"venue_id"`
	VenueName  string `json:"venue_name"`

	Status         string           `json:"status"` // "new" | "changed" | "unchanged"
	Changes        []h4aFieldChange `json:"changes,omitempty"`
	ExistingGameID *int             `json:"existing_game_id,omitempty"`
	// DuplicateOfGameID zeigt auf ein manuell angelegtes Bestandsspiel (ohne
	// external_id) mit gleichem Datum und Gegner — Kandidat für eine Dublette,
	// die der Importeur im Modal manuell verknüpfen/abwählen muss (design.md §6).
	DuplicateOfGameID *int     `json:"duplicate_of_game_id,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
}

type h4aPreviewResponse struct {
	NeedsPeriod bool               `json:"needs_period,omitempty"`
	Periods     []h4aimport.Period `json:"periods,omitempty"`
	New         []h4aPlanGame      `json:"new,omitempty"`
	Changed     []h4aPlanGame      `json:"changed,omitempty"`
	Unchanged   []h4aPlanGame      `json:"unchanged,omitempty"`
	Warnings    []string           `json:"warnings,omitempty"`
}

// PreviewH4AImport meldet sich mit den eingegebenen Zugangsdaten bei H4A an, ruft den
// Spielplan der gewählten Periode ab und liefert einen Diff gegen die bestehenden Spiele.
// Zugangsdaten leben nur für die Dauer dieses Requests (nie geloggt/persistiert).
// Fehlt period_id, wird nur die Periodenliste zurückgegeben (needs_period).
//
// POST /api/games/import/h4a/preview
func (h *Handler) PreviewH4AImport(w http.ResponseWriter, r *http.Request) {
	if !h.canImportH4A(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		User     string `json:"user"`
		Pw       string `json:"pw"`
		PeriodID string `json:"period_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.User == "" || req.Pw == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	client := h.newH4A()
	if err := client.Login(r.Context(), req.User, req.Pw); err != nil {
		// Antwort generisch — nie das Passwort oder H4A-Interna zurückspiegeln.
		// Der Grund gehört aber ins Server-Log, sonst ist ein fehlgeschlagener
		// Import nicht diagnostizierbar (die drei Fehlercodes sind von außen
		// nicht einmal an der Response-Länge unterscheidbar). Die Fehlerwerte
		// des Clients enthalten nur URL/Netzinfo, keine Zugangsdaten.
		logH4AFailure("login", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "h4a_login_failed"})
		return
	}
	// Fehler beim Abmelden sind für den Import folgenlos: die H4A-Session läuft
	// ohnehin serverseitig ab und der Plan ist zu diesem Zeitpunkt schon gebaut.
	defer func() { _ = client.Logout(context.WithoutCancel(r.Context())) }()

	if req.PeriodID == "" {
		periods, err := client.FetchPeriods(r.Context())
		if err != nil {
			logH4AFailure("fetch-periods", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "h4a_fetch_failed"})
			return
		}
		writeJSON(w, http.StatusOK, h4aPreviewResponse{NeedsPeriod: true, Periods: periods})
		return
	}

	htmlStr, err := client.FetchGamesHTML(r.Context(), req.PeriodID)
	if err != nil {
		logH4AFailure("fetch-games", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "h4a_fetch_failed"})
		return
	}
	games, err := h4aimport.ParseGames(htmlStr)
	if err != nil {
		logH4AFailure("parse", err)
		fmt.Fprintf(h4aLogOut, "h4a-import parse: %d Bytes empfangen, Anfang: %.200q\n", len(htmlStr), htmlStr)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "h4a_parse_failed"})
		return
	}

	resp, err := h.buildH4APlan(r.Context(), games)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// buildH4APlan bildet jede Roh-Zeile auf Mannschaft/Venue ab und ordnet sie via
// external_id (= Spielnummer) in new/changed/unchanged ein.
func (h *Handler) buildH4APlan(ctx context.Context, raw []h4aimport.RawGame) (h4aPreviewResponse, error) {
	var resp h4aPreviewResponse
	// Vorschläge einmal plan-weit ableiten: die Regel „Team Stuttgart 2 = niedrigere
	// Staffel" braucht alle Staffeln einer Altersklasse gleichzeitig.
	suggestions, rejectReasons := h.suggestStaffelTeams(ctx, raw)
	for _, g := range raw {
		alias, isHome, _ := ownAlias(g)
		opponent := strings.TrimSpace(g.Home)
		if isHome {
			opponent = strings.TrimSpace(g.Guest)
		}
		eventType := "auswärts"
		if isHome {
			eventType = "heim"
		}
		pg := h4aPlanGame{
			GameNo:     g.GameNo,
			Staffel:    g.Staffel,
			ClubAlias:  alias,
			Opponent:   opponent,
			Date:       g.Date,
			Time:       g.Time,
			IsHome:     isHome,
			EventType:  eventType,
			HallNumber: g.HallNumber,
		}
		if g.Time == "" {
			pg.Warnings = append(pg.Warnings, "keine Uhrzeit im H4A-Datensatz")
		}

		// Staffel → Mannschaft: erst das gelernte Mapping, sonst der aus dem
		// Staffelcode abgeleitete Vorschlag (beim ersten Import ist die Lerntabelle
		// leer — ohne Vorschlag wäre jede Zeile Handarbeit).
		if tid, tname, ok := h.lookupStaffelTeam(ctx, g.Staffel, alias); ok {
			pg.TeamID, pg.TeamName, pg.TeamSource = &tid, tname, "gelernt"
		} else if c, ok := suggestions[staffelKey{g.Staffel, alias}]; ok {
			tid := c.id
			pg.TeamID, pg.TeamName, pg.TeamSource = &tid, c.name, "vorschlag"
			pg.Warnings = append(pg.Warnings, "Mannschaft vorgeschlagen aus Staffel "+g.Staffel+" — bitte prüfen")
		} else if grund := rejectReasons[staffelKey{g.Staffel, alias}]; grund != "" {
			pg.Warnings = append(pg.Warnings, "Mannschaft nicht zugeordnet: "+grund)
		} else {
			pg.Warnings = append(pg.Warnings, "Mannschaft nicht zugeordnet")
		}

		// Hallennummer → Venue.
		if vid, vname, ok := h.lookupVenueByHall(ctx, g.HallNumber); ok {
			pg.VenueID, pg.VenueName = &vid, vname
		} else if g.HallNumber != "" {
			pg.Warnings = append(pg.Warnings, "Halle "+g.HallNumber+" unbekannt")
		}

		// Diff gegen Bestand (Anker external_id).
		h.classifyH4AGame(ctx, &pg)

		switch pg.Status {
		case "changed":
			resp.Changed = append(resp.Changed, pg)
		case "unchanged":
			resp.Unchanged = append(resp.Unchanged, pg)
		default:
			resp.New = append(resp.New, pg)
		}
	}
	return resp, nil
}

// classifyH4AGame setzt pg.Status (new/changed/unchanged) und pg.Changes anhand des
// Bestands mit gleicher external_id.
func (h *Handler) classifyH4AGame(ctx context.Context, pg *h4aPlanGame) {
	var (
		id       int
		date     string
		tim      string
		opponent string
		isHome   int
		venueID  sql.NullInt64
	)
	err := h.db.QueryRowContext(ctx,
		`SELECT id, date, time, opponent, is_home, venue_id FROM games WHERE external_id = ?`,
		pg.GameNo).Scan(&id, &date, &tim, &opponent, &isHome, &venueID)
	if err == sql.ErrNoRows {
		pg.Status = "new"
		h.markPossibleDuplicate(ctx, pg)
		return
	}
	if err != nil {
		pg.Status = "new"
		return
	}
	pg.ExistingGameID = &id
	date = date[:min(10, len(date))]
	var changes []h4aFieldChange
	if date != pg.Date {
		changes = append(changes, h4aFieldChange{"date", date, pg.Date})
	}
	if tim != pg.Time && pg.Time != "" {
		changes = append(changes, h4aFieldChange{"time", tim, pg.Time})
	}
	if strings.TrimSpace(opponent) != pg.Opponent {
		changes = append(changes, h4aFieldChange{"opponent", opponent, pg.Opponent})
	}
	if (isHome == 1) != pg.IsHome {
		changes = append(changes, h4aFieldChange{"is_home", fmt.Sprint(isHome == 1), fmt.Sprint(pg.IsHome)})
	}
	newVenue := int64(0)
	if pg.VenueID != nil {
		newVenue = int64(*pg.VenueID)
	}
	if venueID.Int64 != newVenue && pg.VenueID != nil {
		changes = append(changes, h4aFieldChange{"venue", fmt.Sprint(venueID.Int64), fmt.Sprint(newVenue)})
	}
	if len(changes) == 0 {
		pg.Status = "unchanged"
	} else {
		pg.Status = "changed"
		pg.Changes = changes
	}
}

// markPossibleDuplicate sucht zu einer als „neu" eingestuften Zeile ein manuell
// angelegtes Bestandsspiel (external_id IS NULL) mit gleichem Datum und Gegner.
// Vor dem ersten H4A-Import tragen alle Spiele keine external_id — ohne diesen
// Hinweis würde der Import Dubletten neben die Handarbeit legen (design.md §6).
// Ist die Mannschaft bekannt, muss sie zusätzlich übereinstimmen.
func (h *Handler) markPossibleDuplicate(ctx context.Context, pg *h4aPlanGame) {
	query := `SELECT g.id FROM games g
	           WHERE g.external_id IS NULL AND substr(g.date,1,10) = ?
	             AND lower(trim(g.opponent)) = lower(?)`
	args := []any{pg.Date, pg.Opponent}
	if pg.TeamID != nil {
		query += ` AND EXISTS (SELECT 1 FROM game_teams gt WHERE gt.game_id = g.id AND gt.team_id = ?)`
		args = append(args, *pg.TeamID)
	}
	query += ` LIMIT 1`

	var dupID int
	if err := h.db.QueryRowContext(ctx, query, args...).Scan(&dupID); err != nil {
		return
	}
	pg.DuplicateOfGameID = &dupID
	pg.Warnings = append(pg.Warnings,
		"mögliche Dublette zu einem bereits angelegten Spiel am selben Datum")
}

func (h *Handler) lookupStaffelTeam(ctx context.Context, staffel, alias string) (int, string, bool) {
	var teamID int
	var name string
	err := h.db.QueryRowContext(ctx,
		`SELECT m.team_id, t.name
		   FROM h4a_staffel_team_map m JOIN teams t ON t.id = m.team_id
		  WHERE m.staffel = ? AND m.club_alias = ?`, staffel, alias).Scan(&teamID, &name)
	if err != nil {
		return 0, "", false
	}
	return teamID, name, true
}

func (h *Handler) lookupVenueByHall(ctx context.Context, hallNumber string) (int, string, bool) {
	if strings.TrimSpace(hallNumber) == "" {
		return 0, "", false
	}
	var id int
	var name string
	err := h.db.QueryRowContext(ctx,
		`SELECT id, name FROM venues WHERE hall_number = ?`, hallNumber).Scan(&id, &name)
	if err != nil {
		return 0, "", false
	}
	return id, name, true
}

// --- Apply ---------------------------------------------------------------------

type h4aApplyDecision struct {
	GameNo     string `json:"game_no"`
	Staffel    string `json:"staffel"`
	ClubAlias  string `json:"club_alias"`
	TeamID     int    `json:"team_id"`
	VenueID    *int   `json:"venue_id"`
	Opponent   string `json:"opponent"`
	Date       string `json:"date"`
	Time       string `json:"time"`
	EventType  string `json:"event_type"`
	TemplateID *int   `json:"template_id"`
}

type h4aApplyResponse struct {
	Imported     int          `json:"imported"`
	Updated      int          `json:"updated"`
	Skipped      int          `json:"skipped"`
	RegenSummary RegenSummary `json:"regen_summary"`
}

// ApplyH4AImport schreibt die bestätigten Spiele. Kein H4A-Zugriff mehr, keine
// Zugangsdaten nötig — es wird ausschließlich der übergebene Plan gegen die DB
// re-validiert und geschrieben. Genau ein Broadcast und ein runAutoRegen-Lauf.
//
// POST /api/games/import/h4a/apply
func (h *Handler) ApplyH4AImport(w http.ResponseWriter, r *http.Request) {
	if !h.canImportH4A(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Decisions []h4aApplyDecision `json:"decisions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var seasonID int
	if err := h.db.QueryRowContext(r.Context(),
		`SELECT id FROM seasons WHERE is_active = 1 LIMIT 1`).Scan(&seasonID); err != nil {
		http.Error(w, "keine aktive Saison", http.StatusBadRequest)
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var result h4aApplyResponse
	dateSet := map[string]bool{}

	for _, d := range req.Decisions {
		if d.GameNo == "" || d.TeamID == 0 || d.Date == "" || d.EventType == "" {
			result.Skipped++
			continue
		}
		// Re-Validierung gegen die DB (Client-Plan wird nicht blind vertraut).
		if !existsID(r.Context(), tx, "teams", d.TeamID) {
			result.Skipped++
			continue
		}
		venueVal := sqlNullVenue(r.Context(), tx, d.VenueID)
		templateVal, ok := validTemplate(r.Context(), tx, d.TemplateID)
		if !ok {
			result.Skipped++
			continue
		}
		isHome := d.EventType == "heim"
		timeVal := d.Time
		if timeVal == "" {
			timeVal = "00:00"
		}

		var existingID int
		errExist := tx.QueryRowContext(r.Context(),
			`SELECT id FROM games WHERE external_id = ?`, d.GameNo).Scan(&existingID)

		if errExist == sql.ErrNoRows {
			res, e := tx.ExecContext(r.Context(),
				`INSERT INTO games (season_id, opponent, date, time, is_home, event_type, template_id, venue_id, source, external_id)
				 VALUES (?,?,?,?,?,?,?,?, 'h4a', ?)`,
				seasonID, d.Opponent, d.Date, timeVal, isHome, d.EventType, templateVal, venueVal, d.GameNo)
			if e != nil {
				result.Skipped++
				continue
			}
			gid, _ := res.LastInsertId()
			tx.ExecContext(r.Context(),
				`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, gid, d.TeamID)
			result.Imported++
		} else if errExist == nil {
			if _, e := tx.ExecContext(r.Context(),
				`UPDATE games SET opponent=?, date=?, time=?, is_home=?, event_type=?, template_id=?, venue_id=? WHERE id=?`,
				d.Opponent, d.Date, timeVal, isHome, d.EventType, templateVal, venueVal, existingID); e != nil {
				result.Skipped++
				continue
			}
			// game_teams synchronisieren (nur diese eine Mannschaft).
			tx.ExecContext(r.Context(), `DELETE FROM game_teams WHERE game_id=?`, existingID)
			tx.ExecContext(r.Context(),
				`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, existingID, d.TeamID)
			result.Updated++
		} else {
			result.Skipped++
			continue
		}

		// Gelerntes Mapping festhalten (idempotent).
		if d.Staffel != "" && d.ClubAlias != "" {
			tx.ExecContext(r.Context(),
				`INSERT INTO h4a_staffel_team_map (staffel, club_alias, team_id) VALUES (?,?,?)
				 ON CONFLICT(staffel, club_alias) DO UPDATE SET team_id = excluded.team_id`,
				d.Staffel, d.ClubAlias, d.TeamID)
		}
		dateSet[d.Date] = true
	}

	// EIN Regen-Lauf über die Vereinigungsmenge aller betroffenen Datumsfenster.
	dates := make([]string, 0, len(dateSet)*3)
	for d := range dateSet {
		dates = append(dates, dateWindow(d)...)
	}
	sort.Strings(dates)
	summary, err := h.runAutoRegen(r.Context(), tx, dates, seasonID, nil)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	result.RegenSummary = summary

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// EIN Broadcast; bewusst KEINE Spieler-Pushes beim Massenimport (design.md §7).
	if h.hub != nil {
		h.hub.Broadcast("games")
	}
	h.dispatchRegenNotifications(summary)
	writeJSON(w, http.StatusOK, result)
}

// h4aLogOut ist die Senke der Import-Diagnose. Als Variable, damit der Test
// mitlesen kann, dass hier niemals Zugangsdaten landen (die Zusicherung ist
// wertlos, wenn sie nicht am tatsächlichen Ziel geprüft wird).
var h4aLogOut io.Writer = os.Stderr

// logH4AFailure schreibt den Grund eines fehlgeschlagenen H4A-Schritts ins
// Server-Log. Bewusst nur der Fehlerwert des Clients (URL/HTTP-Status/Netzinfo)
// — Benutzername und Passwort werden nie übergeben und tauchen dort nicht auf.
func logH4AFailure(step string, err error) {
	fmt.Fprintf(h4aLogOut, "h4a-import %s fehlgeschlagen: %v\n", step, err)
}

// canImportH4A: Vorstand-Funktion oder System-Admin.
func (h *Handler) canImportH4A(r *http.Request) bool {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		return false
	}
	return claims.Role == "admin" || claims.HasFunction("vorstand")
}

func existsID(ctx context.Context, tx *sql.Tx, table string, id int) bool {
	var one int
	// table stammt aus konstanten Literalen im Aufrufer, nicht aus User-Input.
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM "+table+" WHERE id = ?", id).Scan(&one)
	return err == nil
}

func sqlNullVenue(ctx context.Context, tx *sql.Tx, venueID *int) interface{} {
	if venueID == nil {
		return nil
	}
	if !existsID(ctx, tx, "venues", *venueID) {
		return nil
	}
	return *venueID
}

// validTemplate prüft ein optionales Template. nil → (nil,true). Ungültige ID → (nil,false).
func validTemplate(ctx context.Context, tx *sql.Tx, templateID *int) (interface{}, bool) {
	if templateID == nil {
		return nil, true
	}
	if !existsID(ctx, tx, "game_templates", *templateID) {
		return nil, false
	}
	return *templateID, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
