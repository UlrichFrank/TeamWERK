// Package httpx bündelt das Schreiben von HTTP-Antworten: einheitliche
// JSON-Bodies, maschinenlesbare Fehlercodes und die Log-Spur für Serverfehler.
//
// Der Kern ist WriteError: der Client bekommt ausschließlich einen kurzen Code,
// nie den Text eines Datenbank- oder Laufzeitfehlers; die Ursache landet
// stattdessen in einer strukturierten Log-Zeile mit Pfad, Methode und Nutzer-ID.
// Vorher beantworteten die Handler Serverfehler mit http.Error(w, err.Error(),
// 500) — der Client sah SQL-Interna, der Betrieb sah gar nichts.
//
// Bewusst kein Wrapper um http.Error: die verbleibenden http.Error-Aufrufe
// sollen per grep zählbar bleiben, solange die Migration der übrigen Domänen
// aussteht (design.md, Entscheidung 1).
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Generische Fehlercodes. Domänenspezifische Codes (range_in_past,
// rotation_requires_normal_behavior, …) bleiben Literale am Aufrufort — das
// Frontend erkennt sie an genau diesen Strings.
const (
	CodeInternal    = "internal"
	CodeInvalidID   = "invalid_id"
	CodeInvalidBody = "invalid_body"
	CodeNotFound    = "not_found"
	CodeForbidden   = "forbidden"
	CodeConflict    = "conflict"
	CodeValidation  = "validation"
)

// WriteJSON schreibt v als JSON-Body mit dem angegebenen Status.
//
// Ein Encode-Fehler kann nicht mehr an den Client gemeldet werden (Status und
// Header sind raus), deshalb bleibt nur die Log-Zeile.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("http json encode", "status", status, "err", err)
	}
}

// WriteError beantwortet den Request mit {"error": code} und hinterlässt für
// jeden 5xx eine Log-Zeile mit Ursache. err darf nil sein; sein Text erreicht
// den Client nie.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code string, err error) {
	if status >= 500 {
		slog.Error("http error",
			"code", code,
			"status", status,
			"method", method(r),
			"path", path(r),
			"user_id", userID(r),
			"err", err)
	} else if err != nil {
		slog.Debug("http error",
			"code", code,
			"status", status,
			"method", method(r),
			"path", path(r),
			"user_id", userID(r),
			"err", err)
	}
	WriteJSON(w, status, map[string]string{"error": code})
}

func method(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Method
}

func path(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}

// userID liefert die Nutzer-ID aus den Claims, 0 für nicht authentifizierte
// Requests (öffentliche Routen, abgelaufenes Token).
func userID(r *http.Request) int {
	if r == nil {
		return 0
	}
	if c := auth.ClaimsFromCtx(r.Context()); c != nil {
		return c.UserID
	}
	return 0
}

// PathID liest einen numerischen Pfadparameter (chi füllt r.PathValue).
// Der zweite Rückgabewert ist false, wenn der Parameter fehlt, keine Zahl ist
// oder nicht positiv — der Aufrufer antwortet dann mit CodeInvalidID.
func PathID(r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// Paging liest ?limit= und ?offset= und klemmt beide in den erlaubten Bereich:
// unbrauchbare oder fehlende Werte fallen auf defaultLimit bzw. 0 zurück, ein
// zu großes limit wird auf maxLimit gedeckelt (sonst zieht ein Client mit
// ?limit=100000 die ganze Tabelle).
func Paging(r *http.Request, defaultLimit, maxLimit int) (limit, offset int) {
	limit = defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
