## Why

Die RSVP-Endpunkte `POST /api/training-sessions/{id}/respond` und `POST /api/games/{id}/respond` enthalten eine **Autorisierungslücke (IDOR)**: Jeder eingeloggte Standard-User kann eine Rückmeldung für **jede beliebige `member_id`** abgeben — nicht nur für sich selbst oder seine eigenen Kinder.

Ursache: Die Handler verzweigen mit `switch claims.Role` auf `case "spieler"` / `case "elternteil"`. Diese Werte existieren als System-Rolle **nicht** (`users.role` hat `CHECK (role IN ('admin','standard'))`). Jeder Nicht-Admin landet daher im `default`-Zweig, der `memberID = req.MemberID` **ohne** `parentHasChild`- oder Ownership-Prüfung übernimmt. Die `parentHasChild`-Funktion und die `case`-Zweige sind toter Code.

Die `eltern-rsvp`-Spec fordert bereits „Elternteil versucht fremdes Kind zu melden → 403", aber dieses Verhalten ist **nicht durchgesetzt** (Spec/Code-Drift). Entdeckt bei der Umsetzung von `erw-kader-eltern-rueckmeldung`.

## What Changes

- **BREAKING (Sicherheit):** RSVP-Schreibzugriff wird auf eigene Person + eigene Kinder eingeschränkt. Bisher de facto unbeschränkt für alle eingeloggten User.
- Den toten `switch claims.Role`-Block in `Respond` (trainings) und `RespondToGame` (games) durch eine rollen-/fähigkeitsbasierte Prüfung ersetzen:
  - **Manage-Berechtigte** (admin · trainer/sportliche_leitung/vorstand des betroffenen Teams) dürfen weiterhin für **jedes** Mitglied antworten.
  - **Sonstige User** dürfen nur für ihr **eigenes** Member-Record antworten oder für ein über `family_links` verifiziertes **Kind** (`parentHasChild`); jede andere `member_id` → **403**.
  - Fehlende `member_id` bei nicht eindeutig auflösbarem eigenem Member → **400** (unverändert).
- Beide Endpunkte erhalten Tests für den 403-Pfad (fremdes Member / fremdes Kind).

## Capabilities

### New Capabilities

_Keine._

### Modified Capabilities

- `eltern-rsvp`: Die Anforderung „Eltern können RSVPs für Kinder abgeben" wird so präzisiert, dass die Autorisierung **rollenmodell-korrekt** (System-Rolle `standard` + `claims.IsParent`/`family_links`, nicht die nicht-existente Rolle `elternteil`) und **tatsächlich durchgesetzt** ist: fremde `member_id` → 403; Manage-Berechtigte ausgenommen.

## Impact

- **Code:** `internal/trainings/handler.go` (`Respond`), `internal/games/handler.go` (`RespondToGame`). Entfernt toten `case "spieler"/"elternteil"`; `parentHasChild` wird endlich genutzt. Ggf. kleine Helper für „darf für member X antworten".
- **APIs:** `POST /api/training-sessions/{id}/respond`, `POST /api/games/{id}/respond` — verschärfte Autorisierung (403 statt stillem Erfolg für fremde Member). Kein Vertrags-/Schemafeld geändert.
- **Verhalten:** Legitime Flows (eigene Rückmeldung, Eltern für eigenes Kind inkl. erw. Kader, Trainer/Vorstand/Admin für Teammitglieder) bleiben unverändert.
- **DB:** keine Migration.
- **Tests:** neue Fehlerfälle in `internal/trainings/handler_test.go`, `internal/games/handler_test.go`.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Status / Erwartung | Invariante |
|---|---|---|---|
| `POST /api/training-sessions/{id}/respond` | `TestRespond_OwnMember_OK` | 204 | Eigene Rückmeldung erlaubt |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_OwnChild_OK` | 204 | Eltern für eigenes Kind erlaubt |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_ForeignMember_Forbidden` | 403 | Fremdes Member/Kind abgelehnt |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_TrainerForAnyMember_OK` | 204 | Manage-Berechtigte ausgenommen |
| `POST /api/games/{id}/respond` | `TestGameRespond_OwnChild_OK` | 204 | Eltern für eigenes Kind erlaubt |
| `POST /api/games/{id}/respond` | `TestGameRespond_ForeignMember_Forbidden` | 403 | Fremdes Member/Kind abgelehnt |
