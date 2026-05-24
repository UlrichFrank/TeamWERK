package members

import (
	"database/sql"
	"encoding/json"
	"time"
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

// getMember retrieves a member by ID from the database
func (h *Handler) getMember(memberID int) (*Member, error) {
	row := h.db.QueryRow(`
		SELECT m.id, m.first_name, m.last_name,
		       COALESCE(m.member_number,''), COALESCE(m.pass_number,''),
		       m.jersey_number, COALESCE(m.position,''), COALESCE(m.gender,'u'), m.status, m.user_id, m.club_function,
		       m.join_date, m.photo_visible
		FROM members m
		WHERE m.id=?`, memberID)

	var m Member
	var jerseyNum, userID sql.NullInt64
	var clubFunc, joinDate sql.NullString
	var photoVisible int64

	err := row.Scan(
		&m.ID, &m.FirstName, &m.LastName,
		&m.MemberNumber, &m.PassNumber,
		&jerseyNum, &m.Position, &m.Gender, &m.Status, &userID, &clubFunc,
		&joinDate, &photoVisible,
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
	if clubFunc.Valid {
		m.ClubFunction = &clubFunc.String
	}
	if joinDate.Valid {
		m.JoinDate = &joinDate.String
	}
	m.PhotoVisible = photoVisible != 0

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

	// UPSERT: Create or replace draft for this field
	_, err = h.db.Exec(`
		INSERT INTO member_change_drafts (member_id, field_name, old_value, new_value, created_by_user_id, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(member_id, field_name) DO UPDATE SET
			new_value = excluded.new_value,
			created_at = CURRENT_TIMESTAMP
	`, memberID, req.FieldName, oldValue, req.NewValue, userID)
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
		return json.Marshal(map[string]interface{}{})
	case "photo_url":
		return json.Marshal(m.PhotoURL)
	case "iban":
		return json.Marshal(nil)
	case "dsgvo":
		return json.Marshal(map[string]bool{
			"verarbeitung": m.DsgvoVerarbeitung,
			"weitergabe":   m.DsgvoWeitergabe,
		})
	case "sepa_mandat":
		return json.Marshal(m.SepaMandat)
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
		_, err := h.db.Exec(`UPDATE members SET iban = ? WHERE id = ?`, iban, memberID)
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

	default:
		return nil
	}
}
