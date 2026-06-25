// Package crypto enthält nach Abschluss der Zero-Knowledge-Bestandsmigration nur noch die
// Erkennung CLIENTseitig (Modell B) verschlüsselter Datei-Blobs. Der Server besitzt KEINEN
// Entschlüsselungsschlüssel mehr: Bank-/SEPA-PII wird ausschließlich im Browser ver- und
// entschlüsselt, der Server speichert nur Ciphertext + gewrappte Schlüssel.
//
// Die frühere serverseitige At-Rest-Verschlüsselung (FIELD_ENCRYPTION_KEY, "v1:"-Format,
// Encrypt/Decrypt, EncryptBytes/DecryptBytes, encrypt-pii) sowie die Migrations-Brücke wurden
// mit dem Abschluss der Migration entfernt.
package crypto

import "bytes"

// clientFileMagic markiert einen clientseitig (Zero-Knowledge, Modell B) verschlüsselten
// Blob. Muss mit dem BLOB_MAGIC in web/src/lib/crypto.ts übereinstimmen.
var clientFileMagic = []byte("TWENC1\n")

// IsClientEncryptedBytes meldet, ob ein Blob den Client-Magic-Header trägt (also clientseitig
// verschlüsselt wurde). Dient dem Upload-Pfad als Schutz gegen versehentlichen Klartext.
func IsClientEncryptedBytes(blob []byte) bool {
	return bytes.HasPrefix(blob, clientFileMagic)
}
