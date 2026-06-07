package files

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const downloadTokenTTL = 5 * time.Minute

type downloadTokenPayload struct {
	FileID    int   `json:"fid"`
	UserID    int   `json:"uid"`
	ExpiresAt int64 `json:"exp"`
}

func generateDownloadToken(fileID, userID int, secret string) string {
	p := downloadTokenPayload{
		FileID:    fileID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(downloadTokenTTL).Unix(),
	}
	data, _ := json.Marshal(p)
	encoded := base64.RawURLEncoding.EncodeToString(data)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return encoded + "." + sig
}

func validateDownloadToken(token string, fileID int, secret string) (userID int, err error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid token format")
	}
	encoded, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encoded))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return 0, fmt.Errorf("invalid signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, fmt.Errorf("invalid encoding")
	}

	var p downloadTokenPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return 0, fmt.Errorf("invalid payload")
	}

	if time.Now().Unix() > p.ExpiresAt {
		return 0, fmt.Errorf("token expired")
	}

	if p.FileID != fileID {
		return 0, fmt.Errorf("file id mismatch")
	}

	return p.UserID, nil
}
