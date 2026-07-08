## 1. Migration 026 — `chat` in den CHECK

- [x] 1.1 `internal/db/migrations/026_notification_preferences_chat_category.up.sql`: `notification_preferences` per Rebuild (neue Tabelle mit CHECK inkl. `chat`, `INSERT … SELECT`, drop, rename) in einer Transaktion; erlaubte Menge `games, trainings, duties, duty_reminders, carpooling, membership, chat`
- [x] 1.2 `.down.sql`: Rebuild zurück auf CHECK **ohne** `chat` (vorhandene `chat`-Zeilen werden verworfen — im Kommentar dokumentieren)
- [x] 1.3 Lokale Dev-DB (war leer + dirty@v13) frisch neu aufgebaut → `migrate up` sauber bis v26. Roundtrip von 026 verifiziert: DOWN weist `chat` ab + verwirft chat-Zeilen + behält `games`; erneutes UP erlaubt `chat` wieder + erhält `games`. (`migrate down` gibt es nur als DDL-Direktcheck — die `migrate`-CLI führt für up/down beide Male Up aus.)

## 2. Backend — Präferenzen transaktional + Whitelist (Defekt 1)

- [x] 2.1 `internal/push/prefs.go`: eine Quelle der Wahrheit `ValidCategories` (Slice/Set) inkl. `chat`; `GetAllPreferences` daraus speisen
- [x] 2.2 `internal/notifications/handler.go` `UpdateNotificationPreferences`: in `sql.Tx` schreiben; unbekannte Kategorie → `Rollback` + HTTP 400; sonst `Commit` + 204
- [x] 2.3 Test: `PUT /api/profile/notification-preferences` mit `chat:{push:false}` → 204, Zeile `(user,'chat',0)` persistiert (Regression Defekt 1)
- [x] 2.4 Test: `PUT` mit unbekannter Kategorie → 400 und **keine** Teil-Persistenz (transaktional zurückgerollt)
- [x] 2.5 Test: `FilterByPushPref(uids,"chat")` mit gespeicherter `chat`-Zeile `push_enabled=0` schließt den Nutzer tatsächlich aus

## 3. Test-Fundament — Send-Seam + Fixtures (Voraussetzung, siehe D5)

- [x] 3.1 `internal/push/push.go`: Package-Var-Seam `var sendNotification = webpush.SendNotification`; `SendToUsers`/`SendToUserWithBadge` rufen `sendNotification(...)` statt der Funktion direkt (im Test überschreibbar)
- [x] 3.2 `internal/testutil/fixtures.go`: `CreatePushSubscription(t, db, userID)` (endpoint/p256dh/auth) und `CreateNotificationPreference(t, db, userID, category, push, email)`
- [x] 3.3 Geprüft: `TestConfig()` **bleibt unverändert** (globale Änderung würde jeden Push-Test beeinflussen). Die Delete-Switch-Tests bauen in `push_send_test.go` einen lokalen `testCfg()` mit Dummy-VAPID-Key und überschreiben `sendNotification` — self-contained (kein testutil-Import wegen Import-Zyklus `testutil→…→push`)

## 4. Backend — Abo-Löschung verengen (Defekt 2)

- [x] 4.1 `internal/push/push.go`: gemeinsame Hilfe `handlePushResponse(db, subID, statusCode)` — Delete nur bei `http.StatusGone`/`http.StatusNotFound`; bei 400/401 `slog.Warn` ohne Delete; in `SendToUsers` und `SendToUserWithBadge` statt der duplizierten Switches nutzen
- [x] 4.2 Test: simulierte 410/404 → Abo gelöscht (Seam aus 3.1 + Fixture aus 3.2)
- [x] 4.3 Test: simulierte 400/401 → Abo **bleibt erhalten** (Regression Defekt 2); simulierte 5xx → bleibt erhalten
- [x] 4.4 Bestehende Tests anpassen, die das alte 401/400-Delete-Verhalten kodieren — **keine gefunden**: der Send-Pfad war bislang völlig ungetestet (VAPID-Gate no-op), es gab kein Test, der das alte Verhalten kodierte

## 5. Frontend — Re-Subscribe beobachtbar (Defekt 2)

- [x] 5.1 `web/src/hooks/usePushSubscription.ts`: leeres `catch {}` durch `catch (err) { console.warn('[push] subscribe failed', err) }` ersetzen
- [x] 5.2 `web/src/components/profile/ProfileMiscTab.tsx` gegen 026 prüfen: Chat-Toggle speichert ohne Fehler und ist nach Reload persistent (manuell/Vitest, sofern Testinfra vorhanden)

## 6. Verifikation

- [ ] 6.1 `/verify-change`: Build/Test/Lint + Invarianten (Route→Tests, Migrationsnummer, `openspec validate`)
- [ ] 6.2 Ein Commit pro Task-Gruppe (Conventional Commits, Scopes `db`/`notifications`/`push`/`pwa`); abschließender Commit archiviert das Proposal
