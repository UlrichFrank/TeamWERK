## Why

Welle 2 der `test-coverage-roadmap` (Priorisierung siehe Capability `test-strategy`). Der
SEPA-Beitragslauf ist finanz- und PII-kritisch: er zieht echtes Geld ein und berührt
verschlüsselte Bankdaten. Zwei zentrale Routen sind heute **funktional ungetestet**:
`POST /api/fee-run/confirm` (schreibt das append-only Saison-Protokoll) und
`GET /api/fee-run/protocol` (liest es zurück) — beide sind zwar in der Persona-Matrix
autorisierungs-geprüft, aber ihr fachliches Verhalten (Protokoll-Format, Append-Only, **keine
IBANs im Protokoll**, 404-Pfade) hat keinen einzigen Test. Bei `export-data` fehlen die
member-bezogenen 400-Fälle, und in der Beitrags-Halbierungsmatrix fehlt eine Zelle.

## What Changes

Ausschließlich **Tests**. Keine Geschäftslogik-, API-, Schema- oder SSE-Änderung. Alle Tests
nutzen die bereits vorhandenen lokalen Helfer des Packages (`setupSrv`, `insertMember`,
`defaultMember`, `insertSeason2027`, `getPreview`, `itemFor`, `tok`) — **keine neuen Fixtures
nötig**.

- **`fee-run/confirm`** (`internal/beitragslauf/handler.go`): Happy-Path (200 + Protokoll-Datei
  geschrieben), Protokoll enthält Mitgliedsnummer/Betrag/Erfolg aber **keine IBAN**, gemischter
  Batch (Erfolg + Fehlschlag → beide Protokoll-Blöcke + Counts), Append-Only (zwei Läufe → beide
  Blöcke), 404 bei unbekannter Saison, 400 bei ungültigem Body.
- **`fee-run/protocol`**: Rücklesen nach Confirm (200, `text/plain`), 404 bei unbekannter Saison,
  und die bewusste Invariante „gültige Saison ohne Lauf → 200 mit leerem Body (nicht 404)".
- **`fee-run/export-data`**: die ungetesteten member-bezogenen 400-Fälle — Mitglied ohne
  SEPA-Mandat, Mitglied ohne Bankdaten-Envelope, unbekannte Member-ID, ungültiger Body. (Die
  Vereins-SEPA- und Fälligkeits-400 sind bereits abgedeckt → nicht dupliziert; liefert weiterhin
  nur Ciphertext.)
- **Preview**: der fehlende Halbierungs-Restfall „unterjähriger Austritt **+** `home_club_id`
  gesetzt → Kategorie `aktiv_mit`, halbiert" (die drei Halbierungs-Bedingungen selbst und die
  `aktiv_ohne`-Variante sind bereits getestet → nur die `aktiv_mit`-Zelle fehlt), plus die bisher
  ungetestete **Summen-Aggregation** (`included_count`/`total_cent`/`gesamtsumme_cent`) — der vom
  Kassierer gelesene Einzugsbetrag.

## Capabilities

### New Capabilities

- `fee-run-audit`: dokumentiert die geprüften Invarianten des Beitragslaufs — append-only
  Protokoll ohne Klartext-Bankdaten, Protokoll-Rücklesen (200/leer/404), Ablehnung ungültiger/
  ausgeschlossener Mitglieder beim Export (400), Vollständigkeit der Halbierungsmatrix.

### Modified Capabilities

_(keine — die getesteten Verhaltensweisen bestehen bereits im Code; hier werden sie nur
mechanisch festgenagelt.)_

## Impact

- **Tests (neu):** `internal/beitragslauf/handler_test.go` (Confirm/Protocol, Halbierungs-Restfall),
  `internal/beitragslauf/encryption_export_test.go` (export-data-400). ~13 neue Tests.
- **Code:** keiner (reiner Test-Change; kein verifizierter Bug in diesem Bereich).
- **Kein** Backend-Vertrag, keine Migration, keine Env-Änderung.
- **Optional/separat** (Roadmap 5.2, NICHT Teil dieses Changes): `auth`-Fehlerpfade
  (Session-Invalidierung nach E-Mail-Änderung, Passwort-Reauth, abgelaufener/manipulierter Token)
  — eigener kleiner Change, wenn gewünscht.
