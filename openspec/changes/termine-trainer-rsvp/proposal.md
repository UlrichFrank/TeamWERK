## Why

Trainer können sich in der Termin-Detailansicht (Trainings + Spiele) heute weder eintragen noch abmelden — sie sind nur Betrachter. Fällt der Trainer krank aus, gibt es keinen offiziellen Weg, das dem Team sichtbar zu machen. Gleichzeitig sind Trainer im Regelfall dabei; ein Opt-in-Zwang wäre reine Klick-Arbeit.

## What Changes

- Trainer eines Kaders werden in der Termin-Detail-Tabelle als eigene Sektion **oberhalb** der Spieler geführt (Reihenfolge: **Trainer → Spieler → Erweiterter Kader**), jede Sektion durch eine `border-t-2` abgetrennt (wie heute der erweiterte Kader). Die bisher namenlose Stammkader-Sektion trägt jetzt den Titel „Spieler".
- Trainer können auf Termin-RSVP zu- und absagen (`POST /api/training-sessions/{id}/respond`, `POST /api/games/{id}/respond`). Rein technisch bestehende Routen — neu ist, dass ein Trainer-`member_id` akzeptiert wird.
- **Trainer-Default = confirmed**, unabhängig vom Session-Setting `rsvp_opt_out`. Ohne explizite Reaktion gilt der Trainer als anwesend; er muss nur bei Abwesenheit reagieren.
- Für Trainer wird **keine Anwesenheit** getrackt: `training_attendances` / `game_attendances` bleibt spieler-only, die Anwesend-Spalte (und die Aufstellung-Spalte bei Spielen) wird in Trainer-Zeilen weggelassen (leere `<td>`, kein Platzhalter-Strich).
- Trainer-Absage folgt dem Session-Setting `rsvp_require_reason`: bei aktiviertem Setting ist ein Grund Pflicht, sonst optional — identisch zur bestehenden Spieler-Regel.
- Trainer werden **nicht** in die Zusagen-Zähler im Header (`confirmed_count`/`declined_count`/`maybe_count`) einbezogen. Diese bleiben spieler-orientiert.
- Trainer sind Mitglieder mit Vereinsfunktion `trainer` und stehen über `kader_trainers` einem Kader zugeordnet. Es gibt keine Spielertrainer (Person ist entweder Trainer *oder* Spieler eines Kaders) — daher keine Dedup-Logik nötig.

## Capabilities

### New Capabilities
- `trainer-rsvp`: Trainer-spezifische RSVP-Semantik für Termine (Opt-out-Default, keine Anwesenheitserfassung, kein Header-Zähler, Absage-Reason folgt Session-Setting).

### Modified Capabilities
- `termine-detail`: Termin-Detail-Tabelle bekommt eine Trainer-Sektion oberhalb der Spieler; Stammkader-Sektion erhält Titel „Spieler"; Anwesend- und Aufstellung-Zelle für Trainer-Zeilen weggelassen.
- `training-rsvp`: `POST /api/training-sessions/{id}/respond` akzeptiert `member_id` eines Trainers des zugeordneten Kaders; `GET /api/training-sessions/{id}/attendances` liefert Trainer mit `is_trainer=true` und default-`confirmed`.
- `game-rsvp`: `POST /api/games/{id}/respond` akzeptiert `member_id` eines Trainers eines am Spiel beteiligten Teams; `GET /api/games/{id}/attendances` liefert Trainer analog.

## Impact

- **Backend**: `internal/trainings/handler.go` (Attendances-Query + Respond-Route), `internal/games/handler.go` (analog). UNION-Erweiterung um `kader_trainers`-Zweig mit `is_trainer=1`. Default-confirmed-Zweig im Result-Loop. Header-Zähler-Query separat um Trainer bereinigt.
- **Frontend**: `web/src/pages/TermineDetailPage.tsx` (Sektionen, Zeilenrendering, leere Zellen für Trainer). `AttendanceItem`-Typ um `is_trainer` erweitert.
- **Datenbank**: **Keine Migration.** `kader_trainers` und `training_responses`/`game_responses` existieren bereits mit passendem Schema.
- **SSE**: Broadcast-Events `trainings`/`games` decken RSVP-Änderungen weiterhin ab; kein neues Event nötig.
- **Berechtigungen**: RSVP-Route prüft heute Ownership (`member_id == claims.MemberID` oder `is_parent` mit passendem `family_link`). Für Trainer muss die Ownership-Prüfung erweitert werden: Trainer darf für sich selbst antworten (Selbst-`member_id`) und — analog zur bestehenden Spieler-Regel — für andere Trainer desselben Kaders (Trainer haben heute schon Erfassungsrechte für alle Kader-Mitglieder).
