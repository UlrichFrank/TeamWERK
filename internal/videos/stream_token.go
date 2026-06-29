package videos

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Stream-Token (siehe design.md "Stream-Token"):
//
// HLS löst pro Video viele HTTP-Requests aus; hls.js kann keine Bearer-Header
// senden. Statt JWT-Auth signieren wir daher einen kurzlebigen, HMAC-gesicherten
// Token (HS256/HMAC-SHA256), den alle HLS-Requests als ?st=… mitführen. Die
// Verifikation ist DB-frei und millisekundenschnell.
//
// Format (kompakt, url-safe): base64url("vid.uid.exp") + "." + base64url(HMAC).
// Der HMAC wird über den ersten (base64url-)Teil gebildet, sodass Manipulation an
// vid/uid/exp die Signatur bricht.

// streamTokenTTL ist die Gültigkeitsdauer eines frisch signierten Tokens.
const streamTokenTTL = time.Hour

// now ist injizierbar, damit Tests Ablauf-Logik ohne Sleeps prüfen können.
var now = time.Now

var (
	// ErrInvalidStreamToken steht für jeden strukturellen/Signatur-/Claim-Fehler.
	ErrInvalidStreamToken = errors.New("videos: invalid stream token")
	// ErrExpiredStreamToken steht speziell für einen abgelaufenen (sonst gültigen)
	// Token, damit Aufrufer/Tests den Ablauf vom Tampering unterscheiden können.
	ErrExpiredStreamToken = errors.New("videos: expired stream token")
)

var b64 = base64.RawURLEncoding

// signStreamToken erzeugt einen Token für (vid, uid) mit explizitem Ablauf-exp
// (Unix-Sekunden). Reine Funktion ohne Uhr — gut testbar.
func signStreamToken(secret string, vid, uid int, exp int64) string {
	payload := fmt.Sprintf("%d.%d.%d", vid, uid, exp)
	enc := b64.EncodeToString([]byte(payload))
	sig := streamHMAC(secret, enc)
	return enc + "." + b64.EncodeToString(sig)
}

// Sign signiert einen Stream-Token für das Video vid und den Nutzer uid mit
// einem Ablauf von now()+1h. Ein leeres Secret ist ein Konfigurationsfehler.
func (h *Handler) Sign(vid, uid int) (string, error) {
	if h.cfg.VideoStreamSecret == "" {
		return "", ErrInvalidStreamToken
	}
	exp := now().Add(streamTokenTTL).Unix()
	return signStreamToken(h.cfg.VideoStreamSecret, vid, uid, exp), nil
}

// verifyStreamToken prüft Signatur, Bindung an wantVID und Ablauf gegen nowUnix.
// Reine Funktion (Uhr als Parameter) — gut testbar.
func verifyStreamToken(secret, token string, wantVID int, nowUnix int64) (uid int, err error) {
	if secret == "" {
		return 0, ErrInvalidStreamToken
	}
	enc, sigPart, ok := strings.Cut(token, ".")
	if !ok || enc == "" || sigPart == "" {
		return 0, ErrInvalidStreamToken
	}
	gotSig, err := b64.DecodeString(sigPart)
	if err != nil {
		return 0, ErrInvalidStreamToken
	}
	wantSig := streamHMAC(secret, enc)
	if subtle.ConstantTimeCompare(gotSig, wantSig) != 1 {
		return 0, ErrInvalidStreamToken
	}
	// Signatur ok ⇒ Payload ist authentisch und sicher zu parsen.
	raw, err := b64.DecodeString(enc)
	if err != nil {
		return 0, ErrInvalidStreamToken
	}
	parts := strings.Split(string(raw), ".")
	if len(parts) != 3 {
		return 0, ErrInvalidStreamToken
	}
	vid, err1 := strconv.Atoi(parts[0])
	uidVal, err2 := strconv.Atoi(parts[1])
	exp, err3 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, ErrInvalidStreamToken
	}
	if vid != wantVID {
		return 0, ErrInvalidStreamToken
	}
	if nowUnix >= exp {
		return 0, ErrExpiredStreamToken
	}
	return uidVal, nil
}

// Verify prüft einen Token gegen das erwartete Video vid und die aktuelle Zeit
// und liefert bei Erfolg die uid des Claims zurück.
func (h *Handler) Verify(token string, vid int) (uid int, err error) {
	return verifyStreamToken(h.cfg.VideoStreamSecret, token, vid, now().Unix())
}

func streamHMAC(secret, msg string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return mac.Sum(nil)
}
