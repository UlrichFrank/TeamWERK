## Context

Das Feature wird im Repo unter drei verschiedenen Namen geführt: „Mitfahrgelegenheiten" (DB-Tabelle, API-Pfade, Frontend-Route, Page-Komponente, drei Specs), „Mitfahrten" (Nav-Label, Paarungs-Tabellenpräfix `mitfahrt_paarungen`, Push-Notification-Titel teilweise) und „Carpooling" (Go-Package, Event-Tabelle `carpooling_events`). Das Nav-Label nutzt bereits „Mitfahrten" — der kürzere Begriff hat sich etabliert. Diese Inkonsistenz erschwert Onboarding und Code-Suche.

Das Go-Package `internal/carpooling/` bleibt unangetastet: Englischer Domain-Begriff, kollidiert nicht mit deutschen UI-Texten, und Package-Rename hätte hohen Diff-Aufwand ohne Nutzergewinn. Die `carpooling_events`-Tabelle bleibt aus demselben Grund.

## Goals / Non-Goals

**Goals:**
- Einheitlicher deutscher Name „Mitfahrten" / „Mitfahrt" überall in UI, API-Pfaden, DB-Tabellen und Specs.
- Spec-Folder-Namen reflektieren neuen Namen (Konsistenz mit künftiger Doku/Suche).
- Atomarer Deploy: nach `make deploy` ist nur noch der neue Name aktiv (kein Übergangszustand mit Doppel-Routen).
- Bestehende Daten (Mitfahrt-Einträge, Paarungen) bleiben vollständig erhalten.

**Non-Goals:**
- Go-Package-Rename (`internal/carpooling/`) — bleibt englisch.
- Tabellen-Rename `mitfahrt_paarungen`, `carpooling_events` — bereits konsistent bzw. internes Eventlog.
- API-Versionierung oder Backwards-Compat-Routen für externe Clients (gibt es nicht).
- Funktionale Änderungen am Mitfahrten-Feature.

## Decisions

### Entscheidung 1: SQLite `ALTER TABLE ... RENAME TO` für `mitfahrgelegenheiten` → `mitfahrten`

SQLite (≥ 3.25) unterstützt `ALTER TABLE alt RENAME TO neu`. Foreign-Key-Referenzen aus `mitfahrt_paarungen.biete_id`/`suche_id` werden bei aktivierter `PRAGMA legacy_alter_table=OFF` (default seit 3.26) und `PRAGMA foreign_keys=ON` automatisch in den FK-Definitionen aktualisiert (siehe SQLite docs: „Foreign key constraints are also rewritten" wenn `PRAGMA foreign_keys=ON`).

**Alternative:** Tabelle neu anlegen, Daten kopieren, alte löschen. Mehr Boilerplate, höheres Risiko bei Crash mitten in Migration. **Verworfen.**

**Risiko:** Falls eine SQLite-Version < 3.25 verwendet wird, schlägt `ALTER TABLE ... RENAME` fehl. Prod nutzt `modernc.org/sqlite` v1.x (SQLite-Engine ≥ 3.40) — kein Risiko. Lokale Migration prüft via `make migrate-up`.

### Entscheidung 2: Indizes ebenfalls umbenennen

Die Unique-Indizes `idx_mitfahr_biete_unique` (Migration 001) und `idx_mitfahr_suche_unique` (Migration 041) heißen bereits „mitfahr*", nicht „mitfahrgelegenheiten*" — kein Index-Rename nötig. Lediglich der `CREATE INDEX`-Tabellenbezug verweist intern auf den neuen Tabellennamen, was SQLite automatisch durchführt.

### Entscheidung 3: SSE-Event-Name umbenennen ohne Doppel-Broadcast

Das SSE-Event-String `"mitfahrgelegenheiten"` wird zu `"mitfahrten"`. Da Backend-Binary und Frontend-Bundle zusammen deployt werden (single binary, `embed.FS`), gibt es keinen Übergangszustand mit altem Frontend gegen neues Backend. Kein Doppel-Broadcast nötig.

### Entscheidung 4: Frontend-Pfad `/mitfahrgelegenheiten` ohne Redirect

Da App intern und URL nicht öffentlich verlinkt ist, kein Redirect von `/mitfahrgelegenheiten` → `/mitfahrten` nötig. Bestehende Browser-Bookmarks zeigen nach Deploy 404, sobald Frontend reloaded. Aufwand für Redirect (zusätzlicher Route mit `<Navigate>`) > Nutzen.

**Alternative:** `<Route path="mitfahrgelegenheiten" element={<Navigate to="/mitfahrten" replace />} />` für 30 Tage einbauen. Kann nachgereicht werden, falls Anwender klagen.

### Entscheidung 5: Spec-Folder physisch umbenennen

Statt REMOVED/ADDED-Akrobatik werden die Spec-Folder unter `openspec/specs/` per `git mv` umbenannt und die Inhalte angepasst. Archivierte Changes (`openspec/changes/archive/...`) bleiben mit altem Namen — historisch korrekt. Die Delta-Files in diesem Change beschreiben die Inhalts-Änderung in den neuen Folder-Namen.

**Alternative:** Old spec mit REMOVED, new spec mit ADDED. Verdoppelt Text, OpenSpec-Archive-Tooling muss zwei Specs synchronisieren. **Verworfen.**

## Risks / Trade-offs

- **[Risiko]** Frontend nach Deploy lädt mit altem Pfad `/mitfahrgelegenheiten` und zeigt 404 → **Mitigation:** Im Release-Hinweis (CHANGELOG) explizit erwähnen; ggf. nachträglich Redirect ergänzen.
- **[Risiko]** Push-Notifications vor Deploy haben `/mitfahrgelegenheiten`-Deeplink im Service-Worker-State → **Mitigation:** akzeptabel, kurzlebig — Klick führt zu 404, Nutzer öffnen App manuell.
- **[Risiko]** Migration 043 läuft auf einem Replica mit alter SQLite-Engine (< 3.25) fehl → **Mitigation:** Prod nutzt nur eine DB auf einem VPS; SQLite-Engine via `modernc.org/sqlite` aktuell. Lokal vor Deploy testen.
- **[Risiko]** Vergessen, alle Stellen umzubenennen → **Mitigation:** Task-Liste enthält finalen `grep -ri "mitfahrgelegenheit"`-Check; nur Archiv-Pfade und Go-Package dürfen Hits liefern.
- **[Trade-off]** Tabellen-Rename ist Breaking gegen Backups: ein Restore eines DB-Backups von vor Migration 043 läuft nur, wenn `migrate up` danach läuft. Standard-Procedure.

## Migration Plan

1. Code-Änderungen (Backend + Frontend + Migration) lokal entwickeln und testen.
2. Lokale Migration prüfen: `make migrate-up`, manueller Smoke-Test der Mitfahrten-Seite.
3. Commit-Reihe pro Task (Conventional Commits, scope=`carpooling` für Code, `db` für Migration, `pwa` falls Push betroffen).
4. `make deploy` — buildet Frontend, embedded ins Binary, rsync auf VPS, systemctl restart, `migrate up` läuft automatisch.
5. Smoke-Test auf Prod: Seite öffnen, neuen Eintrag anlegen, Paarung anfragen.

**Rollback:** `make migrate-remote-down` rollt Migration 043 zurück (Tabelle wird wieder zu `mitfahrgelegenheiten`). Vorheriges Binary aus `bin/`-Archiv (oder Git-Tag) deployen.

## Open Questions

- Sollen wir den 30-Tage-Redirect `/mitfahrgelegenheiten` → `/mitfahrten` doch einbauen? → Default: **nein**, on demand nachreichen.
- `internal/carpooling/` Package-Rename irgendwann? → Default: **nein**, separater Change wenn überhaupt.
