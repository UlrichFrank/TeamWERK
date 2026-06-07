## Why

Push Notifications sind in TeamWERK bereits technisch vollständig implementiert, werden aber nur für Chat genutzt. Zeitkritische Ereignisse wie Spielabsagen oder Trainingsausfälle erreichen Spieler und Eltern bisher nur, wenn sie aktiv die App öffnen — das ist unzuverlässig und führt zu Missverständnissen im Vereinsalltag.

## What Changes

- Push Notifications werden systematisch auf alle relevanten Domains ausgerollt
- Neue event-driven Notifications bei Mutationen in Games, Trainings, Duties und Carpooling
- Scheduled Reminders (via Scheduler/Cronjob) für Dienste, Spiele und Trainings
- Eine `notification_log`-Tabelle sichert Idempotenz für Scheduled Reminders
- Nutzer erhalten nur Notifications die sie direkt betreffen (Team-Zugehörigkeit, eigene Zuordnungen)
- **Nutzer können pro Kategorie steuern, ob sie Push-Notifications und/oder E-Mail-Erinnerungen erhalten** — einstellbar im Profil unter „Sonstiges"

## Kandidaten-Übersicht

Die folgende Tabelle listet alle Push-Notification-Kandidaten. Bitte mit ✓/✗ oder durch Bearbeiten dieser Datei markieren, welche implementiert werden sollen.

### Event-driven: Spiele

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| G1 | Neues Spiel eingetragen | `CreateGame` | Alle Team-Mitglieder + deren Eltern | ★★ | ✓ |
| G2 | Spiel verschoben (Datum/Zeit/Ort) | `UpdateGame` | Alle Team-Mitglieder + deren Eltern | ★★★ | ✓ |
| G3 | Spiel abgesagt | `DeleteGame` | Alle Team-Mitglieder + deren Eltern | ★★★ | ✓ |

### Event-driven: Trainings

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| T1 | Einzelne Session abgesagt | `DeleteSession` | Team-Mitglieder + Eltern | ★★★ | ✓ |
| T2 | Session verschoben (Zeit/Ort) | `UpdateSession` | Team-Mitglieder + Eltern | ★★★ | ✓ |
| T3 | Ganze Serie gelöscht (ab sofort) | `DeleteSeries` | Team-Mitglieder + Eltern | ★★ | ✓ |

### Event-driven: Dienste

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| D1 | Neuer Dienst-Slot verfügbar | `CreateDutySlot` | Alle Berechtigten (spieler + elternteil + trainer) | ★★ | ✓ |
| D2 | Dienst-Slot gelöscht (zu dem man eingetragen ist) | `DeleteDutySlot` | Zugeteilte User | ★★★ | ✓ |
| D3 | Dienst-Zuteilung erhalten | `ClaimDutySlot` (andere Person) | Trainer/Admin des Teams | ★ | ☐ |
| D4 | Dienst als erfüllt markiert | `FulfillAssignment` | Zugeteilter User | ★ | ☐ |

### Event-driven: Fahrgemeinschaften

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| F1 | Fahrgemeinschaft zugesagt | Match accepted | Anfragender User | ★★ | ✓ |
| F2 | Fahrgemeinschaft abgesagt/storniert | Match cancelled | Betroffener User | ★★★ | ✓ |

### Event-driven: Mitgliedschaft

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| M1 | Neue Beitrittsanfrage eingegangen | `RequestMembership` | Alle Admins | ★★ | ✓ |
| M2 | Beitrittsanfrage genehmigt | `ApproveMembership` | User (hat jetzt Account) | ★ | ☐ |
| M3 | Beitrittsanfrage abgelehnt | `RejectMembership` | User | ★ | ☐ |

### Scheduled Reminders (Cronjob)

| # | Reminder | Zeitpunkt | Empfänger | Prio | Implementieren? |
|---|----------|-----------|-----------|------|-----------------|
| R1 | Spielerinnerung | 24h vorher | Team-Mitglieder + Eltern | ★★ | ✓ |
| R2 | Trainingserinnerung | 24h vorher | Team-Mitglieder + Eltern | ★ | ✓ |
| R3 | Dienst-Erinnerung | 48h vorher | Zugeteilter User | ★★★ | ✓ |
| R4 | Fahrgemeinschaft-Erinnerung | 3h vorher | Alle Teilnehmer | ★★ | ✓ |

> **Hinweis zu R3:** Für Dienst-Erinnerungen kann der Nutzer zusätzlich zur Push Notification auch eine E-Mail-Erinnerung aktivieren — konfigurierbar im Profil (s. unten). Push und E-Mail sind unabhängig voneinander ein-/ausschaltbar.

### Notification-Präferenzen (Profil)

Unter `/profil` → „Sonstiges" erscheint ein neuer Abschnitt **„Benachrichtigungen"**. Nutzer können dort pro Kategorie steuern:

| Kategorie | Push | E-Mail |
|-----------|------|--------|
| Spiele (neu, verschoben, abgesagt) | ✓ ein/aus | — |
| Trainings (abgesagt, verschoben) | ✓ ein/aus | — |
| Dienste (neuer Slot, Slot gelöscht) | ✓ ein/aus | — |
| Dienst-Erinnerung (48h vorher) | ✓ ein/aus | ✓ ein/aus |
| Fahrgemeinschaften | ✓ ein/aus | — |

E-Mail-Option ist nur dort sinnvoll, wo ein zeitkritischer Reminder nötig ist (Dienst-Erinnerung). Für reine Event-Notifications reicht Push.

Standard bei neuen Accounts: alle Push-Kategorien **ein**, E-Mail-Erinnerung **aus** (opt-in).

### Nicht implementieren (Begründung)

| Ereignis | Warum nicht |
|----------|-------------|
| Auth-Einladung gesendet | User hat noch keinen Account → kein Push möglich, E-Mail ist der richtige Kanal |
| Mitglieder-Stammdaten geändert | Zu granular, kein unmittelbarer Handlungsbedarf |
| Dienst-Zuteilung durch Admin (Bulk) | Würde Notification-Spam erzeugen bei Massenoperationen |

---

*Nach deiner Auswahl wird das Design auf die markierten Kandidaten zugeschnitten.*

## Capabilities

### New Capabilities

- `push-games`: Push Notifications bei Spiel-Mutationen (erstellt, verschoben, abgesagt)
- `push-trainings`: Push Notifications bei Training-Mutationen (abgesagt, verschoben, Serie gelöscht)
- `push-duties`: Push Notifications bei Dienst-Ereignissen (neuer Slot, Slot gelöscht, Zuteilung)
- `push-carpooling`: Push Notifications bei Fahrgemeinschafts-Ereignissen
- `push-membership`: Push Notifications bei Mitgliedschafts-Anfragen
- `push-reminders`: Scheduled Reminders via Scheduler (Spiele, Trainings, Dienste, Fahrgemeinschaften)
- `notification-preferences`: Nutzer-konfigurierbare Präferenzen pro Kategorie (Push ein/aus, E-Mail ein/aus) im Profil unter „Sonstiges"

### Modified Capabilities

*(keine bestehenden Specs betroffen)*

## Impact

**Backend:**
- `internal/games/handler.go`: `CreateGame`, `UpdateGame`, `DeleteGame` → `go notifications.SendToUsers(...)`
- `internal/trainings/handler.go`: `DeleteSession`, `UpdateSession`, `DeleteSeries` → Push
- `internal/duties/handler.go`: `CreateDutySlot`, `DeleteDutySlot` → Push
- `internal/carpooling/`: Fahrgemeinschafts-Events → Push
- `internal/members/handler.go`: Membership-Approve/Reject → Push
- `internal/scheduler/`: Neue Job-Typen für alle Scheduled Reminders
- `internal/db/migrations/`: Neue Migration für `notification_log`-Tabelle

**Neue DB-Tabellen:**
```sql
notification_log (user_id, ref_type, ref_id, sent_at)
-- Idempotenz für Reminders

notification_preferences (user_id PK FK, category TEXT, push_enabled BOOLEAN, email_enabled BOOLEAN)
-- Pro-Kategorie-Einstellungen; category ∈ {'games','trainings','duties','duty_reminders','carpooling'}
-- Default: push_enabled=1, email_enabled=0 (wird beim ersten Aufruf des Profils ggf. lazy angelegt)
```

**Frontend:**
- `web/src/pages/ProfilePage.tsx`: Neuer Abschnitt „Benachrichtigungen" unter „Sonstiges" mit Toggle-Rows pro Kategorie
- `GET /api/profile/notification-preferences` + `PUT /api/profile/notification-preferences`
- Push-Subscription läuft weiterhin über `usePushSubscription` in `AppShell.tsx` (unverändert)

**Backend (Notification-Logik):**
- Vor jedem `SendToUsers`-Aufruf und E-Mail-Versand: Präferenz des Empfängers prüfen
- Scheduler-Job für R3 liest `email_enabled` aus `notification_preferences` und sendet ggf. zusätzlich E-Mail via `h.mailer.Send(...)`

**Abhängigkeiten:** Keine neuen — `notifications.SendToUsers` ist bereits implementiert.
