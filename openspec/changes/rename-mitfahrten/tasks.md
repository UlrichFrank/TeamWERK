## 1. Datenbank-Migration

- [ ] 1.1 `internal/db/migrations/002_rename_mitfahrgelegenheiten.up.sql` mit `ALTER TABLE mitfahrgelegenheiten RENAME TO mitfahrten;` anlegen
- [ ] 1.2 `internal/db/migrations/002_rename_mitfahrgelegenheiten.down.sql` mit `ALTER TABLE mitfahrten RENAME TO mitfahrgelegenheiten;` anlegen
- [ ] 1.3 `make migrate-up`, dann FK prüfen: `.schema mitfahrt_paarungen` → `biete_id`/`suche_id` referenzieren `mitfahrten(id)`
- [ ] 1.4 `make migrate-down`-Roundtrip testen, danach wieder `make migrate-up`

## 2. Backend — Routen, SQL, Events, Nav

- [ ] 2.1 `internal/app/router.go`: 3 Routen `/api/mitfahrgelegenheiten` → `/api/mitfahrten` (`GET`/`POST`/`DELETE …/{id}`) + Kommentar
- [ ] 2.2 `internal/carpooling/handler.go`: alle SQL-Strings (`SELECT/INSERT/UPDATE/DELETE … mitfahrgelegenheiten`) auf `mitfahrten`; Route-Kommentare aktualisieren
- [ ] 2.3 `internal/carpooling/paarungen_handler.go`: alle SQL-JOINs/SELECTs auf `mitfahrten`
- [ ] 2.4 `internal/dashboard/handler.go`: SQL-JOINs `mitfahrgelegenheiten` → `mitfahrten`
- [ ] 2.5 `internal/scheduler/scheduler.go`: SQL (`FROM mitfahrgelegenheiten`) + Push-Deep-Link `/mitfahrgelegenheiten` → `/mitfahrten`
- [ ] 2.6 SSE-Broadcast: `h.hub.Broadcast("mitfahrgelegenheiten")` → `Broadcast("mitfahrten")` in `handler.go` und `paarungen_handler.go`
- [ ] 2.7 Push in `handler.go`/`paarungen_handler.go`: Deep-Link `/mitfahrgelegenheiten` → `/mitfahrten`, Titel „Mitfahrgelegenheit" → „Mitfahrt", Body „sucht eine Mitfahrgelegenheit" → „sucht eine Mitfahrt"
- [ ] 2.8 `internal/policy/rules.go`: `NavItem{"Mitfahrten", "/mitfahrgelegenheiten"}` → Pfad `/mitfahrten` (Label bleibt)

## 3. Backend — Tests

- [ ] 3.1 `internal/carpooling/handler_test.go`: Routen-Aufrufe + Insert-SQL auf `mitfahrten`
- [ ] 3.2 `internal/carpooling/team_push_test.go`: Insert-/Count-SQL auf `mitfahrten`
- [ ] 3.3 `internal/dashboard/handler_test.go`: Insert-SQL auf `mitfahrten`
- [ ] 3.4 `internal/permissions/matrix_test.go`: Routen-Pfade `/api/mitfahrgelegenheiten` → `/api/mitfahrten` (+ Kommentar)
- [ ] 3.5 Neuer Smoke-Test in `handler_test.go`: `GET /api/mitfahrten` → 200, `GET /api/mitfahrgelegenheiten` → 404
- [ ] 3.6 `/usr/local/go/bin/go test ./internal/carpooling/... ./internal/dashboard/... ./internal/permissions/...` grün

## 4. Frontend — Route, Page, Pfade

- [ ] 4.1 `git mv web/src/pages/MitfahrgelegenheitenPage.tsx web/src/pages/MitfahrtenPage.tsx`
- [ ] 4.2 In `MitfahrtenPage.tsx`: Komponentenname → `MitfahrtenPage`, alle `api.*('/mitfahrgelegenheiten…')` → `/mitfahrten`, `useLiveUpdates`-Filter `'mitfahrgelegenheiten'` → `'mitfahrten'`, sichtbare Texte („Mitfahrgelegenheit eintragen") → „Mitfahrt eintragen"
- [ ] 4.3 `web/src/App.tsx`: Import + Route-Pfad (`<Route path="mitfahrten" element={<MitfahrtenPage />}/>`)
- [ ] 4.4 `web/src/components/AppShell.tsx`: Nav-`to` → `/mitfahrten` (Label bleibt „Mitfahrten")
- [ ] 4.5 `web/src/pages/DashboardPage.tsx`: alle Links auf `/mitfahrten`, `useLiveUpdates`-Event-Filter `'mitfahrgelegenheiten'` → `'mitfahrten'`
- [ ] 4.6 `web/src/test/renderAsPersona.tsx`: Routenliste `'/mitfahrgelegenheiten'` → `'/mitfahrten'`
- [ ] 4.7 `grep -rni "mitfahrgelegenheit" web/src` → leer

## 5. Dokumentation und Changelog

- [ ] 5.1 `CLAUDE.md`: deutsche Routen-Ausnahme `/api/mitfahrgelegenheiten` → `/api/mitfahrten` aktualisieren
- [ ] 5.2 `docs/anleitung-spieler.md` und `docs/anleitung-elternteil.md` auf „Mitfahrten" / `/mitfahrten`
- [ ] 5.3 `docs/berechtigungen.md` Begriffe/Pfade anpassen
- [ ] 5.4 `web/public/CHANGELOG.md`: Eintrag „Mitfahrgelegenheiten heißen jetzt einheitlich Mitfahrten" (alte historische Einträge unverändert lassen)

## 6. Specs — MODIFIED-Deltas (Folder behalten ihre Namen)

> Nur Bezeichner-Referenzen ändern; Folder NICHT umbenennen. Pro betroffenem Requirement Volltext aus der Live-Spec kopieren und Bezeichner ersetzen.

- [ ] 6.1 `mitfahrgelegenheiten-board`, `mitfahrgelegenheiten-nav`, `mitfahrgelegenheiten-team-filter`: Routen/Pfad/Komponente/Begriff → neue Bezeichner
- [ ] 6.2 `carpooling-team-filter`, `carpooling-elternzugang`, `carpooling-notifications`: `/api/mitfahrten`, Push-Titel/Body, Deep-Link
- [ ] 6.3 `dashboard-migration`, `dashboard-carpooling-hint`, `dashboard-offene-gesuche`: Pfad `/mitfahrten` bzw. Tabellenref `mitfahrten.*`
- [ ] 6.4 `sse-live-updates`: Event-String `mitfahrten`, Route, Komponente `MitfahrtenPage`
- [ ] 6.5 `number-spinner`: Datei `MitfahrtenPage.tsx`
- [ ] 6.6 `permissions`: Routenliste + Frontend-Pfad-Zeile
- [ ] 6.7 `openspec validate rename-mitfahrten --strict` → null Fehler

## 7. Verifikation und Smoke-Test

- [ ] 7.1 `make build` fehlerfrei
- [ ] 7.2 `/usr/local/go/bin/go test ./...` grün (inkl. Architektur-Test)
- [ ] 7.3 `grep -rni "mitfahrgelegenheit" --exclude-dir=archive --exclude-dir=node_modules .` → erwartete Resttreffer nur: `internal/carpooling/` (Package-Pfad, englisch), archivierte Changes
- [ ] 7.4 Lokal Backend + Frontend starten, `/mitfahrten` öffnen: Liste lädt, Eintrag anlegbar, Paarung anfragbar
- [ ] 7.5 SSE: zweites Fenster offen lassen, Eintrag anlegen → andere Seite aktualisiert automatisch
- [ ] 7.6 `/verify-change` ausführen

## 8. Commits und Deploy

- [ ] 8.1 Conventional Commits pro Schritt: `chore(db): Migration 002 rename mitfahrgelegenheiten → mitfahrten`, `refactor(carpooling): Routen, SQL, SSE-Event und Push auf mitfahrten`, `refactor(pwa): Frontend-Pfad und Page auf mitfahrten`, `docs: Begriff Mitfahrten konsistent`
- [ ] 8.2 `make deploy` (Migration läuft auf VPS automatisch via embed)
- [ ] 8.3 Prod-Smoke-Test `https://internal.team-stuttgart.org/mitfahrten`
- [ ] 8.4 Change archivieren via `/opsx:archive`

## Test-Anforderungen

| Route / Verhalten | Testname | Erwartung |
|---|---|---|
| `GET /api/mitfahrten` (Happy) | `TestList_OK` (bestehend, Pfad angepasst) | 200 + Liste zukünftiger Spiele |
| `GET /api/mitfahrgelegenheiten` (alt) | `TestList_OldPathGone` | 404 (alte Route existiert nicht mehr) |
| `POST /api/mitfahrten` (Happy) | `TestUpsert_OK` (bestehend) | 200 + Row in `mitfahrten` |
| `DELETE /api/mitfahrten/{id}` fremd | `TestDelete_Forbidden` (bestehend) | 403 |
| Berechtigungsmatrix | `internal/permissions/matrix_test.go` | unverändertes Tier, neue Pfade |

**Garantierte Invariante:** Nach Migration 002 existiert keine Tabelle `mitfahrgelegenheiten` und keine Route `/api/mitfahrgelegenheiten` mehr; alle Bestandsdaten sind unter `mitfahrten` erhalten und die FK aus `mitfahrt_paarungen` bleiben gültig.
