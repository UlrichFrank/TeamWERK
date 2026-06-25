package crypto

import "testing"

// TestIsClientEncryptedBytes prüft die Erkennung clientseitig verschlüsselter Blobs (das
// einzige verbliebene Krypto-Verhalten serverseitig — der Server entschlüsselt nichts mehr).
func TestIsClientEncryptedBytes(t *testing.T) {
	cases := map[string]struct {
		blob []byte
		want bool
	}{
		"client-magic":       {append([]byte("TWENC1\n"), 1, 2, 3), true},
		"klartext-pdf":       {[]byte("%PDF-1.4 ..."), false},
		"leer":               {[]byte{}, false},
		"alter-server-magic": {append([]byte("TWENC1\x00"), 1, 2, 3), false},
		"zu-kurz":            {[]byte("TWENC"), false},
	}
	for name, c := range cases {
		if got := IsClientEncryptedBytes(c.blob); got != c.want {
			t.Errorf("%s: IsClientEncryptedBytes = %v, want %v", name, got, c.want)
		}
	}
}
