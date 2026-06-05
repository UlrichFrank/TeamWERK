## Context

Aktueller Stand:
- `kader_members` + `kader_trainers` sind die zwei Mitglieder-Kategorien je Kader
- `game_responses` speichert RSVP-Antworten (confirmed/declined/maybe)
- Spieldetail (`TermineDetailPage.tsx`) zeigt nur Respondenten (`gameResponses.map(...)`)
- Kein Konzept für Gelegenheitsspieler oder Trainer-kuratierte Aufstellung

Beide Features teilen die `kader_extended_members`-Tabelle: der erweiterte Kader ist der Pool, aus dem die Aufstellung auf Spielebene schöpft.

## Goals / Non-Goals

**Goals:**
- `kader_extended_members` — neue Tabelle analog zu `kader_trainers`
- Kader-Seite: dritter Abschnitt mit eigenem Search-Component und Liste
- `game_lineup` — neue Tabelle für Trainer-Nominierungen pro Spiel
- `GET /api/games/{id}/participants` — liefert alle regulären + erweiterten Kader-Mitglieder mit RSVP- und Lineup-Status
- `POST /api/games/{id}/lineup` — setzt Lineup-Einträge (bulk upsert)
- Spieldetail-Seite: neue Spalte „Aufstellung" mit Checkboxen (Trainer) bzw. read-only (andere)

**Non-Goals:**
- RSVP-Einladungen für erweiterte Kader-Mitglieder
- Aufstellung auf Trainings-Seiten
- Positionsangaben in der Aufstellung (z. B. Torhüter / Feldspieler)
- Push-Notifications bei Aufstellungsänderungen (separate Change denkbar)

## Decisions

### 1. Separate Tabelle statt type-Spalte in kader_members

**Entscheidung:** Neue Tabelle `kader_extended_members`, kein `type`-Feld in `kader_members`.

**Rationale:** `player_memberships`-View (gerade refactored) basiert auf `kader_members`; extended members dürfen dort nie auftauchen. Eine type-Spalte würde überall WHERE-Klauseln erfordern und ist fehleranfällig.

### 2. Neuer `/participants`-Endpoint statt Erweiterung von `/responses`

**Entscheidung:** `GET /api/games/{id}/participants` ist ein eigenständiger Endpoint, der reguläre + erweiterte Kader-Mitglieder mit RSVP- und Lineup-Status zurückgibt.

**Alternativen:**
- `/responses` erweitern → bricht semantische Trennung (responses = Antworten, nicht Kader)
- Frontend joined selbst → zu komplex, mehrere Requests

**Rationale:** Klare API-Semantik. `/responses` bleibt für Anwendungen, die nur Respondenten brauchen.

### 3. Lineup als Bulk-Set (POST überschreibt)

**Entscheidung:** `POST /api/games/{id}/lineup` empfängt die komplette Lineup-Liste und schreibt sie als Upsert + Delete-diff.

**Alternativen:** Toggle-Endpoint pro Mitglied → mehr Round-Trips beim Trainer

**Rationale:** Trainer arbeitet typischerweise die ganze Liste durch; ein einzelner Save ist UX-freundlicher.

### 4. Erweiterter Kader: kader_id-Bezug, nicht team_id

**Entscheidung:** `kader_extended_members` referenziert `kader_id`, nicht `team_id`.

**Rationale:** Ein Team kann mehrere Kader (Saisons) haben. Kader-Bezug ist konsistent mit `kader_members` und `kader_trainers`.

### 5. Participants-Endpoint: wer ist berechtigt?

Gleiche Logik wie bisher bei `/responses`:
- Trainer/Admin: immer
- Spieler: nur wenn eigenes Team
- Eltern: nur wenn Kind im Team

## Risks / Trade-offs

**[Risk] Spieldetail zeigt jetzt mehr Zeilen** → Trainer sieht ggf. 20+ Personen statt bisher nur Respondenten. Mitigation: Spalten-Layout bleibt kompakt; keine Paginierung nötig bei Handball-Teamgrößen.

**[Risk] kader_extended_members erscheinen in `/participants` ohne RSVP** → `rsvp_status` wird `null` zurückgegeben; Frontend zeigt „–". Klar definiert.

**[Trade-off] Lineup bulk-set: concurrent edits überschreiben sich** → akzeptiert (Last-Write-Wins, analog zu `training_attendances`).

## Migration Plan

1. Migration `018_erweiterter_kader.up.sql`: `CREATE TABLE kader_extended_members`
2. Migration `019_game_lineup.up.sql`: `CREATE TABLE game_lineup`
3. Backend: `kader/handler.go` — `loadExtendedMembers`, extended CRUD in `UpdateKader`
4. Backend: `games/handler.go` — `GetParticipants`, `SaveLineup`
5. Frontend: `AdminKaderPage.tsx` — dritter Abschnitt (neuer Component `KaderExtendedSearch`)
6. Frontend: `TermineDetailPage.tsx` — Datenquelle auf `/participants` umstellen, neue Spalte

## Open Questions

*(keine)*
