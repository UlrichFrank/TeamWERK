## 1. Backend: Auto-Vergabe beim Anlegen

- [x] 1.1 Helper `nextMemberNumber(ctx, db) (string, error)` in `internal/members/` ergänzen: `SELECT MAX(CAST(member_number AS INTEGER)) FROM members WHERE member_number GLOB '[0-9]*'`; bei kein Bestand → `"1"`, sonst `max+1`.
- [x] 1.2 Create-Handler (`internal/members/handler.go` ~Z.323-364) so ändern, dass `req.MemberNumber` ignoriert und immer `nextMemberNumber(...)` verwendet wird; Vergabe + INSERT in einer Transaktion (`BEGIN IMMEDIATE`) oder Retry bei Unique-Verletzung.

## 2. Backend: Read-only mit Admin-Override + 409

- [x] 2.1 Update-Handler (`internal/members/handler.go` ~Z.545-631): `claims := auth.ClaimsFromCtx(...)`; `member_number` nur ins `UPDATE` übernehmen, wenn `claims.Role == "admin"`, sonst bestehenden DB-Wert beibehalten.
- [x] 2.2 Vor dem Admin-Override Eindeutigkeit prüfen (`SELECT id FROM members WHERE member_number=? AND id<>?`) → bei Treffer HTTP 409 mit klarer Meldung; `honorar`-Leerungslogik unverändert lassen.

## 3. Backend: Konflikt-Flag im List-Endpoint

- [x] 3.1 `Member`-Struct um Feld `MemberNumberConflict string` (`json:"member_number_conflict"`) erweitern; Werte `"duplicate" | "non_numeric" | "missing" | ""`.
- [x] 3.2 In `GET /api/members` (`~Z.205-218`) Konflikt-Typ je Item bestimmen: Dubletten über Aggregat-Abfrage, nicht-numerisch via GLOB, `missing` für Nicht-`honorar` ohne Nummer; Honorar ausnehmen. Auf Redaction achten (kein Nummernwert leaken).

## 4. Frontend

- [x] 4.1 `MemberStammdatenTab.tsx` (~Z.251-260): Nummer-Feld read-only anzeigen, außer `user.role === 'admin'` (dann editierbarer Input mit `brand-*`-Klassen). 409-Fehler aus dem PUT als Alert (`brand-danger-light`) anzeigen.
- [x] 4.2 `Member`-Interface (Listenseite + Detailseite) um `member_number_conflict` ergänzen; in der Mitglieder-Übersicht bei gesetztem Flag `AlertTriangle` (`brand-danger`, `aria-label`/Tooltip mit Konflikt-Typ) rendern.

## 5. Tests (Test-Anforderungen)

- [x] 5.1 `internal/members/handler_test.go`: Create vergibt `max+1` und ignoriert Client-`member_number` (Happy-Path, 201/200).
- [x] 5.2 PUT als `admin` ändert Nummer erfolgreich (200); PUT mit bereits vergebener Nummer → 409.
- [x] 5.3 PUT als Nicht-Admin (Vorstand) lässt `member_number` unverändert, speichert übrige Felder.
- [x] 5.4 List-Endpoint liefert Konflikt-Flag korrekt für alle drei Typen (Dublette, nicht-numerisch, fehlend) und KEIN `missing` für Honorar.

## 6. Verifikation & Abschluss

- [x] 6.1 `make test` / `go vet` / `pnpm -C web build` grün; `/verify-change` durchlaufen (Route→Tests, brand-Tokens, lucide-Icons).
- [x] 6.2 `openspec validate mitgliedsnummer-auto-vergabe --strict` ohne Fehler.
