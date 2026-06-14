## Context

`GET /api/training-sessions/{id}/attendances` liefert alle Mitglieder eines Teams für eine Trainingseinheit — sowohl aus dem primären Kader (`player_memberships`-View → `kader_members`) als auch aus dem erweiterten Kader (`kader_extended_members`). Das aktuelle SQL verwendet einen `WHERE EXISTS OR EXISTS`-Filter, der beide Quellen zusammenführt, ohne zu unterscheiden, aus welcher Quelle ein Mitglied stammt.

Das `ResponseTable`-Component in `TermineDetailPage.tsx` ist bereits korrekt implementiert: es filtert Zeilen nach `is_extended` und rendert eine eigene Sektion „Erweiterter Kader". Das Feld fehlt nur in der Datenbeschaffung.

Der identische Bug war in `games/handler.go` vorhanden und wurde dort mit einem UNION-Ansatz gelöst (`GetParticipants`).

## Goals / Non-Goals

**Goals:**
- `attendanceItem`-Struct und API-Response enthalten `is_extended`
- Primärer Kader → `is_extended: false`, erweiterter Kader → `is_extended: true`
- Mitglieder, die in beiden Kadern sind, erscheinen nur einmal (in der primären Gruppe)
- Frontend mappt das Feld durch zu `tableRows`

**Non-Goals:**
- Kein neues Endpoint-Design
- Keine DB-Migration
- Kein UI-Redesign (ResponseTable ist fertig)

## Decisions

### UNION statt CASE WHEN

**Entscheidung**: SQL-Umbau auf UNION-Muster (analog `GetParticipants` in `games/handler.go`), statt `CASE WHEN EXISTS(...) THEN 1 ELSE 0 END`.

**Rationale**: Der CASE-Ansatz würde bei Mitgliedern, die in beiden Kadern sind, `is_extended=0` korrekt setzen — aber der Ausdruck wäre komplex und schlechter lesbar. Das UNION-Muster ist schon etabliert im Codebase und macht die Trennung explizit.

**NOT EXISTS-Guard**: Der zweite UNION-Zweig (erweiterter Kader) bekommt ein `NOT EXISTS (SELECT 1 FROM player_memberships ...)`, damit Mitglieder im primären Kader nicht doppelt erscheinen.

### Frontend: Interface-Erweiterung statt Umbau

Das `AttendanceItem`-Interface erhält `is_extended?: boolean` (optional, damit bestehende Code-Stellen nicht brechen). Im `tableRows`-Mapping wird `a.is_extended` durchgereicht — `ResponseTable` macht den Rest.

## Risks / Trade-offs

- **player_memberships ist eine View**: Die View selektiert aus `kader_members`. Der UNION-Guard nutzt `player_memberships` für den NOT EXISTS — das ist korrekt, solange die View die gleichen Mitglieder abbildet wie die JOIN-Kette im ersten UNION-Zweig.
- **Kein Breaking Change**: Das neue Feld ist additiv. Alte Clients (falls vorhanden) ignorieren unbekannte JSON-Felder.
