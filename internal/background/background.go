// Package background startet Hintergrundarbeit (Benachrichtigungs-Fan-out,
// asynchrone Nebenläufer außerhalb des HTTP-Pfads) in einer eigenen Goroutine,
// ohne dass ein Panic darin den gesamten Serverprozess mitreißt
// (Betriebshärtung Welle 2, design.md Entscheidung 2 — das Pendant zu
// health.Recoverer für den HTTP-Pfad).
//
// Jede vom Domänencode gestartete Goroutine MUSS über Go (oder die dünne
// notify.SendAsync-Fassade darüber) laufen. Ausnahmen (Hub-Kanal-Fan-out,
// der Video-Transcode-Worker mit eigenem ctx-Shutdown) sind bewusst außerhalb
// der Prüfung — siehe internal/arch/goroutine_test.go.
package background

import (
	"log/slog"
	"runtime/debug"
	"sync/atomic"
)

// panicsTotal zählt abgefangene Background-Job-Panics seit Prozessstart.
// Exponiert als teamwerk_background_panics_total (internal/health), neben
// teamwerk_panics_total (HTTP-Handler-Panics via health.Recoverer) — zwei
// getrennte Zähler für zwei getrennte Pfade.
var panicsTotal atomic.Int64

// Go startet fn in einer eigenen Goroutine. Ein Panic in fn wird abgefangen,
// strukturiert geloggt (inkl. Stacktrace) und in panicsTotal gezählt statt den
// Prozess zu beenden. name identifiziert den Job in Log und Metrik (z. B.
// "notify.Send", "chat.pushFn") — konventionell "<Paket>.<Zweck>".
func Go(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("background panic",
					"job", name,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				panicsTotal.Add(1)
			}
		}()
		fn()
	}()
}

// PanicsTotal liefert die Anzahl bisher abgefangener Background-Job-Panics.
func PanicsTotal() int64 { return panicsTotal.Load() }
