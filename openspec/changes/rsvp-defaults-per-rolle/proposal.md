## Why

Die RSVP-Voreinstellung eines Termins ist heute ein einziger Boolean (`rsvp_opt_out`) und greift ausschließlich für **Stammkader-Spieler**. Für den **Erweiterten Kader** gibt es keine Voreinstellung — er muss immer aktiv zusagen. Zusätzlich kennt das System nur zwei Modi (Auto-Confirm oder „keine Antwort"); die Voreinstellung **„standardmäßig abgesagt"** existiert nicht, obwohl sie z. B. für lockere Zusatztermine oder ferienbedingte Trainings sinnvoll wäre. Und die UI-Beschriftung (`(Opt-Out)`) ist Fachjargon.

Trainer sind nicht betroffen: Deren Verhalten wurde zuletzt (Change `termine-trainer-rsvp`) auf hart-verdrahtetes „default confirmed" festgelegt und bleibt so.

## What Changes

- **Zwei orthogonale Voreinstellungen pro Termin (Session/Serie/Spiel)** statt einer:
  - `rsvp_default_players` — gilt für Stammkader-Spieler
  - `rsvp_default_extended` — gilt für den Erweiterten Kader
- Jede der beiden Voreinstellungen kennt **drei Modi**:
  - `confirmed` — „standardmäßig zugesagt"
  - `declined` — „standardmäßig abgesagt" (**neu**, existiert heute nicht)
  - `none` — „keine automatische Rückmeldung"
- **UI-Beschriftungen ohne Fachjargon**: „Standardmäßig zugesagt" / „Standardmäßig abgesagt" / „Keine automatische Rückmeldung". Zwei Radio-Gruppen im Termin-Bearbeiten-Modal, eine für Stammkader-Spieler, eine für Erweiterten Kader. Der bisherige Text „Alle Spieler standardmäßig zugesagt (Opt-Out)" entfällt.
- **Konfliktsperre in der UI**: Wenn eine der Voreinstellungen auf `declined` steht, wird die Checkbox „Begründung bei Absage erforderlich" deaktiviert (und umgekehrt). Grund: eine Default-Absage entsteht ohne Nutzerhandlung — es gibt keinen Grund zum Erfassen, die Kombination wäre widersprüchlich. Wird beides trotzdem im Backend eingesendet, antwortet der Server mit HTTP 400.
- **Header-Zähler** (`confirmed_count` / `declined_count` / `maybe_count`) beziehen Default-Werte ein: ein Mitglied ohne Response-Row erscheint im Zähler seiner Rolle-Voreinstellung. Konsistenz mit der Zeilen-Anzeige.
- **Trainer-Verhalten bleibt unverändert**: harter `confirmed`-Default, keine UI-Einstellung, keine neue Enum-Spalte. Trainer werden weiterhin nicht in Header-Zähler einbezogen.
- **Migration** ersetzt `rsvp_opt_out`:
  - `rsvp_opt_out = 1` → `rsvp_default_players = 'confirmed'`
  - `rsvp_opt_out = 0` → `rsvp_default_players = 'none'`
  - `rsvp_default_extended` überall `'none'` (= aktuelles Verhalten für Erweiterten Kader)
  - Danach wird `rsvp_opt_out` per Migration entfernt.

## Capabilities

### Modified Capabilities

- `training-rsvp`: Session/Serie tragen zwei Voreinstellungs-Enums statt `rsvp_opt_out`; die Route `PUT /api/training-sessions/{id}` und `PUT /api/training-series/{id}` akzeptieren `rsvp_default_players` / `rsvp_default_extended`; `GET /api/training-sessions/{id}/attendances` und `GET /api/training-sessions/{id}` liefern die neuen Felder; Header-Zähler bezieht Default-Werte ein.
- `game-rsvp`: `games` trägt die zwei Voreinstellungs-Enums; `POST /api/games` und `PUT /api/games/{id}` akzeptieren sie; `GET /api/games/{id}` und `GET /api/games/my` liefern das effektive `my_rsvp` aus Response ∪ Default; Header-Zähler bezieht Default-Werte ein.
- `termine-detail`: Detail-Tabelle beschriftet die Sektionen unverändert (Trainer/Spieler/Erweiterter Kader). Für Zeilen ohne Response-Row zeigt das UI den jeweiligen Default (Zugesagt/Abgesagt/keine Rückmeldung); die Anzeige-Logik unterscheidet virtuellen Default (grau/kursiv) von aktiver Antwort.

## Impact

- **Datenbank**: Neue Migration `018_rsvp_defaults_per_role.up.sql` (+`.down.sql`) auf `training_sessions`, `training_series`, `games`: zwei `TEXT CHECK (… IN ('confirmed','declined','none')) NOT NULL DEFAULT 'none'`-Spalten, Backfill aus `rsvp_opt_out`, dann `DROP COLUMN rsvp_opt_out`. SQLite braucht dafür je Tabelle ein „`CREATE TABLE …_new`, `INSERT INTO … SELECT`, `DROP TABLE …`, `ALTER TABLE …_new RENAME`"-Muster (analog `011_event_notes.up.sql`).
- **Backend**: `internal/trainings/handler.go` und `internal/games/handler.go` — Struct-Felder, Insert/Update-Payloads, Attendances-Query (Default-Zweig), Header-Zähler-Query (`LEFT JOIN` mit `COALESCE(response.status, session.default_for_this_role)`), Konflikt-Validierung (`declined` + `rsvp_require_reason=1` → HTTP 400). `insertSessions` kopiert die zwei neuen Felder aus der Serie in die Session.
- **Frontend**: `web/src/components/TrainingEditModal.tsx`, `web/src/components/GameEditModal.tsx`, `web/src/pages/AdminTrainingsPage.tsx` — Checkbox durch zwei Radio-Gruppen ersetzen, Konflikt-Sperre implementieren. `web/src/pages/TermineDetailPage.tsx`, `web/src/pages/TerminePage.tsx`, `web/src/pages/KalenderPage.tsx` — TypeScript-Typen um die neuen Felder, virtuelle Default-Anzeige (grau/kursiv) für Zeilen ohne Response-Row. Alle bestehenden `rsvp_opt_out`-Referenzen im Frontend entfernen.
- **SSE**: Bestehendes `trainings`- und `games`-Event deckt Änderungen der Voreinstellungen ab; kein neues Event.
- **Berechtigungen**: Unverändert — Voreinstellung darf ändern, wer auch die Session/das Spiel bearbeiten darf (Vorstand + Trainer/sportliche Leitung).
- **Deploy-Reihenfolge**: Migration muss vor Rollout des neuen Backends laufen, sonst crashen INSERT/SELECT mit unbekannten Spalten. Standard-Deploy-Flow (`make deploy` → `migrate up` → systemd-restart) deckt das ab.
