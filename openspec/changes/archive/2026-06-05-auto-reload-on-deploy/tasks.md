## 1. Backend: Build-Hash einbetten

- [x] 1.1 In `cmd/teamwerk/main.go` eine package-level Variable `var buildHash = "dev"` deklarieren
- [x] 1.2 In `Makefile` den `build`-Target um `-ldflags "-X 'main.buildHash=$(shell git rev-parse --short HEAD)'"` erweitern
- [x] 1.3 Den `buildHash` über einen neuen Parameter in `hub.NewHandler(h *EventHub, buildHash string)` durchreichen und in `hub.Handler` als Feld speichern
- [x] 1.4 In `main.go` `hub.NewHandler(hubInstance, buildHash)` aufrufen

## 2. Backend: SSE-Init-Event senden

- [x] 2.1 In `hub/handler.go` nach dem `Subscribe()`-Aufruf und vor der Hauptschleife sofort `fmt.Fprintf(w, "data: __version:%s\n\n", h.buildHash)` + `flusher.Flush()` einfügen

## 3. Frontend: `useLiveUpdates` filtert `__version:`-Events heraus

- [x] 3.1 In `web/src/hooks/useLiveUpdates.ts` den `onmessage`-Handler so erweitern, dass Events mit `__version:`-Prefix nicht an `onEvent` weitergeleitet werden (stille Filterung, kein Callback)

## 4. Frontend: `useVersionCheck`-Hook

- [x] 4.1 Neue Datei `web/src/hooks/useVersionCheck.ts` anlegen
- [x] 4.2 Hook öffnet eine eigene `EventSource`-Verbindung zu `/api/events?token=...`
- [x] 4.3 Beim ersten `__version:`-Event: Hash als `knownVersion`-Ref speichern, kein State-Update
- [x] 4.4 Bei jedem folgenden `__version:`-Event: wenn Hash ≠ `knownVersion` → `setUpdateAvailable(true)`
- [x] 4.5 Im Dev-Modus (`import.meta.env.DEV`) sofort returnen ohne EventSource zu öffnen
- [x] 4.6 Hook gibt `updateAvailable: boolean` zurück

## 5. Frontend: `UpdateBanner`-Component

- [x] 5.1 Neue Datei `web/src/components/UpdateBanner.tsx` anlegen
- [x] 5.2 Banner ist `fixed bottom-0 left-0 right-0`, brand-yellow Hintergrund, brand-black Text
- [x] 5.3 Enthält Text „Neue Version verfügbar", Button „Jetzt neu laden", Dismiss-Button mit `<X>`-Icon
- [x] 5.4 Buttons haben `py-2.5 sm:py-2` für 44px Touch-Target auf Mobile
- [x] 5.5 Component nimmt Props `onDismiss: () => void` und `onReload: () => void`

## 6. Frontend: SW-Update-Erkennung via `useRegisterSW`

- [x] 6.1 In `web/src/App.tsx` `useRegisterSW` aus `virtual:pwa-register/react` importieren
- [x] 6.2 `useRegisterSW({ onNeedRefresh() { setSwUpdateAvailable(true) } })` aufrufen, `updateServiceWorker`-Funktion aus dem Rückgabewert entnehmen
- [x] 6.3 `swUpdateAvailable`-State anlegen; bei `true` denselben `UpdateBanner` anzeigen; Reload-Button ruft `updateServiceWorker(true)` auf

## 7. Frontend: Intelligenter Reload-Handler

- [x] 7.1 Hilfsfunktion `reloadWithSwActivation()` in `web/src/lib/reload.ts` anlegen: prüft via `navigator.serviceWorker.getRegistration()` ob ein neuer SW bereits im `waiting`-State ist; wenn ja: `postMessage({ type: 'SKIP_WAITING' })` + auf `controllerchange` warten, dann `location.reload()`; sonst direkt `location.reload()`
- [x] 7.2 In `sw.ts` einen `message`-Listener für `{ type: 'SKIP_WAITING' }` ergänzen, der `self.skipWaiting()` aufruft

## 8. Frontend: Alles in `App.tsx` verdrahten

- [x] 8.1 `useVersionCheck()` in `App.tsx` einbinden, `updateAvailable`-State halten
- [x] 8.2 `UpdateBanner` rendern wenn `updateAvailable || swUpdateAvailable`
- [x] 8.3 Reload-Button ruft `reloadWithSwActivation()` auf (sowohl für SSE- als auch SW-Pfad)
- [x] 8.4 Dismiss-Handler setzt den jeweiligen State auf `false`
- [x] 8.5 Sicherstellen dass der Banner über dem restlichen Content liegt (`z-50`)
