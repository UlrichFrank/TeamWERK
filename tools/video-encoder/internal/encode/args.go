// Package encode baut und startet den ffmpeg-Aufruf, der eine Rohaufnahme in
// das Format bringt, das der TeamWERK-Server nur noch umpackt (Stream-Copy):
// H.264/yuv420p, 720p, Keyframes alle 4 s, AAC.
//
// Diese Parameter waren bis zum Change video-offline-encoding-tool die des
// Server-Workers (internal/videos/worker.go). Der Worker prüft vor dem Remux
// Codec und Pixelformat (h264 + yuv420p) — wer hier etwas davon ändert, muss
// die Prüfung dort mitziehen, sonst lehnt der Server jedes Video mit
// unsupported_input_format ab.
package encode

const (
	// Profile und Level sind fest gepinnt statt dem libx264-Default überlassen:
	// der Default hängt vom Inhalt ab (Auflösung, Bildrate, Referenzframes), das
	// Ergebnis soll aber je Tool-Version für alle Quellen gleich sein.
	//
	// Level 4.0 statt des naheliegenden 3.1: 3.1 endet bei 720p mit 30 fps.
	// Hallenaufnahmen mit 50/60 fps lägen darüber, das signalisierte Level wäre
	// dann schlicht falsch. High@4.0 dekodieren Apple TV und alle iOS-Geräte.
	Profile = "high"
	Level   = "4.0"

	// TargetHeight ist die Ausgabehöhe; die Breite folgt dem Seitenverhältnis.
	TargetHeight = 720
	// KeyframeInterval muss zur Server-Segmentlänge (`-hls_time 4`) passen:
	// beim Stream-Copy kann der Server nur an vorhandenen Keyframes schneiden.
	KeyframeInterval = 4
)

// BuildArgs baut die ffmpeg-Argumentliste für den Encode von in nach out
// (MP4). Reine Funktion, damit Tests das exakte Layout prüfen können.
func BuildArgs(in, out string, src Source) []string {
	args := []string{
		"-hide_banner", "-nostdin", "-y",
		"-v", "error",
		"-i", in,
		// Nur die erste Video- und (falls vorhanden) die erste Audiospur: Handys
		// und Action-Cams legen Daten-/Metadatenspuren bei, die den MP4-Muxer
		// scheitern lassen oder beim Server-Remux Ärger machen.
		"-map", "0:v:0", "-map", "0:a:0?",
		"-vf", "scale=-2:720",
		// 8-bit erzwingen: moderne iPhones filmen 10-bit, libx264 würde daraus
		// yuv420p10 machen, das tvOS nicht dekodiert (Ton ohne Bild). Steht vor
		// -c:v, damit das Ausgabeformat vor der Codec-Wahl feststeht.
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-profile:v", Profile,
		"-level:v", Level,
		"-preset", "medium",
		"-crf", "26",
		"-maxrate", "2800k",
		"-bufsize", "5600k",
		// Ohne erzwungene Keyframes wächst die GOP bei Sport-Content auf 16+ s;
		// der Server könnte dann nur dort schneiden → riesige Segmente,
		// Buffer-Underrun auf Mobilgeräten.
		"-force_key_frames", "expr:gte(t,n_forced*4)",
	}
	if src.AudioCodec == "aac" {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "128k")
	}
	return append(args,
		// Fortschritt als key=value-Zeilen auf stdout (siehe parseProgress).
		"-progress", "pipe:1", "-nostats",
		out,
	)
}
