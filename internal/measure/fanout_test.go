//go:build measure

package measure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// sseObserver reads one client's /api/events stream, signalling readiness on
// the first (__version:) frame and counting subsequent real events.
type sseObserver struct {
	label     string
	mu        sync.Mutex
	events    []string
	connected chan struct{}
	closeOnce sync.Once
}

func (o *sseObserver) markConnected() { o.closeOnce.Do(func() { close(o.connected) }) }

func (o *sseObserver) add(ev string) {
	o.mu.Lock()
	o.events = append(o.events, ev)
	o.mu.Unlock()
}

func (o *sseObserver) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.events)
}

// fanoutResult is one row of the fan-out table.
type fanoutResult struct {
	Mutation       string
	ClientsReached int            // clients that received >=1 event
	PerClient      map[string]int // label -> event count
}

// openSSE connects one observer to /api/events using the refresh_token cookie.
// The returned cancel func stops the stream goroutine.
func openSSE(t *testing.T, baseURL string, c fanoutClient) (*sseObserver, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	obs := &sseObserver{label: c.Label, connected: make(chan struct{})}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/events", nil)
	if err != nil {
		cancel()
		t.Fatalf("%s: new SSE request: %v", c.Label, err)
	}
	req.Header.Set("Cookie", "refresh_token="+c.Cookie)

	go func() {
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return // context cancelled or transport closed
		}
		defer res.Body.Close()
		scanner := bufio.NewScanner(res.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			if strings.HasPrefix(payload, "__version:") {
				obs.markConnected()
				continue
			}
			obs.add(payload)
		}
	}()
	return obs, cancel
}

// httpPut sends a JSON PUT with a Bearer token and returns the status code.
func httpPut(t *testing.T, baseURL, path, token string, body any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body for %s: %v", path, err)
		}
	}
	req, err := http.NewRequest(http.MethodPut, baseURL+path, &buf)
	if err != nil {
		t.Fatalf("new PUT %s: %v", path, err)
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", path, err)
	}
	res.Body.Close()
	return res.StatusCode
}

// measureFanoutForMutation opens fresh SSE connections for all roster clients,
// runs mutate() once all are subscribed, waits a delivery window, and returns
// the per-client / clients-reached counts.
func measureFanoutForMutation(t *testing.T, baseURL string, roster []fanoutClient, mutation string, mutate func() int) fanoutResult {
	t.Helper()
	obs := make([]*sseObserver, len(roster))
	cancels := make([]context.CancelFunc, len(roster))
	for i, c := range roster {
		obs[i], cancels[i] = openSSE(t, baseURL, c)
	}
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()

	// Wait until every client is subscribed (received the __version frame).
	deadline := time.After(5 * time.Second)
	for i, o := range obs {
		select {
		case <-o.connected:
		case <-deadline:
			t.Fatalf("mutation %s: client %s did not subscribe in time", mutation, roster[i].Label)
		}
	}

	if status := mutate(); status < 200 || status >= 300 {
		t.Fatalf("mutation %s: trigger returned HTTP %d (want 2xx)", mutation, status)
	}

	// Delivery window: the broadcast is in-process, so this is generous.
	time.Sleep(500 * time.Millisecond)

	res := fanoutResult{Mutation: mutation, PerClient: map[string]int{}}
	for i, o := range obs {
		n := o.count()
		res.PerClient[roster[i].Label] = n
		if n > 0 {
			res.ClientsReached++
		}
	}
	return res
}

// measureFanout runs the three pinned mutations and returns their fan-out rows.
func measureFanout(t *testing.T, baseURL string, data *measureData) []fanoutResult {
	t.Helper()
	return []fanoutResult{
		measureFanoutForMutation(t, baseURL, data.roster, "members (PUT /api/members/{C5}/status)", func() int {
			return httpPut(t, baseURL, fmt.Sprintf("/api/members/%d/status", data.c5MemberID), data.adminToken,
				map[string]string{"status": "aktiv"})
		}),
		measureFanoutForMutation(t, baseURL, data.roster, "games(T1) (PUT /api/games/{T1})", func() int {
			return httpPut(t, baseURL, fmt.Sprintf("/api/games/%d", data.gameT1), data.adminToken,
				map[string]any{
					"date": data.gameT1Date, "time": "18:00", "opponent": "Test Opponent",
					"team_ids": []int{data.teamT1}, "event_type": "heim",
				})
		}),
		measureFanoutForMutation(t, baseURL, data.roster, "settings (PUT /api/club)", func() int {
			return httpPut(t, baseURL, "/api/club", data.adminToken,
				map[string]string{"name": "TeamWERK", "address": "Teststraße 1"})
		}),
	}
}

func TestMeasure_SSEFanoutCountsPerClient(t *testing.T) {
	db := testutil.NewDB(t)
	data := measureSeed(t, db)
	baseURL := startServer(t, data)

	for _, r := range measureFanout(t, baseURL, data) {
		// On main the hub broadcasts globally, so every one of the 8 clients is
		// reached by each mutation. (After scoped-live-updates: 3 / 5 / 8.)
		if r.ClientsReached != len(data.roster) {
			t.Errorf("mutation %q: %d/%d clients reached, want %d (global broadcast)\n  per-client: %v",
				r.Mutation, r.ClientsReached, len(data.roster), len(data.roster), r.PerClient)
		}
	}
}

func TestMeasure_FanoutClientRosterIsFixed(t *testing.T) {
	db := testutil.NewDB(t)
	data := measureSeed(t, db)

	want := []struct {
		label    string
		role     string
		function string // "" for admin/parent
		team     string
	}{
		{"C1", "admin", "", ""},
		{"C2", "standard", "vorstand", ""},
		{"C3", "standard", "kassierer", ""},
		{"C4", "standard", "trainer", "T1"},
		{"C5", "standard", "spieler", "T1"},
		{"C6", "standard", "spieler", "T2"},
		{"C7", "standard", "spieler", "T3"},
		{"C8", "standard", "", "T1"}, // elternteil: parent user, no club function
	}
	if len(data.roster) != len(want) {
		t.Fatalf("roster size %d, want %d", len(data.roster), len(want))
	}
	for i, w := range want {
		c := data.roster[i]
		if c.Label != w.label || c.Role != w.role || c.Team != w.team {
			t.Errorf("client %d: got (%s,%s,team=%s), want (%s,%s,team=%s)",
				i, c.Label, c.Role, c.Team, w.label, w.role, w.team)
		}
		gotFn := ""
		if len(c.Functions) > 0 {
			gotFn = c.Functions[0]
		}
		if gotFn != w.function {
			t.Errorf("client %s: function %q, want %q", c.Label, gotFn, w.function)
		}
		if c.UserID == 0 || c.Cookie == "" {
			t.Errorf("client %s: missing userID/cookie", c.Label)
		}
	}
}
