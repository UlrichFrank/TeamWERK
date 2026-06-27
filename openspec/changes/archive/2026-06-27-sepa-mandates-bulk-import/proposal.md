## Why

SEPA-Mandate liegen aktuell als PDFs in einem lokalen Verzeichnis vor (Dateinamen-Schema `VornameNachname.pdf`). Pro Mitglied müssen sie heute einzeln über „Dokument hochladen" im Mitglieder-Tab Bankdaten zugeordnet werden — bei ~100 Mandaten unverhältnismäßig viele Klicks. Eine Bulk-Import-Funktion neben „Import CSV" auf `/mitglieder` löst das in einem Schritt, ohne dass das bestehende Per-Member-Upload-Flow angefasst wird.

## What Changes

- **Neuer Aktions-Menü-Eintrag „Import SEPA-Mandate"** auf der Mitgliederseite (gleicher Dropdown wie „Import CSV" / „Export CSV", nur sichtbar für `admin`, `vorstand`, `kassierer`).
- **Verzeichnis-Upload im Browser** (`<input type="file" webkitdirectory multiple>`) — der Nutzer wählt das lokale PDF-Verzeichnis; das Frontend filtert auf `*.pdf` und postet alle PDFs in einem Multipart-Request.
- **Neue Route** `POST /api/members/sepa-mandates/import` (Auth: `vorstand` + `kassierer` + `admin` — selbe Tier-Position wie der Einzel-Upload `POST /api/upload/sepa-mandat/{id}`).
- **Match-Logik per Dateinamen**: Pro PDF wird der Basename (ohne `.pdf`) gegen `first_name + last_name` jedes Mitglieds verglichen — Vergleich ist case-insensitive, ignoriert Leerzeichen/Bindestriche/Apostrophe, normalisiert Umlaute (`ä→ae`, `ö→oe`, `ü→ue`, `ß→ss`). Zusätzlich wird beide Reihenfolgen geprüft (`VornameNachname` und `NachnameVorname`).
- **Skip-Logik** (idempotent):
  - **already-exists**: Mitglied hat bereits `sepa_mandat_path` ≠ NULL → PDF wird verworfen, Mandat bleibt unverändert. **Bestehende Dokumente werden nicht überschrieben.**
  - **no-match**: kein Mitglied passt → PDF wird verworfen.
  - **ambiguous**: mehr als ein Mitglied passt → PDF wird verworfen (keine willkürliche Auswahl).
- **Erfolgreicher Match** (`matched`): identisches Verhalten wie der Einzel-Upload-Endpoint — PDF wird unter `sepa-mandats/<uuid>.pdf` im Upload-Verzeichnis abgelegt, `members.sepa_mandat_path` gesetzt, **zusätzlich `members.sepa_mandat=1`** (analog dem Verständnis „Mandat liegt unterschrieben vor"). `sepa_mandat_date` bleibt unverändert und wird in der UI nachgepflegt — das Feld ist Pflicht für den Beitragslauf, aber nicht aus dem PDF ableitbar.
- **Response = Report** mit vier Listen (`imported`, `already_exists`, `no_match`, `ambiguous`) — pro Eintrag `filename` und optional `member_id`/`member_name`. Frontend rendert den Report im Modal mit zählenden Sektionen.
- **Validierung pro Datei** (Reuse der `saveFile`-Helper-Logik): `application/pdf`, ≤ 2 MB. Nicht-PDFs werden vom Frontend gefiltert; serverseitige Verletzung → Datei wandert in `no_match` mit Hinweis „kein PDF".
- **Gesamt-Request-Limit**: 50 MB Multipart-Body (deckt ~250 Mandate ab, ausreichend für eine Vereinsgröße).
- **SSE-Broadcast** `members-updated` nach erfolgreichem Import → die offene Mitgliederliste lädt neu.

## Capabilities

### New Capabilities
- `sepa-mandat-bulk-import`: Verzeichnis-basierter Bulk-Import von SEPA-Mandat-PDFs auf der Mitgliederseite, mit Filename-Match, Skip-Existing und Report-Response.

### Modified Capabilities
- _(keine — der existierende `sepa-mandat-upload` bleibt unangetastet, weil der Per-Member-Flow unverändert weiter funktioniert)_

## Test-Anforderungen

| Route | Testname | Erwartung / Invariante |
|---|---|---|
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_HappyPath_MatchAndStore` | PDF `MaxMustermann.pdf` matcht eindeutig `first_name=Max, last_name=Mustermann` → Datei landet in `sepa-mandats/`, `sepa_mandat_path` gesetzt, `sepa_mandat=1`, Response enthält Eintrag in `imported`. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_SkipsExistingPath` | Mitglied hat bereits `sepa_mandat_path='alt.pdf'` → neuer Upload wird in `already_exists` gemeldet, **DB unverändert**, alte Datei nicht gelöscht. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_NoMatchReported` | PDF `Unbekannt.pdf` → kein DB-Treffer → Eintrag in `no_match`, keine Datei im Upload-Verzeichnis. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_AmbiguousMatchSkipped` | Zwei Mitglieder mit identischem Namen → PDF landet in `ambiguous` mit beiden `member_id`-Kandidaten, **keine DB-Mutation**. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_UmlautNormalization` | Datei `JuergenMueller.pdf` matcht Mitglied `first_name=Jürgen, last_name=Müller`. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_ReverseNameOrder` | Datei `MustermannMax.pdf` matcht ebenfalls (Vor-/Nachname-Reihenfolge tolerant). |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_RejectsNonPDF` | `.jpg`-Datei (auch wenn Frontend versagt) → Eintrag in `no_match` mit Begründung „kein PDF", kein Speichern. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_FileTooLarge` | PDF > 2 MB → Eintrag in `no_match` mit Begründung „zu groß", kein Speichern. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_ForbiddenForSpieler` | Aufruf als Nutzer mit nur `spieler`-Vereinsfunktion → HTTP 403. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_AllowedForKassierer` | Kassierer darf importieren (analog Bank-Details-Whitelist) → HTTP 200. |
| `POST /api/members/sepa-mandates/import` | `TestBulkImport_BroadcastEmitted` | Nach mindestens einem erfolgreichen Match wird `hub.Broadcast("members-updated")` aufgerufen. |

**Garantierte Invarianten**:
1. **Kein Überschreiben**: ein Mitglied mit nicht-leerem `sepa_mandat_path` behält Datei + Pfad nach dem Bulk-Import (Idempotenz beim erneuten Aufruf).
2. **Atomarität pro Datei**: schreibt der DB-`UPDATE` einer gematchten Datei fehl, wird die zuvor abgelegte Datei vom Filesystem entfernt (analog `UploadSepaMandat`).
3. **Berechtigungs-Parität**: derselbe Rollen-Cut wie für den Einzel-Upload (`vorstand`+`kassierer`+`admin`).

## Impact

- **Backend:**
  - `internal/upload/handler.go`: neuer Handler `BulkImportSepaMandate` (Multipart-Iteration, Match-Lookup, Per-File-Save, Report-Aggregation). Reuse von `saveFile`-Validierung über extrahierten Helper.
  - `internal/app/router.go`: neue Route `POST /api/members/sepa-mandates/import` unter dem `Vorstand+Kassierer`-Tier.
  - `internal/permissions/matrix_test.go`: neuer Eintrag analog `/api/upload/sepa-mandat/{id}`.
  - **Kein DB-Schema-Change**, keine neue Migration.
- **Frontend:**
  - `web/src/pages/MembersPage.tsx`: neuer Dropdown-Eintrag „Import SEPA-Mandate" + neues Modal (`<input webkitdirectory>`, Vorschau-Liste der erkannten PDFs, Submit, Report-Anzeige).
  - Reuse des bestehenden `useLiveUpdates`-Hooks für `members-updated`.
- **Keine** neuen Dependencies, **keine** RAM-Auffälligkeiten (Multipart-Verarbeitung streamend pro File, 50 MB Gesamt-Cap).
- **Bestehende Routen**: unverändert. `POST /api/upload/sepa-mandat/{id}` bleibt der kanonische Einzel-Upload-Pfad und wird vom Detail-Tab weiter genutzt.
