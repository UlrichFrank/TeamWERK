package members

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

type Member struct {
	ID           int     `json:"id"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	DateOfBirth  string  `json:"date_of_birth,omitempty"`
	MemberNumber string  `json:"member_number,omitempty"`
	PassNumber   string  `json:"pass_number,omitempty"`
	JerseyNumber *int    `json:"jersey_number,omitempty"`
	Position     string  `json:"position,omitempty"`
	Gender       string  `json:"gender"`
	Status       string  `json:"status"`
	UserID       *int    `json:"user_id,omitempty"`
	ClubFunction *string `json:"club_function,omitempty"`

	// Extended fields (populated by GetMember)
	Street   *string `json:"street,omitempty"`
	Zip      *string `json:"zip,omitempty"`
	City     *string `json:"city,omitempty"`
	JoinDate *string `json:"join_date,omitempty"`
	IBAN          *string `json:"iban,omitempty"`
	AccountHolder *string `json:"account_holder,omitempty"`
	PhotoURL      *string `json:"photo_url,omitempty"`
	PhotoVisible bool `json:"photo_visible,omitempty"`

	DsgvoVerarbeitung     bool    `json:"dsgvo_verarbeitung,omitempty"`
	DsgvoVerarbeitungDate *string `json:"dsgvo_verarbeitung_date,omitempty"`
	DsgvoWeitergabe       bool    `json:"dsgvo_weitergabe,omitempty"`
	DsgvoWeitergabeDate   *string `json:"dsgvo_weitergabe_date,omitempty"`
	SepaMandat            bool    `json:"sepa_mandat,omitempty"`
	SepaMandatDate        *string `json:"sepa_mandat_date,omitempty"`
	SepaMandatURL         *string `json:"sepa_mandat_url,omitempty"`

	// Linked user contact data (shown when user visibility allows)
	UserPhones   []UserPhone `json:"user_phones,omitempty"`
	UserPhotoURL *string     `json:"user_photo_url,omitempty"`

	WelcomeEmailSentAt *string `json:"welcome_email_sent_at,omitempty"`

	HasPendingProfilDraft bool `json:"has_pending_profil_draft,omitempty"`
	HasPendingBankDraft   bool `json:"has_pending_bank_draft,omitempty"`
}

func scanMember(row interface{ Scan(...any) error }) (Member, error) {
	var m Member
	var jerseyNum, userID sql.NullInt64
	var clubFunc sql.NullString
	err := row.Scan(&m.ID, &m.FirstName, &m.LastName, &m.DateOfBirth, &m.MemberNumber, &m.PassNumber,
		&jerseyNum, &m.Position, &m.Gender, &m.Status, &userID, &clubFunc)
	if err != nil {
		return m, err
	}
	if jerseyNum.Valid {
		n := int(jerseyNum.Int64)
		m.JerseyNumber = &n
	}
	if userID.Valid {
		n := int(userID.Int64)
		m.UserID = &n
	}
	if clubFunc.Valid {
		m.ClubFunction = &clubFunc.String
	}
	return m, nil
}

// GET /api/members
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	search := r.URL.Query().Get("search")
	clubFuncFilter := r.URL.Query().Get("club_function")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Build WHERE additions
	whereExtra := ""
	if clubFuncFilter != "" {
		whereExtra += ` AND club_function = ?`
	}
	if search != "" {
		whereExtra += ` AND (first_name LIKE ? OR last_name LIKE ? OR position LIKE ?)`
	}

	var err error
	var total int

	if claims.Role == "admin" {
		countQuery := `SELECT COUNT(*) FROM members WHERE status != 'ausgetreten'` + whereExtra
		args := buildListArgs(nil, clubFuncFilter, search, nil, nil)
		err = h.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total)
	} else {
		countQuery := `SELECT COUNT(DISTINCT m.id) FROM members m
		 JOIN team_memberships tm ON tm.member_id = m.id
		 JOIN team_trainers tt ON tt.team_id = tm.team_id
		 WHERE tt.user_id = ? AND m.status != 'ausgetreten'` + whereExtra
		args := buildListArgs([]any{claims.UserID}, clubFuncFilter, search, nil, nil)
		err = h.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var rows *sql.Rows
	if claims.Role == "admin" {
		query := `SELECT id, first_name, last_name, COALESCE(date_of_birth,''), COALESCE(member_number,''), COALESCE(pass_number,''),
		        jersey_number, COALESCE(position,''), COALESCE(gender,'u'), status, user_id, club_function
		 FROM members WHERE status != 'ausgetreten'` + whereExtra + ` ORDER BY last_name, first_name LIMIT ? OFFSET ?`
		args := buildListArgs(nil, clubFuncFilter, search, &limit, &offset)
		rows, err = h.db.QueryContext(r.Context(), query, args...)
	} else {
		query := `SELECT DISTINCT m.id, m.first_name, m.last_name, COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
		        m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id, m.club_function
		 FROM members m
		 JOIN team_memberships tm ON tm.member_id = m.id
		 JOIN team_trainers tt ON tt.team_id = tm.team_id
		 WHERE tt.user_id = ? AND m.status != 'ausgetreten'` + whereExtra + ` ORDER BY m.last_name, m.first_name LIMIT ? OFFSET ?`
		args := buildListArgs([]any{claims.UserID}, clubFuncFilter, search, &limit, &offset)
		rows, err = h.db.QueryContext(r.Context(), query, args...)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	result := []Member{}
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			continue
		}
		result = append(result, m)
	}

	// For admin: mark members with pending draft types
	if claims.Role == "admin" && len(result) > 0 {
		ids := make([]interface{}, len(result))
		idxMap := make(map[int]int, len(result))
		for i, m := range result {
			ids[i] = m.ID
			idxMap[m.ID] = i
		}
		placeholders := ""
		for i := range ids {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}
		profilRows, err := h.db.QueryContext(r.Context(),
			`SELECT DISTINCT member_id FROM member_change_drafts WHERE field_name='profil' AND member_id IN (`+placeholders+`)`, ids...)
		if err == nil {
			defer profilRows.Close()
			for profilRows.Next() {
				var mid int
				if profilRows.Scan(&mid) == nil {
					if idx, ok := idxMap[mid]; ok {
						result[idx].HasPendingProfilDraft = true
					}
				}
			}
		}
		bankRows, err := h.db.QueryContext(r.Context(),
			`SELECT DISTINCT member_id FROM member_change_drafts WHERE field_name='bankdaten' AND member_id IN (`+placeholders+`)`, ids...)
		if err == nil {
			defer bankRows.Close()
			for bankRows.Next() {
				var mid int
				if bankRows.Scan(&mid) == nil {
					if idx, ok := idxMap[mid]; ok {
						result[idx].HasPendingBankDraft = true
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": result, "total": total})
}

// buildListArgs constructs the args slice for list queries in order:
// prefix args, club_function, search x3, limit, offset
func buildListArgs(prefix []any, clubFunc, search string, limit, offset *int) []any {
	args := append([]any{}, prefix...)
	if clubFunc != "" {
		args = append(args, clubFunc)
	}
	if search != "" {
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if limit != nil {
		args = append(args, *limit)
	}
	if offset != nil {
		args = append(args, *offset)
	}
	return args
}

// POST /api/members
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FirstName    string  `json:"first_name"`
		LastName     string  `json:"last_name"`
		DateOfBirth  string  `json:"date_of_birth"`
		MemberNumber string  `json:"member_number"`
		PassNumber   string  `json:"pass_number"`
		Position     string  `json:"position"`
		Gender       string  `json:"gender"`
		ClubFunction *string `json:"club_function"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FirstName == "" || req.LastName == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Gender == "" {
		req.Gender = "u"
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO members (first_name, last_name, date_of_birth, member_number, pass_number, position, gender, club_function) VALUES (?,?,?,?,?,?,?,?)`,
		req.FirstName, req.LastName, nullableString(req.DateOfBirth), nullableString(req.MemberNumber),
		nullableString(req.PassNumber), nullableString(req.Position), req.Gender, req.ClubFunction)
	if err != nil {
		http.Error(w, "duplicate pass number or internal error", http.StatusConflict)
		return
	}
	id, _ := res.LastInsertId()
	h.hub.Broadcast("members")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// GET /api/members/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id := r.PathValue("id")

	row := h.db.QueryRowContext(r.Context(), `
		SELECT m.id, m.first_name, m.last_name,
		       COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
		       m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id, m.club_function,
		       m.street, m.zip, m.city, m.join_date, m.iban, m.account_holder,
		       m.photo_path, m.photo_visible,
		       m.dsgvo_verarbeitung, m.dsgvo_verarbeitung_date,
		       m.dsgvo_weitergabe, m.dsgvo_weitergabe_date,
		       m.sepa_mandat, m.sepa_mandat_date, m.sepa_mandat_path,
		       m.welcome_email_sent_at
		FROM members m
		LEFT JOIN users u ON u.id = m.user_id
		WHERE m.id=?`, id)

	var base Member
	var jerseyNum, userID sql.NullInt64
	var clubFunc sql.NullString
	var mStreet, mZip, mCity sql.NullString
	var joinDate, iban, accountHolder sql.NullString
	var photoPath sql.NullString
	var photoVisible int64
	var dsgvoVerarb, dsgvoWeiter, sepaMandat int64
	var dsgvoVerarbDate, dsgvoWeiterDate, sepaMandatDate, sepaMandatPath sql.NullString
	var welcomeEmailSentAt sql.NullString

	err := row.Scan(
		&base.ID, &base.FirstName, &base.LastName, &base.DateOfBirth,
		&base.MemberNumber, &base.PassNumber,
		&jerseyNum, &base.Position, &base.Gender, &base.Status, &userID, &clubFunc,
		&mStreet, &mZip, &mCity, &joinDate, &iban, &accountHolder,
		&photoPath, &photoVisible,
		&dsgvoVerarb, &dsgvoVerarbDate,
		&dsgvoWeiter, &dsgvoWeiterDate,
		&sepaMandat, &sepaMandatDate, &sepaMandatPath,
		&welcomeEmailSentAt,
	)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if jerseyNum.Valid {
		n := int(jerseyNum.Int64)
		base.JerseyNumber = &n
	}
	if userID.Valid {
		n := int(userID.Int64)
		base.UserID = &n
	}
	if clubFunc.Valid {
		base.ClubFunction = &clubFunc.String
	}

	isAdmin := claims.Role == "admin"
	isPrivileged := claims.Role == "admin" || claims.Role == "vorstand" || claims.Role == "trainer"
	isOwn := base.UserID != nil && *base.UserID == claims.UserID

	if mStreet.Valid && mStreet.String != "" {
		s := mStreet.String
		z := mZip.String
		c := mCity.String
		base.Street = &s
		base.Zip = &z
		base.City = &c
	}

	// Photo: always for privileged roles; others only if photo_visible=1
	base.PhotoVisible = photoVisible == 1
	if photoPath.Valid && photoPath.String != "" {
		if isPrivileged || base.PhotoVisible {
			url := "/api/uploads/" + photoPath.String
			base.PhotoURL = &url
		}
	}

	// Admin/own-user fields
	if isAdmin || isOwn {
		if joinDate.Valid {
			base.JoinDate = &joinDate.String
		}
		base.DsgvoVerarbeitung = dsgvoVerarb == 1
		if dsgvoVerarbDate.Valid {
			base.DsgvoVerarbeitungDate = &dsgvoVerarbDate.String
		}
		base.DsgvoWeitergabe = dsgvoWeiter == 1
		if dsgvoWeiterDate.Valid {
			base.DsgvoWeitergabeDate = &dsgvoWeiterDate.String
		}
		base.SepaMandat = sepaMandat == 1
		if sepaMandatDate.Valid {
			base.SepaMandatDate = &sepaMandatDate.String
		}
	}

	// welcome_email_sent_at: admin only
	if isAdmin && welcomeEmailSentAt.Valid {
		base.WelcomeEmailSentAt = &welcomeEmailSentAt.String
	}

	// IBAN, account holder + SEPA document URL: admin only
	if isAdmin {
		if iban.Valid {
			base.IBAN = &iban.String
		}
		if accountHolder.Valid {
			base.AccountHolder = &accountHolder.String
		}
		if sepaMandatPath.Valid && sepaMandatPath.String != "" {
			url := "/api/uploads/" + sepaMandatPath.String
			base.SepaMandatURL = &url
		}
	}

	// Linked user contact data — shown to admin/own, or based on user_visibility
	if base.UserID != nil {
		var pv, av, phv int
		var userPhotoPath sql.NullString
		h.db.QueryRowContext(r.Context(),
			`SELECT COALESCE(uv.phones_visible,0), COALESCE(uv.address_visible,0), COALESCE(uv.photo_visible,0), u.photo_path
			 FROM users u LEFT JOIN user_visibility uv ON uv.user_id=u.id WHERE u.id=?`, *base.UserID).
			Scan(&pv, &av, &phv, &userPhotoPath)

		showPhones := isAdmin || isOwn || pv == 1
		showUserPhoto := isPrivileged || isOwn || phv == 1

		if showPhones {
			phoneRows, err := h.db.QueryContext(r.Context(),
				`SELECT id, label, number, sort_order FROM user_phones WHERE user_id=? ORDER BY sort_order, id`,
				*base.UserID)
			if err == nil {
				defer phoneRows.Close()
				base.UserPhones = []UserPhone{}
				for phoneRows.Next() {
					var p UserPhone
					phoneRows.Scan(&p.ID, &p.Label, &p.Number, &p.SortOrder)
					base.UserPhones = append(base.UserPhones, p)
				}
			}
		}

		if showUserPhoto && userPhotoPath.Valid && userPhotoPath.String != "" {
			url := "/api/uploads/" + userPhotoPath.String
			base.UserPhotoURL = &url
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(base)
}

// PUT /api/members/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id := r.PathValue("id")
	var req struct {
		FirstName    string  `json:"first_name"`
		LastName     string  `json:"last_name"`
		DateOfBirth  string  `json:"date_of_birth"`
		MemberNumber string  `json:"member_number"`
		PassNumber   string  `json:"pass_number"`
		JerseyNumber *int    `json:"jersey_number"`
		Position     string  `json:"position"`
		Gender       string  `json:"gender"`
		Status       string  `json:"status"`
		ClubFunction *string `json:"club_function"`

		Street   string `json:"street"`
		Zip      string `json:"zip"`
		City     string `json:"city"`
		JoinDate      string `json:"join_date"`
		IBAN          string `json:"iban"`
		AccountHolder string `json:"account_holder"`

		PhotoVisible bool `json:"photo_visible"`

		DsgvoVerarbeitung     bool   `json:"dsgvo_verarbeitung"`
		DsgvoVerarbeitungDate string `json:"dsgvo_verarbeitung_date"`
		DsgvoWeitergabe       bool   `json:"dsgvo_weitergabe"`
		DsgvoWeitergabeDate   string `json:"dsgvo_weitergabe_date"`
		SepaMandat            bool   `json:"sepa_mandat"`
		SepaMandatDate        string `json:"sepa_mandat_date"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Gender == "" {
		req.Gender = "u"
	}
	if req.Status == "" {
		req.Status = "aktiv"
	}

	_, err := h.db.ExecContext(r.Context(),
		`UPDATE members SET
			first_name=?, last_name=?, date_of_birth=?, member_number=?, pass_number=?,
			jersey_number=?, position=?, gender=?, club_function=?,
			street=?, zip=?, city=?,
			status=?,
			photo_visible=?,
			updated_at=?
		WHERE id=?`,
		req.FirstName, req.LastName, nullableString(req.DateOfBirth), nullableString(req.MemberNumber),
		nullableString(req.PassNumber), req.JerseyNumber, nullableString(req.Position), req.Gender, req.ClubFunction,
		nullableString(req.Street), nullableString(req.Zip), nullableString(req.City),
		req.Status,
		boolToInt(req.PhotoVisible),
		time.Now(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if claims.Role == "admin" {
		ibanVal := interface{}(nil)
		if req.IBAN != "" {
			ibanVal = req.IBAN
		}
		h.db.ExecContext(r.Context(),
			`UPDATE members SET
				join_date=?, iban=COALESCE(?, iban), account_holder=?,
				dsgvo_verarbeitung=?, dsgvo_verarbeitung_date=?,
				dsgvo_weitergabe=?, dsgvo_weitergabe_date=?,
				sepa_mandat=?, sepa_mandat_date=?
			WHERE id=?`,
			nullableString(req.JoinDate), ibanVal, nullableString(req.AccountHolder),
			boolToInt(req.DsgvoVerarbeitung), nullableString(req.DsgvoVerarbeitungDate),
			boolToInt(req.DsgvoWeitergabe), nullableString(req.DsgvoWeitergabeDate),
			boolToInt(req.SepaMandat), nullableString(req.SepaMandatDate),
			id)
	}

	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/members/:id/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Status string `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(), `UPDATE members SET status=?, updated_at=? WHERE id=?`, req.Status, time.Now(), id)
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/admin/members/:id/user
func (h *Handler) LinkUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		UserID *int `json:"user_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`UPDATE members SET user_id=?, updated_at=? WHERE id=?`,
		req.UserID, time.Now(), id)
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/admin/family-links
func (h *Handler) CreateFamilyLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentUserID int `json:"parent_user_id"`
		MemberID     int `json:"member_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM family_links WHERE member_id=?`, req.MemberID).Scan(&count)
	if count >= 2 {
		http.Error(w, "maximal zwei Erziehungsberechtigte erlaubt", http.StatusConflict)
		return
	}

	h.db.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO family_links (parent_user_id, member_id) VALUES (?,?)`,
		req.ParentUserID, req.MemberID)
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/admin/family-links
func (h *Handler) DeleteFamilyLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentUserID int `json:"parent_user_id"`
		MemberID     int `json:"member_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM family_links WHERE parent_user_id=? AND member_id=?`,
		req.ParentUserID, req.MemberID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/members/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT m.member_number, m.first_name, m.last_name, COALESCE(m.date_of_birth,''),
		        m.gender, COALESCE(m.pass_number,''), m.jersey_number,
		        COALESCE(m.position,''), m.status,
		        COALESCE(u.email,'') AS user_email,
		        COALESCE(up1.email,'') AS parent1_email,
		        COALESCE(up2.email,'') AS parent2_email
		 FROM members m
		 LEFT JOIN users u ON u.id = m.user_id
		 LEFT JOIN (
		   SELECT fl.member_id, MIN(fl.parent_user_id) AS uid
		   FROM family_links fl GROUP BY fl.member_id
		 ) fl1 ON fl1.member_id = m.id
		 LEFT JOIN users up1 ON up1.id = fl1.uid
		 LEFT JOIN (
		   SELECT fl.member_id, MIN(fl.parent_user_id) AS uid
		   FROM family_links fl
		   WHERE fl.parent_user_id > (
		     SELECT MIN(fl2.parent_user_id) FROM family_links fl2 WHERE fl2.member_id = fl.member_id
		   )
		   GROUP BY fl.member_id
		 ) fl2 ON fl2.member_id = m.id
		 LEFT JOIN users up2 ON up2.id = fl2.uid
		 ORDER BY m.last_name, m.first_name`)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="mitglieder.csv"`)
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cw.Write([]string{
		"Mitgliedsnummer", "Vorname", "Nachname", "Geburtsdatum", "Geschlecht",
		"Passnummer", "Trikotnummer", "Position", "Status",
		"Benutzer_Email", "Erziehungsberechtigter1_Email", "Erziehungsberechtigter2_Email",
	})
	for rows.Next() {
		var memberNum, position, userEmail, parent1, parent2 sql.NullString
		var firstName, lastName, dob, passNum, gender, status string
		var jerseyNum sql.NullInt64
		rows.Scan(&memberNum, &firstName, &lastName, &dob, &gender, &passNum, &jerseyNum,
			&position, &status, &userEmail, &parent1, &parent2)
		jerseyStr := ""
		if jerseyNum.Valid {
			jerseyStr = fmt.Sprintf("%d", jerseyNum.Int64)
		}
		cw.Write([]string{
			memberNum.String, firstName, lastName, dob, gender,
			passNum, jerseyStr, position.String, status,
			userEmail.String, parent1.String, parent2.String,
		})
	}
	cw.Flush()
}

type ProfileParent struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserPhone struct {
	ID        int    `json:"id"`
	Label     string `json:"label"`
	Number    string `json:"number"`
	SortOrder int    `json:"sort_order"`
}

type UserVisibility struct {
	PhonesVisible  bool `json:"phones_visible"`
	AddressVisible bool `json:"address_visible"`
	PhotoVisible   bool `json:"photo_visible"`
}

type ProfileResponse struct {
	OwnMember  *Member        `json:"own_member,omitempty"`
	Children   []Member       `json:"children"`
	Parents    []ProfileParent `json:"parents"`
	Street     string          `json:"street,omitempty"`
	Zip        string          `json:"zip,omitempty"`
	City       string          `json:"city,omitempty"`
	PhotoURL   string          `json:"photo_url,omitempty"`
	Phones     []UserPhone     `json:"phones"`
	Visibility UserVisibility  `json:"visibility"`
}

// GET /api/profile/me — returns the logged-in user's linked member profile(s) + contact data
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	resp := ProfileResponse{Children: []Member{}, Parents: []ProfileParent{}, Phones: []UserPhone{}}

	// Own linked member (with extended fields for profile display)
	var ownMemberID int
	err := h.db.QueryRowContext(r.Context(), `SELECT id FROM members WHERE user_id=?`, claims.UserID).Scan(&ownMemberID)
	if err == nil {
		m, merr := h.getMember(ownMemberID)
		if merr == nil {
			resp.OwnMember = m
		}
	}

	// Children — alle verknüpften Mitglieder, bei denen dieser User Erziehungsberechtigter ist
	{
		rows, err := h.db.QueryContext(r.Context(),
			`SELECT m.id, m.first_name, m.last_name, COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
			        m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id, m.club_function
			 FROM members m
			 JOIN family_links fl ON fl.member_id = m.id
			 WHERE fl.parent_user_id=?`, claims.UserID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				if child, err := scanMember(rows); err == nil {
					resp.Children = append(resp.Children, child)
				}
			}
		}
	}

	// Parents — Erziehungsberechtigte des eigenen verknüpften Mitglieds
	{
		rows, err := h.db.QueryContext(r.Context(),
			`SELECT u.id, u.first_name || ' ' || u.last_name, u.email
			 FROM users u
			 JOIN family_links fl ON fl.parent_user_id = u.id
			 JOIN members mem ON mem.id = fl.member_id
			 WHERE mem.user_id=?`, claims.UserID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var p ProfileParent
				rows.Scan(&p.ID, &p.Name, &p.Email)
				resp.Parents = append(resp.Parents, p)
			}
		}
	}

	// Contact data: address, photo_url, phones, visibility
	var street, zip, city, photoPath sql.NullString
	h.db.QueryRowContext(r.Context(),
		`SELECT COALESCE(street,''), COALESCE(zip,''), COALESCE(city,''), COALESCE(photo_path,'') FROM users WHERE id=?`,
		claims.UserID).Scan(&street, &zip, &city, &photoPath)
	resp.Street = street.String
	resp.Zip = zip.String
	resp.City = city.String
	if photoPath.String != "" {
		resp.PhotoURL = "/api/uploads/" + photoPath.String
	}

	phoneRows, err := h.db.QueryContext(r.Context(),
		`SELECT id, label, number, sort_order FROM user_phones WHERE user_id=? ORDER BY sort_order, id`,
		claims.UserID)
	if err == nil {
		defer phoneRows.Close()
		for phoneRows.Next() {
			var p UserPhone
			phoneRows.Scan(&p.ID, &p.Label, &p.Number, &p.SortOrder)
			resp.Phones = append(resp.Phones, p)
		}
	}

	var vis UserVisibility
	var pv, av, phv int
	h.db.QueryRowContext(r.Context(),
		`SELECT phones_visible, address_visible, photo_visible FROM user_visibility WHERE user_id=?`,
		claims.UserID).Scan(&pv, &av, &phv)
	vis.PhonesVisible = pv == 1
	vis.AddressVisible = av == 1
	vis.PhotoVisible = phv == 1
	resp.Visibility = vis

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PUT /api/profile/me — update profile fields (name + address)
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Street    string `json:"street"`
		Zip       string `json:"zip"`
		City      string `json:"city"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`UPDATE users SET first_name=?, last_name=?, street=?, zip=?, city=?, updated_at=? WHERE id=?`,
		nullableString(req.FirstName), nullableString(req.LastName),
		nullableString(req.Street), nullableString(req.Zip), nullableString(req.City), time.Now(), claims.UserID)
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/profile/phones
func (h *Handler) AddPhone(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		Label     string `json:"label"`
		Number    string `json:"number"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Number == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO user_phones (user_id, label, number, sort_order) VALUES (?,?,?,?)`,
		claims.UserID, req.Label, req.Number, req.SortOrder)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// PUT /api/profile/phones/{id}
func (h *Handler) UpdatePhone(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	phoneID := r.PathValue("id")
	var req struct {
		Label  string `json:"label"`
		Number string `json:"number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Number == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`UPDATE user_phones SET label=?, number=? WHERE id=? AND user_id=?`,
		req.Label, req.Number, phoneID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/profile/phones/{id}
func (h *Handler) DeletePhone(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	phoneID := r.PathValue("id")
	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM user_phones WHERE id=? AND user_id=?`, phoneID, claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/profile/visibility
func (h *Handler) UpdateVisibility(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req UserVisibility
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO user_visibility (user_id, phones_visible, address_visible, photo_visible)
		 VALUES (?,?,?,?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   phones_visible=excluded.phones_visible,
		   address_visible=excluded.address_visible,
		   photo_visible=excluded.photo_visible`,
		claims.UserID, boolToInt(req.PhonesVisible), boolToInt(req.AddressVisible), boolToInt(req.PhotoVisible))
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/admin/members/{id}/parents
func (h *Handler) GetMemberParents(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id := r.PathValue("id")

	query := `SELECT u.id, u.first_name || ' ' || u.last_name, u.email
		 FROM users u
		 JOIN family_links fl ON fl.parent_user_id = u.id
		 WHERE fl.member_id=?`
	args := []any{id}
	if claims.Role == "elternteil" {
		query += ` AND fl.parent_user_id=?`
		args = append(args, claims.UserID)
	}

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	result := []ProfileParent{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p ProfileParent
			rows.Scan(&p.ID, &p.Name, &p.Email)
			result = append(result, p)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET/PUT /api/profile/vehicle
func (h *Handler) GetVehicle(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var seats int
	var notes string
	h.db.QueryRowContext(r.Context(),
		`SELECT seats, COALESCE(notes,'') FROM vehicle_info WHERE user_id=?`, claims.UserID).
		Scan(&seats, &notes)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"seats": seats, "notes": notes})
}

func (h *Handler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		Seats int    `json:"seats"`
		Notes string `json:"notes"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	h.db.ExecContext(r.Context(),
		`INSERT INTO vehicle_info (user_id, seats, notes, updated_at) VALUES (?,?,?,?)
		 ON CONFLICT(user_id) DO UPDATE SET seats=excluded.seats, notes=excluded.notes, updated_at=excluded.updated_at`,
		claims.UserID, req.Seats, req.Notes, time.Now())
	w.WriteHeader(http.StatusNoContent)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ImportRow holds the result for a single CSV row.
type ImportRow struct {
	Line    int      `json:"line"`
	Status  string   `json:"status"` // created | updated | unchanged | error
	Name    string   `json:"name"`
	DOB     string   `json:"dob,omitempty"`
	Changes []string `json:"changes,omitempty"`
	Message string   `json:"message,omitempty"`
}

// ImportReport is the full response body for POST /api/members/import.
type ImportReport struct {
	Total     int         `json:"total"`
	Created   int         `json:"created"`
	Updated   int         `json:"updated"`
	Unchanged int         `json:"unchanged"`
	Errors    int         `json:"errors"`
	Rows      []ImportRow `json:"rows"`
}

// POST /api/members/import
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	mode := r.FormValue("mode") // "append" or "update"
	if mode != "append" && mode != "update" {
		mode = "append"
	}

	raw, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "cannot read file", http.StatusBadRequest)
		return
	}
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf")) // strip UTF-8 BOM

	// Auto-detect delimiter from first line
	firstNewline := bytes.IndexByte(raw, '\n')
	firstLineBytes := raw
	if firstNewline > 0 {
		firstLineBytes = raw[:firstNewline]
	}
	delim := rune(',')
	if bytes.ContainsRune(firstLineBytes, ';') {
		delim = ';'
	}

	cr := csv.NewReader(bytes.NewReader(raw))
	cr.Comma = delim
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		http.Error(w, "cannot read CSV header", http.StatusBadRequest)
		return
	}
	colIdx := make(map[string]int, len(header))
	for i, name := range header {
		colIdx[strings.TrimSpace(name)] = i
	}
	col := func(row []string, name string) string {
		idx, ok := colIdx[name]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	if _, ok := colIdx["Vorname"]; !ok {
		http.Error(w, "missing required column: Vorname", http.StatusBadRequest)
		return
	}
	if _, ok := colIdx["Nachname"]; !ok {
		http.Error(w, "missing required column: Nachname", http.StatusBadRequest)
		return
	}

	allRows, err := cr.ReadAll()
	if err != nil {
		http.Error(w, "cannot parse CSV", http.StatusBadRequest)
		return
	}

	// Duplicate detection within CSV
	type dupKey struct{ first, last, dob string }
	seenAt := make(map[dupKey]int)            // key → first line number
	dupOf := make(map[int]int)                // later line → first line
	firstDupPartner := make(map[int]int)      // first line → first later duplicate
	isDupLine := make(map[int]bool)
	for i, row := range allRows {
		lineNum := i + 2
		k := dupKey{
			first: strings.ToLower(col(row, "Vorname")),
			last:  strings.ToLower(col(row, "Nachname")),
			dob:   col(row, "Geburtsdatum"),
		}
		if prev, exists := seenAt[k]; exists {
			dupOf[lineNum] = prev
			isDupLine[lineNum] = true
			isDupLine[prev] = true
			if _, alreadyHasPartner := firstDupPartner[prev]; !alreadyHasPartner {
				firstDupPartner[prev] = lineNum
			}
		} else {
			seenAt[k] = lineNum
		}
	}

	report := ImportReport{Rows: make([]ImportRow, 0, len(allRows))}

	for i, row := range allRows {
		lineNum := i + 2
		firstName := col(row, "Vorname")
		lastName := col(row, "Nachname")
		displayName := lastName + ", " + firstName
		dob := col(row, "Geburtsdatum")

		if firstName == "" || lastName == "" {
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "error", Name: displayName,
				Message: "Vorname und Nachname sind Pflichtfelder",
			})
			report.Errors++
			continue
		}

		if isDupLine[lineNum] {
			var msg string
			if first, isLater := dupOf[lineNum]; isLater {
				msg = fmt.Sprintf("Mehrfach in CSV (zuerst Zeile %d)", first)
			} else {
				msg = fmt.Sprintf("Mehrfach in CSV (auch Zeile %d)", firstDupPartner[lineNum])
			}
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "error", Name: displayName, Message: msg,
			})
			report.Errors++
			continue
		}

		// DB lookup by name (+ dob as tiebreaker when present)
		query := `SELECT id, member_number, COALESCE(date_of_birth,''),
		                 pass_number, jersey_number, position, status, gender, user_id
		          FROM members
		          WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?)`
		args := []interface{}{firstName, lastName}
		if dob != "" {
			query += ` AND COALESCE(date_of_birth,'')=?`
			args = append(args, dob)
		}
		query += ` LIMIT 1`

		var (
			existingID                             int
			dbMemberNum, dbPassNum, dbPosition     sql.NullString
			dbDOB, dbGender, dbStatus              string
			dbJerseyNum                            sql.NullInt64
			dbUserID                               sql.NullInt64
		)
		scanErr := h.db.QueryRowContext(r.Context(), query, args...).
			Scan(&existingID, &dbMemberNum, &dbDOB, &dbPassNum, &dbJerseyNum, &dbPosition,
				&dbStatus, &dbGender, &dbUserID)

		if scanErr == sql.ErrNoRows {
			// New member — insert
			gender := col(row, "Geschlecht")
			if gender == "" {
				gender = "u"
			}
			status := col(row, "Status")
			if status == "" {
				status = "aktiv"
			}
			jerseyArg, _ := parseOptionalInt(col(row, "Trikotnummer"))
			res, insErr := h.db.ExecContext(r.Context(),
				`INSERT INTO members (member_number, first_name, last_name, date_of_birth,
				                      pass_number, jersey_number, position, status, gender)
				 VALUES (?,?,?,?,?,?,?,?,?)`,
				nullableString(col(row, "Mitgliedsnummer")), firstName, lastName,
				nullableString(dob), nullableString(col(row, "Passnummer")),
				jerseyArg, nullableString(col(row, "Position")), status, gender)
			if insErr != nil {
				report.Rows = append(report.Rows, ImportRow{
					Line: lineNum, Status: "error", Name: displayName,
					Message: "Fehler beim Anlegen: " + insErr.Error(),
				})
				report.Errors++
				continue
			}
			newID, _ := res.LastInsertId()
			h.applyLinkUpdates(r, int(newID), row, col, sql.NullInt64{}, true)
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "created", Name: displayName, DOB: dob,
			})
			report.Created++
			continue
		}
		if scanErr != nil {
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "error", Name: displayName,
				Message: "DB-Fehler: " + scanErr.Error(),
			})
			report.Errors++
			continue
		}

		// Existing member
		if mode == "append" {
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "unchanged", Name: displayName,
			})
			report.Unchanged++
			continue
		}

		// mode == "update": apply non-empty changed fields
		var setClauses []string
		var setArgs []interface{}
		var changes []string

		addChange := func(csvVal, dbVal, label, column string) {
			if csvVal == "" || csvVal == dbVal {
				return
			}
			setClauses = append(setClauses, column+"=?")
			setArgs = append(setArgs, csvVal)
			changes = append(changes, fmt.Sprintf("%s: %q → %q", label, dbVal, csvVal))
		}
		addNullableChange := func(csvVal string, dbVal sql.NullString, label, column string) {
			if csvVal == "" || csvVal == dbVal.String {
				return
			}
			setClauses = append(setClauses, column+"=?")
			setArgs = append(setArgs, csvVal)
			changes = append(changes, fmt.Sprintf("%s: %q → %q", label, dbVal.String, csvVal))
		}

		addNullableChange(col(row, "Mitgliedsnummer"), dbMemberNum, "Mitgliedsnummer", "member_number")
		addChange(col(row, "Geburtsdatum"), dbDOB, "Geburtsdatum", "date_of_birth")
		addChange(col(row, "Geschlecht"), dbGender, "Geschlecht", "gender")
		addNullableChange(col(row, "Passnummer"), dbPassNum, "Passnummer", "pass_number")
		addNullableChange(col(row, "Position"), dbPosition, "Position", "position")
		addChange(col(row, "Status"), dbStatus, "Status", "status")

		if jerseyRaw := col(row, "Trikotnummer"); jerseyRaw != "" {
			dbJerseyStr := ""
			if dbJerseyNum.Valid {
				dbJerseyStr = fmt.Sprintf("%d", dbJerseyNum.Int64)
			}
			if jerseyRaw != dbJerseyStr {
				n, _ := parseOptionalInt(jerseyRaw)
				setClauses = append(setClauses, "jersey_number=?")
				setArgs = append(setArgs, n)
				changes = append(changes, fmt.Sprintf("Trikotnummer: %q → %q", dbJerseyStr, jerseyRaw))
			}
		}

		linkNotes := h.applyLinkUpdates(r, existingID, row, col, dbUserID, false)
		changes = append(changes, linkNotes...)

		if len(setClauses) > 0 {
			setArgs = append(setArgs, existingID)
			_, updErr := h.db.ExecContext(r.Context(),
				"UPDATE members SET "+strings.Join(setClauses, ", ")+", updated_at=CURRENT_TIMESTAMP WHERE id=?",
				setArgs...)
			if updErr != nil {
				report.Rows = append(report.Rows, ImportRow{
					Line: lineNum, Status: "error", Name: displayName,
					Message: "Fehler beim Aktualisieren: " + updErr.Error(),
				})
				report.Errors++
				continue
			}
		}

		if len(changes) > 0 {
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "updated", Name: displayName, Changes: changes,
			})
			report.Updated++
		} else {
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "unchanged", Name: displayName,
			})
			report.Unchanged++
		}
	}

	report.Total = report.Created + report.Updated + report.Unchanged + report.Errors
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// applyLinkUpdates creates user/family-link associations and returns change notes.
// Pass isNew=true for freshly inserted members to suppress notes (all links are trivially new).
func (h *Handler) applyLinkUpdates(r *http.Request, memberID int, row []string, col func([]string, string) string, dbUserID sql.NullInt64, isNew bool) []string {
	var notes []string

	if email := col(row, "Benutzer_Email"); email != "" && !dbUserID.Valid {
		var uid int
		if err := h.db.QueryRowContext(r.Context(), `SELECT id FROM users WHERE email=?`, email).Scan(&uid); err == nil {
			h.db.ExecContext(r.Context(), `UPDATE members SET user_id=? WHERE id=?`, uid, memberID)
			if !isNew {
				notes = append(notes, fmt.Sprintf("Benutzer_Email: → %q (verknüpft)", email))
			}
		} else if !isNew {
			notes = append(notes, fmt.Sprintf("Benutzer_Email: %q (nicht gefunden)", email))
		}
	}

	for idx, colName := range []string{"Erziehungsberechtigter1_Email", "Erziehungsberechtigter2_Email"} {
		email := col(row, colName)
		if email == "" {
			continue
		}
		var uid int
		if err := h.db.QueryRowContext(r.Context(), `SELECT id FROM users WHERE email=?`, email).Scan(&uid); err != nil {
			if !isNew {
				notes = append(notes, fmt.Sprintf("Erziehungsber. %d: %q (nicht gefunden)", idx+1, email))
			}
			continue
		}
		var exists int
		h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?`, uid, memberID).Scan(&exists)
		if exists == 0 {
			if _, insErr := h.db.ExecContext(r.Context(),
				`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, uid, memberID); insErr == nil && !isNew {
				notes = append(notes, fmt.Sprintf("Erziehungsber. %d: → %q (verknüpft)", idx+1, email))
			}
		}
	}
	return notes
}

// POST /api/admin/users/{id}/create-member
func (h *Handler) CreateMemberFromUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")

	var firstName, lastName string
	err := h.db.QueryRowContext(r.Context(), `SELECT first_name, last_name FROM users WHERE id=?`, userID).Scan(&firstName, &lastName)
	if err == sql.ErrNoRows {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var existingMemberID int
	err = h.db.QueryRowContext(r.Context(), `SELECT id FROM members WHERE user_id=?`, userID).Scan(&existingMemberID)
	if err == nil {
		http.Error(w, "user already has a member record", http.StatusConflict)
		return
	}
	if err != sql.ErrNoRows {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO members (first_name, last_name, status, user_id) VALUES (?,?,?,?)`,
		firstName, lastName, "aktiv", userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	memberID, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"member_id": memberID})
}

func (h *Handler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM members WHERE id=?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseOptionalInt(s string) (interface{}, bool) {
	if s == "" {
		return nil, false
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return nil, false
	}
	return n, true
}
