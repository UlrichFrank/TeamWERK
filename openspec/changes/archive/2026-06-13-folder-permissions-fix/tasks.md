## 1. resolveAccess() — Nearest-Ancestor-Wins

- [x] 1.1 `folderPath()` bleibt unverändert; `resolveAccess()` iteriert `path` von Index 0 (Zielordner) bis zum Root und führt für jeden Eintrag einen separaten Query auf `folder_permissions WHERE folder_id = ?` aus
- [x] 1.2 Beim ersten Ordner mit mindestens einer Berechtigungszeile: Matching-Loop ausführen, danach `return canRead, canWrite, nil` — keine weiteren Ebenen prüfen
- [x] 1.3 Endet die Iteration ohne Treffer (kein Ordner hat Einträge): `return false, false, nil`

## 2. Family-Context-Pre-Fetch

- [x] 2.1 Hilfsfunktion `fetchFamilyContext(db, userID int) (linkedUserIDs []int, linkedFunctions []string)` implementieren — Query: `family_links → members LEFT JOIN member_club_functions` gefiltert auf `parent_user_id = ?`; bei leerem Ergebnis leere Slices zurückgeben
- [x] 2.2 `resolveAccess()` ruft `fetchFamilyContext` zu Beginn auf (nach Admin-Kurzschluss); Ergebnis in lokalen Variablen halten
- [x] 2.3 Matching-Loop erweitern: `club_function`-Case matcht zusätzlich wenn `pr.String ∈ linkedFunctions`; `user`-Case matcht zusätzlich wenn `pr.String` als int in `linkedUserIDs` enthalten ist

## 3. User-Anzeigename im Permissions-Response (Backend)

- [x] 3.1 `permResponse`-Struct um optionales Feld `DisplayName string \`json:"display_name,omitempty"\`` erweitern
- [x] 3.2 `ListPermissions()`: für jeden Eintrag mit `principal_type=user` einen JOIN auf `users` ausführen und `first_name || ' ' || last_name` als `DisplayName` setzen; bei nicht gefundenem User auf `principal_ref` zurückfallen

## 4. Nutzer-Picker-Endpunkt (Backend)

- [x] 4.1 Neuen Handler `GET /api/users/picker` implementieren mit zwei Zweigen: (a) `role=admin` oder `club_function=vorstand` → `SELECT id, first_name||' '||last_name FROM users ORDER BY 2`; (b) alle anderen → UNION-Query über `user_accessible_teams` (aktive Saison) für Trainer (`kader_trainers → members → users`), Spieler (`kader_members → members → users`) und Elternteile (`kader_members → family_links → users`) des Aufrufers; Response: `[{id, name}]`
- [x] 4.2 Route in `main.go` unter der `authenticated`-Gruppe registrieren (kein zusätzliches Rollen-Gate)

## 5. Frontend — PermissionsModal

- [x] 5.1 In `PermissionsModal`: Nutzer-Picker-State (`users: {id: number, name: string}[]`) hinzufügen; Laden per `GET /api/users/picker` beim ersten Öffnen des `user`-Typs (lazy) oder beim Mount des Modals
- [x] 5.2 `{newType === 'user'}` Block: Freitext-Input durch `<select>` ersetzen; Optionen aus geladener Nutzerliste; Value = `String(user.id)`
- [x] 5.3 `permLabel()`: für `principal_type === 'user'` den `display_name` aus dem Permission-Objekt verwenden statt `principal_ref`; `Permission`-Interface um `display_name?: string` erweitern
- [x] 5.4 Label `PRINCIPAL_TYPE_LABELS.user` von `'Person (User-ID)'` auf `'Person'` ändern

## 6. Tests

- [x] 6.1 `TestResolveAccess_NearestAncestorWins`: Elternordner `everyone: read`, Unterordner `club_function=vorstand: read` — Standard-Nutzer fragt Unterordner → 403
- [x] 6.2 `TestResolveAccess_InheritFromParent`: Elternordner `everyone: read`, Unterordner ohne Einträge — Standard-Nutzer fragt Unterordner → 200
- [x] 6.3 `TestResolveAccess_NoRulesAnywhere`: weder Ordner noch Vorfahren haben Einträge — beliebiger Nicht-Admin → false, false
- [x] 6.4 `TestResolveAccess_FamilyContext_ClubFunction`: Ordner hat `club_function=spieler: read`; Nutzer ist kein Spieler, aber via `family_links` mit Spieler verknüpft → canRead=true
- [x] 6.5 `TestResolveAccess_FamilyContext_UserID`: Ordner hat `user=42: read`; Nutzer P ist via `family_links` mit User 42 verknüpft → canRead=true
- [x] 6.6 `TestFolderContents_RestrictedSubfolder`: Integrations-Test gegen echten Handler — Standard-Nutzer fragt `GET /api/folders/{id}/contents` auf restriktivem Unterordner → 403
- [x] 6.7 `TestListPermissions_DisplayName`: `GET /api/folders/{id}/permissions` für user-Eintrag enthält `display_name` mit vollem Namen
- [x] 6.8 `TestUsersPicker_AdminSeesAll`: Admin erhält alle Nutzer
- [x] 6.9 `TestUsersPicker_SpielerSeesTeamOnly`: Spieler in Team T erhält nur Nutzer aus Team T, keine Nutzer anderer Teams
