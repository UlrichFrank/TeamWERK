## 1. Datenbank

- [x] 1.1 Migration `055_broadcast_targets.up.sql` / `.down.sql`: Tabelle `broadcast_targets` anlegen, `broadcasts` ohne `target_type` neu aufbauen (Muster aus `049`, `PRAGMA foreign_keys=OFF`-Hinweis als Kommentar), Bestandswerte als je eine Zielzeile übernehmen; `down` stellt `target_type` verlustbehaftet wieder her (im SQL kommentiert)
- [x] 1.2 Test: Bestands-Broadcast mit `broadcast_reads` überlebt die Migration unverändert und hat danach genau eine `broadcast_targets`-Zeile (`team_id IS NULL`)

## 2. Ziel-Vokabular und Auflösung

- [x] 2.1 `internal/chat/audiences.go`: Ziel-Typ (`kind` + optionale `teamId`), Whitelist um `team_spieler`/`team_eltern`/`team_trainer`/`alle_trainer` erweitern, `legacy` bleibt nicht setzbar
- [x] 2.2 Auflösung der Team-Kinds über die vorhandenen Queries aus `team_groups.go` (`teamGroupMemberQuery`, `allTrainersMemberQuery`) — keine zweite Definition von „Spieler eines Teams"
- [x] 2.3 `resolveTargets`: Vereinigung mehrerer Ziele zu einer deduplizierten User-Menge; Tests für Überschneidung (Elternteil zweier Kinder, Spieler mit Trainerfunktion)

## 3. Autorisierung

- [x] 3.1 `allowedTargets(ctx, userID, claims)` in `internal/chat`: vereinsweite Ziele nur für admin/vorstand/sportliche_leitung, `team_*` für Kader-Trainer der aktiven Saison (`kader_trainers`, **nicht** `user_accessible_teams`), `alle_trainer` für jeden mit Senderecht
- [x] 3.2 `internal/policy/rules.go`: `CanBroadcast` schließt die Vereinsfunktion `trainer` ein; Kommentar erklärt, dass die Capability grob ist und die Zielprüfung serverseitig gegen die Kader läuft
- [x] 3.3 `internal/permissions/matrix_test.go`: Erwartung für `POST /api/chat/broadcasts` und die neue GET-Route nachziehen
- [x] 3.4 `internal/policy/rules_test.go`: die Erwartung „reiner trainer hat NICHT broadcast_messages" umdrehen, mit Begründung im Testnamen

## 4. Routen

- [x] 4.1 `GET /api/chat/broadcast-targets` (Handler + Route in `internal/app/router.go`, Auth-Tier „Authenticated", 403 ohne Senderecht): liefert `kind`, `teamId`, `label`, `count`; Label über `db.TeamDisplayShort`; Ziele mit `count = 0` bleiben drin
- [x] 4.2 `SendBroadcast` auf `targets`-Array umstellen: Validierung (leer / unbekanntes `kind` / `teamId` fehlt oder ist überzählig / `legacy`) → 400, Allowlist-Prüfung je Ziel → 403 für den ganzen Request, danach Vereinigung auflösen und `broadcast_targets`-Zeilen schreiben
- [x] 4.3 Tests für beide Routen gemäß der Tabelle in `proposal.md — Test-Anforderungen` (Happy-Path + jeder Fehlerfall)

## 5. Frontend

- [x] 5.1 `ChatPage.tsx`: Composer holt `GET /chat/broadcast-targets` und rendert zwei Blöcke („Vereinsweit", „Gruppen") als Checkbox-Liste mit Empfängerzahl; Senden schickt `targets`
- [x] 5.2 Leere Ziel-Liste zeigt einen Hinweis statt eines leeren Dropdowns mit aktivem Senden-Button (Trainer ohne Kader der aktiven Saison)
- [x] 5.3 Bestätigung nach dem Senden bleibt „An N Empfänger gesendet" (nicht „N von M") — die Zahl ist der Fan-out, nicht die Summe der Gruppengrößen
- [x] 5.4 Vitest: Zielauswahl rendert beide Blöcke rollenabhängig, Mehrfachauswahl landet vollständig im Request, leere Liste zeigt den Hinweis

## 6. Abschluss

- [ ] 6.1 `/verify-change` (Build/Test/Lint + Projekt-Invarianten: Route→Tests, Mutation→`Broadcast`, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`)
- [ ] 6.2 Ankündigungstext für die Trainer entwerfen (was der neue Kanal ist, wofür weiterhin der Gruppenchat) und Ulrich zur Freigabe vorlegen
- [ ] 6.3 Change archivieren (`openspec archive mitteilung-team-gruppen`)
