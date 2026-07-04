package hub_test

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// drain reads channel ch until an event arrives or the deadline passes.
// It returns ("", false) on timeout.
func recvWithin(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

// TestHub_BroadcastToUsers_OnlyTargets verifies that BroadcastToUsers reaches
// exactly the addressed users' channels and no others.
func TestHub_BroadcastToUsers_OnlyTargets(t *testing.T) {
	h := hub.NewHub()
	chA := h.SubscribeUser(1)
	chB := h.SubscribeUser(2)
	chC := h.SubscribeUser(3)

	h.BroadcastToUsers([]int{1, 2}, "members")

	if ev, ok := recvWithin(chA, 200*time.Millisecond); !ok || ev != "members" {
		t.Errorf("user 1 (target) should receive event, got %q ok=%v", ev, ok)
	}
	if ev, ok := recvWithin(chB, 200*time.Millisecond); !ok || ev != "members" {
		t.Errorf("user 2 (target) should receive event, got %q ok=%v", ev, ok)
	}
	if ev, ok := recvWithin(chC, 100*time.Millisecond); ok {
		t.Errorf("user 3 (non-target) must NOT receive event, got %q", ev)
	}
}

// TestHub_BroadcastToUsers_Deduplicates verifies a user listed twice in the
// audience receives the event only once per stream.
func TestHub_BroadcastToUsers_Deduplicates(t *testing.T) {
	h := hub.NewHub()
	ch := h.SubscribeUser(1)

	h.BroadcastToUsers([]int{1, 1}, "members")

	if ev, ok := recvWithin(ch, 200*time.Millisecond); !ok || ev != "members" {
		t.Fatalf("expected one event, got %q ok=%v", ev, ok)
	}
	if ev, ok := recvWithin(ch, 100*time.Millisecond); ok {
		t.Errorf("duplicate audience entry must not deliver twice, got extra %q", ev)
	}
}

// TestHub_Broadcast_ReachesPerUserStreams verifies that the global Broadcast
// still reaches per-user subscribers — vereinsweite Topics stay global even
// though /api/events now subscribes per user.
func TestHub_Broadcast_ReachesPerUserStreams(t *testing.T) {
	h := hub.NewHub()
	chA := h.SubscribeUser(1)
	chB := h.SubscribeUser(2)

	h.Broadcast("settings")

	if ev, ok := recvWithin(chA, 200*time.Millisecond); !ok || ev != "settings" {
		t.Errorf("user 1 should receive global event, got %q ok=%v", ev, ok)
	}
	if ev, ok := recvWithin(chB, 200*time.Millisecond); !ok || ev != "settings" {
		t.Errorf("user 2 should receive global event, got %q ok=%v", ev, ok)
	}
}

// TestEvents_SubscribesPerUser verifies that a BroadcastToUser event addressed
// to user A reaches A's /api/events stream but not a foreign user's stream.
func TestEvents_SubscribesPerUser(t *testing.T) {
	db := testutil.NewDB(t)
	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")
	tokenA := testutil.CreateRefreshToken(t, db, userA)
	tokenB := testutil.CreateRefreshToken(t, db, userB)

	sharedHub := hub.NewHub()
	h := hub.NewHandler(sharedHub, "test-hash", auth.UserIDFromCtx)
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieMiddleware(db))
		r.Get("/api/events", h.Events)
	})
	srv := httptest.NewServer(r)

	streamA := openSSE(t, srv.URL, tokenA)
	streamB := openSSE(t, srv.URL, tokenB)
	// Cancel both streams (which unblocks the server handlers' r.Context().Done())
	// BEFORE closing the test server — otherwise srv.Close() would block forever
	// waiting for the two long-lived SSE handlers to return.
	defer func() {
		streamA.close()
		streamB.close()
		srv.Close()
	}()

	// Wait until both streams are registered in the hub (2 distinct users).
	waitForUsers(t, sharedHub, 2)

	sharedHub.BroadcastToUser(userA, "members")

	if got := readSSEData(t, streamA.br, time.Second); got != "members" {
		t.Errorf("user A stream expected 'members', got %q", got)
	}
	if got := readSSEData(t, streamB.br, 300*time.Millisecond); got != "" {
		t.Errorf("user B stream must not receive A's event, got %q", got)
	}
}

// sseStream is an open /api/events connection with a cancel func to tear it down
// deterministically (cancelling the request unblocks the server handler so the
// test server can close).
type sseStream struct {
	br     *bufio.Reader
	cancel context.CancelFunc
	body   io.Closer
}

func (s *sseStream) close() {
	s.cancel()
	s.body.Close()
}

// openSSE opens an /api/events stream authenticated by the given refresh-token
// cookie and returns it positioned after the initial __version line.
func openSSE(t *testing.T, baseURL, refreshToken string) *sseStream {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/events", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatalf("open SSE: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		cancel()
		resp.Body.Close()
		t.Fatalf("open SSE: status %d", resp.StatusCode)
	}
	br := bufio.NewReader(resp.Body)
	// Consume the initial "data: __version:..." line.
	if v := readSSEData(t, br, 2*time.Second); v == "" {
		cancel()
		resp.Body.Close()
		t.Fatalf("open SSE: did not receive initial __version line")
	}
	return &sseStream{br: br, cancel: cancel, body: resp.Body}
}

// readSSEData reads the next `data: <payload>` line within d, returning the
// payload (without the "data: " prefix). Returns "" on timeout. Comment lines
// (": ping") are skipped.
func readSSEData(t *testing.T, br *bufio.Reader, d time.Duration) string {
	t.Helper()
	type res struct {
		line string
		err  error
	}
	out := make(chan res, 1)
	go func() {
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				out <- res{"", err}
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(line, "data: ") {
				out <- res{strings.TrimPrefix(line, "data: "), nil}
				return
			}
			// skip blank lines and ": ping" comments
		}
	}()
	select {
	case r := <-out:
		return r.line
	case <-time.After(d):
		return ""
	}
}

// waitForUsers polls the hub until n distinct users are subscribed or fails.
func waitForUsers(t *testing.T, h *hub.EventHub, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if h.SubscribedUserCount() >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d subscribed users (have %d)", n, h.SubscribedUserCount())
}
