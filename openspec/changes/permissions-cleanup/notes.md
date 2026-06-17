# permissions-cleanup — Vor-Design-Notizen

> **Status:** Vor-Design-Stand aus Explore-Session vom 2026-06-17 mit Ulrich.
> **Voraussetzung:** `permissions-baseline-tests` MUSS abgeschlossen sein, bevor
> dieser Change gestartet wird. Die dort entstehenden Persona-Matrix-Tests sind
> der Sicherungsanker für den Refactor.

## Ausgangslage (Ist-Zustand)

Berechtigungs-Logik ist heute an drei Stellen verstreut, mit Drift zwischen
identischen Predicates:

1. **Router** (`internal/app/router.go`): 6 grobe Gates via
   `RequireRole` / `RequireClubFunction`.
2. **Handler** (`internal/*/handler.go`): 62+ Inline-Checks
   (`claims.Role == …`, `claims.HasFunction(…)`, Ownership-IDs, Team-Filter).
3. **Frontend** (`web/src/{App,components,pages}`): ~57 Vorkommen — `RoleRoute`,
   `navModules.roles`, lokale `const isXxx = …`-Konstruktionen in ~10 Pages.

Dieselbe Persona-Definition (z.B. "isTrainer") existiert leicht verschieden in
5 Pages, 8 Handlern und 2 Router-Gates. Das ist die Drift, die dieser Change
beseitigt.

## Architektonische Entscheidung

> **Der Server liefert genau die Daten, die ein Nutzer sehen darf.
> Der Client rendert.**

Es geht **nicht** primär um Gate-Buttons, sondern um **Data-Scoping**:
das Frontend hat keine Persona-Logik mehr und keine
`hasFunction`/`user.role`-Vergleiche (außer absoluten UX-Spezialfällen).

### Was wohin wandert

| Heute im Frontend (verschwindet) | Morgen im Backend |
|---|---|
| `navModules[i].roles: [...]` | `GET /api/me` liefert `{ nav: [{label,route}, …] }` |
| `games.filter(g => g.team_id === me)` | `GET /api/games` filtert via SQL-WHERE pro Persona |
| `isTrainer && <Bearbeiten/>` | Jedes Item trägt `_can: { edit, delete, … }` |

### Modul-Struktur

```
internal/policy/
├── rules.go      — code-driven Predicates (statisch, claims-only)
│                     CanCreateGame(claims) bool
│                     CanEditMember(claims, m) bool
│                     ScopeMembersQuery(claims) → SQL-WHERE-Fragment
│                     NavFor(claims) → []NavItem
├── folders.go    — data-driven Predicates (claims + DB)
│                     CanReadFolder(ctx, db, claims, folderID) bool
│                     (admin/vorstand pass-through, sonst ACL-JOIN)
└── annotate.go   — _can-Annotation-Helpers für DTOs
```

Drei Konsumenten der Policy:

```
internal/policy/
   │       │       │
   ▼       ▼       ▼
Middleware  Handler-     DTO-
(Gate)      Queries      Annotation
            (WHERE)      (_can)
```

## Vereinbarte Entscheidungen (Explore-Session)

1. **Reihenfolge:** `permissions-baseline-tests` zuerst abschließen, danach
   diesen Change starten. Begründung: ohne Baseline-Tests produziert der
   Refactor unsichtbare Regressionen.

2. **`_can` pro Resource:** Listen-Items und Detail-Responses tragen ein
   `_can`-Objekt mit den anwendbaren Aktionen. Frontend rendert via
   `{can.edit && <Btn/>}`. Default-Pattern für Action-Sichtbarkeit.

3. **Endpoint-Strategie:** Default = **ein** Endpoint pro Resource,
   persona-gefiltert + `_can`-annotiert. Modell B (getrennte Endpoints für
   strukturell unterschiedliche Inhalte) nur als **Ausnahme** — wenn der
   Inhalt sich nicht nur in Sichtbarkeit, sondern in Struktur unterscheidet.

4. **`vorstand_beisitzer` / `kassierer`:** Bleiben in der Code-Policy
   bewusst **leer**. Ihre Wirkung ist datengetrieben über Folder-ACLs
   (`folder_permissions` JOIN `member_club_functions`). Das ist **keine Lücke**,
   sondern Architektur-Statement: einige Funktionen sind reine
   Membership-Tags für ACL-Systeme, keine Code-Capabilities. Dokumentieren
   in `CLAUDE.md`-Rollen-Tabelle.

## Konsequenzen, die im Proposal adressiert werden müssen

- **Pagination + Filter:** `total` ist persona-spezifisch. Filter MUSS auch
  in den Count-Query.
- **Listen-Performance:** Pro-Item-`_can` ist bei ≤1000 Items unkritisch.
  Bei größeren Listen (Spiele über Saisons, Mitfahrgelegenheiten-Historie)
  Profiling vor dem Refactor.
- **JWT-Stabilität:** Wenn `NavFor` aus Vereinsfunktionen abgeleitet wird,
  muss `nav` entweder bei JWT-Refresh neu berechnet werden oder über
  `/api/me` separat geladen werden. Empfehlung: `/api/me` ist die Quelle,
  JWT enthält nur die rohen Claims.
- **Migrations-Risiko:** Keine DB-Migrations geplant (alles Code).
  Aber: Phantom-Funktion `sportvorstand` (siehe `cleanup-legacy-roles`)
  muss vorher weg sein.
- **Test-Strategie:** Die Baseline-Matrix-Tests aus
  `permissions-baseline-tests` sind die Sicherung. Refactor darf keinen
  Test brechen, sonst ist eine Regression entstanden.

## Was im Proposal noch offen ist

Zu klären, bevor `proposal.md` geschrieben wird:

- **Reihenfolge der Module:** Welche Domäne zuerst auf Policy umstellen?
  Vorschlag: `members` (kleinste Domäne mit allen Patterns) als Pilot,
  dann iterativ. NICHT alles auf einmal.
- **`_can`-Schema:** Welcher Naming-Standard? Vorschlag: snake_case
  `{ can: { edit: true, delete: false } }`, weil JSON-Output und konsistent
  mit übrigen DTOs.
- **Migration der Inline-Checks:** Big-Bang oder schrittweise?
  Vorschlag: schrittweise pro Domäne, Old + New parallel bis Test grün.
- **`/api/me`-Schema:** Was genau liefert es?
  Vorschlag: `{ user, capabilities: string[], nav: NavItem[] }` —
  alles, was das Frontend zum Aufbau braucht, in einem Call.

## Was wir explizit NICHT machen

- **Backwards-Compat-Shims** für Frontend-Konstrukte wie `hasFunction(...)`.
  Diese werden ersatzlos entfernt.
- **UI-Konzepte im Server:** Kein "view": "manage" in Responses. Server
  liefert Daten + Capabilities, Frontend entscheidet Darstellung.
- **Pre-Fetch im Frontend** ("vielleicht kann ich, vielleicht nicht, probier's
  und sieh, was kommt"). Wenn `_can.edit === false`, ist der Button nicht da.
  Keine Toast-getriebene UX.

## Verweise

- Vorgänger-Change: `permissions-baseline-tests` (in Arbeit, 0/36 Tasks)
- Verwandt: `cleanup-legacy-roles` (Phantom-Funktion `sportvorstand` entfernen)
- Doku: `CLAUDE.md` — Abschnitt "Rollen und Vereinsfunktionen"
- Auth-Helpers heute: `internal/auth/middleware.go`, `internal/auth/claims.go`
- Frontend-Helpers heute: `web/src/contexts/AuthContext.tsx`
  (`hasFunction`, `hasAnyFunction`)
