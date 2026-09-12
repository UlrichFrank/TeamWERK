package httpx_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/httpx"
)

// captureLogs lenkt den Default-Logger für die Dauer des Tests in einen Buffer
// um und setzt ihn danach zurück.
func captureLogs(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func requestWithUser(t *testing.T, method, target string, userID int) *http.Request {
	t.Helper()
	r := httptest.NewRequest(method, target, nil)
	if userID != 0 {
		r = r.WithContext(auth.ContextWithClaims(r.Context(), &auth.Claims{UserID: userID, Role: "standard"}))
	}
	return r
}

// TestWriteError_ServerErrorLoggedNotLeaked: ein 5xx hinterlässt eine Log-Zeile
// mit Pfad, Methode und Nutzer-ID; der Body trägt nur den Code, nie den
// Fehlertext (der hier absichtlich nach SQL aussieht).
func TestWriteError_ServerErrorLoggedNotLeaked(t *testing.T) {
	buf := captureLogs(t, slog.LevelInfo)

	w := httptest.NewRecorder()
	r := requestWithUser(t, http.MethodPost, "/api/games/7/participants", 42)
	httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal,
		errors.New("sql: no such table: games"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected JSON content type, got %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v (%q)", err, w.Body.String())
	}
	if body["error"] != httpx.CodeInternal {
		t.Errorf(`expected {"error":"internal"}, got %q`, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "sql") || strings.Contains(w.Body.String(), "no such") {
		t.Errorf("error text leaked into body: %q", w.Body.String())
	}

	logged := buf.String()
	for _, want := range []string{"/api/games/7/participants", "POST", "user_id=42", "no such table"} {
		if !strings.Contains(logged, want) {
			t.Errorf("log line misses %q: %s", want, logged)
		}
	}
}

// TestWriteError_ServerErrorWithoutClaims: ohne Authentifizierung steht 0 im
// Log statt einer Panic auf nil-Claims.
func TestWriteError_ServerErrorWithoutClaims(t *testing.T) {
	buf := captureLogs(t, slog.LevelInfo)

	w := httptest.NewRecorder()
	r := requestWithUser(t, http.MethodGet, "/api/games", 0)
	httpx.WriteError(w, r, http.StatusInternalServerError, httpx.CodeInternal, errors.New("boom"))

	if !strings.Contains(buf.String(), "user_id=0") {
		t.Errorf("expected user_id=0 in log, got: %s", buf.String())
	}
}

// TestWriteError_ClientErrorNotLoggedAsError: 4xx erzeugt keine Error-Zeile —
// ein falsch getippter Pfad ist kein Betriebsvorfall.
func TestWriteError_ClientErrorNotLoggedAsError(t *testing.T) {
	buf := captureLogs(t, slog.LevelInfo)

	w := httptest.NewRecorder()
	r := requestWithUser(t, http.MethodGet, "/api/games/abc", 42)
	httpx.WriteError(w, r, http.StatusBadRequest, httpx.CodeInvalidID, errors.New("strconv: bad"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"error":"invalid_id"}` {
		t.Errorf(`expected {"error":"invalid_id"}, got %q`, got)
	}
	if buf.Len() != 0 {
		t.Errorf("4xx must not produce a log line at info level, got: %s", buf.String())
	}
}

// TestWriteError_DomainCodePassesThrough: domänenspezifische Codes bleiben
// unverändert — das Frontend erkennt sie an genau diesem String.
func TestWriteError_DomainCodePassesThrough(t *testing.T) {
	w := httptest.NewRecorder()
	r := requestWithUser(t, http.MethodPost, "/api/duty-slots/bulk-regen/apply", 1)
	httpx.WriteError(w, r, http.StatusBadRequest, "range_in_past", nil)

	if got := strings.TrimSpace(w.Body.String()); got != `{"error":"range_in_past"}` {
		t.Errorf("domain code mangled: %q", got)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	httpx.WriteJSON(w, http.StatusCreated, map[string]int{"id": 5})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected JSON content type, got %q", ct)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"id":5}` {
		t.Errorf("unexpected body %q", got)
	}
}

// TestWriteJSON_NilBody: Status ohne Inhalt (204) schreibt keinen Body.
func TestWriteJSON_NilBody(t *testing.T) {
	w := httptest.NewRecorder()
	httpx.WriteJSON(w, http.StatusNoContent, nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
}

func TestPaging(t *testing.T) {
	cases := []struct {
		query      string
		wantLimit  int
		wantOffset int
		name       string
	}{
		{"", 50, 0, "Defaults ohne Parameter"},
		{"?limit=10&offset=20", 10, 20, "gültige Werte unverändert"},
		{"?limit=100000", 200, 0, "limit auf maxLimit gedeckelt"},
		{"?limit=0", 50, 0, "limit=0 fällt auf Default"},
		{"?limit=-5", 50, 0, "negatives limit fällt auf Default"},
		{"?limit=abc", 50, 0, "unparsbares limit fällt auf Default"},
		{"?offset=-1", 50, 0, "negatives offset wird 0"},
		{"?offset=abc", 50, 0, "unparsbares offset wird 0"},
		{"?limit=200&offset=1000", 200, 1000, "maxLimit exakt erlaubt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/games"+tc.query, nil)
			limit, offset := httpx.Paging(r, 50, 200)
			if limit != tc.wantLimit || offset != tc.wantOffset {
				t.Errorf("Paging(%q) = (%d,%d), want (%d,%d)", tc.query, limit, offset, tc.wantLimit, tc.wantOffset)
			}
		})
	}
}

func TestPathID(t *testing.T) {
	cases := []struct {
		value  string
		wantID int
		wantOK bool
	}{
		{"7", 7, true},
		{"", 0, false},
		{"abc", 0, false},
		{"0", 0, false},
		{"-3", 0, false},
		{"1.5", 0, false},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodGet, "/api/games/x", nil)
		r.SetPathValue("id", tc.value)
		id, ok := httpx.PathID(r, "id")
		if id != tc.wantID || ok != tc.wantOK {
			t.Errorf("PathID(%q) = (%d,%v), want (%d,%v)", tc.value, id, ok, tc.wantID, tc.wantOK)
		}
	}
}
