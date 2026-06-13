package auth

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const claimsKey contextKey = "claims"

func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string
			if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
				tokenStr = strings.TrimPrefix(header, "Bearer ")
			}
			if tokenStr == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := ParseAccessToken(secret, tokenStr)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CookieMiddleware authenticates SSE endpoints via the HttpOnly refresh-token cookie.
// It validates the cookie against the database and sets minimal Claims in the context.
// Use exclusively for the /api/events endpoint — never for regular API routes.
func CookieMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("refresh_token")
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			tokenHash := HashToken(cookie.Value)
			var userID int
			var email, role string
			var expiresAt time.Time
			err = db.QueryRowContext(r.Context(),
				`SELECT u.id, u.email, u.role, rt.expires_at
				 FROM refresh_tokens rt JOIN users u ON u.id = rt.user_id
				 WHERE rt.token_hash = ?`, tokenHash,
			).Scan(&userID, &email, &role, &expiresAt)
			if err != nil || time.Now().After(expiresAt) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims := &Claims{UserID: userID, Email: email, Role: role}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole gates access by system role (admin|standard). Use for admin-only endpoints.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromCtx(r.Context())
			if claims == nil || !allowed[claims.Role] {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireClubFunction gates access by Vereinsfunktion. Admins always pass through.
func RequireClubFunction(functions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromCtx(r.Context())
			if claims == nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			if claims.Role == "admin" {
				next.ServeHTTP(w, r)
				return
			}
			for _, f := range functions {
				if claims.HasFunction(f) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}

func ClaimsFromCtx(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}
