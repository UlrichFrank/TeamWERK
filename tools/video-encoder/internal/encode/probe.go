package encode

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Source beschreibt die für den Encode relevanten Eigenschaften der Quelle.
type Source struct {
	Duration   time.Duration // 0, wenn unbekannt (Fortschritt dann unbestimmt)
	HasVideo   bool
	AudioCodec string // Codec der ersten Audiospur, "" ohne Audio
}

var (
	durationRe = regexp.MustCompile(`Duration:\s*(\d+):(\d{2}):(\d{2}(?:\.\d+)?)`)
	// „Stream #0:1[0x2](und): Audio: aac (LC) …" — zwischen Stream-Nummer und
	// Typ stehen optional [id] und (sprache), beide ohne Doppelpunkt.
	streamRe = regexp.MustCompile(`Stream #\d+:\d+[^:]*:\s*(Video|Audio):\s*([A-Za-z0-9_]+)`)
)

// Probe liest Dauer und Spuren der Quelle aus `ffmpeg -i` — das Tool bündelt
// bewusst nur ffmpeg, kein ffprobe (spart eine zweite Binary im Download).
func Probe(ctx context.Context, ffmpegPath, in string) (Source, error) {
	cmd := exec.CommandContext(ctx, ffmpegPath, "-hide_banner", "-nostdin", "-i", in)
	hideWindow(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// Exit-Code 1 ist hier der Normalfall: ohne Ausgabedatei bricht ffmpeg nach
	// der Analyse ab. Ausgewertet wird nur, was es bis dahin geschrieben hat.
	_ = cmd.Run()
	if err := ctx.Err(); err != nil {
		return Source{}, err
	}
	src := ParseProbe(stderr.String())
	if !src.HasVideo {
		return src, fmt.Errorf("keine Videospur gefunden (%s)", lastLines(stderr.String(), 1))
	}
	return src, nil
}

// ParseProbe wertet die stderr-Ausgabe von `ffmpeg -i` aus.
func ParseProbe(out string) Source {
	var src Source
	if m := durationRe.FindStringSubmatch(out); m != nil {
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		s, _ := strconv.ParseFloat(m[3], 64)
		src.Duration = time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute +
			time.Duration(s*float64(time.Second))
	}
	for _, m := range streamRe.FindAllStringSubmatch(out, -1) {
		switch m[1] {
		case "Video":
			src.HasVideo = true
		case "Audio":
			if src.AudioCodec == "" {
				src.AudioCodec = strings.ToLower(m[2])
			}
		}
	}
	return src
}

// lastLines liefert die letzten n nicht-leeren Zeilen von s, mit " | " verbunden.
func lastLines(s string, n int) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}
