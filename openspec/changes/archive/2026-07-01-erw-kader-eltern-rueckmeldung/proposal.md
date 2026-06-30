## Why

Eltern können für Kinder im **erweiterten Kader** (`kader_extended_members`) auf `/termine` keine Rückmeldung (zu-/absagen) geben, sehen auf der Detailseite keine Rückmeldung, und das Team des Kindes fehlt im Teamfilter. Für Stammkader-Kinder funktioniert alles. Spieler selbst (eigener Account) sind über `erweiterter-kader-sichtbarkeit` / `erweiterter-kader-trainings-access` bereits abgedeckt — die **Eltern-Perspektive** wurde dabei übersehen. Ein abgesetzter Spieler ohne eigenen Account hängt damit komplett fest: niemand kann für ihn zu-/absagen.

Wurzel ist eine Inkonsistenz im Code: Drei Eltern-bezogene Abfragen berücksichtigen nur `kader_members`, während die kanonische View `user_accessible_teams` und die Trainer-Sichten (`GetAttendances`/`GetParticipants`) den erweiterten Kader längst einschließen.

## What Changes

- **Eltern-`children_rsvp` umfasst erw.-Kader-Kinder** (Symptom 1 + 2): `attachChildrenRSVPToSessions` (`internal/trainings/handler.go`) und `attachChildrenRSVPToGames` (`internal/games/handler.go`) bekommen einen UNION-Zweig auf `kader_extended_members` (analog zu `GetAttendances`). Damit erscheinen erw.-Kader-Kinder im Eltern-Termin und bekommen Zu-/Absagen-Buttons; ihr Status wird auf der Detailseite sichtbar.
- **Kein Auto-Confirm für erw. Kader** (deine Vorgabe): Die `rsvp_opt_out`-Auto-Zusage gilt weiterhin **nicht** für erw.-Kader-Kinder — sie müssen immer explizit zu-/absagen. Die Abfragen führen dazu ein `is_extended`-Flag mit, das das Auto-Confirm unterdrückt (konsistent mit `erweiterter-kader-sichtbarkeit` und `GetAttendances`).
- **Teamfilter enthält erw.-Kader-Teams für Eltern** (Symptom 3): Der Eltern-/Spieler-Zweig von `ListTeamsForUser` (`GET /api/teams`) wird von der View `team_memberships` (Stammkader + Trainer, **ohne** erw. Kader) auf `user_accessible_teams` (deckt Stamm-/erw. Kader **und** Eltern bereits ab) umgestellt.

Kein Schema-/Migrationsänderung — nur Query-Logik. Keine API-Vertragsänderung (`children_rsvp`-Struktur bleibt; ein bisher leeres Array wird befüllt).

## Capabilities

### New Capabilities

_Keine._

### Modified Capabilities

- `eltern-rsvp`: `children_rsvp` (Training **und** Spiel) MUSS auch Kinder umfassen, die nur über `kader_extended_members` im Team sind; für diese gilt kein `rsvp_opt_out`-Auto-Confirm (immer explizite Rückmeldung).
- `erweiterter-kader-sichtbarkeit`: `GET /api/teams` MUSS das erw.-Kader-Team auch für **Eltern** des abgesetzten Spielers zurückgeben (bisher nur für den Spieler selbst spezifiziert).

## Impact

- **Code:** `internal/trainings/handler.go` (`attachChildrenRSVPToSessions`), `internal/games/handler.go` (`attachChildrenRSVPToGames`, `ListTeamsForUser`). Ggf. `childRSVP`-Struct um internes `is_extended`-Flag (nur zur Auto-Confirm-Unterdrückung; muss nicht im JSON nach außen).
- **APIs:** `GET /api/training-sessions`, `GET /api/games/my` (Inhalt von `children_rsvp`), `GET /api/teams` (Eltern-Zweig). Keine Vertrags-/Breaking-Änderung.
- **Frontend:** keine Änderung nötig — `TerminePage.tsx` rendert Buttons je `children_rsvp`-Eintrag und die Filter-Chips je `/api/teams`-Eintrag; beide werden durch die Backend-Korrektur automatisch korrekt.
- **DB:** keine Migration. `team_memberships`-View bleibt unangetastet (breit genutzt → Risiko vermieden).
- **Tests:** neue Happy-/Fehlerfälle in `internal/trainings/handler_test.go`, `internal/games/handler_test.go`.

## Test-Anforderungen

| Route | Testname (Vorschlag) | Status / Erwartung | Garantierte Invariante |
|---|---|---|---|
| `GET /api/training-sessions` | `TestSessions_ParentExtendedChild_InChildrenRSVP` | 200, `children_rsvp` enthält erw.-Kader-Kind | Erw.-Kader-Kinder sind für Eltern sichtbar |
| `GET /api/training-sessions` | `TestSessions_ExtendedChild_NoAutoConfirm` | 200, `rsvp: null` (erw.) vs `confirmed` (Stamm) bei `rsvp_opt_out=1` | Kein Auto-Confirm für erw. Kader |
| `GET /api/training-sessions` | `TestSessions_ChildInBothKaders_SingleEntry` | 200, genau ein Eintrag | Dedup: Stammkader hat Vorrang |
| `GET /api/games/my` | `TestMyGames_ParentExtendedChild_InChildrenRSVP` | 200, `children_rsvp` enthält erw.-Kader-Kind | Erw.-Kader-Kinder sind für Eltern sichtbar (Spiel) |
| `GET /api/games/my` | `TestMyGames_ExtendedChild_NoAutoConfirm` | 200, `rsvp: null` bei `rsvp_opt_out=1` | Kein Auto-Confirm für erw. Kader (Spiel) |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_ParentForExtendedChild_OK` | 204, `training_responses`-Zeile | Eltern können für erw.-Kader-Kind antworten |
| `POST /api/games/{id}/respond` | `TestGameRespond_ParentForExtendedChild_OK` | 204, `game_responses`-Zeile | Eltern können für erw.-Kader-Kind antworten (Spiel) |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_ParentForUnlinkedChild_Forbidden` | 403 | `parentHasChild` schützt fremde Kinder |
| `GET /api/teams` | `TestTeams_ParentExtendedChild_TeamListed` | 200, Team enthalten | Teamfilter zeigt erw.-Kader-Team für Eltern |
| `GET /api/teams` | `TestTeams_ParentNoKader_TeamNotListed` | 200, Team **nicht** enthalten | Keine Über-Sichtbarkeit |
