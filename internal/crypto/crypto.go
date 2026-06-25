// Package crypto kapselt die serverseitige At-Rest-Verschlüsselung der
// Bank-/SEPA-PII (AES-256-GCM, stdlib, kein CGo). Es hält einen app-weiten,
// aus der Umgebung geladenen Schlüssel (FIELD_ENCRYPTION_KEY).
//
// Zwei Formate:
//   - String-Werte (IBAN, Kontoinhaber, …): "v1:" + base64(nonce ‖ ciphertext)
//   - Datei-Blobs (SEPA-Mandat-PDFs): Magic-Header ‖ nonce ‖ ciphertext
//
// Decrypt/DecryptBytes sind tolerant: Werte ohne Prefix bzw. Blobs ohne
// Magic-Header gelten als (noch) unverschlüsselter Klartext und werden
// unverändert zurückgegeben. Das erlaubt einen Zero-Downtime-Rollout und eine
// idempotente Erstverschlüsselung (siehe cmd encrypt-pii).
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// KeySize ist die geforderte Schlüssellänge (AES-256).
const KeySize = 32

// EnvKeyName ist die Umgebungsvariable, aus der der Schlüssel geladen wird.
const EnvKeyName = "FIELD_ENCRYPTION_KEY"

// ErrNoKey signalisiert einen NICHT gesetzten Brücken-Schlüssel. Nach abgeschlossener
// Bestandsmigration ist das der Normalzustand; der Serverstart behandelt ihn als Warnung,
// nicht als Fehler (siehe InitFromEnv).
var ErrNoKey = errors.New("FIELD_ENCRYPTION_KEY ist nicht gesetzt (Migrations-Brücke deaktiviert)")

// prefix markiert einen verschlüsselten String-Wert.
const prefix = "v1:"

// fileMagic markiert einen verschlüsselten Datei-Blob. Bewusst kein gültiger
// PDF-Header (%PDF), damit unverschlüsselte Bestands-PDFs sicher als Klartext
// erkannt werden.
var fileMagic = []byte("TWENC1\x00")

// clientFileMagic markiert einen CLIENTseitig (Zero-Knowledge, Modell B) verschlüsselten
// Blob — bewusst anders als fileMagic (\n statt \x00), damit der Server ihn nie als
// serverseitig entschlüsselbar behandelt (er besitzt den DEK nicht). Muss mit dem
// BLOB_MAGIC in web/src/lib/crypto.ts übereinstimmen.
var clientFileMagic = []byte("TWENC1\n")

// IsClientEncryptedBytes meldet, ob ein Blob den Client-Magic-Header trägt (also clientseitig
// verschlüsselt wurde). Dient dem Upload-Pfad als Schutz gegen versehentlichen Klartext.
func IsClientEncryptedBytes(blob []byte) bool {
	return bytes.HasPrefix(blob, clientFileMagic)
}

// activeKey ist der app-weite Schlüssel. Wird beim Start via InitFromEnv bzw.
// in Tests via Init gesetzt.
var activeKey []byte

// LoadKey dekodiert und validiert einen base64-kodierten 32-Byte-Schlüssel.
func LoadKey(b64 string) ([]byte, error) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return nil, fmt.Errorf("%s ist nicht gesetzt", EnvKeyName)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("%s ist kein gültiges base64: %w", EnvKeyName, err)
	}
	if len(raw) != KeySize {
		return nil, fmt.Errorf("%s muss %d Byte sein (war %d)", EnvKeyName, KeySize, len(raw))
	}
	return raw, nil
}

// Init setzt den app-weiten Schlüssel (für Tests und nach LoadKey).
func Init(key []byte) error {
	if len(key) != KeySize {
		return fmt.Errorf("crypto: Schlüssel muss %d Byte sein (war %d)", KeySize, len(key))
	}
	activeKey = key
	return nil
}

// HasKey meldet, ob ein gültiger Brücken-Schlüssel geladen ist. Nach Abschluss der
// Bestandsmigration läuft der Server ohne Schlüssel (HasKey()==false); dann ist der
// serverseitige Migrations-/Decrypt-Pfad deaktiviert.
func HasKey() bool {
	return len(activeKey) == KeySize
}

// ClearKey entfernt den app-weiten Schlüssel (HasKey()==false). Spiegelt im Test den
// Zustand „Server läuft ohne FIELD_ENCRYPTION_KEY" (Brücke deaktiviert).
func ClearKey() {
	activeKey = nil
}

// InitFromEnv lädt FIELD_ENCRYPTION_KEY (Migrations-Brücke) und setzt den app-weiten
// Schlüssel. Tolerant gegenüber einem FEHLENDEN Schlüssel: nach der Migration startet der
// Server ohne ihn (ErrNoKey, vom Aufrufer als Warnung behandelt) — dann sind nur der
// Migrationspfad und das Entschlüsseln von Legacy-`v1:`-Bestand deaktiviert; alle regulären
// Routen sind envelope-only. Ein GESETZTER, aber ungültiger Schlüssel bleibt ein harter
// Fehler (sonst liefe der Server mit einer Fehlkonfiguration).
func InitFromEnv() error {
	raw := strings.TrimSpace(os.Getenv(EnvKeyName))
	if raw == "" {
		return ErrNoKey
	}
	key, err := LoadKey(raw)
	if err != nil {
		return err
	}
	return Init(key)
}

// Encrypt verschlüsselt einen String-Wert zu "v1:" + base64(nonce ‖ ciphertext).
func Encrypt(plaintext string) (string, error) {
	blob, err := seal([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return prefix + base64.StdEncoding.EncodeToString(blob), nil
}

// Decrypt entschlüsselt einen mit Encrypt erzeugten Wert. Werte ohne "v1:"-Prefix
// gelten als Klartext und werden unverändert zurückgegeben. Bei gebrochener
// GCM-Authentifizierung liefert Decrypt einen Fehler statt eines Klartextwerts.
func Decrypt(value string) (string, error) {
	if !strings.HasPrefix(value, prefix) {
		return value, nil
	}
	raw, err := base64.StdEncoding.DecodeString(value[len(prefix):])
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: base64: %w", err)
	}
	pt, err := open(raw)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// EncryptBytes verschlüsselt einen Datei-Blob (Magic-Header ‖ nonce ‖ ciphertext).
func EncryptBytes(plaintext []byte) ([]byte, error) {
	blob, err := seal(plaintext)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(fileMagic)+len(blob))
	out = append(out, fileMagic...)
	out = append(out, blob...)
	return out, nil
}

// DecryptBytes entschlüsselt einen mit EncryptBytes erzeugten Blob. Blobs ohne
// Magic-Header gelten als Klartext und werden unverändert zurückgegeben.
func DecryptBytes(blob []byte) ([]byte, error) {
	if !IsEncryptedBytes(blob) {
		return blob, nil
	}
	return open(blob[len(fileMagic):])
}

// IsEncryptedString meldet, ob ein Wert bereits den "v1:"-Prefix trägt.
func IsEncryptedString(v string) bool {
	return strings.HasPrefix(v, prefix)
}

// IsEncryptedBytes meldet, ob ein Blob bereits den Magic-Header trägt.
func IsEncryptedBytes(blob []byte) bool {
	return bytes.HasPrefix(blob, fileMagic)
}

// seal erzeugt nonce ‖ ciphertext (GCM) mit zufälligem 12-Byte-Nonce.
func seal(plaintext []byte) ([]byte, error) {
	gcm, err := newGCM()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// open zerlegt nonce ‖ ciphertext und entschlüsselt (mit Authentifizierung).
func open(blob []byte) ([]byte, error) {
	gcm, err := newGCM()
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("crypto: ciphertext zu kurz")
	}
	pt, err := gcm.Open(nil, blob[:ns], blob[ns:], nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: entschlüsseln fehlgeschlagen (Authentifizierung): %w", err)
	}
	return pt, nil
}

func newGCM() (cipher.AEAD, error) {
	if len(activeKey) != KeySize {
		return nil, errors.New("crypto: kein Schlüssel initialisiert (InitFromEnv/Init aufrufen)")
	}
	block, err := aes.NewCipher(activeKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: aes: %w", err)
	}
	return cipher.NewGCM(block)
}
