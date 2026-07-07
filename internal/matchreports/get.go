package matchreports

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Report ist die JSON-Repräsentation eines Berichts für GET-Responses.
type Report struct {
	ID                  int             `json:"id"`
	GameID              int             `json:"game_id"`
	DutySlotID          *int            `json:"duty_slot_id"`
	AuthorUserID        int             `json:"author_user_id"`
	State               string          `json:"state"`
	Title               string          `json:"title"`
	HomeGoals           *int            `json:"home_goals"`
	AwayGoals           *int            `json:"away_goals"`
	HomeGoalsHT         *int            `json:"home_goals_ht"`
	AwayGoalsHT         *int            `json:"away_goals_ht"`
	Tournament          bool            `json:"tournament"`
	Abstract            string          `json:"abstract"`
	BodyMarkdown        string          `json:"body_md"`
	PublishedURL        *string         `json:"published_url"`
	Typo3PageUID        *int            `json:"typo3_page_uid"`
	ErrorMessage        *string         `json:"error_message"`
	Images              []Image         `json:"images"`
	PhotoConsentMissing []ConsentMember `json:"photo_consent_missing"`
}

// Image ist ein Bilder-Eintrag am Bericht.
type Image struct {
	ID       int    `json:"id"`
	Position int    `json:"position"`
	Caption  string `json:"caption"`
	URL      string `json:"url"`
}

// Get liefert einen Bericht. Sichtbarkeitsregeln:
//
//   - state=published: alle Authenticated (Referenz auf öffentliche Homepage).
//
//   - state=draft: nur Autor + Admin.
//
//   - state=pending_review: Autor (read-only) + Freigeber (Prüfung) + Admin.
//
//   - state=publishing / publish_failed: Autor + Freigeber + Admin
//     (Freigeber muss Fehler sehen und retryen können).
//
//     GET /api/match-reports/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, ok := parsePathID(r, "id")
	if !ok {
		writeErr(w, http.StatusBadRequest, "bad_id")
		return
	}

	rep, err := h.loadReport(id)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		logErr("matchreports.Get load", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}

	if !canReadReport(claims, rep) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}

	// Bilder anhängen.
	imgs, err := h.listImages(id)
	if err != nil {
		logErr("matchreports.Get list images", err, "id", id)
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	rep.Images = imgs

	// Photo-Consent-Warnhinweis nur für Draft (nach Publish irrelevant).
	if rep.State == StateDraft || rep.State == StatePublishFailed {
		rep.PhotoConsentMissing = h.consentMissing(rep.GameID)
	}

	writeJSON(w, http.StatusOK, rep)
}

// loadReport lädt einen Bericht ohne Bilder/Consent-Warnhinweis.
func (h *Handler) loadReport(id int) (*Report, error) {
	var (
		r                Report
		dutySlotID       sql.NullInt64
		homeGoals        sql.NullInt64
		awayGoals        sql.NullInt64
		homeGoalsHT      sql.NullInt64
		awayGoalsHT      sql.NullInt64
		tournamentInt    int
		publishedURL     sql.NullString
		typo3PageUID     sql.NullInt64
		errorMessageText sql.NullString
	)
	err := h.db.QueryRow(
		`SELECT id, game_id, duty_slot_id, author_user_id, state, title,
		        home_goals, away_goals, home_goals_ht, away_goals_ht,
		        tournament, abstract, body_md,
		        published_url, typo3_page_uid, error_message
		 FROM match_reports WHERE id=?`, id,
	).Scan(
		&r.ID, &r.GameID, &dutySlotID, &r.AuthorUserID, &r.State, &r.Title,
		&homeGoals, &awayGoals, &homeGoalsHT, &awayGoalsHT,
		&tournamentInt, &r.Abstract, &r.BodyMarkdown,
		&publishedURL, &typo3PageUID, &errorMessageText,
	)
	if err != nil {
		return nil, err
	}
	r.DutySlotID = nullInt64Ptr(dutySlotID)
	r.HomeGoals = nullInt64Ptr(homeGoals)
	r.AwayGoals = nullInt64Ptr(awayGoals)
	r.HomeGoalsHT = nullInt64Ptr(homeGoalsHT)
	r.AwayGoalsHT = nullInt64Ptr(awayGoalsHT)
	r.Tournament = tournamentInt != 0
	if publishedURL.Valid {
		s := publishedURL.String
		r.PublishedURL = &s
	}
	r.Typo3PageUID = nullInt64Ptr(typo3PageUID)
	if errorMessageText.Valid {
		s := errorMessageText.String
		r.ErrorMessage = &s
	}
	return &r, nil
}

func nullInt64Ptr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// canReadReport implementiert die Read-Sichtbarkeits-Matrix. Nicht zu
// verwechseln mit guardMutation (das ist für Schreib-Rechte).
func canReadReport(claims *auth.Claims, rep *Report) bool {
	if claims.Role == auth.RoleAdmin {
		return true
	}
	// Published-Berichte sind Referenz — jeder Authenticated darf sehen.
	if rep.State == StatePublished {
		return true
	}
	// Autor sieht seinen eigenen Bericht in allen States.
	if rep.AuthorUserID == claims.UserID {
		return true
	}
	// Freigeber (medien|vorstand) sehen alle Nicht-Draft-Berichte —
	// Draft ist Privatzone des Autors, sonst würden auch Halb-Sätze im UI
	// der Freigeber landen.
	if isReviewer(claims) && rep.State != StateDraft {
		return true
	}
	return false
}
