## Why

Beim Löschen eines Spiels oder generischen Ereignisses (`DELETE /api/kalender/{id}`) werden die verknüpften Dienste seit Migration 027 zwar per FK-Cascade entfernt — aber:

1. **Die Zugewiesenen erfahren nichts gezielt.** Heute fliegt nur eine pauschale Push „Spiel abgesagt" an das gesamte Team + Eltern (Kategorie `games`). Ein Elternteil, der einen Tagesdienst übernommen hatte, erfährt es nur, falls er Spiel-Pushes überhaupt abonniert hat — und der Text spricht ihn nicht persönlich an.
2. **`fulfilled`-Dienste verschwinden, die Konto-Buchung bleibt.** Wurde ein Dienst bereits abgehakt (`duty_assignments.status = 'fulfilled'`), zählen die Stunden in `duty_accounts.ist`. Beim Cascade-Delete verschwindet der Dienst, der Ist-Wert nicht — das Konto wird falsch.
3. **Email-Versand ist nirgends an die Nutzer-Präferenz gekoppelt.** Im Profil-Tab „Sonstiges" können Nutzer pro Kategorie (Spiele, Trainings, Dienste, Fahrgemeinschaften) Email einschalten. Der Schalter wird gespeichert, aber kein einziger Code-Pfad liest ihn — Email wird heute nur über das separate `duty_reminders`-Setting verschickt.

Die saubere Lösung für (3) ist eine Notification-Fassade, die anhand der Kategorie automatisch Push **und/oder** Email auslöst — dann muss kein Handler je wieder explizit `mailer.Send` aufrufen.

## What Changes

- **NEU: `internal/notifications/Send`-Fassade** — eine zentrale Funktion `notifications.Send(db, cfg, uids, category, title, body, url)`, die intern in Push-Cohorte (`push_enabled=1`) und Email-Cohorte (`email_enabled=1`) aufteilt und beides versendet. Verhalten ist neutral, solange keine Nutzer Email anhaken (Default email=false).
- **Bestehende Notify-Aufrufe migrieren** auf die neue Fassade: `games.DeleteGame`, `games.CreateGame`, `games.UpdateGame`, `trainings.DeleteSeries`, `trainings.DeleteSession`, `trainings.CreateSession`, `duties.DeleteSlot`, `duties.CreateSlot`, `auth.RequestMembership`, `carpooling`-Mutationen, `scheduler`-Jobs. Keine Verhaltensänderung außer: wer Email für eine Kategorie eingeschaltet hat, kriegt sie ab sofort auch.
- **Gezielte Notification an Dienst-Zugewiesene** beim Löschen eines Events: Vor dem Cascade-Delete werden die `duty_assignments.user_id` der betroffenen Slots gesammelt und mit `notifications.Send(..., "duties", "Dienst entfällt", "Dein Dienst zum <Eventname> am <Datum> wurde gelöscht", "/dienste")` benachrichtigt. Disjunkt zur bestehenden „Spiel abgesagt"-Push, die an Spiel-Responder geht.
- **`fulfilled`-Stunden rückbuchen** beim Cascade-Delete: für jedes betroffene `(user, season)`-Paar wird `duty_accounts.ist` neu aus den verbliebenen `fulfilled`-Assignments berechnet. Pending-Assignments wirken sich nicht aufs Konto aus und brauchen keine Rückbuchung.
- **`?delete_slots`-Query entfernen** aus `DeleteGame` — der Parameter ist seit Migration 027 wirkungslos (FK-Cascade greift immer). Frontend-Aufrufe in `GameEditModal.tsx` und `SpieltagDetailPage.tsx` werden auf parameterlose DELETEs umgestellt.

## Capabilities

### New Capabilities

- `notifications` — Kategoriegebundene Push- und Email-Verteilung über eine einzige Fassade.

### Modified Capabilities

- `push-duties` — Neuer Trigger: Event-Löschung benachrichtigt Dienst-Zugewiesene, inkl. Email-Pfad bei aktiviertem `email_enabled` für Kategorie `duties`.
- `game-deletion-cascade` — Klarstellung: `?delete_slots`-Query ist entfallen, Dienst-Konten werden bei `fulfilled`-Rückbuchung konsistent gehalten.

## Impact

- `internal/notifications/notifications.go` (NEU, ~80 Zeilen): Fassade + Email-Versand
- `internal/games/handler.go` (`DeleteGame`, ~30 Zeilen): Assignees vorab fetchen, Rollback-Logik, Notify-Call
- `internal/games/handler.go`, `internal/trainings/handler.go`, `internal/duties/handler.go`, `internal/auth/handler.go`, `internal/carpooling/handler.go`, `internal/scheduler/scheduler.go` — bestehende `FilterByPushPref` + `SendToUsers`-Paare durch `notifications.Send` ersetzen (~1 Zeile pro Stelle)
- `web/src/components/GameEditModal.tsx`, `web/src/pages/SpieltagDetailPage.tsx`: `?delete_slots=true` aus dem URL entfernen
- **Keine Migration nötig** — Schema bleibt unverändert
- **Trainings explizit ausgenommen**: `duty_slots` hat keinen `training_session_id`-Bezug; Training-Löschung verändert keine Dienste.
