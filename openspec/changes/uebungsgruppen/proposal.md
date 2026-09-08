## Why

Für Torwart-, Athletik- oder Sichtungstraining gibt es heute kein passendes Gefäß. Wer
so etwas ansetzen will, muss einen **Kader** anlegen — und bekommt damit zwangsläufig
Altersklasse, Geschlecht, Jahrgang, laufende Teamnummer, einen `teams`-Zwilling und in
dessen Gefolge Spiele, Dienste, Mannschaftskasse, Strafen, Videos, Ordner-Principals und
den erweiterten Kader. Nichts davon ist gemeint. Gemeint ist: *ein Name, ein paar Leute,
ein Trainer, Termine.*

Der Bestandsmechanismus aus Migration `034` (`training_group_categories`, „Förderkader"/
„Perspektivkader") löst das über die `age_class`-Achse — also über genau die Eigenschaft,
die eine solche Gruppe **nicht** hat. Er bleibt unverändert bestehen (siehe
`design.md — Entscheidung 5`), löst das Problem aber nicht.

Der Befund, der diesen Change klein macht: **`training_sessions.team_id` wird im
Trainings-Paket nirgends als Team benutzt.** Alle 63 Vorkommen in
`internal/trainings/handler.go` haben dieselbe Form —
`k.team_id = ts.team_id AND k.season_id = ts.season_id` — und dienen ausschließlich dazu,
den besitzenden Kader zu finden. Dieser ist eindeutig (`kader` trägt weder `type` noch
`is_active`, `UNIQUE(season_id, age_class, gender, team_number)` steht, `team_id` wird aus
genau diesen Spalten abgeleitet). `team_id` ist dort ein *Handle*, kein Team.

## What Changes

- **Eine Übungsgruppe ist eine `kader`-Zeile ohne `teams`-Zwilling.** Neue Spalten
  `kader.kind` (`'team'` | `'practice'`) und `kader.name`; `age_class` und `gender` werden
  nullable. Bei `kind='practice'` gilt: `age_class`, `gender`, `dedicated_birth_year` und
  `team_id` sind **NULL**, `name` ist gesetzt. Ein CHECK erzwingt das als
  Alles-oder-nichts.
- **`training_sessions` und `training_series` bekommen `kader_id` als echten Besitzer**
  (NOT NULL, `ON DELETE RESTRICT`, Backfill über `(team_id, season_id)`). `team_id` bleibt
  erhalten, wird aber **nullable** und damit zur Projektion: gesetzt = gehört einer
  Mannschaft, NULL = gehört einer Übungsgruppe.
- **Löschschutz für Trainingshistorie.** Anders als `teams` (die nie gelöscht, nur über
  `is_active` stillgelegt werden) sind Kader löschbar. `DELETE /api/kader/{id}` und
  `DELETE /api/practice-groups/{id}` lehnen deshalb mit HTTP 409 ab, solange Trainings am
  Kader hängen — dieselbe Form wie die bestehende Mitglieder-Guard.
- **`CopyFromSeason` überspringt Übungsgruppen.** Der Saisonkopierer keyt auf
  `age_class|gender` und ruft `ensureTeam` — bei zwei NULLs kollabierten alle
  Übungsgruppen auf denselben Schlüssel und es entstünde ein Müll-Team.
- **`internal/trainings` stellt die 63 Joins auf `kader_id` um.** Das ist eine
  Vereinfachung, keine Erweiterung: aus zwei Bedingungen wird eine, aus
  `player_memberships` wird `kader_members`. RSVP, Anwesenheit, Serien, Abmeldungen und
  Erinnerungen funktionieren dadurch für beide Varianten über **einen** Pfad.
- **Neue Routen `/api/practice-groups`** für Anlage, Umbenennung, Mitglieder und Trainer.
  Die Kader-Maske wird als Oberfläche wiederverwendet, ohne Altersklasse, Geschlecht,
  Jahrgang und erweiterten Kader.
- **Chat:** `GET /api/chat/team-groups` liefert Übungsgruppen mit aus, eine neue
  Auflöse-Route `GET /api/chat/practice-groups/{id}/{kind}/members` gibt ihre Mitglieder.
- **Ein Gate:** `PUT /api/kader/{id}` lehnt `extended_members_*` bei `kind='practice'` mit
  HTTP 409 ab. Es ist das **einzige** nötige Gate — siehe unten.
- **Begriffe in der UI:** die neue Entität heißt **„Übungsgruppe"**. Die bestehende
  `optgroup` „Trainingsgruppen" in `AdminKaderPage.tsx` (zwei Fundstellen) wird zu
  **„Sonderkader"** umbeschriftet. Reine Label-Änderung, keine Migration. Das Wort
  „Trainingsgruppe" ist danach in der Oberfläche nicht mehr belegt.

## Was *nicht* gebaut werden muss

Der Ausschluss ist strukturell, nicht als Filter implementiert. Ohne `teams`-Zeile fällt
alles Folgende von selbst weg — **null Zeilen Code, kein Guard, kein Arch-Test:**

| Fläche | warum |
|---|---|
| Strafen, Kasse, Aufgaben, Strafenwarte, Kassenwarte (22 Routen) | adressieren `/api/teams/{id}/…` und lösen mit `WHERE k.team_id = ?` auf (`teams/access.go:27`, `penalties.go:82`) — ohne `teams`-Zeile nicht adressierbar |
| Spiele, Dienste, Mitfahrten, Aufstellung, Bewirtungsrotation, H4A | brauchen einen `game_teams`-Eintrag |
| Videos, Ordner-Principals (`team`/`team_parents`), Mitgliedsanträge | hängen an `teams.id` |
| Anwesenheits-**Statistik**, Trainingstagebuch-Statistik | `internal/attendance` liest über `ts.team_id` → NULL → keine Zeilen |
| iCal-Feed, Abwesenheitskalender, Dashboard-Kacheln | dieselben 28 externen `team_id`-Referenzen |
| `player_memberships`, `team_memberships`, `trainer_memberships`, `user_accessible_teams` | tragen **schon heute** `WHERE k.team_id IS NOT NULL` |

Die Trennung fällt exakt auf die Paketgrenze: Anwesenheit **erfassen** liegt in
`internal/trainings` (folgt `kader_id`, ist drin), Anwesenheit **auswerten** in
`internal/attendance` (folgt `team_id`, ist draußen).

## Capabilities

### Added Capabilities

- `uebungsgruppen`: Anlage und Pflege benannter Übungsgruppen ohne Altersklasse,
  Geschlecht, Jahrgang und `teams`-Zwilling; Mitglieder und Trainer; strukturelle
  Abgrenzung gegen alle team-gebundenen Flächen.

### Modified Capabilities

- `trainings`: Besitzer eines Trainings ist der Kader (`kader_id`), nicht mehr das Team.
  Trainings, Serien, RSVP, Anwesenheit und Erinnerungen gelten für Mannschaften **und**
  Übungsgruppen.
- `chat-team-groups`: Standard-Gruppen umfassen Übungsgruppen.

## Impact

**Migrationen**

- `057_uebungsgruppen` — `kader`-Rebuild (`age_class`/`gender` nullable, `+kind`, `+name`,
  CHECK, Partial-Unique-Index auf `(season_id, name)`). Muster wie `018`/`034`, mit
  `PRAGMA legacy_alter_table=ON` wegen der vier Views.
- `058_trainings_kader_owner` — Rebuild von `training_sessions` und `training_series`:
  `+kader_id NOT NULL` (Backfill), `team_id NOT NULL → NULL`.

**Backend**

- `internal/trainings/handler.go` (63 Joins), `unavailabilities.go`
- `internal/practicegroups/` — neues Package (Handler, Zugriff)
- `internal/chat/team_groups.go`
- `internal/hub/audience.go` — `trainingTeams` (Zeile 80) findet über `team_id` nichts mehr,
  wenn NULL; SSE-Zielmenge muss über `kader_id` aufgelöst werden
- `internal/kader/handler.go` — Gate gegen `extended_members_*`, Trainings-Guard in
  `DeleteKader`
- `internal/kader/copy.go` — `CopyFromSeason` überspringt `kind='practice'`
- `internal/app/router.go`, `cmd/teamwerk/main.go`

**Frontend**

- `web/src/pages/UebungsgruppenPage.tsx` (neu), Route in `App.tsx`, Nav in `AppShell.tsx`
- `web/src/pages/AdminKaderPage.tsx` — zwei `optgroup`-Labels „Trainingsgruppen" →
  „Sonderkader"

**Unverändert**

Kein Bestandsdatensatz wird inhaltlich geändert. Förderkader und Perspektivkader bleiben
reguläre Kader mit `kind='team'`. Der Change ist rein additiv.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `POST /api/practice-groups` | `TestCreatePracticeGroup_Erfolg` | 201, Zeile mit `kind='practice'`, `team_id IS NULL`, `age_class IS NULL`, `gender IS NULL` |
| `POST /api/practice-groups` | `TestCreatePracticeGroup_KeinTeamZwilling` | nach der Anlage existiert **keine** neue `teams`-Zeile |
| `POST /api/practice-groups` | `TestCreatePracticeGroup_NameDoppeltInSaison` | 409 bei gleichem Namen in derselben Saison |
| `POST /api/practice-groups` | `TestCreatePracticeGroup_OhneVorstand` | 403 |
| `POST /api/practice-groups` | `TestCreatePracticeGroup_OhneAktiveSaison` | 400 |
| `PUT /api/practice-groups/{id}` | `TestUpdatePracticeGroup_MitgliederUndTrainer` | 200, Mitglieder/Trainer werden gesetzt, Broadcast `practice-groups` |
| `DELETE /api/practice-groups/{id}` | `TestDeletePracticeGroup_MitTrainingsAbgelehnt` | 409 mit `training_count`, Gruppe und Termine bleiben |
| `DELETE /api/practice-groups/{id}` | `TestDeletePracticeGroup_OhneTrainingsErfolg` | 204, Gruppe verschwindet |
| `DELETE /api/kader/{id}` | `TestDeleteKader_MitTrainingsAbgelehnt` | 409; ein leerer Altkader mit Trainingshistorie ist nicht löschbar |
| `POST /api/kader/copy-from-season` | `TestCopyFromSeason_UeberspringtUebungsgruppen` | keine neue `teams`-Zeile, keine `kind='practice'`-Zeile in der Zielsaison |
| `GET /api/practice-groups` | `TestListPracticeGroups_NurAktiveSaison` | Gruppen anderer Saisons fehlen |
| `PUT /api/kader/{id}` | `TestUpdateKader_ErweiterterKaderBeiUebungsgruppeAbgelehnt` | 409, `kader_extended_members` bleibt leer |
| `POST /api/training-sessions` | `TestCreateTraining_FuerUebungsgruppe` | 201, `kader_id` gesetzt, `team_id IS NULL` |
| `GET /api/training-sessions` | `TestListTrainings_MitgliedSiehtUebungsgruppenTermin` | Mitglied der Gruppe sieht den Termin |
| `GET /api/training-sessions` | `TestListTrainings_FremderSiehtUebungsgruppenTerminNicht` | Nicht-Mitglied sieht ihn nicht |
| `GET /api/training-sessions` | `TestListTrainings_ElternSehenUebungsgruppenTermin` | Elternteil über `family_links` sieht ihn |
| `GET /api/training-sessions` | `TestListTrainings_TrainerSiehtUebungsgruppenTermin` | Trainer der Gruppe sieht ihn |
| `POST /api/training-sessions/{id}/rsvp` | `TestRsvpUebungsgruppe_Erfolg` | 201/200, Antwort wird gespeichert |
| `POST /api/training-sessions/{id}/rsvp` | `TestRsvpUebungsgruppe_FremderAbgelehnt` | 403 |
| `PUT /api/training-sessions/{id}/attendance` | `TestAttendanceUebungsgruppe_TrainerDarf` | 200 |
| `GET /api/teams/{id}/attendance-stats` | `TestAttendanceStats_UebungsgruppeTaucthNichtAuf` | Übungsgruppen-Trainings fließen **nicht** in die Team-Statistik |
| `GET /api/chat/team-groups` | `TestListTeamGroups_EnthaeltUebungsgruppe` | Mitglied sieht die Gruppe mit `groupType='practice'` |
| `GET /api/chat/practice-groups/{id}/{kind}/members` | `TestResolvePracticeGroup_Mitglieder` | 200, Caller herausgefiltert |
| `GET /api/chat/practice-groups/{id}/{kind}/members` | `TestResolvePracticeGroup_Fremder` | 403 |
| `GET /api/calendar/feed` | `TestIcalFeed_OhneUebungsgruppe` | Übungsgruppen-Trainings erscheinen **nicht** im Feed |
| — (Migration) | `TestMigration058_BackfillKaderId` | jedes Bestands-Training erhält genau den Kader seines `(team_id, season_id)` |
| — (Arch) | `TestPracticeGroup_KeineTeamRoute` | keine Route unter `/api/teams/{id}/…` liefert für eine Übungsgruppe ein 2xx |

**Garantierte Invarianten:**

1. **`training_sessions.team_id IS NULL` bedeutet „gehört einer Übungsgruppe" — nicht
   „Datenfehler".** Diese NULL trägt den gesamten Ausschluss. Sie darf nie durch einen
   Backfill „repariert" werden.
2. **Eine Übungsgruppe hat nie eine `teams`-Zeile.** Alle team-gebundenen Flächen bleiben
   dadurch strukturell unerreichbar, ohne dass eine von ihnen etwas prüfen muss.
3. **Ein Training hat genau einen Besitzer.** `kader_id` ist NOT NULL; `team_id` ist die
   abgeleitete Projektion, nie die Quelle.
4. **Trainingshistorie überlebt jedes Löschen.** Kein Codepfad entfernt einen Kader,
   solange Trainings an ihm hängen — Anwesenheits- und RSVP-Daten vergangener Saisons sind
   dadurch gegen versehentliches Aufräumen geschützt.
