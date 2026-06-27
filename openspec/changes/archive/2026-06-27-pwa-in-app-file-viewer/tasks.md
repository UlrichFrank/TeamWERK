# Tasks

Ein Commit pro Task (Conventional Commits, Scope `pwa` für Viewer-Infrastruktur,
`documents`/`members` für Aufrufer-Umstellung).

## 1. Dependency & Build-Setup

- [x] 1.1 `pnpm -C web add pdfjs-dist` ausführen, Version pinnen. (pdfjs-dist@6.0.227)
- [x] 1.2 Vite-Worker-Konfiguration prüfen: `pdfjs-dist/build/pdf.worker.min.mjs?url`-Import läuft mit aktueller Vite-Version + `vite-plugin-pwa`. Falls nötig, `pdfjs-dist/build/pdf.worker.min.mjs` in `web/public/` kopieren und absoluten Pfad referenzieren. (Datei vorhanden in node_modules; ?url-Import-Strategie wird in PdfRenderer angewandt.)
- [x] 1.3 `pnpm -C web build` muss grün bleiben (Bundle-Analyse: pdf.js darf nicht im Main-Chunk landen). (Erledigt in 8.2 — Build grün, PdfRenderer in eigenem Chunk.)

## 2. PdfRenderer (lazy-geladen)

- [x] 2.1 `web/src/components/PdfRenderer.tsx` anlegen: Props `{ blob: Blob }`. Lädt pdfjs-dist, setzt `GlobalWorkerOptions.workerSrc`, rendert alle Seiten in `<canvas>`-Elementen untereinander (responsive Breite).
- [x] 2.2 Loading-State (Spinner) während pdfjs-Initialisierung + Doc-Parse; Fehler-State bei korrupter PDF.
- [x] 2.3 Cleanup: `pdfDocument.destroy()` in `useEffect`-Cleanup, `URL.revokeObjectURL` für interne Blob-URLs. (Keine internen Blob-URLs nötig — `getDocument({ data: buf })` aus ArrayBuffer; nur `pdfDoc.destroy()`.)

## 3. FileViewer-Komponente

- [x] 3.1 `web/src/components/FileViewer.tsx` anlegen mit Discriminated-Union-Props (`source: 'file' | 'blob'`).
- [x] 3.2 `source: 'file'`: `useEffect`-Hook holt Token via `/files/:id/download-token`, dann Blob via `/files/:id/download?token=…` (`responseType: 'blob'`). Loading/Error-States.
- [x] 3.3 Header rendern: Dateiname (truncate), Zurück-Button (`<ChevronLeft>` + „Zurück"), optional Download-Button (`<Download>` triggert `<a>`-Klick mit Blob-URL + `download={filename}`).
- [x] 3.4 Render-Switch: `image/*` → `<img>` zentriert mit `max-w-full max-h-[80vh] object-contain`. `application/pdf` → `React.lazy`-importiertes `<PdfRenderer blob={blob} />` in `<Suspense>`. Sonst → Card mit „Diese Datei kann nicht in der App angezeigt werden" + Download-Button.
- [x] 3.5 `goBack()`-Helper: `window.history.length > 1` → `navigate(-1)`, sonst `navigate(fallbackPath, { replace: true })`.

## 4. FileViewerPage (Route /dokumente/anzeigen/:fileId)

- [x] 4.1 `web/src/pages/FileViewerPage.tsx` anlegen: liest `:fileId` aus URL. **Keine Metadaten-Route nötig** — `<FileViewer source="file">` extrahiert `filename` (aus `Content-Disposition`) und `mimeType` (aus `Content-Type`) aus den Response-Headern des Blob-Downloads.
- [x] 4.2 Rendert `<FileViewer source="file" fileId={id} fallbackPath="/dokumente" />`. Während Load: neutraler Header („Datei wird geladen…").
- [x] 4.3 Route in `web/src/App.tsx` unter `AppShell`-Outlet: `<Route path="dokumente/anzeigen/:fileId" element={<FileViewerPage />} />`.

## 5. SepaMandatViewerPage (Route /mitglieder/:memberId/sepa-mandat/anzeigen)

- [x] 5.1 `web/src/pages/SepaMandatViewerPage.tsx` anlegen: liest `:memberId`, prüft `privateKey` aus `useVault()`.
- [x] 5.2 Vault gelockt → Card mit „Zum Anzeigen Bankdaten-Tresor entsperren (Menü „Tresor")." + Zurück-Button.
- [x] 5.3 Vault offen → Token holen (`/members/:id/sepa-mandat/download-token`), verschlüsselte Datei laden (`responseType: 'arraybuffer'`), `decryptFile()` aus `bankCrypto`, dann `<FileViewer source="blob" blob filename="sepa-mandat.pdf" mimeType="application/pdf" fallbackPath="/mitglieder/:id" />`.
- [x] 5.4 Fehler-States: 403 („Kein Zugriff"), 404 („Kein Mandat hinterlegt"), Decrypt-Fehler („Entschlüsselung fehlgeschlagen — falscher Tresor-Inhalt?").
- [x] 5.5 Route in `App.tsx`: `<Route path="mitglieder/:memberId/sepa-mandat/anzeigen" element={<SepaMandatViewerPage />} />`.

## 6. Aufrufer umstellen

- [x] 6.1 `DocumentsPage.openFile` (web/src/pages/DocumentsPage.tsx:609): Implementierung ersetzen durch `navigate(`/dokumente/anzeigen/${file.id}`)`. `window.open`-Workaround + Token-Fetch entfernen. `fileError`-State bleibt für andere Fehlerpfade (z.B. delete).
- [x] 6.2 `DocumentFileLinkPage` (web/src/pages/DocumentFileLinkPage.tsx): `useEffect` mit `window.location.replace` entfernen, stattdessen direkt `<Navigate to={`/dokumente/anzeigen/${fileId}`} replace />` rendern (oder Komponente komplett auf `FileViewerPage` umleiten — Route bleibt für Backwards-Compat bestehen).
- [x] 6.3 `MemberKontaktTab.openSepaMandat` (web/src/components/admin/MemberKontaktTab.tsx:146-168): Body ersetzen durch `navigate(`/mitglieder/${memberId}/sepa-mandat/anzeigen`)`. Decrypt-Logik wandert nach `SepaMandatViewerPage`. `openError`-State entfernt (nicht mehr nötig — Fehler werden in der Viewer-Route gezeigt).

## 7. Tests

- [x] 7.1 `web/src/components/__tests__/FileViewer.test.tsx`: Image-Render, PDF-Lazy-Load (Mock pdfjs-dist), Unknown-MIME-Fallback, Back-Button (`navigate(-1)` + Fallback). (4 Tests)
- [x] 7.2 `web/src/pages/__tests__/FileViewerPage.test.tsx`: Happy-Path (Token → Blob → Render), 403, 404, ungültige ID. (4 Tests)
- [x] 7.3 `web/src/pages/__tests__/SepaMandatViewerPage.test.tsx`: Vault zu (Hinweis), Vault offen (Decrypt + Render), Decrypt-Fehler, 404. (4 Tests)
- [x] 7.4 `web/src/pages/__tests__/DocumentsPage.openFile.test.tsx`: Klick auf Datei → `navigate('/dokumente/anzeigen/:id')`, kein `window.open`. (1 Test)
- [x] 7.5 `web/src/pages/__tests__/MemberKontaktTab.openSepa.test.tsx`: „Dokument öffnen"-Button → `navigate('/mitglieder/:id/sepa-mandat/anzeigen')`, kein `window.open`. (1 Test). Zusätzlich: bestehender `MemberKontaktTab.permissions.test.tsx` von `renderAsPersonaNoRouter` auf `renderAsPersona` umgestellt (useNavigate braucht jetzt Router).

## 8. Verifikation

- [x] 8.1 `pnpm -C web lint` + `pnpm -C web test` grün. (427/427 Tests grün, 0 ESLint-Errors)
- [x] 8.2 `pnpm -C web build` grün; Bundle-Analyse: pdf.js in eigenem Chunk, Main-Chunk Größe ±0. (Output: `PdfRenderer-*.js` 421 KB / 125 KB gzipped + `pdf.worker.min-*.mjs` 1245 KB als Worker-Asset; Main-Chunk `index-*.js` 838 KB / 195 KB gzipped — pdf.js nicht enthalten.)
- [x] 8.3 `openspec validate pwa-in-app-file-viewer --strict` grün.
- [ ] 8.4 Lokale Manual-Verifikation: Build deployen oder via `vite preview` in Chrome DevTools (Application → Manifest → „Standalone") simulieren — Dokument öffnen, PDF rendern, Zurück-Button → zurück in Dokumente-Liste, **ohne** Tab-Wechsel.
- [ ] 8.5 (Optional, falls Testgerät verfügbar): iPhone Home-Screen-PWA + Android Chrome PWA — Dokument-Öffnen-Flow vollständig durchspielen.

## 9. Archivierung

- [ ] 9.1 Nach Merge: `openspec archive pwa-in-app-file-viewer` ausführen, `openspec/changes/archive/` Commit anhängen.
