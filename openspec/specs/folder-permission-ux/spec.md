## ADDED Requirements

### Requirement: Anzeigename statt User-ID in der Berechtigungsliste
Das System SHALL für `principal_type=user`-Einträge in `GET /api/folders/{id}/permissions` einen `display_name`-String im Response mitliefern (`VORNAME NACHNAME`), der aus der `users`-Tabelle stammt. Ist der Nutzer gelöscht oder nicht auffindbar, MUSS `display_name` auf `principal_ref` (die User-ID) zurückfallen. Das Frontend MUSS diesen Namen anstelle der rohen User-ID darstellen.

#### Scenario: Berechtigung für existierenden User wird mit Name angezeigt
- **WHEN** `GET /api/folders/{id}/permissions` gibt einen Eintrag mit `principal_type=user, principal_ref="42"` zurück
- **THEN** enthält der Response-Eintrag `display_name: "Max Mustermann"` und das Frontend zeigt „Person: Max Mustermann" statt „Person (User-ID): 42"

#### Scenario: Berechtigung für gelöschten User fällt auf ID zurück
- **WHEN** `principal_type=user, principal_ref="99"` und User 99 existiert nicht mehr in der DB
- **THEN** ist `display_name: "99"` und das Frontend zeigt die ID

### Requirement: Team-scoped Nutzer-Picker beim Hinzufügen von User-Berechtigungen
Das System SHALL in der `PermissionsModal`-Komponente für den `principal_type=user`-Fall ein Dropdown (`<select>`) anstelle des Freitext-Inputs rendern. Das Dropdown MUSS ausschließlich Nutzer enthalten, die der anfragende Nutzer gemäß der bestehenden Teamsichtbarkeits-Logik sehen darf. Beim Absenden MUSS die ausgewählte User-ID als `principal_ref` übermittelt werden.

Die Sichtbarkeitsregel für `GET /api/users/picker` lautet:
- `role=admin` oder `club_function=vorstand`: alle Nutzer (`SELECT … FROM users`)
- Alle anderen: Nutzer, die in mindestens einem Team sichtbar sind, das der Aufrufer in der aktiven Saison über `user_accessible_teams` erreicht — d.h. Trainer, Spieler und Elternteile dieser Teams (identisch mit der Logik in `GET /api/teams/{id}/roster`)

#### Scenario: Admin sieht alle Nutzer im Picker
- **WHEN** ein Admin `GET /api/users/picker` aufruft
- **THEN** erhält er alle Nutzer des Systems sortiert nach Name

#### Scenario: Spieler sieht nur Nutzer seiner Teams
- **WHEN** ein Spieler U in Team T ist und `GET /api/users/picker` aufruft
- **THEN** enthält die Liste genau die Trainer, Spieler und Elternteile aller Teams, in denen U oder seine Kinder in der aktiven Saison eingetragen sind; Nutzer anderer Teams erscheinen nicht

#### Scenario: Elternteil sieht Teams seiner Kinder
- **WHEN** ein Elternteil E via `family_links` mit Mitglied M (in Team T) verknüpft ist und `GET /api/users/picker` aufruft
- **THEN** enthält die Liste die Nutzer aus Team T (Trainer, Spieler, Elternteile)

#### Scenario: Dropdown zeigt sichtbare Nutzer nach Name sortiert
- **WHEN** Nutzer wählt `principal_type=user` im Berechtigungen-Formular
- **THEN** erscheint ein `<select>` mit den erlaubten Nutzern sortiert nach Name; kein Freitext-Input

#### Scenario: Ausgewählter Name wird als ID gespeichert
- **WHEN** Nutzer wählt „Anna Beispiel" (user_id=7) und klickt „Hinzufügen"
- **THEN** sendet der POST-Request `principal_ref: "7"` ans Backend

#### Scenario: Nicht eingeloggte Nutzer werden abgewiesen
- **WHEN** ein nicht authentifizierter Request an `GET /api/users/picker` geht
- **THEN** antwortet das Backend mit 401

## Test-Anforderungen

- Route `GET /api/folders/{id}/permissions`: display_name im Response für user-Einträge vorhanden
- Route `GET /api/users/picker`: Admin sieht alle; Spieler sieht nur Team-Nutzer; 401 ohne Auth
