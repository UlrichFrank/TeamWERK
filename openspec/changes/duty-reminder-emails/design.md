## Context

TeamWERK hat bereits einen minütlich laufenden Cron-Scheduler (`scheduler:run`) und einen funktionierenden SMTP-Mailer. Duty-Slots haben `event_date`, `slots_total`, `slots_filled` und `team_id`. Duty-Types tragen `target_role` (wer den Dienst leisten soll). Die Team-Zugehörigkeit von Usern ist je nach Rolle unterschiedlich aufgebaut (direkt via `team_trainers`, indirekt via `members`/`family_links`).

## Goals / Non-Goals

**Goals:**
- Automatische Erinnerungsmail 2 Tage vor Events mit offenen Slots
- Empfänger: rollenbasiert + teambasiert, nur User ohne eigenen Slot-Eintrag
- Aggregierung: eine Mail pro User pro Tag (nicht pro Slot)
- User kann Reminder im Profil deaktivieren

**Non-Goals:**
- Konfigurierbare Vorlaufzeit pro Duty-Type (nur global 2 Tage)
- Reminder für bereits eingetragene User ("vergiss deinen Dienst nicht")
- Push-Notifications oder In-App-Benachrichtigungen
- Mehrfach-Reminder (z.B. 7 Tage + 2 Tage)

## Decisions

### 1. Scheduler-Integration statt separatem Service

Der vorhandene `scheduler:run`-Cron (läuft jede Minute) bekommt einen neuen Job `sendDutyReminders()`. Kein neuer systemd-Timer, kein separates Binary.

**Warum:** Kein Overhead, keine zusätzliche Infra. Der Scheduler läuft bereits zuverlässig auf dem VPS.

**Alternativ betrachtet:** Täglicher separater Cron-Job — aber dann müsste `setup-vps.sh` angepasst werden und das Deployment wird komplexer.

### 2. Deduplizierung via `duty_reminder_log`-Tabelle

Neue Tabelle mit `PRIMARY KEY (user_id, event_date)`. Der Scheduler schreibt beim Mailversand einen Eintrag; beim nächsten Minutentick ist das INSERT ein No-Op (UNIQUE-Konflikt = übersprungen).

**Warum:** Einfachste idempotente Lösung ohne Lock-Mechanismus. Da der Scheduler single-process ist (kein paralleles Ausführen), gibt es keine Race-Conditions.

**Alternativ betrachtet:** `reminder_sent_at` direkt auf `duty_slots` — aber das deckt nur einen Slot ab, nicht den aggregierten User-Fall.

### 3. Empfängerbestimmung als eine SQL-Query pro Slot-Gruppe

Alle offenen Slots für `target_date` werden einmal geladen. Dann wird pro `(team_id, target_role)`-Kombination eine Query für eligible User ausgeführt. Die Ergebnisse werden im Speicher zu einer User→Slots-Map zusammengeführt.

**Empfänger-SQL-Logik je target_role:**
```sql
-- spieler: über members → team_memberships (aktive Saison)
SELECT DISTINCT u.id, u.email, u.name
FROM users u
JOIN members m ON m.user_id = u.id
JOIN team_memberships tm ON tm.member_id = m.id
JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
WHERE u.role = 'spieler'
  AND tm.team_id = ?          -- slot.team_id
  AND u.duty_reminder_days IS NOT NULL
  AND NOT EXISTS (
    SELECT 1 FROM duty_assignments da
    WHERE da.user_id = u.id AND da.duty_slot_id IN (/* slot IDs für diesen Tag */)
  )

-- elternteil: über family_links → members → team_memberships
SELECT DISTINCT u.id, u.email, u.name
FROM users u
JOIN family_links fl ON fl.parent_user_id = u.id
JOIN members m ON m.id = fl.member_id
JOIN team_memberships tm ON tm.member_id = m.id
JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
WHERE u.role = 'elternteil'
  AND tm.team_id = ?
  AND u.duty_reminder_days IS NOT NULL
  AND NOT EXISTS (...)

-- trainer: über team_trainers
SELECT DISTINCT u.id, u.email, u.name
FROM users u
JOIN team_trainers tt ON tt.user_id = u.id
WHERE u.role = 'trainer'
  AND tt.team_id = ?
  AND u.duty_reminder_days IS NOT NULL
  AND NOT EXISTS (...)
```

Wenn `duty_slot.team_id IS NULL`: kein Team-Filter, nur Rollen-Match.

### 4. Mail-Template als Plain-Text

Kein HTML-Mail, kein Templating-Framework. Plain-Text mit `fmt.Sprintf` — konsistent mit dem restlichen Mailer.

**Warum:** Der Mailer unterstützt bereits Plain-Text. Ein HTML-Template würde eine neue Abhängigkeit (html/template) und Wartungsaufwand bedeuten.

### 5. Preference als `INT NULL` auf `users`

`duty_reminder_days INT NULL DEFAULT NULL` — `NULL` = deaktiviert (Standard), `2` = 2 Tage vorher. Kein neues Enum-Feld, kein separates Settings-Objekt.

**Warum:** Einfach, keine JOIN-Kosten, leicht erweiterbar auf andere Werte (z.B. `1`, `3`). Default NULL = opt-in statt opt-out.

## Risks / Trade-offs

- **Kein aktiver Reminder bei leerem `duty_reminder_log`-Eintrag nach Event-Verschiebung** → Wenn ein Event auf einen anderen Tag verschoben wird, ist der Log-Eintrag für den alten Tag nicht mehr relevant — der neue Termin D-2 erzeugt einen neuen Eintrag. Kein Problem.
- **Scheduler-Ausfälle** → Wenn der Scheduler an D-2 nicht läuft (VPS-Neustart etc.), gibt es keinen Nachholmechanismus. Akzeptiert — "best effort" ist für diese Funktion ausreichend.
- **Viele offene Slots → viele Mails beim ersten Rollout** → Beim ersten Deploy bekommen alle berechtigten User für alle Events in den nächsten 2 Tagen eine Mail. Akzeptiert; einmalig und korrekt.

## Migration Plan

1. Migration 003 deployen (ALTER TABLE users, CREATE TABLE duty_reminder_log)
2. Binary deployen (beinhaltet Migrations automatisch via `make deploy`)
3. Kein Rollback-Risiko: neue Spalte ist nullable, neue Tabelle ist additiv
