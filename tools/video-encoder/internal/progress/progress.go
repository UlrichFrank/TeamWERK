// Package progress drosselt Fortschrittsmeldungen auf eine UI-verträgliche Rate
// und schätzt die Restzeit.
//
// Der tus-Upload meldet pro gelesenem Block (tausende Male pro Sekunde); jede
// Meldung ungebremst in die UI zu reichen hieße, den Render-Thread mit
// Neuzeichnen zu fluten (dieselbe Fehlerklasse wie der frühere Browser-Upload,
// der dadurch ein Vielfaches langsamer wurde).
package progress

import (
	"fmt"
	"sync"
	"time"
)

// Throttle lässt höchstens eine Meldung pro Interval durch.
type Throttle struct {
	interval time.Duration
	now      func() time.Time

	mu   sync.Mutex
	last time.Time
}

// NewThrottle baut eine Drossel mit dem gegebenen Mindestabstand.
func NewThrottle(interval time.Duration) *Throttle {
	return &Throttle{interval: interval, now: time.Now}
}

// Allow meldet, ob eine Meldung jetzt durchgelassen wird. final (Abschluss)
// passiert immer, damit die Anzeige nicht bei 99 % stehen bleibt.
func (t *Throttle) Allow(final bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	if !final && !t.last.IsZero() && now.Sub(t.last) < t.interval {
		return false
	}
	t.last = now
	return true
}

// Remaining schätzt die Restzeit linear aus bisheriger Dauer und Anteil. Unter
// 1 % oder nach weniger als 5 s ist die Schätzung zu wacklig; dann 0.
func Remaining(elapsed time.Duration, fraction float64) time.Duration {
	if fraction < 0.01 || fraction >= 1 || elapsed < 5*time.Second {
		return 0
	}
	return time.Duration(float64(elapsed) * (1 - fraction) / fraction)
}

// FormatRemaining formatiert eine Restzeit für die Anzeige („noch ca. 12 min").
// 0 ergibt einen leeren String.
func FormatRemaining(d time.Duration) string {
	switch {
	case d <= 0:
		return ""
	case d < time.Minute:
		return "noch unter 1 min"
	case d < time.Hour:
		return fmt.Sprintf("noch ca. %d min", int(d.Round(time.Minute)/time.Minute))
	default:
		h := int(d / time.Hour)
		m := int((d % time.Hour).Round(time.Minute) / time.Minute)
		return fmt.Sprintf("noch ca. %d h %02d min", h, m)
	}
}
