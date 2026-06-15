## 1. Datenbank-Migration

- [ ] 1.1 Neue Migration `internal/db/migrations/043_rename_mitfahrgelegenheiten.up.sql` mit `ALTER TABLE mitfahrgelegenheiten RENAME TO mitfahrten;` anlegen
- [ ] 1.2 Down-Migration `043_rename_mitfahrgelegenheiten.down.sql` mit `ALTER TABLE mitfahrten RENAME TO mitfahrgelegenheiten;` anlegen
- [ ] 1.3 Lokal `make migrate-up` ausführen, prüfen dass FK aus `mitfahrt_paarungen` auf `mitfahrten(id)` zeigen (`.schema mitfahrt_paarungen`)
- [ ] 1.4 Lokal `make migrate-down` testen (roundtrip), danach wieder `make migrate-up`

## 2. Backend — SQL-Strings und Routen

- [ ] 2.1 In `internal/carpooling/handler.go` alle SQL-Strings mit `mitfahrgelegenheiten` zu `mitfahrten` ändern (List, Upsert, Delete-Queries)
- [ ] 2.2 In `internal/carpooling/paarungen_handler.go` alle SQL-JOINs auf `mitfahrgelegenheiten` zu `mitfahrten` ändern
- [ ] 2.3 In `internal/dashboard/handler.go` und `internal/scheduler/scheduler.go` SQL-Strings auf `mitfahrten` aktualisieren
- [ ] 2.4 In `cmd/teamwerk/main.go` Routen umbenennen: `/api/mitfahrgelegenheiten` → `/api/mitfahrten` (3 Routen: GET, POST, DELETE)
- [ ] 2.5 SSE-Broadcast-Strings `h.hub.Broadcast("mitfahrgelegenheiten")` → `Broadcast("mitfahrten")` (Handler + Paarungen-Handler)
- [ ] 2.6 Push-Deep-Link-Pfade `/mitfahrgelegenheiten` → `/mitfahrten` in `internal/carpooling/handler.go` und `paarungen_handler.go`
- [ ] 2.7 Push-Notification-Titel „Mitfahrgelegenheit" → „Mitfahrt" in `internal/carpooling/handler.go`
- [ ] 2.8 Code-Kommentare in `internal/carpooling/*.go` (`// GET /api/mitfahrgelegenheiten` etc.) auf neue Pfade aktualisieren

## 3. Backend — Tests

- [ ] 3.1 `internal/carpooling/handler_test.go` durchsehen: Routen-Aufrufe und ggf. Hilfs-Inserts auf neuen Tabellennamen ändern
- [ ] 3.2 Neuer Test `TestList_PathRenamed` (smoke): `GET /api/mitfahrten` liefert 200, `GET /api/mitfahrgelegenheiten` liefert 404
- [ ] 3.3 `/usr/local/go/bin/go test ./internal/carpooling/... ./internal/dashboard/...` grün

## 4. Frontend — Routen, Page, Pfade

- [ ] 4.1 Datei umbenennen: `web/src/pages/MitfahrgelegenheitenPage.tsx` → `web/src/pages/MitfahrtenPage.tsx` (`git mv`)
- [ ] 4.2 In `MitfahrtenPage.tsx` Komponenten-Name `MitfahrgelegenheitenPage` → `MitfahrtenPage`, alle `api.get/post/delete('/mitfahrgelegenheiten...')`-Aufrufe auf `/mitfahrten` umstellen
- [ ] 4.3 In `web/src/App.tsx` Import + Route-Pfad anpassen (`<Route path="mitfahrten" element={<MitfahrtenPage />}/>`)
- [ ] 4.4 In `web/src/components/AppShell.tsx` Nav-Link-`to` auf `/mitfahrten` ändern (Label bleibt „Mitfahrten")
- [ ] 4.5 In `web/src/pages/DashboardPage.tsx` Links auf `/mitfahrten` und `useLiveUpdates`-Event-Filter auf `'mitfahrten'` umstellen
- [ ] 4.6 Globaler Suchlauf in `web/src/`: `grep -rni "mitfahrgelegenheit" web/src` — Ergebnis muss leer sein (nur kommentierte Übergangsstellen erlaubt, hier aber keine)

## 5. Dokumentation und Changelog

- [ ] 5.1 `CLAUDE.md` API-Routen-Liste aktualisieren: `/api/mitfahrgelegenheiten*` → `/api/mitfahrten*`
- [ ] 5.2 `docs/anleitung-spieler.md` und `docs/anleitung-elternteil.md` auf „Mitfahrten" / `/mitfahrten` umstellen
- [ ] 5.3 `web/public/CHANGELOG.md` Eintrag „Mitfahrgelegenheiten heißen jetzt einheitlich Mitfahrten" hinzufügen
- [ ] 5.4 `docs/berechtigungen.md` durchsehen und ggf. Begriffe anpassen

## 6. Specs umbenennen

- [ ] 6.1 `git mv openspec/specs/mitfahrgelegenheiten-board openspec/specs/mitfahrten-board` und Spec-Inhalt anpassen (Routen, Begriffe → siehe Delta `specs/mitfahrten-board/spec.md`)
- [ ] 6.2 `git mv openspec/specs/mitfahrgelegenheiten-team-filter openspec/specs/mitfahrten-team-filter` und Inhalt anpassen
- [ ] 6.3 `git mv openspec/specs/mitfahrgelegenheiten-nav openspec/specs/mitfahrten-nav` und Inhalt anpassen
- [ ] 6.4 `git mv openspec/specs/mitfahrgelegenheiten-meine-filter openspec/specs/mitfahrten-meine-filter` und Inhalt anpassen
- [ ] 6.5 In `openspec/specs/mitfahrt-paarungen/spec.md`, `carpooling-notifications/spec.md`, `dashboard-migration/spec.md`, `sse-live-updates/spec.md`, `carpooling-team-filter/spec.md` Begriffe + Routen anpassen

## 7. Verifikation und Smoke-Test

- [ ] 7.1 `make build` läuft fehlerfrei durch
- [ ] 7.2 `/usr/local/go/bin/go test ./...` grün
- [ ] 7.3 Globaler grep `grep -rni "mitfahrgelegenheit" --exclude-dir=archive --exclude-dir=node_modules .` — erwartete Treffer: nur historische Archive-Pfade und ggf. `internal/carpooling/` Package-Pfad (englisch, bleibt)
- [ ] 7.4 Lokal Backend + Frontend starten, in Browser einloggen, `/mitfahrten` öffnen: Liste lädt, neuer Eintrag anlegbar, Paarung anfragbar
- [ ] 7.5 SSE-Event prüfen: zweites Browser-Fenster offen lassen, Eintrag anlegen → andere Seite aktualisiert automatisch

## 8. Commits und Deploy

- [ ] 8.1 Pro logischem Schritt einen Conventional-Commit (`chore(db): Migration 043 rename mitfahrgelegenheiten → mitfahrten`, `refactor(carpooling): Routen und SQL auf mitfahrten umbenannt`, `refactor(pwa): Frontend-Pfad und Page auf mitfahrten`, `docs: Begriff Mitfahrten konsistent`)
- [ ] 8.2 `make deploy` — Migration läuft auf VPS automatisch via embed
- [ ] 8.3 Prod-Smoke-Test auf `https://internal.team-stuttgart.org/mitfahrten`
- [ ] 8.4 OpenSpec-Change archivieren via `/opsx:archive`
