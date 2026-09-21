package gamestats

import (
	"context"
	"time"
)

// pollTimeout deckt einen Staffel-Lauf ab: ein Katalog-Abruf, ein
// Spielplan-Abruf und bis zu einigen Dutzend Berichts-Downloads mit Pause.
const pollTimeout = 15 * time.Minute

// detachedContext liefert einen Context, der die HTTP-Antwort überlebt.
//
// Arbeit nach der Antwort darf NICHT an r.Context() hängen: der ist mit der
// Antwort gecancelt, und der Abruf scheiterte still (Betriebshärtung Welle 2).
func detachedContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), pollTimeout)
}
