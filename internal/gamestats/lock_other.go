//go:build !unix

package gamestats

// tryLock ist außerhalb von Unix ein No-op: der Server läuft nur unter Linux,
// die Variante existiert nur, damit das Paket überall baut.
func tryLock(string) (release func(), ok bool, err error) {
	return func() {}, true, nil
}
