## Context

`internal/notifications/SendToUsers(db, cfg, userIDs []int, title, body, url string)` ist die einheitliche Utility-Funktion für alle Push-Sends. Sie läuft als fire-and-forget Goroutine und bereinigt ungültige Subscriptions (HTTP 410) automatisch. Der bestehende Scheduler (`/usr/local/bin/teamwerk scheduler:run`, läuft minütlich via Cron) kann um Job-Typen erweitert werden.

## Goals / Non-Goals

**Goals:**
- Alle im Proposal genannten Trigger implementieren
- Keine Push-Sends blockieren den HTTP-Response-Weg
- Scheduled Jobs idempotent: doppeltes Ausführen in derselben Minute schickt keine doppelten Notifications

**Non-Goals:**
- Nutzer-seitige Notification-Präferenzen (opt-out pro Kanal) — späteres Feature
- E-Mail-Fallback für User ohne Push-Subscription — besteht bereits separat (SMTP)
- Notifications für RSVP-Änderungen (Spieler sagt Training ab) — noch offen

## Decisions

### 1. Wer erhält Training-Notifications?

**Entscheidung:** Alle User, die via `team_memberships` oder `users.team_id` dem betroffenen Team zugeordnet sind, plus der Trainer des Teams.

**Begründung:** `training_sessions` ist an eine `team_id` gebunden. Über einen JOIN `team_memberships → members → users` plus `users WHERE team_id = ?` lassen sich alle betroffenen User-IDs in einer Query ermitteln.

### 2. Admin-Notifications: welche Admins?

**Entscheidung:** `SELECT id FROM users WHERE role IN ('admin')` — alle Admins erhalten Mitgliedsanfragen und Datenänderungs-Requests. Trainer erhalten Datenänderungs-Requests ihres Teams zusätzlich.

### 3. Idempotenz der Scheduled Reminders

**Entscheidung:** Die Scheduler-Jobs prüfen ob bereits eine Notification für dieselbe (user_id, slot_id/session_id, datum) gesendet wurde — via einer neuen Tabelle `notification_log (id, user_id, ref_type, ref_id, sent_at)` mit `UNIQUE(user_id, ref_type, ref_id)`.

**Begründung:** Der Cron läuft minütlich. Ohne Idempotenz-Check würde jede Minute eine Reminder-Notification gesendet, solange der Bedingung erfüllt ist.

### 4. Uhrzeit der Duty-Reminder

**Entscheidung:** Der Scheduler prüft: `date(event_date) = date('now', '+1 day') AND strftime('%H', 'now') = '18'`. Nur um 18 Uhr wird gesendet.

**Begründung:** Hält die Logik einfach, kein separater Cron-Zeitplan nötig. Der minütliche Cron schickt nur in der 18:xx-Stunde — idempotent durch `notification_log`.

## Risks / Trade-offs

- **Notification-Log-Tabelle** ist ein neues Schema-Element — Migration erforderlich (011)
- **Team-Mitglieder-Query** bei großen Teams könnte leicht wachsen — bei max. ~30 Spielern pro Team vernachlässigbar
- Trainer-/Admin-Zuordnung für Datenänderungs-Notifications erfordert JOIN auf `team_id` — einfach mit bestehenden Strukturen
