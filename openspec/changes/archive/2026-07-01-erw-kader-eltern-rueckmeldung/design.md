## Context

Stamm- und erweiterter Kader sind zwei getrennte Tabellen (`kader_members`, `kader_extended_members`). Der erweiterte Kader wurde nachträglich eingeführt; mehrere Code-Pfade berücksichtigen ihn, einige nicht:

| Stelle | erw. Kader? |
|---|---|
| `user_accessible_teams` (View) | ✓ inkl. Spieler **und** Eltern |
| `GetAttendances` / `GetParticipants` (Trainer-Sicht) | ✓ Zeile mit `is_extended=1` |
| `attachChildrenRSVPToSessions` (Eltern, Training) | ✗ nur `kader_members` |
| `attachChildrenRSVPToGames` (Eltern, Spiel) | ✗ nur `kader_members` |
| `ListTeamsForUser` Eltern/Spieler-Zweig (`GET /api/teams`) | ✗ nutzt `team_memberships` |
| `team_memberships` (View) | ✗ `kader_members ∪ kader_trainers` |

Die `eltern-rsvp`-Capability deckt nur Stammkader-Kinder ab; die erw.-Kader-Capabilities decken nur den Spieler mit eigenem Account ab. Die Eltern-Perspektive auf erw.-Kader-Kinder fällt durchs Raster.

## Goals / Non-Goals

**Goals:**
- Eltern können für erw.-Kader-Kinder auf `/termine` (Liste + Detail) zu-/absagen — identisch zu Stammkader.
- Erw.-Kader-Kinder erhalten **kein** `rsvp_opt_out`-Auto-Confirm; sie müssen immer explizit antworten.
- Das erw.-Kader-Team erscheint im Teamfilter auch für Eltern.
- Korrektur ausschließlich Backend; Frontend rendert die korrigierten Daten unverändert korrekt.

**Non-Goals:**
- Keine Schema-/Migrationsänderung.
- Keine Änderung an der breit genutzten `team_memberships`-View.
- Keine Änderung am Respond-Handler-Berechtigungsmodell (`parentHasChild` über `family_links` bleibt; der Pfad funktioniert bereits, wird nur über die UI nun erreichbar).
- Keine fachliche Änderung für Stammkader.

## Decisions

**1. `children_rsvp`-Queries um UNION auf `kader_extended_members` erweitern, mit `is_extended`-Flag.**
Sowohl `attachChildrenRSVPToSessions` als auch `attachChildrenRSVPToGames` bekommen einen zweiten SELECT-Zweig (`JOIN kader_extended_members`), der per `NOT EXISTS (kader_members …)` Doppelungen ausschließt — exakt das Muster aus `GetAttendances` (`internal/trainings/handler.go:1235ff`). Jede Zeile trägt `0/1 AS is_extended`.

_Warum so:_ spiegelt die bereits erprobte, getestete Logik der Trainer-Sicht; minimal-invasiv; kein Schema-Eingriff.

**2. Auto-Confirm nur für Nicht-erw.-Kader.**
Die Auto-Zusage bei `rsvp_opt_out=1` (Status `null` → `confirmed`) wird nur angewendet, wenn `is_extended == 0`. Mirror der bestehenden Bedingung `… && !item.IsExtended` in `GetAttendances` und der Requirement „Kein Auto-Confirm für erweiterte Kader-Mitglieder" aus `erweiterter-kader-sichtbarkeit`.

_Alternative verworfen:_ erw. Kader ebenfalls auto-confirmen — widerspricht der ausdrücklichen fachlichen Vorgabe (abgesetzte Spieler sind nicht automatisch dabei).

**3. `ListTeamsForUser` Eltern/Spieler-Zweig auf `user_accessible_teams` umstellen — nicht die View ändern.**
Statt `JOIN team_memberships … EXISTS(member)/EXISTS(parent)` ein `WHERE t.id IN (SELECT team_id FROM user_accessible_teams WHERE user_id = ? AND season_id = <aktiv>)`. `user_accessible_teams` deckt Stamm-/erw. Kader und Eltern bereits ab (siehe `001_initial.up.sql:607ff`).

_Alternative verworfen:_ `team_memberships`-View um `kader_extended_members` erweitern. Sauberere Datenbasis, aber die View wird an vielen Stellen genutzt → unkalkulierbarer Blast-Radius, höherer Testbedarf. Lokale Korrektur in `ListTeamsForUser` ist risikoärmer und genau auf das Symptom gerichtet.

## Risks / Trade-offs

- **Risk:** Doppelzählung, wenn ein Kind sowohl im Stamm- als auch im erw. Kader desselben Teams steht → **Mitigation:** `NOT EXISTS (kader_members …)`-Guard im erw.-Zweig (wie bei `GetAttendances`), Stammkader gewinnt (inkl. Auto-Confirm).
- **Risk:** Inkonsistenz zwischen `team_memberships` (weiter ohne erw. Kader) und `user_accessible_teams` bleibt bestehen → **Mitigation:** bewusst akzeptiert; `team_memberships` bleibt der „reine Spieler-Roster" für andere Verwender. Im Design dokumentiert.
- **Risk:** Auto-Confirm-Regel an mehreren Stellen dupliziert (`GetAttendances`, beide `attachChildrenRSVP…`) → **Mitigation:** identisches Muster, durch Tests abgesichert; bewusst keine verfrühte Abstraktion.

## Migration Plan

Reiner Code-Change, keine DB-Migration. Deploy via `make deploy`. Rollback = vorheriges Binary; keine Datenänderung, daher gefahrlos reversibel.

## Open Questions

Keine offen — die fachliche Regel (voll gleichstellen außer Opt-out-Ausnahme) und die Teamfilter-Quelle (`user_accessible_teams`) sind entschieden.
