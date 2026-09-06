## Why

Die System-Rolle `presseteam` (Migration 019, hierarchisch `admin ⊇ presseteam ⊇
standard`) trägt genau **eine** Unterscheidung: wer einen Spielbericht schreiben darf. Der
Preis dafür ist eine dritte Achse im Berechtigungsmodell, die neben den System-Rollen und
den Vereinsfunktionen mitgepflegt werden muss — im `users.role`-CHECK, in
`invitation_tokens.role`, in der Nutzerverwaltung, in `RoleRoute`, in der Permissions-Matrix
und im Architektur-Gate.

Der Nutzen rechtfertigt das nicht: Wer einen Bericht schreibt, ist ohnehin dadurch bestimmt,
dass ihm der **Spielbericht-Dienst** gehört (`duty_slots.assigned_user_id`) — diese Prüfung
bleibt unverändert bestehen. Die Rolle ist eine zweite Hürde vor derselben Tür. Und
veröffentlicht wird ohnehin nichts ohne Freigabe: der Publish-Pfad hängt an der
Vereinsfunktion `medien`/`vorstand`, nicht an dieser Rolle.

Die Rolle entfällt daher ersatzlos. Was bisher nur ein Presseteam-Nutzer durfte, darf künftig
jeder eingeloggte Nutzer.

## What Changes

- **Migration** (`056`): `users.role` und `invitation_tokens.role` werden auf
  `CHECK (role IN ('admin','standard'))` zurückgebaut; Bestandszeilen mit
  `role='presseteam'` werden vorher auf `standard` gesetzt.
- **Autor-Tier der Spielberichte** (`GET /api/match-reports/my`, `POST /api/match-reports`,
  `DELETE /api/match-reports/{id}`, `POST /api/match-reports/{id}/submit-for-review`)
  verliert sein `RequireRole`-Gate und liegt künftig im Tier *Authenticated*. Die
  fachlichen Prüfungen bleiben: `Create` verlangt weiterhin den Besitz des referenzierten
  Spielbericht-Slots, `submit-for-review`/`Delete` weiterhin die Autorenschaft.
- **Spielbericht-Dienst ziehen**: der Rollen-Guard in `duties` (`assertSlotTakePermitted`,
  HTTP 403 `role_required`) entfällt. Ein Spielbericht-Slot wird gezogen wie jeder andere
  Dienst.
- **Nav/Routing**: „Spielberichte" (`/spielberichte`) ist für alle eingeloggten Nutzer
  sichtbar und erreichbar.
- **Nutzerverwaltung**: die Rolle ist nicht mehr vergebbar; `PUT /api/users/{id}/role` und
  die Einladung akzeptieren nur noch `admin` und `standard` (`400 invalid role` sonst).

Unberührt bleibt die Vereinsfunktion **`medien`** — sie ist die Freigeber-Achse
(`GET /pending`, `POST /publish`) und hat mit dieser Rolle nichts zu tun.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `auth`: `users.role` kennt nur noch `admin` und `standard`
- `match-reports`: Draft-Erstellung ohne Rollen-Gate
- `duties`: Spielbericht-Slot ohne Rollen-Sonderregel
- `pii-route-authz`: Spielbericht-Slot-Guard entfällt

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `POST /api/duty-slots/{id}/take` | `TestClaimSlot_Spielbericht_StandardUserDarfZiehen` | 204 statt 403 — der Rollen-Guard ist weg |
| `POST /api/match-reports` | `TestCreate_StandardUserMitSlot` | 201 für einen `standard`-User, dem der Spielbericht-Slot gehört |
| `POST /api/match-reports` | `TestCreate_StandardUserOhneSlot` | 403 `slot_not_owned` — der Slot-Besitz bleibt die einzige Hürde |
| `PUT /api/users/{id}/role` | `TestUpdateUserRole_PresseteamAbgelehnt` | 400 `invalid role`; `users.role` unverändert |
| Migration `056` | `TestMigration056_PresseteamWirdStandard` | Bestandszeile mit `role='presseteam'` steht danach auf `standard`; ein `INSERT … 'presseteam'` scheitert am CHECK |

Garantierte Invariante: Nach der Migration existiert keine Zeile mit `role='presseteam'` in
`users` oder `invitation_tokens`, und keine Route verlangt die Rolle noch.
