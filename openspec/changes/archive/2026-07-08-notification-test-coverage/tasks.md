## 1. Voraussetzung

- [x] 1.1 Sicherstellen, dass das Fundament aus `push-notification-reliability` vorhanden ist (`push.sendNotification`-Seam, `testutil.CreatePushSubscription`, `testutil.CreateNotificationPreference`); sonst dort zuerst umsetzen

## 2. Präferenz-Logik (internal/push)

- [x] 2.1 `internal/push/prefs_test.go`: `FilterByPushPref` — Default (kein Row ⇒ enthalten), `push_enabled=0` ⇒ ausgefiltert, Kategorie-Isolation, leere Eingabe
- [x] 2.2 `internal/push/prefs_test.go`: `HasEmailEnabled` — true/false/kein Row (Default false), Kategorie-Trennung
- [x] 2.3 `internal/push/prefs_test.go`: `GetAllPreferences` — alle Kategorien inkl. `chat` mit Defaults, Override durch DB-Rows

## 3. notify-Fassade (internal/notify)

- [x] 3.1 Mail-Seam als `notify.sendMail`-Package-Var (statt in `testutil`) — der Email-Versand sitzt in `notify`, dort ist der Seam am wirkungsvollsten; Capture ohne echten SMTP
- [x] 3.2 `internal/notify/notify_test.go`: `Send` — getrennte Push-/Email-Zweige (ein Nutzer nur Push, einer nur Email), leere Liste ⇒ kein Versand
- [x] 3.3 `internal/notify/notify_test.go`: `sendCategoryEmail` — Direktlink-Zeile mit `BaseURL+url`, fehlende E-Mail wird stillschweigend übersprungen

## 4. Abo-/Präferenz-Endpoints (internal/notifications)

- [x] 4.1 `handler_test.go`: `POST /api/push/subscribe` — 204 + Row angelegt; fehlendes Pflichtfeld ⇒ 400; Upsert per Endpoint (zweiter Call gleiches Endpoint ⇒ Update, kein Duplikat)
- [x] 4.2 `handler_test.go`: `DELETE /api/push/subscribe` — Cross-User-Schutz (B löscht A nicht); 204
- [x] 4.3 `handler_test.go`: `GET /api/profile/notification-preferences` — Defaults (alle Kategorien, push=true/email=false); 401 ohne Token

## 5. Chat-Breite (internal/chat)

- [x] 5.1 Test: Gruppenkonversation mit 3 aktiven Mitgliedern ⇒ Push-Seam für beide Nicht-Sender aufgerufen (Sender ausgeschlossen), Badge je Empfänger korrekt
- [x] 5.2 Test: Broadcast ⇒ Push-Seam für Nicht-Sender-Empfänger aufgerufen

## 6. Kategorie-Korrektheit je Trigger

- [x] 6.1a **membership** (repräsentativ): `notify.Send`-Seam eingeführt; `RequestMembership` benachrichtigt Admins mit Kategorie `membership` (`internal/auth/notify_category_test.go`)
- [x] 6.1b **games / trainings / duties / carpooling**: je ein Test via `notify.Send`-Seam + `prodserver` — CreateGame→`games`, UpdateSession→`trainings`, CreateSlot→`duties`, Upsert→`carpooling` (`internal/{games,trainings,duties,carpooling}/notify_category_test.go`)

## 7. Preference-Bypass festnageln

- [x] 7.1 `push.SendToUsers` als überschreibbarer Package-Var-Seam; Pinning-Tests für alle **6** Sites, dass ein Empfänger mit `push_enabled=0` die Push dennoch erhält: attendance-/match-report-review-/video-retention-reminder (`internal/scheduler/push_bypass_test.go`), match-report-submitted (`internal/matchreports/push_bypass_test.go`), video-ready (`internal/videos/push_bypass_test.go`), carpool-pairing-request (`internal/carpooling/push_bypass_test.go`). Jeder Test markiert „bewusst vs. Bug" als **offene Design-Frage** (nicht hier entschieden).

## 8. Verifikation

- [x] 8.1 `go test ./...` grün (1358 Tests, 45 Pakete); `go vet ./...` sauber
- [x] 8.2 Build/Test/Lint + `openspec validate` manuell gefahren (grün)
- [x] 8.3 Ein Commit pro Task-Gruppe (Conventional Commits); Archivierung des Proposals separat/auf Wunsch
