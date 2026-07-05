package settings

import (
	"net/http"
	"strings"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// authPathPrefix darf trotz aktivem Wartungsmodus mutieren, damit sich
// niemand aussperren kann (Login/Refresh/Logout und flankierende Reset-
// Flows). Als Prefix — nicht als Route-Enum — damit künftige Auth-Routen
// (`request-membership`, `reset-password`, …) automatisch mit erlaubt sind.
const authPathPrefix = "/api/auth/"

// MaintenanceMiddleware weist bei aktivem Wartungsmodus alle Mutations-
// Requests (POST/PUT/PATCH/DELETE) mit HTTP 503 ab. Ausnahmen:
//   - Requests unter dem Prefix /api/auth/ (Selbst-Aussperrschutz).
//   - Requester mit System-Rolle "admin" (kann Modus wieder ausschalten).
//
// GET/HEAD/OPTIONS bleiben unangetastet. Der 503-Response trägt den Header
// X-Maintenance-Mode: 1 und einen JSON-Body {"error":"maintenance_mode",…},
// damit der Frontend-Interceptor die Wartungs-Antwort sicher vom generischen
// 503 (Upstream-Timeout, LB-Fehler) unterscheiden kann.
//
// Die Middleware sitzt bewusst VOR den Auth-Middlewares, um auch unauthen-
// tifizierte Mutations-Endpoints erfassen zu können. Sie parsed die JWT-
// Claims dafür selbst mit auth.ParseAccessToken — Parse-Fehler werden
// toleriert (dann eben kein Admin, geblockt).
func MaintenanceMiddleware(store *Store, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutating(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if !store.MaintenanceMode() {
				next.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(r.URL.Path, authPathPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			if isAdmin(r, jwtSecret) {
				next.ServeHTTP(w, r)
				return
			}
			writeMaintenance503(w)
		})
	}
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

func isAdmin(r *http.Request, jwtSecret string) bool {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	tokenStr := strings.TrimPrefix(header, "Bearer ")
	if tokenStr == "" {
		return false
	}
	claims, err := auth.ParseAccessToken(jwtSecret, tokenStr)
	if err != nil || claims == nil {
		return false
	}
	return claims.Role == "admin"
}

func writeMaintenance503(w http.ResponseWriter) {
	w.Header().Set("X-Maintenance-Mode", "1")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":"maintenance_mode","message":"Wartungsmodus aktiv — Änderungen sind vorübergehend gesperrt."}`))
}
