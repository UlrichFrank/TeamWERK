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
- [ ] 6.1b **games / trainings / duties / carpooling** — **zurückgestellt**: benötigen schwere Domänen-Fixtures (aktive Saison, Team, Kader, Slots) + Goroutine-Sync. Muster steht (notify.Send-Seam); pro Domäne ein analoger Test. Bewusst separater Folge-Schritt (Grenznutzen-Cliff).

## 7. Preference-Bypass festnageln

- [ ] 7.1 **zurückgestellt**: Die 6 rohen `push.SendToUsers`-Sites lassen sich nur durch Beobachtung des tatsächlichen Sendens pinnen; das braucht pro Site einen Seam (der `push.sendNotification`-Seam ist unexportiert und aus den Domänen-Testpaketen nicht erreichbar) **oder** einen exportierten Seam je Aufrufer. Zudem ist „bewusst vs. Bug" eine offene **Design-Frage** — erst entscheiden, dann pinnen. Kein stiller Verzicht: hier dokumentiert.

## 8. Verifikation

- [ ] 8.1 `make test` grün; `make coverage` als Indikator prüfen (kein Gate)
- [ ] 8.2 `/verify-change`: Build/Test/Lint + `openspec validate`
- [ ] 8.3 Ein Commit pro Task-Gruppe (Conventional Commits, Scope `test`/jeweiliges Domänen-Package); abschließender Commit archiviert das Proposal
