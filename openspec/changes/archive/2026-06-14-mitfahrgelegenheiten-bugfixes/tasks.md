## 1. Migration — UNIQUE INDEX für suche-Einträge

- [x] 1.1 Migration `0NN_mitfahr_suche_unique.up.sql`: bestehende suche-Duplikate bereinigen (nur neuesten Row pro `game_id, user_id` behalten), dann `CREATE UNIQUE INDEX idx_mitfahr_suche_unique ON mitfahrgelegenheiten(game_id, user_id) WHERE typ = 'suche'`
- [x] 1.2 Migration `0NN_mitfahr_suche_unique.down.sql`: `DROP INDEX IF EXISTS idx_mitfahr_suche_unique`
- [x] 1.3 `make migrate-up` lokal ausführen und prüfen

## 2. Backend — UPSERT für suche

- [x] 2.1 In `Upsert()` (`internal/carpooling/handler.go`): `suche`-Zweig auf SELECT-exists → UPDATE oder INSERT umstellen (analog zum `biete`-Zweig)
- [x] 2.2 `isNewEntry` korrekt setzen (nur bei INSERT)
- [x] 2.3 Kompilieren und manuell testen: zweimaliges POST für dasselbe Spiel mit `suche` erzeugt nur einen DB-Row

## 3. Backend — generische Events einmalig (GROUP BY)

- [x] 3.1 Alle vier Query-Varianten in `List()` auf `GROUP BY g.id` + `GROUP_CONCAT(t.name, ', ') AS team_names` umstellen (statt `JOIN teams` mit Einzelrow pro Team)
- [x] 3.2 Prüfen: generisches Event mit zwei Teams erscheint genau einmal, Team-Name enthält beide Namen

## 4. Frontend — Modal zeigt bestehende Einträge

- [x] 4.1 Beim Öffnen des Modals: bestehenden eigenen Eintrag des Typs aus `response.games` heraussuchen und als `initialData` an `FormModal` übergeben (`treffpunkt`, `notiz`, `plaetze`)
- [x] 4.2 `FormModal` initialisiert Felder aus `initialData` wenn vorhanden (statt immer leer)
- [x] 4.3 Beim Typ-Wechsel im Modal: den bestehenden Eintrag des neuen Typs laden (falls vorhanden) und Felder damit befüllen — andernfalls auf Defaults setzen
- [x] 4.4 Testen: Nutzer mit bestehendem biete-Eintrag öffnet Modal → sieht seine Daten; wechselt zu suche → sieht bestehenden suche-Eintrag oder leere Felder

## 5. Frontend — Team-Dropdown

- [x] 5.1 State `filterTeamId: number | null` anlegen (Default: `null`)
- [x] 5.2 `GET /teams/my` beim Laden aufrufen und in `myTeams`-State speichern
- [x] 5.3 Team-Dropdown nur rendern wenn `myTeams.length > 1` (Option „Alle" + eine Option pro Team)
- [x] 5.4 `load()`-Funktion: bei gesetztem `filterTeamId` → `?team_id={filterTeamId}` anhängen
- [x] 5.5 Bei Dropdown-Änderung: `filterTeamId` setzen und `load()` neu aufrufen
- [x] 5.6 Testen: Nutzer mit mehreren Teams sieht Dropdown; Nutzer mit einem Team nicht; Filter schränkt Events korrekt ein
