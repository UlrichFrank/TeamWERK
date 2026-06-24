package members

import (
	"database/sql"

	"github.com/teamstuttgart/teamwerk/internal/crypto"
)

// encBankField verschlüsselt einen nicht-leeren Bank-Wert (IBAN, Kontoinhaber)
// für die At-Rest-Speicherung. Leere Werte werden zu NULL (any(nil)). Jeder
// Schreibpfad auf members.iban/account_holder MUSS hierüber gehen, damit nie
// Klartext in der DB landet.
func encBankField(s string) (any, error) {
	if s == "" {
		return nil, nil
	}
	return crypto.Encrypt(s)
}

// decBankField entschlüsselt einen Bank-Wert aus der DB. Ungültige/nicht
// entschlüsselbare Werte ergeben nil (Feld wird weggelassen, kein Klartext-Leak).
// Werte ohne "v1:"-Prefix gelten als noch nicht migrierter Klartext (Passthrough).
func decBankField(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	pt, err := crypto.Decrypt(ns.String)
	if err != nil {
		return nil
	}
	return &pt
}

// decryptMemberBank entschlüsselt die Bank-Felder eines geladenen Members in-place
// (getMember liefert den rohen, verschlüsselten Wert). Nur aufrufen, nachdem die
// Berechtigung geprüft wurde (Eigentümer/Eltern bzw. CanDecryptBankData).
func decryptMemberBank(m *Member) {
	if m == nil {
		return
	}
	if m.IBAN != nil {
		if pt, err := crypto.Decrypt(*m.IBAN); err == nil {
			m.IBAN = &pt
		} else {
			m.IBAN = nil
		}
	}
	if m.AccountHolder != nil {
		if pt, err := crypto.Decrypt(*m.AccountHolder); err == nil {
			m.AccountHolder = &pt
		} else {
			m.AccountHolder = nil
		}
	}
}
