## Why

Offene Duty-Slots werden häufig erst kurz vor dem Event oder gar nicht belegt, weil User keine aktive Benachrichtigung erhalten. Eine automatische Erinnerungsmail 2 Tage vor dem Event gibt Berechtigten die Chance, sich noch rechtzeitig einzutragen.

## What Changes

- Neuer User-Preference-Schalter: "Erinnerungsmail 2 Tage vor Event" oder "Nie"
- Neuer Scheduler-Job: aggregierte Erinnerungsmail pro User mit allen offenen Slots am Zieltag
- Empfängerlogik: rollenbasiert (duty_type.target_role) + teambasiert (duty_slot.team_id), nur User die noch nicht eingetragen sind
- Deduplizierung via Log-Tabelle (keine Doppelmails trotz Minuten-Cron)

## Capabilities

### New Capabilities

- `duty-reminder-emails`: Automatische Erinnerungsmails für offene Duty-Slots, aggregiert pro User und Zieltag, mit rollenbasierter + teambasierter Empfängerbestimmung
- `user-reminder-preference`: User-seitige Einstellung im Profil zur Steuerung des Reminder-Verhaltens (2 Tage vorher / nie)

### Modified Capabilities

(keine bestehenden Specs betroffen)

## Impact

- **DB:** Migration 003 — `users.duty_reminder_days INT NULL DEFAULT NULL`, neue Tabelle `duty_reminder_log`
- **Backend:** `internal/scheduler` (neuer Job), `internal/mailer` (neues Template), neuer API-Endpoint `PUT /api/profile/reminder-preference`
- **Frontend:** Profil-Seite — neuer Toggle für Reminder-Preference
- **Cron:** Vorhandener Minuten-Cron auf dem VPS wird ohne Änderung mitgenutzt
