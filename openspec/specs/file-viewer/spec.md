# file-viewer Specification

## Purpose
TBD - created by archiving change pwa-in-app-file-viewer. Update Purpose after archive.
## Requirements
### Requirement: Generische File-Viewer-Komponente

Das System SHALL eine wiederverwendbare Komponente `<FileViewer>`
bereitstellen, die Dateien aus zwei Quellen rendern kann:
- `source: 'file'` mit `fileId`, `filename`, `mimeType` — die Komponente
  beschafft die Datei selbst über `/api/files/:id/download-token` + `/api/files/:id/download?token=…` als Blob.
- `source: 'blob'` mit `blob`, `filename`, `mimeType` — die Komponente
  rendert direkt eine bereits im Speicher vorliegende Datei.

Die Komponente SHALL einen eigenen Header mit dem Dateinamen und einem
Zurück-Button rendern. Der Zurück-Button SHALL bei vorhandener History
`navigate(-1)` auslösen, sonst auf einen übergebenen `fallbackPath`
zurückspringen.

Die Komponente SHALL Render-Strategien anhand des `mimeType` wählen:
- `image/*` → `<img>` mit Blob-URL.
- `application/pdf` → lazy-geladener `<PdfRenderer>` (eigene Datei, via
  `React.lazy` + dynamischem Import).
- alle anderen → Hinweis-Card mit Dateiname + Download-Button.

#### Scenario: Bild-Datei anzeigen

- **WHEN** `<FileViewer>` mit `mimeType="image/png"` und einem Bild-Blob gerendert wird
- **THEN** rendert die Komponente ein `<img>`-Element, dessen `src` aus
  `URL.createObjectURL(blob)` stammt
- **THEN** ist der Zurück-Button im Header sichtbar und aktivierbar

#### Scenario: PDF-Datei anzeigen

- **WHEN** `<FileViewer>` mit `mimeType="application/pdf"` und einem PDF-Blob
  gerendert wird
- **THEN** wird das `PdfRenderer`-Modul dynamisch (lazy) importiert
- **THEN** zeigt die Komponente während des Imports einen Loading-State
- **THEN** rendert der `PdfRenderer` alle Seiten des PDFs in
  `<canvas>`-Elementen untereinander

#### Scenario: Unbekannter MIME-Type

- **WHEN** `<FileViewer>` mit `mimeType="application/octet-stream"` (oder
  einem anderen nicht unterstützten Typ) aufgerufen wird
- **THEN** rendert die Komponente einen Hinweis „Diese Datei kann nicht in der
  App angezeigt werden" mit dem Dateinamen
- **THEN** bietet die Komponente einen Download-Button, der die Datei via
  `<a download>` lokal speichert

#### Scenario: Zurück-Button mit leerer History

- **WHEN** der Nutzer den Viewer per Deep-Link in einer frischen
  PWA-Session öffnet (kein History-Eintrag dahinter)
- **AND** auf den Zurück-Button klickt
- **THEN** navigiert die App per `navigate(fallbackPath, { replace: true })`
  zur übergebenen Fallback-Route — kein Verbleib auf einer toten Seite

#### Scenario: Datei-Fetch schlägt fehl

- **WHEN** `<FileViewer source="file">` einen 403 vom Token-Endpoint erhält
- **THEN** rendert die Komponente eine Fehler-Card „Du hast keinen Zugriff auf
  diese Datei."
- **THEN** bleibt der Zurück-Button funktional

### Requirement: Dokument-Viewer-Route

Das System SHALL eine Route `/dokumente/anzeigen/:fileId` bereitstellen, die
über `<FileViewerPage>` die Datei-Metadaten lädt und `<FileViewer source="file">`
rendert. `fallbackPath` ist `/dokumente`.

Die bestehenden Aufrufer SHALL keine eigenen `window.open`- oder
`window.location`-Calls für Dateianzeige mehr enthalten:
- `DocumentsPage.openFile` ruft `navigate('/dokumente/anzeigen/:fileId')`.
- `DocumentFileLinkPage` (`/dokumente/datei/:fileId`) leitet via `<Navigate>`
  auf die neue Viewer-Route weiter (Deep-Link-Kompatibilität).

#### Scenario: Datei aus Dokumenten-Liste öffnen

- **WHEN** der Nutzer in `/dokumente` auf eine Datei klickt
- **THEN** navigiert die App zu `/dokumente/anzeigen/${fileId}`
- **THEN** rendert `<FileViewerPage>` den Viewer mit der Datei
- **THEN** wird **kein** `window.open` und **kein** `window.location.replace`
  aufgerufen

#### Scenario: Geteilter Datei-Link öffnen

- **WHEN** der Nutzer einen Link `/dokumente/datei/${fileId}` öffnet
- **THEN** leitet `DocumentFileLinkPage` auf `/dokumente/anzeigen/${fileId}`
  weiter
- **THEN** der Viewer rendert die Datei in der PWA, **ohne** die App zu
  verlassen

### Requirement: SEPA-Mandat-Viewer-Route

Das System SHALL eine Route `/mitglieder/:memberId/sepa-mandat/anzeigen`
bereitstellen, die das (clientseitig verschlüsselte) SEPA-Mandat eines Members
über den Bankdaten-Tresor entschlüsselt und via
`<FileViewer source="blob">` rendert. `fallbackPath` ist `/mitglieder/:id`.

Die Route SHALL den Vault-Zustand (`privateKey` aus `VaultContext`) prüfen,
bevor sie versucht zu entschlüsseln.

`MemberKontaktTab.openSepaMandat` SHALL die Decrypt-Logik nicht mehr selbst
ausführen; der Button navigiert nur noch auf die Viewer-Route.

#### Scenario: SEPA-Mandat mit entsperrtem Vault öffnen

- **WHEN** `privateKey` im `VaultContext` gesetzt ist
- **AND** der Nutzer die Viewer-Route betritt
- **THEN** lädt die Page Token + verschlüsselte Datei, entschlüsselt sie via
  `decryptFile()` und rendert `<FileViewer source="blob">` mit dem PDF-Blob

#### Scenario: SEPA-Mandat mit gesperrtem Vault öffnen

- **WHEN** `privateKey` im `VaultContext` `null` ist
- **AND** der Nutzer die Viewer-Route betritt
- **THEN** zeigt die Page eine Hinweis-Card „Zum Anzeigen Bankdaten-Tresor
  entsperren (Menü „Tresor")."
- **THEN** wird **kein** Token-Fetch und **kein** Decrypt-Versuch ausgelöst
- **THEN** bleibt ein Zurück-Button funktional

#### Scenario: SEPA-Mandat existiert nicht

- **WHEN** der Server beim Token-Fetch HTTP 404 liefert
- **THEN** rendert die Page „Kein Mandat hinterlegt." + Zurück-Button

#### Scenario: Entschlüsselung schlägt fehl

- **WHEN** `decryptFile()` einen Fehler wirft (z.B. falscher Schlüssel)
- **THEN** rendert die Page „Entschlüsselung fehlgeschlagen." + Zurück-Button

### Requirement: Lazy-Loading des PDF-Renderers

Das System SHALL `pdfjs-dist` und den `PdfRenderer` ausschließlich beim ersten
PDF-View laden. Im Initial-Bundle-Chunk SHALL `pdfjs-dist` **nicht** enthalten
sein.

Der pdf.js-Worker SHALL über einen Vite-`?url`-Asset-Import als statisches
Asset eingebunden und zur Laufzeit über `pdfjsLib.GlobalWorkerOptions.workerSrc`
gesetzt werden.

#### Scenario: Bundle enthält pdf.js nicht im Main-Chunk

- **WHEN** `pnpm -C web build` ausgeführt wird
- **THEN** ist `pdfjs-dist` in einem separaten Chunk (sichtbar in der
  Bundle-Analyse)
- **THEN** lädt eine Nutzer-Session, die nur Bilder öffnet, das pdf.js-Modul
  **nicht** herunter

#### Scenario: Erstes PDF-Öffnen lädt pdf.js dynamisch

- **WHEN** der Nutzer zum ersten Mal in seiner Session eine PDF-Datei öffnet
- **THEN** löst React.lazy einen dynamischen Import von `pdfjs-dist` aus
- **THEN** zeigt der Viewer einen Loading-State, bis Modul + Worker geladen
  sind
- **THEN** rendert anschließend der `PdfRenderer` das PDF

