package progress

import (
	"testing"
	"time"
)

// 1000 Meldungen innerhalb von 100 ms → nur die erste passiert (plus die finale).
func TestThrottle_BurstIsCollapsed(t *testing.T) {
	clock := time.Unix(0, 0)
	th := NewThrottle(200 * time.Millisecond)
	th.now = func() time.Time { return clock }

	passed := 0
	for i := 0; i < 1000; i++ {
		if th.Allow(false) {
			passed++
		}
		clock = clock.Add(100 * time.Microsecond)
	}
	if passed != 1 {
		t.Fatalf("expected 1 update in a 100 ms burst, got %d", passed)
	}
	if !th.Allow(true) {
		t.Fatal("final update must always pass")
	}
	clock = clock.Add(250 * time.Millisecond)
	if !th.Allow(false) {
		t.Fatal("update after the interval must pass")
	}
}

func TestRemaining(t *testing.T) {
	if got := Remaining(10*time.Minute, 0.5); got != 10*time.Minute {
		t.Fatalf("got %v", got)
	}
	if Remaining(2*time.Second, 0.5) != 0 || Remaining(time.Hour, 0.001) != 0 || Remaining(time.Hour, 1) != 0 {
		t.Fatal("unreliable estimates must be 0")
	}
}

func TestFormatRemaining(t *testing.T) {
	cases := map[time.Duration]string{
		0:                               "",
		30 * time.Second:                "noch unter 1 min",
		12*time.Minute + 20*time.Second: "noch ca. 12 min",
		time.Hour + 5*time.Minute:       "noch ca. 1 h 05 min",
	}
	for d, want := range cases {
		if got := FormatRemaining(d); got != want {
			t.Errorf("FormatRemaining(%v) = %q, want %q", d, got, want)
		}
	}
}
