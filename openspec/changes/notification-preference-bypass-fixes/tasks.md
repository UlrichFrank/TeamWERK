## 1. Schema + Whitelist

- [ ] 1.1 Migration `027_notification_preferences_operativ_sonstiges.{up,down}.sql`: Rebuild, CHECK um `operativ`,`sonstiges` erweitern; Down entfernt beide (verwirft solche Zeilen)
- [ ] 1.2 `push.ValidCategories` um `operativ`,`sonstiges` erweitern
- [ ] 1.3 `migrate up` lokal + Roundtrip prüfen

## 2. Bypass beheben (Präferenz respektieren)

- [ ] 2.1 `carpooling/paarungen_handler.go` RequestPairing: `FilterByPushPref(…, "carpooling")` vor dem Push
- [ ] 2.2 `videos/worker.go` notifyReady: `FilterByPushPref(…, "sonstiges")` vor pushSend (leer ⇒ nichts senden)
- [ ] 2.3 `scheduler/attendance_reminders.go`: Trainer via `FilterByPushPref(…, "operativ")` vor notification_log aussortieren
- [ ] 2.4 `scheduler/scheduler.go` match-report-review-reminder: Reviewer via `FilterByPushPref(…, "operativ")` aussortieren
- [ ] 2.5 `matchreports/notify.go` notifyReviewers: `ids` via `FilterByPushPref(…, "operativ")` filtern
- [ ] 2.6 #3 video-retention **unverändert** lassen (Kommentar: bewusst harter Bypass)

## 3. Frontend

- [ ] 3.1 `ProfileMiscTab.tsx`: Kategorien `operativ`,`sonstiges` + Labels + Kurzbeschreibungen; Push-only-Darstellung (kein wirkungsloser E-Mail-Toggle)

## 4. Tests drehen + Positiv-Fall

- [ ] 4.1 `scheduler/push_bypass_test.go`: attendance + match-report-review von „Bypass" auf „respektiert operativ" (Opt-out ⇒ kein Push) drehen; Positiv-Fall Default ⇒ Push; video-retention-Test als Bypass BEHALTEN
- [ ] 4.2 `matchreports/push_bypass_test.go`: auf „respektiert operativ" drehen
- [ ] 4.3 `videos/push_bypass_test.go`: auf „respektiert sonstiges" drehen (+ Positiv-Fall)
- [ ] 4.4 `carpooling/push_bypass_test.go`: auf „respektiert carpooling" drehen (+ Positiv-Fall)

## 5. Verifikation

- [ ] 5.1 `go test ./...` + `go vet` grün; Frontend `lint`/`tsc`
- [ ] 5.2 `openspec validate`
- [ ] 5.3 Commit pro Task-Gruppe
