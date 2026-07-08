## 1. Voraussetzung

- [ ] 1.1 Sicherstellen, dass das Fundament aus `push-notification-reliability` vorhanden ist (`push.sendNotification`-Seam, `testutil.CreatePushSubscription`, `testutil.CreateNotificationPreference`); sonst dort zuerst umsetzen

## 2. Präferenz-Logik (internal/push)

- [ ] 2.1 `internal/push/prefs_test.go`: `FilterByPushPref` — Default (kein Row ⇒ enthalten), `push_enabled=0` ⇒ ausgefiltert, Kategorie-Isolation, leere Eingabe
- [ ] 2.2 `internal/push/prefs_test.go`: `HasEmailEnabled` — true/false/kein Row (Default false), Kategorie-Trennung
- [ ] 2.3 `internal/push/prefs_test.go`: `GetAllPreferences` — alle Kategorien inkl. `chat` mit Defaults, Override durch DB-Rows

## 3. notify-Fassade (internal/notify)

- [ ] 3.1 Fake-Mailer-Seam in `testutil` (Capture ohne echten SMTP), analog `chat.pushFn`/`videos`-Muster
- [ ] 3.2 `internal/notify/notify_test.go`: `Send` — getrennte Push-/Email-Zweige (ein Nutzer nur Push, einer nur Email), leere Liste ⇒ kein Versand
- [ ] 3.3 `internal/notify/notify_test.go`: `sendCategoryEmail` — Direktlink-Zeile mit `BaseURL+url`, fehlende E-Mail wird stillschweigend übersprungen

## 4. Abo-/Präferenz-Endpoints (internal/notifications)

- [ ] 4.1 `handler_test.go`: `POST /api/push/subscribe` — 204 + Row angelegt; fehlendes Pflichtfeld ⇒ 400; Upsert per Endpoint (zweiter Call gleiches Endpoint ⇒ Update, kein Duplikat)
- [ ] 4.2 `handler_test.go`: `DELETE /api/push/subscribe` — Cross-User-Schutz (B löscht A nicht); 204
- [ ] 4.3 `handler_test.go`: `GET /api/profile/notification-preferences` — Defaults (alle Kategorien, push=true/email=false); 401 ohne Token

## 5. Chat-Breite (internal/chat)

- [ ] 5.1 Test: Gruppenkonversation mit 3 aktiven Mitgliedern ⇒ Push-Seam für beide Nicht-Sender aufgerufen (Sender ausgeschlossen), Badge je Empfänger korrekt
- [ ] 5.2 Test: Broadcast ⇒ Push-Seam für Nicht-Sender-Empfänger aufgerufen

## 6. Kategorie-Korrektheit je Trigger

- [ ] 6.1 Je Domäne (games, trainings, duties, carpooling, membership) ein Test, dass der auslösende Handler `notify.Send` mit der erwarteten Kategorie aufruft (bestehende Domänen-Tests ergänzen oder je einen neuen Test)

## 7. Preference-Bypass festnageln

- [ ] 7.1 Für die 6 rohen `push.SendToUsers`-Sites (match-report-reminder, attendance-reminder, video-retention, video-ready, carpool-pairing-request, match-report-submitted) je einen Pinning-Test: Empfänger mit `push_enabled=0` erhält Push dennoch; Kommentar markiert dies als offene Design-Frage (nicht in diesem Change entschieden)

## 8. Verifikation

- [ ] 8.1 `make test` grün; `make coverage` als Indikator prüfen (kein Gate)
- [ ] 8.2 `/verify-change`: Build/Test/Lint + `openspec validate`
- [ ] 8.3 Ein Commit pro Task-Gruppe (Conventional Commits, Scope `test`/jeweiliges Domänen-Package); abschließender Commit archiviert das Proposal
