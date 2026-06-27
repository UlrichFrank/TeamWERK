# Tasks

Ein Commit pro Task (Conventional Commits). Scope variiert pro Task: `db`,
`trainings`, `games`, `scheduler`, `calendar`, `pwa`, `openspec`.

## 1. Datenbank-Migration

- [ ] 1.1 `internal/db/migrations/011_event_notes.up.sql` + `.down.sql`:
      `games.note` hinzufügen (`TEXT NOT NULL DEFAULT ''`), für `games.note`
      und `training_sessions.note` jeweils `CHECK (length(note) <= 200)` per
      Tabellen-Rebuild nachrüsten (SQLite-Recipe), Tabelle
      `pending_event_notes_push (ref_type, ref_id, note_text, notify_after,
      updated_by)` + `idx_pending_event_notes_due` anlegen. Down-Migration
      entfernt Tabelle + Spalte + CHECKs.
      _Commit: `feat(db): event-notes — games.note + Debounce-Queue (011)`_

- [ ] 1.2 Migrations-Test in `internal/db/migrations_test.go` (falls neu, sonst
      bestehendes Pattern): `011 up` → Spalten/Tabelle vorhanden, CHECK
      verhindert `length(note) > 200`-Insert; `011 down` → Spalten/Tabelle weg.

- [ ] 1.3 Aufräum-Logik in den bestehenden DELETE-Handlern (Game/Training):
      vor dem `DELETE` des Events auch `DELETE FROM pending_event_notes_push
      WHERE ref_type=? AND ref_id=?` — verhindert Karteileichen.

## 2. Backend — Routen für Hinweis-Edit

- [ ] 2.1 `internal/trainings/handler.go`: neue Methode `UpdateTrainingNote`
      (Body `{note}`, ≤ 200, Berechtigung via geteiltem `canEditTrainingNote`-
      Helper). Atomar: `UPDATE training_sessions SET note = ?`, UPSERT bzw.
      DELETE in `pending_event_notes_push` (siehe `design.md`), abschließend
      `h.hub.Broadcast("event-note")`. Route in `internal/app/router.go`
      hinzufügen unter dem authenticated-Tier mit `auth.RequireClubFunction`
      bzw. Custom-Check.
      _Commit: `feat(trainings): PUT /api/trainings/{id}/note für Hinweisfeld`_

- [ ] 2.2 Tests `internal/trainings/handler_test.go`:
      - `Trainings_SetNote_TrainerOwnTeam_Returns200`
      - `Trainings_SetNote_TrainerOtherTeam_Returns403`
      - `Trainings_SetNote_TooLong_Returns400`
      - `Trainings_SetNote_SecondEditResetsTimer` (zweiter Aufruf rückt
        `notify_after` auf `now+5min`)
      - `Trainings_SetNote_EmptyDeletesPending`

- [ ] 2.3 `internal/games/handler.go`: neue Methode `UpdateGameNote` analog
      mit Berechtigung Vorstand / Trainer-eines-beteiligten-Teams /
      sportliche_leitung / Admin. Route in `router.go`.
      _Commit: `feat(games): PUT /api/games/{id}/note für Hinweisfeld`_

- [ ] 2.4 Tests `internal/games/handler_test.go`:
      - `Games_SetNote_Vorstand_Returns200`
      - `Games_SetNote_StandardUser_Returns403`
      - `Games_SetNote_GenericEvent_Returns200`
      - `Games_SetNote_TooLong_Returns400`
      - `Games_SetNote_EmptyDeletesPending`

## 3. Backend — Scheduler-Job

- [ ] 3.1 Neuer Job in `internal/scheduler/event_notes_push.go`:
      `RunPendingEventNotesPush(ctx, db, cfg)` — fällige pending-Rows
      verarbeiten, Push nur bei `event_date >= today`, Row immer DELETE.
      Empfänger über bestehende Helper (`teamMembersAndParents`), Push über
      `notify.Send` mit Kategorie `trainings` bzw. `games`, URL auf
      `/termine/...`. Im Scheduler-Main (`cmd/teamwerk` Subcommand
      `scheduler:run`) als zusätzlicher Minuten-Tick einhängen.
      _Commit: `feat(scheduler): debounced Push für Termin-Hinweise`_

- [ ] 3.2 Tests `internal/scheduler/event_notes_push_test.go`:
      - `EventNotesPush_FutureEvent_SendsPush_DeletesRow`
      - `EventNotesPush_PastEvent_SkipsPush_DeletesRow`
      - `EventNotesPush_NotYetDue_KeepsRow`
      - `EventNotesPush_DeletedEvent_DeletesRow_NoPush` (Termin in der
        Zwischenzeit gelöscht → kein Crash, Row weg)

## 4. Backend — iCal-Feed

- [ ] 4.1 `internal/calendar/handler.go`:
      - `fetchGames`: SELECT um `g.note` erweitern, `calEvent.Description`
        setzen.
      - `fetchTrainings`: SELECT um `ts.note` erweitern, `calEvent.Description`
        setzen.
      _Commit: `feat(calendar): Termin-Hinweise im iCal-Feed`_

- [ ] 4.2 Tests `internal/calendar/handler_test.go`:
      - `ICal_TrainingWithNote_DescriptionInFeed`
      - `ICal_GameWithNote_DescriptionInFeed`
      - `ICal_TrainingEmptyNote_NoDescriptionLine` (oder leerer Wert — wie
        der bestehende Renderer mit leeren `Description` umgeht; im Test
        festschreiben, damit es nicht still bricht)

## 5. Frontend — Komponenten

- [ ] 5.1 `web/src/components/EventNoteIndicator.tsx`: zwei Varianten
      (`icon` mit `title`-Tooltip, `inline` mit voller Zeile). Rendert
      nichts bei leerem `note`. `lucide-react`-Icon `AlertTriangle`,
      Token `text-brand-danger`. Story/Snapshot-Test minimal.
      _Commit: `feat(pwa): EventNoteIndicator-Komponente für Termin-Hinweise`_

- [ ] 5.2 `web/src/components/EventNoteEditor.tsx`: Textarea (200 max),
      Counter `x/200`, Speichern-Button (Brand-Button), ruft
      `api.put('/trainings/{id}/note'|'/games/{id}/note', {note})`. Inline-
      Fehleranzeige, Loading-State, `onSaved`-Callback. Test mit MSW oder
      Axios-Mock: Save-Roundtrip, Längen-Block.
      _Commit: `feat(pwa): EventNoteEditor-Komponente für Termin-Hinweise`_

## 6. Frontend — Anzeigen verteilen

- [ ] 6.1 `web/src/pages/DashboardPage.tsx`: `MeineTermineSection` →
      `DashboardRow`-`badge`-Slot um `<EventNoteIndicator variant="icon" />`
      ergänzen. `NextEvent`-Typ um `note: string` erweitern + Backend-
      Endpoint anpassen, damit `note` mitkommt (`/dashboard`-API).
      `useLiveUpdates`-Branch um `'event-note'` ergänzen.
      _Commit: `feat(pwa): Hinweissymbol im Dashboard-Termin`_

- [ ] 6.2 `web/src/pages/KalenderPage.tsx`: Game-Tile + Training-Tile —
      `<EventNoteIndicator variant="icon" />` neben Time-Zeile. Backend
      liefert `note` bereits in `Game`/`Training`-Typ (Felder müssen ggf.
      nachgezogen werden — `note` ist im Frontend-Training-Interface zwar
      vorhanden, kommt aber bisher nicht aus dem GET-Response).
      `useLiveUpdates` um `'event-note'` ergänzen.
      _Commit: `feat(pwa): Hinweissymbol in Kalender-Tag-Tiles`_

- [ ] 6.3 `web/src/components/EventInfoModal.tsx`: bei vorhandenem
      `note` `<EventNoteIndicator variant="inline" />` + (für Berechtigte)
      `<EventNoteEditor />`. `useLiveUpdates` um `'event-note'` ergänzen
      (über Parent-Page).
      _Commit: `feat(pwa): Hinweis-Anzeige + Edit im EventInfoModal`_

- [ ] 6.4 `web/src/pages/TerminePage.tsx`: Training-Card + Game-Card —
      `<EventNoteIndicator variant="inline" />` als eigene Zeile unter
      `MapsLink`, analog zum existierenden `cancel_reason`-Pattern.
      `useLiveUpdates` um `'event-note'` ergänzen.
      _Commit: `feat(pwa): Hinweis-Zeile in Termine-Cards`_

- [ ] 6.5 `web/src/pages/TermineDetailPage.tsx`: eigene Sektion „Hinweis"
      mit `<EventNoteIndicator variant="inline" />` und (für Berechtigte)
      `<EventNoteEditor />`. `useLiveUpdates` um `'event-note'` ergänzen.
      _Commit: `feat(pwa): Hinweis-Sektion auf Termin-Detailseite`_

- [ ] 6.6 `web/src/components/GameEditModal.tsx` +
      `web/src/components/TrainingEditModal.tsx`: Textarea-Feld
      „Hinweis (optional)" mit 200-Zeichen-Counter. Das große Edit-Form
      schickt den Hinweis **nicht** über den großen PUT — sondern beim
      Speichern parallel den schmalen `/note`-Endpoint, damit die Debounce-
      Queue konsistent angesprochen wird. (Alternative: Hinweis aus dem
      großen Edit-Modal ganz raus und nur über `EventNoteEditor` editieren.
      Beim Implementieren entscheiden.)
      _Commit: `feat(pwa): Hinweis-Feld in Game-/Training-Edit-Modal`_

## 7. Tests-Anpassung Frontend

- [ ] 7.1 Existierende Tests für `KalenderPage`/`TerminePage`/
      `EventInfoModal`/`DashboardPage` so anpassen, dass `note`-Felder
      im Mock-Response vorhanden sind (leer + befüllt). Snapshot-Tests
      mit Hinweis-Indikator.

## 8. Architektur & Lint

- [ ] 8.1 `internal/arch/arch_test.go`: prüfen, ob neue Cross-Package-
      Importe entstehen (Scheduler → trainings/games-Helper?). Falls neue
      Imports nötig sind, Klassifikation anpassen oder über Foundation
      (`internal/notify`, `internal/hub`) entkoppeln.

- [ ] 8.2 Lint grün: `make lint`, `pnpm -C web lint`.

## 9. Verifikation

- [ ] 9.1 `make test` grün (inkl. Architektur-Test, Migrations-Test,
      neue Handler-Tests, Scheduler-Test).
- [ ] 9.2 `pnpm -C web test` + `pnpm -C web build` grün.
- [ ] 9.3 `openspec validate event-notes --strict` grün.
- [ ] 9.4 Manuelle Verifikation lokal:
      - Hinweis am Training anlegen → Icon im Kalender + Dashboard + Termin-
        Liste sichtbar, voller Text im Detail/Modal.
      - 5 min warten → Push erscheint (lokal mit kurzem Override testen).
      - Zweiter Edit innerhalb 5 min → Timer resettet, nur **ein** Push.
      - Hinweis am vergangenen Training → kein Push.
      - iCal-Feed importieren (Apple Calendar) → DESCRIPTION sichtbar.

## 10. Archivierung

- [ ] 10.1 Nach Merge: `openspec archive event-notes`.
