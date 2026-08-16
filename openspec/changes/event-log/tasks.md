# Tasks — Event-Log

## 1. Datenbank

- [x] 1.1 Migrationsnummer bestimmen: `ls internal/db/migrations/ | sort -V | tail -1`. Nie eine Nummer ≤ aktueller DB-Version vergeben (golang-migrate überspringt sie lautlos).
- [x] 1.2 `0NN_user_events.up.sql`: Tabelle `user_events` nach dem Schema in `design.md` — inkl. `CHECK` über die **acht Nicht-Chat-Kategorien** (Decision 8: der Ausschluss steht im Schema, nicht in einem Kommentar) und `ON DELETE CASCADE` auf `users`.
- [x] 1.3 Indizes: `idx_user_events_user_created (user_id, created_at DESC)` für den Dashboard-Read, `idx_user_events_retention (seen_at, created_at)` für den Purge.
- [x] 1.4 `0NN_user_events.down.sql`: Tabelle droppen.
- [x] 1.5 Up/Down gegen eine **isolierte Kopie** verifizieren, nicht gegen die echte `teamwerk.db`. Bekannte Repo-Einschränkung: `make migrate-down` ist ein No-Op (`db.Migrate()` ruft nur `m.Up()`), das Down also direkt per `sqlite3 < …down.sql` prüfen.
- [x] 1.6 `internal/db/migrations_test.go` um die neuen Objekte ergänzen.

## 2. Foundation-Package `internal/eventlog`

- [x] 2.1 `internal/eventlog/eventlog.go`: `Record(db, userIDs []int, category, title, body, url string)` — ein `INSERT` mit Multi-Row-`VALUES` in **einer** Anweisung, nicht N Einzel-Inserts (ein Vereins-Fan-out sind bis zu ~180 Zeilen). Fehler nur loggen, nie zurückgeben: ein Log-Fehler darf keinen Versand killen.
- [x] 2.2 `ListForUser(ctx, db, userID int, limit int) ([]Event, error)` — absteigend nach `created_at`, dann `id` (stabile Sortierung bei gleicher Sekunde; SQLite-`CURRENT_TIMESTAMP` hat Sekundenauflösung).
- [x] 2.3 `MarkSeen(ctx, tx, ids []int)` — `UPDATE … SET seen_at = CURRENT_TIMESTAMP WHERE id IN (…) AND seen_at IS NULL`. Nimmt IDs, **nie** eine `user_id` (Decision 4).
- [x] 2.4 `Purge(db) (int64, error)` — der `DELETE` aus `design.md` Decision 5, beide Zweige in einer Anweisung.
- [x] 2.5 `internal/eventlog` in `internal/arch/arch_test.go` unter `foundation` eintragen (sonst schlägt der Klassifikations-Test fehl).
- [x] 2.6 Tests `internal/eventlog/eventlog_test.go`: Multi-Row-Insert, Sortierung, `MarkSeen` überschreibt nicht, `Purge`-Zweige (3 Tage gesehen / ungesehen bleibt / 90-Tage-Kappe).

## 3. Fassade: Optionen und Log-Fan-out

- [x] 3.1 `internal/notify/notify.go`: `type Option func(*options)` mit `NoEmail()` und `SkipPushPref()`; `Send` variadisch erweitern (`opts ...Option`), damit die 20 bestehenden Aufrufstellen unverändert bleiben.
- [x] 3.2 `eventlog.Record(...)` als erste Anweisung nach dem Leer-Guard einfügen — **vor** beiden Filtern (Decision 1). Der Guard `len(userIDs) == 0` bleibt davor: leere Liste schreibt nichts.
- [x] 3.3 `SkipPushPref` überspringt `push.FilterByPushPref`; `NoEmail` überspringt den gesamten Email-Zweig inkl. `filterByEmailPref`.
- [x] 3.4 Die acht Test-Doubles (`notify.Send = func(...)`) um `...Option` erweitern: `duties/delete_slot_cancellation_test.go`, `duties/notify_category_test.go`, `trainings/notify_category_test.go`, `trainings/cancellation_test.go`, `auth/notify_category_test.go`, `games/notify_category_test.go`, `games/cancellation_test.go`, `carpooling/notify_category_test.go`.
- [x] 3.5 Tests `internal/notify/notify_test.go`: Fan-out über die ungefilterte Liste, Log trotz `push_enabled=0`, Log trotz fehlender Subscription, `NoEmail`/`SkipPushPref`-Semantik, leere Liste schreibt nichts.

## 4. Die sechs Bypässe auf die Fassade zurückführen

- [x] 4.1 `internal/matchreports/notify.go`: `notifyReviewers` ruft `notify.Send(..., "operativ", ..., notify.NoEmail())`; eigenes `FilterByPushPref` entfällt.
- [x] 4.2 `internal/carpooling/paarungen_handler.go` ×3 (`RequestPairing`, `ConfirmPairing`, `RejectPairing`): je `notify.Send(..., "carpooling", ..., notify.NoEmail())`; eigenes `FilterByPushPref` entfällt.
- [x] 4.3 `internal/scheduler/scheduler.go` `sendVideoRetentionWarnings`: `notify.Send(..., "sonstiges", ..., notify.SkipPushPref())`. Der manuelle Email-Zweig (`FilterByEmailPref` + `SendEmail`) entfällt — die Fassade macht ihn. Der `notification_log`-Claim bleibt **vor** dem Aufruf.
- [x] 4.4 `internal/scheduler/scheduler.go` `sendDutyReminders`: Push-Zweig auf `notify.Send(..., "duty_reminders", ..., notify.NoEmail())`. Die strukturierte Reminder-Mail (`buildReminderMail` + `duty_reminder_log`) bleibt unverändert daneben. `notification_log`-Claim bleibt vor dem Aufruf.
- [x] 4.5 `internal/videos/worker.go`: `workerConfig.pushSend`/`emailSend` durch ein `notifySend(userIDs []int, category, title, body, url string)` ersetzen; die Vorfilterung in `runNotify` (`FilterByPushPref`/`FilterByEmailPref` auf `"sonstiges"`) entfällt, die Fassade übernimmt sie. `fakeConfig` im Test mitziehen.
- [x] 4.6 Bestehende Tests der fünf Stellen laufen lassen — insbesondere `internal/scheduler/push_bypass_test.go`, dessen `capturePush`-Seam weiterhin greift (`notify.Send` ruft `push.SendToUsers` intern auf).

## 5. Architektur-Test

- [x] 5.1 `internal/arch/pushfanout_test.go` (nur stdlib `go/parser`/`go/ast`, konsistent mit `arch_test.go`/`broadcast_test.go`): alle `internal/`-Packages parsen, jeden `push.SendToUsers`-Aufruf außerhalb `internal/notify` und `internal/push` melden.
- [x] 5.2 Allowlist mit Begründung analog `broadcastAllowlist`. Erwarteter Inhalt: nur `internal/chat` für `push.SendToUserWithBadge`.
- [x] 5.3 Anti-Verrottung: `TestArchitecture_PushAllowlistOhneWaisen` — ein Eintrag ohne realen Aufrufer lässt den Test fehlschlagen.

## 6. Dashboard-Endpunkt

- [x] 6.1 `internal/dashboard/handler.go`: Typ `Event` + Feld `Events []Event` in `Response`. Feldnamen im JSON camelCase wie die übrigen (`createdAt`, nicht `created_at`).
- [x] 6.2 `queryEvents(ctx, userID)`: `eventlog.ListForUser(..., 30)`, danach `eventlog.MarkSeen(ctx, tx, ids der gelieferten Zeilen)` in einer Transaktion (Decision 4). Fehler beim Stempeln loggen, aber den Response ausliefern — der Log ist nachrangig gegenüber der Anzeige.
- [x] 6.3 `Response.Events` mit `[]Event{}` initialisieren (nicht `nil`), damit das JSON `[]` statt `null` liefert — wie bei den bestehenden Feldern.
- [x] 6.4 Tests `internal/dashboard/handler_test.go`: Reihenfolge, Cap 30, Stempel nur auf gelieferten Zeilen, zweiter Abruf verschiebt nicht, fremde Events unsichtbar, 401 ohne Auth.

## 7. Retention-Job

- [x] 7.1 `internal/scheduler/eventlog_retention.go`: `func (s *Scheduler) purgeEventLog()` ruft `eventlog.Purge`, loggt die Anzahl bei `> 0`.
- [x] 7.2 In `Scheduler.Run()` einhängen (Minutentakt wie die übrigen Jobs; kein Idempotenzschutz nötig).
- [x] 7.3 Tests `internal/scheduler/eventlog_retention_test.go`: die vier Szenarien aus dem Spec (4 Tage gesehen → weg, 2 Tage gesehen → bleibt, 30 Tage ungesehen → bleibt, 91 Tage ungesehen → weg).

## 8. Frontend

- [x] 8.1 `web/src/pages/DashboardPage.tsx`: Typ `EventItem` + `events` in `DashboardData`.
- [x] 8.2 `GeschehenSection`-Component nach dem Muster der bestehenden Sections; `DashboardRow` wiederverwenden. Einträge mit leerer `url` sind nicht anklickbar.
- [x] 8.3 `Accordion id="ereignisse" title="Ereignisse"` einhängen — **nicht** „Benachrichtigungen" (Spec: Abgrenzung zur Section „Nachrichten"). Icon aus `lucide-react` (`Activity`), keine Unicode-Zeichen. In den `openSections`-Default aufnehmen.
- [x] 8.4 Relative Zeitangabe („vor 2 Std.") — prüfen, ob im Repo bereits ein Helfer existiert, bevor ein neuer entsteht. Gefunden: lokale `relativeTime` in `AdminUsersPage.tsx` — nach `web/src/lib/relativeTime.ts` extrahiert und in beiden Stellen wiederverwendet (statt Duplikat).
- [x] 8.5 Kein Eingriff in `useLiveUpdates` nötig, sofern die bestehende Bedingung (`games`/`trainings`/`duties`/`mitfahrgelegenheiten`/`absences`/`event-note`) unverändert bleibt — verifizieren, dass sie `load(true)` auslöst. Verifiziert, unverändert gelassen.
- [x] 8.6 Der App-Icon-Badge bleibt unberührt: keine Änderung an `AppShell.tsx`/`sw.ts`.
- [x] 8.7 Tests: Rendering mit/ohne Einträge, Leerzustand, Eintrag ohne `url` ist kein Link. Datei `web/src/pages/__tests__/DashboardPage.geschehen.test.tsx` (Verzeichnis/Namensschema der bereits existierenden `DashboardPage.nachrichten.test.tsx` gefolgt statt `DashboardPage.test.tsx`).
- [x] 8.8 Nur `brand-*`-Tokens, verbindliche Klassen-Strings aus `docs/agent/05-frontend.md`.

## 9. Absagegrund-Umkehr

- [x] 9.1 `internal/games/…`: `TestDeleteGame_GrundWirdNichtPersistiert` durch `TestDeleteGame_GrundStehtImEventLog` ersetzen. Der DB-Scan bleibt sinnvoll, aber mit `user_events` als **erwarteter** Fundstelle und allen anderen Tabellen als verbotenen.
- [x] 9.2 Prüfen, ob die Schwester-Tests bei Trainings/Serien/Diensten dieselbe Aussage treffen, und gleich mitziehen.
- [x] 9.3 Sicherstellen, dass `silent: true` weiterhin **gar keine** Benachrichtigung auslöst — und damit auch keine Log-Zeile.

## 10. Dokumentation

- [x] 10.1 `docs/agent/06-gotchas.md`, Absatz „Absage-Benachrichtigungen": Punkt (3) umschreiben — der Grund lebt jetzt für die Dauer der Event-Log-Retention in `user_events.body`, nicht mehr nur im Zustellkanal. Die ursprüngliche Begründung (kein `games.status='cancelled'`) bleibt gültig.
- [x] 10.2 `docs/agent/06-gotchas.md`: neuer Absatz „Event-Log" — Einfügepunkt vor den Filtern, eingefrorene Empfängermenge, Stempel nur auf gelieferten Zeilen, 90-Tage-Kappe, Chat bewusst außen vor.
- [x] 10.3 `docs/agent/08-verification.md`: das neue Push-Fan-out-Gate neben Broadcast-Gate und SEPA-XSD-Gate aufnehmen.
- [x] 10.4 `docs/agent/04-api-db.md`: `user_events` bei den Schema-Konventionen erwähnen, soweit nicht-ableitbar (kein Fremdschlüssel auf Domänen-Objekte, `category`-CHECK ohne `chat`).

## 11. Abschluss

- [x] 11.1 `/verify-change` — Build/Test/Lint + Projekt-Invarianten (Route→Tests, Mutation→`Broadcast`/`useLiveUpdates`, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`).
- [x] 11.2 `make metrics-gate` — der Change entfernt Duplikation (sechs Handbauten), die Schwellwerte sollten sich nicht verschlechtern.
- [ ] 11.3 Ein Commit pro Task-Gruppe, Conventional Commits mit Domänen-Scope.
