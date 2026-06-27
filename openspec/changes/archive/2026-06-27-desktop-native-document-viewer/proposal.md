## Why

Die letzte Änderung (`pwa-in-app-file-viewer`) hat das Datei-Öffnen für alle
Viewports auf den In-App-Viewer (pdf.js / `<img>`) umgestellt. Auf dem Handy
ist das die richtige Lösung — die PWA-Standalone-Sackgasse ist weg. **Auf dem
Desktop ist das eine Regression:** der native Browser-PDF-Viewer (Chrome,
Firefox, Safari) bringt mehr Komfort (Volltextsuche, Drucken, Speichern,
schnelleres Rendering) und Desktop-Browser haben sowieso Browser-Chrome mit
Tab/Zurück — die Sackgasse, die wir auf Mobile fürchten, existiert dort nicht.

Vor der Umstellung war Desktop gut. Genau dorthin wollen wir zurück, ohne den
Mobile-Fortschritt zu verlieren.

## What Changes

Eine **Viewport-basierte Weiche** an genau zwei Aufrufer-Stellen — der
`sm:`-Breakpoint (640 px), den die Codebase als einzige Mobile/Desktop-Grenze
nutzt (CLAUDE.md `docs/agent/05-frontend.md`).

**Neuer Helper** `web/src/lib/viewport.ts`:
```ts
export function isMobileViewport(): boolean {
  return window.matchMedia('(max-width: 639px)').matches
}
```

**`DocumentsPage.openFile`** verzweigt:
- `isMobileViewport()` → `navigate('/dokumente/anzeigen/${file.id}')` (Status quo).
- sonst → Pre-Change-Verhalten zurückholen: `window.open('about:blank', '_blank')`
  *im* Click (gegen Popup-Blocker) → Token holen → `tab.location.href = downloadUrl`.

**`DocumentFileLinkPage`** verzweigt im `useEffect`:
- `isMobileViewport()` → `<Navigate to="/dokumente/anzeigen/${fileId}" replace />`.
- sonst → Token holen → `window.location.replace(downloadUrl)` (Pre-Change-Code).

**Unverändert:**
- `<FileViewer>`, `<FileViewerPage>` (Route + Komponente bleiben für Mobile).
- `<SepaMandatViewerPage>` — clientseitig entschlüsselt, **muss** in-app
  bleiben, viewport-unabhängig.
- Backend, Routen, Auth, Tests für unbeteiligte Komponenten.

## Scope

**In scope:**
- `lib/viewport.ts` mit `isMobileViewport()`.
- Branch in `DocumentsPage.openFile`.
- Branch in `DocumentFileLinkPage`.
- Test-Anpassungen für die zwei Branches + Unit-Test für den Helper.

**Out of scope:**
- SEPA-Mandat-Pfad (bleibt in-app, unabhängig vom Viewport).
- Entfernung des In-App-Viewers oder von pdfjs-dist (wird weiter gebraucht).
- display-mode-Detektion / standalone-PWA-Sonderlogik (`sm:` reicht; Desktop-PWA-Installs sind irrelevant).
- Tablet-Sonderbehandlung (iPad Landscape ≥ 640 px → nativer Viewer; Safari hat Chrome, kein Problem).

## Test-Anforderungen

| Komponente / Helper | Testname | Erwartetes Ergebnis |
|---|---|---|
| `isMobileViewport()` | `viewport_returnsTrueWhenSmall` | matchMedia `(max-width: 639px) matches: true` → `true` |
| `isMobileViewport()` | `viewport_returnsFalseWhenWide` | matchMedia matches: false → `false` |
| `DocumentsPage.openFile` (Mobile) | `openFile_mobileNavigatesToInAppViewer` | matchMedia mock matches:true → `navigate('/dokumente/anzeigen/:id')`, `window.open` nicht aufgerufen |
| `DocumentsPage.openFile` (Desktop) | `openFile_desktopOpensNewTabWithDownloadUrl` | matchMedia mock matches:false → `window.open('about:blank', '_blank')`, danach Token-Fetch und `tab.location.href` gesetzt |
| `DocumentsPage.openFile` (Desktop, Token-Fehler) | `openFile_desktopClosesTabOnTokenError` | Token-API antwortet 500 → `tab.close()` aufgerufen, Fehler-State gesetzt |
| `DocumentFileLinkPage` (Mobile) | `linkPage_mobileRedirectsToInAppViewer` | `<Navigate replace>` auf `/dokumente/anzeigen/:fileId` |
| `DocumentFileLinkPage` (Desktop) | `linkPage_desktopReplacesLocationWithDownloadUrl` | Token-Fetch, dann `window.location.replace` mit `/api/files/:id/download?token=…` |
| `DocumentFileLinkPage` (Desktop, 403) | `linkPage_desktopShowsErrorOn403` | Fehler-UI „Kein Zugriff", Zurück-Link verfügbar |
