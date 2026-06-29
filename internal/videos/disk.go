package videos

import (
	"errors"
	"syscall"
)

// ErrInsufficientDiskSpace wird zurückgegeben, wenn der freie Speicher unter dem
// Disk-Guard liegt. Aufrufer können darauf via errors.Is(...) prüfen und z.B.
// HTTP 507 Insufficient Storage antworten.
var ErrInsufficientDiskSpace = errors.New("videos: insufficient disk space")

// FreeBytes liefert den für nicht-privilegierte Prozesse verfügbaren freien
// Speicher (in Bytes) des Dateisystems, das dir enthält.
func FreeBytes(dir string) (uint64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, err
	}
	// Bavail = für Nicht-Root verfügbare Blöcke; Bsize = Blockgröße in Bytes.
	return uint64(st.Bavail) * uint64(st.Bsize), nil
}

// RequireFreeBytes prüft, ob nach dem Verbrauch von needed Bytes mindestens
// reserved Bytes frei bleiben. Reicht der Platz nicht, wird ein in
// ErrInsufficientDiskSpace eingewickelter Fehler zurückgegeben.
func RequireFreeBytes(dir string, needed, reserved uint64) error {
	free, err := FreeBytes(dir)
	if err != nil {
		return err
	}
	if free < needed+reserved {
		return ErrInsufficientDiskSpace
	}
	return nil
}
