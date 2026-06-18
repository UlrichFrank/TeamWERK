## Why

Berechtigungslogik ist heute an drei Stellen verstreut: 5 Router-Gates, 71 Handler-Inline-Checks
und ~40 Frontend-Vorkommen. Dieselbe Persona-Definition (z.B. „ist Trainer-ähnlich") existiert
leicht verschieden in mehreren Pages, Handlern und Router-Gates — das ist Drift, die bei jeder
Rollenänderung zu versteckten Regressionen führt. Zusätzlich filtert das Frontend Daten, die der
Server längst kennt, was zu doppelter Logik und API-Overhead führt.

## What Changes

- **Neues Package `internal/policy/`** konsolidiert alle Berechtigungs-Predicates an einem Ort
  (statisch/claims-only in `rules.go`, datenbankgestützt in `folders.go`, DTO-Annotation in
  `annotate.go`)
- **Alle API-Listen-Responses** tragen ein `_can`-Objekt pro Item (`{ edit: bool, delete: bool, … }`);
  das Frontend rendert Buttons nur noch via `can.edit` — kein `hasFunction`-Aufruf mehr
- **Persona-gefilterte Queries**: `GET /api/members`, `GET /api/games` etc. liefern serverseitig
  nur die Daten, die der anfragende Nutzer sehen darf; kein clientseitiges `.filter(g => …)` mehr
- **`GET /api/me` erweitert** um `capabilities: string[]` und `nav: NavItem[]`; das Frontend
  bezieht seine Navigations- und Feature-Sichtbarkeit daraus statt aus `navModules[i].roles`
- **Frontend-Bereinigung**: `hasFunction`, `hasAnyFunction`, lokale `const isTrainer = …`-Konstrukte
  und `user.role`-Vergleiche in Pages werden ersatzlos entfernt

Kein Breaking Change in der HTTP-API (neue Felder additive). Keine DB-Migration.

Pilot-Domäne: `members` — kleinste Domäne mit allen Patterns, danach iterativ weitere Domänen.

## Capabilities

### New Capabilities

- `policy-engine`: Zentrales `internal/policy/`-Package mit code-driven Predicates (statisch,
  claims-only), data-driven Predicates (mit DB-Lookup für Folder-ACLs) und `_can`-Annotation-Helpers
  für DTOs
- `can-annotations`: Jedes List- und Detail-DTO trägt ein `_can`-Objekt mit den anwendbaren Aktionen;
  Schema: `{ can: { edit: bool, delete: bool, … } }` (snake_case, erweiterbar pro Domäne)
- `me-capabilities`: `GET /api/me` liefert neben User-Daten auch `capabilities: string[]` (z.B.
  `"create_game"`, `"manage_members"`) und `nav: [{ label, route }]` — alles, was das Frontend zum
  Aufbau braucht, in einem Call

### Modified Capabilities

- `nav-visibility`: Nav-Items wechseln von frontend-computed (`navModules[i].items[j].roles`) zu
  backend-driven (`GET /api/me` → `nav`-Array); bestehende Sichtbarkeits-Regeln bleiben inhaltlich
  identisch, ändern sich aber in der Herkunft

## Impact

**Backend:**
- Neues Package `internal/policy/` (ca. 3 Dateien, kein neues Dependency)
- Handler in `internal/members/`, `internal/games/`, `internal/duties/`, `internal/kader/` erhalten
  Policy-Calls statt Inline-Checks; Ownership-Checks bleiben als `claims.UserID`-Vergleiche
- `internal/auth/handler.go` (`/api/me`-Endpoint) wird um `capabilities` und `nav` erweitert
- Router-Gates (`internal/app/router.go`) bleiben als grobe Gruppen-Gates erhalten; feinere Checks
  wandern in Policy

**Frontend:**
- `web/src/contexts/AuthContext.tsx`: `hasFunction`, `hasAnyFunction` bleiben vorerst als Wrapper
  erhalten bis alle Aufrufer migriert sind, werden dann final entfernt
- `web/src/components/AppShell.tsx`: `navModules`-Konfiguration wird durch `/api/me`-Response
  ersetzt
- Ca. 40 Vorkommen in Pages werden auf `can.*`-Props oder entfernt

**Keine:**
- Keine neuen externen Abhängigkeiten
- Keine DB-Migrationen
- Keine Änderungen an Refresh-Token- oder JWT-Claims-Struktur

**Voraussetzung:** `permissions-baseline-tests` muss abgeschlossen sein — die Persona-Matrix-Tests
sind der Sicherungsanker für diesen Refactor.
