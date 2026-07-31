## ADDED Requirements

### Requirement: Mannschafts-Auswahl im Berechtigungsdialog

Das Typ-Dropdown in `PermissionsModal` SHALL zusätzlich zu `Alle Nutzer`, `Rolle`,
`Vereinsfunktion` und `Person` die Einträge **„Team"** (`team`) und **„Eltern"**
(`team_parents`) anbieten. Für beide Typen MUSS dasselbe zweite Dropdown erscheinen, das die
Kaderteams der aktiven Saison zur Auswahl stellt — analog zu den bestehenden bedingten Dropdowns
für `Rolle`, `Vereinsfunktion` und `Person`. Die Struktur des Dialogs bleibt im Übrigen unverändert;
es kommen keine zusätzlichen Bedienelemente hinzu.

Datenquelle ist `GET /api/teams/names` (aktive Saison, `teams.is_active = 1`), beschriftet mit dem
Kurznamen aus `buildTeamShortNames` („mA1", „mA2", „wB"). Die Liste MUSS wie der bestehende
Nutzer-Picker erst beim Wechsel auf einen der beiden Typen geladen werden. Beim Absenden MUSS die
`teams.id` als `principal_ref` übermittelt werden. Solange keine Mannschaft gewählt ist, bleibt der
Absende-Button deaktiviert.

#### Scenario: Team-Typ blendet die Mannschaftsauswahl ein
- **WHEN** ein Nutzer im Typ-Dropdown „Team" wählt
- **THEN** erscheint ein zweites `<select>` mit den Kaderteams der aktiven Saison, beschriftet mit
  ihren Kurznamen
- **THEN** ist der Button „Hinzufügen" deaktiviert, solange keine Mannschaft gewählt ist

#### Scenario: Eltern-Typ nutzt dieselbe Mannschaftsauswahl
- **WHEN** ein Nutzer im Typ-Dropdown „Eltern" wählt
- **THEN** erscheint dasselbe Mannschafts-Dropdown wie bei „Team"

#### Scenario: Gewählte Mannschaft wird als ID gespeichert
- **WHEN** ein Nutzer „Eltern" und die Mannschaft „mA1" (`teams.id = 7`) wählt, „Lesen" ankreuzt
  und absendet
- **THEN** sendet der POST-Request `principal_type: "team_parents"` und `principal_ref: "7"`

#### Scenario: Team-Eintrag wird mit Kurznamen dargestellt
- **WHEN** die Berechtigungsliste einen Eintrag mit `principal_type=team, principal_ref="7"`
  enthält und die Mannschaftsliste geladen ist
- **THEN** zeigt die Liste „Team: mA1"; für `team_parents` entsprechend „Eltern: mA1"

#### Scenario: Fallback ohne geladene Mannschaftsliste
- **WHEN** die Mannschaftsliste (noch) nicht geladen ist
- **THEN** zeigt die Liste den vom Server gelieferten `display_name` (Langname der Mannschaft aus
  `teams.name`), ersatzweise die rohe `principal_ref`

### Requirement: Eigentümer als nicht löschbarer Eintrag in der Berechtigungsliste

`GET /api/folders/{id}/permissions` SHALL der Liste einen synthetischen Eintrag voranstellen, der
den Ersteller des Ordners (`file_folders.created_by`) mit `principal_type: "owner"`, `id: 0`,
`can_read: true` und `can_write: true` ausweist. `display_name` MUSS `VORNAME NACHNAME` des
Erstellers enthalten und bei nicht auflösbarem Nutzer auf die User-ID zurückfallen.

Das Frontend MUSS diesen Eintrag ohne Löschen-Schaltfläche und als Eigentümer erkennbar darstellen.
Ohne ihn wäre das stärkste Recht am Ordner das einzige, das in keiner Oberfläche sichtbar ist.

#### Scenario: Eigentümer erscheint in der Liste
- **WHEN** ein Berechtigter `GET /api/folders/{id}/permissions` für einen von Nutzer F angelegten
  Ordner aufruft
- **THEN** enthält die Antwort als ersten Eintrag
  `{id: 0, principal_type: "owner", display_name: "<Name von F>", can_read: true, can_write: true}`

#### Scenario: Eigentümer-Eintrag ist nicht löschbar
- **WHEN** die Berechtigungsliste im Frontend gerendert wird
- **THEN** trägt die Eigentümer-Zeile keine Löschen-Schaltfläche

#### Scenario: Unauflösbarer Ersteller fällt auf die ID zurück
- **WHEN** der in `created_by` hinterlegte Nutzer nicht mehr auflösbar ist
- **THEN** enthält `display_name` die User-ID

## Test-Anforderungen

- Route `GET /api/folders/{id}/permissions`: `TestListPermissions_OwnerEntryFirst` — synthetischer
  `owner`-Eintrag mit `id=0` und Anzeigename an erster Stelle; `display_name` für
  `team`/`team_parents`-Zeilen entspricht `teams.name`.
- Route `POST /api/folders/{id}/permissions`: `TestAddPermission_OwnerPseudoTypeRejected` → 400.
- `DocumentsPage.permissions.test.tsx`: Wechsel auf „Team" rendert das Mannschafts-Dropdown und löst
  genau einen `GET /api/teams/names` aus (kein Nachladen bei erneutem Wechsel).
- `DocumentsPage.permissions.test.tsx`: Absenden mit Typ „Eltern" + Mannschaft sendet
  `principal_type=team_parents` und die `teams.id` als `principal_ref`.
- `DocumentsPage.permissions.test.tsx`: Bestandseintrag `team/7` wird als „Team: mA1" gerendert;
  die Eigentümer-Zeile wird ohne Löschen-Button gerendert.
