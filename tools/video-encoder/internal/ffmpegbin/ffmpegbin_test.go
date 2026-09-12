package ffmpegbin

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func gzBytes(t *testing.T, content string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	if _, err := zw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// Erster Start: die eingebettete Binary landet ausführbar im Cache.
func TestExtract_FirstStartWritesExecutable(t *testing.T) {
	cache := t.TempDir()
	p, err := extract(cache, gzBytes(t, "fake-ffmpeg-v1"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p, cache) {
		t.Fatalf("binary must live in the cache dir, got %q", p)
	}
	data, err := os.ReadFile(p)
	if err != nil || string(data) != "fake-ffmpeg-v1" {
		t.Fatalf("unexpected content %q (err %v)", data, err)
	}
	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(p)
		if fi.Mode().Perm()&0o100 == 0 {
			t.Fatalf("binary must be executable, mode %v", fi.Mode())
		}
	}
	// Keine Temp-Reste neben der Binary.
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Fatalf("expected only the binary in %s, got %d entries", filepath.Dir(p), len(entries))
	}
}

// Folgestart mit derselben Version entpackt nicht erneut.
func TestExtract_SecondStartReusesCache(t *testing.T) {
	cache := t.TempDir()
	gz := gzBytes(t, "fake-ffmpeg-v1")
	p1, err := extract(cache, gz)
	if err != nil {
		t.Fatal(err)
	}
	// Markierung setzen: ein erneutes Entpacken würde sie überschreiben.
	if err := os.WriteFile(p1, []byte("marker"), 0o755); err != nil {
		t.Fatal(err)
	}
	p2, err := extract(cache, gz)
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatalf("same embedded version must map to the same path: %q vs %q", p1, p2)
	}
	if data, _ := os.ReadFile(p2); string(data) != "marker" {
		t.Fatalf("second start must not re-extract, content now %q", data)
	}
}

// Eine andere eingebettete Version entpackt neu und räumt die alte weg.
func TestExtract_NewVersionReplacesOld(t *testing.T) {
	cache := t.TempDir()
	p1, err := extract(cache, gzBytes(t, "fake-ffmpeg-v1"))
	if err != nil {
		t.Fatal(err)
	}
	p2, err := extract(cache, gzBytes(t, "fake-ffmpeg-v2"))
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p2 {
		t.Fatal("a different embedded binary must extract to a different path")
	}
	if data, _ := os.ReadFile(p2); string(data) != "fake-ffmpeg-v2" {
		t.Fatalf("unexpected content %q", data)
	}
	if _, err := os.Stat(filepath.Dir(p1)); !os.IsNotExist(err) {
		t.Fatalf("stale version dir must be removed, stat err=%v", err)
	}
}

func TestExtract_CorruptArchiveFails(t *testing.T) {
	if _, err := extract(t.TempDir(), []byte("not gzip")); err == nil {
		t.Fatal("expected error for corrupt archive")
	}
}
