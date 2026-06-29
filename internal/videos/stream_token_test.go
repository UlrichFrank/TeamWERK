package videos

import (
	"errors"
	"strings"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
)

func tokenHandler(secret string) *Handler {
	return &Handler{cfg: &appconfig.Config{VideoStreamSecret: secret}}
}

func TestStreamToken_ValidRoundTrip(t *testing.T) {
	h := tokenHandler("super-secret")
	tok, err := h.Sign(42, 7)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	uid, err := h.Verify(tok, 42)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if uid != 7 {
		t.Errorf("uid = %d, want 7", uid)
	}
}

func TestStreamToken_Expired(t *testing.T) {
	secret := "super-secret"
	// exp 1 second in the past relative to a fixed now.
	fixed := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	tok := signStreamToken(secret, 42, 7, fixed.Add(-time.Second).Unix())

	_, err := verifyStreamToken(secret, tok, 42, fixed.Unix())
	if !errors.Is(err, ErrExpiredStreamToken) {
		t.Fatalf("expected ErrExpiredStreamToken, got %v", err)
	}
}

func TestStreamToken_NotYetExpiredBoundary(t *testing.T) {
	secret := "super-secret"
	fixed := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	// exp exactly one second in the future ⇒ still valid.
	tok := signStreamToken(secret, 42, 7, fixed.Add(time.Second).Unix())
	if _, err := verifyStreamToken(secret, tok, 42, fixed.Unix()); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestStreamToken_WrongVid(t *testing.T) {
	h := tokenHandler("super-secret")
	tok, err := h.Sign(42, 7)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := h.Verify(tok, 99); !errors.Is(err, ErrInvalidStreamToken) {
		t.Fatalf("wrong vid: expected ErrInvalidStreamToken, got %v", err)
	}
}

func TestStreamToken_Tampered(t *testing.T) {
	secret := "super-secret"
	now := time.Now().Add(time.Hour).Unix()
	tok := signStreamToken(secret, 42, 7, now)

	// Flip a character in the signature part.
	enc, sig, _ := strings.Cut(tok, ".")
	var tamperedSig string
	if sig[0] == 'A' {
		tamperedSig = "B" + sig[1:]
	} else {
		tamperedSig = "A" + sig[1:]
	}
	tampered := enc + "." + tamperedSig
	if _, err := verifyStreamToken(secret, tampered, 42, time.Now().Unix()); !errors.Is(err, ErrInvalidStreamToken) {
		t.Fatalf("tampered sig: expected ErrInvalidStreamToken, got %v", err)
	}

	// Tamper the payload (uid) without re-signing.
	other := signStreamToken(secret, 42, 999, now)
	otherEnc, _, _ := strings.Cut(other, ".")
	mixed := otherEnc + "." + sig // payload of "other", signature of original
	if _, err := verifyStreamToken(secret, mixed, 42, time.Now().Unix()); !errors.Is(err, ErrInvalidStreamToken) {
		t.Fatalf("tampered payload: expected ErrInvalidStreamToken, got %v", err)
	}
}

func TestStreamToken_WrongSecret(t *testing.T) {
	tok := signStreamToken("secret-a", 42, 7, time.Now().Add(time.Hour).Unix())
	if _, err := verifyStreamToken("secret-b", tok, 42, time.Now().Unix()); !errors.Is(err, ErrInvalidStreamToken) {
		t.Fatalf("wrong secret: expected ErrInvalidStreamToken, got %v", err)
	}
}

func TestStreamToken_EmptySecret(t *testing.T) {
	h := tokenHandler("")
	if _, err := h.Sign(1, 1); err == nil {
		t.Fatal("Sign with empty secret should error")
	}
	if _, err := verifyStreamToken("", "x.y", 1, time.Now().Unix()); !errors.Is(err, ErrInvalidStreamToken) {
		t.Fatalf("empty secret verify: expected ErrInvalidStreamToken, got %v", err)
	}
}

func TestStreamToken_Malformed(t *testing.T) {
	secret := "super-secret"
	now := time.Now().Unix()
	cases := []string{"", "noseparator", ".", "abc.", ".abc", "!!!.@@@"}
	for _, c := range cases {
		if _, err := verifyStreamToken(secret, c, 42, now); !errors.Is(err, ErrInvalidStreamToken) {
			t.Errorf("malformed %q: expected ErrInvalidStreamToken, got %v", c, err)
		}
	}
}
