package media

import (
	"bytes"
	"image"
	_ "image/gif"  // register GIF decoder for image.DecodeConfig
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder

	"golang.org/x/image/webp"
)

// decodeDimensions liest nur den Header eines Bild-Byte-Slices und liefert
// die natürlichen Pixel-Dimensionen zurück. Kein Full-Decode — für ein
// 1-MB-JPEG spart das ~50 ms und ~4 MB Peak-Speicher. Scheitert die Probe
// (korrupter Header, unbekanntes Format), ist ok=false; der Aufrufer schreibt
// dann NULL in die DB und der Frontend-Fallback greift.
func decodeDimensions(data []byte, mimeType string) (w, h int, ok bool) {
	r := bytes.NewReader(data)
	if mimeType == "image/webp" {
		cfg, err := webp.DecodeConfig(r)
		if err != nil {
			return 0, 0, false
		}
		return cfg.Width, cfg.Height, true
	}
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}
