## Context

Push-Notification-Infrastruktur (VAPID, `push_subscriptions`, `SendToUsers`) ist bereits vollständig implementiert und für Chat genutzt. Die `sendDutyReminders`-E-Mail-Funktion im Scheduler läuft ebenfalls bereits via `duty_reminder_days`-Spalte (null = kein Reminder, NOT NULL = Reminder aktiv).

Ziel: Push auf alle relevanten Domains ausrollen + nutzer-konfigurierbare Präferenzen (Push/E-Mail pro Kategorie) im Profil.

## Goals / Non-Goals

**Goals:**
- Push Notifications bei game-, training-, duty-, carpooling- und membership-Ereignissen
- Scheduled Push-Reminder für Spiele, Trainings, Dienste, Fahrgemeinschaften
- `notification_preferences`-Tabelle als einheitliches Opt-in/out für Push und E-Mail
- Profil-UI: Toggles pro Kategorie unter „Sonstiges"
- Scheduler nutzt Präferenzen statt `duty_reminder_days`

**Non-Goals:**
- `duty_reminder_days`-Spalte wird nicht entfernt (SQLite-Altlast, migriert aber nicht aktiv genutzt)
- Keine In-App Notification-Inbox
- Keine Notification-Gruppen oder zeitverzögertes Batching

## Decisions

### 1. Präferenz-Tabelle statt Spalten auf `users`

**Entscheidung:** Neue Tabelle `notification_preferences (user_id, category, push_enabled, email_enabled)`.

**Warum:** Skaliert auf beliebige Kategorien ohne Schema-Migration für jede neue Kategorie. Abwesenheit einer Zeile = Default (push on, email off).

**Alternative verworfen:** `users`-Spalten (`push_games`, `push_trainings`, …) — jede neue Kategorie braucht ALTER TABLE.

### 2. Default: Push an, E-Mail aus

**Entscheidung:** Fehlt eine Zeile in `notification_preferences`, gilt: `push_enabled=1, email_enabled=0`.

**Warum:** Nutzer, die nie die Einstellungen aufrufen, bekommen Push-Benachrichtigungen (sinnvolle Default-Erfahrung). E-Mail ist explizites Opt-in um Spam zu vermeiden.

**Migration:** Bestehende Nutzer mit `duty_reminder_days IS NOT NULL` erhalten beim nächsten Profil-Aufruf keine automatische Migration — sie müssen E-Mail selbst aktivieren. Die alten `duty_reminder_days`-Reminders werden durch Scheduler-Check auf `notification_preferences` ersetzt.

### 3. Preference-Check vor SendToUsers

**Entscheidung:** Neue Hilfsfunktion `notifications.FilterByPushPref(db, userIDs, category) []int` filtert die User-ID-Liste vor jedem `SendToUsers`-Aufruf.

**Warum:** Kapselt die DB-Logik zentral. Handler bleiben schlank. Der Aufruf bleibt `go notifications.SendToUsers(...)`.

### 4. E-Mail-Reminder nur für duty_reminders-Kategorie

**Entscheidung:** Nur die Kategorie `duty_reminders` hat eine E-Mail-Option. Alle anderen Kategorien haben nur Push.

**Warum:** E-Mail bei Spiel-/Trainings-Absagen würde zu Spam führen. Push ist ausreichend reaktionsfähig für Event-Notifications.

### 5. notification_log für Scheduled Reminders

**Entscheidung:** Neue Tabelle `notification_log (user_id, ref_type, ref_id, sent_at)` für Idempotenz aller Scheduled Reminder (Spiele, Trainings, Dienste, Fahrgemeinschaften).

**Warum:** Bestehende `duty_reminder_log`-Tabelle ist nur für E-Mail-Dienst-Reminders. Eine generische Tabelle deckt alle Reminder-Typen ab und vermeidet Duplikate bei mehrfachem Scheduler-Lauf.

## Risks / Trade-offs

- **[Risk] Bestehende E-Mail-Reminder fallen weg für User mit `duty_reminder_days IS NOT NULL`** → Mitigation: In der UI einen Hinweis zeigen, dass die neue Einstellung die alte ersetzt; `duty_reminder_days`-Spalte bleibt für Kompatibilität, wird aber vom Scheduler nicht mehr gelesen.
- **[Risk] Push-Spam bei vielen Events** → Mitigation: Handler-Logik aggregiert nicht; jedes Event triggert maximal eine Notification. Nutzer können per Kategorie deaktivieren.
- **[Risk] Race Condition bei Scheduler-Idempotenz (notification_log)** → SQLite serialisiert Writes nativ; `INSERT OR IGNORE` sichert Idempotenz.

## Migration Plan

1. Migration `028_notification_preferences.up.sql` — `notification_preferences` + `notification_log`
2. Deployment via `make deploy` (führt `migrate up` automatisch aus)
3. Rollback: `028_notification_preferences.down.sql` droppt beide Tabellen

## Open Questions

- Sollen Spiel-/Trainings-Reminder auch E-Mail bekommen? → Aktuell nein (nur Push-Reminder).
- Scheduler-Interval für Spiel-/Trainings-Reminder: 24h vorher — soll der Scheduler täglich früh morgens laufen? → Bestehender Cronjob `* * * * *` reicht, Scheduler prüft selbst den Zeitfenster.
