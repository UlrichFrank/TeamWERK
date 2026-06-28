## Context

Der Scheduler (`internal/scheduler/scheduler.go`, minütlicher Cron via `teamwerk-scheduler.sh`) löst Event-Reminder aus, indem er Zeitfenster relativ zu `time.Now()` berechnet und gegen die DB-Spalten vergleicht:

```go
// sendGameReminders (heute)
now := time.Now()                              // Server-TZ = UTC auf dem VPS
from := now.Add(20 * time.Hour).Format("2006-01-02")
to   := now.Add(28 * time.Hour).Format("2006-01-02")
// WHERE g.date BETWEEN ? AND ?
```

Event-Zeiten liegen als **naive Wandzeit** vor: `games.date DATE` + `games.time TEXT` (HH:MM), analog `training_sessions.date`+`start_time` und `games.time` für Mitfahrten. Gemeint ist immer `Europe/Berlin`. Da der VPS in UTC läuft, driftet die Auslösung im Sommer um +2h und an Tagesgrenzen um bis zu einen Tag.

Ein **korrektes Pattern existiert bereits** im iCal-Export, `internal/calendar/handler.go`:

```go
func parseDT(date, timeStr string, loc *time.Location) time.Time {
    if len(date) > 10 { date = date[:10] }      // modernc liefert "2026-08-15T00:00:00Z"
    if timeStr == "" { timeStr = "00:00" }
    t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, loc)
    if err != nil { t, _ = time.ParseInLocation("2006-01-02", date, loc) }
    return t
}
```

## Goals / Non-Goals

**Goals:**
- Reminder feuern **wandzeit-korrekt** in `Europe/Berlin`, unabhängig von der Server-Zeitzone und über DST-Wechsel hinweg.
- Spiele und Trainings erhalten **zwei** Reminder: 24h vorher (Planung) und 3h vorher (am Event-Tag).
- Garantie „**maximal 24h vorher**": kein Reminder feuert früher als der 24h-Slot.
- Jeder Slot bleibt **idempotent** (genau ein Versand pro Nutzer+Event+Slot).
- Minimaler Footprint: Stdlib-`time`, keine Migration, kein Frontend.

**Non-Goals:**
- **Keine** UTC-Speicherung und **keine** Migration der Event-Spalten (siehe Decision 1).
- **Keine** Änderung der Dienst-Erinnerung (`sendDutyReminders`, 48h „offene Dienste") — bewusst lange Vorlaufzeit.
- **Keine** Änderung ereignisgesteuerter Pushes (z. B. „Neues Spiel angelegt").
- **Keine** Pro-Team-/Pro-Serie-Konfigurierbarkeit des 3h-Reminders (3h gilt fix für alle Trainings).

## Decisions

### Decision 1 — Wandzeit beibehalten statt auf UTC migrieren

Event-Daten bleiben naive `DATE`+`TEXT` mit impliziter Zeitzone `Europe/Berlin`. Der Scheduler interpretiert sie beim Vergleich in `Europe/Berlin`.

**Warum, nicht UTC-Speicherung:**
- Es sind **ortsfeste Wandzeit-Ereignisse** (Spiel um 15:00 in Stuttgart). Die fachlich relevante Größe ist die Wandzeit am Ort, nicht ein Instant.
- **Keine Multi-Zeitzonen-Nutzer** — alle Browser stehen auf Berlin; eine „Anzeige in lokaler Browser-Zeitzone" löst kein reales Problem.
- UTC-Speicherung würde **DST-Mehrdeutigkeit** einführen: Für ein Event in 6 Monaten müsste man heute den künftigen UTC-Offset (CET vs. CEST) annehmen; ändern sich DST-Regeln, verrutscht die Wandzeit.
- UTC-Umstellung wäre **breit und riskant** (Migration aller `games`/`training_sessions`/`duty_slots` + ~30 Frontend-`.slice(0,10)`-Stellen) — der Scheduler-Fix ist in beiden Modellen identisch.

*Alternative verworfen:* „Wandzeit + venue-Zeitzone explizit" — sinnvoll erst bei Multi-Region-Betrieb, heute Overhead ohne Nutzen.

### Decision 2 — `parseDT` in gemeinsamen Helper extrahieren

`parseDT` wird aus `internal/calendar` in einen für `scheduler` und `calendar` nutzbaren Ort gezogen (z. B. ein kleines Zeit-Util-Package unter `internal/` oder eine exportierte Funktion), **ohne** das Verhalten des iCal-Exports zu ändern. Damit gibt es genau **eine** Stelle, die Berlin-Wandzeit korrekt parst (inkl. der `len>10`-Normalisierung für modernc-Timestamps).

*Architektur-Test beachten:* Das Helper-Package muss in `internal/arch/arch_test.go` als Foundation klassifiziert werden und darf keine Domain-Packages importieren.

### Decision 3 — Zwei Slots über Zeitpunkt-Vergleich statt Datums-Fenster

Pro Event wird der Berlin-Instant `eventAt` gebildet und mit `now := time.Now().In(berlin)` verglichen:

```
24h-Slot: now <= eventAt && eventAt <= now.Add(24h)   → ref_type "<domain>_reminder_24h"
 3h-Slot: now <= eventAt && eventAt <= now.Add(3h)     → ref_type "<domain>_reminder_3h"
```

Beim minütlichen Lauf feuert jeder Slot beim **ersten** Lauf, in dem das Event sein Zeitfenster betritt; `notification_log` (Insert-vor-Send, `RowsAffected==1`) verhindert Wiederholung. Der 3h-Slot ist eine Teilmenge des 24h-Fensters, kollidiert aber nicht, weil er einen eigenen `ref_type` hat und der 24h-Slot dann längst protokolliert ist.

*Alternative verworfen:* exakte Fenster wie „BETWEEN now+2h59 AND now+3h01" — fragil gegenüber Cron-Jitter und verpassten Läufen (z. B. nach Neustart). Der „≤-Schwelle + Idempotenz"-Ansatz fängt verpasste Läufe sauber nach (feuert dann eben verspätet, aber genau einmal).

### Decision 4 — Idempotenz pro Slot via eigener `ref_type`

`notification_log(user_id, ref_type, ref_id)` bekommt neue String-Werte: `game_reminder_24h`, `game_reminder_3h`, `training_reminder_24h`, `training_reminder_3h`. Kein Schema-Eingriff. Mitfahrt behält `carpooling_reminder` (weiterhin ein Slot). Versand-Reihenfolge bleibt **Insert-vor-Send** (`INSERT OR IGNORE` → bei `RowsAffected==1` senden), wie bei der Dienst-Erinnerung dokumentiert.

## Risks / Trade-offs

- **Mehr Pushes pro Nutzer (1 → 2 je Spiel/Training)** → Bewusst akzeptiert; kleiner Verein, idempotent. 3h-Training-Ping kann als Spam empfunden werden (bewusste Option a; spätere Pro-Serie-Abschaltung bleibt möglich, ohne diese Spec zu brechen).
- **Verpasster Scheduler-Lauf** (VPS-Neustart, Cron-Ausfall) → „≤-Schwelle"-Logik feuert beim nächsten Lauf nach; Nutzer erhält den Reminder ggf. verspätet, aber genau einmal. Akzeptabel.
- **Event wird <24h vor Start angelegt** → 24h-Slot feuert beim nächsten Lauf sofort (Fenster bereits betreten). Gewünscht.
- **Vergangene Events** → `now <= eventAt` schließt sie aus; kein Reminder für Events in der Vergangenheit.
- **DST-Umstellungsnacht** → `ParseInLocation` in `Europe/Berlin` löst Offset korrekt auf; ein Event in der „doppelten"/„fehlenden" Stunde ist im Handballkontext praktisch irrelevant.
- **Helper-Extraktion ändert iCal-Verhalten** → Mitigation: reine Verschiebung ohne Logikänderung; bestehende `calendar`-Tests müssen grün bleiben.

## Migration Plan

1. Helper extrahieren, `calendar` darauf umstellen (Verhalten unverändert, Tests grün).
2. `sendGameReminders` / `sendTrainingReminders` auf Berlin-Instant + 24h/3h-Slots umstellen.
3. `sendCarpoolingReminders` auf exakt-3h-Slot + Berlin-Instant umstellen.
4. Tests ergänzen (siehe Spec-Szenarien), `make test` / Gate grün.
5. Deploy via `make deploy` — **keine** DB-Migration nötig.

**Rollback:** rein code-seitig (`git revert`), da keine Schema-/Datenänderung. Bereits gesetzte neue `ref_type`-Einträge in `notification_log` sind harmlos (würden nach Rollback nur ignoriert).

## Open Questions

- Keine offenen Punkte. Die Pro-Serie-Konfigurierbarkeit des 3h-Trainings-Reminders ist bewusst verschoben (Option a fix).
