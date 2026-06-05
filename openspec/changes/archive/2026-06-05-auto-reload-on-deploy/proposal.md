## Why

Nach `make deploy` laufen verbundene Clients weiter mit dem alten JS-Bundle und dem alten API-Kontext — sie merken nicht, dass Server und Frontend aktualisiert wurden. Das erzwingt manuelles Abmelden, Force-Reload oder „Reload from origin", besonders in der installierten PWA.

## What Changes

- Build-Hash (Git Short-SHA) wird zur Compile-Zeit in das Go-Binary eingebettet und beim SSE-Connect als `__version`-Event an den Client gesendet
- Neuer `useVersionCheck`-Hook erkennt Versionsänderungen beim SSE-Reconnect (der nach jedem Server-Neustart automatisch erfolgt)
- Neuer `UpdateBanner`-Component zeigt „Neue Version verfügbar — Jetzt neu laden" am unteren Bildschirmrand
- Service-Worker-Update-Erkennung via `useRegisterSW(onNeedRefresh)` als zweite Erkennungslinie für reine Frontend-Deployments

## Capabilities

### New Capabilities

- `deploy-version-detection`: Erkennt nach `make deploy` automatisch, dass eine neue Version verfügbar ist, und fordert den Nutzer per Banner zur Aktualisierung auf — sowohl im Browser als auch in der installierten PWA.

### Modified Capabilities

- `sse-live-updates`: Der SSE-Endpoint sendet beim Verbindungsaufbau zusätzlich ein `__version:<hash>`-Event, bevor reguläre Mutations-Events gesendet werden.
- `pwa-support`: Der Service Worker meldet erkannte Updates per `onNeedRefresh`-Callback an die App statt sie lautlos zu ignorieren.

## Impact

- **Makefile**: `BUILD_HASH`-Variable via `-ldflags` in `go build` eingebaut
- **`cmd/teamwerk/main.go`**: `var buildHash string` als ldflags-Target
- **`internal/hub/handler.go`**: Sendet `data: __version:<hash>\n\n` beim Connect
- **`web/src/hooks/useVersionCheck.ts`**: Neuer Hook (lauscht auf `__version`-Events)
- **`web/src/components/UpdateBanner.tsx`**: Neuer Banner-Component
- **`web/src/App.tsx`**: Bindet `useVersionCheck` + `useRegisterSW` ein, rendert `UpdateBanner`
