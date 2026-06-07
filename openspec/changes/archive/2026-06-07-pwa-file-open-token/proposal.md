## Why

In der installierten PWA (iOS Standalone-Modus) öffnet ein Klick auf eine Datei nichts — `window.open(blobUrl, '_blank')` wird vom Browser geblockt. Als Workaround können Nutzer Dateien nur über das Kontextmenü herunterladen, ohne zu wissen, wo die Datei landet. Die Funktion muss für alle Dateitypen zuverlässig und plattformübergreifend funktionieren.

## What Changes

- **Neuer Backend-Endpoint** `GET /api/files/{id}/download-token` — gibt ein kurzlebiges, HMAC-signiertes Token zurück (5 Minuten TTL, kein DB-Eintrag nötig)
- **Erweiterter Download-Endpoint** `GET /api/files/{id}/download` — akzeptiert zusätzlich `?token=` als Alternative zum `Authorization`-Header
- **Frontend `openFile()`** in `DocumentsPage.tsx` — holt zuerst ein Token, öffnet dann `/api/files/{id}/download?token=xyz` direkt via `window.open()` statt Blob-URL

## Capabilities

### New Capabilities

- `file-download-token`: Kurzlebiges HMAC-signiertes Download-Token, das authentifizierten Nutzern erlaubt, eine Datei ohne Authorization-Header über eine direkte URL abzurufen

### Modified Capabilities

- `documents-ui`: Der Klick-Handler für Dateien verwendet künftig Token statt Blob-URLs

## Impact

- **Backend:** `internal/files/` — neuer Handler `HandleDownloadToken`, angepasster `HandleDownload`
- **Frontend:** `web/src/pages/DocumentsPage.tsx` — Funktion `openFile()` (~10 Zeilen)
- **Keine neuen Dependencies**, kein DB-Schema-Change, keine Migration
- Bestehende Authorization-Header-basierte Downloads (z. B. für Inline-Blob-Nutzung) bleiben weiterhin funktionsfähig
