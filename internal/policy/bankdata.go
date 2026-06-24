package policy

import "database/sql"

// CanDecryptBankData meldet, ob der Principal die Klartext-Bankdaten des Mitglieds
// memberID lesen darf. Berechtigt sind: admin, Vereinsfunktion vorstand oder
// kassierer, der Eigentümer (das mit dem Nutzer verknüpfte Mitglied selbst) sowie
// ein über family_links verbundenes Elternteil.
//
// Die Funktion ist die einzige Stelle, an der diese Regel definiert ist; jeder
// Lesepfad MUSS sie aufrufen, bevor er entschlüsselte Bankdaten ausliefert (D5).
func CanDecryptBankData(db *sql.DB, p *Principal, memberID int) bool {
	// IsKassiererLike deckt admin + vorstand + kassierer ab.
	if IsKassiererLike(p) {
		return true
	}
	if p.UserID == 0 {
		return false
	}
	// Eigentümer: das Mitglied ist mit dem aufrufenden Nutzer verknüpft.
	var memberUserID sql.NullInt64
	if err := db.QueryRow(`SELECT user_id FROM members WHERE id=?`, memberID).Scan(&memberUserID); err == nil {
		if memberUserID.Valid && int(memberUserID.Int64) == p.UserID {
			return true
		}
	}
	// Elternteil: über family_links mit dem Mitglied verbunden.
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?`,
		p.UserID, memberID).Scan(&count); err == nil && count > 0 {
		return true
	}
	return false
}
