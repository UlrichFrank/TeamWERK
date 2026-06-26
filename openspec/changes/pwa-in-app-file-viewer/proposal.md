## Why

Wer in der TeamWERK-PWA auf iPhone/iPad/Android ein Dokument (Bild, PDF) öffnet,
landet in einer Sackgasse: das Standalone-Fenster rendert die Datei direkt
(Inline-Disposition), aber die PWA hat keine Browser-Chrome (kein Tab, kein
Zurück-Pfeil) und der iOS-PDF-Viewer bringt keine eigene mit. Der Nutzer muss
die App im App-Switcher beenden und neu starten, um zur Anwendung
zurückzukehren.

Drei Aufrufer sind betroffen:

| Datei | Aufruf | Mechanik |
|---|---|---|
| `web/src/pages/DocumentsPage.tsx:609` | `window.open('about:blank', '_blank')` → Token-URL | Öffnet "Tab" — im Standalone-PWA verlässt das die App (iOS Safari / Android Custom Tab) |
| `web/src/pages/DocumentFileLinkPage.tsx:18` | `window.location.replace(downloadUrl)` | Navigiert die PWA selbst weg, der Inline-Render hat keine Zurück-UI |
| `web/src/components/admin/MemberKontaktTab.tsx:163` | `window.open(blobUrl, '_blank')` | SEPA-Mandat (clientseitig entschlüsselt) — gleicher Effekt |

Backend (`internal/files/handler.go:741, 770`) setzt für Token-Downloads
`Content-Disposition: inline`. Der Mechanismus ist überall identisch:
**Browser-Inline-Render in einem Kontext ohne Navigation-UI**.

## What Changes

**Neue gemeinsame Komponente `<FileViewer>`** (`web/src/components/FileViewer.tsx`):
- Eigener Header (Dateiname links, Zurück-Button rechts, optional
  Download-Button).
- Body wählt Renderer nach MIME-Type:
  - `image/*` → `<img>` zentriert
  - `application/pdf` → lazy-geladener PdfRenderer (pdfjs-dist)
  - sonst → Hinweis + Download-Button
- Zwei Quellen via Discriminated Union:
  - `{ source: 'file', fileId, filename, mimeType }` — holt Datei via
    `/api/files/:id/download?token=...` als Blob.
  - `{ source: 'blob', blob, filename, mimeType }` — Datei ist bereits im
    Speicher (SEPA-Mandat nach Vault-Entschlüsselung).
- Back-Button: `navigate(-1)`; bei leerem History-Stack Fallback auf eine
  übergebene `fallbackPath`-Prop.

**Zwei neue Routen** in `web/src/App.tsx`:
- `dokumente/anzeigen/:fileId` → `<FileViewerPage source="file" />` — generischer
  Dokument-Viewer; lädt File-Metadaten (`mime_type`, `original_name`) + Token,
  rendert via `<FileViewer>`.
- `mitglieder/:memberId/sepa-mandat/anzeigen` → `<SepaMandatViewerPage>` —
  entschlüsselt das Mandat über `VaultContext` und rendert via
  `<FileViewer source="blob">`. Vault gelockt → Hinweis "Tresor entsperren",
  kein Crash.

**Aufrufer umstellen:**
- `DocumentsPage.openFile` → `navigate('/dokumente/anzeigen/${file.id}')` statt
  `window.open`.
- `DocumentFileLinkPage` rendert direkt den Viewer (statt
  `window.location.replace`).
- `MemberKontaktTab.openSepaMandat` → `navigate('/mitglieder/${memberId}/sepa-mandat/anzeigen')`
  statt `window.open(blobUrl)`. Vault-Check + Decrypt wandern in die
  Viewer-Page.

**Neue Frontend-Dependency:** `pdfjs-dist` (~500 KB gzipped). Wird **nur**
beim ersten PDF-View per `React.lazy()` + dynamischem Import geladen — kein
Initial-Bundle-Wachstum für Nutzer, die nie ein PDF öffnen. Der pdf.js-Worker
wird über Vites `?url`-Import als statisches Asset eingebunden.

**Backend bleibt unverändert.** Keine Migrations, keine API-Änderungen, keine
Auth-Anpassungen. `Content-Disposition: inline` bleibt — wird jetzt aber nur
noch von `<FileViewer>` konsumiert, das die Datei als Blob lädt und intern
rendert.

## Scope

**In scope:**
- `<FileViewer>` + lazy PdfRenderer-Subkomponente (Bild / PDF / Fallback)
- Route `/dokumente/anzeigen/:fileId` + `<FileViewerPage>`
- Route `/mitglieder/:memberId/sepa-mandat/anzeigen` + `<SepaMandatViewerPage>`
- Umstellung der drei Aufrufer (DocumentsPage, DocumentFileLinkPage,
  MemberKontaktTab)
- `pdfjs-dist` als pnpm-Dependency + Vite-Worker-Konfiguration
- Tests für alle drei Aufrufer + die Viewer-Routen (Vault-Gate inkl.)

**Out of scope:**
- Bild-Zoom (Pinch/Doppeltap) — späterer Pull-Request, jetzt nur Anzeige
- PDF-Annotation, Markup, Suche
- Offline-Caching im Service Worker (profitiert automatisch vom bestehenden
  Network-First-Cache für `/api/*`)
- Backend-Änderungen an `/api/files/:id/download` oder dem Token-Schema
- Andere `window.open`-Stellen, die keine Dateien öffnen (Maps-Link, WhatsApp,
  Mail-Links) — diese öffnen externe Apps, nicht Inline-Render in der PWA
- XML-/CSV-Downloads in `BeitragslaufPage` und `MembersPage` — die nutzen
  bereits `<a download>` und landen sauber im Files-Verzeichnis

## Test-Anforderungen

| Komponente / Route | Testname | Erwartetes Ergebnis |
|---|---|---|
| `<FileViewer>` (image) | `FileViewer_rendersImageForImageMime` | `image/png`-Blob wird als `<img>` mit `URL.createObjectURL`-src gerendert |
| `<FileViewer>` (PDF) | `FileViewer_lazyLoadsPdfRendererForPdfMime` | Mock-pdfjs-dist wird dynamisch importiert, Loading-State vor Render |
| `<FileViewer>` (unbekannt) | `FileViewer_showsDownloadFallbackForUnknownMime` | `application/octet-stream` → Download-Button + Dateiname |
| `<FileViewer>` (Back) | `FileViewer_backButtonNavigatesBack` | Klick auf Zurück ruft `navigate(-1)`; leerer Stack → `fallbackPath` |
| `<FileViewerPage>` | `FileViewerPage_fetchesTokenAndBlob` | Erst `/files/:id/download-token`, dann `download?token=…` als Blob, dann Viewer |
| `<FileViewerPage>` 403 | `FileViewerPage_shows403Error` | Fehler-UI „Kein Zugriff", Zurück-Button funktioniert |
| `<FileViewerPage>` 404 | `FileViewerPage_shows404Error` | Fehler-UI „Datei nicht gefunden" |
| `<SepaMandatViewerPage>` Vault offen | `SepaViewer_decryptsAndRendersWhenVaultUnlocked` | Decrypt → Viewer mit PDF |
| `<SepaMandatViewerPage>` Vault zu | `SepaViewer_promptsToUnlockVaultWhenLocked` | „Tresor entsperren"-Hinweis, kein Decrypt-Versuch, kein Crash |
| `DocumentsPage` | `DocumentsPage_openFileNavigatesToViewer` | Klick auf Datei ruft `navigate('/dokumente/anzeigen/${id}')`, kein `window.open` mehr |
| `DocumentFileLinkPage` | `DocumentFileLinkPage_rendersViewerInline` | Komponente rendert `<FileViewerPage>` direkt, kein `window.location.replace` |
| `MemberKontaktTab` | `MemberKontaktTab_sepaButtonNavigatesToViewer` | Klick auf „Mandat öffnen" → `navigate('/mitglieder/:id/sepa-mandat/anzeigen')`, kein `window.open` |
