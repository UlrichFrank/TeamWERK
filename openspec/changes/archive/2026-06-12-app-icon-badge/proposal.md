## Why

Wenn TeamWERK als PWA auf dem Homescreen installiert ist, gibt es keinen visuellen Hinweis auf ausstehende Aktionen — der Nutzer muss die App öffnen, um zu sehen ob etwas wartet. Das `navigator.setAppBadge(n)`-API erlaubt es, eine Zahl direkt auf dem App-Icon anzuzeigen (wie bei nativen Apps), ohne dass die App geöffnet sein muss.

## What Changes

- Service Worker setzt bei eingehender Push-Notification das Badge via `self.registration.setAppBadge(count)`
- Badge-Zahl wird im Push-Payload vom Backend mitgeliefert
- Badge wird auf 0 zurückgesetzt wenn die App geöffnet wird (`navigator.setAppBadge(0)` in AppShell)
- Backend berechnet den Badge-Zähler pro User: Summe aus offenen Dienst-Slots + unbeantworteten RSVPs
- Neuer API-Endpunkt `GET /api/push/badge-count` liefert die aktuelle Zahl für den eingeloggten User

## Capabilities

### New Capabilities

- `app-icon-badge`: Badge-Zähler auf dem PWA-Icon zeigt Anzahl ausstehender Aktionen

### Modified Capabilities

- `push-notifications`: Push-Payload wird um ein `badge`-Zahlenfeld erweitert
- `service-worker`: SW setzt/löscht Badge bei Push-Events und App-Focus

## Impact

- `internal/notifications/handler.go`: Badge-Count-Berechnung + neuer GET-Endpunkt
- `web/src/sw.ts`: `setAppBadge` bei push event
- `web/src/components/AppShell.tsx`: `setAppBadge(0)` beim Öffnen der App
- `cmd/teamwerk/main.go`: Route für Badge-Count-Endpunkt

## Scope / Abgrenzung

- Badge-Zähler ist eine Annäherung, keine exakte "ungelesen"-Markierung
- Zähler setzt sich beim Öffnen der App zurück (nicht beim Lesen einzelner Einträge)
- Nur sichtbar wenn PWA installiert ist — kein Fallback nötig (API ist no-op im Browser)
- iOS: funktioniert nur im Standalone-Modus (wie Push selbst)
