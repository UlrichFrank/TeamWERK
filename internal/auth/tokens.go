package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 2 * 24 * time.Hour
)

type Claims struct {
	UserID        int      `json:"uid"`
	Email         string   `json:"email"`
	Role          string   `json:"role"`
	ClubFunctions []string `json:"club_functions"`
	IsParent      bool     `json:"is_parent"`
	jwt.RegisteredClaims
}

func (c *Claims) HasFunction(f string) bool {
	return slices.Contains(c.ClubFunctions, f)
}

func (c *Claims) HasAnyFunction(fns ...string) bool {
	for _, f := range fns {
		if slices.Contains(c.ClubFunctions, f) {
			return true
		}
	}
	return false
}

func (c *Claims) IsTrainerLike() bool {
	return c.HasFunction("trainer") || c.HasFunction("sportliche_leitung")
}

// CanOverrideRSVPCutoff returns true for users who may submit or change RSVP
// responses after the cutoff (T-2h für Trainings und Spiele). These users plan
// the squad and need to keep the attendance list realistic.
func (c *Claims) CanOverrideRSVPCutoff() bool {
	return c.Role == "admin" || c.HasFunction("vorstand") || c.IsTrainerLike()
}

func IssueAccessToken(secret string, userID int, email, role string, clubFunctions []string, isParent bool) (string, error) {
	if clubFunctions == nil {
		clubFunctions = []string{}
	}
	claims := Claims{
		UserID:        userID,
		Email:         email,
		Role:          role,
		ClubFunctions: clubFunctions,
		IsParent:      isParent,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseAccessToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func GenerateOpaqueToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plain = hex.EncodeToString(b)
	hash = HashToken(plain)
	return
}

func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func RefreshTokenExpiry() time.Time {
	return time.Now().Add(refreshTokenDuration)
}

func InvitationExpiry() time.Time {
	return time.Now().Add(48 * time.Hour)
}

func PasswordResetExpiry() time.Time {
	return time.Now().Add(1 * time.Hour)
}
