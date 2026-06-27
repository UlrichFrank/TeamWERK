# Tasks

Ein Commit pro Task (Conventional Commits, Scope `pwa`).

> **Hinweis:** Bei der Implementierung festgestellt, dass es bereits einen
> `useMediaQuery`-Hook gibt (`web/src/lib/useMediaQuery.ts`) und
> `DocumentsPage` bereits eine `isMobile`-Weiche nutzt (Commit `0f75c13`).
> Ein zusätzlicher `viewport.ts`-Helper entfällt; wir nutzen den vorhandenen
> Hook konsistent in beiden Aufrufern.

## 1. DocumentsPage.openFile — Desktop-Branch auf nativen Viewer umstellen

- [x] 1.1 In `web/src/pages/DocumentsPage.tsx` den Desktop-Branch (`if (!isMobile)`) umstellen: statt `navigate('/dokumente/anzeigen/...')` jetzt das Pre-Change-Verhalten: `window.open('about:blank', '_blank')` **synchron im Click** (gegen Popup-Blocker) → `api.get('/files/:id/download-token')` → `tab.location.href = '/api/files/:id/download?token=...'`. Bei Fehler `tab.close()` + `setFileError(...)`.
- [x] 1.2 `web/src/pages/__tests__/DocumentsPage.openFile.test.tsx`: bestehende Tests prüfen + neue Tests ergänzen — Desktop-Pfad (`window.open` wird gerufen, `tab.location.href` gesetzt mit Download-URL), Desktop-Token-Fehler (`tab.close()`).

## 2. DocumentFileLinkPage — Viewport-Weiche

- [x] 2.1 In `web/src/pages/DocumentFileLinkPage.tsx`: `useMediaQuery('(max-width: 639px)')` einführen. Mobile → bisheriges Verhalten (`<Navigate to="/dokumente/anzeigen/:fileId" replace />`). Desktop → `useEffect` mit Token-Fetch + `window.location.replace('/api/files/:id/download?token=...')`. 403/404 → Fehler-UI mit Zurück-Link nach `/dokumente`.
- [x] 2.2 `web/src/pages/__tests__/DocumentFileLinkPage.test.tsx` neu anlegen: Mobile-Redirect, Desktop-`window.location.replace`-Pfad (mit `Object.defineProperty(window, 'location', ...)`-Mock), Desktop-403 → Fehler-UI.

## 3. Verifikation

- [x] 3.1 `pnpm -C web lint` + `pnpm -C web test` grün.
- [x] 3.2 `pnpm -C web build` grün.
- [x] 3.3 `openspec validate desktop-native-document-viewer --strict` grün.
- [ ] 3.4 Lokale Manual-Verifikation: Desktop-Browser → Klick auf Dokument öffnet nativen PDF-Viewer im neuen Tab; Mobile-Simulator (DevTools Responsive ≤ 639 px) → wie vorher (nativer Viewer via Download-Trick).

## 4. Archivierung

- [ ] 4.1 Nach Merge: `openspec archive desktop-native-document-viewer`.
