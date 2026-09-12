package background_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/background"
)

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("waitFor: Bedingung nicht innerhalb 1s erfüllt")
}

// TestGo_PanicWirdAbgefangenGezaehltUndGeloggt (2.1): ein Panic in der über Go
// gestarteten Funktion darf weder den Test-Prozess noch — im Betrieb — den
// Serverprozess beenden. Er muss stattdessen strukturiert geloggt werden
// (inkl. Job-Name) und die Metrik erhöhen.
func TestGo_PanicWirdAbgefangenGezaehltUndGeloggt(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	before := background.PanicsTotal()
	background.Go("test.job-xyz", func() {
		panic("kaboom")
	})

	waitFor(t, func() bool { return background.PanicsTotal() > before })

	if got := background.PanicsTotal(); got != before+1 {
		t.Errorf("erwartet PanicsTotal=%d nach einem Panic, bekam %d", before+1, got)
	}
	logged := buf.String()
	if !strings.Contains(logged, "test.job-xyz") {
		t.Errorf("Log soll den Job-Namen enthalten, bekam:\n%s", logged)
	}
	if !strings.Contains(logged, "background panic") {
		t.Errorf("Log soll die Panic-Meldung enthalten, bekam:\n%s", logged)
	}
}

// TestGo_FolgeGoroutineLaeuftNachPanicWeiter belegt, dass ein Panic in einer
// über Go gestarteten Funktion andere Goroutinen (und damit den Prozess) nicht
// mitreißt.
func TestGo_FolgeGoroutineLaeuftNachPanicWeiter(t *testing.T) {
	done := make(chan struct{})
	background.Go("test.after-panic", func() {
		panic("boom")
	})
	background.Go("test.survivor", func() {
		close(done)
	})
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Folge-Goroutine hat nicht signalisiert")
	}
}

// TestGo_OhnePanicLaeuftFnDurch stellt sicher, dass der Wrapper den
// Erfolgsfall nicht verändert (kein versehentliches Schlucken des Aufrufs).
func TestGo_OhnePanicLaeuftFnDurch(t *testing.T) {
	done := make(chan struct{})
	background.Go("test.ok", func() {
		close(done)
	})
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("fn wurde nicht ausgeführt")
	}
}
