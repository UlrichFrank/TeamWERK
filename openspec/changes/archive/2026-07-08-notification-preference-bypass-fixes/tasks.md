## 1. Schema + Whitelist

- [x] 1.1 Migration `027_notification_preferences_operativ_sonstiges.{up,down}.sql`: Rebuild, CHECK um `operativ`,`sonstiges` erweitern; Down entfernt beide (verwirft solche Zeilen)
- [x] 1.2 `push.ValidCategories` um `operativ`,`sonstiges` erweitern
- [x] 1.3 `migrate up` lokal + Roundtrip prüfen

## 2. Bypass beheben (Präferenz respektieren)

- [x] 2.1 `carpooling/paarungen_handler.go` RequestPairing: `FilterByPushPref(…, "carpooling")` vor dem Push
- [x] 2.2 `videos/worker.go` notifyReady: `FilterByPushPref(…, "sonstiges")` vor pushSend (leer ⇒ nichts senden)
- [x] 2.3 `scheduler/attendance_reminders.go`: Trainer via `FilterByPushPref(…, "operativ")` vor notification_log aussortieren
- [x] 2.4 `scheduler/scheduler.go` match-report-review-reminder: Reviewer via `FilterByPushPref(…, "operativ")` aussortieren
- [x] 2.5 `matchreports/notify.go` notifyReviewers: `ids` via `FilterByPushPref(…, "operativ")` filtern
- [x] 2.6 #3 video-retention **unverändert** lassen (Kommentar: bewusst harter Bypass)

## 3. Frontend

- [x] 3.1 `ProfileMiscTab.tsx`: Kategorien `operativ`,`sonstiges` + Labels + Kurzbeschreibungen; Push-only-Darstellung (kein wirkungsloser E-Mail-Toggle)

## 4. Tests drehen + Positiv-Fall

- [x] 4.1 `scheduler/push_bypass_test.go`: attendance + match-report-review von „Bypass" auf „respektiert operativ" (Opt-out ⇒ kein Push) drehen; Positiv-Fall Default ⇒ Push; video-retention-Test als Bypass BEHALTEN
- [x] 4.2 `matchreports/push_bypass_test.go`: auf „respektiert operativ" drehen
- [x] 4.3 `videos/push_bypass_test.go`: auf „respektiert sonstiges" drehen (+ Positiv-Fall)
- [x] 4.4 `carpooling/push_bypass_test.go`: auf „respektiert carpooling" drehen (+ Positiv-Fall)

## 5. Verifikation

- [x] 5.1 `go test ./...` + `go vet` grün; Frontend `lint`/`tsc`
- [x] 5.2 `openspec validate`
- [x] 5.3 Commit pro Task-Gruppe
