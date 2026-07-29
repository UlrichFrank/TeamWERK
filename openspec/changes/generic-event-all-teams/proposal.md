## Why

Auf `/kalender` kann ein reiner Trainer bei generischen Events („Sonstiges Event") nur seine
eigenen Mannschaften auswählen — beim Anlegen wie beim Bearbeiten. Genau das ist der Hauptfall
für generische Events (Vereinsfest, Trainingslager, Hallendienst-Aktion): mannschaftsübergreifend.
Die bestehende Spec `event-wizard` fordert dieses Verhalten bereits ("Trainer bei Sonstigem Event
→ alle aktiven Mannschaften wählbar (kein Filter)"), es ist aber nie so implementiert worden.

Beim Aufräumen fällt eine zweite, gegenläufige Lücke auf: `PUT /api/games/{id}` und
`DELETE /api/games/{id}` haben **überhaupt keine** Team-Scope-Prüfung. Ein reiner Trainer darf
heute jedes fremde Spiel umdatieren, umhängen oder löschen. Beide Punkte gehören in einen Change,
weil sie dieselbe Frage beantworten: *woran* hängt die Berechtigung eines Trainers — an der
Team-Auswahlliste (falscher Ort, heute) oder an einer Server-Prüfung pro Event (richtiger Ort).

## What Changes

- **Generische Events: alle Mannschaften wählbar.** Bei `event_type='generisch'` speisen sich die
  Mannschafts-Checkboxen aus der vereinsweiten Liste (`GET /api/teams/names`) statt aus der
  nutzergefilterten `GET /api/teams` — im Kalender-Wizard (Anlegen) **und** im `GameEditModal`
  (Bearbeiten). Heim-/Auswärtsspiele bleiben unverändert auf die eigenen Mannschaften beschränkt.
- **Backend-Scope-Prüfung auf `POST /api/games` wird typabhängig.** Die heutige Prüfung („alle
  `team_ids` müssen eigene Teams sein") gilt künftig nur noch für `heim`/`auswärts`. Bei
  `generisch` sind beliebige `team_ids` erlaubt — der Trainer muss aber mindestens eine **eigene**
  Mannschaft mit im Event haben (sonst legt er ein Event an, das er selbst nicht sieht, siehe
  `event-team-visibility`).
- **Neue Scope-Prüfung auf `PUT`/`DELETE /api/games/{id}`** (schließt die bestehende Lücke): Ein
  reiner Trainer darf ein Event nur mutieren, wenn er Trainer mindestens einer **aktuell
  beteiligten** Mannschaft ist → sonst HTTP 403. Für `heim`/`auswärts` müssen die neuen `team_ids`
  zusätzlich vollständig eigene Teams sein; bei `generisch` sind beliebige `team_ids` erlaubt,
  solange mindestens eine eigene Mannschaft im Ergebnis bleibt.
- **Kein neuer Endpoint, keine neue Rechte-Dimension.** `GET /api/teams` bleibt unverändert
  nutzergefiltert (Kalender-Filter-Dropdown, Trainings, Video-Upload-Ziele hängen daran).
  `GET /api/teams/names` ist bereits vereinsweit lesbar für jeden eingeloggten Nutzer und liefert
  exakt die Felder, die der Picker braucht.

**Explizite Nicht-Ziele:** Der Trainer bekommt dadurch **keine** zusätzlichen Rechte auf fremden
Terminen — weder Sichtbarkeit (`ScopeGamesQuery` bleibt unangetastet), noch Anwesenheiten
(`canRecordGameAttendance`), noch Kader/Dienste. Er darf ausschließlich fremde Mannschaften
*einladen* zu einem Event, an dem seine eigene Mannschaft teilnimmt.

## Capabilities

### New Capabilities
- `game-mutation-team-scope`: Server-seitige Autorisierung pro Event für `POST`/`PUT`/`DELETE
  /api/games/{id}` — wer darf welches Event mutieren und mit welchen `team_ids`, abhängig von
  Vereinsfunktion und `event_type`.

### Modified Capabilities
- `event-wizard`: Requirement „Backend-Validierung Trainer-Scope" wird auf `heim`/`auswärts`
  eingegrenzt und um die „mindestens ein eigenes Team"-Bedingung für `generisch` ergänzt.
  Requirement „Mannschafts-Scoping nach Rolle und Typ" bekommt die Datenquelle des Pickers
  (`/api/teams/names` bei generisch) als prüfbares Kriterium.
- `game-edit-modal`: Der Mannschafts-Picker im Bearbeiten-Dialog folgt derselben Regel wie der
  Wizard — bei generischen Events alle aktiven Mannschaften, bei Spielen nur die eigenen.

## Impact

**Backend**
- `internal/games/handler.go`: `CreateGame` (bestehende Scope-Prüfung typabhängig machen),
  `UpdateGame` + `DeleteGame` (neue Scope-Prüfung), neuer Helper für die Prüfung.
- `internal/policy/rules.go`: ggf. Helper für „ist Trainer von Team X in aktiver Saison".
- Keine Migration, kein Schema-Change, keine neue Route → `internal/app/router.go` unverändert.

**Frontend**
- `web/src/components/GameEditModal.tsx`: zweite Team-Quelle für generische Events.
- `web/src/pages/KalenderPage.tsx`: Wizard-Checkboxen speisen sich bei `generisch` aus
  `allTeamNames` (bereits geladen) statt aus `teams`.
- Anzeige über `buildTeamShortNames` — kein Rückfall auf rohe DB-Namen (`team-names-endpoint`).

**Risiko / Regression**
- Verhaltensänderung für reine Trainer bei `PUT`/`DELETE /api/games/{id}`: Was heute
  (unbeabsichtigt) durchgeht, antwortet danach mit 403. Betrifft nur Zugriffe auf Events ohne
  eigene Mannschaft — in der UI heute ohnehin nicht erreichbar, da solche Events für den Trainer
  gar nicht gelistet werden (`ScopeGamesQuery`).

## Test-Anforderungen

| Route | Testname | Erwartung | Garantierte Invariante |
|---|---|---|---|
| `POST /api/games` | `TestCreateGame_TrainerGenericForeignTeamsAllowed` | 201 | Trainer darf bei `generisch` fremde `team_ids` setzen, solange ein eigenes Team dabei ist |
| `POST /api/games` | `TestCreateGame_TrainerGenericWithoutOwnTeam` | 403 | Trainer legt kein Event an, das er selbst nicht sähe |
| `POST /api/games` | `TestCreateGame_TrainerHomeGameForeignTeam` | 403 | Heim/Auswärts bleibt strikt auf eigene Teams beschränkt (Bestandsverhalten) |
| `PUT /api/games/{id}` | `TestUpdateGame_TrainerNotOnEvent` | 403 | Trainer mutiert kein Event ohne eigene beteiligte Mannschaft |
| `PUT /api/games/{id}` | `TestUpdateGame_TrainerGenericAddsForeignTeam` | 200 | Happy-Path des Features |
| `PUT /api/games/{id}` | `TestUpdateGame_TrainerRemovesOwnLastTeam` | 403 | Trainer schneidet sich nicht selbst von seinem Event ab |
| `PUT /api/games/{id}` | `TestUpdateGame_SportlicheLeitungUnrestricted` | 200 | sportliche_leitung/vorstand/admin bleiben ungefiltert |
| `DELETE /api/games/{id}` | `TestDeleteGame_TrainerNotOnEvent` | 403 | Löschlücke geschlossen |
| Frontend (Vitest) | `GameEditModal.genericTeams.test.tsx` | — | Bei `event_type='generisch'` erscheinen alle aktiven Mannschaften als Checkbox; bei `heim` nur die eigenen |
