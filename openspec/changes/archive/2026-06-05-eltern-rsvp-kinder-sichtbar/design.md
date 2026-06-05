## Context

Die Terminseite (`/termine`) zeigt Trainings und Spiele mit RSVP-Buttons. Der RSVP-Status wird pro Session via `my_rsvp` aus `training_responses`/`game_responses` gelesen, wobei `member_id = memberIDForUser(callerUserID)` als Filter dient. Eltern (Rolle `elternteil`) haben keine eigene Spieler-Rolle und damit keinen eigenen Member-Record — `memberIDForUser` liefert `0`, `my_rsvp` bleibt immer `null`.

Zusätzlich erwartet der `Respond`-Handler für `elternteil` ein `member_id`-Feld im Body. Das Frontend sendet es nicht → stiller 400-Fehler, kein catch-Block, keine Rückmeldung.

Die Detail-Seite (`/termine/training/:id`) zeigt die vollständige Kaderliste nur für Trainer (via `GetAttendances` mit `hasTeamAccess`-Guard). Spieler und Eltern sehen nur die Responses-Liste (nur wer bereits geantwortet hat).

## Goals / Non-Goals

**Goals:**
- Eltern sehen pro Kind einen eigenen RSVP-Abschnitt auf der Terminliste
- Eltern können RSVPs stellvertretend für Kinder abgeben und ändern
- Kommentare werden korrekt gespeichert und sind für den Verfasser sichtbar (Spieler: eigener, Elternteil: Kinder, Trainer: alle)
- Vollständige Kaderliste auf der Detail-Seite für Spieler und Eltern sichtbar (ohne Anwesenheits-Checkboxen)

**Non-Goals:**
- Keine DB-Migration notwendig (alle Felder bereits vorhanden)
- Keine Änderung am RSVP-Flow für Spieler oder Trainer
- Keine Push-Notifications beim Kind-RSVP
- Spiel-Detail-Seite erhält ebenfalls die vollständige Kaderliste (analog Training, gleiche Logik)

## Decisions

### 1. `children_rsvp` als eigenes Feld in der List-Response

**Entscheidung:** `children_rsvp: [{member_id, name, rsvp}]` wird als zusätzliches Feld in `sessionListItem` und `gameListItem` hinzugefügt — nur befüllt wenn `claims.IsParent`.

**Alternativen:**
- `my_rsvp` durch ein Union-Typ ersetzen: würde alle bestehenden Clients brechen.
- Separater Endpoint `/api/children/rsvp?from=&to=`: extra Round-Trip, schwieriger zu cachen.

**Rationale:** Additives Feld, keine Breaking Change, einfach konsumierbar im Frontend.

### 2. Batch-Query für children_rsvp (keine N+1)

**Entscheidung:** Nach dem Laden der Session-Liste wird eine einzelne Query ausgeführt, die alle Kinder-RSVPs für alle Session-IDs des Zeitraums auf einmal lädt — in Go nach `training_id` gruppiert.

```sql
SELECT tr.training_id, m.id, m.first_name || ' ' || m.last_name, tr.status
FROM family_links fl
JOIN members m ON m.id = fl.member_id
LEFT JOIN training_responses tr ON tr.member_id = fl.member_id
  AND tr.training_id IN (/* alle session IDs */)
WHERE fl.parent_user_id = ?
```

**Rationale:** Zeitfenster enthält typisch ~20–60 Sessions; N+1 wäre akzeptabel bei SQLite-Latenz im ns-Bereich, aber ein einzelner Query ist sauberer.

### 3. `GetAttendances` öffnen für alle authentifizierten Team-Mitglieder

**Entscheidung:** Den `hasTeamAccess`-Guard durch eine dreistufige Prüfung ersetzen:
1. Admin/Trainer → voller Zugriff (inkl. `present`-Checkboxen und alle Kommentare)
2. Spieler, der Kader-Mitglied dieses Teams ist → Kaderliste ohne `present`, nur eigener Kommentar
3. Elternteil, dessen Kind Kader-Mitglied ist → Kaderliste ohne `present`, nur Kinder-Kommentare

**Alternativen:**
- Neuer Endpoint `/training-sessions/:id/participants`: saubere Trennung, aber doppelte Query-Logik.

**Rationale:** `GetAttendances` liefert bereits die Kaderliste mit RSVP-Status — nur die Access-Control und das Reason-Feld fehlen. Minimale Änderung.

### 4. `reason` im Attendances-Response mit Rollen-Filter

**Entscheidung:** `reason` wird zur `GetAttendances`-Query via LEFT JOIN auf `training_responses` hinzugefügt. Die Filterlogik ist identisch zum bestehenden `GetSession`-Handler:

```go
canSeeReason := isTrainerLike ||
    (myMemberID > 0 && row.MemberID == myMemberID) ||
    childMemberIDs[row.MemberID]
```

`ListGameResponses` erhält analoge Logik für Spiele, gibt zusätzlich alle Kader-Mitglieder zurück (nicht nur Responder).

### 5. Frontend: `isParent` aus AuthContext ableiten

**Entscheidung:** `user.is_parent` aus dem JWT-Payload lesen (bereits im `Claims`-Struct und im Frontend-User-Objekt verfügbar via `AuthContext`). Kein neuer API-Call.

**Rationale:** `is_parent` ist bereits im JWT — kein Extra-Request nötig.

## Risks / Trade-offs

- **SQLite IN-Clause mit vielen IDs** → Bei sehr langen Zeiträumen (Vergangenheit eingeblendet: bis 365 Tage) entstehen ~100–150 Session-IDs im IN-Clause. SQLite unterstützt bis 32.766 Parameter — kein Problem.
- **`present`-Feld für Nicht-Trainer immer null** → Könnte verwirrend sein, wenn das Frontend versehentlich die Checkbox trotzdem rendert. Mitigierung: Frontend prüft `isTrainer && isPast` vor dem Rendern der Checkbox-Spalte (bereits so).
- **Spieler sieht Namen aller Kader-Mitglieder** → Gewolltes Verhalten (Mannschaftsübersicht). Nur Kommentare sind eingeschränkt.
