package members

// clearMemberBank entfernt alle Bank-Felder eines geladenen Members. Modell B/G2:
// Nur die Finance-Gruppe liest Bankdaten (clientseitig über den Envelope); Eigentümer
// und Eltern erhalten sie nicht — auch nicht als Ciphertext.
func clearMemberBank(m *Member) {
	if m == nil {
		return
	}
	m.BankCiphertext = nil
	m.BankDekEnc = nil
}
