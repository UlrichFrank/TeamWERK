## 1. Notification-Fassade

- [x] 1.1 Neues Package `internal/notifications/` mit `notifications.go` anlegen
- [x] 1.2 `filterByEmailPref(db, uids, category) []int` implementieren — Spiegel zu `push.FilterByPushPref`, prüft `notification_preferences.email_enabled=1`, Default 0
- [x] 1.3 `sendCategoryEmail(db, cfg, uid, category, title, body, url)` implementieren — lädt User-Email, ruft `mailer.Send(to, title, body+"\n\nDirektlink: …")`
- [x] 1.4 `Send(db, cfg, uids, category, title, body, url)` Fassade implementieren: ruft `push.SendToUsers` für Push-Cohorte (sync wie bisher), startet `go sendCategoryEmail` für Email-Cohorte
- [x] 1.5 Unit-Test in `notifications_test.go`: Disjunkte Listen werden je nach Preference-Setup korrekt befüllt (kann mit Inline-SQLite ohne Mocks laufen — sofern Test-DB-Helper vorhanden ist, sonst Test-Stub mit fester Liste)

## 2. Event-Delete: Notification + Konto-Rekomputation

- [x] 2.1 In `internal/games/handler.go`, `DeleteGame`: vor dem `DELETE FROM games` per `SELECT DISTINCT da.user_id, da.status, g.season_id, g.opponent, g.date FROM duty_assignments da JOIN duty_slots ds ON ds.id=da.duty_slot_id JOIN games g ON g.id=ds.game_id WHERE g.id=?` die Listen `assignedUIDs` und `fulfilledUIDs` plus Event-Metadaten holen
- [x] 2.2 In derselben Transaction nach dem Cascade-Delete: für jedes Element von `fulfilledUIDs` `duty_accounts.ist` per Aggregat-`UPDATE` neu setzen (siehe design.md Decisions)
- [x] 2.3 Nach `tx.Commit()` und nach `h.hub.Broadcast("games")`: `notifications.Send(h.db, h.cfg, assignedUIDs, "duties", "Dienst entfällt", "Dein Dienst zum {eventName} am {dd.mm.yyyy} wurde gelöscht.", "/dienste")` aufrufen
- [x] 2.4 `?delete_slots`-Query-Verzweigung in `DeleteGame` entfernen — eine einzige Transaction für alle Pfade
- [x] 2.5 Bestehende Team-weite Push (`"Spiel abgesagt"` an Team+Eltern) auf `notifications.Send(..., "games", ...)` migrieren

## 3. Migration der weiteren Notify-Aufrufer

- [x] 3.1 `internal/duties/handler.go` (`CreateSlot`, `DeleteSlot`) auf `notifications.Send(..., "duties", ...)` umstellen
- [x] 3.2 `internal/trainings/handler.go` (`CreateSession`, `DeleteSession`, `DeleteSeries`) auf `notifications.Send(..., "trainings", ...)` umstellen (CreateSession hatte gar keinen Push-Call — nichts zu migrieren)
- [x] 3.3 `internal/games/handler.go` (`CreateGame`, `UpdateGame` falls vorhanden) auf `notifications.Send(..., "games", ...)` umstellen
- [x] 3.4 `internal/carpooling/handler.go` Mutationsendpunkte auf `notifications.Send(..., "carpooling", ...)` umstellen
- [x] 3.5 `internal/auth/handler.go` (`RequestMembership`) auf `notifications.Send(..., "membership", ...)` umstellen
- [x] 3.6 `internal/scheduler/scheduler.go` Jobs für `games`, `trainings`, `carpooling` auf Fassade umstellen — `duty_reminders`-Job bleibt vorerst unverändert (eigener Email-Pfad), Migration in separatem Change
- [x] 3.7 Imports aufräumen: `internal/push` wird in den migrierten Files nur noch in den `notifications`-Internals verwendet (Hinweis: Package umbenannt in `internal/notify`, da `internal/notifications` bereits existiert und `internal/auth` importiert → Cycle)

## 4. Frontend-Cleanup

- [x] 4.1 `web/src/components/GameEditModal.tsx:80`: `?delete_slots=true` aus URL entfernen → `api.delete(\`/kalender/${game.id}\`)`
- [x] 4.2 `web/src/pages/SpieltagDetailPage.tsx:210`: `?delete_slots=true` aus URL entfernen → `api.delete(\`/kalender/${gameId}\`)`

## 5. Verifikation

- [x] 5.1 Integrationstest in `internal/games/handler_test.go`: Spiel mit 2 Diensten (1× assigned, 1× fulfilled mit 1h) löschen → Cascade + `duty_accounts.ist` für (Helper, Saison) wird auf 2.0 reduziert (verbleibende fulfilled-Stunden eines anderen Events bleiben drin)
- [ ] 5.2 Manuell: Generisches Event mit 1 Dienst (assigned) löschen → Push an Zugewiesenen, kein Konto-Effekt
- [ ] 5.3 Manuell: User mit `email_enabled=1` für „Dienste" und `push_enabled=0` → bei Event-Delete kommt Email, keine Push
- [x] 5.4 Integrationstest in `internal/games/handler_test.go`: Event ohne Dienste löschen → 204, Game-Zeile weg, kein Crash bei leerer Empfängerliste
- [ ] 5.5 Manuell: Spiel-Lösch-Dialog im `GameEditModal` und `SpieltagDetailPage` öffnen → Dialog-Kopie zeigt korrekt die Cascade-Hinweise (keine Regression)
