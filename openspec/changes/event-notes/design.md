# Design — event-notes

## Architektur-Überblick

```
                    EDIT-FLOW
─────────────────────────────────────────────────────────────

Trainer tippt
"Halle gesperrt"  ──►  PUT /api/trainings/42/note
                              │
                              ▼
                    ┌──────────────────────────┐
                    │ UPDATE training_sessions │
                    │   SET note = ?           │
                    │ UPSERT pending_event_…   │
                    │   notify_after = now+5m  │
                    │ Hub.Broadcast            │
                    │   "event-note"           │
                    └──────────────────────────┘
                              │
                ┌─────────────┴─────────────┐
                ▼                           ▼
       Live UI-Update             Trainer tippt Korrektur
       (alle offenen              → UPSERT, notify_after reset
        Browser sehen
        Icon sofort)
                                            │
                                            ▼
                                    … 5 min Ruhe …
                                            │
                                            ▼
                              Scheduler-Tick (* * * * *)
                              SELECT pending
                              WHERE notify_after <= now
                                            │
                       ┌────────────────────┼────────────────────┐
                       ▼                    ▼                    ▼
            event_date >= today    event_date < today    keine Aktion
            push.SendToUsers(      DELETE row             (notify_after > now)
              teamMembersAnd      (kein Push)
              Parents)
            DELETE row
```

## Datenbankschema

### Migration `011_event_notes`

```sql
-- games.note hinzufügen mit CHECK ≤ 200
-- (SQLite-Pattern: Spalte hinzufügen, dann Tabelle rebuild für CHECK)
ALTER TABLE games ADD COLUMN note TEXT NOT NULL DEFAULT '';

-- training_sessions.note: bestehende Spalte, CHECK nachrüsten
-- → Tabelle rebuild (Standard-SQLite-Recipe)
PRAGMA foreign_keys = OFF;
CREATE TABLE training_sessions_new (... mit CHECK (length(note) <= 200) ...);
INSERT INTO training_sessions_new SELECT ... FROM training_sessions;
DROP TABLE training_sessions; ALTER TABLE training_sessions_new RENAME TO training_sessions;
-- Indices/Trigger neu anlegen.
PRAGMA foreign_keys = ON;

-- Analog games-Tabelle rebuild für CHECK auf games.note.

-- Debounce-Queue
CREATE TABLE pending_event_notes_push (
    ref_type     TEXT     NOT NULL CHECK (ref_type IN ('training','game')),
    ref_id       INTEGER  NOT NULL,
    note_text    TEXT     NOT NULL,
    notify_after DATETIME NOT NULL,
    updated_by   INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (ref_type, ref_id)
);
CREATE INDEX idx_pending_event_notes_due ON pending_event_notes_push(notify_after);
```

Begründung Schema:
- `note_text` als Snapshot vermeidet Race „User editiert Hinweis weg, Scheduler
  pusht trotzdem den alten Text" — die Queue trägt selbst den zu sendenden Text.
- Primary Key `(ref_type, ref_id)` macht UPSERT trivial und garantiert
  „max. ein pending Push pro Termin".
- `ON DELETE` auf `games`/`training_sessions` löst pending-Rows **nicht**
  kaskadiert — der Scheduler entfernt sie beim nächsten Tick (FK auf
  `ref_id` wäre nicht typed über `ref_type`). Beim DELETE des Events räumen
  die zuständigen Handler die pending-Row mit auf, siehe `tasks.md` § 1.3.

## Routen

| Route | Auth | Body | Response |
|---|---|---|---|
| `PUT /api/trainings/{id}/note` | Trainer eig. Team / Vorstand / Admin | `{"note": "..."}` (≤ 200) | 200 `{}` · 400 zu lang · 403 falsch berechtigt · 404 unbekannt |
| `PUT /api/games/{id}/note` | Vorstand / Trainer / sportliche_leitung / Admin | `{"note": "..."}` (≤ 200) | wie oben |

**Berechtigungslogik** parallel zu den bestehenden Edit-Pfaden:
- Training: `UpdateTrainingSession` in `internal/trainings/handler.go` —
  prüft `claims.HasFunction("vorstand")` ODER „Trainer dieser Mannschaft"
  ODER Admin.
- Game: `UpdateGame` in `internal/games/handler.go` — `vorstand` ODER
  `trainer`/`sportliche_leitung` (beliebiges Team eines Game-Teams)
  ODER Admin.

Beide neuen Endpoints lagern die Berechtigungsprüfung in eine
wiederverwendete private Funktion `canEditEventNote(claims, eventID)` aus,
damit der bestehende große Update-Handler nicht angefasst werden muss.

## Debounce-Mechanik

```go
// Pseudo-Code beider Handler nach Validierung:
tx, _ := h.db.BeginTx(ctx, nil)
defer tx.Rollback()

_, _ = tx.ExecContext(ctx, `UPDATE games SET note = ? WHERE id = ?`, note, id)

if strings.TrimSpace(note) == "" {
    _, _ = tx.ExecContext(ctx,
        `DELETE FROM pending_event_notes_push WHERE ref_type = ? AND ref_id = ?`,
        "game", id)
} else {
    _, _ = tx.ExecContext(ctx, `
        INSERT INTO pending_event_notes_push (ref_type, ref_id, note_text, notify_after, updated_by)
        VALUES (?, ?, ?, datetime('now', '+5 minutes'), ?)
        ON CONFLICT(ref_type, ref_id) DO UPDATE SET
            note_text    = excluded.note_text,
            notify_after = excluded.notify_after,
            updated_by   = excluded.updated_by`,
        "game", id, note, claims.UserID)
}
tx.Commit()
h.hub.Broadcast("event-note")
```

Eigenschaften:
- **Atomar** mit dem Note-Update.
- **Idempotent** über UPSERT auf PK `(ref_type, ref_id)`.
- **Reset bei jedem Edit** — `notify_after` immer neu auf `now+5min`.
- **Leerer Text** → pending-Row weg, kein Push.

## Scheduler-Job

Neuer Job in `internal/scheduler/`, läuft im bestehenden Minuten-Cron:

```go
func runPendingEventNotesPush(ctx context.Context, db *sql.DB, cfg *config.Config) error {
    rows, _ := db.QueryContext(ctx, `
        SELECT ref_type, ref_id, note_text
        FROM pending_event_notes_push
        WHERE notify_after <= datetime('now')`)
    // … für jede Row:
    //   if eventDate(refType, refID) >= today:
    //       notify.Send(db, cfg, teamMembersAndParents(...),
    //                   category, title, noteText, deepLink)
    //   DELETE FROM pending_event_notes_push WHERE ref_type=? AND ref_id=?
    // Fehler einzelner Rows blockieren den Lauf nicht.
}
```

**Push-Inhalt**

| Feld | Wert |
|---|---|
| category | `"trainings"` bzw. `"games"` (bestehende Kategorie) |
| title | `Hinweis zu <Termin-Kurzbeschreibung>` (z. B. „Hinweis zu Training Mi 17:00") |
| body | `noteText` (max. 200 Zeichen, passt in Push-Limits) |
| url | `/termine/training/{id}` bzw. `/termine/{spiel\|ereignis}/{id}` |
| empfänger | `teamMembersAndParents(team_ids)` |

**Vergangenheits-Filter:** Vergleich auf `event_date >= date('now')` in der
Server-Zeitzone. Tagesgenau, nicht stundengenau — ein Hinweis, der morgens
zum heutigen Abendtraining gepostet wird, geht raus; ein Hinweis zum gestrigen
Training nicht. Das matcht die Real-World-Intention („nachträglich" für
laufende oder kommende Termine).

## UI-Verteilung

```
═══════════════════════════════════════════════════════════════════════════
 STELLE                                       INDIKATOR              TEXT?
═══════════════════════════════════════════════════════════════════════════
 /  DashboardPage › MeineTermineSection       AlertTriangle w-4 h-4   NEIN
    DashboardRow badge-Slot                   title="Hinweis: <text>"
    → Row ist dicht, Subtitle bereits truncated.

 /kalender › Monatsgrid Tag-Tile              AlertTriangle w-3 h-3   NEIN
    Game-Button: in der gap-1-Headerleiste    title="Hinweis: <text>"
    Training-Button: dito
    → @container Tile zu schmal; Tooltip füllt's.

 /kalender › EventInfoModal (Click-Popup)     AlertTriangle w-4 h-4   JA, voll
    Eigene Zeile + EventNoteEditor (inline,
    nur sichtbar wenn canEdit)

 /termine › TerminePage › Termin-Card         AlertTriangle w-4 h-4   JA, voll
    Eigene Zeile unter MapsLink, Pattern wie
    cancel_reason.

 /termine/:id  TermineDetailPage              AlertTriangle w-5 h-5   JA, voll
    Eigene Sektion + EventNoteEditor inline

 GameEditModal / TrainingEditModal            keiner                  EDIT-FELD
    Textarea im Formular, 200-Zeichen-Counter
═══════════════════════════════════════════════════════════════════════════
```

**Komponentenverträge**

```tsx
type EventNoteIndicatorProps =
  | { variant: 'icon'; note: string; className?: string }
  | { variant: 'inline'; note: string; className?: string }

// 'icon'   → <button aria-label="Hinweis vorhanden" title={`Hinweis: ${note}`}>
//              <AlertTriangle className="w-4 h-4 text-brand-danger" />
//            </button>
//            (rendert nur, wenn note.trim() !== "")
//
// 'inline' → <div className="flex items-start gap-2 text-sm text-brand-danger">
//              <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
//              <span className="whitespace-pre-wrap">{note}</span>
//            </div>

type EventNoteEditorProps = {
  eventType: 'training' | 'game'
  eventId: number
  initialNote: string
  onSaved?: (newNote: string) => void
}
// Textarea (200 max), Counter "x/200", Speichern-Button (deaktiviert
// solange unverändert oder > 200), Live-Validierung, Fehler-Inline.
```

## iCal-Feed

`internal/calendar/handler.go`:
- `fetchGames`: `SELECT ... g.note FROM games g ...`, in `calEvent{...,
  Description: note}` setzen.
- `fetchTrainings`: `SELECT ... ts.note FROM training_sessions ts ...`,
  ebenfalls `Description: note` setzen.

`renderICal` schreibt bereits `DESCRIPTION` heute — keine Änderung am Renderer
nötig. Externe Kalender (Apple/Google) zeigen den Hinweis dann automatisch.

## Sicherheits- und Konsistenzbetrachtungen

1. **Kein PII / kein Crypto:** Hinweise sind freier Klartext für die
   Mannschaft, kein Bank-/SEPA-Datum → kein Vault, kein Envelope.
2. **XSS-Vektor:** Note ist beliebiger Text. Im Frontend nur über React
   rendern (keine `dangerouslySetInnerHTML`), im iCal-Feed über das
   bestehende `escapeText` laufen lassen (existierender Helper für
   `SUMMARY`/`DESCRIPTION` in `handler.go`).
3. **Längen-Validierung doppelt:** Frontend-Counter + Server-`CHECK` —
   nicht „nur DB" (zu späte Fehlermeldung) und nicht „nur Frontend"
   (umgeh­bar via direktem HTTP-Aufruf).
4. **Race „Termin gelöscht während pending":** Scheduler-Job liest pending
   und resolved Event-Daten in derselben Query (LEFT JOIN auf
   `games`/`training_sessions`); fehlt der Termin → `DELETE` der pending-Row
   ohne Push.
5. **Race „User-Account gelöscht":** `updated_by` ist `ON DELETE SET NULL`,
   stört nichts.
6. **Hub-Broadcast nicht-debounced:** Live-UI-Update kommt sofort beim PUT —
   ein Trainer, der Hinweis korrigiert, sieht im Live-Browser direkt das
   Ergebnis seines letzten Saves, ohne 5-min-Wartezeit. Nur der Push selbst
   ist debounced.

## Architekturklassifikation

Wenn ein neues Package `internal/eventnotes` entsteht (für die beiden neuen
Handler + Scheduler-Job), muss es in `internal/arch/arch_test.go`
klassifiziert werden (Domain, importiert nur Foundation + `notify` + `hub`).

Alternativ: die zwei `PUT …/note`-Handler bleiben in den bestehenden
Packages `internal/trainings` und `internal/games`, der Scheduler-Job in
`internal/scheduler`. Tendenz: **letzteres** — vermeidet ein neues Package
und hält die Handler nah an den existierenden Edit-Pfaden, die sie ohnehin
referenzieren. Die geteilte Berechtigungslogik wandert in einen kleinen
Helper im jeweiligen Domain-Package.

## Verworfene Alternativen

- **Push bei jedem Save** (kein Debounce): Spam bei normalem Tippen, der
  Trainer würde es nicht nutzen.
- **Push manuell auslösen (Checkbox „benachrichtigen")**: ein Klick mehr,
  und Trainer würden ihn vergessen — Empfänger sähen Hinweise gar nicht.
- **Eigene Tabelle `event_notes` statt Spalte:** zwei zusätzliche Joins
  für eine einzelne Zeile pro Event, kein Mehrwert. Verworfen.
- **Versions-Historie:** spannender Use-Case, aber für „Halle gesperrt"
  überengineered. Letzter Stand reicht. Kann bei Bedarf nachgezogen werden.
- **Push an alle Termin-RSVPs** (nicht nur Mannschaft + Eltern):
  konsistenter mit Game-Karten quer durch Mannschaften, aber inkonsistent
  zu allen anderen Trainings-/Spiele-Benachrichtigungen. Verworfen, um die
  Empfänger-Regel projektweit einheitlich zu halten.
