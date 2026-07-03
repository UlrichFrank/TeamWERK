## Context

Die Termin-Detailseite `TermineDetailPage.tsx` bedient Trainings **und** Spiele. Die Backend-Handler `internal/trainings/handler.go` und `internal/games/handler.go` bilden dazu je eine `GET /…/attendances`-Route mit einem `UNION` aus Stammkader (`player_memberships`) und erweitertem Kader (`kader_extended_members`). Trainer eines Kaders (`kader_trainers`) fehlen in dieser Query.

Die RSVP-Routen (`POST …/respond`) prüfen heute Ownership: Selbst-Antwort (Spieler) oder Kind-Antwort (Eltern via `family_links`) oder — für den Trainer als Erfasser fremder Antworten — Kader-Trainerschaft. Der Trainer kann heute für andere antworten, aber nicht **für sich selbst als Teilnehmer** eingetragen werden, weil er nicht Teil der Kaderliste ist.

Die Sektionsstruktur in `TermineDetailPage.tsx:569` ist bereits generisch (`TableSection[]`) — der Umbau ist deklarativ, kein Refactoring.

## Goals / Non-Goals

**Goals:**
- Trainer erhalten Rückmelde-Fähigkeit in Trainings + Spielen mit Opt-out-Default (implizit anwesend).
- Termin-Detailtabelle bekommt drei benannte Sektionen: Trainer / Spieler / Erweiterter Kader (in dieser Reihenfolge).
- Anwesend- und Aufstellung-Zelle werden in Trainer-Zeilen strukturell weggelassen (leere `<td>`, kein Platzhalter).
- Header-Zählungen (`confirmed_count`/`declined_count`/`maybe_count`) bleiben spieler-orientiert.
- Symmetrische Umsetzung für Trainings und Spiele (dieselbe UI-Datei, analoge Handler).

**Non-Goals:**
- Kein Attendance-Tracking für Trainer (Tabelle `training_attendances` / `game_attendances` bleibt unangetastet).
- Kein neues Schema, keine Migration.
- Kein Spielertrainer-Fall (per Anforderung nicht existent).
- Keine Änderung am `groupByTeam`-Zweig für generische Multi-Team-Events — Trainer erscheinen pro Team-Sektion innerhalb des jeweiligen Team-Blocks (später ausbaubar).
- Keine neuen Push-/Mail-Benachrichtigungen an Trainer für RSVP-Erinnerungen.

## Decisions

### Trainer-Semantik als eigene Capability `trainer-rsvp`

Trainer haben eine **andere** RSVP-Semantik als Spieler (Opt-out-Default, kein Attendance, nicht im Zähler). Das ist keine reine Erweiterung der Spieler-Regeln, sondern ein eigenständiges Verhalten.

- **Alternative**: Alles unter `training-rsvp` / `game-rsvp` bündeln.
- **Warum abgelehnt**: Die Sonderregeln würden die bestehenden Requirements verwässern; Test- und Suchbarkeit leidet, wenn "Trainer-Default confirmed" in einer Requirement steht, die "Spieler brauchen expliziten RSVP" heißt. Analog wurde `eltern-rsvp` als eigene Capability geführt.

### `is_trainer`-Flag in der Attendances-Response

Backend liefert im `attendanceItem` ein neues Feld `is_trainer bool`. Frontend nutzt es sowohl für die Sektions-Zuweisung als auch für das bedingte Rendering der Anwesenheit-/Aufstellung-Zelle.

- **Alternative**: Rolle über ein `role`-Enum (`player | trainer | extended`).
- **Warum abgelehnt**: `is_extended` existiert schon als bool-Flag; ein zweites bool ist konsistent und minimalinvasiv. Enum würde bestehende API brechen.

### Default-confirmed für Trainer im Query-Result-Loop

Analog zum existierenden `rsvpOptOut`-Zweig (`handler.go:1365`): wenn `rsvp.Valid == false` und `is_trainer == true` → `RSVPStatus = "confirmed"` setzen. **Keine INSERT-Row** in `training_responses` — der Default ist rein virtuell und wird bei jedem GET neu berechnet.

- **Alternative**: Bei Kader-Trainer-Anlage automatisch eine `confirmed`-Row anlegen.
- **Warum abgelehnt**: Verlagert Semantik in ein anderes Domänen-Package (Kader), erzeugt persistenten State ohne Nutzen und muss bei Trainer-Entfernung/Session-Neuanlage synchronisiert werden. Virtueller Default ist stateless und günstiger.

### Header-Counter separieren

Die drei Zähler (`confirmed_count`, `declined_count`, `maybe_count`) im Session/Game-Response werden über eine separate Query berechnet (heute Teil des `GetSession`-Handlers). Diese Query MUSS um einen `WHERE`-Filter erweitert werden, der Trainer ausschließt: `AND member_id NOT IN (SELECT member_id FROM kader_trainers WHERE kader_id = ?)`.

- **Alternative**: Zähler in der Aggregation im Frontend berechnen.
- **Warum abgelehnt**: Backend liefert die Zahl heute schon und ist die single source of truth. Doppelte Berechnung würde driften.

### RSVP-Ownership: Trainer für sich selbst und für andere Trainer

Trainer-Selbstantwort: `member_id` im Payload entspricht `claims.MemberID` oder wird über `kader_trainers.member_id = claims.MemberID` verifiziert.

Trainer für andere Trainer: derselbe Kader-Bezug wie bei Spielern (Trainer darf für alle Kader-Mitglieder antworten). Kein zusätzlicher Code-Pfad — nur die bestehende Trainer-Berechtigung greift auch für die neuen Trainer-Zeilen.

### UI: leere Zellen statt colspan

Anwesend- und Aufstellung-`<td>` werden in Trainer-Zeilen einfach nicht befüllt (leere `<td/>`). Layout-Alternative wäre `colspan={n}` auf der RSVP-Zelle — bricht aber die vertikale Ausrichtung der Rückmeldung-Icons zwischen Trainer- und Spieler-Sektion, was den Blick stört.

## Risks / Trade-offs

**[Alt-RSVPs verwaister Ex-Trainer]** — Wird ein Member aus `kader_trainers` entfernt, bleiben seine `training_responses`/`game_responses`-Rows bestehen. → **Mitigation**: Keine, akzeptiert. Bei nächster Antwort wird upsertet; kein Sichtbarkeits-Problem, weil der Member dann weder als Trainer noch als Spieler in der UNION erscheint.

**[Doppelte Antworten in Session-Detail]** — Wenn `rsvp_opt_out=1` (Session-Setting) und Trainer aktiv `confirmed` setzt, existiert einerseits ein virtueller Default, andererseits eine echte Row. Beide zeigen `confirmed` — kein sichtbarer Unterschied, aber im Datenmodell inkonsistent. → **Mitigation**: Aktives POST erzeugt eine Row (Upsert), der virtuelle Default greift nur bei Fehlen der Row. Konsistent, wenn auch redundant.

**[Trainer-Absage-Reason bei `rsvp_require_reason=1` durchsetzen]** — Die Server-Validierung muss die bestehende Regel auch auf Trainer-Rows anwenden. → **Mitigation**: Regel gilt heute schon per Payload-Validierung unabhängig von der Rolle des Absenders; kein Extra-Code, nur Test-Deckung.

**[Zähler-Query wird komplexer]** — Ein zusätzlicher `NOT IN`-Subquery auf `kader_trainers` in einer bereits nicht ganz kurzen SELECT-Aggregation. → **Mitigation**: SQLite plant das mit vorhandenem PK-Index `(kader_id, member_id)` günstig; erwartete Trainer-Zahl pro Kader ist ≤5. Kein Performance-Risiko.

**[Frontend-Sektion für generische Multi-Team-Events]** — Der `groupByTeam`-Zweig sortiert Trainer heute nicht separat innerhalb der Team-Sektion. → **Mitigation**: Für v1 werden Trainer im Multi-Team-Fall ans Ende ihrer Team-Sektion sortiert (natürliche Sortierung nach Vorname bleibt). Falls das später unklar wirkt, in Folge-Change lösen.

## Migration Plan

1. Backend-Handler + Query anpassen (Trainings + Games), Handler-Tests erweitern.
2. Frontend-Anpassung `TermineDetailPage.tsx` mit `is_trainer`-Feld und drei Sektionen.
3. Deploy als Standard-Release (kein Feature-Flag). Bestehende RSVP-Antworten von Trainern (falls durch UI-Bug oder Direktanlage existent) bleiben gültig; Neu-Antworten laufen durch dieselbe Route.
4. Kein Rollback-Risiko: Umkehr ist reines Code-Revert; keine Migration, keine Daten-Änderung.

## Open Questions

Keine offen — die Klärungen aus Explore-Mode sind in Decisions eingeflossen.
