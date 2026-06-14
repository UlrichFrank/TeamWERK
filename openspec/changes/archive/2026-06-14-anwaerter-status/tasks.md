## 1. Datenbank-Migration

- [x] 1.1 Migration `038_anwaerter_status.up.sql` anlegen — members-Tabelle per Rebuild neu anlegen mit CHECK (`... 'honorar','anwaerter'`), alle Views und Indizes neu erstellen
- [x] 1.2 Migration `038_anwaerter_status.down.sql` anlegen — Tabelle zurückbauen ohne `anwaerter` im CHECK-Constraint
- [x] 1.3 `make migrate-up` lokal ausführen und prüfen dass INSERT mit `status='anwaerter'` akzeptiert wird

## 2. Backend

- [x] 2.1 `normalizeStatus` in `internal/members/handler.go` um `anwaerter` erweitern (wie `honorar` — direkter Match, kein Mapping)
- [x] 2.2 Prüfen ob `PUT /api/members/:id/status` eine Allowlist hat und `anwaerter` dort eintragen
- [x] 2.3 Tests: `TestMemberStatus_Anwaerter` — POST /api/members mit status=anwaerter (201), PUT /api/members/:id/status mit status=anwaerter (200), PUT mit ungültigem Status (400)

## 3. Frontend — Mitgliederverwaltung

- [x] 3.1 Status-Dropdown in der Mitglied-Anlegen/Bearbeiten-Ansicht um Option „Anwärter" (`anwaerter`) erweitern
- [x] 3.2 Statusanzeige in der Mitgliederliste — `anwaerter` mit passendem Label rendern (analog zu `honorar`, `passiv` etc.)

## 4. Frontend — Kader-Ansicht

- [x] 4.1 In der Kader-Mitgliederliste für Einträge mit `status === 'anwaerter'` ein Badge „Anwärter" neben dem Namen anzeigen (Brand-Farbe, klein, gut sichtbar aber nicht dominant)
- [x] 4.2 Badge gilt für primären und erweiterten Kader gleichermaßen

## Test-Anforderungen

- Route `POST /api/members`: `TestMemberStatus_Anwaerter_Create` (201 mit status=anwaerter)
- Route `PUT /api/members/:id/status`: `TestMemberStatus_Anwaerter_Update` (200), `TestMemberStatus_Invalid` (400)
- Invariante: Kein Member kann mit einem Status außerhalb der erlaubten Werte angelegt werden
