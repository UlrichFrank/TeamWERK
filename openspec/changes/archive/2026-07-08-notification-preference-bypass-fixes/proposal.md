## Why

Sechs Call-Sites versenden Push über `push.SendToUsers` **ohne** `push.FilterByPushPref` (in `notification-test-coverage` als offene Design-Frage festgenagelt). Die Durchsicht ergab: einige Bypässe sind gewollt, andere nicht. Diese Änderung setzt die getroffenen Entscheidungen um — Nutzer bekommen wieder Kontrolle, wo sie sie haben sollten, kritische Alarme bleiben unabschaltbar.

## What Changes

- **Zwei neue Präferenz-Kategorien** in `notification_preferences` (Migration 027, CHECK-Erweiterung) und in `push.ValidCategories`:
  - `operativ` — Vereins-/Funktionärs-Erinnerungen (Default an).
  - `sonstiges` — „Sonstige Events" (Default an).
- **Bypass behoben / Präferenz respektiert:**
  - **#5 carpool-pairing-request** (`carpooling/paarungen_handler.go`): filtert jetzt `FilterByPushPref(…, "carpooling")` — konsistent mit `ConfirmPairing`/`RejectPairing` (war Bug).
  - **#4 video-ready** (`videos/worker.go` `notifyReady`): filtert `FilterByPushPref(…, "sonstiges")`.
  - **#1 match-report-review-reminder**, **#2 attendance-reminder**, **#6 match-report-submitted**: filtern `FilterByPushPref(…, "operativ")`.
- **Bewusst unverändert (harter Bypass):**
  - **#3 video-retention-warning** — Datenverlust-Warnung (Video wird gelöscht); muss zuverlässig zustellen, keine Kategorie.
- **Profil-UI** (`ProfileMiscTab`): zwei neue Toggle-Zeilen (`operativ`, `sonstiges`) inkl. Kurzbeschreibung.
- **Tests:** die Pinning-Tests der 5 gefixten Sites werden von „sendet trotz Opt-out" auf „respektiert Opt-out" gedreht (+ Positiv-Fall Default→sendet); der #3-Bypass-Test bleibt.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `notification-preferences`: zwei neue Kategorien (`operativ`, `sonstiges`); fünf bislang präferenz-ignorierende Push-Trigger respektieren nun die jeweilige Kategorie; die Datenverlust-Warnung (`video-retention`) bleibt bewusst nicht abschaltbar.

## Impact

- **Migration:** `internal/db/migrations/027_notification_preferences_operativ_sonstiges.{up,down}.sql` (Rebuild, CHECK + `operativ`,`sonstiges`).
- **Backend:** `internal/push/prefs.go` (ValidCategories); `internal/carpooling/paarungen_handler.go`, `internal/videos/worker.go`, `internal/scheduler/scheduler.go`, `internal/scheduler/attendance_reminders.go`, `internal/matchreports/notify.go` (je ein `FilterByPushPref`).
- **Frontend:** `web/src/components/profile/ProfileMiscTab.tsx` (2 Kategorien + Beschreibungen).
- **Tests:** Pinning-Tests in `scheduler`/`matchreports`/`videos`/`carpooling` angepasst (5 gedreht, #3 bleibt).
- **Kein** neuer externer Dienst. Bestehende Präferenz-Zeilen bleiben erhalten (Rebuild kopiert 1:1).
