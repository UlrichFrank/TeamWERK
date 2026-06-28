package config

import (
	"testing"
)

// TestLoad_VideoStreamSecretFailFast verifies the production fail-fast for the
// HLS stream secret: in production mode (LOG_FORMAT != "text") an empty
// VIDEO_STREAM_SECRET must abort startup, while local mode (LOG_FORMAT=text)
// tolerates it.
func TestLoad_VideoStreamSecretFailFast(t *testing.T) {
	// JWT_SECRET is required by Load regardless; set it for all cases.
	t.Setenv("JWT_SECRET", "test-jwt")

	t.Run("production without stream secret fails", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test-jwt")
		t.Setenv("LOG_FORMAT", "json")
		t.Setenv("VIDEO_STREAM_SECRET", "")
		if _, err := Load(); err == nil {
			t.Fatal("expected error for missing VIDEO_STREAM_SECRET in production")
		}
	})

	t.Run("production with stream secret succeeds", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test-jwt")
		t.Setenv("LOG_FORMAT", "json")
		t.Setenv("VIDEO_STREAM_SECRET", "s3cr3t")
		if _, err := Load(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("local mode tolerates missing stream secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test-jwt")
		t.Setenv("LOG_FORMAT", "text")
		t.Setenv("VIDEO_STREAM_SECRET", "")
		if _, err := Load(); err != nil {
			t.Fatalf("local mode should tolerate empty secret, got %v", err)
		}
	})
}
