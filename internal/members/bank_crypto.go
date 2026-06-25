package members

import (
	"github.com/teamstuttgart/teamwerk/internal/crypto"
)

// encBankField verschlüsselt einen nicht-leeren Bank-Wert (IBAN, Kontoinhaber)
// für die At-Rest-Speicherung. Leere Werte werden zu NULL (any(nil)).
//
// HINWEIS (Modell B): Dies ist serverseitige Verschlüsselung und wird nur noch von
// den NOCH NICHT auf den Zero-Knowledge-Envelope umgestellten Schreibpfaden genutzt
// (CSV-Import, Member-Create, Bankdaten-Drafts). Diese Pfade werden in den folgenden
// Tasks (3.2/3.5) auf clientseitige Verschlüsselung umgestellt.
func encBankField(s string) (any, error) {
	if s == "" {
		return nil, nil
	}
	return crypto.Encrypt(s)
}

// clearMemberBank entfernt alle Bank-Felder eines geladenen Members. Modell B/G2:
// Nur die Finance-Gruppe liest Bankdaten (clientseitig über den Envelope); Eigentümer
// und Eltern erhalten sie nicht — auch nicht als Ciphertext.
func clearMemberBank(m *Member) {
	if m == nil {
		return
	}
	m.IBAN = nil
	m.AccountHolder = nil
	m.BankCiphertext = nil
	m.BankDekEnc = nil
}
