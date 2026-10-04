//go:build unix

package gamestats

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// tryLock nimmt eine exklusive, nicht blockierende flock-Sperre auf path.
//
// ok=false heißt: ein anderer Prozess hält die Sperre. Das Betriebssystem gibt
// sie mit dem Prozess frei — ein abgestürzter Lauf hinterlässt also keine
// Leiche, die alle folgenden Ticks blockiert (anders als eine bloße
// Existenz-Prüfung auf die Datei).
func tryLock(path string) (release func(), ok bool, err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o640)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, true, nil
}
