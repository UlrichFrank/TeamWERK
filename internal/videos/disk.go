package videos

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
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

// StorageStats liefert freien und gesamten Speicher (in Bytes) des
// Dateisystems, das dir enthält — eine einzige statfs(2) für beide Werte
// (für die "X GB frei von Y GB"-Anzeige auf /videos).
func StorageStats(dir string) (free, total uint64, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, 0, err
	}
	return uint64(st.Bavail) * uint64(st.Bsize), uint64(st.Blocks) * uint64(st.Bsize), nil
}

// DirSize summiert die Dateigrößen unter dir rekursiv. Ein (noch) nicht
// vorhandenes dir liefert 0 statt eines Fehlers — üblich für ein Video, dessen
// processed/{id}/ vor dem ersten Transcode-Lauf noch nicht existiert.
func DirSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
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
