package videos

import "testing"

func TestPathHelpers(t *testing.T) {
	const root = "/storage/videos"
	const id = 42

	if got, want := RawPath(root, id), "/storage/videos/raw/42.mp4"; got != want {
		t.Errorf("RawPath = %q, want %q", got, want)
	}
	if got, want := ProcessedDir(root, id), "/storage/videos/processed/42"; got != want {
		t.Errorf("ProcessedDir = %q, want %q", got, want)
	}
	if got, want := MasterManifestPath(root, id), "/storage/videos/processed/42/master.m3u8"; got != want {
		t.Errorf("MasterManifestPath = %q, want %q", got, want)
	}
	if got, want := RenditionDir(root, id, "720p"), "/storage/videos/processed/42/720p"; got != want {
		t.Errorf("RenditionDir = %q, want %q", got, want)
	}
}
