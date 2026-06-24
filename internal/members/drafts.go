package members

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/crypto"
)

type ChangeRequest struct {
	FieldName string          `json:"field_name"`
	NewValue  json.RawMessage `json:"new_value"`
}

type ChangeDraft struct {
	ID              int             `json:"id"`
	MemberID        int             `json:"member_id"`
	FieldName       string          `json:"field_name"`
	OldValue        json.RawMessage `json:"old_value"`
	NewValue        json.RawMessage `json:"new_value"`
	CreatedAt       time.Time       `json:"created_at"`
	CreatedByUserID sql.NullInt64   `json:"created_by_user_id,omitempty"`
}

// GetChangeDrafts retrieves all change drafts for a member
func (h *Handler) GetChangeDrafts(memberID int) ([]ChangeDraft, error) {
	rows, err := h.db.Query(`
		SELECT id, member_id, field_name, old_value, new_value, created_at, created_by_user_id
		FROM member_change_drafts
		WHERE member_id = ?
		ORDER BY created_at DESC
	`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drafts []ChangeDraft
	for rows.Next() {
		var d ChangeDraft
		if err := rows.Scan(&d.ID, &d.MemberID, &d.FieldName, &d.OldValue, &d.NewValue, &d.CreatedAt, &d.CreatedByUserID); err != nil {
			return nil, err
		}
		drafts = append(drafts, d)
	}
	return drafts, rows.Err()
}

// redactBankDrafts bereitet bankdaten-Entwürfe für die Anzeige auf: für
// Berechtigte werden new_value (ganzer "v1:"-String) und old_value (pro Feld
// verschlüsseltes JSON-Objekt) entschlüsselt; sonst auf null geschwärzt. Beides
// ist nötig, damit der verschlüsselte new_value die JSON-Antwort nicht zerstört.
func redactBankDrafts(drafts []ChangeDraft, reveal bool) {
	for i := range drafts {
		if drafts[i].FieldName != "bankdaten" {
			continue
		}
		if !reveal {
			drafts[i].NewValue = json.RawMessage("null")
			drafts[i].OldValue = json.RawMessage("null")
			continue
		}
		drafts[i].NewValue = decryptWholeBankValue(drafts[i].NewValue)
		drafts[i].OldValue = decryptBankObjectFields(drafts[i].OldValue)
	}
}

// decryptWholeBankValue entschlüsselt einen als ganzen String verschlüsselten
// new_value zurück zum ursprünglichen JSON. Schlägt das fehl, wird null geliefert.
func decryptWholeBankValue(raw json.RawMessage) json.RawMessage {
	pt, err := crypto.Decrypt(string(raw))
	if err != nil || !json.Valid([]byte(pt)) {
		return json.RawMessage("null")
	}
	return json.RawMessage(pt)
}

// decryptBankObjectFields entschlüsselt die String-Werte eines JSON-Objekts
// (old_value von bankdaten). Werte ohne "v1:"-Prefix bleiben unverändert.
func decryptBankObjectFields(raw json.RawMessage) json.RawMessage {
	var obj map[string]string
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw
	}
	for k, v := range obj {
		if pt, err := crypto.Decrypt(v); err == nil {
			obj[k] = pt
		}
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return raw
	}
	return out
}

// getMember retrieves a member by ID from the database
func (h *Handler) getMember(memberID int) (*Member, error) {
	row := h.db.QueryRow(`
		SELECT m.id, m.first_name, m.last_name,
		       COALESCE(m.date_of_birth,''), COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
		       m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id,
		       COALESCE((SELECT GROUP_CONCAT(mcf.function,',') FROM member_club_functions mcf WHERE mcf.member_id=m.id),''),
		       m.street, m.zip, m.city, m.join_date, m.iban, m.account_holder,
		       m.photo_visible, m.photo_path,
		       COALESCE(m.phones_visible,0), COALESCE(m.address_visible,0), COALESCE(m.email_visible,0),
		       COALESCE(m.absences_public,0),
		       COALESCE(m.cross_team_visible,0),
		       m.dsgvo_verarbeitung, m.dsgvo_verarbeitung_date,
		       m.dsgvo_weitergabe, m.dsgvo_weitergabe_date
		FROM members m
		WHERE m.id=?`, memberID)

	var m Member
	var jerseyNum, userID sql.NullInt64
	var clubFunctionsStr string
	var street, zip, city, joinDate, iban, accountHolder, photoPath sql.NullString
	var photoVisible, phonesVisible, addressVisible, emailVisible int64
	var crossTeamVisible, dsgvoVerarbeitung, dsgvoWeitergabe int64
	var dsgvoVerarbDate, dsgvoWeiterDate sql.NullString

	err := row.Scan(
		&m.ID, &m.FirstName, &m.LastName, &m.DateOfBirth,
		&m.MemberNumber, &m.PassNumber,
		&jerseyNum, &m.Position, &m.Gender, &m.Status, &userID, &clubFunctionsStr,
		&street, &zip, &city, &joinDate, &iban, &accountHolder,
		&photoVisible, &photoPath,
		&phonesVisible, &addressVisible, &emailVisible,
		&m.AbsencesPublic,
		&crossTeamVisible,
		&dsgvoVerarbeitung, &dsgvoVerarbDate,
		&dsgvoWeitergabe, &dsgvoWeiterDate,
	)
	if err != nil {
		return nil, err
	}

	if jerseyNum.Valid {
		n := int(jerseyNum.Int64)
		m.JerseyNumber = &n
	}
	if userID.Valid {
		uid := int(userID.Int64)
		m.UserID = &uid
	}
	m.ClubFunctions = parseFunctions(clubFunctionsStr)
	if street.Valid {
		m.Street = &street.String
	}
	if zip.Valid {
		m.Zip = &zip.String
	}
	if city.Valid {
		m.City = &city.String
	}
	if joinDate.Valid {
		m.JoinDate = &joinDate.String
	}
	if iban.Valid {
		m.IBAN = &iban.String
	}
	if accountHolder.Valid {
		m.AccountHolder = &accountHolder.String
	}
	m.PhotoVisible = photoVisible != 0
	if photoPath.Valid && photoPath.String != "" {
		u := "/api/uploads/" + photoPath.String
		m.PhotoURL = &u
	}
	m.PhonesVisible = phonesVisible != 0
	m.AddressVisible = addressVisible != 0
	m.EmailVisible = emailVisible != 0
	m.CrossTeamVisible = crossTeamVisible != 0
	m.DsgvoVerarbeitung = dsgvoVerarbeitung != 0
	if dsgvoVerarbDate.Valid {
		m.DsgvoVerarbeitungDate = &dsgvoVerarbDate.String
	}
	m.DsgvoWeitergabe = dsgvoWeitergabe != 0
	if dsgvoWeiterDate.Valid {
		m.DsgvoWeitergabeDate = &dsgvoWeiterDate.String
	}

	return &m, nil
}

// CreateOrUpdateDraft creates or updates a change draft (UPSERT)
func (h *Handler) CreateOrUpdateDraft(memberID, userID int, req ChangeRequest) (*ChangeDraft, error) {
	// Get current member data to store as old_value
	member, err := h.getMember(memberID)
	if err != nil {
		return nil, err
	}

	oldValue, err := h.extractFieldValue(member, req.FieldName)
	if err != nil {
		return nil, err
	}

	// Bankdaten-Entwürfe werden at-rest verschlüsselt abgelegt (ganzer new_value
	// als ein "v1:"-String). old_value enthält die — nach Migration ebenfalls
	// verschlüsselten — Bestandswerte des Mitglieds, daher kein Klartext-Leak.
	newValue := req.NewValue
	if req.FieldName == "bankdaten" {
		enc, encErr := crypto.Encrypt(string(req.NewValue))
		if encErr != nil {
			return nil, encErr
		}
		newValue = json.RawMessage(enc)
	}

	// UPSERT: Create or replace draft for this field
	_, err = h.db.Exec(`
		INSERT INTO member_change_drafts (member_id, field_name, old_value, new_value, created_by_user_id, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(member_id, field_name) DO UPDATE SET
			new_value = excluded.new_value,
			created_at = CURRENT_TIMESTAMP
	`, memberID, req.FieldName, oldValue, newValue, userID)
	if err != nil {
		return nil, err
	}

	// Retrieve created/updated draft
	var draft ChangeDraft
	err = h.db.QueryRow(`
		SELECT id, member_id, field_name, old_value, new_value, created_at, created_by_user_id
		FROM member_change_drafts
		WHERE member_id = ? AND field_name = ?
	`, memberID, req.FieldName).Scan(
		&draft.ID, &draft.MemberID, &draft.FieldName, &draft.OldValue, &draft.NewValue, &draft.CreatedAt, &draft.CreatedByUserID,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return &draft, nil
}

// AcceptDraft merges draft into member record and deletes draft
func (h *Handler) AcceptDraft(draftID int) error {
	// Get draft
	var d ChangeDraft
	err := h.db.QueryRow(`
		SELECT id, member_id, field_name, new_value
		FROM member_change_drafts
		WHERE id = ?
	`, draftID).Scan(&d.ID, &d.MemberID, &d.FieldName, &d.NewValue)
	if err != nil {
		return err
	}

	// Update member record based on field_name
	if err := h.applyDraftToMember(d.MemberID, d.FieldName, d.NewValue); err != nil {
		return err
	}

	// Delete draft
	_, err = h.db.Exec(`DELETE FROM member_change_drafts WHERE id = ?`, draftID)
	return err
}

// RejectDraft deletes a draft and sends rejection email to user
func (h *Handler) RejectDraft(draftID int) error {
	// Get draft info
	var d ChangeDraft
	var memberID int
	err := h.db.QueryRow(`
		SELECT id, member_id, field_name
		FROM member_change_drafts
		WHERE id = ?
	`, draftID).Scan(&d.ID, &memberID, &d.FieldName)
	if err != nil {
		return err
	}

	// Delete draft
	if _, err := h.db.Exec(`DELETE FROM member_change_drafts WHERE id = ?`, draftID); err != nil {
		return err
	}

	// TODO: Send rejection email to member's user
	// This will be handled in the HTTP handler after mailer integration

	return nil
}

func (h *Handler) extractFieldValue(m *Member, fieldName string) (json.RawMessage, error) {
	switch fieldName {
	case "name":
		return json.Marshal(map[string]string{
			"first_name": m.FirstName,
			"last_name":  m.LastName,
		})
	case "address":
		addr := map[string]interface{}{}
		if m.Street != nil {
			addr["street"] = *m.Street
		}
		if m.Zip != nil {
			addr["zip"] = *m.Zip
		}
		if m.City != nil {
			addr["city"] = *m.City
		}
		return json.Marshal(addr)
	case "photo_url":
		return json.Marshal(m.PhotoURL)
	case "iban":
		return json.Marshal(m.IBAN)
	case "account_holder":
		return json.Marshal(m.AccountHolder)
	case "dsgvo":
		return json.Marshal(map[string]bool{
			"verarbeitung": m.DsgvoVerarbeitung,
			"weitergabe":   m.DsgvoWeitergabe,
		})
	case "sepa_mandat":
		return json.Marshal(m.SepaMandat)
	case "profil":
		p := map[string]interface{}{
			"first_name": m.FirstName,
			"last_name":  m.LastName,
		}
		if m.Street != nil {
			p["street"] = *m.Street
		}
		if m.Zip != nil {
			p["zip"] = *m.Zip
		}
		if m.City != nil {
			p["city"] = *m.City
		}
		return json.Marshal(p)
	case "bankdaten":
		b := map[string]interface{}{}
		if m.IBAN != nil {
			b["iban"] = *m.IBAN
		}
		if m.AccountHolder != nil {
			b["account_holder"] = *m.AccountHolder
		}
		return json.Marshal(b)
	default:
		return json.Marshal(nil)
	}
}

func (h *Handler) applyDraftToMember(memberID int, fieldName string, newValue json.RawMessage) error {
	switch fieldName {
	case "name":
		var data struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		}
		if err := json.Unmarshal(newValue, &data); err != nil {
			return err
		}
		_, err := h.db.Exec(`UPDATE members SET first_name = ?, last_name = ? WHERE id = ?`,
			data.FirstName, data.LastName, memberID)
		return err

	case "address":
		var data struct {
			Street string `json:"street"`
			Zip    string `json:"zip"`
			City   string `json:"city"`
		}
		if err := json.Unmarshal(newValue, &data); err != nil {
			return err
		}
		_, err := h.db.Exec(`UPDATE members SET street = ?, zip = ?, city = ? WHERE id = ?`,
			data.Street, data.Zip, data.City, memberID)
		return err

	case "photo_url":
		var photoURL string
		if err := json.Unmarshal(newValue, &photoURL); err != nil {
			return err
		}
		_, err := h.db.Exec(`UPDATE members SET photo_url = ? WHERE id = ?`, photoURL, memberID)
		return err

	case "iban":
		var iban string
		if err := json.Unmarshal(newValue, &iban); err != nil {
			return err
		}
		enc, err := encBankField(iban)
		if err != nil {
			return err
		}
		_, err = h.db.Exec(`UPDATE members SET iban = ? WHERE id = ?`, enc, memberID)
		return err

	case "account_holder":
		var val string
		if err := json.Unmarshal(newValue, &val); err != nil {
			return err
		}
		enc, err := encBankField(val)
		if err != nil {
			return err
		}
		_, err = h.db.Exec(`UPDATE members SET account_holder = ? WHERE id = ?`, enc, memberID)
		return err

	case "dsgvo":
		var data struct {
			Verarbeitung bool `json:"verarbeitung"`
			Weitergabe   bool `json:"weitergabe"`
		}
		if err := json.Unmarshal(newValue, &data); err != nil {
			return err
		}
		_, err := h.db.Exec(
			`UPDATE members SET dsgvo_verarbeitung = ?, dsgvo_weitergabe = ? WHERE id = ?`,
			boolToInt(data.Verarbeitung), boolToInt(data.Weitergabe), memberID,
		)
		return err

	case "sepa_mandat":
		var val bool
		if err := json.Unmarshal(newValue, &val); err != nil {
			return err
		}
		_, err := h.db.Exec(`UPDATE members SET sepa_mandat = ? WHERE id = ?`, boolToInt(val), memberID)
		return err

	case "profil":
		var data struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Street    string `json:"street"`
			Zip       string `json:"zip"`
			City      string `json:"city"`
		}
		if err := json.Unmarshal(newValue, &data); err != nil {
			return err
		}
		_, err := h.db.Exec(
			`UPDATE members SET first_name=?, last_name=?, street=?, zip=?, city=? WHERE id=?`,
			data.FirstName, data.LastName, data.Street, data.Zip, data.City, memberID)
		return err

	case "bankdaten":
		// new_value liegt als ganzer "v1:"-String verschlüsselt vor (siehe
		// CreateOrUpdateDraft) — erst entschlüsseln, dann das JSON parsen.
		plain, err := crypto.Decrypt(string(newValue))
		if err != nil {
			return err
		}
		var data struct {
			IBAN          string `json:"iban"`
			AccountHolder string `json:"account_holder"`
		}
		if err := json.Unmarshal([]byte(plain), &data); err != nil {
			return err
		}
		encIBAN, err := encBankField(data.IBAN)
		if err != nil {
			return err
		}
		encHolder, err := encBankField(data.AccountHolder)
		if err != nil {
			return err
		}
		_, err = h.db.Exec(
			`UPDATE members SET iban=?, account_holder=? WHERE id=?`,
			encIBAN, encHolder, memberID)
		return err

	default:
		return nil
	}
}
