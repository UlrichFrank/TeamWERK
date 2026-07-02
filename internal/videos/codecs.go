package videos

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Codec-String-Ableitung für die HLS-`CODECS`-Attribute im master.m3u8.
//
// Warum überhaupt dynamisch: die Video-Seite wäre bei fixen ffmpeg-Args stabil
// (typisch `avc1.640028` = H.264 High L4.0), aber die Audio-Seite ist es nicht.
// Bei `-c:a copy` reicht ffmpeg den Quell-Codec durch — HE-AAC (`mp4a.40.5`)
// oder LC (`mp4a.40.2`) je nach Aufnahmegerät. Ein hartcodierter String läge
// dann falsch und tvOS zeigt Ton ohne Bild (siehe design.md, video-tv-streaming).
//
// Format: `avc1.PPCCLL,mp4a.40.X`.
//   PP = Profile-ID hex (Baseline=42, Main=4D, High=64, High10=6E)
//   CC = Constraint-Byte hex (bei unseren Preset-Ausgaben 00)
//   LL = Level hex (Level × 10 als Dezimal → hex; 3.0=1E, 3.1=1F, 4.0=28, 4.1=29)
//   X  = AAC-Object-Type (LC=2, HE-AAC v1=5, HE-AAC v2=29)

// probeSegmentCodecs ermittelt den CODECS-String durch getrennte ffprobe-Aufrufe
// auf den Video- und den Audio-Stream des gegebenen TS-Segments und setzt sie zu
// `avc1.PPCCLL,mp4a.40.X` zusammen.
func probeSegmentCodecs(ctx context.Context, segmentPath string) (string, error) {
	vProfile, vLevel, err := ffprobeVideoProfileLevel(ctx, segmentPath)
	if err != nil {
		return "", fmt.Errorf("video probe: %w", err)
	}
	aProfile, err := ffprobeAudioProfile(ctx, segmentPath)
	if err != nil {
		return "", fmt.Errorf("audio probe: %w", err)
	}
	video, err := h264CodecString(vProfile, vLevel)
	if err != nil {
		return "", err
	}
	audio, err := aacCodecString(aProfile)
	if err != nil {
		return "", err
	}
	return video + "," + audio, nil
}

// ffprobeVideoProfileLevel ruft ffprobe für den ersten Video-Stream und liefert
// (profile, level_x10) — z.B. ("High", 40) für H.264 High @ L4.0.
func ffprobeVideoProfileLevel(ctx context.Context, path string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=profile,level",
		"-of", "default=noprint_wrappers=1:nokey=0",
		path)
	out, err := cmd.Output()
	if err != nil {
		return "", 0, err
	}
	var profile string
	var level int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "profile":
			profile = strings.TrimSpace(v)
		case "level":
			n, convErr := strconv.Atoi(strings.TrimSpace(v))
			if convErr == nil {
				level = n
			}
		}
	}
	if profile == "" || level == 0 {
		return "", 0, fmt.Errorf("ffprobe returned incomplete video info: %q", string(out))
	}
	return profile, level, nil
}

// ffprobeAudioProfile ruft ffprobe für den ersten Audio-Stream und liefert das
// Profil ("LC", "HE-AAC", "HE-AACv2").
func ffprobeAudioProfile(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=profile",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	profile := strings.TrimSpace(string(out))
	if profile == "" {
		return "", fmt.Errorf("ffprobe returned empty audio profile")
	}
	return profile, nil
}

// h264CodecString baut den `avc1.PPCCLL`-Substring aus ffprobe-Profil und
// -Level. Level ist die ffprobe-Konvention „Level × 10" (Level 4.0 → 40).
func h264CodecString(profile string, level int) (string, error) {
	var pp string
	switch strings.ToLower(profile) {
	case "baseline", "constrained baseline":
		pp = "42"
	case "main":
		pp = "4D"
	case "high":
		pp = "64"
	case "high 10":
		pp = "6E"
	default:
		return "", fmt.Errorf("unknown H.264 profile %q", profile)
	}
	// Constraint-Byte 00 — unsere ffmpeg-Args produzieren keine Constraint-Flags,
	// die ein anderes Byte rechtfertigen würden.
	return fmt.Sprintf("avc1.%s00%02X", pp, level), nil
}

// aacCodecString bildet den `mp4a.40.X`-Substring aus dem ffprobe-Audio-Profil.
func aacCodecString(profile string) (string, error) {
	switch strings.ToUpper(profile) {
	case "LC":
		return "mp4a.40.2", nil
	case "HE-AAC":
		return "mp4a.40.5", nil
	case "HE-AACV2":
		return "mp4a.40.29", nil
	default:
		return "", fmt.Errorf("unknown AAC profile %q", profile)
	}
}
