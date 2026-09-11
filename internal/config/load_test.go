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
	// Muss >= 32 Byte sein (config.Load lehnt kürzere Secrets ab).
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes-long")

	t.Run("production without stream secret fails", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes-long")
		t.Setenv("LOG_FORMAT", "json")
		t.Setenv("VIDEO_STREAM_SECRET", "")
		if _, err := Load(); err == nil {
			t.Fatal("expected error for missing VIDEO_STREAM_SECRET in production")
		}
	})

	t.Run("production with stream secret succeeds", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes-long")
		t.Setenv("LOG_FORMAT", "json")
		t.Setenv("VIDEO_STREAM_SECRET", "s3cr3t")
		if _, err := Load(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("local mode tolerates missing stream secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes-long")
		t.Setenv("LOG_FORMAT", "text")
		t.Setenv("VIDEO_STREAM_SECRET", "")
		if _, err := Load(); err != nil {
			t.Fatalf("local mode should tolerate empty secret, got %v", err)
		}
	})
}

// TestConfig_BaseURLDefault verifies that without a BASE_URL environment
// variable the configured base URL defaults to the primary hostname
// teamwerk.team-stuttgart.org (nicht mehr internal.*).
func TestConfig_BaseURLDefault(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-bytes-long")
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("BASE_URL", "") // empty forces getEnv fallback

	c, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "https://teamwerk.team-stuttgart.org"; c.BaseURL != want {
		t.Fatalf("BaseURL = %q, want %q", c.BaseURL, want)
	}
}

// TestLoad_JWTSecretMinLength: Load lehnt ein zu kurzes JWT_SECRET ab
// (security-haertung-welle-1, Entscheidung 6) — ein 16-Byte-Secret ist per
// Brute-Force gegen HMAC angreifbar.
func TestLoad_JWTSecretMinLength(t *testing.T) {
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("JWT_SECRET", "0123456789abcdef") // 16 Byte

	if _, err := Load(); err == nil {
		t.Fatal("expected error for JWT_SECRET shorter than 32 bytes")
	}
}

// TestLoad_JWTSecretRejectsExampleValue: der unveränderte Platzhalter aus
// .env.example darf den Server nicht produktiv machen — er steht im Klartext
// im Repo und ist damit jedem bekannt.
func TestLoad_JWTSecretRejectsExampleValue(t *testing.T) {
	t.Setenv("LOG_FORMAT", "text")
	t.Setenv("JWT_SECRET", "change-me-to-a-random-secret")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for JWT_SECRET equal to the .env.example placeholder")
	}
}

// TestLoad_JWTSecretAccepts48ByteValue: ein per `openssl rand -base64 48`
// erzeugtes Secret (Referenzbefehl aus der Fehlermeldung/.env.example) wird
// akzeptiert.
func TestLoad_JWTSecretAccepts48ByteValue(t *testing.T) {
	t.Setenv("LOG_FORMAT", "text")
	// 48 zufällige Bytes, base64-kodiert (Länge > 32 Byte als String).
	t.Setenv("JWT_SECRET", "kQ8sSTn3z2m9pXqv7Yc4Ld6Rj1Wb0Fh5Ae2Nt8Gy9Uo3Ix6Pk7Vz1Cs4Bm=")

	if _, err := Load(); err != nil {
		t.Fatalf("unexpected error for valid 32+ byte JWT_SECRET: %v", err)
	}
}
