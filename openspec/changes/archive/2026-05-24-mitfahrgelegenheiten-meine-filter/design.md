## Context

Die Mitfahrgelegenheiten-Seite zeigt alle Spiele mit Carpooling-Einträgen. Nutzer mit vielen Spielen (mehrere Teams, ganzer Kalender) sehen alle Spiele auf einmal — auch solche, bei denen sie selbst nicht eingetragen sind. Die API gibt bereits `isOwn: true/false` auf jedem Eintrag zurück, sodass der Filter rein client-seitig umsetzbar ist.

Vergleichspunkt: `DutyPage.tsx` hat denselben Toggle („Alle Dienste / Meine Dienste"), der dort einen Query-Parameter `?view=mine` setzt. Bei Mitfahrgelegenheiten ist kein Backend-Parameter nötig, da die Daten bereits vollständig geladen werden.

## Goals / Non-Goals

**Goals:**
- Toggle-Button-Gruppe „Alle | Meine" oben rechts neben `<h1>`, sichtbar für alle Rollen
- Im Modus „Meine": Spiele ausblenden, bei denen `[...biete, ...suche].every(e => !e.isOwn)`
- Tab-Counts (Auswärtsspiele / Heimspiele / Events) zeigen im „Meine"-Modus die gefilterte Anzahl

**Non-Goals:**
- Keine Backend-Änderungen oder neuer Query-Parameter
- Kein Persistieren des Filter-States (kein localStorage, kein URL-Param)
- Keine Änderung an der Ladereihenfolge oder Caching

## Decisions

**Client-seitiger Filter statt Backend-Filter**
Der `isOwn`-Flag ist bereits im API-Response. Ein Backend-Query-Parameter würde eine neue Route oder Kondition im Handler erfordern ohne zusätzlichen Nutzen, da alle Spiele ohnehin geladen werden. Konsequenz: Beim ersten Laden werden immer alle Spiele übertragen — bei typischen Kalendergrößen (< 50 Spiele pro Saison) ist das unbedeutend.

**Kein URL-State**
Der Filter wird nicht in der URL gespeichert (z.B. `?view=mine`). Die Seite ist keine Deep-Link-Destination; der State zurückzusetzen beim Navigieren weg ist das erwartete Verhalten.

**Filterung vor Tab-Filterung**
Die Filterung nach `viewMine` wird auf `response.games` angewendet, bevor die Tab-Filterung nach `eventType` greift. Das stellt sicher, dass die Tab-Counts korrekt sind:

```
filteredGames = viewMine
  ? response.games.filter(d => [...d.biete, ...d.suche].some(e => e.isOwn))
  : response.games

tabGames = filteredGames.filter(d => d.game.eventType === activeTab)
countForTab(tab) = filteredGames.filter(d => d.game.eventType === tab).length
```

## Risks / Trade-offs

**Veraltete Daten nach Speichern** → kein neues Risiko; das bestehende `load()` nach Speichern/Löschen aktualisiert die Daten bereits.

**„Meine"-Tab leer nach Löschen des letzten Eintrags** → Korrekt: Spiel verschwindet aus der gefilterten Liste. Das ist das erwartete Verhalten.
