// Package ffmpegbin liefert den Pfad zu einer ausführbaren ffmpeg-Binary.
//
// Release-Builds betten einen statischen GPL-Build von ffmpeg für die jeweilige
// Zielplattform ein (bin/ffmpeg.gz samt Lizenz-/Herkunftsdateien, abgelegt vom
// Job `video-encoder` in .github/workflows/release.yml vor dem Build). Beim
// ersten Start wird die Binary ins Cache-Verzeichnis entpackt und von dort als
// eigenständiger Prozess aufgerufen — gegen libav*/libx264 wird nicht gelinkt
// (design.md, Entscheidung 4). Entwickler-Builds ohne eingebettete Binary fallen
// auf ffmpeg aus dem PATH zurück.
package ffmpegbin

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// bin enthält im Repo nur README.md (damit das Embed-Muster immer trifft); die
// ffmpeg-Dateien legt der Release-Build daneben.
//
//go:embed bin
var binFS embed.FS

const (
	gzPath      = "bin/ffmpeg.gz"
	licensePath = "bin/ffmpeg.LICENSE"
	readmePath  = "bin/ffmpeg.README"
	sourcePath  = "bin/ffmpeg.SOURCE"
	dirPrefix   = "ffmpeg-"
)

// ErrUnavailable: weder eingebettet noch im PATH.
var ErrUnavailable = errors.New("ffmpeg nicht verfügbar: nicht eingebettet und nicht im PATH")

// Embedded meldet, ob dieser Build eine ffmpeg-Binary mitbringt.
func Embedded() bool {
	_, err := fs.Stat(binFS, gzPath)
	return err == nil
}

// Path liefert den Pfad zur ffmpeg-Binary. Ist eine eingebettet, wird sie bei
// Bedarf nach cacheRoot entpackt; sonst gilt ffmpeg aus dem PATH.
func Path(cacheRoot string) (string, error) {
	gz, err := binFS.ReadFile(gzPath)
	if errors.Is(err, fs.ErrNotExist) {
		if p, lerr := exec.LookPath("ffmpeg"); lerr == nil {
			return p, nil
		}
		return "", ErrUnavailable
	}
	if err != nil {
		return "", err
	}
	return extract(cacheRoot, gz)
}

// extract entpackt gz nach cacheRoot/ffmpeg-<hash>/ffmpeg[.exe], sofern dort
// noch nichts liegt. Der Hash des eingebetteten Archivs steht im
// Verzeichnisnamen: eine Tool-Version mit anderem ffmpeg entpackt neu, dieselbe
// nie ein zweites Mal. Geschrieben wird über eine temporäre Datei plus Rename,
// damit ein abgebrochener Erststart keine halbe Binary hinterlässt, die der
// nächste Start für fertig hielte.
func extract(cacheRoot string, gz []byte) (string, error) {
	sum := sha256.Sum256(gz)
	dirName := dirPrefix + hex.EncodeToString(sum[:8])
	dir := filepath.Join(cacheRoot, dirName)
	target := filepath.Join(dir, exeName())
	if fi, err := os.Stat(target); err == nil && fi.Mode().IsRegular() {
		return target, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, exeName()+".*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name()) // nach erfolgreichem Rename ein No-op

	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		tmp.Close()
		return "", fmt.Errorf("eingebettetes ffmpeg ist beschädigt: %w", err)
	}
	if _, err := io.Copy(tmp, zr); err != nil {
		tmp.Close()
		return "", fmt.Errorf("ffmpeg entpacken: %w", err)
	}
	if err := zr.Close(); err != nil {
		tmp.Close()
		return "", fmt.Errorf("ffmpeg entpacken: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		// Parallel entpackt (Hintergrund-Warmup und erster Encode): hat der
		// andere gewonnen, ist das Ziel fertig. Unter Windows scheitert das
		// Überschreiben einer bereits laufenden ffmpeg.exe genau so.
		if fi, serr := os.Stat(target); serr == nil && fi.Mode().IsRegular() {
			return target, nil
		}
		return "", err
	}
	removeStale(cacheRoot, dirName)
	return target, nil
}

// removeStale löscht (best effort) die von früheren Tool-Versionen entpackten
// ffmpeg-Verzeichnisse — je bis zu 80 MB, die sonst im Cache liegen blieben.
func removeStale(cacheRoot, keep string) {
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), dirPrefix) && e.Name() != keep {
			_ = os.RemoveAll(filepath.Join(cacheRoot, e.Name()))
		}
	}
}

func exeName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

// Notice liefert den Lizenz- und Herkunftshinweis zur ffmpeg-Binary für den
// Über-Dialog.
func Notice() string {
	if !Embedded() {
		return "Entwickler-Build ohne eingebettetes ffmpeg — verwendet wird das ffmpeg aus dem PATH."
	}
	var b strings.Builder
	b.WriteString("Dieses Programm liefert eine unveränderte ffmpeg-Binary mit und ruft sie als " +
		"eigenständigen Prozess auf. ffmpeg ist ein Projekt von ffmpeg.org und steht unter der " +
		"GNU General Public License (GPL); der Quelltext ist unter https://ffmpeg.org/download.html erhältlich.\n")
	for _, p := range []string{sourcePath, readmePath, licensePath} {
		if data, err := binFS.ReadFile(p); err == nil {
			b.WriteString("\n")
			b.Write(bytes.TrimSpace(data))
			b.WriteString("\n")
		}
	}
	return b.String()
}
