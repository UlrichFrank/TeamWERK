## 1. Migration

- [x] 1.1 `ls internal/db/migrations/ | sort -V | tail -1` prüfen — nächste freie Nummer
      ermitteln (Stand Proposal: 044; erneut prüfen, falls parallel andere Migrationen
      gelandet sind).
- [x] 1.2 `internal/db/migrations/0NN_duty_template_item_team_scope.up.sql`:
      `ALTER TABLE game_template_items ADD COLUMN team_ids TEXT;`
- [x] 1.3 `internal/db/migrations/0NN_duty_template_item_team_scope.down.sql`:
      `ALTER TABLE game_template_items DROP COLUMN team_ids;`
- [x] 1.4 `make migrate-up` lokal ausführen, prüfen dass Bestandsdaten unverändert bleiben
      (`team_ids` ist bei allen bestehenden Zeilen NULL).

## 2. Backend — Speichern/Lesen der Vorlage

- [x] 2.1 `templateItem`-Struct (`internal/games/handler.go:1699-1707`) um
      `TeamIDs []int \`json:"team_ids,omitempty"\`` erweitern.
- [x] 2.2 `scanTemplateItems` (`handler.go:1709ff`, liest u.a. für `GET
      /api/duty-templates/{id}`) um `team_ids`-Spalte erweitern, JSON-Array parsen (analog zu
      `audiencesFromDB`, ggf. als neue kleine Helper-Funktion `teamIDsFromDB`/`teamIDsToDB` im
      selben Stil).
- [x] 2.3 `UpdateTemplate` (`handler.go:1825-1901`): Validierungsschleife nach der bestehenden
      `duty_type_id`-Prüfung (Zeile ~1849-1856) ergänzen — jede ID aus `it.TeamIDs` gegen
      `SELECT COUNT(*) FROM teams WHERE id=?` prüfen, bei Nichttreffer HTTP 400
      `{"error":"invalid_team"}` (eigener Fehlercode statt des generischen `"bad request"`,
      damit das Frontend gezielt reagieren kann).
- [x] 2.4 `INSERT INTO game_template_items` (`handler.go:1884-1887`) um die Spalte `team_ids`
      erweitern (`teamIDsToDB(it.TeamIDs)`).
- [x] 2.5 Bestehenden `h.hub.Broadcast("games")`-Aufruf (`handler.go:1899`) unverändert lassen
      — keine neue Route, kein neuer Broadcast-Bedarf.

## 3. Backend — Regen-Filter

- [x] 3.1 `templateItemRow` (`internal/games/regen.go`, nahe den anderen Feldern wie
      `Audiences`) um `TeamIDs []int` erweitern.
- [x] 3.2 `loadTemplateItemsTx` (`regen.go:626-648`) SELECT um `gti.team_ids` erweitern, Scan
      in `sql.NullString`, danach mit derselben JSON-Decode-Helper wie in 2.2 in `[]int`
      wandeln (Helper ggf. in ein gemeinsames Utility statt Duplikat pro Package — prüfen ob
      `internal/games` und der Lese-Pfad in `handler.go` sich einen Helper teilen können, da
      beide im selben Package `games` liegen).
- [x] 3.3 `regenGameItems` (`regen.go:377-462`): in der `for _, tid := range teamIDs`-Schleife
      (Zeile ~456-461) vor `insertOne` einen Filter einbauen: Team überspringen, wenn
      `len(it.TeamIDs) > 0 && !containsInt(it.TeamIDs, tid)`. Kleine `containsInt`-Helper-
      Funktion ergänzen (linearer Scan reicht, siehe design.md Open Questions).
- [x] 3.4 Zählung für `RegenSummary.Created[].Count`/`ReducedEntry.Count`
      (`regen.go:468/474`, aktuell `max(1, len(teamIDs)) * n`) auf die **gefilterte**
      Teammenge pro Item umstellen, nicht auf `len(teamIDs)` des gesamten Spiels.
- [x] 3.5 `generisch`-Zweig (`regen.go:451-455`, `insertOne(sql.NullInt64{})` ohne Team) bleibt
      unangetastet — `team_ids`-Filter gilt nur für `heim`/`auswärts`, da generische Events
      keine Team-Zuordnung pro Slot kennen (bestehendes Verhalten, siehe Kommentar
      „generisch never reaches here" im Code).

## 4. Frontend — Vorlagen-Editor

- [x] 4.1 `TemplateItem`-Interface (`web/src/pages/AdminDutyTemplateDetailPage.tsx:19-25`) um
      `team_ids?: number[] | null` erweitern; `newItem()` (Zeile 35-37) mit `team_ids: []`
      initialisieren.
- [x] 4.2 Aktive-Kaderteams beim Laden zusätzlich abrufen — **abweichend umgesetzt**: nur
      `GET /api/teams/names`, kein zweiter `/api/kader?season_id=`-Abruf. Die Annahme in
      design.md (Endpoint liefere alle aktiven Teams global, nicht saisongebunden) trifft
      nicht zu: `ListTeamNames` (`internal/games/handler.go`) joint bereits
      `kader ON k.season_id = (SELECT id FROM seasons WHERE is_active=1)` und liefert damit
      exakt die Kaderteams der aktiven Saison — die Schnittmenge aus Entscheidung 4 ist
      serverseitig schon gebildet (so auch in `GameEditModal.tsx:96` kommentiert). Kurznamen
      weiterhin über `buildTeamShortNames` (`web/src/lib/teamName.ts`).
- [x] 4.3 Neue Checkbox-Zeile "Kaderteams" pro Item, Mobile-Layout (nahe den bestehenden
      Zielgruppe-Checkboxen, Zeile ~226-244) und Desktop-Layout (Zeile ~300-317) — Muster von
      `AUDIENCE_OPTIONS` übernehmen, aber Optionen aus 4.2 statt aus `lib/constants`. Hinweis
      unterhalb: "leer = **alle** Kaderteams" (umgekehrte Semantik zu Zielgruppe, muss visuell
      abgesetzt sein, damit es nicht mit dem "leer = keine" der Zielgruppe verwechselt wird).
- [x] 4.4 `updateItem`/`removeItem`/`addItem` (Zeile 63-73) benötigen keine Änderung —
      arbeiten bereits generisch auf dem `TemplateItem`-Objekt.
- [x] 4.5 Gespeicherte `team_ids`, die auf ein Team verweisen, das in der aktiven Saison nicht
      mehr im Kader ist (Auswahlliste aus 4.2 enthält es nicht mehr): Checkbox-Zustand darf
      beim Speichern nicht verloren gehen — sicherstellen, dass `updateItem` nur die
      angezeigten Checkboxen toggelt und nicht-angezeigte `team_ids`-Einträge im Array
      unangetastet lässt (kein Neuaufbau des Arrays ausschließlich aus den sichtbaren
      Optionen).

## 5. Tests — Backend

- [x] 5.1 `internal/games/handler_test.go`: `TestUpdateTemplate_TeamIdsGespeichertUndGeladen`
      — Item mit `team_ids: [id1, id2]` speichern, per `GET` erneut laden, Werte stimmen
      überein.
- [x] 5.2 `TestUpdateTemplate_LeereTeamIdsBedeutetAlleTeams` — Item ohne `team_ids` speichern
      und laden, Feld ist leer/NULL (kein Default-Array mit falschen Werten).
- [x] 5.3 `TestUpdateTemplate_UnbekannteTeamId` — `team_ids` mit nicht existierender ID →
      HTTP 400 `invalid_team`, Vorlage in der DB unverändert (Vorher/Nachher-Vergleich wie bei
      bestehenden ähnlichen Tests in diesem File).
- [x] 5.4 `TestUpdateTemplate_StandardNutzerVerboten` — Standard-Nutzer ohne `vorstand` → 403
      (falls nicht schon durch bestehenden Route-Test abgedeckt — nur ergänzen, falls Lücke).
- [x] 5.5 Regen-Charakterisierungssuite: neuer Test
      `TestRegen_TeamEingeschraenktesItemNurFuerGelisteteTeams` — Spiel mit 2 Teams (game_teams),
      Item mit `team_ids` = nur eines der beiden Teams → nach Regen existiert der Slot nur für
      das gelistete Team.
- [x] 5.6 Bestehende Regen-Tests (Items ohne `team_ids`) laufen unverändert grün — als
      Nachweis der Rückwärtskompatibilität explizit im PR-Review nennen, kein neuer Testcode
      nötig, wenn Bestandstests bereits die Multi-Team-Situation ohne Filter abdecken (sonst
      einen ergänzen: `TestRegen_ItemOhneTeamIdsGiltFuerAlleTeams`).
- [x] 5.7 `TestRegen_ZaehlungBeruecksichtigtTeamFilter` — `RegenSummary.Created[].Count` bei
      einem team-eingeschränkten Item entspricht der gefilterten Teamzahl, nicht
      `len(teamIDs)` des Spiels.

## 6. Tests — Frontend

- [x] 6.1 `AdminDutyTemplateDetailPage.teamScope.test.tsx` neu angelegt (es existierte noch
      keine Test-Datei zu dieser Seite; Namenszusatz `.teamScope` folgt der Konvention der
      anderen Page-Tests in `web/src/pages/__tests__/` — `find web/src/pages -iname "AdminDutyTemplateDetailPage*"`
      vorher prüfen): Checkbox-Auswahl für Kaderteams rendert nur aktive Kaderteams, togglen
      aktualisiert `team_ids` im Item-State, Speichern sendet die erwartete Payload.
- [x] 6.2 Test für den "leer = alle Teams"-Hinweistext (sichtbar, nicht mit dem
      Zielgruppen-Hinweis verwechselbar).

## 7. Abschluss

- [x] 7.1 `/verify-change` laufen lassen (Build/Test/Lint + Projekt-Invarianten).
- [x] 7.2 `openspec validate duty-template-team-scope --strict` vor Archivierung.
- [x] 7.3 Commit(s) nach Conventional-Commits-Konvention, Scope `duties` oder `games`
      (`feat(games): Dienstplan-Vorlagen-Items auf Kaderteams einschränkbar` o.ä.), ein Commit
      pro sinnvoll abgeschlossenem Task-Block statt eines Monolith-Commits.

## 8. Nachtrag — Vorschau spiegelt den Team-Filter

Aufgedeckt beim Live-Test nach Abschluss von 1–7: der Wizard zeigte „Kamera" für mA2, obwohl
die Regeneration den Slot nicht erzeugt. Ursache war der ungefilterte Vorschau-Pfad (siehe
design.md Entscheidung 6).

- [x] 8.1 `PreviewSlots` (`internal/games/handler.go`): Query-Parameter `team_ids`
      (komma-separiert) einlesen; fehlt er und ist `game_id` gesetzt, Teams über
      `game_teams` nachladen. Keine der beiden Quellen vorhanden → ungefiltert (Bestand).
- [x] 8.2 `template_type` in die bestehende `SELECT id, duration_minutes …`-Query aufnehmen
      und den Filter **nur** für `heim`/`auswärts` anwenden — bei `generisch` ignoriert der
      Regen `team_ids`, die Vorschau muss das ebenso tun.
- [x] 8.3 Filter in der Item-Schleife: Item überspringen, wenn `len(it.TeamIDs) > 0` und
      keines der Request-Teams in der Allowlist steht (gleiche Bedingung wie
      `regenGameItems`, gemeinsame Helper-Funktion statt Copy-Paste).
- [x] 8.4 `KalenderPage.tsx:581`: `selectedTeamIds` als `&team_ids=…` an die Vorschau hängen.
- [x] 8.5 Kaderteams-Auswahl im Editor bei `template_type='generisch'` ausblenden (kein
      wirkungsloses Bedienelement). Bestehende `team_ids` solcher Vorlagen bleiben in der DB.
- [x] 8.6 Go-Tests: Item ausgeblendet bei nicht-passendem Team · sichtbar bei Treffer ·
      sichtbar ohne `team_ids`-Parameter · sichtbar bei `generisch` · Teams aus `game_id`
      abgeleitet.
- [x] 8.7 Vitest: Wizard sendet `team_ids`; Editor zeigt die Auswahl bei `generisch` nicht.
      Der URL-Bau wurde dafür als reine Funktion nach `web/src/lib/dutyPreview.ts`
      ausgelagert (`buildPreviewUrl`) — den 4-Schritt-Wizard durchzuklicken wäre teuer und
      spröde, und ein Nicht-Komponenten-Export in `KalenderPage.tsx` hebelt Fast Refresh aus
      (`react-refresh/only-export-components`).
- [x] 8.8 Gates erneut grün (`make test`, `make lint`, `pnpm -C web build`),
      `openspec validate --strict`, Commit.
