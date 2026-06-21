package health_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/health"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// healthzServer mounts the public healthz route exactly like router.go (no auth).
func healthzServer(t *testing.T, h *health.Handler) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/api/healthz", h.Healthz)
	r.Get("/api/metrics", h.Metrics)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func decodeHealthz(t *testing.T, res *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		t.Fatalf("decode healthz body: %v", err)
	}
	res.Body.Close()
	return m
}

func TestHealthz_OK(t *testing.T) {
	db := testutil.NewDB(t)
	srv := healthzServer(t, health.NewHandler(db, "", ""))

	res, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatalf("GET healthz: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	body := decodeHealthz(t, res)
	if body["status"] != "ok" || body["db"] != "ok" {
		t.Fatalf("body = %v, want status=ok db=ok", body)
	}
}

func TestHealthz_DBDown(t *testing.T) {
	db := testutil.NewDB(t)
	db.Close() // simulate an unreachable database

	srv := healthzServer(t, health.NewHandler(db, "", ""))
	res, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatalf("GET healthz: %v", err)
	}
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.StatusCode)
	}
	if body := decodeHealthz(t, res); body["db"] != "fail" {
		t.Fatalf("db = %v, want fail", body["db"])
	}
}

func TestHealthz_NoAuthRequired(t *testing.T) {
	db := testutil.NewDB(t)
	srv := healthzServer(t, health.NewHandler(db, "", ""))

	// No Authorization header at all — public tier.
	res, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatalf("GET healthz: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 without token", res.StatusCode)
	}
}

func TestHealthz_SchedulerAgeReported(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := db.Exec(`INSERT INTO monitoring_heartbeat (id, updated_at) VALUES (1, ?)`,
		time.Now().Add(-30*time.Second).UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("seed heartbeat: %v", err)
	}
	srv := healthzServer(t, health.NewHandler(db, "", ""))

	res, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatalf("GET healthz: %v", err)
	}
	body := decodeHealthz(t, res)
	age, ok := body["scheduler_age_sec"].(float64)
	if !ok {
		t.Fatalf("scheduler_age_sec missing/not a number: %v", body["scheduler_age_sec"])
	}
	if age < 25 || age > 120 {
		t.Fatalf("scheduler_age_sec = %v, want ~30", age)
	}
}

func TestMetrics_RequiresToken(t *testing.T) {
	db := testutil.NewDB(t)

	// Token unset ⇒ endpoint disabled (404).
	srvOff := healthzServer(t, health.NewHandler(db, "", ""))
	res, err := http.Get(srvOff.URL + "/api/metrics")
	if err != nil {
		t.Fatalf("GET metrics (off): %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when METRICS_TOKEN unset", res.StatusCode)
	}

	// Token set but missing/wrong on request ⇒ 401.
	srvOn := healthzServer(t, health.NewHandler(db, "", "s3cret"))
	res, err = http.Get(srvOn.URL + "/api/metrics")
	if err != nil {
		t.Fatalf("GET metrics (no token): %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 without bearer token", res.StatusCode)
	}
}

func TestMetrics_ExposesSignals(t *testing.T) {
	db := testutil.NewDB(t)
	srv := healthzServer(t, health.NewHandler(db, "", "s3cret"))

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/metrics", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET metrics: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(res.Body)
	res.Body.Close()
	body := buf.String()
	for _, want := range []string{
		"teamwerk_disk_free_ratio",
		"teamwerk_scheduler_age_seconds",
		"teamwerk_panics_total",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics body missing %q\n---\n%s", want, body)
		}
	}
}

func TestRecover_Panic_IncrementsCounterAndRecovers(t *testing.T) {
	before := health.PanicsTotal()

	handler := health.Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := health.PanicsTotal(); got != before+1 {
		t.Fatalf("PanicsTotal = %d, want %d (panic must be counted)", got, before+1)
	}
	// Reaching this line proves the process did not crash.
}

func TestRecover_Panic_StructuredLog(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	handler := health.Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("kaboom")
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/boom", nil))

	logged := buf.String()
	if !strings.Contains(logged, `"event":"panic"`) {
		t.Fatalf("structured log missing event=panic\n---\n%s", logged)
	}
}
