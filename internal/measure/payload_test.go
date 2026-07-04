//go:build measure

package measure

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// routeResult is one row of the payload table.
type routeResult struct {
	Label  string
	Path   string
	Status int
	Bytes  int
}

// revalResult is one row of the revalidation table: first call vs. second call
// (with If-None-Match of the first call's ETag, if any).
type revalResult struct {
	Label   string
	Path    string
	Status1 int
	Bytes1  int
	ETag    string
	Status2 int
	Bytes2  int
}

// measuredRoutes returns the heavy/list GET routes to size, with path params
// resolved from the seeded data. All are reachable with an admin Bearer token.
func measuredRoutes(data *measureData) []struct{ label, path string } {
	return []struct{ label, path string }{
		{"kader", "/api/kader"},
		{"duty-slots", "/api/duty-slots"},
		{"duty-board", "/api/duty-board"},
		{"duty-types", "/api/duty-types"},
		{"games", "/api/games"},
		{"game-participants", fmt.Sprintf("/api/games/%d/participants", data.gameT1)},
		// Explicit from/to spanning the seeded window: the endpoint's default
		// range is relative to wall-clock now, which would exclude the fixed
		// measureRefTime-anchored sessions and make the baseline non-deterministic.
		{"training-sessions", fmt.Sprintf("/api/training-sessions?from=%s&to=%s", dayOffset(-60), dayOffset(60))},
		{"chat-messages", fmt.Sprintf("/api/chat/conversations/%d/messages", data.convID)},
		{"teams", "/api/teams"},
		{"team-names", "/api/teams/names"},
		{"seasons", "/api/seasons"},
		{"venues", "/api/venues"},
		{"age-class-rules", "/api/age-class-rules"},
		{"encryption-pubkey", "/api/encryption-pubkey"},
		{"vapid-public-key", "/api/push/vapid-public-key"},
	}
}

// referenceRoutes are the quasi-static routes whose HTTP-cache behaviour we
// revalidate (second request with If-None-Match). On main they carry no ETag,
// so the second call is a full 200 — that is the baseline this harness records.
func referenceRoutes() []struct{ label, path string } {
	return []struct{ label, path string }{
		{"seasons", "/api/seasons"},
		{"teams", "/api/teams"},
		{"venues", "/api/venues"},
		{"age-class-rules", "/api/age-class-rules"},
		{"duty-types", "/api/duty-types"},
		{"encryption-pubkey", "/api/encryption-pubkey"},
		{"vapid-public-key", "/api/push/vapid-public-key"},
	}
}

// httpGet performs a GET with a Bearer token and optional If-None-Match, and
// returns status, response-body length in bytes, and the ETag header.
func httpGet(t *testing.T, baseURL, path, token, ifNoneMatch string) (status, nbytes int, etag string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		t.Fatalf("new request %s: %v", path, err)
	}
	req.Header.Set("Authorization", token)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body %s: %v", path, err)
	}
	return res.StatusCode, len(body), res.Header.Get("ETag")
}

func measurePayloads(t *testing.T, baseURL, token string, data *measureData) []routeResult {
	t.Helper()
	var out []routeResult
	for _, r := range measuredRoutes(data) {
		status, n, _ := httpGet(t, baseURL, r.path, token, "")
		out = append(out, routeResult{Label: r.label, Path: r.path, Status: status, Bytes: n})
	}
	return out
}

func measureRevalidation(t *testing.T, baseURL, token string) []revalResult {
	t.Helper()
	var out []revalResult
	for _, r := range referenceRoutes() {
		s1, b1, etag := httpGet(t, baseURL, r.path, token, "")
		s2, b2, _ := httpGet(t, baseURL, r.path, token, etag)
		out = append(out, revalResult{
			Label: r.label, Path: r.path,
			Status1: s1, Bytes1: b1, ETag: etag, Status2: s2, Bytes2: b2,
		})
	}
	return out
}

func TestMeasure_RecordsPayloadPerRoute(t *testing.T) {
	db := testutil.NewDB(t)
	data := measureSeed(t, db)
	baseURL := startServer(t, data)

	results := measurePayloads(t, baseURL, data.adminToken, data)
	if len(results) != len(measuredRoutes(data)) {
		t.Fatalf("expected %d results, got %d", len(measuredRoutes(data)), len(results))
	}
	for _, r := range results {
		if r.Status == 0 {
			t.Errorf("%s (%s): no HTTP status recorded", r.Label, r.Path)
		}
		if r.Status >= 500 {
			t.Errorf("%s (%s): server error status %d — measurement route is broken", r.Label, r.Path, r.Status)
		}
		if r.Status == http.StatusOK && r.Bytes <= 0 {
			t.Errorf("%s (%s): 200 but non-positive byte count %d", r.Label, r.Path, r.Bytes)
		}
	}

	reval := measureRevalidation(t, baseURL, data.adminToken)
	for _, r := range reval {
		if r.Status1 == 0 || r.Status2 == 0 {
			t.Errorf("%s: revalidation did not record both calls (s1=%d s2=%d)", r.Label, r.Status1, r.Status2)
		}
	}
}
