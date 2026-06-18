## Context

Berechtigungslogik ist heute an drei Stellen verstreut: Router-Gates (grobe Gruppen), Handler-Inline-Checks
(71× `claims.Role`/`claims.HasFunction` + 181× `claims.UserID`) und Frontend-Konstrukte (~40 Vorkommen).
Dieselbe Persona-Definition existiert leicht verschieden an mehreren Stellen — jede Änderung am
Rollenmodell muss manuell an allen Stellen nachgezogen werden.

**Constraints:**
- Keine neuen externen Dependencies (RAM-Limit 1 GB VPS)
- Keine DB-Migration
- Keine Breaking Changes an der HTTP-API (neue Felder additiv)
- `permissions-baseline-tests` muss vorher grün sein (Sicherungsnetz)

## Goals / Non-Goals

**Goals:**
- Alle Berechtigungs-Predicates an einem Ort (`internal/policy/`)
- Server liefert genau die Daten, die ein Nutzer sehen darf (Data-Scoping)
- Frontend hat keine `hasFunction`/`user.role`-Vergleiche mehr
- Jeder API-Response trägt `_can`-Objekt, das Button-Sichtbarkeit steuert
- `/api/me` liefert Nav-Items + Capabilities

**Non-Goals:**
- Kein Policy-as-Code-Framework (kein Casbin, OPA, etc.)
- Kein RBAC-Admin-UI
- Keine Änderungen an JWT-Claims-Struktur oder Token-Lifecycle
- `kassierer` und `vorstand_beisitzer` bleiben bewusst ohne Code-Gates (reine ACL-Tags für
  Folder-Permissions)
- Kein Big-Bang-Refactor — iterativ pro Domäne, Pilot: `members`

## Decisions

### D1: Policy-Package statt Middleware-Erweiterung

`internal/policy/` ist ein eigenständiges Package, kein Mixin in `internal/auth/`.

**Alternativen:**
- *Middleware-Erweiterung*: Zu viel Logik in der HTTP-Schicht, nicht unit-testbar ohne HTTP-Context
- *Inline weiter*: Status quo, löst das Drift-Problem nicht

**Rationale:** Policy-Predicates müssen von Handlern, Queries *und* dem `me`-Endpoint genutzt werden.
Ein separates Package ohne HTTP-Abhängigkeit ist überall importierbar.

### D2: Drei Dateien im Policy-Package

```
internal/policy/
├── rules.go      — statische Predicates (claims only, kein DB)
│                     CanCreateGame(claims) bool
│                     CanEditMember(claims, memberUserID int) bool
│                     ScopeMembersQuery(claims) → SQL-WHERE-Fragment
│                     NavFor(claims) → []NavItem
├── folders.go    — datenbankgestützte Predicates (claims + DB)
│                     CanReadFolder(ctx, db, claims, folderID) bool
└── annotate.go   — _can-Annotation-Helpers für DTOs
                      MemberCan(claims, m Member) CanFlags
```

**Rationale:** Trennung statisch/datenbankgestützt ermöglicht vollständige Unit-Tests von `rules.go`
ohne DB-Fixture. `folders.go` braucht DB und bleibt separat.

### D3: `_can`-Schema

```json
{ "can": { "edit": true, "delete": false } }
```

snake_case, inline im DTO (nicht als separater Endpoint). Erweiterbar pro Domäne (z.B.
`can.fulfill` für Duty-Assignments).

**Alternativen:**
- *Separater `/api/permissions/{resource}/{id}`-Endpoint*: Extra Round-Trip, schlechtere DX
- *camelCase `_can`*: Inkonsistent mit übrigen DTOs, die snake_case nutzen

### D4: Single-Endpoint-Strategie

Ein Endpoint pro Resource, persona-gefiltert + `_can`-annotiert. Getrennte Endpoints nur wenn
Inhalt sich strukturell unterscheidet (nicht nur in Sichtbarkeit).

**Rationale:** Vereinfacht Client-Code; kein Request-Routing basierend auf Rolle im Frontend.

### D5: `/api/me` als Capabilities-Quelle

`GET /api/me` wird erweitert um:
```json
{
  "user": { … },
  "capabilities": ["create_game", "manage_members"],
  "nav": [{ "label": "Mitglieder", "route": "/members" }]
}
```

Nav-Items kommen aus `policy.NavFor(claims)`, nicht mehr aus `navModules[i].items[j].roles`.
`capabilities` ist eine string-Liste für Feature-Flags im Frontend (z.B. `"manage_duty_types"`).

**JWT-Stabilität:** `nav` und `capabilities` kommen aus `/api/me`, nicht aus dem JWT. Kein
Token-Refresh nötig bei Rollenänderung — nächster `/api/me`-Call liefert aktuellen Stand.

### D6: Pilot-Domäne `members`

`members` hat alle Patterns (Ownership-Check, Role-Gate, Liste + Detail, Schreiboperationen)
und ist die kleinste Domäne. Erst nach grünen Tests dort werden weitere Domänen migriert.

## Risks / Trade-offs

- **Regressions beim Query-Scoping** → Mitigation: Baseline-Tests vor Start;
  jede Domäne einzeln migrieren und Tests laufen lassen
- **Performance bei Pro-Item-`_can`** → Mitigation: Bei ≤1000 Items unkritisch (pure Go-Logik,
  kein extra DB-Call pro Item); bei größeren Listen Profiling vorher
- **Frontend-Migration zieht sich** → Mitigation: `hasFunction`/`hasAnyFunction` bleiben als
  Deprecated-Wrapper bis alle Aufrufer migriert; kein Feature-Flag nötig
- **`NavFor`-Logik divergiert von tatsächlichen Router-Gates** → Mitigation: Policy-Tests prüfen
  Konsistenz; NavFor-Output ist deklarativ in `rules.go` gehalten, neben den Gates

## Migration Plan

1. `permissions-baseline-tests` abschließen (Voraussetzung)
2. `internal/policy/rules.go` + `annotate.go` anlegen (keine Verhaltensänderung)
3. Pilot `members`: Handler-Inline-Checks durch Policy-Calls ersetzen, `_can` an DTOs
4. Tests laufen lassen — kein Baseline-Test darf brechen
5. `/api/me` um `capabilities` + `nav` erweitern
6. Frontend-Migration: AppShell auf `/api/me`-Nav umstellen, dann Pages einzeln
7. Iterativ weitere Domänen: `games`, `duties`, `kader`, `folders`
8. `internal/policy/folders.go` für data-driven Folder-ACLs
9. `hasFunction`/`hasAnyFunction` aus AuthContext entfernen (letzter Schritt)

**Rollback:** Jeder Schritt ist unabhängig deploybar. Policy-Calls sind semantisch äquivalent zu
den Inline-Checks, die sie ersetzen — kein Verhalten ändert sich.
