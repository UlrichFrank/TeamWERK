## 1. Migration — team_memberships zur VIEW umwandeln

- [x] 1.1 `internal/db/migrations/005_team_memberships_view.up.sql` anlegen: `DROP TABLE team_memberships`, dann `CREATE VIEW team_memberships` über `kader_members` UNION `kader_trainers` (beide via `kader.team_id`)
- [x] 1.2 `internal/db/migrations/005_team_memberships_view.down.sql` anlegen: VIEW droppen, ursprüngliche Tabelle wiederherstellen

## 2. Backend — Board-Handler anpassen

- [x] 2.1 In `internal/duties/handler.go`: Admin-Zweig im `Board`-Handler einbauen — wenn `claims.Role == "admin"`, kein team_id-Filter, alle Slots der aktiven Saison
- [x] 2.2 Query-Parameter `view=mine` auswerten: zusätzlicher JOIN auf `duty_assignments` und Filter `da.user_id = current_user` wenn `view=mine`

## 3. Backend — Aufräumen

- [x] 3.1 `AssignTeam`-Handler aus `internal/members/handler.go` entfernen
- [x] 3.2 Route `POST /api/members/{id}/team-assignment` aus `cmd/teamwerk/main.go` entfernen

## 4. Frontend — Alte Seiten entfernen

- [x] 4.1 `web/src/pages/DutyBoardPage.tsx` löschen
- [x] 4.2 `web/src/pages/DutySlotsPage.tsx` löschen
- [x] 4.3 In `web/src/App.tsx`: Imports und Routen `/dienstboerse` und `/dienste → DutySlotsPage` entfernen

## 5. Frontend — Neue DutyPage anlegen

- [x] 5.1 `web/src/pages/DutyPage.tsx` erstellen: Basis-Struktur mit `useAuth`, Laden via `GET /api/duty-board`, gruppierte Ansicht (wie bisherige DutyBoardPage)
- [x] 5.2 Toggle „Meine Dienste" / „Alle Dienste" einbauen — nur für Admin+Trainer sichtbar; Meine-Ansicht ruft `GET /api/duty-board?view=mine` ab
- [x] 5.3 Vergangene-Toggle beibehalten (`showPast`-State)
- [x] 5.4 Eintragen/Austragen für alle Rollen (wie bisher DutyBoardPage)
- [x] 5.5 Zuteilungen aufklappen per Klick (`GET /api/duty-slots/{id}/assignments`) — nur für Admin+Trainer
- [x] 5.6 Erfüllt- und Geldersatz-Aktionen inline in aufgeklappten Zuteilungen — nur für Admin+Trainer, nur bei Status `pending`
- [x] 5.7 Löschen-Button (🗑) pro Slot — nur für Admin+Trainer; bei `slots_filled > 0` Bestätigungsdialog, sonst direkt löschen via `DELETE /api/duty-slots/{id}`

## 6. Frontend — Navigation und Routing

- [x] 6.1 In `web/src/App.tsx`: `DutyPage` importieren, Route `/dienste` auf `DutyPage` zeigen
- [x] 6.2 In `web/src/components/AppShell.tsx`: Nav-Einträge „Dienstbörse" und „Dienst-Planung" entfernen, neuen Eintrag „Dienste" → `/dienste` hinzufügen (alle eingeloggten Rollen)

## 7. Qualitätssicherung

- [x] 7.1 Manueller Test: Spieler sieht Slots seines Kader-Teams
- [x] 7.2 Manueller Test: Trainer ohne Mitgliedsprofil sieht Slots seiner Trainer-Teams
- [x] 7.3 Manueller Test: Admin sieht alle Slots ungefiltert
- [x] 7.4 Manueller Test: Toggle Meine/Alle funktioniert für Admin und Trainer
- [x] 7.5 Manueller Test: Löschen mit Bestätigung bei belegtem Slot
- [ ] 7.6 Deploy auf VPS, Smoke-Test auf https://intern.team-stuttgart.org


