package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// realPNGBytes erzeugt ein echtes, decodierbares PNG mit den gewünschten
// Dimensionen — für Backfill-Tests, die die Header-Probe (decodeDimensions)
// prüfen. Interner Test-Helper (parallel zum externen realPNG in
// handler_test.go, aber vom internen media-Package aus sichtbar).
func realPNGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{0, 255, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}
