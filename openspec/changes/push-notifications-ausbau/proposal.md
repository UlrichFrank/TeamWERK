## Why

Push Notifications sind in TeamWERK bereits technisch vollständig implementiert, werden aber nur für Chat genutzt. Zeitkritische Ereignisse wie Spielabsagen oder Trainingsausfälle erreichen Spieler und Eltern bisher nur, wenn sie aktiv die App öffnen — das ist unzuverlässig und führt zu Missverständnissen im Vereinsalltag.

## What Changes

- Push Notifications werden systematisch auf alle relevanten Domains ausgerollt
- Neue event-driven Notifications bei Mutationen in Games, Trainings, Duties und Carpooling
- Scheduled Reminders (via Scheduler/Cronjob) für Dienste, Spiele und Trainings
- Eine `notification_log`-Tabelle sichert Idempotenz für Scheduled Reminders
- Nutzer erhalten nur Notifications die sie direkt betreffen (Team-Zugehörigkeit, eigene Zuordnungen)

## Kandidaten-Übersicht

Die folgende Tabelle listet alle Push-Notification-Kandidaten. Bitte mit ✓/✗ oder durch Bearbeiten dieser Datei markieren, welche implementiert werden sollen.

### Event-driven: Spiele

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| G1 | Neues Spiel eingetragen | `CreateGame` | Alle Team-Mitglieder + deren Eltern | ★★ | ☐ |
| G2 | Spiel verschoben (Datum/Zeit/Ort) | `UpdateGame` | Alle Team-Mitglieder + deren Eltern | ★★★ | ☐ |
| G3 | Spiel abgesagt | `DeleteGame` | Alle Team-Mitglieder + deren Eltern | ★★★ | ☐ |

### Event-driven: Trainings

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| T1 | Einzelne Session abgesagt | `DeleteSession` | Team-Mitglieder + Eltern | ★★★ | ☐ |
| T2 | Session verschoben (Zeit/Ort) | `UpdateSession` | Team-Mitglieder + Eltern | ★★★ | ☐ |
| T3 | Ganze Serie gelöscht (ab sofort) | `DeleteSeries` | Team-Mitglieder + Eltern | ★★ | ☐ |

### Event-driven: Dienste

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| D1 | Neuer Dienst-Slot verfügbar | `CreateDutySlot` | Alle Berechtigten (spieler + elternteil + trainer) | ★★ | ☐ |
| D2 | Dienst-Slot gelöscht (zu dem man eingetragen ist) | `DeleteDutySlot` | Zugeteilte User | ★★★ | ☐ |
| D3 | Dienst-Zuteilung erhalten | `ClaimDutySlot` (andere Person) | Trainer/Admin des Teams | ★ | ☐ |
| D4 | Dienst als erfüllt markiert | `FulfillAssignment` | Zugeteilter User | ★ | ☐ |

### Event-driven: Fahrgemeinschaften

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| F1 | Fahrgemeinschaft zugesagt | Match accepted | Anfragender User | ★★ | ☐ |
| F2 | Fahrgemeinschaft abgesagt/storniert | Match cancelled | Betroffener User | ★★★ | ☐ |

### Event-driven: Mitgliedschaft

| # | Ereignis | Trigger | Empfänger | Prio | Implementieren? |
|---|----------|---------|-----------|------|-----------------|
| M1 | Neue Beitrittsanfrage eingegangen | `RequestMembership` | Alle Admins | ★★ | ☐ |
| M2 | Beitrittsanfrage genehmigt | `ApproveMembership` | User (hat jetzt Account) | ★ | ☐ |
| M3 | Beitrittsanfrage abgelehnt | `RejectMembership` | User | ★ | ☐ |

### Scheduled Reminders (Cronjob)

| # | Reminder | Zeitpunkt | Empfänger | Prio | Implementieren? |
|---|----------|-----------|-----------|------|-----------------|
| R1 | Spielerinnerung | 24h vorher | Team-Mitglieder + Eltern | ★★ | ☐ |
| R2 | Trainingserinnerung | 24h vorher | Team-Mitglieder + Eltern | ★ | ☐ |
| R3 | Dienst-Erinnerung | 48h vorher | Zugeteilter User | ★★★ | ☐ |
| R4 | Fahrgemeinschaft-Erinnerung | 3h vorher | Alle Teilnehmer | ★★ | ☐ |

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

**Neue DB-Tabelle:**
```sql
notification_log (user_id, ref_type, ref_id, sent_at)  -- Idempotenz für Reminders
```

**Frontend:** Keine Änderungen — Push-Subscription läuft bereits über `usePushSubscription` in `AppShell.tsx`.

**Abhängigkeiten:** Keine neuen — `notifications.SendToUsers` ist bereits implementiert.
