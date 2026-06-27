## Context

Die TeamWERK-PWA wird auf iOS und Android im Standalone-Modus (Home-Screen-Icon)
betrieben. In diesem Modus existiert die Browser-Chrome (Tab-Bar, Adressleiste,
Zurück-Pfeil) **nicht**. Solange der Nutzer innerhalb der React-SPA bleibt,
übernimmt unsere AppShell die Navigation. Sobald jedoch der Browser eine
Inline-Datei rendert (PDF im iOS-PDF-Viewer, Bild als Standalone-Resource),
gibt es keine UI mehr, um zur App zurückzukehren — der App-Switcher ist die
einzige Rettung.

Drei Stellen lösen heute genau diesen Effekt aus (`DocumentsPage.openFile`,
`DocumentFileLinkPage`, `MemberKontaktTab.openSepaMandat`). Backend liefert
Dateien mit `Content-Disposition: inline`, was im normalen Browser-Tab korrekt
ist (Vorschau), in einer Standalone-PWA aber zur Sackgasse wird.

## Goals / Non-Goals

**Goals:**
- Datei-Vorschau (Bilder, PDFs) ohne Verlassen der PWA — egal ob Standalone
  oder normaler Browser.
- Konsistente Zurück-Navigation über eigenen Header-Button.
- Wiederverwendung für die drei bestehenden Aufrufer plus zukünftige
  Datei-Ansichten.
- Kein Initial-Bundle-Wachstum für Nutzer, die nie PDFs öffnen.

**Non-Goals:**
- PDF-Annotation, Markup, Suche, Drucken aus dem Viewer.
- Bild-Bearbeitung, Zoom-Gesten (Pinch). Initial nur Anzeige in passender Größe.
- Native iOS-/Android-„Datei in App teilen"-Integration. Download bleibt
  Browser-Standard.
- Backend-Refactor des Download-Pfads.

## Decisions

### D1: Eine Komponente, zwei Routen — statt Modal

`<FileViewer>` ist die Render-Komponente; sie wird von zwei Route-Pages
gehostet:
- `/dokumente/anzeigen/:fileId` (generisch, ID-basiert)
- `/mitglieder/:memberId/sepa-mandat/anzeigen` (Vault-gated, Blob-basiert)

**Alternativen:**
- **Modal über bestehender Seite**: Slide-up fühlt sich nativ an, aber bricht
  Deep-Linking (z.B. der „Link kopieren"-Flow in DocumentsPage). Außerdem
  liegt der Modal-Container im Stack der Aufrufer-Seite — bei vielen offenen
  Modalen wird die DOM-Hierarchie unübersichtlich.
- **Eine einzige generische Route mit dynamischer Source**: Würde funktionieren,
  vermischt aber Verantwortungen (Token-Fetch vs. Vault-Decrypt) und macht
  Vault-Gate-Logik zur Sonderlogik in einer „generischen" Komponente.

Zwei dünne Route-Pages halten die jeweilige Datenbeschaffung lokal und
explizit.

### D2: PDF-Rendering mit `pdfjs-dist` über Lazy-Load

`pdfjs-dist` rendert PDFs in `<canvas>` — komplett unabhängig vom nativen
iOS-/Android-PDF-Viewer. Das ist der einzige Weg, der die Standalone-PWA-Falle
sicher umgeht.

Bundle-Strategie:
- `pdfjs-dist` als reguläre `pnpm`-Dependency.
- `<PdfRenderer>` ist eine **separate Datei** und wird via `React.lazy(() =>
  import('./PdfRenderer'))` geladen, **nur** wenn `<FileViewer>` einen
  `application/pdf`-MIME sieht.
- pdf.js-Worker (`pdf.worker.min.mjs`) wird via Vite-Import-URL
  (`import workerSrc from 'pdfjs-dist/build/pdf.worker.min.mjs?url'`) als
  statisches Asset eingebunden und zur Laufzeit
  `pdfjsLib.GlobalWorkerOptions.workerSrc = workerSrc` gesetzt.

**Alternativen verworfen:**
- **`<iframe src=blob:…>` mit Browser-PDF-Viewer**: Auf iOS Safari historisch
  fragil (nur erste Seite, kein Scrollen, gelegentlich kompletter Render-Stop).
  Genau das Problem, das wir beheben wollen.
- **`react-pdf` als Wrapper**: Bequemer, aber zusätzliche Abstraktionsschicht
  und Dependency. Direkter `pdfjs-dist`-Import ist nur ~50 Zeilen mehr Code
  und gibt uns volle Kontrolle.

### D3: Discriminated Union als `<FileViewer>`-API

```ts
type FileViewerSource =
  | { source: 'file'; fileId: number; filename: string; mimeType: string }
  | { source: 'blob'; blob: Blob; filename: string; mimeType: string }
```

`source: 'file'` lädt selbst: Token holen → Blob-Download → Render.
`source: 'blob'` rendert direkt, weil die Datei (SEPA-Mandat) bereits
clientseitig entschlüsselt vorliegt.

Vorteil: ein Render-Pfad, zwei klar getrennte Datenquellen. Tests können beide
Quellen isoliert prüfen.

### D4: Back-Button-Verhalten mit Fallback

`navigate(-1)` ist der Normalfall (Klick aus DocumentsPage → ein Eintrag in
History). Bei Deep-Links (Link kopiert → in frischer PWA-Session geöffnet) ist
History leer; ohne Fallback würde `navigate(-1)` ins Nichts gehen.

Beide Viewer-Pages reichen eine `fallbackPath`-Prop an `<FileViewer>` weiter
(`'/dokumente'` bzw. `'/mitglieder/:id'`). Logik:

```ts
function goBack() {
  if (window.history.length > 1) navigate(-1)
  else navigate(fallbackPath, { replace: true })
}
```

### D5: Vault-Gate für SEPA-Viewer

`<SepaMandatViewerPage>` prüft `privateKey` aus `VaultContext`:
- `privateKey != null` → Token fetchen, Datei laden, mit `decryptFile()`
  entschlüsseln, `<FileViewer source="blob">` rendern.
- `privateKey == null` → Hinweis-Card: „Zum Anzeigen Bankdaten-Tresor
  entsperren (Menü „Tresor")." + Zurück-Button.

Kein automatisches Re-Render nach Entsperren nötig (würde
`VaultContext`-Observer brauchen); der Nutzer kann nach Entsperren erneut auf
„Mandat öffnen" klicken. Das spiegelt das aktuelle Verhalten von
`MemberKontaktTab.openSepaMandat` 1:1.

### D6: Aufrufer behalten ihre Token-Vorbereitung NICHT

Heute holt `DocumentsPage.openFile` selbst den Token, bevor `window.open`
gefüllt wird (Workaround für Popup-Blocker). Mit der Route-Navigation entfällt
das: `<FileViewerPage>` holt Token und Blob selbst. Aufrufer machen nur noch
`navigate(...)`.

Vorteil: einheitlicher Token-Flow, kein toter Workaround-Code mehr.

## Risks / Trade-offs

**Bundle-Wachstum durch pdf.js:**
~500 KB gzipped sind nicht trivial. Mitigiert durch Lazy-Load — Nutzer, die
nie PDFs öffnen, zahlen nichts. Erstes PDF-Öffnen ist messbar langsamer als
heute (Worker + Modul-Download). Akzeptabel: die Alternative ist die heutige
Sackgasse, die Nutzer schon jetzt zur App-Beendigung zwingt.

**pdf.js + Vite-Worker-Config:**
Vite hat ein dokumentiertes Rezept für pdf.js-Worker (`?url`-Import). Falls
das mit unserer Vite-Version oder dem `vite-plugin-pwa` zickt: Fallback ist
direkter Worker-File-Import oder Inline-Worker. Risiko: niedrig, bekanntes
Pattern.

**Datei-Größenlimit beim Blob-Download:**
Wir laden die komplette Datei in den Speicher (Blob). Für die typische
Vereinsgröße (PDFs <5 MB, Bilder <2 MB) unkritisch. Bei sehr großen
Bestandsdateien (>50 MB) könnte das auf älteren Phones knapp werden. Aktuell
gibt es **kein** Upload-Limit serverseitig (`internal/files/handler.go`); falls
das in Zukunft relevant wird, ist ein chunked PDF-Stream-Read von pdf.js
möglich, aber heute aus dem Scope.

**SEPA-Vault Re-Render:**
Wer den Viewer mit gelocktem Vault öffnet, muss nach dem Entsperren auf
„Mandat öffnen" zurückklicken (kein Auto-Reload). Verbesserung wäre ein
`useEffect([privateKey])`, der bei Wechsel auf „entsperrt" automatisch
weitermacht — bewusst out-of-scope, um VaultContext-Coupling klein zu halten.

**Service-Worker-Cache:**
Der bestehende Network-First-Cache für `/api/*` greift transparent — wenn die
Datei einmal geladen wurde, ist sie offline verfügbar. Kein zusätzlicher Code
nötig. Falls offline die Token-Anfrage scheitert: Fehler-UI greift, Zurück
funktioniert weiter.

**Telegram/Mail/Maps-Links bleiben `window.open`:**
Bewusst nicht angefasst — die öffnen externe Apps (Telefon, WhatsApp, Maps),
nicht Inline-Render in der PWA. iOS/Android wechseln dort sauber in die
Ziel-App und zurück. Wäre eine andere Bug-Klasse.

## Migration / Rollout

Keine Datenmigration. Reines Frontend-Refactoring. Nach Deploy ist der neue
Pfad sofort aktiv; alte Deep-Links (`/dokumente/datei/:fileId`) bleiben
funktional — `DocumentFileLinkPage` wird jetzt zum Viewer-Renderer statt zum
Redirect.

Keine Feature-Flags. Risiko niedrig (rein additive Routen, drei kleine
Call-Site-Umstellungen).
