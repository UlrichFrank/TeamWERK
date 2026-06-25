package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
)

// newTestKey erzeugt einen zufälligen 32-Byte-Schlüssel und setzt ihn aktiv.
func newTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("rand: %v", err)
	}
	if err := Init(key); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return key
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	newTestKey(t)
	for _, pt := range []string{"DE89370400440532013000", "Max Mustermann", "", "äöü ß €"} {
		ct, err := Encrypt(pt)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", pt, err)
		}
		if !strings.HasPrefix(ct, prefix) {
			t.Errorf("Encrypt(%q) = %q, fehlt %q-Prefix", pt, ct, prefix)
		}
		if ct == pt && pt != "" {
			t.Errorf("Ciphertext gleich Klartext für %q", pt)
		}
		got, err := Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if got != pt {
			t.Errorf("Roundtrip: got %q, want %q", got, pt)
		}
	}
}

func TestEncrypt_NonceIsRandom(t *testing.T) {
	newTestKey(t)
	a, _ := Encrypt("DE89370400440532013000")
	b, _ := Encrypt("DE89370400440532013000")
	if a == b {
		t.Error("zweimal gleicher Ciphertext — Nonce nicht zufällig")
	}
}

func TestDecrypt_PlaintextPassthrough(t *testing.T) {
	newTestKey(t)
	// Wert ohne "v1:"-Prefix gilt als noch nicht migrierter Klartext.
	for _, pt := range []string{"DE89370400440532013000", "", "irgendwas"} {
		got, err := Decrypt(pt)
		if err != nil {
			t.Fatalf("Decrypt(%q) plaintext: %v", pt, err)
		}
		if got != pt {
			t.Errorf("Passthrough: got %q, want %q", got, pt)
		}
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	newTestKey(t)
	ct, err := Encrypt("geheim")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	newTestKey(t) // anderer Schlüssel aktiv
	if _, err := Decrypt(ct); err == nil {
		t.Error("Decrypt mit falschem Schlüssel lieferte keinen Fehler")
	}
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	newTestKey(t)
	ct, err := Encrypt("geheim")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(ct[len(prefix):])
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // letztes Byte (im GCM-Tag) kippen
	tampered := prefix + base64.StdEncoding.EncodeToString(raw)
	if _, err := Decrypt(tampered); err == nil {
		t.Error("manipulierter Ciphertext lieferte keinen Fehler")
	}
}

func TestEncryptBytes_Roundtrip(t *testing.T) {
	newTestKey(t)
	pdf := []byte("%PDF-1.4\n... binär ...\x00\x01\x02")
	enc, err := EncryptBytes(pdf)
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}
	if !IsEncryptedBytes(enc) {
		t.Error("EncryptBytes-Ausgabe trägt keinen Magic-Header")
	}
	if bytes.HasPrefix(enc, []byte("%PDF")) {
		t.Error("verschlüsselter Blob beginnt noch mit %PDF")
	}
	dec, err := DecryptBytes(enc)
	if err != nil {
		t.Fatalf("DecryptBytes: %v", err)
	}
	if !bytes.Equal(dec, pdf) {
		t.Error("Datei-Roundtrip stimmt nicht überein")
	}
}

func TestDecryptBytes_PlaintextPassthrough(t *testing.T) {
	newTestKey(t)
	pdf := []byte("%PDF-1.4 unverschlüsselt")
	got, err := DecryptBytes(pdf)
	if err != nil {
		t.Fatalf("DecryptBytes passthrough: %v", err)
	}
	if !bytes.Equal(got, pdf) {
		t.Error("unverschlüsselter Blob wurde verändert")
	}
}

func TestIsEncrypted_IdempotentReEncryptSkip(t *testing.T) {
	newTestKey(t)
	ct, _ := Encrypt("DE89370400440532013000")
	// Eine Migration prüft IsEncryptedString und überspringt bereits verschlüsselte Werte.
	if !IsEncryptedString(ct) {
		t.Error("IsEncryptedString(ct) = false")
	}
	if IsEncryptedString("DE89370400440532013000") {
		t.Error("IsEncryptedString(plaintext) = true")
	}
	enc, _ := EncryptBytes([]byte("blob"))
	if !IsEncryptedBytes(enc) {
		t.Error("IsEncryptedBytes(enc) = false")
	}
	if IsEncryptedBytes([]byte("%PDF blob")) {
		t.Error("IsEncryptedBytes(plaintext) = true")
	}
}

func TestLoadKey_Validation(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString(make([]byte, KeySize))
	if _, err := LoadKey(valid); err != nil {
		t.Errorf("LoadKey(valid): %v", err)
	}
	cases := map[string]string{
		"leer":          "",
		"kein base64":   "!!!nope!!!",
		"falsche Länge": base64.StdEncoding.EncodeToString(make([]byte, 16)),
	}
	for name, in := range cases {
		if _, err := LoadKey(in); err == nil {
			t.Errorf("LoadKey(%s) lieferte keinen Fehler", name)
		}
	}
}

// TestInitFromEnv_Tolerance prüft die Migrations-Brücken-Semantik: fehlender Schlüssel ist
// kein harter Fehler (ErrNoKey → Serverstart läuft als Warnung weiter), ein gesetzter aber
// ungültiger Schlüssel bleibt ein Fehler, ein gültiger Schlüssel aktiviert die Brücke.
func TestInitFromEnv_Tolerance(t *testing.T) {
	// Paketweiten Schlüssel für diesen Test isolieren und danach zurücksetzen.
	saved := activeKey
	t.Cleanup(func() { activeKey = saved })

	t.Run("fehlender Schlüssel → ErrNoKey, HasKey()==false", func(t *testing.T) {
		activeKey = nil
		t.Setenv(EnvKeyName, "")
		if err := InitFromEnv(); !errors.Is(err, ErrNoKey) {
			t.Fatalf("InitFromEnv ohne Key: erwartet ErrNoKey, war %v", err)
		}
		if HasKey() {
			t.Error("HasKey() sollte ohne Schlüssel false sein")
		}
	})

	t.Run("ungültiger Schlüssel → harter Fehler", func(t *testing.T) {
		activeKey = nil
		t.Setenv(EnvKeyName, base64.StdEncoding.EncodeToString(make([]byte, 16)))
		err := InitFromEnv()
		if err == nil || errors.Is(err, ErrNoKey) {
			t.Fatalf("InitFromEnv mit ungültigem Key: erwartet harten Fehler, war %v", err)
		}
		if HasKey() {
			t.Error("HasKey() sollte nach ungültigem Schlüssel false sein")
		}
	})

	t.Run("gültiger Schlüssel → Brücke aktiv", func(t *testing.T) {
		activeKey = nil
		t.Setenv(EnvKeyName, base64.StdEncoding.EncodeToString(make([]byte, KeySize)))
		if err := InitFromEnv(); err != nil {
			t.Fatalf("InitFromEnv mit gültigem Key: %v", err)
		}
		if !HasKey() {
			t.Error("HasKey() sollte mit gültigem Schlüssel true sein")
		}
	})
}
