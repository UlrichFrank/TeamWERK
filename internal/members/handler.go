package members

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/policy"
	"github.com/teamstuttgart/teamwerk/internal/sepa"
)

type Handler struct {
	db  *sql.DB
	hub *hub.EventHub
}

func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { return &Handler{db: db, hub: h} }

type Member struct {
	ID            int      `json:"id"`
	FirstName     string   `json:"first_name"`
	LastName      string   `json:"last_name"`
	DateOfBirth   string   `json:"date_of_birth,omitempty"`
	BirthYear     *int     `json:"birth_year,omitempty"`
	MemberNumber  string   `json:"member_number,omitempty"`
	PassNumber    string   `json:"pass_number,omitempty"`
	JerseyNumber  *int     `json:"jersey_number,omitempty"`
	Position      string   `json:"position,omitempty"`
	Gender        string   `json:"gender"`
	Status        string   `json:"status"`
	UserID        *int     `json:"user_id,omitempty"`
	ClubFunctions []string `json:"club_functions"`

	// Extended fields (populated by GetMember)
	Street           *string `json:"street,omitempty"`
	Zip              *string `json:"zip,omitempty"`
	City             *string `json:"city,omitempty"`
	HomeClub         *string `json:"home_club,omitempty"`
	HomeClubID       *int    `json:"home_club_id,omitempty"`
	HomeClubName     *string `json:"home_club_name,omitempty"`
	JoinDate         *string `json:"join_date,omitempty"`
	IBAN             *string `json:"iban,omitempty"`
	AccountHolder    *string `json:"account_holder,omitempty"`
	PhotoURL         *string `json:"photo_url,omitempty"`
	PhotoVisible     bool    `json:"photo_visible,omitempty"`
	PhonesVisible    bool    `json:"phones_visible,omitempty"`
	AddressVisible   bool    `json:"address_visible,omitempty"`
	EmailVisible     bool    `json:"email_visible,omitempty"`
	CrossTeamVisible bool    `json:"cross_team_visible,omitempty"`

	DsgvoVerarbeitung     bool    `json:"dsgvo_verarbeitung,omitempty"`
	DsgvoVerarbeitungDate *string `json:"dsgvo_verarbeitung_date,omitempty"`
	DsgvoWeitergabe       bool    `json:"dsgvo_weitergabe,omitempty"`
	DsgvoWeitergabeDate   *string `json:"dsgvo_weitergabe_date,omitempty"`
	SepaMandat            bool    `json:"sepa_mandat,omitempty"`
	SepaMandatDate        *string `json:"sepa_mandat_date,omitempty"`
	SepaMandatURL         *string `json:"sepa_mandat_url,omitempty"`
	Beitragsfrei          bool    `json:"beitragsfrei,omitempty"`
	Zweitspielrecht       bool    `json:"zweitspielrecht,omitempty"`

	// Linked user contact data (shown when user visibility allows)
	UserPhones   []UserPhone `json:"user_phones,omitempty"`
	UserPhotoURL *string     `json:"user_photo_url,omitempty"`

	WelcomeEmailSentAt *string `json:"welcome_email_sent_at,omitempty"`

	HasPendingProfilDraft bool `json:"has_pending_profil_draft,omitempty"`
	HasPendingBankDraft   bool `json:"has_pending_bank_draft,omitempty"`

	AbsencesPublic int `json:"absences_public"`

	// MemberNumberConflict markiert Nummern-Konflikte für die Übersicht:
	// "duplicate" | "non_numeric" | "missing" | "" (kein Konflikt). Nur für
	// Admin/Vorstand befüllt, da es ein administrativer Hinweis ist.
	MemberNumberConflict string `json:"member_number_conflict,omitempty"`
}

// nextMemberNumber liefert die nächste freie Mitgliedsnummer (höchste vorhandene
// numerische Nummer + 1, ohne Lücken-Reuse). Nicht-numerische Bestandswerte werden
// über das GLOB-Muster von der Maximum-Bestimmung ausgenommen.
func nextMemberNumber(ctx context.Context, db *sql.DB) (string, error) {
	var maxNum sql.NullInt64
	err := db.QueryRowContext(ctx,
		`SELECT MAX(CAST(member_number AS INTEGER)) FROM members WHERE member_number GLOB '[0-9]*'`).Scan(&maxNum)
	if err != nil {
		return "", err
	}
	if maxNum.Valid {
		return strconv.FormatInt(maxNum.Int64+1, 10), nil
	}
	return "1", nil
}

// isNumericMemberNumber prüft, ob eine Mitgliedsnummer rein numerisch ist.
func isNumericMemberNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// memberNumberConflict klassifiziert den Nummern-Konflikt eines Mitglieds.
// honorar-Mitglieder ohne Nummer sind kein Konflikt.
func memberNumberConflict(number, status string, duplicates map[string]bool) string {
	if number == "" {
		if status == "honorar" {
			return ""
		}
		return "missing"
	}
	if duplicates[number] {
		return "duplicate"
	}
	if !isNumericMemberNumber(number) {
		return "non_numeric"
	}
	return ""
}

func scanMember(row interface{ Scan(...any) error }) (Member, error) {
	var m Member
	var jerseyNum, userID sql.NullInt64
	var clubFunctionsStr string
	err := row.Scan(&m.ID, &m.FirstName, &m.LastName, &m.DateOfBirth, &m.MemberNumber, &m.PassNumber,
		&jerseyNum, &m.Position, &m.Gender, &m.Status, &userID, &clubFunctionsStr)
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
	m.ClubFunctions = parseFunctions(clubFunctionsStr)
	return m, nil
}

// redactMemberForScopedViewer reduces a member to the fields a kader-scoped viewer
// (pure trainer) may see: names, year of birth (not the exact date), pass number,
// club functions, plus sport fields (jersey/position/gender/status). Administrative
// fields — exact date of birth, member number, account linkage — are stripped.
func redactMemberForScopedViewer(m Member) Member {
	if len(m.DateOfBirth) >= 4 {
		if y, err := strconv.Atoi(m.DateOfBirth[:4]); err == nil {
			m.BirthYear = &y
		}
	}
	m.DateOfBirth = ""
	m.MemberNumber = ""
	m.UserID = nil
	return m
}

func parseFunctions(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
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
		whereExtra += ` AND EXISTS(SELECT 1 FROM member_club_functions WHERE member_id=m.id AND function=?)`
	}
	if search != "" {
		whereExtra += ` AND (
			m.first_name LIKE ? OR m.last_name LIKE ? OR
			COALESCE(m.position,'') LIKE ? OR
			COALESCE(m.member_number,'') LIKE ? OR
			COALESCE(m.pass_number,'') LIKE ? OR
			COALESCE(CAST(m.jersey_number AS TEXT),'') LIKE ? OR
			COALESCE(m.street,'') LIKE ? OR
			COALESCE(m.zip,'') LIKE ? OR
			COALESCE(m.city,'') LIKE ? OR
			COALESCE(m.home_club,'') LIKE ? OR
			m.status LIKE ? OR
			EXISTS(SELECT 1 FROM users u WHERE u.id = m.user_id AND u.email LIKE ?)
		)`
	}

	var err error
	var total int

	const clubFuncSubquery = `COALESCE((SELECT GROUP_CONCAT(mcf.function, ',') FROM member_club_functions mcf WHERE mcf.member_id = m.id), '')`

	p := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions, IsParent: claims.IsParent}
	scopeWhere, scopeNeedsUserID := policy.ScopeMembersQuery(p)
	// Trainer searching specifically for trainers gets club-wide results for kader assignment.
	trainerWide := slices.Contains(p.ClubFunctions, "trainer") && clubFuncFilter == "trainer"
	wideSearch := scopeWhere == "1=1" || trainerWide

	// Wide-only filters (not meaningful for team-scoped searches)
	if wideSearch {
		if r.URL.Query().Get("unlinked_user") == "1" {
			whereExtra += ` AND m.user_id IS NULL AND NOT EXISTS (SELECT 1 FROM family_links WHERE member_id = m.id)`
		}
		if r.URL.Query().Get("has_draft") == "1" {
			whereExtra += ` AND EXISTS (SELECT 1 FROM member_change_drafts WHERE member_id = m.id)`
		}
	}

	if wideSearch {
		countQuery := `SELECT COUNT(*) FROM members m WHERE status != 'ausgetreten'` + whereExtra
		args := buildListArgs(nil, clubFuncFilter, search, nil, nil)
		err = h.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total)
	} else {
		countQuery := `SELECT COUNT(*) FROM members m WHERE m.status != 'ausgetreten' AND ` + scopeWhere + whereExtra
		var prefix []any
		if scopeNeedsUserID {
			prefix = []any{p.UserID}
		}
		args := buildListArgs(prefix, clubFuncFilter, search, nil, nil)
		err = h.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var rows *sql.Rows
	if wideSearch {
		query := `SELECT m.id, m.first_name, m.last_name, COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
		        m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id, ` + clubFuncSubquery + `
		 FROM members m WHERE m.status != 'ausgetreten'` + whereExtra + ` ORDER BY m.last_name, m.first_name LIMIT ? OFFSET ?`
		args := buildListArgs(nil, clubFuncFilter, search, &limit, &offset)
		rows, err = h.db.QueryContext(r.Context(), query, args...)
	} else {
		query := `SELECT m.id, m.first_name, m.last_name, COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
		        m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id, ` + clubFuncSubquery + `
		 FROM members m WHERE m.status != 'ausgetreten' AND ` + scopeWhere + whereExtra + ` ORDER BY m.last_name, m.first_name LIMIT ? OFFSET ?`
		var prefix []any
		if scopeNeedsUserID {
			prefix = []any{p.UserID}
		}
		args := buildListArgs(prefix, clubFuncFilter, search, &limit, &offset)
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

	// For admin/vorstand: mark members with pending draft types
	if policy.IsVorstandLike(p) && len(result) > 0 {
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

	type memberItem struct {
		Member
		Can policy.CanFlags `json:"can"`
	}
	redact := !policy.CanReadMemberAdminFields(p)

	// Nummern-Konflikte nur für Admin/Vorstand ermitteln (administrativer Hinweis;
	// für redacted Viewer wird keine Nummer und kein Flag ausgeliefert).
	if !redact && len(result) > 0 {
		duplicates := map[string]bool{}
		dupRows, derr := h.db.QueryContext(r.Context(),
			`SELECT member_number FROM members WHERE member_number IS NOT NULL AND member_number <> '' GROUP BY member_number HAVING COUNT(*) > 1`)
		if derr == nil {
			for dupRows.Next() {
				var n string
				if dupRows.Scan(&n) == nil {
					duplicates[n] = true
				}
			}
			dupRows.Close()
		}
		for i := range result {
			result[i].MemberNumberConflict = memberNumberConflict(result[i].MemberNumber, result[i].Status, duplicates)
		}
	}

	annotated := make([]memberItem, len(result))
	for i, m := range result {
		memberUserID := 0
		if m.UserID != nil {
			memberUserID = *m.UserID
		}
		can := policy.MemberCan(p, memberUserID)
		if redact {
			m = redactMemberForScopedViewer(m)
		}
		annotated[i] = memberItem{Member: m, Can: can}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": annotated, "total": total})
}

// buildListArgs constructs the args slice for list queries in order:
// prefix args, club_function, search x12, limit, offset
func buildListArgs(prefix []any, clubFunc, search string, limit, offset *int) []any {
	args := append([]any{}, prefix...)
	if clubFunc != "" {
		args = append(args, clubFunc)
	}
	if search != "" {
		s := "%" + search + "%"
		for range 12 {
			args = append(args, s)
		}
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
		FirstName     string   `json:"first_name"`
		LastName      string   `json:"last_name"`
		DateOfBirth   string   `json:"date_of_birth"`
		MemberNumber  string   `json:"member_number"`
		PassNumber    string   `json:"pass_number"`
		Position      string   `json:"position"`
		Gender        string   `json:"gender"`
		ClubFunctions []string `json:"club_functions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FirstName == "" || req.LastName == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Gender == "" {
		req.Gender = "u"
	}
	// Die Mitgliedsnummer ist systemverwaltet: ein im Request mitgeschickter Wert
	// wird ignoriert, es wird immer automatisch die nächste freie Nummer vergeben.
	// Bei seltener Race (parallele Anlage → Unique-Verletzung) wird die Nummer neu
	// bestimmt und der INSERT wiederholt.
	var id int64
	for attempt := 0; attempt < 3; attempt++ {
		memberNumber, err := nextMemberNumber(r.Context(), h.db)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		res, err := h.db.ExecContext(r.Context(),
			`INSERT INTO members (first_name, last_name, date_of_birth, member_number, pass_number, position, gender) VALUES (?,?,?,?,?,?,?)`,
			req.FirstName, req.LastName, nullableString(req.DateOfBirth), nullableString(memberNumber),
			nullableString(req.PassNumber), nullableString(req.Position), req.Gender)
		if err != nil {
			if attempt == 2 {
				http.Error(w, "duplicate pass number or internal error", http.StatusConflict)
				return
			}
			continue
		}
		id, _ = res.LastInsertId()
		break
	}
	h.writeClubFunctions(r.Context(), int(id), req.ClubFunctions)
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
		       m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id,
		       COALESCE((SELECT GROUP_CONCAT(mcf.function,',') FROM member_club_functions mcf WHERE mcf.member_id=m.id),''),
		       m.street, m.zip, m.city, m.home_club, m.home_club_id, COALESCE(sv.name,''), m.join_date, m.iban, m.account_holder,
		       m.photo_path, m.photo_visible,
		       m.dsgvo_verarbeitung, m.dsgvo_verarbeitung_date,
		       m.dsgvo_weitergabe, m.dsgvo_weitergabe_date,
		       m.sepa_mandat, m.sepa_mandat_date, m.sepa_mandat_path,
		       m.welcome_email_sent_at,
		       m.beitragsfrei, m.zweitspielrecht
		FROM members m
		LEFT JOIN users u ON u.id = m.user_id
		LEFT JOIN stammvereine sv ON sv.id = m.home_club_id
		WHERE m.id=?`, id)

	var base Member
	var jerseyNum, userID, homeClubID sql.NullInt64
	var clubFunctionsStr string
	var mStreet, mZip, mCity, mHomeClub, mHomeClubName sql.NullString
	var joinDate, iban, accountHolder sql.NullString
	var photoPath sql.NullString
	var photoVisible int64
	var dsgvoVerarb, dsgvoWeiter, sepaMandat int64
	var dsgvoVerarbDate, dsgvoWeiterDate, sepaMandatDate, sepaMandatPath sql.NullString
	var welcomeEmailSentAt sql.NullString
	var beitragsfrei, zweitspielrecht int64

	err := row.Scan(
		&base.ID, &base.FirstName, &base.LastName, &base.DateOfBirth,
		&base.MemberNumber, &base.PassNumber,
		&jerseyNum, &base.Position, &base.Gender, &base.Status, &userID, &clubFunctionsStr,
		&mStreet, &mZip, &mCity, &mHomeClub, &homeClubID, &mHomeClubName, &joinDate, &iban, &accountHolder,
		&photoPath, &photoVisible,
		&dsgvoVerarb, &dsgvoVerarbDate,
		&dsgvoWeiter, &dsgvoWeiterDate,
		&sepaMandat, &sepaMandatDate, &sepaMandatPath,
		&welcomeEmailSentAt,
		&beitragsfrei, &zweitspielrecht,
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
	base.ClubFunctions = parseFunctions(clubFunctionsStr)

	isAdmin := claims.Role == "admin" || claims.HasFunction("vorstand")
	isPrivileged := claims.Role == "admin" || claims.HasFunction("vorstand") || claims.HasFunction("trainer") || claims.HasFunction("sportliche_leitung")
	isOwn := base.UserID != nil && *base.UserID == claims.UserID
	isParent := h.isParentOf(r.Context(), claims.UserID, base.ID)

	if mStreet.Valid && mStreet.String != "" {
		s := mStreet.String
		z := mZip.String
		c := mCity.String
		base.Street = &s
		base.Zip = &z
		base.City = &c
	}
	if mHomeClub.Valid && mHomeClub.String != "" {
		base.HomeClub = &mHomeClub.String
	}
	if homeClubID.Valid {
		n := int(homeClubID.Int64)
		base.HomeClubID = &n
		if mHomeClubName.Valid && mHomeClubName.String != "" {
			base.HomeClubName = &mHomeClubName.String
		}
	}
	base.Zweitspielrecht = zweitspielrecht == 1

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
		base.Beitragsfrei = beitragsfrei == 1
	}

	// welcome_email_sent_at: admin only
	if isAdmin && welcomeEmailSentAt.Valid {
		base.WelcomeEmailSentAt = &welcomeEmailSentAt.String
	}

	// IBAN, account holder: admin only
	if isAdmin {
		if iban.Valid {
			base.IBAN = &iban.String
		}
		if accountHolder.Valid {
			base.AccountHolder = &accountHolder.String
		}
	}
	// SEPA document URL: admin, own member, parent, vorstand
	if (isAdmin || isOwn || isParent || claims.HasFunction("vorstand")) && sepaMandatPath.Valid && sepaMandatPath.String != "" {
		url := fmt.Sprintf("/api/members/%d/sepa-mandat/download", base.ID)
		base.SepaMandatURL = &url
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

	memberUserID := 0
	if base.UserID != nil {
		memberUserID = *base.UserID
	}
	p2 := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions, IsParent: claims.IsParent}
	resp := struct {
		Member
		Can policy.CanFlags `json:"can"`
	}{Member: base, Can: policy.MemberCan(p2, memberUserID)}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PUT /api/members/:id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id := r.PathValue("id")
	var req struct {
		FirstName     string   `json:"first_name"`
		LastName      string   `json:"last_name"`
		DateOfBirth   string   `json:"date_of_birth"`
		MemberNumber  string   `json:"member_number"`
		PassNumber    string   `json:"pass_number"`
		JerseyNumber  *int     `json:"jersey_number"`
		Position      string   `json:"position"`
		Gender        string   `json:"gender"`
		Status        string   `json:"status"`
		ClubFunctions []string `json:"club_functions"`

		Street        string `json:"street"`
		Zip           string `json:"zip"`
		City          string `json:"city"`
		HomeClub      string `json:"home_club"`
		HomeClubID    *int   `json:"home_club_id"`
		JoinDate      string `json:"join_date"`
		IBAN          string `json:"iban"`
		AccountHolder string `json:"account_holder"`

		PhotoVisible     bool `json:"photo_visible"`
		CrossTeamVisible bool `json:"cross_team_visible"`

		DsgvoVerarbeitung     bool   `json:"dsgvo_verarbeitung"`
		DsgvoVerarbeitungDate string `json:"dsgvo_verarbeitung_date"`
		DsgvoWeitergabe       bool   `json:"dsgvo_weitergabe"`
		DsgvoWeitergabeDate   string `json:"dsgvo_weitergabe_date"`
		SepaMandat            bool   `json:"sepa_mandat"`
		SepaMandatDate        string `json:"sepa_mandat_date"`
		Beitragsfrei          bool   `json:"beitragsfrei"`
		Zweitspielrecht       bool   `json:"zweitspielrecht"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Gender == "" {
		req.Gender = "u"
	}
	if req.Status == "" {
		req.Status = "aktiv"
	}
	if req.Status == "honorar" {
		req.MemberNumber = ""
		req.PassNumber = ""
		req.HomeClub = ""
		req.HomeClubID = nil
		filtered := []string{}
		for _, f := range req.ClubFunctions {
			if f == "trainer" {
				filtered = append(filtered, f)
			}
		}
		req.ClubFunctions = filtered
	}

	// home_club_id muss (falls gesetzt) auf einen existierenden Stammverein zeigen.
	if req.HomeClubID != nil {
		var exists int
		if err := h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM stammvereine WHERE id=?`, *req.HomeClubID).Scan(&exists); err != nil || exists == 0 {
			http.Error(w, "ungültige home_club_id", http.StatusBadRequest)
			return
		}
	}

	// Die Mitgliedsnummer ist systemverwaltet und read-only. Nur Admins dürfen sie
	// nachträglich korrigieren; für alle anderen bleibt der bestehende Wert erhalten.
	// (honorar leert die Nummer oben bereits für alle — bewusst beibehalten.)
	memberNumber := req.MemberNumber
	if req.Status != "honorar" {
		if claims.Role == "admin" {
			if memberNumber != "" {
				var otherID int
				if h.db.QueryRowContext(r.Context(),
					`SELECT id FROM members WHERE member_number=? AND id<>?`, memberNumber, id).Scan(&otherID) == nil {
					http.Error(w, fmt.Sprintf("Mitgliedsnummer %s ist bereits vergeben", memberNumber), http.StatusConflict)
					return
				}
			}
		} else {
			var current sql.NullString
			h.db.QueryRowContext(r.Context(), `SELECT member_number FROM members WHERE id=?`, id).Scan(&current)
			memberNumber = current.String
		}
	}

	_, err := h.db.ExecContext(r.Context(),
		`UPDATE members SET
			first_name=?, last_name=?, date_of_birth=?, member_number=?, pass_number=?,
			jersey_number=?, position=?, gender=?,
			street=?, zip=?, city=?, home_club=?, home_club_id=?,
			status=?,
			photo_visible=?,
			cross_team_visible=?,
			zweitspielrecht=?,
			updated_at=?
		WHERE id=?`,
		req.FirstName, req.LastName, nullableString(req.DateOfBirth), nullableString(memberNumber),
		nullableString(req.PassNumber), req.JerseyNumber, nullableString(req.Position), req.Gender,
		nullableString(req.Street), nullableString(req.Zip), nullableString(req.City), nullableString(req.HomeClub), req.HomeClubID,
		req.Status,
		boolToInt(req.PhotoVisible),
		boolToInt(req.CrossTeamVisible),
		boolToInt(req.Zweitspielrecht),
		time.Now(), id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	pu := &policy.Principal{UserID: claims.UserID, Role: claims.Role, ClubFunctions: claims.ClubFunctions}
	if policy.IsVorstandLike(pu) {
		ibanVal := any(nil)
		if req.IBAN != "" {
			ibanVal = req.IBAN
		}
		h.db.ExecContext(r.Context(),
			`UPDATE members SET
				join_date=?, iban=COALESCE(?, iban), account_holder=?,
				dsgvo_verarbeitung=?, dsgvo_verarbeitung_date=?,
				dsgvo_weitergabe=?, dsgvo_weitergabe_date=?,
				sepa_mandat=?, sepa_mandat_date=?,
				beitragsfrei=?
			WHERE id=?`,
			nullableString(req.JoinDate), ibanVal, nullableString(req.AccountHolder),
			boolToInt(req.DsgvoVerarbeitung), nullableString(req.DsgvoVerarbeitungDate),
			boolToInt(req.DsgvoWeitergabe), nullableString(req.DsgvoWeitergabeDate),
			boolToInt(req.SepaMandat), nullableString(req.SepaMandatDate),
			boolToInt(req.Beitragsfrei),
			id)
	}

	if idInt, err2 := strconv.Atoi(id); err2 == nil {
		h.writeClubFunctions(r.Context(), idInt, req.ClubFunctions)
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
	valid := map[string]bool{
		"aktiv": true, "verletzt": true, "pausiert": true,
		"ausgetreten": true, "passiv": true, "honorar": true, "anwaerter": true,
	}
	if !valid[req.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}
	h.db.ExecContext(r.Context(), `UPDATE members SET status=?, updated_at=? WHERE id=?`, req.Status, time.Now(), id)
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/members/{id}/bank-details
// Aktualisiert ausschließlich die bankrelevanten Felder (Feld-Whitelist),
// damit der Kassierer korrigieren kann, ohne Stammdaten/Status zu verändern.
func (h *Handler) UpdateBankdaten(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		IBAN           string `json:"iban"`
		SepaMandat     bool   `json:"sepa_mandat"`
		SepaMandatDate string `json:"sepa_mandat_date"`
		AccountHolder  string `json:"account_holder"`
		Street         string `json:"street"`
		Zip            string `json:"zip"`
		City           string `json:"city"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ungültiger Body", http.StatusBadRequest)
		return
	}
	req.IBAN = sepa.NormalizeIBAN(req.IBAN)
	if req.IBAN != "" && !sepa.IsValidIBAN(req.IBAN) {
		http.Error(w, "ungültige IBAN", http.StatusBadRequest)
		return
	}
	mandat := 0
	if req.SepaMandat {
		mandat = 1
	}
	res, err := h.db.ExecContext(r.Context(),
		`UPDATE members SET iban=?, sepa_mandat=?, sepa_mandat_date=?, account_holder=?, street=?, zip=?, city=?, updated_at=?
		 WHERE id=?`,
		nullStr(req.IBAN), mandat, nullStr(req.SepaMandatDate), nullStr(req.AccountHolder),
		nullStr(req.Street), nullStr(req.Zip), nullStr(req.City), time.Now(), id)
	if err != nil {
		http.Error(w, "DB-Fehler", http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "Mitglied nicht gefunden", http.StatusNotFound)
		return
	}
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// POST /api/members/{id}/proxy-account
func (h *Handler) CreateProxyAccount(w http.ResponseWriter, r *http.Request) {
	memberID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Email *string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var existingUserID sql.NullInt64
	var firstName, lastName string
	err = h.db.QueryRowContext(r.Context(),
		`SELECT user_id, first_name, last_name FROM members WHERE id = ?`, memberID,
	).Scan(&existingUserID, &firstName, &lastName)
	if err != nil {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}
	if existingUserID.Valid {
		http.Error(w, "member already has an account", http.StatusConflict)
		return
	}

	var emailVal interface{}
	if req.Email != nil && *req.Email != "" {
		emailVal = *req.Email
	}

	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO users (email, password, first_name, last_name, can_login) VALUES (?,?,?,?,0)`,
		emailVal, "", firstName, lastName,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	newUserID, _ := res.LastInsertId()
	h.db.ExecContext(r.Context(),
		`UPDATE members SET user_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		newUserID, memberID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"user_id": newUserID})
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
	h.hub.Broadcast("members")
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
	h.hub.Broadcast("members")
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
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/members/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT m.member_number, m.first_name, m.last_name, COALESCE(m.date_of_birth,''),
		        m.gender, COALESCE(m.pass_number,''), m.jersey_number,
		        COALESCE(m.position,''), m.status, COALESCE(m.home_club,''),
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
		"Passnummer", "Trikotnummer", "Position", "Status", "Stammverein",
		"Benutzer_Email", "Erziehungsberechtigter1_Email", "Erziehungsberechtigter2_Email",
	})
	for rows.Next() {
		var memberNum, position, userEmail, parent1, parent2 sql.NullString
		var firstName, lastName, dob, passNum, gender, status, homeClub string
		var jerseyNum sql.NullInt64
		rows.Scan(&memberNum, &firstName, &lastName, &dob, &gender, &passNum, &jerseyNum,
			&position, &status, &homeClub, &userEmail, &parent1, &parent2)
		jerseyStr := ""
		if jerseyNum.Valid {
			jerseyStr = fmt.Sprintf("%d", jerseyNum.Int64)
		}
		cw.Write([]string{
			memberNum.String, firstName, lastName, dob, gender,
			passNum, jerseyStr, position.String, status, homeClub,
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
	PhonesVisible   bool `json:"phones_visible"`
	AddressVisible  bool `json:"address_visible"`
	PhotoVisible    bool `json:"photo_visible"`
	EmailVisible    bool `json:"email_visible"`
	WhatsAppVisible bool `json:"whatsapp_visible"`
}

type ProfileResponse struct {
	OwnMember        *Member         `json:"own_member,omitempty"`
	Children         []Member        `json:"children"`
	Parents          []ProfileParent `json:"parents"`
	Street           string          `json:"street,omitempty"`
	Zip              string          `json:"zip,omitempty"`
	City             string          `json:"city,omitempty"`
	PhotoURL         string          `json:"photo_url,omitempty"`
	RecoveryEmail    string          `json:"recovery_email,omitempty"`
	Phones           []UserPhone     `json:"phones"`
	Visibility       UserVisibility  `json:"visibility"`
	DutyReminderDays *int            `json:"duty_reminder_days"`
	MapsProvider     string          `json:"maps_provider"`
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
			        m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id,
			        COALESCE((SELECT GROUP_CONCAT(mcf.function,',') FROM member_club_functions mcf WHERE mcf.member_id=m.id),'')
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

	// Contact data: address, photo_url, phones, visibility, reminder preference, maps provider
	var street, zip, city, photoPath sql.NullString
	var reminderDays sql.NullInt64
	var mapsProvider string
	var recoveryEmail sql.NullString
	h.db.QueryRowContext(r.Context(),
		`SELECT COALESCE(street,''), COALESCE(zip,''), COALESCE(city,''), COALESCE(photo_path,''), duty_reminder_days, maps_provider, COALESCE(recovery_email,'') FROM users WHERE id=?`,
		claims.UserID).Scan(&street, &zip, &city, &photoPath, &reminderDays, &mapsProvider, &recoveryEmail)
	resp.Street = street.String
	resp.Zip = zip.String
	resp.City = city.String
	resp.RecoveryEmail = recoveryEmail.String
	if photoPath.String != "" {
		resp.PhotoURL = "/api/uploads/" + photoPath.String
	}
	if reminderDays.Valid {
		v := int(reminderDays.Int64)
		resp.DutyReminderDays = &v
	}
	if mapsProvider == "" {
		mapsProvider = "auto"
	}
	resp.MapsProvider = mapsProvider

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
	var pv, av, phv, ev, wv int
	h.db.QueryRowContext(r.Context(),
		`SELECT phones_visible, address_visible, photo_visible, COALESCE(email_visible,0), COALESCE(whatsapp_visible,0) FROM user_visibility WHERE user_id=?`,
		claims.UserID).Scan(&pv, &av, &phv, &ev, &wv)
	vis.PhonesVisible = pv == 1
	vis.AddressVisible = av == 1
	vis.PhotoVisible = phv == 1
	vis.EmailVisible = ev == 1
	vis.WhatsAppVisible = wv == 1
	resp.Visibility = vis

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PUT /api/profile/me — update profile fields (name + address + maps provider)
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		Street       string `json:"street"`
		Zip          string `json:"zip"`
		City         string `json:"city"`
		MapsProvider string `json:"maps_provider"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.MapsProvider != "" {
		switch req.MapsProvider {
		case "auto", "google", "apple":
		default:
			http.Error(w, "maps_provider must be auto, google, or apple", http.StatusBadRequest)
			return
		}
	}
	mapsProvider := req.MapsProvider
	if mapsProvider == "" {
		mapsProvider = "auto"
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET first_name=?, last_name=?, street=?, zip=?, city=?, maps_provider=?, updated_at=? WHERE id=?`,
		req.FirstName, req.LastName,
		nullableString(req.Street), nullableString(req.Zip), nullableString(req.City), mapsProvider, time.Now(), claims.UserID,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/profile/reminder-preference
func (h *Handler) UpdateAbsenceVisibility(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		Public bool `json:"public"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	val := 0
	if req.Public {
		val = 1
	}
	h.db.ExecContext(r.Context(),
		`UPDATE members SET absences_public = ? WHERE user_id = ?`, val, claims.UserID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateReminderPreference(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req struct {
		DutyReminderDays *int `json:"duty_reminder_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.DutyReminderDays != nil && *req.DutyReminderDays != 2 {
		http.Error(w, "duty_reminder_days must be 2 or null", http.StatusBadRequest)
		return
	}
	h.db.ExecContext(r.Context(),
		`UPDATE users SET duty_reminder_days=? WHERE id=?`,
		req.DutyReminderDays, claims.UserID)
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
	h.hub.Broadcast("members")
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
	h.hub.Broadcast("members")
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
	h.hub.Broadcast("members")
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
		`INSERT INTO user_visibility (user_id, phones_visible, address_visible, photo_visible, email_visible, whatsapp_visible)
		 VALUES (?,?,?,?,?,?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   phones_visible=excluded.phones_visible,
		   address_visible=excluded.address_visible,
		   photo_visible=excluded.photo_visible,
		   email_visible=excluded.email_visible,
		   whatsapp_visible=excluded.whatsapp_visible`,
		claims.UserID, boolToInt(req.PhonesVisible), boolToInt(req.AddressVisible), boolToInt(req.PhotoVisible), boolToInt(req.EmailVisible), boolToInt(req.WhatsAppVisible))
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
	if claims.IsParent {
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

// GET /api/family/proxy-accounts — returns proxy-account children linked to the logged-in user
func (h *Handler) GetFamilyProxyAccounts(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT u.id, m.id, u.first_name || ' ' || u.last_name
		 FROM family_links fl
		 JOIN members m ON m.id = fl.member_id
		 JOIN users u ON u.id = m.user_id
		 WHERE fl.parent_user_id = ? AND u.can_login = 0`,
		claims.UserID,
	)
	type proxyChild struct {
		UserID   int    `json:"user_id"`
		MemberID int    `json:"member_id"`
		Name     string `json:"name"`
	}
	result := []proxyChild{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c proxyChild
			rows.Scan(&c.UserID, &c.MemberID, &c.Name)
			result = append(result, c)
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
	h.hub.Broadcast("members")
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

// normalizeDate converts German DD.MM.YY / DD.MM.YYYY to ISO YYYY-MM-DD.
// Leaves already-ISO or unrecognized strings unchanged.
func normalizeDate(s string) string {
	return normalizeDateAt(s, time.Now().Year())
}

// normalizeDateAt wandelt deutsches DD.MM.YY / DD.MM.YYYY in ISO YYYY-MM-DD.
// Für 2-stellige Jahre wird das Jahrhundert so gewählt, dass das Datum NICHT in
// der Zukunft liegt — Geburts-/Beitrittsdaten sind nie zukünftig: 20YY, sofern
// das nicht nach currentYear läge, sonst 19YY. Damit wird z.B. "67" korrekt als
// 1967 (statt 2067) interpretiert; sonst verfehlt der Import-Abgleich ältere
// Mitglieder und füllt keine Felder. Bereits ISO- oder unbekannte Strings
// bleiben unverändert.
func normalizeDateAt(s string, currentYear int) string {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return s
	}
	day, month, year := parts[0], parts[1], parts[2]
	switch len(year) {
	case 2:
		y, err := strconv.Atoi(year)
		if err != nil {
			return s
		}
		if 2000+y > currentYear {
			year = strconv.Itoa(1900 + y)
		} else {
			year = strconv.Itoa(2000 + y)
		}
	case 4:
		// already full year
	default:
		return s
	}
	return year + "-" + month + "-" + day
}

// validateIBAN checks the MOD-97 checksum and length (22 chars for DE IBANs).
// Returns (true, "") on success or (false, reason) on failure.
func validateIBAN(s string) (bool, string) {
	s = strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	if len(s) < 4 {
		return false, "zu kurz"
	}
	if strings.HasPrefix(s, "DE") && len(s) != 22 {
		return false, fmt.Sprintf("DE-IBAN muss 22 Zeichen haben, hat %d", len(s))
	}
	rearranged := s[4:] + s[:4]
	var sb strings.Builder
	for _, c := range rearranged {
		switch {
		case c >= '0' && c <= '9':
			sb.WriteRune(c)
		case c >= 'A' && c <= 'Z':
			sb.WriteString(strconv.Itoa(int(c-'A') + 10))
		default:
			return false, fmt.Sprintf("ungültiges Zeichen: %q", c)
		}
	}
	n := new(big.Int)
	n.SetString(sb.String(), 10)
	if new(big.Int).Mod(n, big.NewInt(97)).Int64() != 1 {
		return false, "Prüfziffer falsch"
	}
	return true, ""
}

// ImportRow holds the result for a single CSV row.
type ImportRow struct {
	Line        int      `json:"line"`
	Status      string   `json:"status"` // created | updated | unchanged | error | not_found
	Name        string   `json:"name"`
	DOB         string   `json:"dob,omitempty"`
	Changes     []string `json:"changes,omitempty"`
	Message     string   `json:"message,omitempty"`
	IBANWarning string   `json:"iban_warning,omitempty"`
}

// ImportReport is the full response body for POST /api/members/import.
type ImportReport struct {
	Total     int         `json:"total"`
	Created   int         `json:"created"`
	Updated   int         `json:"updated"`
	Unchanged int         `json:"unchanged"`
	Errors    int         `json:"errors"`
	NotFound  int         `json:"not_found"`
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
	mode := r.FormValue("mode") // "append", "update", "enrich", or legacy "preview"
	if mode != "append" && mode != "update" && mode != "enrich" && mode != "preview" {
		mode = "append"
	}
	dryRun := mode == "preview" || r.FormValue("preview") == "1"
	if mode == "preview" {
		mode = "update" // legacy: preview mode uses update semantics
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
	// Canonical names for columns that external tools export differently.
	columnAliases := map[string]string{
		"Name":          "Nachname",
		"geboren am":    "Geburtsdatum",
		"Mitglied seit": "join_date",
	}
	colIdx := make(map[string]int, len(header))
	for i, name := range header {
		trimmed := strings.TrimSpace(name)
		colIdx[trimmed] = i
		if canonical, ok := columnAliases[trimmed]; ok {
			colIdx[canonical] = i
		}
	}
	col := func(row []string, name string) string {
		idx, ok := colIdx[name]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	normalizeGender := func(s string) string {
		switch s {
		case "m":
			return "m"
		case "w", "f":
			return "f"
		default:
			return "u"
		}
	}
	normalizeStatus := func(s string) string {
		switch s {
		case "aktiv", "verletzt", "pausiert", "ausgetreten", "passiv", "honorar", "anwaerter":
			return s
		case "gekündigt", "Vereinswechsel":
			return "ausgetreten"
		case "kein aktiver Sportler mehr":
			return "passiv"
		case "beitragsfrei":
			return "passiv"
		default:
			return "aktiv"
		}
	}
	normalizeSepa := func(s string) int {
		if strings.TrimSpace(s) == "vorliegend" {
			return 1
		}
		return 0
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
	seenAt := make(map[dupKey]int)       // key → first line number
	dupOf := make(map[int]int)           // later line → first line
	firstDupPartner := make(map[int]int) // first line → first later duplicate
	isDupLine := make(map[int]bool)
	for i, row := range allRows {
		lineNum := i + 2
		k := dupKey{
			first: strings.ToLower(col(row, "Vorname")),
			last:  strings.ToLower(col(row, "Nachname")),
			dob:   normalizeDate(col(row, "Geburtsdatum")),
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
		dob := normalizeDate(col(row, "Geburtsdatum"))

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

		// Enrich mode without DOB: check for ambiguous name matches before proceeding.
		if mode == "enrich" && dob == "" {
			var cnt int
			h.db.QueryRowContext(r.Context(),
				`SELECT COUNT(*) FROM members WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?)`,
				firstName, lastName).Scan(&cnt)
			if cnt >= 2 {
				report.Rows = append(report.Rows, ImportRow{
					Line:    lineNum,
					Status:  "error",
					Name:    displayName,
					Message: fmt.Sprintf("Mehrdeutig (%d Treffer) – Geburtsdatum in CSV fehlt", cnt),
				})
				report.Errors++
				continue
			}
		}

		// DB lookup by name (+ dob as tiebreaker when present)
		query := `SELECT id, member_number, COALESCE(date_of_birth,''),
		                 pass_number, jersey_number, position, status, gender, user_id, home_club,
		                 COALESCE(street,''), COALESCE(zip,''), COALESCE(city,''),
		                 COALESCE(join_date,''), COALESCE(iban,''), COALESCE(account_holder,''),
		                 COALESCE(sepa_mandat,0), COALESCE(beitragsfrei,0)
		          FROM members
		          WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?)`
		args := []interface{}{firstName, lastName}
		if dob != "" {
			// Standard: exakter Vergleich nur auf den Datumsanteil — date_of_birth
			// kann reines ISO-Datum ("2007-10-14") ODER ISO-Timestamp
			// ("2007-10-14T00:00:00Z") sein (SQLite-DATE-Gotcha).
			dobClause := ` AND substr(COALESCE(date_of_birth,''),1,10)=?`
			useDobArg := true

			// Fall B: Findet der exakte Abgleich nichts und ist beim Bestands-
			// mitglied gar kein Geburtsdatum gepflegt, matchen wir in Füll-Modi
			// ersatzweise über den Namen allein (Geburtsdatum wird per enrich
			// ergänzt) — aber NUR wenn genau ein gleichnamiges Mitglied ohne
			// Geburtsdatum existiert (Eindeutigkeits-Schutz), damit keine andere
			// gleichnamige Person befüllt wird.
			if mode == "enrich" || mode == "update" {
				var exactCnt int
				h.db.QueryRowContext(r.Context(),
					`SELECT COUNT(*) FROM members WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?) AND substr(COALESCE(date_of_birth,''),1,10)=?`,
					firstName, lastName, dob).Scan(&exactCnt)
				if exactCnt == 0 {
					var emptyCnt int
					h.db.QueryRowContext(r.Context(),
						`SELECT COUNT(*) FROM members WHERE lower(first_name)=lower(?) AND lower(last_name)=lower(?) AND COALESCE(date_of_birth,'')=''`,
						firstName, lastName).Scan(&emptyCnt)
					switch {
					case emptyCnt == 1:
						dobClause = ` AND COALESCE(date_of_birth,'')=''`
						useDobArg = false
					case emptyCnt >= 2:
						report.Rows = append(report.Rows, ImportRow{
							Line:    lineNum,
							Status:  "error",
							Name:    displayName,
							Message: fmt.Sprintf("Mehrdeutig (%d gleichnamige ohne Geburtsdatum) – bitte manuell zuordnen", emptyCnt),
						})
						report.Errors++
						continue
					}
				}
			}

			query += dobClause
			if useDobArg {
				args = append(args, dob)
			}
		}
		query += ` LIMIT 1`

		var (
			existingID                          int
			dbMemberNum, dbPassNum, dbPosition  sql.NullString
			dbDOB, dbGender, dbStatus           string
			dbJerseyNum                         sql.NullInt64
			dbUserID                            sql.NullInt64
			dbHomeClub                          sql.NullString
			dbStreet, dbZip, dbCity             string
			dbJoinDate, dbIBAN, dbAccountHolder string
			dbSepaMandat                        int
			dbBeitragsfrei                      int
		)
		scanErr := h.db.QueryRowContext(r.Context(), query, args...).
			Scan(&existingID, &dbMemberNum, &dbDOB, &dbPassNum, &dbJerseyNum, &dbPosition,
				&dbStatus, &dbGender, &dbUserID, &dbHomeClub,
				&dbStreet, &dbZip, &dbCity,
				&dbJoinDate, &dbIBAN, &dbAccountHolder, &dbSepaMandat, &dbBeitragsfrei)

		if scanErr == sql.ErrNoRows {
			if mode == "enrich" {
				report.Rows = append(report.Rows, ImportRow{
					Line:   lineNum,
					Status: "not_found",
					Name:   displayName,
					DOB:    dob,
				})
				report.NotFound++
				continue
			}
			// New member — insert
			csvStatusRaw := col(row, "Status")
			csvBeitragsfrei := strings.EqualFold(strings.TrimSpace(csvStatusRaw), "beitragsfrei")
			gender := normalizeGender(col(row, "Geschlecht"))
			status := normalizeStatus(csvStatusRaw)
			jerseyArg, _ := parseOptionalInt(col(row, "Trikotnummer"))
			joinDate := normalizeDate(col(row, "join_date"))

			var ibanWarn string
			var ibanArg interface{}
			if raw := strings.ToUpper(strings.ReplaceAll(col(row, "IBAN"), " ", "")); raw != "" {
				if ok, msg := validateIBAN(raw); ok {
					ibanArg = raw
				} else {
					ibanWarn = "IBAN nicht gespeichert: " + msg
				}
			}

			if !dryRun {
				_, insErr := h.db.ExecContext(r.Context(),
					`INSERT INTO members (member_number, first_name, last_name, date_of_birth,
					                      pass_number, jersey_number, position, status, gender, home_club,
					                      street, zip, city, join_date, iban, account_holder, sepa_mandat,
					                      beitragsfrei)
					 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
					nullableString(col(row, "Mitgliedsnummer")), firstName, lastName,
					nullableString(dob), nullableString(col(row, "Passnummer")),
					jerseyArg, nullableString(col(row, "Position")), status, gender,
					nullableString(col(row, "Stammverein")),
					nullableString(col(row, "Adresse")), nullableString(col(row, "PLZ")), nullableString(col(row, "Ort")),
					nullableString(joinDate), ibanArg, nullableString(col(row, "Kontoinhaber")),
					normalizeSepa(col(row, "SEPA Mandat")),
					boolToInt(csvBeitragsfrei))
				if insErr != nil {
					report.Rows = append(report.Rows, ImportRow{
						Line: lineNum, Status: "error", Name: displayName,
						Message: "Fehler beim Anlegen: " + insErr.Error(),
					})
					report.Errors++
					continue
				}
			}
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "created", Name: displayName, DOB: dob, IBANWarning: ibanWarn,
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

		// mode == "update", "enrich", or legacy preview: apply non-empty changed fields
		enrichOnly := mode == "enrich"
		var setClauses []string
		var setArgs []interface{}
		var changes []string

		addChange := func(csvVal, dbVal, label, column string) {
			if csvVal == "" || csvVal == dbVal {
				return
			}
			if enrichOnly && dbVal != "" {
				return // enrich: never overwrite existing values
			}
			setClauses = append(setClauses, column+"=?")
			setArgs = append(setArgs, csvVal)
			changes = append(changes, fmt.Sprintf("%s: %q → %q", label, dbVal, csvVal))
		}
		addNullableChange := func(csvVal string, dbVal sql.NullString, label, column string) {
			if csvVal == "" || csvVal == dbVal.String {
				return
			}
			if enrichOnly && dbVal.Valid && dbVal.String != "" {
				return // enrich: never overwrite existing values
			}
			setClauses = append(setClauses, column+"=?")
			setArgs = append(setArgs, csvVal)
			changes = append(changes, fmt.Sprintf("%s: %q → %q", label, dbVal.String, csvVal))
		}

		addNullableChange(col(row, "Mitgliedsnummer"), dbMemberNum, "Mitgliedsnummer", "member_number")
		addChange(dob, dbDOB, "Geburtsdatum", "date_of_birth")
		addChange(normalizeGender(col(row, "Geschlecht")), dbGender, "Geschlecht", "gender")
		addNullableChange(col(row, "Passnummer"), dbPassNum, "Passnummer", "pass_number")
		addNullableChange(col(row, "Position"), dbPosition, "Position", "position")
		addChange(normalizeStatus(col(row, "Status")), dbStatus, "Status", "status")
		addNullableChange(col(row, "Stammverein"), dbHomeClub, "Stammverein", "home_club")

		if jerseyRaw := col(row, "Trikotnummer"); jerseyRaw != "" {
			dbJerseyStr := ""
			if dbJerseyNum.Valid {
				dbJerseyStr = fmt.Sprintf("%d", dbJerseyNum.Int64)
			}
			if jerseyRaw != dbJerseyStr && (!enrichOnly || !dbJerseyNum.Valid) {
				n, _ := parseOptionalInt(jerseyRaw)
				setClauses = append(setClauses, "jersey_number=?")
				setArgs = append(setArgs, n)
				changes = append(changes, fmt.Sprintf("Trikotnummer: %q → %q", dbJerseyStr, jerseyRaw))
			}
		}

		// New fields: address, join_date, account_holder, sepa_mandat
		joinDate := normalizeDate(col(row, "join_date"))
		addChange(col(row, "Adresse"), dbStreet, "Adresse", "street")
		addChange(col(row, "PLZ"), dbZip, "PLZ", "zip")
		addChange(col(row, "Ort"), dbCity, "Ort", "city")
		addChange(joinDate, dbJoinDate, "Mitglied seit", "join_date")
		addChange(col(row, "Kontoinhaber"), dbAccountHolder, "Kontoinhaber", "account_holder")

		if sepaRaw := col(row, "SEPA Mandat"); sepaRaw != "" && !enrichOnly {
			sepaVal := normalizeSepa(sepaRaw)
			if sepaVal != dbSepaMandat {
				setClauses = append(setClauses, "sepa_mandat=?")
				setArgs = append(setArgs, sepaVal)
				changes = append(changes, fmt.Sprintf("SEPA Mandat: %d → %d", dbSepaMandat, sepaVal))
			}
		}

		// beitragsfrei aus CSV-Status ableiten
		if csvStatusRaw2 := col(row, "Status"); csvStatusRaw2 != "" && !enrichOnly {
			csvBeitragsfrei2 := boolToInt(strings.EqualFold(strings.TrimSpace(csvStatusRaw2), "beitragsfrei"))
			if csvBeitragsfrei2 != dbBeitragsfrei {
				setClauses = append(setClauses, "beitragsfrei=?")
				setArgs = append(setArgs, csvBeitragsfrei2)
				changes = append(changes, fmt.Sprintf("Beitragsfrei: %v → %v", dbBeitragsfrei == 1, csvBeitragsfrei2 == 1))
			}
		}

		// IBAN with MOD-97 validation; in enrich mode only fill if DB field is empty.
		var ibanWarn string
		if raw := strings.ToUpper(strings.ReplaceAll(col(row, "IBAN"), " ", "")); raw != "" {
			if ok, msg := validateIBAN(raw); ok {
				if raw != dbIBAN && (!enrichOnly || dbIBAN == "") {
					setClauses = append(setClauses, "iban=?")
					setArgs = append(setArgs, raw)
					changes = append(changes, fmt.Sprintf("IBAN: %q → %q", dbIBAN, raw))
				}
			} else if !enrichOnly || dbIBAN == "" {
				ibanWarn = "IBAN nicht gespeichert: " + msg
			}
		}

		if len(setClauses) > 0 && !dryRun {
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

		if len(changes) > 0 || ibanWarn != "" {
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "updated", Name: displayName, Changes: changes, IBANWarning: ibanWarn,
			})
			report.Updated++
		} else {
			report.Rows = append(report.Rows, ImportRow{
				Line: lineNum, Status: "unchanged", Name: displayName,
			})
			report.Unchanged++
		}
	}

	report.Total = report.Created + report.Updated + report.Unchanged + report.Errors + report.NotFound
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
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

func (h *Handler) writeClubFunctions(ctx context.Context, memberID int, functions []string) {
	h.db.ExecContext(ctx, `DELETE FROM member_club_functions WHERE member_id = ?`, memberID)
	for _, f := range functions {
		h.db.ExecContext(ctx, `INSERT OR IGNORE INTO member_club_functions (member_id, function) VALUES (?,?)`, memberID, f)
	}
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

// GET /api/users/:id/contact
func (h *Handler) GetContact(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	type phoneEntry struct {
		Label  string `json:"label"`
		Number string `json:"number"`
	}
	type contactResponse struct {
		Name            string       `json:"name"`
		PhotoURL        *string      `json:"photo_url,omitempty"`
		Phones          []phoneEntry `json:"phones,omitempty"`
		Address         *string      `json:"address,omitempty"`
		Email           *string      `json:"email,omitempty"`
		WhatsAppVisible bool         `json:"whatsapp_visible"`
	}

	var resp contactResponse
	var photoURL, phonesJSON, address, email sql.NullString
	var wv int
	err := h.db.QueryRowContext(r.Context(), `
		SELECT u.first_name || ' ' || u.last_name,
		       CASE WHEN COALESCE(uv.photo_visible,0)=1 AND COALESCE(u.photo_path,'') != ''
		            THEN '/api/uploads/' || u.photo_path END,
		       CASE WHEN COALESCE(uv.phones_visible,0)=1 THEN
		           (SELECT json_group_array(json_object('label', p.label, 'number', p.number))
		            FROM user_phones p WHERE p.user_id=u.id)
		       END,
		       CASE WHEN COALESCE(uv.address_visible,0)=1 AND COALESCE(u.street,'') != '' THEN
		           u.street || COALESCE(', ' || NULLIF(TRIM(COALESCE(u.zip,'') || ' ' || COALESCE(u.city,'')), ''), '')
		       END,
		       CASE WHEN COALESCE(uv.email_visible,0)=1 THEN u.email END,
		       COALESCE(uv.whatsapp_visible,0)
		FROM users u
		LEFT JOIN user_visibility uv ON uv.user_id = u.id
		WHERE u.id = ?`, id).Scan(&resp.Name, &photoURL, &phonesJSON, &address, &email, &wv)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if photoURL.Valid && photoURL.String != "" {
		resp.PhotoURL = &photoURL.String
	}
	if phonesJSON.Valid && phonesJSON.String != "" && phonesJSON.String != "[]" {
		json.Unmarshal([]byte(phonesJSON.String), &resp.Phones)
	}
	if address.Valid && address.String != "" {
		resp.Address = &address.String
	}
	if email.Valid && email.String != "" {
		resp.Email = &email.String
	}
	resp.WhatsAppVisible = wv == 1
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// PUT /api/members/{id}/cross-team-visible
//
// Persönliche Privacy-Präferenz pro Member — kein Draft-Workflow. Zulässig für:
// das eigene Member (m.user_id = caller), Eltern eines Kind-Members (via
// family_links), sowie admin/vorstand.
func (h *Handler) UpdateCrossTeamVisible(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	memberID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var ownerUserID sql.NullInt64
	if err := h.db.QueryRowContext(r.Context(), `SELECT user_id FROM members WHERE id=?`, memberID).Scan(&ownerUserID); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	allowed := claims.Role == "admin" || claims.HasFunction("vorstand")
	if !allowed && ownerUserID.Valid && int(ownerUserID.Int64) == claims.UserID {
		allowed = true
	}
	if !allowed && h.isParentOf(r.Context(), claims.UserID, memberID) {
		allowed = true
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		CrossTeamVisible bool `json:"cross_team_visible"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE members SET cross_team_visible=?, updated_at=? WHERE id=?`,
		boolToInt(req.CrossTeamVisible), time.Now(), memberID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// isParentOf returns true if parentUserID has a family_links entry for memberID.
func (h *Handler) isParentOf(ctx context.Context, parentUserID, memberID int) bool {
	var count int
	h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?`,
		parentUserID, memberID).Scan(&count)
	return count > 0
}

// GET /api/profile/kind/:memberId
func (h *Handler) GetChildProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.isParentOf(r.Context(), claims.UserID, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	m, err := h.getMember(memberID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	type parentEntry struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	pRows, _ := h.db.QueryContext(r.Context(),
		`SELECT u.id, u.first_name || ' ' || u.last_name, u.email FROM users u JOIN family_links fl ON fl.parent_user_id=u.id WHERE fl.member_id=?`,
		memberID)
	parents := []parentEntry{}
	if pRows != nil {
		defer pRows.Close()
		for pRows.Next() {
			var p parentEntry
			pRows.Scan(&p.ID, &p.Name, &p.Email)
			parents = append(parents, p)
		}
	}

	type phoneEntry struct {
		ID        int    `json:"id"`
		Label     string `json:"label"`
		Number    string `json:"number"`
		SortOrder int    `json:"sort_order"`
	}

	// user_contact: wenn Kind User-Account hat, User-Strang-Daten laden
	type visibilityEntry struct {
		PhonesVisible   bool `json:"phones_visible"`
		AddressVisible  bool `json:"address_visible"`
		PhotoVisible    bool `json:"photo_visible"`
		EmailVisible    bool `json:"email_visible"`
		WhatsAppVisible bool `json:"whatsapp_visible"`
	}
	type userContactEntry struct {
		FirstName     string          `json:"first_name"`
		LastName      string          `json:"last_name"`
		Street        string          `json:"street"`
		Zip           string          `json:"zip"`
		City          string          `json:"city"`
		RecoveryEmail string          `json:"recovery_email"`
		Phones        []phoneEntry    `json:"phones"`
		Visibility    visibilityEntry `json:"visibility"`
	}

	var userContact *userContactEntry
	if m.UserID != nil {
		uc := userContactEntry{Phones: []phoneEntry{}}
		var street, zip, city, recoveryEmail sql.NullString
		h.db.QueryRowContext(r.Context(),
			`SELECT first_name, last_name, COALESCE(street,''), COALESCE(zip,''), COALESCE(city,''), COALESCE(recovery_email,'') FROM users WHERE id=?`,
			*m.UserID).Scan(&uc.FirstName, &uc.LastName, &street, &zip, &city, &recoveryEmail)
		uc.RecoveryEmail = recoveryEmail.String
		uc.Street = street.String
		uc.Zip = zip.String
		uc.City = city.String

		upRows, _ := h.db.QueryContext(r.Context(),
			`SELECT id, label, number, sort_order FROM user_phones WHERE user_id=? ORDER BY sort_order, id`,
			*m.UserID)
		if upRows != nil {
			defer upRows.Close()
			for upRows.Next() {
				var p phoneEntry
				upRows.Scan(&p.ID, &p.Label, &p.Number, &p.SortOrder)
				uc.Phones = append(uc.Phones, p)
			}
		}

		var pv, av, phv, ev, wv int
		h.db.QueryRowContext(r.Context(),
			`SELECT COALESCE(phones_visible,0), COALESCE(address_visible,0), COALESCE(photo_visible,0), COALESCE(email_visible,0), COALESCE(whatsapp_visible,0) FROM user_visibility WHERE user_id=?`,
			*m.UserID).Scan(&pv, &av, &phv, &ev, &wv)
		uc.Visibility = visibilityEntry{
			PhonesVisible:   pv == 1,
			AddressVisible:  av == 1,
			PhotoVisible:    phv == 1,
			EmailVisible:    ev == 1,
			WhatsAppVisible: wv == 1,
		}
		userContact = &uc
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"member": m, "parents": parents, "phones": nil, "user_contact": userContact})
}

// PUT /api/profile/kind/:memberId/account — aktualisiert users-Datensatz des Kindes (User-Strang)
func (h *Handler) UpdateChildAccount(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.isParentOf(r.Context(), claims.UserID, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var childUserID sql.NullInt64
	if err := h.db.QueryRowContext(r.Context(), `SELECT user_id FROM members WHERE id=?`, memberID).Scan(&childUserID); err != nil || !childUserID.Valid {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Street    string `json:"street"`
		Zip       string `json:"zip"`
		City      string `json:"city"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if _, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET first_name=?, last_name=?, street=?, zip=?, city=?, updated_at=? WHERE id=?`,
		req.FirstName, req.LastName,
		nullableString(req.Street), nullableString(req.Zip), nullableString(req.City),
		time.Now(), childUserID.Int64,
	); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/profile/kind/:memberId/member
func (h *Handler) UpdateChildMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.isParentOf(r.Context(), claims.UserID, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		DateOfBirth  string `json:"date_of_birth"`
		JerseyNumber *int   `json:"jersey_number"`
		Position     string `json:"position"`
		Street       string `json:"street"`
		Zip          string `json:"zip"`
		City         string `json:"city"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_, err = h.db.ExecContext(r.Context(),
		`UPDATE members SET first_name=?, last_name=?, date_of_birth=?, jersey_number=?, position=?, street=?, zip=?, city=?, updated_at=? WHERE id=?`,
		req.FirstName, req.LastName, nullableString(req.DateOfBirth),
		req.JerseyNumber, nullableString(req.Position),
		nullableString(req.Street), nullableString(req.Zip), nullableString(req.City),
		time.Now(), memberID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/profile/kind/:memberId/phones
func (h *Handler) AddChildPhone(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.isParentOf(r.Context(), claims.UserID, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var childUserID sql.NullInt64
	if err := h.db.QueryRowContext(r.Context(), `SELECT user_id FROM members WHERE id=?`, memberID).Scan(&childUserID); err != nil || !childUserID.Valid {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Label     string `json:"label"`
		Number    string `json:"number"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := h.db.ExecContext(r.Context(),
		`INSERT INTO user_phones (user_id, label, number, sort_order) VALUES (?,?,?,?)`,
		childUserID.Int64, req.Label, req.Number, req.SortOrder)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

// DELETE /api/profile/kind/:memberId/phones/:phoneId
func (h *Handler) DeleteChildPhone(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.isParentOf(r.Context(), claims.UserID, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var childUserID sql.NullInt64
	if err := h.db.QueryRowContext(r.Context(), `SELECT user_id FROM members WHERE id=?`, memberID).Scan(&childUserID); err != nil || !childUserID.Valid {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	phoneID, err := strconv.Atoi(r.PathValue("phoneId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h.db.ExecContext(r.Context(),
		`DELETE FROM user_phones WHERE id=? AND user_id=?`, phoneID, childUserID.Int64)
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/profile/kind/:memberId/visibility
func (h *Handler) UpdateChildVisibility(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.isParentOf(r.Context(), claims.UserID, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var childUserID sql.NullInt64
	if err := h.db.QueryRowContext(r.Context(), `SELECT user_id FROM members WHERE id=?`, memberID).Scan(&childUserID); err != nil || !childUserID.Valid {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		PhonesVisible   bool `json:"phones_visible"`
		AddressVisible  bool `json:"address_visible"`
		PhotoVisible    bool `json:"photo_visible"`
		EmailVisible    bool `json:"email_visible"`
		WhatsAppVisible bool `json:"whatsapp_visible"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	h.db.ExecContext(r.Context(),
		`INSERT INTO user_visibility (user_id, phones_visible, address_visible, photo_visible, email_visible, whatsapp_visible)
		 VALUES (?,?,?,?,?,?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   phones_visible=excluded.phones_visible,
		   address_visible=excluded.address_visible,
		   photo_visible=excluded.photo_visible,
		   email_visible=excluded.email_visible,
		   whatsapp_visible=excluded.whatsapp_visible`,
		childUserID.Int64,
		boolToInt(req.PhonesVisible), boolToInt(req.AddressVisible), boolToInt(req.PhotoVisible), boolToInt(req.EmailVisible), boolToInt(req.WhatsAppVisible))
	w.WriteHeader(http.StatusNoContent)
}

// PUT /api/profile/kind/:memberId/bank
func (h *Handler) UpdateChildBank(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	memberID, err := strconv.Atoi(r.PathValue("memberId"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.isParentOf(r.Context(), claims.UserID, memberID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		IBAN          string `json:"iban"`
		AccountHolder string `json:"account_holder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	_, err = h.db.ExecContext(r.Context(),
		`UPDATE members SET iban=?, account_holder=?, updated_at=? WHERE id=?`,
		nullableString(req.IBAN), nullableString(req.AccountHolder), time.Now(), memberID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.hub.Broadcast("members")
	w.WriteHeader(http.StatusNoContent)
}
