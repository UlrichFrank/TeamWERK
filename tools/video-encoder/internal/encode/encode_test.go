package encode

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func idx(args []string, s string) int {
	for i, a := range args {
		if a == s {
			return i
		}
	}
	return -1
}

func hasPair(args []string, a, b string) bool {
	i := idx(args, a)
	return i >= 0 && i+1 < len(args) && args[i+1] == b
}

// Zielformat laut Spec: H.264/yuv420p, 720p, CRF 26/preset medium, 4-s-Keyframes,
// festes Profil/Level. -pix_fmt muss vor -c:v stehen.
func TestBuildArgs_TargetFormat(t *testing.T) {
	args := BuildArgs("in.mov", "out.mp4", Source{AudioCodec: "pcm_s16le"})
	for _, p := range [][2]string{
		{"-pix_fmt", "yuv420p"},
		{"-c:v", "libx264"},
		{"-profile:v", "high"},
		{"-level:v", "4.0"},
		{"-preset", "medium"},
		{"-crf", "26"},
		{"-vf", "scale=-2:720"},
		{"-force_key_frames", "expr:gte(t,n_forced*4)"},
		{"-c:a", "aac"},
		{"-b:a", "128k"},
		{"-progress", "pipe:1"},
	} {
		if !hasPair(args, p[0], p[1]) {
			t.Errorf("missing %s %s in %v", p[0], p[1], args)
		}
	}
	if idx(args, "-pix_fmt") > idx(args, "-c:v") {
		t.Errorf("-pix_fmt must precede -c:v: %v", args)
	}
	if idx(args, "-i") > idx(args, "-c:v") || args[len(args)-1] != "out.mp4" {
		t.Errorf("input must come first and output last: %v", args)
	}
}

// Das Profil hängt nicht von der Quelle ab (Spec: „Profil bleibt über
// verschiedene Quellinhalte stabil").
func TestBuildArgs_ProfileIndependentOfSource(t *testing.T) {
	a := BuildArgs("a", "o", Source{AudioCodec: "aac", Duration: time.Hour})
	b := BuildArgs("b", "o", Source{AudioCodec: "", Duration: time.Second})
	for _, flag := range []string{"-profile:v", "-level:v"} {
		if a[idx(a, flag)+1] != b[idx(b, flag)+1] {
			t.Fatalf("%s differs between sources", flag)
		}
	}
}

func TestBuildArgs_AudioCopyForAAC(t *testing.T) {
	args := BuildArgs("in.mp4", "out.mp4", Source{AudioCodec: "aac"})
	if !hasPair(args, "-c:a", "copy") || idx(args, "-b:a") >= 0 {
		t.Fatalf("AAC source must be stream-copied: %v", args)
	}
}

func TestParseProbe(t *testing.T) {
	out := `Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'IMG_0001.MOV':
  Duration: 01:02:03.50, start: 0.000000, bitrate: 25000 kb/s
  Stream #0:0[0x1](und): Video: hevc (Main 10) (hvc1 / 0x31637668), yuv420p10le(tv, bt2020nc), 3840x2160, 50 fps
  Stream #0:1[0x2](und): Audio: aac (LC) (mp4a / 0x6134706D), 48000 Hz, stereo, fltp, 191 kb/s
  Stream #0:2[0x3](und): Data: none (mebx / 0x7862656D), 0 kb/s
At least one output file must be specified`
	src := ParseProbe(out)
	want := time.Hour + 2*time.Minute + 3500*time.Millisecond
	if src.Duration != want || !src.HasVideo || src.AudioCodec != "aac" {
		t.Fatalf("got %+v, want duration %v, video, aac", src, want)
	}

	noAudio := ParseProbe("  Duration: N/A\n  Stream #0:0: Video: mjpeg, yuvj420p")
	if noAudio.Duration != 0 || !noAudio.HasVideo || noAudio.AudioCodec != "" {
		t.Fatalf("got %+v", noAudio)
	}
	if ParseProbe("x.txt: Invalid data found when processing input").HasVideo {
		t.Fatal("no stream lines must mean no video")
	}
}

func TestParseProgress(t *testing.T) {
	in := strings.Join([]string{
		"frame=0", "out_time_us=N/A", "out_time_ms=N/A", "progress=continue",
		"out_time_us=2500000", "out_time_ms=2500000", "progress=continue",
		"out_time_us=10000000", "out_time_ms=10000000", "progress=continue",
		"out_time_us=10500000", "progress=end",
	}, "\n")
	var got []float64
	parseProgress(strings.NewReader(in), 10*time.Second, func(f float64) { got = append(got, f) })
	want := []float64{0.25, 1, 1}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v (out_time_ms must not double-report)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}

	// Ältere ffmpeg-Versionen kennen nur out_time_ms.
	got = nil
	parseProgress(strings.NewReader("out_time_ms=5000000\n"), 10*time.Second, func(f float64) { got = append(got, f) })
	if len(got) != 1 || got[0] != 0.5 {
		t.Fatalf("legacy out_time_ms: got %v", got)
	}
}

// Exit-Code ≠ 0 wird zu einer lesbaren Meldung statt eines Absturzes.
func TestRun_FailureReportsExitCode(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not in PATH")
	}
	missing := filepath.Join(t.TempDir(), "gibt-es-nicht.mov")
	err = Run(context.Background(), ffmpeg, BuildArgs(missing, filepath.Join(t.TempDir(), "o.mp4"), Source{}), time.Second, func(float64) {})
	if err == nil || !strings.Contains(err.Error(), "Exit-Code") {
		t.Fatalf("expected exit-code error, got %v", err)
	}
}

// Integrationstest mit echtem ffmpeg: eine 10-bit-4:2:2-Quelle mit PCM-Ton
// kommt als H.264 High@4.0/yuv420p, 720p, AAC mit Keyframes bei 0/4/8 s heraus.
// Ohne ffmpeg/ffprobe/libx264 im PATH übersprungen.
func TestEncode_ResultMatchesTargetFormat(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not in PATH")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not in PATH")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src.mkv")
	gen := exec.Command(ffmpeg, "-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=640x360:rate=50:duration=9",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=9",
		"-c:v", "ffv1", "-pix_fmt", "yuv422p10le", "-c:a", "pcm_s16le", "-shortest", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Skipf("cannot generate source: %v: %s", err, out)
	}

	ctx := context.Background()
	s, err := Probe(ctx, ffmpeg, src)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if s.AudioCodec != "pcm_s16le" || s.Duration < 8*time.Second {
		t.Fatalf("unexpected probe result %+v", s)
	}

	out := filepath.Join(dir, "out.mp4")
	var last float64
	if err := Run(ctx, ffmpeg, BuildArgs(src, out, s), s.Duration, func(f float64) { last = f }); err != nil {
		if strings.Contains(err.Error(), "libx264") {
			t.Skipf("ffmpeg without libx264: %v", err)
		}
		t.Fatalf("Run: %v", err)
	}
	if last != 1 {
		t.Fatalf("final progress must be 1, got %v", last)
	}

	probe := func(args ...string) string {
		t.Helper()
		b, err := exec.Command("ffprobe", append([]string{"-v", "error"}, append(args, out)...)...).Output()
		if err != nil {
			t.Fatalf("ffprobe: %v", err)
		}
		return strings.TrimSpace(string(b))
	}
	v := probe("-select_streams", "v:0", "-show_entries", "stream=codec_name,pix_fmt,height,profile,level", "-of", "default=noprint_wrappers=1")
	for _, want := range []string{"codec_name=h264", "pix_fmt=yuv420p", "height=720", "profile=High", "level=40"} {
		if !strings.Contains(v, want) {
			t.Errorf("video stream: want %s in\n%s", want, v)
		}
	}
	if a := probe("-select_streams", "a:0", "-show_entries", "stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1"); a != "aac" {
		t.Errorf("audio codec = %q, want aac", a)
	}
	keys := probe("-select_streams", "v:0", "-skip_frame", "nokey", "-show_entries", "frame=pts_time", "-of", "csv=p=0")
	got := strings.Fields(keys)
	if len(got) != 3 || !strings.HasPrefix(got[1], "4.") || !strings.HasPrefix(got[2], "8.") {
		t.Errorf("keyframes at %v, want 0/4/8 s", got)
	}
}
