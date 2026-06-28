## Context

Heute werden Trainings-Anwesenheiten in `training_attendances` (present 0/1) erfasst, aber nirgends aggregiert. Spiel-Anwesenheit ist überhaupt nicht modelliert — es existiert nur `game_responses` (RSVP) und `game_lineup` (Nominierung). Trainer haben deshalb keine Übersicht, ob ein Spieler zuverlässig erscheint; Spieler/Eltern haben keinen Einblick in die eigene Quote.

Diese Spec führt **(a)** eine post-hoc Spiel-Anwesenheitserfassung analog zu Trainings ein, **(b)** ein einheitliches Drei-Säulen-Statistik-Modell (anwesend/entschuldigt/fehlt) über Trainings und Spiele und **(c)** einen täglichen Reminder-Mechanismus, der Trainer zur Pflege der Daten anhält.

Bestehende Bausteine, auf die wir aufbauen:
- `training_attendances` (Tabelle + Routen)
- `training_responses` / `game_responses` mit `absence_id` (auto-decline durch `member_absences`)
- `kader` / `kader_members` / `kader_extended_members` (Saison-Bezug)
- `internal/scheduler/` (idempotenter Cron-Wrapper) und `notification_log` (Idempotenz-Tabelle)
- `internal/push/` mit `SendToUsers` (VAPID-Web-Push)
- `internal/hub/` (SSE-Broadcasts) und `useLiveUpdates`-Hook

## Goals / Non-Goals

**Goals**

- Konsistentes, einfach erklärbares Statistik-Modell mit genau drei Säulen.
- Saubere Trennung Trainer-/SL-Sicht (Team) vs. Spieler-/Eltern-Sicht (eine Person).
- Pflicht-Charakter für Trainer-Erfassung durch tägliche aggregierte Push.
- Additives Datenmodell (keine breaking changes, keine Backfills).
- Live-Updates wie überall sonst im Projekt.

**Non-Goals**

- Keine "no-show"-Sonderkategorie (RSVP=confirmed + present=0). Daten sind vorhanden, kann später nachgezogen werden.
- Keine spielerseitige Korrektur der Anwesenheit ("ich war doch da") — Trainer ist die Wahrheit.
- Keine historischen Statistiken über mehrere Saisons hinweg. Eine Statistik bezieht sich immer auf **eine** Saison (Default: aktive).
- Keine Statistik für Vorstand/Kassierer/andere Rollen in diesem Change — wird bei Bedarf später freigeschaltet.
- Keine Schemaänderung an `kader_members` (kein `added_at`). Spieler, die mid-season eintreten, starten mit einer strukturell niedrigen Quote — bewusst akzeptiert (Option A).
- Keine Quoten-Definition als Hauptkennzahl: die drei Säulen sind die Wahrheit, die Quote ist nur eine abgeleitete Hilfsanzeige.

## Decisions

### D1: Drei-Säulen-Klassifikation pro Termin und Spieler

Pro (Termin × Mitglied) wird **maximal** eine Säule gezählt. Reihenfolge der Auswertung:

```
1. attendance.present = 1                           → ANWESEND
2. attendance.present = 0                           → FEHLT
3. response.status = 'declined' AND absence_id ≠ ∅  → ENTSCHULDIGT
4. sonst                                            → IGNORIERT (Datenloch)
```

Ist beides erfasst (attendance + auto-decline-response durch nachträgliche Abwesenheit), gewinnt die **explizite Trainer-Erfassung**. Cancelled Sessions/Games (`training_sessions.status='cancelled'`, `games.status='cancelled'`) werden komplett aus der Bezugsmenge entfernt.

**Quote (nur abgeleitete Anzeige):** `anwesend / (anwesend + fehlt)` — entschuldigte und ignorierte zählen weder im Zähler noch im Nenner. Begründung: ein Spieler mit 2 Trainings (1 anwesend, 1 Urlaub) soll als 100% angezeigt werden, nicht 50%.

**Alternative verworfen:** "fehlt = alles ohne Erfassung". Bestraft Spieler für Trainer-Faulheit.

### D2: Datenloch wird ignoriert (Variante 4 aus der Explore-Phase)

Vergangene Termine ohne `attendance`-Eintrag und ohne auto-decline werden weder als anwesend noch als fehlt gezählt — sie tauchen in der Statistik nicht auf. **Stattdessen:** Trainer-UI zeigt einen Banner "N offene Erfassungen", und ein täglicher Reminder-Job (siehe D6) sendet eine Push. Datenqualität wird über Workflow erzwungen, nicht über die Bestrafung von Spielern.

### D3: Stammkader vs. erweiterter Kader getrennt darstellen

Im Trainer-UI und in `GET /api/teams/{id}/attendance-stats`:
- Stammkader (`kader_members`) im Haupt-Ranking.
- Erweiterter Kader (`kader_extended_members`, abzüglich Spieler, die auch Stamm sind) als separater Block mit eigenem Mini-Durchschnitt.
- Beide Blöcke nutzen dieselben Säulen-Definitionen.

Begründung: Erweiterte Spieler nehmen sporadisch teil — eine gemeinsame Quote wäre verzerrend. Spieler in beiden Listen gelten als Stamm (analog `training-attendance`-Spec).

### D4: Saisonbezug — Option A (Saison-Start gilt für alle gleich)

Bezugsmenge der Termine = alle `training_sessions` + `games` der Teams, in denen das Mitglied Kader-Mitglied ist, mit `date BETWEEN season.start_date AND today()`. Spieler, die mid-season dem Kader hinzugefügt wurden, starten mit Termine-vor-Beitritt als "ignoriert" (weil dort weder `attendance` noch `response` für sie existiert) — strukturell niedrige Quote ist bewusst in Kauf genommen.

**Alternative verworfen:** Neue Spalte `kader_members.added_at` einführen + backfill. Vermeidet Schemaänderung; falls sich das Problem in der Praxis als störend erweist, kann es nachgezogen werden.

### D5: Schema — minimale additive Migration

```sql
-- 012_game_attendances.up.sql
CREATE TABLE game_attendances (
    id        INTEGER  PRIMARY KEY AUTOINCREMENT,
    game_id   INTEGER  NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    member_id INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    present   INTEGER  NOT NULL CHECK (present IN (0, 1)),
    noted_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (game_id, member_id)
);
CREATE INDEX idx_game_attendances_game ON game_attendances(game_id);
```

Strikt analog zu `training_attendances`. Kein `noted_by` (auch Trainings haben keinen) — der pflegende Trainer ist nicht audit-relevant. Down-Migration: `DROP TABLE game_attendances;`.

### D6: Reminder-Job — täglich, aggregiert, idempotent

- **Wo:** Neue Funktion im `internal/scheduler/`-Package, registriert in der Liste der täglichen Jobs. Trigger: existierender systemd-Cron `* * * * * /usr/local/bin/teamwerk-scheduler.sh`; der Job selbst entscheidet anhand der Uhrzeit, ob er heute schon gelaufen ist (Standard-Pattern im Repo).
- **Tageszeit:** 19:00 lokal (typischerweise nach Trainingszeit).
- **Adressaten:** Alle `users.id`, die via `kader_trainers` + `members` + `users` als Trainer eines Teams gelten und in einem Kader der **aktiven Saison** sind.
- **Bezugsmenge offener Termine pro Trainer:** vergangene (`date < today()` oder `date = today() AND end_time < now()`), nicht cancelled `training_sessions`/`games` seiner Teams in der aktiven Saison, für die noch **keine** `attendance`-Zeile existiert.
- **Stop-Bedingung pro Termin:** Sobald irgendein Trainer dieses Teams gespeichert hat (mindestens 1 `attendance`-Row für den Termin), gilt der Termin als erledigt — die nächste Push enthält ihn nicht mehr. Das ist die natürliche Folge des "fehlt keine Attendance"-Filters.
- **Idempotenz:** Vor dem Senden Zeile `(user_id, kind='attendance-reminder', context=YYYY-MM-DD)` in `notification_log` einfügen — `INSERT OR IGNORE`; nur wenn die Zeile neu angelegt wurde, wird gesendet. So bleibt es bei max. 1 Push/Trainer/Tag, selbst wenn der Job mehrfach startet.
- **Cut-off:** Termine außerhalb `seasons WHERE is_active=1` werden nicht berücksichtigt — wenn die aktive Saison vorbei ist (oder es keine gibt), gibt es keine Push.
- **Push-Inhalt:** Title `"Anwesenheiten fehlen"`, Body `"3 offene Erfassungen: D-Jugend Di 14.10., Spiel HSC 18.10., …"` (max. 3 Termine in den Body, der Rest impliziert). Tap-Ziel: `/team/{firstTeamId}/anwesenheit` (bei mehreren Teams wird das erste mit offenen Erfassungen genommen).
- **Versand:** Wie bei anderen Push-Sendern als `go push.SendToUsers(...)` (nicht blockierend).

### D7: Authz-Matrix

| Endpoint | Public | Auth | Spieler/Eltern | Trainer/SL | Vorstand | Admin |
|---|---|---|---|---|---|---|
| `POST /api/games/{id}/attendances` | ✗ | ✗ | ✗ | Trainer für eigenes Team / SL überall | ✗ | ✓ |
| `GET /api/games/{id}/attendances` | ✗ | ✗ | ✗ | Trainer für eigenes Team / SL überall | ✗ | ✓ |
| `GET /api/teams/{id}/attendance-stats` | ✗ | ✗ | ✗ | Trainer eigenes Team / SL alle | ✗ | ✓ |
| `GET /api/teams/{id}/attendance-open` | ✗ | ✗ | ✗ | Trainer eigenes Team / SL alle | ✗ | ✓ |
| `GET /api/members/{id}/attendance-stats` | ✗ | ✗ | eigenes / Kinder via `family_links` | Trainer/SL der Teams des Members | ✗ | ✓ |

Vorstand/Kassierer bleiben außen vor — bewusst, kann später per `RequireClubFunction("vorstand", "sportliche_leitung", "trainer")` ergänzt werden.

### D8: Live-Updates

`POST /api/games/{id}/attendances` ruft `h.hub.Broadcast("attendance-changed")` auf (analog zu `attendance` für Trainings, falls vorhanden — sonst neuer Event-Name in `useLiveUpdates`-Konsumenten). Frontend-Seiten abonnieren mit `useLiveUpdates((event) => { if (event === 'attendance-changed') reload() })`.

### D9: Routing & Package-Struktur

- `POST/GET /api/games/{id}/attendances` werden direkt in `internal/games/handler.go` ergänzt (parallel zu Trainings-Attendances in `internal/trainings/`).
- Aggregations-Endpoints (`/api/teams/{id}/attendance-stats`, `/api/teams/{id}/attendance-open`, `/api/members/{id}/attendance-stats`) wandern in ein neues, dünnes `internal/attendance/`-Package — sie lesen aus mehreren Domänen (trainings + games + responses + absences + kader + members) und gehören in keine existierende Domäne sauber rein.
- Architektur-Test `internal/arch/arch_test.go` muss das neue Package klassifizieren (Composition-Layer; darf trainings/games/members lesen).
- Routen-Eintragung in `internal/app/router.go`:
  - Trainer-Tier: `POST/GET /api/games/{id}/attendances`, `GET /api/teams/{id}/attendance-open`, `GET /api/teams/{id}/attendance-stats` (Trainer + sportliche_leitung).
  - Authenticated-Tier: `GET /api/members/{id}/attendance-stats` (Authz-Check im Handler).

### D10: Frontend-Struktur

- `web/src/pages/TeamAnwesenheitPage.tsx` (Trainer / SL) — Route `/team/:id/anwesenheit`.
- `web/src/pages/ProfilAnwesenheitPage.tsx` (Spieler / Eltern) — Route `/profil/anwesenheit` oder als neuer Tab in der bestehenden Profil-Komponente (Entscheidung im Implementations-Task).
- Spiel-Detailseite (`/termine/spiel/:id`): neue Sektion "Anwesenheit" für Trainer (analog Training).
- Mobile: Tabellen als `MobileCard`-Layout, Touch-Targets `py-2.5`, brand-Tokens, `lucide-react`.
- Statistik-Säulen visualisieren: ein horizontaler Stacked-Bar (grün/gelb/rot) plus Zahlen.

## Risks / Trade-offs

- **Strukturell niedrige Quoten für Mid-Season-Beitritte (D4)** → Akzeptiert (Option A); ggf. später `kader_members.added_at` nachziehen.
- **Datenloch wird ignoriert (D2)** → Push-Reminder ist die einzige Garantie für Datenqualität. Wenn der Job hängt, leidet die Statistik still. Mitigation: scheduler-Heartbeat ist bereits via Better Stack instrumentiert; neuer Job sollte denselben Heartbeat-Mechanismus benutzen.
- **Eine Push/Trainer/Tag** kann bei Trainern mit vielen Teams länglich werden → max. 3 Termine im Body, Rest impliziert ("… und 4 weitere"); Vollliste im UI nach Tap.
- **`game_attendances` ohne `noted_by`** → keine Nachvollziehbarkeit, wer zuletzt eingetragen hat. Konsistent mit Trainings-Attendances; falls Audit nötig, kann später eine Spalte ergänzt werden.
- **`attendance-changed` triggert Reload an vielen Stellen** → vertretbar; existierende SSE-Konsumenten machen das ebenso, Network-Footprint klein (Stats-Endpoint ist günstig).
- **Race im Reminder-Job** (Trainer speichert, während Job läuft) → unkritisch: spätestens am nächsten Tag ist der Termin aus der Liste; kurzfristige falsche Erinnerung tolerabel.

## Migration Plan

1. Migration `012_game_attendances.up.sql/.down.sql` ausrollen (`make migrate-remote-up`).
2. Backend deployen — neue Routen sind sofort verfügbar, alte Pfade unverändert.
3. Frontend deployen — neue Seiten/Tabs werden sichtbar.
4. Scheduler-Job läuft beim nächsten Cron-Tick mit; vorher kein Daten-Cleanup nötig.
5. Rollback: Migration `012` herunter (`make migrate-remote-down 1`), Binary zurückrollen.

## Open Questions

- Soll `ProfilAnwesenheitPage` eine eigene Route bekommen oder als Tab in der bestehenden Profil-Seite leben? — Entscheidung im Implementations-Task; beides ist mit brand-Tokens und bestehender Profil-Tab-Struktur verträglich.
- Wenn ein Trainer Mitglied **mehrerer** Teams ist, soll die Push pro Team oder pro Trainer eine sein? — Festgelegt: **eine pro Trainer**, alle Teams aggregiert (siehe D6). Falls sich das in der Praxis als unübersichtlich erweist, später aufsplitten.
