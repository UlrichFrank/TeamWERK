# Tasks

Ein Commit pro Abschnitt. Conventional Commits, Scope `files` bzw. `db`.

## 1. Migration — Principal-Typen erweitern

- [x] 1.1 `internal/db/migrations/038_folder_permissions_team_principals.up.sql`: Table-Rebuild von
  `folder_permissions` mit erweitertem CHECK (`'everyone','role','club_function','user','team','team_parents'`).
  Spaltendefinitionen und den ausgehenden FK auf `file_folders(id) ON DELETE CASCADE` unverändert
  übernehmen, Daten per `INSERT … SELECT *` kopieren, alte Tabelle droppen, neue umbenennen.
- [x] 1.2 `038_…down.sql`: **zuerst** `DELETE FROM folder_permissions WHERE principal_type IN ('team','team_parents')`,
  dann Rebuild mit dem alten Vier-Werte-CHECK. Datenverlust in Rückrichtung als Kommentar in beiden
  Dateien vermerken.
- [x] 1.3 `make migrate-up` und `make migrate-down` lokal gegen eine Kopie durchspielen; prüfen dass
  Bestandszeilen erhalten bleiben und `folder_permissions` danach wieder beschreibbar ist.
  **Befund:** `make migrate-down` ist ein No-Op — `runMigrate` (`cmd/teamwerk/main.go:501`) ruft
  unabhängig vom Argument immer `db.Migrate` (up). Das Down-SQL wurde stattdessen direkt per
  `sqlite3` gegen eine Wegwerf-DB verifiziert: `everyone`-Zeile inkl. ID erhalten, Team-Zeilen
  gelöscht, alter CHECK wiederhergestellt, `PRAGMA foreign_key_check` leer. Der fehlende
  Down-Pfad im CLI ist ein bestehender, separater Defekt — nicht Teil dieses Changes.

## 2. Policy — Eigentümer-Vorrang

- [x] 2.1 In `internal/policy/folders.go` `ownsAnyOf(db, userID int, path []int) (bool, error)`
  ergänzen: eine Query `SELECT 1 FROM file_folders WHERE created_by = ? AND id IN (…) LIMIT 1` über
  den bereits berechneten Pfad. Fehler nicht schlucken.
- [x] 2.2 In `FolderAccess` den Aufruf **nach** `folderPath` und **vor** der Walk-Schleife einsetzen;
  bei Treffer `return true, true, nil`.
- [x] 2.3 Doc-Kommentar über `FolderAccess` um die Eigentümerregel ergänzen (absolut, unterbaumweit,
  vor dem Walk) — die bestehende Nearest-Ancestor-Erklärung bleibt stehen.

## 3. Policy — Lazy Principal-Kontext

- [x] 3.1 `principalCtx`-Struct mit `db`, `userID` sowie den Feldern für Family- und Team-Kontext
  anlegen; `fetchFamilyContext` in einen Lazy-Getter `family()` überführen (lädt einmalig, cacht).
- [x] 3.2 Getter `teams() (playerTeams, parentTeams []int, err error)` mit den beiden Queries aus
  `design.md` (Entscheidung 2). Beide filtern auf `seasons.is_active = 1` und `k.team_id IS NOT NULL`.
- [x] 3.3 Walk-Schleife auf die Getter umstellen: `club_function`/`user` rufen `family()`,
  `team`/`team_parents` rufen `teams()`. `everyone` und `role` lösen keine Query aus.
- [x] 3.4 `switch pt.String` um die Fälle `"team"` und `"team_parents"` erweitern
  (`principal_ref` per `strconv.Atoi`, ungültige Werte matchen nicht).

## 4. Policy — Tests

- [x] 4.1 Fixtures prüfen: `testutil.CreateKader`, `CreateFolder`, `CreateMember` decken die Setups
  ab; falls ein Helfer für „Member in Kader eintragen" bzw. `family_links` fehlt, lokalen Helper in
  `folders_test.go` ergänzen (kein `player_memberships`-INSERT — das ist eine View).
- [x] 4.2 `TestFolderAccess_OwnerKeepsRightsAfterGrantingToOthers`
- [x] 4.3 `TestFolderAccess_OwnerRightsSpanSubtree`
- [x] 4.4 `TestFolderAccess_OwnerWithoutClubFunction`
- [x] 4.5 `TestFolderAccess_NonOwnerNoAccess`
- [x] 4.6 `TestFolderAccess_TeamPlayerMatches` / `_TeamTrainerMatches` / `_TeamExtendedMemberMatches`
- [x] 4.7 `TestFolderAccess_TeamParentNotMatchedByTeam` / `_TeamParentsMatches` /
  `_TeamParentsDoesNotMatchPlayer`
- [x] 4.8 `TestFolderAccess_TeamOtherTeamNoAccess` / `_TeamInactiveSeasonNoAccess` /
  `_TeamNoActiveSeasonFailsClosed`
- [x] 4.9 Bestehende Nearest-Ancestor-Tests durchsehen: überall dort, wo die Einschränkung geprüft
  wird, MUSS der Ordner-Ersteller ein *anderer* Nutzer als der anfragende sein — sonst maskiert der
  Eigentümer-Vorrang die Assertion. Betroffene Fixtures anpassen.
  **Befund:** keine Anpassung nötig — in `internal/files/handler_test.go` legt durchgehend `adminID`
  die Ordner an, die anfragenden Nutzer sind Nicht-Admins. In `folders_test.go` ebenso. Zusätzlich
  per Poison-Check abgesichert: mit deaktiviertem Owner-Kurzschluss fallen die drei Owner-Tests,
  `TestFolderAccess_NonOwnerNoAccess` bleibt korrekt grün.

## 5. Backend — Konsolidierung auf `policy.FolderAccess`

- [x] 5.1 Helfer `(h *Handler) access(r *http.Request, folderID int) (bool, bool, error)` in
  `internal/files/handler.go` anlegen (baut `policy.Principal` aus den Claims, ruft
  `policy.FolderAccess`).
  **Abweichung:** stattdessen paketweite Funktion `folderAccess(db, claims, folderID)` mit der
  Signatur der abgelösten `resolveAccess`. Grund: `DownloadFile` baut sich im Token-Pfad die Claims
  über `claimsForUser` selbst; sie stehen dort **nicht** im Request-Kontext, eine
  `*http.Request`-Signatur hätte diesen Aufruf nicht abgedeckt. Nil-Claims (öffentliche
  Download-Route) lösen fail-closed zu `false, false` auf.
- [x] 5.2 Alle 14 `resolveAccess(...)`-Aufrufstellen auf den Helfer umstellen; `resolveAccess`,
  `folderPath` und `fetchFamilyContext` aus `files/handler.go` ersatzlos entfernen.
- [x] 5.3 `checkAntiEscalation` auf den Helfer umstellen (Signatur nimmt künftig `*http.Request`
  statt `*auth.Claims`, oder erhält den Principal übergeben).
- [x] 5.4 `go build ./...`, `make lint` und `go test ./internal/arch/...` grün — insbesondere der
  Architektur-Test für die Kante `files → policy`.

## 6. Backend — Routen

- [x] 6.1 `AddPermission`: `validTypes` um `team` und `team_parents` erweitern. `owner` bleibt
  ungültig → 400.
- [x] 6.2 `AddPermission`: `principal_ref` für die neuen Typen als Pflichtfeld prüfen (nicht leer,
  numerisch) → sonst 400.
- [x] 6.3 `ListPermissions`: `display_name` für `team`/`team_parents` aus `teams.name` auflösen,
  Fallback auf `principal_ref` (Muster von der bestehenden `user`-Auflösung übernehmen).
- [x] 6.4 `ListPermissions`: synthetischen `owner`-Eintrag (`id: 0`) an den Anfang der Liste setzen,
  Name aus `users` über `file_folders.created_by`, Fallback auf die ID.
- [x] 6.5 Kein Router- und kein Broadcast-Eingriff nötig (`Files.*` steht in der
  `broadcastAllowlist`) — durch `go test ./internal/arch/...` bestätigen.

## 7. Backend — Routen-Tests

- [x] 7.1 `TestListPermissions_OwnerNotLockedOut` → 200 für den Ersteller ohne passenden ACL-Eintrag.
- [x] 7.2 `TestListPermissions_OwnerEntryFirst` → `owner`-Eintrag mit `id=0` und Anzeigename an
  Position 0; `display_name` für eine `team`-Zeile entspricht `teams.name`.
- [x] 7.3 `TestAddPermission_OwnerMayGrant` → 201.
- [x] 7.4 `TestAddPermission_TeamType` → 201 und gespeicherte Zeile in `folder_permissions`.
- [x] 7.5 `TestAddPermission_OwnerPseudoTypeRejected` → 400, keine Zeile geschrieben.
- [x] 7.6 `TestAddPermission_TeamTypeMissingRef` → 400.
- [x] 7.7 `TestFolderContents_TeamPermission` → 200 für Kadermitglied, 403 für Nutzer einer anderen
  Mannschaft.

## 8. Frontend — Berechtigungsdialog

- [x] 8.1 `PRINCIPAL_TYPE_LABELS` in `web/src/pages/DocumentsPage.tsx` um `team: 'Team'` und
  `team_parents: 'Eltern'` erweitern.
- [x] 8.2 Lazy-Loader `loadTeams()` nach dem Vorbild von `loadPickerUsers` (`:252`) auf
  `GET /api/teams/names`; im `onChange` des Typ-Selects für beide neuen Typen auslösen.
- [x] 8.3 **Einen** gemeinsamen bedingten `<select>`-Block für `team | team_parents` ergänzen,
  Optionen aus `buildTeamShortNames`, Wert = `teams.id`. Klassen-Strings von den bestehenden
  Blöcken übernehmen (`brand-*`-Tokens, keine Raw-Farben).
- [x] 8.4 `permLabel` (`:294`) um die neuen Typen erweitern: Kurzname aus der geladenen Teamliste,
  Fallback `display_name`, dann `principal_ref`.
- [x] 8.5 Eigentümer-Zeile rendern: als Eintrag der Liste, ohne Löschen-Button (`principal_type === 'owner'`).
- [x] 8.6 Der bestehende Disabled-Guard `newType !== 'everyone' && !newRef` (`:398`) deckt die neuen
  Typen bereits ab — verifizieren, nicht anfassen.

## 9. Frontend — Tests

- [x] 9.1 `web/src/pages/__tests__/DocumentsPage.permissions.test.tsx` (`PermissionsModal` dafür
  exportiert — Muster wie die Progress-Throttle-Factory in `VideoUploadPage`): Wechsel auf „Team" rendert das
  Mannschafts-Dropdown; `GET /api/teams/names` wird genau einmal aufgerufen (auch bei erneutem
  Typwechsel).
- [x] 9.2 Absenden mit „Eltern" + Mannschaft sendet `principal_type=team_parents` und die `teams.id`.
- [x] 9.3 Bestandseintrag `team/7` wird als „Team: mA1" gerendert.
- [x] 9.4 Eigentümer-Zeile wird ohne Löschen-Button gerendert.
- [x] 9.5 `pnpm -C web test` und `pnpm -C web lint` grün.

## 10. Abschluss

- [x] 10.1 `/verify-change` durchlaufen (Build/Test/Lint + Projekt-Invarianten: Route→Tests,
  brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`).
- [x] 10.2 Gotcha in `docs/agent/06-gotchas.md` ergänzen: Eigentümer-Vorrang bei Ordnerrechten
  (absolut, unterbaumweit, heilt Bestandsordner ohne Backfill) und Team-Principals (Auflösung gegen
  die aktive Saison, Fail-Closed ohne aktive Saison).
- [x] 10.3 Proposal archivieren.
