## Context

Das Feature wird im Repo unter drei Namen geführt: „Mitfahrgelegenheiten" (DB-Tabelle, API-Pfade, Frontend-Route, Page-Komponente, SSE-Event, mehrere Specs), „Mitfahrten" (Nav-Label in `policy/rules.go` und `AppShell.tsx`, Paarungs-Tabellenpräfix `mitfahrt_paarungen`, Push-Titel teilweise) und „Carpooling" (Go-Package, Tabelle `carpooling_events`). Das Nav-Label nutzt bereits „Mitfahrten" — der kürzere Begriff hat sich etabliert. Diese Inkonsistenz erschwert Onboarding und Code-Suche.

Das Go-Package `internal/carpooling/` bleibt unangetastet: englischer Domain-Begriff, kollidiert nicht mit deutschen UI-Texten, Package-Rename hätte hohen Diff-Aufwand ohne Nutzergewinn. `carpooling_events` bleibt aus demselben Grund.

> **Stand der Migrations:** Das Repo hat die Migrations zu einer einzigen `001_initial.up.sql` konsolidiert. Die nächste freie Nummer ist daher **002** (nicht 043 wie in einer früheren Fassung dieses Changes angenommen).

## Goals / Non-Goals

**Goals:**
- Einheitlicher deutscher Name „Mitfahrten" / „Mitfahrt" in UI, API-Pfaden, DB-Tabelle, SSE-Event und Page-Komponente.
- Atomarer Deploy: nach `make deploy` ist nur der neue Name aktiv (keine Doppel-Routen).
- Bestehende Daten (Mitfahrt-Einträge, Paarungen) bleiben vollständig erhalten.
- Specs konsistent: betroffene Routen-/Pfad-/Event-/Tabellen-Referenzen in den Live-Specs aktualisiert.

**Non-Goals:**
- Go-Package-Rename `internal/carpooling/` — bleibt englisch.
- Tabellen-Rename `mitfahrt_paarungen`, `carpooling_events`.
- OpenSpec-Capability-/Folder-Rename (siehe Entscheidung 5).
- API-Versionierung / Backwards-Compat-Routen — keine externen Clients.
- Frontend-Redirect vom alten Pfad.
- Funktionale Änderungen am Feature.

## Decisions

### Entscheidung 1: SQLite `ALTER TABLE … RENAME TO` (Migration 002)

`ALTER TABLE mitfahrgelegenheiten RENAME TO mitfahrten;`. Bei `PRAGMA foreign_keys=ON` (Default beim DB-Open dieses Projekts) und modernem SQLite (`legacy_alter_table=OFF`, Default seit 3.26) schreibt SQLite die FK-Definitionen referenzierender Tabellen automatisch um — d. h. `mitfahrt_paarungen.biete_id`/`suche_id REFERENCES mitfahrgelegenheiten(id)` wird zu `REFERENCES mitfahrten(id)`.

**Alternative:** Tabelle neu anlegen, Daten kopieren, alte löschen. Mehr Boilerplate, höheres Crash-Risiko. **Verworfen.**

**Risiko:** SQLite < 3.25 würde fehlschlagen. Prod nutzt `modernc.org/sqlite` (Engine ≥ 3.40) — kein Risiko. Down-Migration: `ALTER TABLE mitfahrten RENAME TO mitfahrgelegenheiten;`. Roundtrip lokal via `make migrate-up`/`make migrate-down` verifizieren, danach `.schema mitfahrt_paarungen` prüfen (FK zeigt auf `mitfahrten`).

### Entscheidung 2: Keine Index-Umbenennung nötig

Die partiellen Unique-Indizes heißen bereits `idx_mitfahr_biete_unique` / `idx_mitfahr_suche_unique` (nicht `…mitfahrgelegenheiten…`). Ihr interner Tabellenbezug wird beim `RENAME TO` automatisch auf `mitfahrten` gezogen. Kein `DROP/CREATE INDEX` erforderlich.

### Entscheidung 3: SSE-Event-Name ohne Doppel-Broadcast umbenennen

Event-String `"mitfahrgelegenheiten"` → `"mitfahrten"` in allen `h.hub.Broadcast(...)`-Aufrufen (`handler.go`, `paarungen_handler.go`) und im Frontend-`useLiveUpdates`-Filter (`MitfahrtenPage.tsx`, `DashboardPage.tsx`). Da Backend-Binary und Frontend-Bundle gemeinsam deployt werden (Single Binary, `embed.FS`), gibt es keinen Übergangszustand → kein Doppel-Broadcast.

### Entscheidung 4: Frontend-Pfad ohne Redirect

App intern, URL nicht öffentlich verlinkt → kein Redirect von `/mitfahrgelegenheiten`. Alte Bookmarks zeigen nach Reload den Standard-Not-Found-Zustand. Aufwand (`<Navigate>`-Route) > Nutzen.

**Alternative:** befristeter Redirect für 30 Tage — kann on demand nachgereicht werden.

### Entscheidung 5: Specs per MODIFIED-Delta statt Folder-Rename

Die Capability-/Spec-Folder unter `openspec/specs/` behalten ihre Namen (`mitfahrgelegenheiten-board`, `-nav`, `-team-filter`, `-meine-filter`). Nur die **Inhalte** der betroffenen Requirements werden per `## MODIFIED Requirements`-Delta auf die neuen Bezeichner gebracht.

**Begründung:** Symmetrisch zur Go-Package-Entscheidung — der Folder-Name ist ein interner Bezeichner ohne Nutzergewinn beim Rename. Ein echter Capability-Rename in OpenSpec erfordert REMOVED-(alt)+ADDED-(neu)-Deltas mit **Volltextkopie** aller Requirements; das hatte sich in der ersten Fassung dieses Changes bereits als veraltet/driftend erwiesen (die ADDED-Deltas beschrieben einen alten Board-Stand, während die Live-Spec inzwischen UI-Requirements wie chronologische Liste, Pills, Farbcodierung und Compact-Header trägt). MODIFIED-Deltas halten den Diff klein, robust und valide.

**Alternative:** REMOVED+ADDED-Folder-Rename. Verdoppelt Text, driftet erneut, höheres Fehlerrisiko. **Verworfen.**

### Entscheidung 6: Navigation an zwei Stellen anpassen

Der Nav-Eintrag existiert doppelt: backendseitig in `internal/policy/rules.go` (`NavItem{"Mitfahrten", "/mitfahrgelegenheiten"}`) und frontendseitig hartkodiert in `web/src/components/AppShell.tsx`. Beide Pfade müssen auf `/mitfahrten` — sonst zeigt einer der beiden ins Leere. Das Label „Mitfahrten" bleibt unverändert.

## Risks / Trade-offs

- **[Risiko]** Frontend nach Deploy mit altem Pfad → Not-Found. **Mitigation:** im CHANGELOG erwähnen; Redirect bei Bedarf nachreichen.
- **[Risiko]** Push-Notifications mit altem Deep-Link im Service-Worker-State. **Mitigation:** kurzlebig, akzeptabel; Klick führt zu Not-Found, Nutzer öffnen App manuell.
- **[Risiko]** Eine Stelle wird beim Rename vergessen. **Mitigation:** finaler `grep -rni "mitfahrgelegenheit"` — erlaubte Resttreffer nur in `internal/carpooling/` (Package-Pfad, englisch), `mitfahrt_paarungen`/`carpooling_events` und archivierten Changes.
- **[Trade-off]** Tabellen-Rename ist breaking gegen Backups: Restore eines Backups von vor Migration 002 läuft erst nach `migrate up`. Standard-Procedure.

## Migration Plan

1. Code (Backend + Frontend + Migration 002) lokal entwickeln und testen.
2. `make migrate-up`, dann `.schema mitfahrt_paarungen` prüfen (FK → `mitfahrten`); Smoke-Test der Seite. `make migrate-down`-Roundtrip, danach wieder `migrate-up`.
3. Commit-Reihe pro Task (Conventional Commits; scope `db` für Migration, `carpooling` für Code, `pwa`/Frontend für UI, `docs`).
4. `make deploy` — baut Frontend embedded, rsync, restart, `migrate up` läuft automatisch.
5. Prod-Smoke-Test: `/mitfahrten` öffnen, Eintrag anlegen, Paarung anfragen, SSE-Aktualisierung prüfen.

**Rollback:** `make migrate-remote-down` rollt Migration 002 zurück; vorheriges Binary deployen.

## Open Questions

- 30-Tage-Redirect `/mitfahrgelegenheiten` → `/mitfahrten` doch einbauen? → Default **nein**, on demand.
- `internal/carpooling/` irgendwann umbenennen? → Default **nein**, ggf. separater Change.
