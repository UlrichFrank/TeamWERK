## Why

Der Spielplan von Team Stuttgart lebt in Handball4All (BWHV/SRM). Aktuell werden Spiele
im Kalender **von Hand** angelegt — für ~146 Spiele pro Saison (Heim + Auswärts, über 21
Staffeln) fehleranfällig und mühsam, und bei jeder Auslosungs-/Verlegungswelle erneut. Die
Daten liegen bei H4A strukturiert vor; ein Direkt-Import spart die Doppelerfassung und hält
den Kalender aktuell.

Zwei Quellen wurden untersucht und verifiziert (siehe `design.md`):

- **`edit.php` (xajax)** liefert die **Vollsaison** inkl. stabiler **Spielnummer** (`Nr.`) und
  interner H4A-game-ID. Mit dem Filter „Nur Spiele mit eigener Beteiligung" (`opOwnGames`)
  genau die 146 Team-Stuttgart-Spiele — verifiziert per Chrome DevTools (644 → 146).
- Die **BWHV-Hallenliste** (1020 Hallen) ist die Autorität für Hallenadressen. Sie wurde in
  einem früheren Change bereits nach `venues` importiert — **aber ohne die Hallennummer**,
  die H4A als Fremdschlüssel referenziert. Verifiziert: 1017/1018 Bestands-Venues sind über
  `(Name, Ort, Straße)` verlustfrei auf ihre Hallennummer rück-mappbar; alle
  spielrelevanten Hallennummern lösen eindeutig auf (z. B. `3029 → venue 968`).

Ohne die Hallennummer an `venues` kann der Spiel-Import den Spielort (`Hallennummer`) nicht
auf ein `venue` abbilden — deshalb gehört der Hallenlisten-Backfill in **denselben** Change.

## What Changes

- **Neue DB-Felder** (Migration `042`): `venues.hall_number` (nullable, partial-unique) und
  `games.external_id` (= BWHV-Spielnummer, Idempotenz-Anker).
- **Hallenlisten-Import erweitert**: liest jetzt die Spalte `Nummer` mit und backfillt
  `hall_number` an bestehenden Venues per `(Name, Ort, Straße)`-Match. Mehrdeutige oder
  fehlende Zuordnungen bleiben `NULL` und werden im Ergebnis gemeldet (fail-safe).
- **Neuer H4A-Spiel-Import**: Admin/Vorstand tippt im Import-Dialog H4A-Zugangsdaten ein
  (User/Passwort, **nirgends gespeichert**). Der Server loggt sich bei H4A ein, ruft
  `edit.php` (S-kodierter xajax-Call, `all;all` + `opOwnGames`), parst die Spieltabelle,
  bildet Staffel → Kader-Mannschaft und Hallennummer → Venue ab und liefert einen **Diff**
  (neu / geändert / unverändert) zurück.
- **Zwei-Phasen-Flow**: `preview` (mit Zugangsdaten, liest H4A) liefert einen
  self-contained Plan; `apply` (ohne Zugangsdaten, nur DB) schreibt die bestätigten Spiele.
  Passwort lebt damit ausschließlich für die Dauer des `preview`-Requests.
- **Diff-Modal** im Kalender: zeigt Änderungen an bestehenden Terminen, Zuordnung
  Staffel→Mannschaft, Gegner, Spielzeit, Typ (heim/auswärts), Spielort — je einzeln
  bestätigbar. **Dienst-Templates** pro Spiel wählbar, als Batch (alle Heimspiele →
  Template X) und selektiv pro Zeile.
- **Idempotenz** über `games.external_id = Nr.`: erneuter Import erkennt Bestandsspiele,
  meldet nur echte Änderungen, dupliziert nicht.

## Capabilities

### Added Capabilities

- **`h4a-game-import`** — Server-seitiger Login bei Handball4All mit vom Admin eingegebenen
  Zugangsdaten, Abruf und Parsing der Spieltabelle, Zwei-Phasen-Diff-Import mit
  Template-/Mannschafts-/Venue-Zuordnung, Idempotenz über die BWHV-Spielnummer.

### Modified Capabilities

- **`venue-csv-import`** — Hallennummer (`Nummer`) wird beim Import gelesen und als
  `venues.hall_number` gespeichert/backfilled; Rück-Mapping auf Bestands-Venues per
  `(Name, Ort, Straße)`; Mehrdeutigkeit/No-Match → `NULL` + Report.

## Impact

- **Migration** `internal/db/migrations/042_h4a_import_ids.up.sql`/`.down.sql`:
  `venues.hall_number`, partial-unique Index, `games.external_id` + Index.
- **Neues Package** `internal/h4aimport/` — H4A-HTTP-Client (Login, xajax-Request mit
  `S`-Kodierung), Tabellen-Parser, Diff-Logik, `Handler` mit `preview`/`apply`.
- `internal/venues/handler.go` — `Import` liest `Nummer`, setzt `hall_number` (Backfill
  per `(name,city,street)`), erweitertes `importResult` (matched/ambiguous/unmatched).
- `internal/app/router.go` — Routen `POST /api/games/import/h4a/preview` und `.../apply`
  im Vorstand-Tier (Admin-Bypass).
- `web/src/pages/KalenderPage.tsx` + neue Komponente `H4AImportModal.tsx` — Import-Dialog
  (Credentials-Eingabe, Diff-Anzeige, Template-/Mannschafts-Picker).
- `web/src/pages/AdminVenuesPage.tsx` — Ergebnisanzeige um hall_number-Report erweitern.
- **Tests**: `internal/h4aimport/*_test.go` (Parser gegen HTML-Fixture, Diff-Logik,
  Auth-Gating, Credential-Nichtpersistenz), `internal/venues/handler_test.go`
  (hall_number-Backfill, Ambiguität), Vitest für das Import-Modal.
- **Sicherheit/Doku**: neue Vertrauensklasse „TeamWERK nimmt fremde Zugangsdaten
  entgegen" — Regeln (nie loggen/persistieren, HTTPS-only) in `design.md` und
  `docs/agent/06-gotchas.md`.
