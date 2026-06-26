## Why

`UpdateUserRole` (`PUT /api/users/{id}/role`, `internal/auth/handler.go`) hängt unter `RequireClubFunction("vorstand")`. Das Vergeben der `admin`-Rolle durch Nicht-Admins ist korrekt geblockt (keine Selbst-Eskalation), aber es fehlt ein Schutz, der einen `vorstand` daran hindert, einen bestehenden `admin` auf `standard` **herabzustufen**; ebenso fehlt ein Selbst-/Last-Admin-Schutz. Rollenverwaltung ist sonst admin-only (Impersonate via `RequireRole("admin")`) → Separation-of-Duties-Lücke, Sabotage-Vektor (Sicherheitsaudit 2026-06-26, **B-7 Low**).

Mildernd (daher Low): dasselbe Tier exponiert bereits `DELETE /api/users/{id}`, womit derselbe Akteur den Admin-Account ohnehin löschen kann — die Demotion gewährt kaum neue Sabotagemacht. Trotzdem ist die Rollenänderung an einem Admin durch Nicht-Admins ein unsauberer Zustand.

## What Changes

- **Invariante:** Ein Aufrufer ohne System-Rolle `admin` kann (a) keinen Account mit aktueller Rolle `admin` herabstufen und (b) nicht die eigene Rolle ändern → jeweils HTTP 403. Das Vergeben der Rolle `admin` bleibt `admin` vorbehalten (bestehendes Verhalten).
- **Mechanismus (im Design zu entscheiden):** entweder Route nach `RequireRole("admin")` verschieben (konsistent mit Impersonate) **oder** Handler-Guard (`caller.Role != "admin"` lehnt Ziel-Admin-Änderung + Selbständerung ab). Empfehlung: Handler-Guard, da Vorstand legitime Nicht-Admin-Rollenpflege behalten soll — final im Design.
- **Permissions-Matrix** (`internal/permissions/matrix_test.go`) entsprechend anpassen.
- **Tests:** Nicht-Admin-`vorstand` → 403 beim Herabstufen eines Admins und bei Selbst-Rollenänderung; `admin` weiterhin erlaubt.

**Diese Proposal wird vorerst NICHT umgesetzt** (nur angelegt).

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `auth`: Neue Anforderung zur Autorisierung von Rollenänderungen (Schutz bestehender Admins vor Degradierung durch Nicht-Admins; Verbot der Selbst-Rollenänderung).

## Impact

- **Code:** `internal/auth/handler.go` (`UpdateUserRole`), ggf. `internal/app/router.go` (Tier), `internal/permissions/matrix_test.go`.
- **API-Verhalten:** zusätzliche 403-Fälle für Nicht-Admins; legitime Admin-Operationen unverändert.
- **Daten/Migration:** keine.
