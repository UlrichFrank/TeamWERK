## Why

Nach jedem Deployment muss die App manuell neu gestartet werden — und die Update-Benachrichtigung erscheint nicht zuverlässig, weil ein Bug im SSE-Reconnect den Banner dauerhaft unterdrückt. Außerdem gibt es keine Möglichkeit zu kontrollieren, welche Version gerade im Browser läuft.

## What Changes

- SSE-Reconnect-Bug in `useVersionCheck.ts` behoben: `onerror`-Handler entfernt, der `es.close()` zu früh aufruft und automatische Reconnection verhindert
- `useVersionCheck`-Hook gibt zusätzlich die aktuell laufende Version (`version: string | null`) zurück
- Sidebar-Footer zeigt die Version unauffällig an, abgetrennt durch eine Trennlinie

## Capabilities

### New Capabilities

- `version-display`: Anzeige der laufenden App-Version im Sidebar-Footer (unterhalb E-Mail/Abmelden, durch Linie abgesetzt)

### Modified Capabilities

- `sse-version-check`: Bestehender Hook wird um Rückgabe der `version` erweitert und der Reconnect-Bug behoben

## Impact

- `web/src/hooks/useVersionCheck.ts` — Hook-Interface ändert sich (gibt jetzt Objekt statt boolean zurück)
- `web/src/App.tsx` — Konsument des Hooks muss angepasst werden
- `web/src/components/AppShell.tsx` — Version im Footer anzeigen
