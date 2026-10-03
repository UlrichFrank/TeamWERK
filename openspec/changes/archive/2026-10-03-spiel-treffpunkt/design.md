# Design

## Context

Motivation siehe `proposal.md`, Verhalten siehe `specs/game-meeting-point/spec.md` und `specs/push-reminders/spec.md`.

Vorlage ist der Hinweistext eines Spiels (`games.note`, Capability `event-notes`): eigene schmale Route `PUT /api/games/{id}/note` neben dem großen `PUT /api/games/{id}`, Rechteprüfung `canEditGameNote` (`internal/games/handler.go`), Debounce-Tabelle `pending_event_notes_push` mit Scheduler-Job `processPendingEventNotes` (`internal/scheduler/event_notes_push.go`), Editor `EventNoteEditor` im `EventInfoModal`.

Den Anwurf `games.time` lesen heute zehn Pakete direkt per SQL (`games`, `calendar`, `dashboard`, `carpooling`, `scheduler`, `attendance`, `duties`, `gamestats`, `videos`). Domain-Pakete dürfen sich nicht gegenseitig importieren (Arch-Test).

## Goals / Non-Goals

**Goals:**
- Eine einzige Umrechnung Abstand → Uhrzeit, die alle lesenden Pakete teilen.
- Keine veraltete Treffzeit nach einer Anwurf-Verlegung, ohne dass Importe oder Bearbeiten-Pfade davon wissen müssen.
- Änderungs-Push ohne Fehlalarm bei zurückgenommenen Korrekturen.

**Non-Goals:**
- Treffzeit im `GameEditModal` oder im Anlegen-Dialog (spätere Erweiterung über dieselbe Route).
- Treffzeit für Trainings.
- Erinnerungen an der Treffzeit statt am Anwurf ausrichten (siehe Open Questions).
- Anzeige in Attendance-Matrix, Dienst-Export, Staffel-Spielplan, Video-Spielauswahl. Diese Pakete lesen `time` für andere Zwecke und bleiben unverändert.

## Decisions

### 1. Speicherung als Abstand, Eingabe als Uhrzeit

`games.meet_offset_minutes INTEGER NULL CHECK (meet_offset_minutes BETWEEN 0 AND 720)` und `games.meet_place TEXT NOT NULL DEFAULT '' CHECK (length(meet_place) <= 100)`. `NULL` heißt „keine Treffzeit".

Die Route nimmt eine Uhrzeit entgegen und rechnet sie gegen den **zum Schreibzeitpunkt gespeicherten** Anwurf um. Gelesen wird immer `Anwurf − Abstand`.

- *Alternative: absolute Uhrzeit speichern.* Sie veraltet still, sobald der H4A-Import (`h4aimport_handler.go`, `UPDATE games SET … time=?`) oder ein Bearbeiten den Anwurf verlegt. Jeder Schreibpfad auf `time` müsste die Treffzeit mitziehen. Verworfen.
- *Alternative: Abstand eingeben („90 min vorher").* Ist sauber, aber Trainer denken in Uhrzeiten. Die Umrechnung im Server kostet eine Zeile, der Editor zeigt den Hinweis „verschiebt sich mit dem Anwurf".

Die Obergrenze 720 Minuten schließt Tippfehler aus (z. B. 01:30 statt 13:30), ohne frühe Abfahrten zu Turnieren zu verbieten. Eine Treffzeit am Vortag lässt sich nicht **eingeben** (die Eingabe ist eine Uhrzeit am Spieltag und muss ≤ Anwurf sein), kann aber durch eine spätere Verlegung des Anwurfs **entstehen**. Deshalb liefert die API `meet_date` mit.

`CHECK` und `NULL`-Default erlauben `ALTER TABLE … ADD COLUMN`, ein Tabellen-Rebuild wie in Migration `011` ist nicht nötig.

### 2. Gemeinsame Umrechnung in `internal/timez`

`timez.MeetTime(date, startHHMM string, offset sql.NullInt64) (meetDate, meetHHMM string, ok bool)`. `timez` ist Foundation (siehe Arch-Test), also dürfen `games`, `calendar`, `dashboard`, `carpooling` und `scheduler` es importieren. Die Funktion schneidet `date` auf `[:10]` (DATE-Gotcha) und rechnet über `time.Date` in Minuten, damit der Vortag korrekt herauskommt.

- *Alternative: berechnete Spalte in SQL (`time(g.time, '-' || offset || ' minutes')`).* Das verliert den Tageswechsel und müsste in jedem Query wiederholt werden. Verworfen.

### 3. Rechte: `canEditGameNote` wiederverwenden, umbenennen

Der Personenkreis ist identisch. Die Funktion wird zu `canEditGameInfo` umbenannt und von beiden Routen aufgerufen. So gibt es keine zweite Kopie, die auseinanderlaufen kann. Die Route kommt in denselben Router-Block wie `PUT /api/games/{id}/note`. Prüfreihenfolge wie dort: ID → Body → Existenz (404) → Recht (403) → Validierung (400). Damit fällt die Objektrechte-Matrix nicht auf „Validierung vor Autorisierung" herein.

### 4. Live-Update über das bestehende Ereignis `games`

`h.broadcastGame(ctx, id, "games")` statt eines neuen Ereignisnamens. `TerminePage`, `TermineDetailPage`, `KalenderPage` und `DashboardPage` laden bei `games` ohnehin neu. Ein eigener Name (wie `event-note`) müsste an vier Stellen nachgezogen werden, und fehlt er an einer Stelle, bleibt die Seite still stumm.

### 5. Debounce-Tabelle mit Ausgangsstand

```
pending_game_meeting_push
  game_id       INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE
  prev_offset   INTEGER NULL      -- Stand vor der ersten Änderung im Fenster
  prev_place    TEXT NOT NULL
  notify_after  DATETIME NOT NULL
  updated_by    INTEGER NULL
```

Der Handler schreibt in derselben Transaktion wie das `UPDATE games`. Die alten Werte liest er davor. `ON CONFLICT(game_id) DO UPDATE SET notify_after=…, updated_by=…` lässt `prev_*` stehen. So bleibt der Ausgangsstand des Fensters erhalten, egal wie oft korrigiert wird.

Der Scheduler-Job liest fällige Zeilen, lädt den **aktuellen** Stand aus `games` und vergleicht ihn mit `prev_*`. Bei gleichem Stand sendet er nichts. Bei gesetzter Treffzeit meldet er „Treffen …", bei entfernter Treffzeit „Treffzeit entfällt". Das geht über `notify.Send(…, "games", …)` an `notify.TeamAudience(gameTeamIDs)`. Danach löscht er die Zeile immer. `ON DELETE CASCADE` erledigt gelöschte Spiele. Der Spieltag wird gegen `timez.Berlin()` geprüft.

- *Alternative: `pending_event_notes_push` mit neuem `ref_type` mitbenutzen.* Das scheitert am `CHECK (ref_type IN ('training','game'))`, der einen Tabellen-Rebuild bräuchte. Außerdem friert die Tabelle `note_text` ein, statt den Ausgangsstand zu halten. Verworfen.
- *Alternative: sofort senden ohne Debounce.* Zwei Korrekturen ergäben zwei Pushes an 30–40 Empfänger. Verworfen.

Der Meldungstext entsteht erst beim Senden aus dem aktuellen Stand. Deshalb ist er auch nach einer zwischenzeitlichen Anwurf-Verlegung korrekt.

### 6. Erinnerungen und iCal hängen nur einen Textteil an

`scheduler.go` (24 h/3 h) und `calendar/handler.go` lesen `meet_offset_minutes`/`meet_place` mit und hängen den Text über einen kleinen Formatierer an (`… — heute um 15:00 Uhr · Treffen 13:30 Uhr, Parkplatz VH`; iCal: erster Absatz über `joinDescription`). Den Formatierer pro Paket privat zu halten genügt: die Formate unterscheiden sich, gemeinsam ist nur die Umrechnung aus Entscheidung 2.

### 7. Frontend

- `MeetingPointEditor` (neu, `web/src/components/`): Uhrzeitfeld (`type="time"`), Ort-Textfeld (`maxLength=100`), Speichern, Entfernen, Hinweistext. Klassen aus `buttonStyles.ts`. Icon `Flag` aus lucide. Fehlercodes `meet_after_start`, `meet_offset_out_of_range`, `meet_place_too_long` und `meet_place_without_time` kommen in `web/src/lib/errors.ts`.
- Kleine Anzeige-Komponente `MeetingPointLine` für Liste, Detail und Modal (gleiches Format an allen Stellen, inklusive Vortag-Hinweis).
- Sichtbarkeit des Editors: das `can_edit` des Spiels, das `EventInfoModal` und die Detailseite heute schon für den `EventNoteEditor` nutzen.
- Mitfahrgelegenheiten: `FormModal` bekommt `defaultTreffpunkt` aus dem Spielkopf und nutzt ihn nur, wenn kein bestehender Eintrag übergeben wird (`fieldsFromEntry`).

## Risks / Trade-offs

- [Trainer erwartet nach einer Verlegung die alte Uhrzeit] → Hinweistext im Editor. Außerdem erhalten alle Betroffenen bei einer Verlegung ohnehin „Spielinfo geändert", und die neue Treffzeit steht überall dort, wo der neue Anwurf steht.
- [Spiele mit Platzhalter-Anwurf `00:00`] → Für diese ist keine Treffzeit eingebbar (`meet_after_start`). Das wird bewusst hingenommen, denn eine Treffzeit ohne echten Anwurf wäre eine erfundene Angabe.
- [3-h-Erinnerung kommt nach der Treffzeit, wenn das Treffen > 3 h vor dem Anwurf liegt] → Wird hingenommen, die 24-h-Erinnerung nennt die Treffzeit rechtzeitig. Siehe Open Questions.
- [Zwei Pushes bei gleichzeitiger Verlegung und Treffzeit-Änderung] („Spielinfo geändert" sofort, Treffzeit-Meldung nach 5 min) → Selten und inhaltlich korrekt. Ein Zusammenlegen würde zwei unabhängige Meldungswege koppeln.

## Migration Plan

Migration `073_game_meeting_point` (additiv: zwei Spalten, eine Tabelle). Bestandsspiele haben keine Treffzeit. Läuft automatisch mit `make deploy`. Rollback per `make deploy-rollback`: das alte Binary ignoriert die neuen Spalten und die Tabelle. `.down.sql` löscht die Tabelle und beide Spalten (`ALTER TABLE … DROP COLUMN`, SQLite ≥ 3.35 über modernc).

## Open Questions

- Soll die 3-h-Erinnerung bei gesetzter Treffzeit stattdessen 3 h vor dem **Treffen** feuern? Das ändert nur den Auslösezeitpunkt in `scheduler.go` und kann nach erster Nutzung entschieden werden.
