## Why

Trainer:innen und Vorstand brauchen einen Weg, an einzelnen Terminen kurzfristig
einen **Hinweis** zu hinterlegen — z. B. „Halle gesperrt, wir joggen am See"
oder „Bringt zusätzlich Hallenschuhe mit". Heute geht das nur per Chat-
Broadcast, der den Kontext zum Termin verliert und im Kalender niemandem
auffällt, der den Chat gerade nicht liest.

Ziel: ein **terminbezogenes Hinweisfeld**, das im Kalender sichtbar markiert
ist, jederzeit (auch nachträglich) editiert werden kann und die betroffenen
Mannschaftsmitglieder + Eltern per Push erreicht — aber **nicht** bei jedem
Tippfehler einen Push absetzt.

`training_sessions.note` existiert bereits seit `001_initial.up.sql`, wird im
Backend aber weder gelesen noch im Frontend angezeigt — wir reaktivieren das
Feld statt eine neue Spalte einzuführen. Für `games` fehlt das Pendant.

## What Changes

**Neue Capability `event-notes`** — quer über `trainings` und `games`, weil
Debounce-Mechanik, Indikator-Komponente und Push-Logik identisch sind und
nicht in beiden Capability-Specs dupliziert werden sollen.

### Backend

- **Migration `011_event_notes`:**
  - `ALTER TABLE games ADD COLUMN note TEXT NOT NULL DEFAULT ''`
  - `CHECK (length(note) <= 200)` für `games.note` **und** `training_sessions.note`
    (per Table-Rebuild, weil SQLite kein `ADD CONSTRAINT` kennt).
  - Neue Tabelle `pending_event_notes_push (ref_type, ref_id, note_text,
    notify_after, updated_by)` als Debounce-Queue.
- **Neue Routen** (schmal, unabhängig vom bestehenden „großen" Edit):
  - `PUT /api/trainings/{id}/note` (Trainer eig. Team / Vorstand / Admin)
  - `PUT /api/games/{id}/note` (Vorstand / Trainer / sportliche_leitung / Admin)
  - Body `{note: string}` mit `len ≤ 200`. Beide Routen:
    - validieren Länge (400 bei > 200 chars),
    - `UPDATE … SET note = ?`,
    - `UPSERT pending_event_notes_push … notify_after = now + 5min`,
    - leerer Text → `DELETE` aus pending-Queue (kein Push für „Hinweis weg"),
    - `h.hub.Broadcast("event-note")` für Live-UI-Update.
- **Scheduler-Job** in `internal/scheduler/`: liest fällige pending-Rows,
  sendet Push nur wenn `event_date >= today` (Vergangenheits-Pushes
  unterdrückt), `DELETE` der Row in jedem Fall (auch ohne Push, idempotent).
- **iCal-Feed** (`internal/calendar/handler.go`): `calEvent.Description` wird
  für Games und Trainings aus dem `note`-Feld befüllt (Feld existiert bereits,
  wird heute nur nicht gesetzt).

### Frontend

- **Zwei neue Komponenten in `web/src/components/`:**
  - `EventNoteIndicator` — `<AlertTriangle>` mit zwei Varianten:
    - `variant="icon"` (kompakt, `title`-Tooltip mit Hinweistext)
    - `variant="inline"` (Icon + voller Text in eigener Zeile)
  - `EventNoteEditor` — Textarea mit 200-Zeichen-Counter, ruft
    `PUT /api/{trainings|games}/{id}/note`, zeigt Loading/Fehler.
- **Verteilung der Indikatoren** (siehe `design.md` für Tabelle):
  - `/` DashboardPage Termin-Row → `icon`
  - `/kalender` Tag-Tile (Game + Training) → `icon`
  - `/kalender` EventInfoModal → `inline` + Inline-`EventNoteEditor`
  - `/termine` TerminePage Card → `inline`
  - `/termine/:id` TermineDetailPage → `inline` + Inline-`EventNoteEditor`
  - `GameEditModal` / `TrainingEditModal` → Textarea-Feld im Formular
- **Live-Updates:** `useLiveUpdates(e => { if (e === 'event-note') reload() })`
  in den vier Seiten + `EventInfoModal`.

## Scope

**In scope**

- `training_sessions.note` (reuse, sichtbar machen) + `games.note` (neu).
- Debounce-Queue + Scheduler-Job (5-min-Verzögerung, Reset bei jedem Edit).
- Push an `teamMembersAndParents(team_id[])` (bestehende Helper, keine neue
  Empfänger-Logik).
- iCal-Feed `DESCRIPTION` für Games und Trainings.
- UI-Anzeige + Edit an den oben gelisteten sechs Stellen.

**Out of scope**

- `duty_slots`, `member_absences`, `mitfahrgelegenheiten` — explizit nicht
  betroffen.
- Versions-/Audit-Historie der Hinweise (nur letzter Stand zählt).
- Mehrere parallele Hinweise / Threaded Comments pro Termin.
- Eigene Push-Kategorie / Notification-Preference — Hinweise nutzen die
  bestehenden Kategorien `trainings` bzw. `games`.
- Push an Termin-RSVPs (Spieler ohne Mannschaftsbindung) — Empfängermenge
  bleibt `teamMembersAndParents`.

## Test-Anforderungen

| Route / Komponente | Testname | Erwartetes Ergebnis |
|---|---|---|
| `PUT /api/trainings/{id}/note` | `trainings_setNote_trainerOwnTeam_ok` | 200 + `note` in DB + pending-Row mit `notify_after = now+5min` |
| `PUT /api/trainings/{id}/note` | `trainings_setNote_trainerOtherTeam_forbidden` | 403, DB unverändert |
| `PUT /api/trainings/{id}/note` | `trainings_setNote_tooLong_badRequest` | 400 bei `len(note) > 200`, DB unverändert |
| `PUT /api/trainings/{id}/note` | `trainings_setNote_secondEditResetsTimer` | zweiter Aufruf innerhalb 5 min → `notify_after` rückt auf `now+5min` |
| `PUT /api/trainings/{id}/note` | `trainings_setNote_emptyDeletesPending` | Body `note=""` → pending-Row entfernt, kein Push |
| `PUT /api/games/{id}/note` | `games_setNote_vorstand_ok` | 200 + `note` in DB + pending-Row |
| `PUT /api/games/{id}/note` | `games_setNote_standard_forbidden` | 403, DB unverändert |
| `PUT /api/games/{id}/note` | `games_setNote_genericEvent_ok` | 200 für `event_type=generisch` |
| Scheduler-Job | `scheduler_pendingNote_futureEvent_sendsPush` | `event_date >= today` & fällig → `notify.Send` an `teamMembersAndParents`, Row gelöscht |
| Scheduler-Job | `scheduler_pendingNote_pastEvent_skipsPush` | `event_date < today` → **kein** Push, Row trotzdem gelöscht |
| Scheduler-Job | `scheduler_pendingNote_notYetDue_keepsRow` | `notify_after > now` → keine Aktion, Row bleibt |
| iCal-Feed | `ical_trainingWithNote_descriptionSet` | `GET /api/calendar/feed` enthält `DESCRIPTION:<note>` für betroffenen Termin |
| iCal-Feed | `ical_gameWithNote_descriptionSet` | dito für Game |
| Architektur-Test | `arch_event_notes_classification` | neues Package `internal/eventnotes` (falls eingeführt) ist im `arch_test.go` klassifiziert |

**Invariante:** Ein Push-Reminder wird **nie** ohne einen Hinweistext
abgesetzt; ein Push wird **nie** für ein in der Vergangenheit liegendes
Event abgesetzt; ein Push wird **frühestens 5 Minuten** nach der letzten
Änderung am Hinweistext abgesetzt.
