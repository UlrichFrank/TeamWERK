package matchreports

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/db"
)

// ErrNoActiveSeason signalisiert, dass keine `seasons.is_active=1`-Zeile
// existiert. Der Publisher braucht die aktive Saison für den Slug-Pfad; ohne
// sie ist der Publish nicht durchführbar. Der Handler mappt das auf HTTP 500
// `no_active_season` — bewusst VOR dem State-Wechsel `→ publishing`, damit der
// Bericht in `pending_review` bleibt (siehe design.md dieses Change).
var ErrNoActiveSeason = errors.New("no_active_season")

type publishResp struct {
	PageUID int    `json:"pageUid"`
	URL     string `json:"url"`
}

// Publish transitioniert einen eingereichten Bericht (State pending_review
// oder publish_failed-Retry) über die State-Machine → publishing → published
// | publish_failed und ruft den TYPO3-Publisher auf.
//
//	POST /api/match-reports/{id}/publish
//
// Guard: Vereinsfunktion 'medien' oder 'vorstand' oder Admin (siehe
// spielbericht-medien-gate spec.md, D-1). Der Autor **ohne** Freigeber-Fkt darf
// NICHT publishen — auch nicht seinen eigenen Bericht (F2: Vier-Augen weich,
// aber die Freigabe-Autorität liegt strukturell beim Freigeber).
//
// Der atomare State-Wechsel `pending_review|publish_failed → publishing`
// verhindert Doppel-Publish bei paralleler Ausführung (zweiter Aufruf 409).
// Nach erfolgreichem 2xx vom Publisher:
//   - state=published, published_url + typo3_page_uid + published_at + reviewer_user_id gesetzt
//   - Duty-Slot auf fulfilled
//   - Bilder-Dateien + DB-Zeilen gelöscht
func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !isReviewer(claims) {
		writeErr(w, http.StatusForbidden, "role_required")
		return
	}

	id, ok := parsePathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_id")
		return
	}

	// State-Check vor Übergang. Autor-Check entfällt — Guard läuft über die
	// Freigeber-Rolle, nicht über Ownership.
	var state string
	err := h.db.QueryRow(
		`SELECT state FROM match_reports WHERE id=?`, id,
	).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logErr("matchreports.Publish select", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	switch state {
	case StateDraft:
		writeErr(w, http.StatusConflict, "not_submitted")
		return
	case StatePublished:
		writeErr(w, http.StatusConflict, "already_published")
		return
	case StatePublishing:
		writeErr(w, http.StatusConflict, "in_progress")
		return
	}

	// Vorab-Check: aktive Saison muss existieren. Bewusst VOR dem
	// State-Wechsel `→ publishing`, damit der Bericht in seinem aktuellen
	// State (`pending_review` / `publish_failed`) bleibt und der Freigeber
	// nach dem Anlegen der Saison einfach nochmal drücken kann. Danach
	// dieselben Fehler nochmal in `assemblePublishRequest` (Kanonische
	// Fehlerquelle), aber dieser hier verhindert das eklige `publish_failed`
	// wegen fehlender Saison.
	var seasonExists int
	if err := h.db.QueryRow(`SELECT 1 FROM seasons WHERE is_active = 1 LIMIT 1`).Scan(&seasonExists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeErr(w, http.StatusInternalServerError, "no_active_season")
			return
		}
		logErr("matchreports.Publish season precheck", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	// Atomarer Übergang zu 'publishing'. Ausgangszustände: pending_review
	// (Regelfall) oder publish_failed (Retry). Bei Race verliert der zweite
	// Requester (0 Zeilen betroffen).
	res, err := h.db.Exec(
		`UPDATE match_reports SET state=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=? AND state IN (?, ?)`,
		StatePublishing, id, StatePendingReview, StatePublishFailed,
	)
	if err != nil {
		logErr("matchreports.Publish state transition", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeErr(w, http.StatusConflict, "in_progress")
		return
	}
	h.broadcast()

	// Payload zusammensetzen und feuern.
	req, err := h.assemblePublishRequest(id)
	if err != nil {
		h.finalizeFailed(id, fmt.Sprintf("assemble: %s", err.Error()))
		logErr("matchreports.Publish assemble", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "assemble_failed", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	result, err := h.publisher.Publish(ctx, req)
	if err != nil {
		msg := err.Error()
		h.finalizeFailed(id, msg)
		logErr("matchreports.Publish publisher", err, "id", id)

		if errors.Is(err, ErrPublisherNotConfigured) {
			writeErr(w, http.StatusInternalServerError, "publisher_not_configured")
			return
		}
		// PublisherError vom TYPO3-Endpoint → 502 (Bad Gateway).
		var pe *PublisherError
		if errors.As(err, &pe) {
			writeErr(w, http.StatusBadGateway, "publisher_failed", pe.Error())
			return
		}
		writeErr(w, http.StatusBadGateway, "publisher_failed", msg)
		return
	}

	// Erfolgs-Finalisierung. Reviewer_user_id wird auf den aktuellen Requester
	// gesetzt (letzter Publisher gewinnt — siehe design.md D-6).
	if err := h.finalizePublished(id, claims.UserID, result); err != nil {
		// Report ist auf TYPO3 live, aber Nachbereitung schlug fehl.
		// State geht trotzdem auf 'published' — der Bericht steht ja online.
		logErr("matchreports.Publish finalize", err, "id", id)
	}

	h.broadcast()
	writeJSON(w, http.StatusOK, publishResp{PageUID: result.PageUID, URL: result.URL})
}

// assemblePublishRequest lädt alle Daten aus der DB und baut das PublishRequest.
//
// Fehler-Semantik: `ErrNoActiveSeason` wird durchgereicht, wenn keine
// aktive Saison existiert — der Handler muss das VOR dem State-Wechsel
// `→ publishing` abfangen (siehe `Publish`).
func (h *Handler) assemblePublishRequest(reportID int) (*PublishRequest, error) {
	// Bericht + verlinktes Spiel + Team-Kürzel (aus Kader-Konvention).
	var (
		gameID        int
		title         string
		abstract      string
		bodyMD        string
		tournamentInt int
		homeGoals     sql.NullInt64
		awayGoals     sql.NullInt64
		homeGoalsHT   sql.NullInt64
		awayGoalsHT   sql.NullInt64
		opponent      string
		matchDate     string
		teamShortName sql.NullString
		teamName      sql.NullString
		clubName      string
	)

	// `db.TeamDisplayShort("t")` referenziert `t.id` und die kader-Tabelle intern.
	// `ORDER BY t.id LIMIT 1` sichert Determinismus, wenn mehrere game_teams
	// existieren (bewusst gleiche Konvention wie andere Publisher-Selects).
	query := `SELECT r.game_id, r.title, r.abstract, r.body_md, r.tournament,
	        r.home_goals, r.away_goals, r.home_goals_ht, r.away_goals_ht,
	        g.opponent, g.date,
	        ` + db.TeamDisplayShort("t") + ` AS team_short_name,
	        t.name,
	        COALESCE((SELECT name FROM clubs LIMIT 1), 'Team Stuttgart')
	 FROM match_reports r
	 JOIN games g ON g.id = r.game_id
	 LEFT JOIN game_teams gt ON gt.game_id = g.id
	 LEFT JOIN teams t ON t.id = gt.team_id
	 WHERE r.id = ?
	 ORDER BY t.id
	 LIMIT 1`

	err := h.db.QueryRow(query, reportID).Scan(
		&gameID, &title, &abstract, &bodyMD, &tournamentInt,
		&homeGoals, &awayGoals, &homeGoalsHT, &awayGoalsHT,
		&opponent, &matchDate,
		&teamShortName, &teamName,
		&clubName,
	)
	if err != nil {
		return nil, fmt.Errorf("load report meta: %w", err)
	}

	matchDateUnix, err := parseMatchDate(matchDate)
	if err != nil {
		return nil, err
	}

	// Aktive Saison → Slug-Segment "YYYY-YYYY".
	var seasonName string
	err = h.db.QueryRow(`SELECT name FROM seasons WHERE is_active = 1 LIMIT 1`).Scan(&seasonName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoActiveSeason
	}
	if err != nil {
		return nil, fmt.Errorf("load active season: %w", err)
	}
	seasonSegment, err := ParseSeasonName(seasonName)
	if err != nil {
		return nil, fmt.Errorf("parse active season %q: %w", seasonName, err)
	}

	// Fallback: falls `title` leer ist (Altbestand aus vor-Migration-025), aus
	// Datum + Gegner ableiten. Neu angelegte Berichte haben `title` gesetzt.
	if title == "" {
		title = BuildTitle(matchDateUnix, opponent)
	}
	// Nur der title-slug — die Extension setzt den Pfad-Präfix
	// /spielberichte/{season}/ selbst zusammen.
	slug := TitleSlug(title)
	matchTeams := fmt.Sprintf("%s – %s", nullStringOr(teamName, clubName), opponent)

	// Body sanitisieren (Markdown → allowlist-HTML).
	bodyHTML, err := SanitizeBody(bodyMD)
	if err != nil {
		return nil, fmt.Errorf("sanitize body: %w", err)
	}

	// Bilder aus DB laden (inkl. Datei-Pfade + Captions).
	images, err := h.loadPublishImages(reportID)
	if err != nil {
		return nil, err
	}

	imageMetas := make([]PublishImageMeta, len(images))
	for i, img := range images {
		imageMetas[i] = PublishImageMeta{Caption: img.Caption}
	}

	homeInt := nullInt64Ptr(homeGoals)
	awayInt := nullInt64Ptr(awayGoals)
	htHomeInt := nullInt64Ptr(homeGoalsHT)
	htAwayInt := nullInt64Ptr(awayGoalsHT)

	return &PublishRequest{
		Meta: PublishMeta{
			Title:            title,
			Slug:             slug,
			Season:           seasonSegment,
			Abstract:         abstract,
			MatchDate:        matchDateUnix,
			MatchScore:       FormatMatchScore(homeInt, awayInt, htHomeInt, htAwayInt, tournamentInt != 0),
			MatchTeams:       matchTeams,
			Tournament:       tournamentInt != 0,
			TeamCategoryName: nullString(teamShortName),
			BodyHTML:         bodyHTML,
			ExternalReportID: fmt.Sprintf("teamwerk-report-%d", reportID),
			Images:           imageMetas,
		},
		Images: images,
	}, nil
}

// loadPublishImages holt Bilder aus der DB, konvertiert Storage-Pfade in
// PublishImage-Structs (mit Caption).
func (h *Handler) loadPublishImages(reportID int) ([]PublishImage, error) {
	rows, err := h.db.Query(
		`SELECT storage_path, caption FROM match_report_images
		 WHERE report_id=? ORDER BY position ASC`, reportID,
	)
	if err != nil {
		return nil, fmt.Errorf("load images: %w", err)
	}
	defer rows.Close()

	var out []PublishImage
	for rows.Next() {
		var img PublishImage
		if err := rows.Scan(&img.Path, &img.Caption); err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

// finalizePublished markiert den Bericht als veröffentlicht, hält den
// freigebenden User als reviewer_user_id fest, quittiert den zugehörigen
// Duty-Slot und räumt die Bilder auf.
func (h *Handler) finalizePublished(reportID, reviewerUserID int, result *PublishResult) error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE match_reports
		 SET state=?, published_url=?, typo3_page_uid=?, published_at=CURRENT_TIMESTAMP,
		     reviewer_user_id=?, error_message=NULL, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		StatePublished, result.URL, result.PageUID, reviewerUserID, reportID,
	); err != nil {
		return err
	}

	// Duty-Slot fulfillen (Best-Effort — falls kein Slot: kein Fehler).
	if _, err := tx.Exec(
		`UPDATE duty_assignments
		 SET status='fulfilled', fulfilled_at=CURRENT_TIMESTAMP
		 WHERE duty_slot_id = (SELECT duty_slot_id FROM match_reports WHERE id=?)
		   AND user_id       = (SELECT author_user_id FROM match_reports WHERE id=?)
		   AND status = 'assigned'`,
		reportID, reportID,
	); err != nil {
		return err
	}

	// Bilder-DB-Zeilen löschen.
	if _, err := tx.Exec(
		`DELETE FROM match_report_images WHERE report_id=?`, reportID,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Datei-Cleanup außerhalb der Transaktion — Zombie-Files sind hässlich,
	// aber kein Konsistenz-Problem für die DB.
	h.removeAllImageFiles(reportID)
	return nil
}

// finalizeFailed schreibt state=publish_failed + error_message.
// Bilder bleiben liegen für den manuellen Retry.
func (h *Handler) finalizeFailed(reportID int, msg string) {
	if _, err := h.db.Exec(
		`UPDATE match_reports
		 SET state=?, error_message=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		StatePublishFailed, msg, reportID,
	); err != nil {
		logErr("matchreports.finalizeFailed", err, "id", reportID)
	}
}

// parseMatchDate wandelt games.date (String "YYYY-MM-DD" oder ISO-Timestamp)
// in einen Unix-Timestamp (Mittags UTC — vermeidet Zeitzonen-Kippen).
func parseMatchDate(raw string) (int64, error) {
	if len(raw) >= 10 {
		raw = raw[:10]
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return 0, fmt.Errorf("parse match_date %q: %w", raw, err)
	}
	// 12:00 UTC — SQLite DATE-Felder haben keine Zeit, wir liefern Mittag,
	// damit der TYPO3-Endpoint einen sinnvollen match_date-Wert bekommt.
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.UTC).Unix(), nil
}

func nullString(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

func nullStringOr(s sql.NullString, fallback string) string {
	if s.Valid && s.String != "" {
		return s.String
	}
	return fallback
}

// parsePathIDStr existiert nur zur Rückwärts-Kompat mit älteren Test-Helpern.
// Kann sofort raus, sobald keine Tests mehr davon abhängen.
var _ = strconv.Atoi
